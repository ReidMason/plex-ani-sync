package app

import (
	"context"
	"errors"
	"myapp/internal/domain"
	"myapp/internal/port"
	"sort"
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

		episodes := flattenEpisodes(a)
		for _, mapping := range mappings {
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
