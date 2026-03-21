package plex

import (
	"context"
	"myapp/internal/domain"
)

// MockRepository is a stub implementation of port.MediaHostRepository for
// local development. Replace with a real Plex API client when ready.
type MockRepository struct{}

func NewMockRepository() *MockRepository {
	return &MockRepository{}
}

func (r *MockRepository) GetAnime(_ context.Context) ([]domain.MediaHostAnime, error) {
	// 24 episodes in one season: first 12 fully watched, episodes 23-24 not yet watched.
	episodes := make([]domain.MediaHostEpisode, 24)
	for i := range episodes {
		episodes[i] = domain.MediaHostEpisode{
			Number:  domain.MediaHostEpisodeNumber(i + 1),
			Watched: i < 22,
		}
	}

	return []domain.MediaHostAnime{
		{
			ID:    "295222",
			Title: "Mock Plex Anime",
			Seasons: []domain.MediaHostSeason{
				{Number: 1, Episodes: episodes},
			},
		},
	}, nil
}
