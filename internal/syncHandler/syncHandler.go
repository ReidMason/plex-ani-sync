package synchandler

import (
	"fmt"
	"sort"
	"time"

	"log/slog"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/logger"
	"github.com/ReidMason/plex-ani-sync/internal/mapping"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"golang.org/x/exp/slices"
)

type SyncHandlerInterface interface {
	GetUpdate(currentAnimeList []animeList.ListEntry, series mediaHost.Series, seasons mediaHost.Season) []UpdateV2
}

type UpdateV2 struct {
	AnimeId  animeList.AnimeId
	Status   animeList.Status
	Progress int
}

type SyncHandlerV2 struct {
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

func NewSyncHandlerV2(logger logger.Logger, timeProvider TimeProvider) *SyncHandlerV2 {
	return &SyncHandlerV2{log: logger, time: timeProvider}
}

func (s SyncHandlerV2) GetUpdate(currentAnimeList []animeList.ListEntry, seasons []Season, mappings []storage.Mapping) []UpdateV2 {
	updates := make([]UpdateV2, 0)
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

func (s SyncHandlerV2) getAnimeUpdate(animeId animeList.AnimeId, mappings []storage.Mapping, seasons []Season) UpdateV2 {
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

	fmt.Println("bleh", lastWatched)
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

	return UpdateV2{
		AnimeId:  animeId,
		Status:   status,
		Progress: watchedEpisodes,
	}
}

type SyncHandler struct {
	mediaHostService mediaHost.MediaHost
	storageService   *storage.Sqlite
	mappingFinder    mapping.MappingFinder
	animeListService animeList.AnimeList
	log              *slog.Logger
}

func NewSyncHandler(mediaHostService mediaHost.MediaHost, storageService *storage.Sqlite, mappingFinder mapping.MappingFinder, animeListService animeList.AnimeList, logger *slog.Logger) *SyncHandler {
	return &SyncHandler{mediaHostService: mediaHostService, storageService: storageService, mappingFinder: mappingFinder, animeListService: animeListService, log: logger}
}

type Update struct {
	Name           string
	AnimeId        animeList.AnimeId
	Status         animeList.Status
	Progress       int
	LastWatched    time.Time
	New            bool
	UpdateRequired bool
}

func (s SyncHandler) Sync() {
	s.log.Info("Starting sync...")
	s.log.Info("Getting user")
	user, err := s.storageService.GetUser()
	if err != nil {
		s.log.Error("Failed to get user", slog.Any("error", err))
		return
	}

	s.log.Info("Getting libraries")
	libraries, err := s.mediaHostService.GetLibraries()
	if err != nil {
		s.log.Error("Failed to get libraries", slog.Any("error", err))
		return
	}

	var allSeries = make([]mediaHost.Series, 0)
	for _, library := range libraries {
		if slices.ContainsFunc(user.Libraries, func(userLibrary storage.Library) bool {
			return userLibrary.LibraryKey == library.Key
		}) {
			s.log.Info("Need to sync library", slog.String("library", library.Title))
			series, err := s.mediaHostService.GetSeries(library.Key)
			if err != nil {
				s.log.Error("Failed to get series", slog.Any("error", err))
				continue
			}
			allSeries = append(allSeries, series...)
		}
	}

	// Now we have a list of all the series with watched episodes
	// We need to get all the sesasons and episodes for each series
	allSeasons := s.getAllSeasons(allSeries)

	// Now we need all the mappings for these seasons
	allMappings := make([]storage.Mapping, 0)
	for _, season := range allSeasons {
		mappings, err := s.storageService.GetMappings(season.Id)
		if err != nil {
			s.log.Error("Failed to get mappings", slog.Any("error", err))
			continue
		}

		allMappings = append(allMappings, mappings...)
	}

	// Now we have all the mappings
	// We need to find the number of watched episodes for each anime
	// So we go through the mappings and find the corresponding season and add the number of watched episodes to the map

	allUpdates := make(map[string]Update)
	for _, mapping := range allMappings {
		for _, season := range allSeasons {
			if mapping.SeasonId == season.Id {
				update, ok := allUpdates[mapping.AnimeId]
				if !ok {
					update = Update{
						Name:        season.Title,
						AnimeId:     animeList.AnimeId(mapping.AnimeId),
						Status:      animeList.Planning,
						Progress:    0,
						New:         false,
						LastWatched: season.LastViewedAt,
					}
				}

				update.Progress += season.WatchedEpisodes
				allUpdates[mapping.AnimeId] = update
				break
			}
		}
	}

	// Now we can compare to the anime list to find the watch status
	if user.AnimeListToken == nil {
		s.log.Error("No anime list token found")
		return
	}
	animeListUser, err := s.animeListService.GetCurrentUser(*user.AnimeListToken)
	if err != nil {
		s.log.Error("Failed to get anime list user", slog.Any("error", err))
		return
	}

	currentAnimeList, err := s.animeListService.GetAnimeList(animeListUser.Id)
	if err != nil {
		s.log.Error("Failed to get anime list", slog.Any("error", err))
		return
	}

	updates := make([]Update, 0)
	for _, update := range allUpdates {
		newUpdate := s.getUpdate(currentAnimeList, update)
		if newUpdate.UpdateRequired {
			updates = append(updates, newUpdate)
		}
	}

	for _, update := range updates {
		s.log.Info("Updating anime", slog.String("name", update.Name), slog.Any("animeId", update.AnimeId), slog.String("status", fmt.Sprint(update.Status)), slog.Int("progress", update.Progress), slog.Bool("new", update.New), slog.Time("LastWatched", update.LastWatched))
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

func (s SyncHandler) getUpdate(currentAnimeList []animeList.ListEntry, update Update) Update {
	found := false

	if update.Progress > 0 {
		update.Status = animeList.Current
	}

	for _, anime := range currentAnimeList {
		if anime.AnimeId == update.AnimeId {
			found = true

			sevenDaysAgo := time.Now().AddDate(0, 0, -7)
			if update.LastWatched.Before(sevenDaysAgo) {
				update.Status = animeList.Paused
			}

			thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
			if update.LastWatched.Before(thirtyDaysAgo) {
				update.Status = animeList.Dropped
			}

			if update.Progress == anime.TotalEpisodes {
				update.Status = animeList.Completed
			}

			if update.Status == animeList.Planning {
				return Update{}
			}

			changeRequired := update.Status != anime.Status && statusToWeighting(update.Status) > statusToWeighting(anime.Status)
			if !changeRequired || anime.Status == animeList.Dropped {
				return Update{}
			}

			update.UpdateRequired = true
			return update
		}
	}

	if update.Status == animeList.Planning {
		return Update{}
	}

	if !found {
		anime, err := s.animeListService.GetAnime(update.AnimeId)
		if err != nil {
			s.log.Error("Failed to get anime", slog.Any("error", err))
			return Update{}
		}

		sevenDaysAgo := time.Now().AddDate(0, 0, -7)
		if update.LastWatched.Before(sevenDaysAgo) {
			update.Status = animeList.Paused
		}

		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		if update.LastWatched.Before(thirtyDaysAgo) {
			update.Status = animeList.Dropped
		}

		if update.Progress == anime.Episodes {
			update.Status = animeList.Completed
		}

		update.UpdateRequired = true
		update.New = true
		return update
	}

	return update
}

func (s SyncHandler) getAllSeasons(allSeries []mediaHost.Series) []mediaHost.Season {
	s.log.Info("Getting seasons to update")
	allSeasons := make([]mediaHost.Season, 0)
	for _, series := range allSeries {
		seasons, err := s.mediaHostService.GetSeasons(series.Id)
		if err != nil {
			s.log.Error("Failed to get seasons", slog.Any("error", err))
			continue
		}

		for _, season := range seasons {
			if season.Index == 0 {
				continue
			}
			allSeasons = append(allSeasons, season)
		}
	}

	return allSeasons
}
