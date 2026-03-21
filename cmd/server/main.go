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
	"myapp/internal/port"
	"os"

	"github.com/joho/godotenv"
)

const cacheDir = "data"

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("loading .env: %v", err)
	}

	ctx := context.Background()

	mappingRepo := animelists.NewMappingRepository(cacheDir)
	mappingService := app.NewMappingService(mappingRepo)

	var plexRepo port.MediaHostRepository
	if url, token := os.Getenv("PLEX_URL"), os.Getenv("PLEX_TOKEN"); url != "" && token != "" {
		plexRepo = plex.NewRepository(url, token)
	} else {
		log.Println("PLEX_URL or PLEX_TOKEN not set — using mock Plex repository")
		plexRepo = plex.NewMockRepository()
	}

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
