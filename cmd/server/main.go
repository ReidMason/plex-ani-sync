// Package main is the composition root. All dependency wiring happens here;
// no other package is allowed to perform DI.
package main

import (
	"context"
	"fmt"
	"log"
	"myapp/internal/adapter/anilist"
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

	anilistToken := os.Getenv("ANILIST_TOKEN")
	if anilistToken == "" {
		log.Fatal("ANILIST_TOKEN not set — see internal/adapter/anilist/repository.go for how to obtain one")
	}
	anilistRepo := anilist.NewRepository(anilistToken)

	syncService := app.NewSyncService(anilistRepo, plexRepo, mappingService)

	plexStatuses, err := syncService.SyncAnime(ctx)
	if err != nil {
		log.Fatalf("fetching Plex statuses: %v", err)
	}

	results, err := syncService.CompareWithAniList(ctx, plexStatuses)
	if err != nil {
		log.Fatalf("comparing with AniList: %v", err)
	}

	needsUpdate := 0
	for _, r := range results {
		if r.PlexStatus != r.AnilistStatus {
			fmt.Printf("~ %-40s  plex=%-12s  anilist=%s\n", r.Title, r.PlexStatus, r.AnilistStatus)
			needsUpdate++
		}
	}
	fmt.Printf("\n%d/%d entries need updating\n", needsUpdate, len(results))
}
