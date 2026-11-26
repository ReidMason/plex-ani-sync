package server

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/media"
)

type episodeJSON struct {
	Number        int        `json:"number"`
	Title         string     `json:"title"`
	Watched       bool       `json:"watched"`
	LastWatchedAt *time.Time `json:"lastWatchedAt"`
}

type seasonJSON struct {
	Number   int           `json:"number"`
	Episodes []episodeJSON `json:"episodes"`
}

type showJSON struct {
	MediaSourceID int64        `json:"mediaSourceId"`
	Title         string       `json:"title"`
	Seasons       []seasonJSON `json:"seasons"`
}

func (p *plexLink) registerAnime(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("GET /api/anime", a.authenticated(p.anime))
}

func (p *plexLink) anime(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	rows, err := p.queries.ListAnimeLibrariesForUser(r.Context(), s.UserID)
	if err != nil {
		internalError(w, p.logger, err)
		return
	}
	out := []showJSON{}
	if len(rows) > 0 {
		token, err := p.accountToken(r, s.UserID)
		if err != nil {
			internalError(w, p.logger, err)
			return
		}
		for _, row := range rows {
			shows, err := p.sources[mediaKind(row.Kind)].Shows(r.Context(), media.Connection{URL: row.Url, Token: token}, row.LibraryKey)
			switch {
			case errors.Is(err, media.ErrUnreachable):
				writeError(w, http.StatusBadGateway, "unreachable")
				return
			case errors.Is(err, media.ErrNoAccess):
				writeError(w, http.StatusForbidden, "no_access")
				return
			case err != nil:
				internalError(w, p.logger, err)
				return
			}
			for _, show := range shows {
				out = append(out, toShowJSON(row.MediaSourceID, show))
			}
		}
	}
	slices.SortStableFunc(out, func(a, b showJSON) int { return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)) })
	writeJSON(w, out)
}

func toShowJSON(sourceID int64, show media.Show) showJSON {
	out := showJSON{MediaSourceID: sourceID, Title: show.Title, Seasons: []seasonJSON{}}
	for _, season := range show.Seasons {
		if season.Number == 0 {
			continue
		}
		sj := seasonJSON{Number: season.Number, Episodes: []episodeJSON{}}
		for _, e := range season.Episodes {
			sj.Episodes = append(sj.Episodes, episodeJSON{Number: e.Number, Title: e.Title, Watched: e.Watched, LastWatchedAt: e.LastWatchedAt})
		}
		out.Seasons = append(out.Seasons, sj)
	}
	return out
}
