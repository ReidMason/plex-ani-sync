package server

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/media"
)

const specialsSeason = 0

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

type mappingJSON struct {
	Season            int   `json:"season"`
	FirstEpisode      int   `json:"firstEpisode"`
	LastEpisode       *int  `json:"lastEpisode"`
	AniListID         int64 `json:"anilistId"`
	EntryFirstEpisode int   `json:"entryFirstEpisode"`
	EntryLastEpisode  *int  `json:"entryLastEpisode"`
}

const (
	statusMapped   = "mapped"
	statusUnmapped = "unmapped"
)

type showJSON struct {
	MediaSourceID int64             `json:"mediaSourceId"`
	Title         string            `json:"title"`
	Status        string            `json:"status"`
	ExternalIDs   map[string]string `json:"externalIds"`
	Mappings      []mappingJSON     `json:"mappings"`
	Seasons       []seasonJSON      `json:"seasons"`

	tvdbID int64
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
		for _, sel := range groupSelections(rows) {
			shows, err := p.showsOf(r.Context(), sel, token)
			if err != nil {
				p.writeMediaError(w, err)
				return
			}
			for _, show := range shows {
				out = append(out, toShowJSON(sel.sourceID, show))
			}
		}
	}
	if err := p.attachMappings(r.Context(), out); err != nil {
		internalError(w, p.logger, err)
		return
	}
	slices.SortStableFunc(out, func(a, b showJSON) int { return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)) })
	writeJSON(w, out)
}

type selection struct {
	sourceID int64
	url      string
	kind     string
	keys     []string
}

func groupSelections(rows []dbgen.ListAnimeLibrariesForUserRow) []selection {
	var out []selection
	for _, row := range rows {
		if n := len(out); n == 0 || out[n-1].sourceID != row.MediaSourceID {
			out = append(out, selection{sourceID: row.MediaSourceID, url: row.Url, kind: row.Kind})
		}
		last := &out[len(out)-1]
		last.keys = append(last.keys, row.LibraryKey)
	}
	return out
}

// showsOf skips selected libraries that no longer exist on the server.
func (p *plexLink) showsOf(ctx context.Context, sel selection, token string) ([]media.Show, error) {
	conn := media.Connection{URL: sel.url, Token: token}
	existing, err := p.librariesOf(ctx, sel.kind, conn)
	if err != nil {
		return nil, err
	}
	var out []media.Show
	for _, key := range sel.keys {
		if !slices.ContainsFunc(existing, func(l media.Library) bool { return l.Key == key }) {
			continue
		}
		shows, err := p.sources[mediaKind(sel.kind)].Shows(ctx, conn, key)
		if err != nil {
			return nil, err
		}
		out = append(out, shows...)
	}
	return out, nil
}

func toShowJSON(sourceID int64, show media.Show) showJSON {
	out := showJSON{MediaSourceID: sourceID, Title: show.Title, Seasons: []seasonJSON{}, Mappings: []mappingJSON{}, Status: statusUnmapped, ExternalIDs: map[string]string{}}
	for key, id := range map[string]string{"tvdb": show.ExternalIDs.TVDB, "tmdb": show.ExternalIDs.TMDB, "imdb": show.ExternalIDs.IMDB} {
		if id != "" {
			out.ExternalIDs[key] = id
		}
	}
	out.tvdbID, _ = strconv.ParseInt(show.ExternalIDs.TVDB, 10, 64)
	for _, season := range show.Seasons {
		if season.Number == specialsSeason {
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

func (p *plexLink) attachMappings(ctx context.Context, shows []showJSON) error {
	var ids []int64
	for _, s := range shows {
		if s.tvdbID > 0 {
			ids = append(ids, s.tvdbID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := p.queries.ListMappingsForShows(ctx, ids)
	if err != nil {
		return err
	}
	for i := range shows {
		s := &shows[i]
		for _, row := range rows {
			if row.TvdbID != s.tvdbID || !slices.ContainsFunc(s.Seasons, func(se seasonJSON) bool { return se.Number == int(row.TvdbSeason) }) {
				continue
			}
			s.Mappings = append(s.Mappings, mappingJSON{
				Season: int(row.TvdbSeason), FirstEpisode: int(row.FirstEpisode), LastEpisode: optionalInt(row.LastEpisode),
				AniListID: row.AnilistID, EntryFirstEpisode: int(row.EntryFirstEpisode), EntryLastEpisode: optionalInt(row.EntryLastEpisode),
			})
		}
		if len(s.Mappings) > 0 {
			s.Status = statusMapped
		}
	}
	return nil
}

func optionalInt(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}
