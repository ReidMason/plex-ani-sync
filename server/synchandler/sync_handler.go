package synchandler

import (
	"math"
	"plex-ani-sync/config"
	"plex-ani-sync/plex"
	"time"
)

type ISyncHandler interface {
	GetWatchStatus(season plex.Season) string
}

type SyncHandler struct {
	ConfigHandler config.IConfigHandler
}

func NewSyncHandler(configHandler config.IConfigHandler) *SyncHandler {
	return &SyncHandler{ConfigHandler: configHandler}
}

func (sh *SyncHandler) GetWatchStatus(season plex.Season) string {
	if season.EpisodesWatched == season.Episodes {
		return "Completed"
	}

	config, _ := sh.ConfigHandler.GetConfig()
	now := time.Now().Unix()
	daysSinceLastWatched := int64(math.Abs(float64((season.LastViewedAt - now) / 86400)))

	if season.EpisodesWatched > 0 && daysSinceLastWatched > int64(config.Sync.DaysUntilDropped) {
		return "Dropped"
	}

	if season.EpisodesWatched > 0 && daysSinceLastWatched > int64(config.Sync.DaysUntilPaused) {
		return "Paused"
	}

	if season.EpisodesWatched != 0 {
		return "Watching"
	}

	return "Planning"
}
