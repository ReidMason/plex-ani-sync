package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/internal/secretbox"
)

const clientIdentifierSetting = "plex_client_identifier"

type plexLink struct {
	client   plex.Client
	server   plex.ServerClient
	clientID string
	box      *secretbox.Box
	queries  *dbgen.Queries
	logger   *slog.Logger
}

func (p *plexLink) register(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("POST /api/plex/pins", a.authenticated(p.createPIN))
	mux.HandleFunc("POST /api/plex/pins/{id}/poll", a.authenticated(p.pollPIN))
}

// loadClientIdentifier returns the install-wide Plex client identifier, generating it on first use.
func loadClientIdentifier(ctx context.Context, q *dbgen.Queries) (string, error) {
	if err := q.InsertSettingIfMissing(ctx, dbgen.InsertSettingIfMissingParams{Key: clientIdentifierSetting, Value: uuid.NewString()}); err != nil {
		return "", err
	}
	return q.GetSetting(ctx, clientIdentifierSetting)
}

func (p *plexLink) createPIN(w http.ResponseWriter, r *http.Request, _ dbgen.GetSessionUserRow) {
	pin, err := p.client.CreatePIN(r.Context(), p.clientID)
	if err != nil {
		p.upstreamError(w, err)
		return
	}
	writeJSON(w, map[string]any{"id": pin.ID, "code": pin.Code, "authUrl": plex.AuthURL(p.clientID, pin)})
}

func (p *plexLink) pollPIN(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	code := r.URL.Query().Get("code")
	if err != nil || code == "" {
		http.Error(w, "invalid pin", http.StatusBadRequest)
		return
	}
	token, err := p.client.CheckPIN(r.Context(), p.clientID, plex.PIN{ID: id, Code: code})
	if err != nil {
		p.upstreamError(w, err)
		return
	}
	if token == "" {
		writeJSON(w, map[string]string{"status": "pending"})
		return
	}
	username, err := p.client.Username(r.Context(), p.clientID, token)
	if err != nil {
		p.upstreamError(w, err)
		return
	}
	sealed, err := p.box.Seal([]byte(token))
	if err != nil {
		p.internalError(w, err)
		return
	}
	if err := p.queries.UpsertPlexAccount(r.Context(), dbgen.UpsertPlexAccountParams{UserID: s.UserID, TokenEncrypted: sealed, Username: username}); err != nil {
		p.internalError(w, err)
		return
	}
	writeJSON(w, map[string]string{"status": "linked", "username": username})
}

func (p *plexLink) upstreamError(w http.ResponseWriter, err error) {
	p.logger.Warn("plex.tv request failed", "error", err)
	http.Error(w, "plex.tv request failed", http.StatusBadGateway)
}

func (p *plexLink) internalError(w http.ResponseWriter, err error) {
	p.logger.Error("request failed", "error", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}
