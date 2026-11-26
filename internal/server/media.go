package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/media"
	"github.com/ReidMason/plex-ani-sync/internal/plex"
)

type mediaKind string

const mediaKindPlex mediaKind = "plex"

type mediaSourceJSON struct {
	ID   int64     `json:"id"`
	URL  string    `json:"url"`
	Kind mediaKind `json:"kind"`
}

func (p *plexLink) registerMediaSources(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("GET /api/media-sources", a.authenticated(p.listMediaSources))
	mux.HandleFunc("POST /api/media-sources", a.authenticated(p.addMediaSource))
	mux.HandleFunc("GET /api/media-sources/{id}/libraries", a.authenticated(p.listLibraries))
	mux.HandleFunc("PUT /api/media-sources/{id}/anime-libraries", a.authenticated(p.selectAnimeLibraries))
	mux.HandleFunc("DELETE /api/media-sources/{id}", a.authenticated(p.removeMediaSource))
}

func (p *plexLink) removeMediaSource(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	n, err := p.queries.DeleteMediaSource(r.Context(), dbgen.DeleteMediaSourceParams{ID: id, UserID: s.UserID})
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	if n == 0 {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *plexLink) listMediaSources(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	rows, err := p.queries.ListMediaSources(r.Context(), s.UserID)
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	out := make([]mediaSourceJSON, 0, len(rows))
	for _, row := range rows {
		out = append(out, mediaSourceJSON{ID: row.ID, URL: row.Url, Kind: mediaKind(row.Kind)})
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
		internalError(w, p.logger, err)
		return
	}
	switch err := p.server.CheckAccess(r.Context(), p.clientID, serverURL, token); {
	case errors.Is(err, plex.ErrUnreachable):
		writeError(w, http.StatusBadGateway, "unreachable")
		return
	case errors.Is(err, plex.ErrNotPlexServer):
		p.logger.Warn("plex returned an unexpected response", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "not_plex_server")
		return
	case errors.Is(err, plex.ErrNoAccess):
		writeError(w, http.StatusForbidden, "no_access")
		return
	case err != nil:
		internalError(w, p.logger, err)
		return
	}
	row, err := p.queries.InsertMediaSource(r.Context(), dbgen.InsertMediaSourceParams{UserID: s.UserID, Url: serverURL, Kind: string(mediaKindPlex)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "already_added")
		return
	}
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, mediaSourceJSON{ID: row.ID, URL: row.Url, Kind: mediaKind(row.Kind)})
}

func (p *plexLink) accountToken(r *http.Request, userID int64) (string, error) {
	sealed, err := p.queries.GetPlexAccount(r.Context(), userID)
	if err != nil {
		return "", err
	}
	token, err := p.box.Open(sealed)
	return string(token), err
}

type libraryJSON struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Selected bool   `json:"selected"`
}

func (p *plexLink) listLibraries(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	src, libs, ok := p.loadLibraries(w, r, s)
	if !ok {
		return
	}
	keys, err := p.queries.ListAnimeLibraryKeys(r.Context(), src.ID)
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	out := make([]libraryJSON, 0, len(libs))
	for _, l := range libs {
		out = append(out, libraryJSON{Key: l.Key, Title: l.Title, Selected: slices.Contains(keys, l.Key)})
	}
	writeJSON(w, out)
}

func (p *plexLink) selectAnimeLibraries(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	var in struct {
		Keys []string `json:"keys"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	src, libs, ok := p.loadLibraries(w, r, s)
	if !ok {
		return
	}
	for _, k := range in.Keys {
		if !slices.ContainsFunc(libs, func(l media.Library) bool { return l.Key == k }) {
			writeError(w, http.StatusUnprocessableEntity, "unknown_library")
			return
		}
	}
	if err := p.replaceAnimeLibraries(r.Context(), src.ID, in.Keys); err != nil {
		internalError(w, p.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *plexLink) replaceAnimeLibraries(ctx context.Context, sourceID int64, keys []string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := p.queries.WithTx(tx)
	if err := q.ClearAnimeLibraries(ctx, sourceID); err != nil {
		return err
	}
	if err := q.AddAnimeLibraries(ctx, dbgen.AddAnimeLibrariesParams{MediaSourceID: sourceID, LibraryKeys: slices.Compact(slices.Sorted(slices.Values(keys)))}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// loadLibraries writes the error response itself and reports ok=false when it cannot return the libraries.
func (p *plexLink) loadLibraries(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) (dbgen.GetMediaSourceRow, []media.Library, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return dbgen.GetMediaSourceRow{}, nil, false
	}
	src, err := p.queries.GetMediaSource(r.Context(), dbgen.GetMediaSourceParams{ID: id, UserID: s.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return src, nil, false
	}
	if err != nil {
		internalError(w, p.logger, err)
		return src, nil, false
	}
	token, err := p.accountToken(r, s.UserID)
	if err != nil {
		internalError(w, p.logger, err)
		return src, nil, false
	}
	libs, err := p.librariesOf(r.Context(), src.Kind, media.Connection{URL: src.Url, Token: token})
	if err != nil {
		p.writeMediaError(w, err)
		return src, nil, false
	}
	return src, libs, true
}

func (p *plexLink) librariesOf(ctx context.Context, kind string, c media.Connection) ([]media.Library, error) {
	source, ok := p.sources[mediaKind(kind)]
	if !ok {
		return nil, fmt.Errorf("no media source for kind %q", kind)
	}
	return source.Libraries(ctx, c)
}

func (p *plexLink) writeMediaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, media.ErrUnreachable):
		writeError(w, http.StatusBadGateway, "unreachable")
	case errors.Is(err, media.ErrNoAccess):
		writeError(w, http.StatusForbidden, "no_access")
	case errors.Is(err, plex.ErrNotPlexServer):
		p.logger.Warn("plex returned an unexpected response", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "not_plex_server")
	default:
		internalError(w, p.logger, err)
	}
}
