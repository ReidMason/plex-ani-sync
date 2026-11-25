package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/plex"
)

type mediaSourceJSON struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

func (p *plexLink) registerMediaSources(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("GET /api/media-sources", a.authenticated(p.listMediaSources))
	mux.HandleFunc("POST /api/media-sources", a.authenticated(p.addMediaSource))
}

func (p *plexLink) listMediaSources(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	rows, err := p.queries.ListMediaSources(r.Context(), s.UserID)
	if err != nil {
		p.internalError(w, err)
		return
	}
	out := make([]mediaSourceJSON, 0, len(rows))
	for _, row := range rows {
		out = append(out, mediaSourceJSON{ID: row.ID, URL: row.Url})
	}
	writeJSON(w, out)
}

func (p *plexLink) addMediaSource(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	var in struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_url")
		return
	}
	serverURL, err := plex.NormalizeServerURL(in.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_url")
		return
	}
	token, err := p.accountToken(r, s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "plex_not_linked")
		return
	}
	if err != nil {
		p.internalError(w, err)
		return
	}
	switch err := p.server.CheckAccess(r.Context(), p.clientID, serverURL, token); {
	case errors.Is(err, plex.ErrUnreachable):
		writeError(w, http.StatusBadGateway, "unreachable")
		return
	case errors.Is(err, plex.ErrNotPlexServer):
		writeError(w, http.StatusUnprocessableEntity, "not_plex_server")
		return
	case errors.Is(err, plex.ErrNoAccess):
		writeError(w, http.StatusForbidden, "no_access")
		return
	case err != nil:
		p.internalError(w, err)
		return
	}
	row, err := p.queries.InsertMediaSource(r.Context(), dbgen.InsertMediaSourceParams{UserID: s.UserID, Url: serverURL})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "already_added")
		return
	}
	if err != nil {
		p.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, mediaSourceJSON{ID: row.ID, URL: row.Url})
}

func (p *plexLink) accountToken(r *http.Request, userID int64) (string, error) {
	account, err := p.queries.GetPlexAccount(r.Context(), userID)
	if err != nil {
		return "", err
	}
	token, err := p.box.Open(account.TokenEncrypted)
	return string(token), err
}

func writeError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
