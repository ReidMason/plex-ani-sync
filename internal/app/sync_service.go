package app

import (
	"context"
	"errors"
	"fmt"
	"myapp/internal/domain"
	"myapp/internal/port"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SyncConfig holds the thresholds used when determining paused/dropped status.
type SyncConfig struct {
	// PausedAfter is how long since the last watched episode before a
	// show transitions from in_progress to paused.
	PausedAfter time.Duration
	// DroppedAfter is how long since the last watched episode before a
	// show transitions from in_progress (or paused) to dropped.
	DroppedAfter time.Duration
}

// DefaultSyncConfig returns sensible defaults: paused after 2 weeks,
// dropped after 30 days.
func DefaultSyncConfig() SyncConfig {
	return SyncConfig{
		PausedAfter:  14 * 24 * time.Hour,
		DroppedAfter: 30 * 24 * time.Hour,
	}
}

type SyncService struct {
	animeListRepo  port.AnimeListRepository
	mediaHostRepo  port.MediaHostRepository
	mappingService *MappingService
	cfg            SyncConfig
	now            func() time.Time
}

func NewSyncService(animeListRepo port.AnimeListRepository, mediaHostRepo port.MediaHostRepository, mappingService *MappingService, cfg SyncConfig) *SyncService {
	return &SyncService{
		animeListRepo:  animeListRepo,
		mediaHostRepo:  mediaHostRepo,
		mappingService: mappingService,
		cfg:            cfg,
		now:            time.Now,
	}
}

func (s *SyncService) SyncAnime(ctx context.Context) ([]domain.AnimeStatus, error) {
	anime, err := s.mediaHostRepo.GetAnime(ctx)
	if err != nil {
		return nil, err
	}

	byAnilist := make(map[domain.AniListID][]domain.AnimeStatus)
	var anilistGlobalOrder []domain.AniListID

	for _, a := range anime {
		mappings, err := s.mappingService.GetMapping(ctx, a.ID)
		if errors.Is(err, ErrNoMappingFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		sorted := append([]domain.Mapping(nil), mappings...)
		sort.SliceStable(sorted, func(i, j int) bool {
			return mappingSortLess(sorted[i], sorted[j])
		})

		// Several Anime-List rows (per season / cour) may point at the same AniList
		// ID for one TVDB series — merge slices so we emit one status per AniList entry.
		merged := make(map[domain.AniListID][]domain.MediaHostEpisode)
		var anilistOrder []domain.AniListID

		for i, mapping := range sorted {
			var episodes []domain.MediaHostEpisode
			if mapping.TvDbSeason == "a" {
				episodes = flattenEpisodes(a)
			} else {
				episodes = seasonEpisodes(a, mapping.TvDbSeason)
			}

			offset := int(mapping.EpisodeOffset)
			count := episodeSpanForPlex(sorted, i, len(episodes))
			if count <= 0 {
				count = mapping.EpisodeCount
			}
			if offset < 0 {
				continue
			}
			if offset+count > len(episodes) {
				count = len(episodes) - offset
			}
			if count <= 0 || offset+count > len(episodes) {
				continue
			}
			slice := episodes[offset : offset+count]
			id := mapping.AnilistId
			if _, ok := merged[id]; !ok {
				anilistOrder = append(anilistOrder, id)
			}
			merged[id] = append(merged[id], slice...)
		}

		for _, id := range anilistOrder {
			eps := merged[id]
			if len(eps) == 0 {
				continue
			}
			st := domain.AnimeStatus{
				AnilistId:       id,
				PlexTitle:       string(a.Title),
				Status:          watchStatus(eps, s.now(), s.cfg),
				WatchedEpisodes: countWatched(eps),
				TotalEpisodes:   len(eps),
			}
			if _, ok := byAnilist[id]; !ok {
				anilistGlobalOrder = append(anilistGlobalOrder, id)
			}
			byAnilist[id] = append(byAnilist[id], st)
		}
	}

	statuses := make([]domain.AnimeStatus, 0, len(anilistGlobalOrder))
	for _, id := range anilistGlobalOrder {
		statuses = append(statuses, mergeAnimeStatusesForAnilist(id, byAnilist[id]))
	}
	return collapseDuplicatePlexTitles(statuses), nil
}

// collapseDuplicatePlexTitles merges rows that share the same Plex show title
// (trimmed, case-insensitive). Different TheTVDB matches often map to different
// AniDB rows and thus different AniList ids for the same library title; we keep
// one row: prefer the copy with the most episodes watched (complete library over
// a sparse duplicate), then wider mapping scope, then smallest AniList id.
// Empty titles are not merged with each other.
func collapseDuplicatePlexTitles(statuses []domain.AnimeStatus) []domain.AnimeStatus {
	if len(statuses) <= 1 {
		return statuses
	}
	var keys []string
	groups := make(map[string][]domain.AnimeStatus)
	for _, st := range statuses {
		k := plexTitleMergeKey(st)
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], st)
	}
	out := make([]domain.AnimeStatus, 0, len(keys))
	for _, k := range keys {
		parts := groups[k]
		if len(parts) == 1 {
			out = append(out, parts[0])
			continue
		}
		out = append(out, pickBestAnimeStatusForDuplicateTitle(parts))
	}
	return out
}

func plexTitleMergeKey(st domain.AnimeStatus) string {
	t := strings.TrimSpace(strings.ToLower(string(st.PlexTitle)))
	if t == "" {
		return "\x00" + string(st.AnilistId)
	}
	return t
}

func pickBestAnimeStatusForDuplicateTitle(parts []domain.AnimeStatus) domain.AnimeStatus {
	best := parts[0]
	for _, p := range parts[1:] {
		if animeStatusCoversMoreThan(p, best) {
			best = p
		}
	}
	return best
}

// animeStatusCoversMoreThan picks the Plex copy to keep when titles collide.
// Prefer more episodes actually watched (complete library over a sparse duplicate),
// then wider mapping scope, then stable AniList id.
func animeStatusCoversMoreThan(a, b domain.AnimeStatus) bool {
	if a.WatchedEpisodes != b.WatchedEpisodes {
		return a.WatchedEpisodes > b.WatchedEpisodes
	}
	if a.TotalEpisodes != b.TotalEpisodes {
		return a.TotalEpisodes > b.TotalEpisodes
	}
	return a.AnilistId < b.AnilistId
}

// mergeAnimeStatusesForAnilist folds multiple Plex library items (different
// TheTVDB ids can still map to the same AniList id) into one row. Identical
// watched/total across parts means duplicate listings — keep one. Otherwise sum
// counts (disjoint splits) and derive status from totals (paused/dropped not preserved).
func mergeAnimeStatusesForAnilist(id domain.AniListID, parts []domain.AnimeStatus) domain.AnimeStatus {
	if len(parts) == 0 {
		return domain.AnimeStatus{AnilistId: id}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	t0, w0 := parts[0].TotalEpisodes, parts[0].WatchedEpisodes
	allSame := true
	for _, p := range parts[1:] {
		if p.TotalEpisodes != t0 || p.WatchedEpisodes != w0 {
			allSame = false
			break
		}
	}
	if allSame {
		return parts[0]
	}
	w, t := 0, 0
	title := parts[0].PlexTitle
	for _, p := range parts {
		w += p.WatchedEpisodes
		t += p.TotalEpisodes
	}
	return domain.AnimeStatus{
		AnilistId:       id,
		PlexTitle:       title,
		Status:          watchStatusFromCounts(w, t),
		WatchedEpisodes: w,
		TotalEpisodes:   t,
	}
}

func watchStatusFromCounts(watched, total int) domain.WatchStatus {
	if total <= 0 {
		return domain.WatchStatusNotStarted
	}
	if watched >= total {
		return domain.WatchStatusCompleted
	}
	if watched == 0 {
		return domain.WatchStatusNotStarted
	}
	return domain.WatchStatusInProgress
}

// mappingSortLess orders mappings so same-TvDB-season rows sit together by
// episode offset (matches Anime-Lists / ScudLee conventions).
func mappingSortLess(a, b domain.Mapping) bool {
	sa, sb := a.TvDbSeason, b.TvDbSeason
	if sa != sb {
		na, ea := strconv.Atoi(sa)
		nb, eb := strconv.Atoi(sb)
		if ea == nil && eb == nil {
			return na < nb
		}
		return sa < sb
	}
	return a.EpisodeOffset < b.EpisodeOffset
}

// episodeSpanForPlex is how many consecutive Plex episodes belong to one
// AniDB/AniList slice for the same TvDB season: from this row's episodeoffset
// up to (but not including) the next row's offset, or the rest of the season.
// Anime-Lists encodes TV splits this way; the anime-offline-database "episodes"
// field can be wrong (e.g. 1 for a full cour), so we prefer span when > 0.
func episodeSpanForPlex(sorted []domain.Mapping, index int, episodeListLen int) int {
	m := sorted[index]
	off := int(m.EpisodeOffset)
	if off < 0 || off >= episodeListLen {
		return 0
	}
	for j := index + 1; j < len(sorted); j++ {
		if sorted[j].TvDbSeason != m.TvDbSeason {
			continue
		}
		nextOff := int(sorted[j].EpisodeOffset)
		if nextOff > off {
			return nextOff - off
		}
	}
	return episodeListLen - off
}

// flattenEpisodes returns all episodes from a media host anime in ascending
// season then episode order, giving them an implicit absolute episode index.
func flattenEpisodes(anime domain.MediaHostAnime) []domain.MediaHostEpisode {
	seasons := make([]domain.MediaHostSeason, len(anime.Seasons))
	copy(seasons, anime.Seasons)
	sort.Slice(seasons, func(i, j int) bool {
		return seasons[i].Number < seasons[j].Number
	})

	var episodes []domain.MediaHostEpisode
	for _, season := range seasons {
		eps := make([]domain.MediaHostEpisode, len(season.Episodes))
		copy(eps, season.Episodes)
		sort.Slice(eps, func(i, j int) bool {
			return eps[i].Number < eps[j].Number
		})
		episodes = append(episodes, eps...)
	}
	return episodes
}

// CompareWithAniList fetches the current AniList list (status + progress) and returns a
// SyncResult for every Plex entry that is in_progress, paused, dropped, or
// completed — except entries already marked completed on AniList (those are
// up to date and need no action). The AniList title is preferred; the Plex
// title is used as a fallback when the entry has no AniList title.
func (s *SyncService) CompareWithAniList(ctx context.Context, plexStatuses []domain.AnimeStatus) ([]domain.SyncResult, error) {
	entries, err := s.animeListRepo.GetAnimeList(ctx)
	if err != nil {
		return nil, err
	}

	type anilistEntry struct {
		title    string
		status   domain.WatchStatus
		progress int
	}
	anilistByID := make(map[domain.AniListID]anilistEntry, len(entries))
	for _, e := range entries {
		anilistByID[e.AnilistId] = anilistEntry{title: e.Title, status: e.Status, progress: e.Progress}
	}

	var results []domain.SyncResult
	for _, ps := range plexStatuses {
		switch ps.Status {
		case domain.WatchStatusInProgress, domain.WatchStatusPaused,
			domain.WatchStatusDropped, domain.WatchStatusCompleted:
		default:
			continue
		}

		al := anilistByID[ps.AnilistId]

		// Already fully up to date on AniList — nothing to do.
		if al.status == domain.WatchStatusCompleted {
			continue
		}

		title := al.title
		if title == "" {
			title = ps.PlexTitle
		}

		results = append(results, domain.SyncResult{
			AnilistId:           ps.AnilistId,
			Title:               title,
			PlexStatus:          ps.Status,
			AnilistStatus:       al.status,
			AnilistProgress:     al.progress,
			PlexWatchedEpisodes: ps.WatchedEpisodes,
			TotalEpisodes:       ps.TotalEpisodes,
		})
	}

	return results, nil
}

// ApplyAniListUpdates runs SaveMediaListEntry for every result that
// NeedsAniListChange, using Plex-derived status and TargetWatchedForAniList.
// Returns how many saves succeeded; failures are joined into the returned error.
func (s *SyncService) ApplyAniListUpdates(ctx context.Context, results []domain.SyncResult) (applied int, err error) {
	var errs []error
	for _, r := range results {
		if !r.NeedsAniListChange() {
			continue
		}
		if err := s.animeListRepo.SaveAnimeListEntry(ctx, r.AnilistId, r.PlexStatus, r.TargetWatchedForAniList()); err != nil {
			errs = append(errs, fmt.Errorf("%s (AniList %s): %w", r.Title, r.AnilistId, err))
			continue
		}
		applied++
	}
	return applied, errors.Join(errs...)
}

// seasonEpisodes returns the sorted episodes from the TvDB season identified by
// the season string (e.g. "1", "0"). Returns nil if the season is not present.
func seasonEpisodes(anime domain.MediaHostAnime, season string) []domain.MediaHostEpisode {
	seasonNum, err := strconv.Atoi(season)
	if err != nil {
		return nil
	}
	for _, s := range anime.Seasons {
		if int(s.Number) == seasonNum {
			eps := make([]domain.MediaHostEpisode, len(s.Episodes))
			copy(eps, s.Episodes)
			sort.Slice(eps, func(i, j int) bool {
				return eps[i].Number < eps[j].Number
			})
			return eps
		}
	}
	return nil
}

// watchStatus derives the watch status for a slice of episodes. If some but
// not all episodes are watched, the last-watched timestamp is used to
// determine whether the show is in_progress, paused, or dropped.
func countWatched(episodes []domain.MediaHostEpisode) int {
	var n int
	for _, ep := range episodes {
		if ep.Watched {
			n++
		}
	}
	return n
}

func watchStatus(episodes []domain.MediaHostEpisode, now time.Time, cfg SyncConfig) domain.WatchStatus {
	var watched int
	var lastWatched *time.Time
	for _, ep := range episodes {
		if ep.Watched {
			watched++
			if ep.LastWatchedAt != nil {
				if lastWatched == nil || ep.LastWatchedAt.After(*lastWatched) {
					lastWatched = ep.LastWatchedAt
				}
			}
		}
	}

	switch {
	case watched == len(episodes):
		return domain.WatchStatusCompleted
	case watched > 0:
		if lastWatched != nil {
			since := now.Sub(*lastWatched)
			if since >= cfg.DroppedAfter {
				return domain.WatchStatusDropped
			}
			if since >= cfg.PausedAfter {
				return domain.WatchStatusPaused
			}
		}
		return domain.WatchStatusInProgress
	default:
		return domain.WatchStatusNotStarted
	}
}
