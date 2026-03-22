package app

import (
	"context"
	"errors"
	"myapp/internal/domain"
	"myapp/internal/port"
	"sort"
	"strconv"
)

type SyncService struct {
	animeListRepo  port.AnimeListRepository
	mediaHostRepo  port.MediaHostRepository
	mappingService *MappingService
}

func NewSyncService(animeListRepo port.AnimeListRepository, mediaHostRepo port.MediaHostRepository, mappingService *MappingService) *SyncService {
	return &SyncService{animeListRepo: animeListRepo, mediaHostRepo: mediaHostRepo, mappingService: mappingService}
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
			statuses = append(statuses, domain.AnimeStatus{
				AnilistId: mapping.AnilistId,
				Status:    watchStatus(episodes[offset : offset+count]),
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

// CompareWithAniList fetches the current AniList statuses and returns a
// SyncResult for every Plex entry that is in_progress or completed,
// paired with what AniList currently has (empty string if not on the list).
func (s *SyncService) CompareWithAniList(ctx context.Context, plexStatuses []domain.AnimeStatus) ([]domain.SyncResult, error) {
	entries, err := s.animeListRepo.GetAnimeList(ctx)
	if err != nil {
		return nil, err
	}

	type anilistEntry struct {
		title  string
		status domain.WatchStatus
	}
	anilistByID := make(map[domain.AniListID]anilistEntry, len(entries))
	for _, e := range entries {
		anilistByID[e.AnilistId] = anilistEntry{title: e.Title, status: e.Status}
	}

	var results []domain.SyncResult
	for _, ps := range plexStatuses {
		if ps.Status != domain.WatchStatusInProgress && ps.Status != domain.WatchStatusCompleted {
			continue
		}
		al := anilistByID[ps.AnilistId]
		results = append(results, domain.SyncResult{
			AnilistId:     ps.AnilistId,
			Title:         al.title,
			PlexStatus:    ps.Status,
			AnilistStatus: al.status,
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

func watchStatus(episodes []domain.MediaHostEpisode) domain.WatchStatus {
	watched := 0
	for _, ep := range episodes {
		if ep.Watched {
			watched++
		}
	}
	switch {
	case watched == len(episodes):
		return domain.WatchStatusCompleted
	case watched > 0:
		return domain.WatchStatusInProgress
	default:
		return domain.WatchStatusNotStarted
	}
}
