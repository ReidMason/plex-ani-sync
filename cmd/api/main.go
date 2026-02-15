package main

import (
	"fmt"
	"log"

	"github.com/ReidMason/plex-ani-sync/internal/application"
	"github.com/ReidMason/plex-ani-sync/internal/service"
)

func main() {
	fmt.Println("Plex-Ani-Sync Starting...")
	fmt.Println()

	// Wire up dependencies
	plexService := service.NewPlexService()
	getAnimeListUseCase := application.NewGetAnimeListUseCase(plexService)

	// Execute use case
	animeList, err := getAnimeListUseCase.Execute()
	if err != nil {
		log.Fatalf("Failed to get anime list: %v", err)
	}

	fmt.Printf("Found %d anime:\n", len(animeList))
	for _, anime := range animeList {
		fmt.Printf("  [%s] %s\n", anime.GetID(), anime.GetTitle())
	}
}
