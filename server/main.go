package main

import (
	"fmt"
	"log"
	"plex-ani-sync/config"
	"plex-ani-sync/plex"
	"plex-ani-sync/synchandler"
	"plex-ani-sync/utils"
	"sync"
	"time"
)

var wg sync.WaitGroup

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("function %s took %s", name, elapsed)
}

func main() {
	defer timeTrack(time.Now(), "Main")

	plexConnection := plex.New(config.NewConfigHandler(), plex.NewRequestHandler(config.NewConfigHandler()))
	syncHandler := synchandler.NewSyncHandler(config.NewConfigHandler())

	libraries, _ := plexConnection.GetAllLibraries()
	library, _ := utils.GetSliceItem(libraries, func(x plex.Library) bool { return x.Title == "Anime" })
	allSeries, _ := plexConnection.GetAllSeries(library.Key)

	for _, series := range allSeries {
		wg.Add(1)
		go showWatchStatus(series, *plexConnection, *syncHandler)
	}
	wg.Wait()
}

func showWatchStatus(series plex.Series, plexConnection plex.Connection, syncHandler synchandler.SyncHandler) {
	defer wg.Done()

	seasons, _ := plexConnection.GetSeasons(series.RatingKey)
	for _, season := range seasons {
		watchStatus := syncHandler.GetWatchStatus(season)
		fmt.Printf("%s %s - %s\n", series.Title, season.Title, watchStatus)
	}
}
