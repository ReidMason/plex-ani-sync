package server

import (
	"net/http"
	"slices"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
)

type skippedJSON struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type syncPlanJSON struct {
	Changes []planChange  `json:"changes"`
	Skipped []skippedJSON `json:"skipped"`
}

func (p *plexLink) registerSync(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("POST /api/sync", a.authenticated(p.syncNow))
}

func (p *plexLink) syncNow(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	username, err := p.aniListUsername(r.Context(), s.UserID)
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	if username == nil {
		writeError(w, http.StatusConflict, "anilist_not_linked")
		return
	}
	if _, running := p.syncing.LoadOrStore(s.UserID, struct{}{}); running {
		writeError(w, http.StatusConflict, "sync_in_progress")
		return
	}
	defer p.syncing.Delete(s.UserID)

	list, err := p.anilistClient.UserList(r.Context(), *username)
	if err != nil {
		p.writeAniListError(w, err)
		return
	}
	shows, err := p.userShows(r, s.UserID)
	if err != nil {
		p.writeMediaError(w, err)
		return
	}
	current := map[int64]listState{}
	for _, e := range list {
		current[int64(e.MediaID)] = listState{Status: e.Status, Progress: e.Progress}
	}

	plan := syncPlanJSON{Changes: []planChange{}, Skipped: []skippedJSON{}}
	for _, show := range shows {
		if show.Status != statusMapped {
			plan.Skipped = append(plan.Skipped, skippedJSON{Title: show.Title, Status: show.Status, Reason: show.Reason})
		}
	}
	for _, f := range entryFactsOf(shows) {
		var cur *listState
		if state, ok := current[f.anilistID]; ok {
			cur = &state
		}
		if change, ok := planEntry(f, cur); ok {
			plan.Changes = append(plan.Changes, change)
		}
	}
	writeJSON(w, plan)
}

func entryFactsOf(shows []showJSON) []entryFacts {
	var out []entryFacts
	index := map[int64]int{}
	for _, show := range shows {
		if show.Status != statusMapped {
			continue
		}
		for _, season := range show.Seasons {
			for _, e := range season.Episodes {
				i := slices.IndexFunc(show.Mappings, func(m mappingJSON) bool { return m.covers(season.Number, e.Number) })
				if i < 0 {
					continue
				}
				id := show.Mappings[i].AniListID
				at, seen := index[id]
				if !seen {
					at = len(out)
					index[id] = at
					out = append(out, entryFacts{anilistID: id, title: show.Title})
				}
				if !e.Watched {
					continue
				}
				out[at].watched++
				if e.LastWatchedAt != nil && (out[at].lastWatched == nil || e.LastWatchedAt.After(*out[at].lastWatched)) {
					out[at].lastWatched = e.LastWatchedAt
				}
			}
		}
	}
	return out
}
