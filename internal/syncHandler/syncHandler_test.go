package synchandler

import (
	"fmt"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/clock"
	"github.com/ReidMason/plex-ani-sync/internal/logger"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

func TestGetUpdate(t *testing.T) {
	testCases := []struct {
		name     string
		seasons  []Season
		mappings []storage.Mapping
		expected []Update
	}{
		{
			name: "Completed series not in anime list",
			seasons: []Season{
				{
					Id: "1",
					Episodes: []Episode{
						{Id: "1", Watched: true},
						{Id: "2", Watched: true},
						{Id: "3", Watched: true},
						{Id: "4", Watched: true},
						{Id: "5", Watched: true},
						{Id: "6", Watched: true},
						{Id: "7", Watched: true},
						{Id: "8", Watched: true},
						{Id: "9", Watched: true},
						{Id: "10", Watched: true},
						{Id: "11", Watched: true},
						{Id: "12", Watched: true},
					},
				},
			},
			mappings: []storage.Mapping{
				{
					Id:                 1,
					AnimeId:            "1",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SeasonEpisodeEnd:   12,
				},
			},
			expected: []Update{
				{
					AnimeId:  "1",
					Status:   animeList.Completed,
					Progress: 12,
				},
			},
		},
		{
			name: "Two mappings for one season",
			seasons: []Season{
				{
					Id: "1",
					Episodes: []Episode{
						{Id: "1", Watched: true},
						{Id: "2", Watched: true},
						{Id: "3", Watched: true},
						{Id: "4", Watched: true},
						{Id: "5", Watched: true},
						{Id: "6", Watched: true},
						{Id: "7", Watched: true},
						{Id: "8", Watched: true},
						{Id: "9", Watched: true},
						{Id: "10", Watched: true},
						{Id: "11", Watched: true},
						{Id: "12", Watched: true},
					},
				},
			},
			mappings: []storage.Mapping{
				{
					Id:                 1,
					AnimeId:            "1",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 1,
					SeasonEpisodeEnd:   6,
				},
				{
					Id:                 2,
					AnimeId:            "2",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 7,
					SeasonEpisodeEnd:   12,
				},
			},
			expected: []Update{
				{
					AnimeId:  "1",
					Status:   animeList.Completed,
					Progress: 6,
				},
				{
					AnimeId:  "2",
					Status:   animeList.Completed,
					Progress: 6,
				},
			},
		},
		{
			name: "Two mappings for one season, season two watching",
			seasons: []Season{
				{
					Id: "1",
					Episodes: []Episode{
						{Id: "1", Watched: true},
						{Id: "2", Watched: true},
						{Id: "3", Watched: true},
						{Id: "4", Watched: true},
						{Id: "5", Watched: true},
						{Id: "6", Watched: true},
						{Id: "7", Watched: true},
						{Id: "8", Watched: true},
						{Id: "9", Watched: true, LastWatched: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)},
						{Id: "10", Watched: false},
						{Id: "11", Watched: false},
						{Id: "12", Watched: false},
					},
				},
			},
			mappings: []storage.Mapping{
				{
					Id:                 1,
					AnimeId:            "1",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 1,
					SeasonEpisodeEnd:   6,
				},
				{
					Id:                 2,
					AnimeId:            "2",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 7,
					SeasonEpisodeEnd:   12,
				},
			},
			expected: []Update{
				{
					AnimeId:  "1",
					Status:   animeList.Completed,
					Progress: 6,
				},
				{
					AnimeId:  "2",
					Status:   animeList.Current,
					Progress: 3,
				},
			},
		},
		{
			name: "Two mappings for one season, season two dropped",
			seasons: []Season{
				{
					Id: "1",
					Episodes: []Episode{
						{Id: "1", Watched: true},
						{Id: "2", Watched: true},
						{Id: "3", Watched: true},
						{Id: "4", Watched: true},
						{Id: "5", Watched: true},
						{Id: "6", Watched: true},
						{Id: "7", Watched: true},
						{Id: "8", Watched: true},
						{Id: "9", Watched: true, LastWatched: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
						{Id: "10", Watched: false},
						{Id: "11", Watched: false},
						{Id: "12", Watched: false},
					},
				},
			},
			mappings: []storage.Mapping{
				{
					Id:                 1,
					AnimeId:            "1",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 1,
					SeasonEpisodeEnd:   6,
				},
				{
					Id:                 2,
					AnimeId:            "2",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 7,
					SeasonEpisodeEnd:   12,
				},
			},
			expected: []Update{
				{
					AnimeId:  "1",
					Status:   animeList.Completed,
					Progress: 6,
				},
				{
					AnimeId:  "2",
					Status:   animeList.Dropped,
					Progress: 3,
				},
			},
		},
		{
			name: "Two mappings for one season, season two paused",
			seasons: []Season{
				{
					Id: "1",
					Episodes: []Episode{
						{Id: "1", Watched: true},
						{Id: "2", Watched: true},
						{Id: "3", Watched: true},
						{Id: "4", Watched: true},
						{Id: "5", Watched: true},
						{Id: "6", Watched: true},
						{Id: "7", Watched: true},
						{Id: "8", Watched: true},
						{Id: "9", Watched: true, LastWatched: time.Date(2024, 1, 24, 0, 0, 0, 0, time.UTC)},
						{Id: "10", Watched: false},
						{Id: "11", Watched: false},
						{Id: "12", Watched: false},
					},
				},
			},
			mappings: []storage.Mapping{
				{
					Id:                 1,
					AnimeId:            "1",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 1,
					SeasonEpisodeEnd:   6,
				},
				{
					Id:                 2,
					AnimeId:            "2",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    6,
					SeasonEpisodeStart: 7,
					SeasonEpisodeEnd:   12,
				},
			},
			expected: []Update{
				{
					AnimeId:  "1",
					Status:   animeList.Completed,
					Progress: 6,
				},
				{
					AnimeId:  "2",
					Status:   animeList.Paused,
					Progress: 3,
				},
			},
		},
	}

	mockLogger := logger.MockLogger{}
	mockTimeProvider := clock.NewMockClock(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	syncHandler := NewSyncHandler(mockLogger, mockTimeProvider)

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			t.Parallel()

			result := syncHandler.GetUpdate(tc.seasons, tc.mappings)

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d updates, got %d", len(tc.expected), len(result))
			}

			for i, update := range result {
				if update.AnimeId != tc.expected[i].AnimeId {
					t.Errorf("Expected AnimeId %s, got %s", tc.expected[i].AnimeId, update.AnimeId)
				}
				if update.Status != tc.expected[i].Status {
					t.Errorf("Expected Status %s, got %s", tc.expected[i].Status, update.Status)
				}
				if update.Progress != tc.expected[i].Progress {
					t.Errorf("Expected Progress %d, got %d", tc.expected[i].Progress, update.Progress)
				}
			}
		})
	}
}
