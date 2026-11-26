package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
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

func (m mappingJSON) covers(season, episode int) bool {
	return m.Season == season && episode >= m.FirstEpisode && (m.LastEpisode == nil || episode <= *m.LastEpisode)
}

const (
	statusMapped      = "mapped"
	statusNeedsReview = "needs_review"
	statusUnmapped    = "unmapped"
)

type uncoveredJSON struct {
	Season   int   `json:"season"`
	Episodes []int `json:"episodes"`
}

type showJSON struct {
	MediaSourceID int64             `json:"mediaSourceId"`
	Title         string            `json:"title"`
	Status        string            `json:"status"`
	Reason        string            `json:"reason"`
	ExternalIDs   map[string]string `json:"externalIds"`
	Mappings      []mappingJSON     `json:"mappings"`
	Seasons       []seasonJSON      `json:"seasons"`
	Uncovered     []uncoveredJSON   `json:"uncoveredEpisodes"`

	tvdbID int64
	tmdbID int64
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
		selections := groupSelections(rows)
		perSource := make([][]media.Show, len(selections))
		g, gctx := errgroup.WithContext(r.Context())
		for i, sel := range selections {
			g.Go(func() (err error) {
				perSource[i], err = p.listShows(gctx, sel, token)
				return err
			})
		}
		if err := g.Wait(); err != nil {
			p.writeMediaError(w, err)
			return
		}
		for i, sel := range selections {
			for _, show := range perSource[i] {
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

// Selected libraries that no longer exist on the server are skipped.
func (p *plexLink) listShows(ctx context.Context, sel selection, token string) ([]media.Show, error) {
	conn := media.Connection{URL: sel.url, Token: token}
	existing, err := p.librariesOf(ctx, sel.kind, conn)
	if err != nil {
		return nil, err
	}
	keys := slices.DeleteFunc(slices.Clone(sel.keys), func(key string) bool {
		return !slices.ContainsFunc(existing, func(l media.Library) bool { return l.Key == key })
	})
	perLibrary := make([][]media.Show, len(keys))
	g, gctx := errgroup.WithContext(ctx)
	for i, key := range keys {
		g.Go(func() (err error) {
			perLibrary[i], err = p.shows.get(gctx, sel.sourceID, key, func() ([]media.Show, error) {
				return p.sources[mediaKind(sel.kind)].Shows(gctx, conn, key)
			})
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return slices.Concat(perLibrary...), nil
}

func toShowJSON(sourceID int64, show media.Show) showJSON {
	out := showJSON{MediaSourceID: sourceID, Title: show.Title, Seasons: []seasonJSON{}, Mappings: []mappingJSON{}, Uncovered: []uncoveredJSON{}, Status: statusUnmapped, ExternalIDs: map[string]string{}}
	for key, id := range map[string]string{"tvdb": show.ExternalIDs.TVDB, "tmdb": show.ExternalIDs.TMDB, "imdb": show.ExternalIDs.IMDB} {
		if id != "" {
			out.ExternalIDs[key] = id
		}
	}
	out.tvdbID, _ = strconv.ParseInt(show.ExternalIDs.TVDB, 10, 64)
	out.tmdbID, _ = strconv.ParseInt(show.ExternalIDs.TMDB, 10, 64)
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
	if err := p.resolveTVDBIDs(ctx, shows); err != nil {
		return err
	}
	var ids []int64
	for _, s := range shows {
		if s.tvdbID > 0 {
			ids = append(ids, s.tvdbID)
		}
	}
	if len(ids) == 0 {
		for i := range shows {
			shows[i].Reason = reasonFor(shows[i])
		}
		return nil
	}
	rows, err := p.queries.ListMappingsForShows(ctx, ids)
	if err != nil {
		return err
	}
	entries := p.anilist.mediaFor(ctx, entryIDs(rows))
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
			s.Uncovered = uncoveredEpisodes(*s)
			reasons := countMismatches(*s, entries)
			if len(s.Uncovered) > 0 {
				reasons = append(reasons, "episodes not covered by any mapping: "+describeUncovered(s.Uncovered))
			}
			if titleMismatch(*s, entries) {
				reasons = append(reasons, "title looks very different from the mapped AniList entry's titles")
			}
			if len(reasons) > 0 {
				s.Status = statusNeedsReview
				s.Reason = strings.Join(reasons, "; ")
			}
		}
		if s.Status == statusUnmapped {
			s.Reason = reasonFor(*s)
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

func uncoveredEpisodes(s showJSON) []uncoveredJSON {
	out := []uncoveredJSON{}
	for _, season := range s.Seasons {
		var missing []int
		for _, e := range season.Episodes {
			covered := slices.ContainsFunc(s.Mappings, func(m mappingJSON) bool {
				return m.covers(season.Number, e.Number)
			})
			if !covered {
				missing = append(missing, e.Number)
			}
		}
		if len(missing) > 0 {
			out = append(out, uncoveredJSON{Season: season.Number, Episodes: missing})
		}
	}
	return out
}

func describeUncovered(u []uncoveredJSON) string {
	parts := make([]string, 0, len(u))
	for _, s := range u {
		nums := make([]string, len(s.Episodes))
		for i, n := range s.Episodes {
			nums[i] = strconv.Itoa(n)
		}
		parts = append(parts, "season "+strconv.Itoa(s.Season)+" episodes "+strings.Join(nums, ", "))
	}
	return strings.Join(parts, "; ")
}

func entryIDs(rows []dbgen.ListMappingsForShowsRow) []int {
	var ids []int
	for _, r := range rows {
		ids = append(ids, int(r.AnilistID))
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

func countMismatches(s showJSON, entries map[int]anilist.Media) []string {
	var out []string
	var seen []int64
	for _, m := range s.Mappings {
		if slices.Contains(seen, m.AniListID) {
			continue
		}
		seen = append(seen, m.AniListID)
		info, ok := entries[int(m.AniListID)]
		if !ok || info.Episodes == nil {
			continue
		}
		covered := 0
		for _, season := range s.Seasons {
			for _, e := range season.Episodes {
				if slices.ContainsFunc(s.Mappings, func(o mappingJSON) bool {
					return o.AniListID == m.AniListID && o.covers(season.Number, e.Number)
				}) {
					covered++
				}
			}
		}
		if covered != *info.Episodes {
			out = append(out, fmt.Sprintf("episode count mismatch: AniList entry %d has %d episodes but %d mapped episodes exist", m.AniListID, *info.Episodes, covered))
		}
	}
	return out
}

func titleMismatch(s showJSON, entries map[int]anilist.Media) bool {
	var mapped []anilist.Media
	for _, m := range s.Mappings {
		if info, ok := entries[int(m.AniListID)]; ok {
			mapped = append(mapped, info)
		}
	}
	return titlesDiffer(s.Title, mapped)
}

func reasonFor(s showJSON) string {
	if s.Status != statusUnmapped {
		return ""
	}
	if s.tvdbID <= 0 && s.tmdbID > 0 {
		return "No TVDB ID found for TMDB ID " + strconv.FormatInt(s.tmdbID, 10)
	}
	if s.tvdbID <= 0 {
		return "Plex has no TVDB ID for this show"
	}
	return "No mappings found for TVDB ID " + strconv.FormatInt(s.tvdbID, 10)
}

func (p *plexLink) resolveTVDBIDs(ctx context.Context, shows []showJSON) error {
	var tmdbIDs []int64
	for _, s := range shows {
		if s.tvdbID == 0 && s.tmdbID > 0 {
			tmdbIDs = append(tmdbIDs, s.tmdbID)
		}
	}
	if len(tmdbIDs) == 0 {
		return nil
	}
	links, err := p.queries.ListTVDBIDsForTMDB(ctx, tmdbIDs)
	if err != nil {
		return err
	}
	tvdbOf := map[int64]int64{}
	for _, l := range links {
		tvdbOf[l.TmdbID] = l.TvdbID
	}
	for i := range shows {
		if shows[i].tvdbID == 0 {
			shows[i].tvdbID = tvdbOf[shows[i].tmdbID]
		}
	}
	return nil
}
