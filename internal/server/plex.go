package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/media"
	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/internal/secretbox"
)

const clientIdentifierSetting = "plex_client_identifier"

type plexLink struct {
	usernames     *usernameCache
	client        plex.Client
	server        plex.ServerClient
	sources       map[mediaKind]media.Source
	clientID      string
	box           *secretbox.Box
	queries       *dbgen.Queries
	pool          *pgxpool.Pool
	logger        *slog.Logger
	anilist       *aniListCache
	anilistClient *anilist.Client
	shows         *showsCache
	syncing       sync.Map
}

func (p *plexLink) register(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("GET /api/me", a.authenticated(p.me))
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
		internalError(w, p.logger, err)
		return
	}
	if err := p.queries.UpsertPlexAccount(r.Context(), dbgen.UpsertPlexAccountParams{UserID: s.UserID, TokenEncrypted: sealed}); err != nil {
		internalError(w, p.logger, err)
		return
	}
	p.usernames.evict(s.UserID)
	writeJSON(w, map[string]string{"status": "linked", "username": username})
}

func (p *plexLink) me(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	body := map[string]any{"displayName": s.DisplayName}
	tokenEncrypted, err := p.queries.GetPlexAccount(r.Context(), s.UserID)
	switch {
	case err == nil:
		username, needsReSignIn := p.refreshUsername(r.Context(), s.UserID, tokenEncrypted)
		plexBody := map[string]any{"needsReSignIn": needsReSignIn}
		if username != "" {
			plexBody["username"] = username
		}
		body["plex"] = plexBody
	case !errors.Is(err, pgx.ErrNoRows):
		internalError(w, p.logger, err)
		return
	}
	writeJSON(w, body)
}

func (p *plexLink) refreshUsername(ctx context.Context, userID int64, tokenEncrypted []byte) (string, bool) {
	if name, ok := p.usernames.fresh(userID); ok {
		return name, false
	}
	token, err := p.box.Open(tokenEncrypted)
	if err != nil {
		p.logger.Warn("open plex token failed", "error", err)
		p.usernames.evict(userID)
		return "", true
	}
	username, err := p.client.Username(ctx, p.clientID, string(token))
	if errors.Is(err, plex.ErrUnauthorized) {
		p.usernames.evict(userID)
		return "", true
	}
	if err != nil {
		p.logger.Warn("plex username refresh failed", "error", err)
		name, _ := p.usernames.stale(userID)
		return name, false
	}
	p.usernames.store(userID, username)
	return username, false
}

func (p *plexLink) upstreamError(w http.ResponseWriter, err error) {
	p.logger.Warn("plex.tv request failed", "error", err)
	http.Error(w, "plex.tv request failed", http.StatusBadGateway)
}
