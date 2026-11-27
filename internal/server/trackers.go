package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
)

const trackerKindAniList = "anilist"

func (p *plexLink) registerTrackers(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("GET /api/trackers/anilist", a.authenticated(p.getAniListTracker))
	mux.HandleFunc("PUT /api/trackers/anilist", a.authenticated(p.linkAniListTracker))
	mux.HandleFunc("DELETE /api/trackers/anilist", a.authenticated(p.unlinkAniListTracker))
	mux.HandleFunc("GET /auth/anilist/start", a.authenticated(p.startAniListAuthorisation))
	mux.HandleFunc("GET /auth/anilist/callback", a.authenticated(p.finishAniListAuthorisation))
}

func (p *plexLink) aniListUsername(ctx context.Context, userID int64) (*string, error) {
	name, err := p.queries.GetTrackerUsername(ctx, dbgen.GetTrackerUsernameParams{UserID: userID, Kind: trackerKindAniList})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &name, nil
}

func (p *plexLink) getAniListTracker(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	row, err := p.queries.GetTrackerStatus(r.Context(), dbgen.GetTrackerStatusParams{UserID: s.UserID, Kind: trackerKindAniList})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, p.trackerStatus(nil, false))
		return
	}
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	writeJSON(w, p.trackerStatus(&row.Username, row.Authorised))
}

func (p *plexLink) trackerStatus(username *string, authorised bool) trackerStatus {
	return trackerStatus{Username: username, Authorised: authorised, AuthorisationAvailable: p.aniListOAuth != nil}
}

type trackerStatus struct {
	Username               *string `json:"username"`
	Authorised             bool    `json:"authorised"`
	AuthorisationAvailable bool    `json:"authorisationAvailable"`
}

func (p *plexLink) linkAniListTracker(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	var body struct {
		Username string `json:"username"`
	}
	username := ""
	if json.NewDecoder(r.Body).Decode(&body) == nil {
		username = strings.TrimSpace(body.Username)
	}
	if username == "" {
		writeError(w, http.StatusBadRequest, "invalid_username")
		return
	}
	if _, err := p.anilistClient.UserList(r.Context(), username); err != nil {
		p.writeAniListError(w, err)
		return
	}
	if err := p.queries.UpsertTracker(r.Context(), dbgen.UpsertTrackerParams{UserID: s.UserID, Kind: trackerKindAniList, Username: username}); err != nil {
		internalError(w, p.logger, err)
		return
	}
	status, err := p.queries.GetTrackerStatus(r.Context(), dbgen.GetTrackerStatusParams{UserID: s.UserID, Kind: trackerKindAniList})
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	writeJSON(w, p.trackerStatus(&username, status.Authorised))
}

func (p *plexLink) unlinkAniListTracker(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	if err := p.queries.DeleteTracker(r.Context(), dbgen.DeleteTrackerParams{UserID: s.UserID, Kind: trackerKindAniList}); err != nil {
		internalError(w, p.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *plexLink) writeAniListError(w http.ResponseWriter, err error) {
	var throttled *anilist.ThrottledError
	switch {
	case errors.Is(err, anilist.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user_not_found")
	case errors.Is(err, anilist.ErrPrivateList):
		writeError(w, http.StatusUnprocessableEntity, "private_list")
	case errors.As(err, &throttled):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "throttled", "until": throttled.Until.UTC().Format(time.RFC3339)})
	default:
		p.logger.Warn("anilist request failed", "error", err)
		writeError(w, http.StatusBadGateway, "anilist_unavailable")
	}
}
