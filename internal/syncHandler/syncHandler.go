package synchandler

import (
	"sort"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/logger"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

type SyncHandlerInterface interface {
	GetUpdate(seasons []Season, mappings []storage.Mapping) []Update
}

type Update struct {
	AnimeId  animeList.AnimeId
	Status   animeList.Status
	Progress int
}

type SyncHandler struct {
	log  logger.Logger
	time TimeProvider
}

type Season struct {
	Id       string
	Episodes []Episode
}

type Episode struct {
	Id          string
	Watched     bool
	LastWatched time.Time
}

type TimeProvider interface {
	Now() time.Time
}

func NewSyncHandler(logger logger.Logger, timeProvider TimeProvider) *SyncHandler {
	return &SyncHandler{log: logger, time: timeProvider}
}

func (s SyncHandler) GetUpdate(seasons []Season, mappings []storage.Mapping) []Update {
	updates := make([]Update, 0)
	relevantMappings := make([]storage.Mapping, 0)

	// Get all the mappings for the seasons
	for _, season := range seasons {
		for _, mapping := range mappings {
			if mapping.SeasonId == season.Id {
				relevantMappings = append(relevantMappings, mapping)
			}
		}
	}

	// Group the mappings by anime
	goupedMappings := make(map[string][]storage.Mapping)
	for _, mapping := range relevantMappings {
		goupedMappings[mapping.AnimeId] = append(goupedMappings[mapping.AnimeId], mapping)
	}

	// Get the updates for each anime
	for animeId, mappings := range goupedMappings {
		updates = append(updates, s.getAnimeUpdate(animeList.AnimeId(animeId), mappings, seasons))
	}

	// Make sure the updates are ordered consistently
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].AnimeId < updates[j].AnimeId
	})

	return updates
}

func (s SyncHandler) getAnimeUpdate(animeId animeList.AnimeId, mappings []storage.Mapping, seasons []Season) Update {
	totalEpisodes := 0
	for _, mapping := range mappings {
		totalEpisodes += mapping.AnimeEpisodeEnd - mapping.AnimeEpisodeStart + 1
	}

	episodes := make([]Episode, 0)
	for _, season := range seasons {
		for _, mapping := range mappings {
			if mapping.SeasonId == season.Id {
				for i := mapping.SeasonEpisodeStart - 1; i < mapping.SeasonEpisodeEnd; i++ {
					episodes = append(episodes, season.Episodes[i])
				}
			}
		}
	}

	watchedEpisodes := 0
	lastWatched := time.Time{}
	for _, episode := range episodes {
		if episode.Watched {
			watchedEpisodes++
			if episode.LastWatched.After(lastWatched) {
				lastWatched = episode.LastWatched
			}
		}
	}

	status := animeList.Planning
	if watchedEpisodes == totalEpisodes {
		status = animeList.Completed
	} else if watchedEpisodes > 0 && lastWatched.Before(s.time.Now().AddDate(0, 0, -30)) {
		status = animeList.Dropped
	} else if watchedEpisodes > 0 && lastWatched.Before(s.time.Now().AddDate(0, 0, -7)) {
		status = animeList.Paused
	} else if watchedEpisodes > 0 {
		status = animeList.Current
	}

	return Update{
		AnimeId:  animeId,
		Status:   status,
		Progress: watchedEpisodes,
	}
}

func statusToWeighting(status animeList.Status) int {
	switch status {
	case animeList.Planning:
		return 0
	case animeList.Current:
		return 1
	case animeList.Completed:
		return 2
	default:
		return 0
	}
}
