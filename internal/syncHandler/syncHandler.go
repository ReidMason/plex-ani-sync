package synchandler

import (
	"fmt"
	"time"

	"log/slog"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/mapping"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/charmbracelet/log"
	"golang.org/x/exp/slices"
)

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

func (s SyncHandler) Sync() {
	log.Info("Starting sync...")
	log.Info("Getting user")
	user, err := s.storageService.GetUser()
	if err != nil {
		log.Error("Failed to get user", slog.Any("error", err))
		return
	}

	log.Info("Getting libraries")
	libraries, err := s.mediaHostService.GetLibraries()
	if err != nil {
		log.Error("Failed to get libraries", slog.Any("error", err))
		return
	}

	var allSeries = make([]mediaHost.Series, 0)
	for _, library := range libraries {
		if slices.ContainsFunc(user.Libraries, func(userLibrary storage.Library) bool {
			return userLibrary.LibraryKey == library.Key
		}) {
			log.Info("Need to sync library", slog.String("library", library.Title))
			series, err := s.mediaHostService.GetSeries(library.Key)
			if err != nil {
				log.Error("Failed to get series", slog.Any("error", err))
				continue
			}
			allSeries = append(allSeries, series...)
		}
	}

	log.Info("Got all series", slog.Int("count", len(allSeries)))

	count := 0
	for _, series := range allSeries {
		if series.WatchedEpisodes == 0 {
			continue
		}

		count += 1
	}

	log.Info("Total series with watched episodes", slog.Int("count", count))
	// Now we have a list of all the series with watched episodes
	// We need to get all the sesasons and episodes for each series
	allSeasons := s.getAllSeasons(allSeries)

	// Now we need all the mappings for these seasons
	allMappings := make([]storage.Mapping, 0)
	for _, season := range allSeasons {
		mappings, err := s.storageService.GetMappings(season.Id)
		if err != nil {
			log.Error("Failed to get mappings", slog.Any("error", err))
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
		log.Error("No anime list token found")
		return
	}
	animeListUser, err := s.animeListService.GetCurrentUser(*user.AnimeListToken)
	if err != nil {
		log.Error("Failed to get anime list user", slog.Any("error", err))
		return
	}

	currentAnimeList, err := s.animeListService.GetAnimeList(animeListUser.Id)
	if err != nil {
		log.Error("Failed to get anime list", slog.Any("error", err))
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

type Update struct {
	Name           string
	AnimeId        animeList.AnimeId
	Status         animeList.Status
	Progress       int
	LastWatched    time.Time
	New            bool
	UpdateRequired bool
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
	log.Info("Getting seasons to update")
	allSeasons := make([]mediaHost.Season, 0)
	for _, series := range allSeries {
		seasons, err := s.mediaHostService.GetSeasons(series.Id)
		if err != nil {
			log.Error("Failed to get seasons", slog.Any("error", err))
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
