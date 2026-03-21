// Package main is the composition root. All dependency wiring happens here;
// no other package is allowed to perform DI.
package main

import (
	"context"
	"fmt"
	"log"
	"myapp/internal/adapter/animelists"
	"myapp/internal/adapter/plex"
	"myapp/internal/app"
)

const cacheDir = "data"

func main() {
	ctx := context.Background()

	mappingRepo := animelists.NewMappingRepository(cacheDir)
	mappingService := app.NewMappingService(mappingRepo)

	plexRepo := plex.NewMockRepository()

	// TODO: wire port.AnimeListRepository (AniList adapter)
	syncService := app.NewSyncService(nil, plexRepo, mappingService)

	statuses, err := syncService.SyncAnime(ctx)
	if err != nil {
		log.Fatalf("sync failed: %v", err)
	}

	for _, s := range statuses {
		fmt.Printf("anilist=%s status=%s\n", s.AnilistId, s.Status)
	}
}
