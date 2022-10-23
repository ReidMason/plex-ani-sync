package synchandler

import (
	"plex-ani-sync/services/config"
	"plex-ani-sync/services/plex"
	"testing"
	"time"
)

type Test struct {
	ExpectedResult string
	Season         plex.Season
}

func TestWatchStatus(t *testing.T) {
	t.Parallel()
	configHandler := config.NewMock(config.GetDefaultConfig())
	cfg, _ := configHandler.GetConfig()

	testCases := []Test{
		{
			ExpectedResult: "Completed",
			Season: plex.Season{
				Episodes:        12,
				EpisodesWatched: 12,
				LastViewedAt:    time.Now().Unix(),
			},
		},
		{
			ExpectedResult: "Watching",
			Season: plex.Season{
				Episodes:        12,
				EpisodesWatched: 6,
				LastViewedAt:    time.Now().Unix(),
			},
		},
		{
			ExpectedResult: "Paused",
			Season: plex.Season{
				Episodes:        12,
				EpisodesWatched: 1,
				LastViewedAt:    time.Now().Unix() - int64(cfg.Sync.DaysUntilPaused+1)*86400,
			},
		},
		{
			ExpectedResult: "Dropped",
			Season: plex.Season{
				Episodes:        12,
				EpisodesWatched: 1,
				LastViewedAt:    time.Now().Unix() - int64(cfg.Sync.DaysUntilDropped+1)*86400,
			},
		},
	}

	syncHandler := NewSyncHandler(configHandler)
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.ExpectedResult, func(t *testing.T) {
			t.Parallel()
			if status := syncHandler.GetWatchStatus(tc.Season); status != tc.ExpectedResult {
				t.Fatalf("Wrong watch status for '%s' got '%s' insead", tc.ExpectedResult, status)
			} else {
				t.Logf("Correct watch status for '%s'", tc.ExpectedResult)
			}
		})
	}
}
