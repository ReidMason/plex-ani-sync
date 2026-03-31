package app

import (
	"context"
	"errors"
	"myapp/internal/domain"
	"myapp/internal/port"
	"sort"
	"strconv"
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

	var statuses []domain.AnimeStatus
	for _, a := range anime {
		mappings, err := s.mappingService.GetMapping(ctx, a.ID)
		if errors.Is(err, ErrNoMappingFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		for _, mapping := range mappings {
			var episodes []domain.MediaHostEpisode
			if mapping.TvDbSeason == "a" {
				episodes = flattenEpisodes(a)
			} else {
				episodes = seasonEpisodes(a, mapping.TvDbSeason)
			}

			offset := int(mapping.EpisodeOffset)
			count := mapping.EpisodeCount
			if offset < 0 || offset+count > len(episodes) {
				continue
			}
			slice := episodes[offset : offset+count]
			statuses = append(statuses, domain.AnimeStatus{
				AnilistId:       mapping.AnilistId,
				PlexTitle:       string(a.Title),
				Status:          watchStatus(slice, s.now(), s.cfg),
				WatchedEpisodes: countWatched(slice),
				TotalEpisodes:   count,
			})
		}
	}

	return statuses, nil
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
