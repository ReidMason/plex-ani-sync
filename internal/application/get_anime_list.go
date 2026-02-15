package application

import "github.com/ReidMason/plex-ani-sync/internal/domain"

// GetAnimeListUseCase handles the business logic for retrieving anime list
type GetAnimeListUseCase struct {
	mediaProvider MediaProvider
}

// MediaProvider defines the interface for external media sources
type MediaProvider interface {
	FetchAnimeList() ([]domain.Anime, error)
}

func NewGetAnimeListUseCase(provider MediaProvider) *GetAnimeListUseCase {
	return &GetAnimeListUseCase{
		mediaProvider: provider,
	}
}

// Execute runs the use case
func (uc *GetAnimeListUseCase) Execute() ([]domain.Anime, error) {
	// Business logic goes here (filtering, validation, etc.)
	animeList, err := uc.mediaProvider.FetchAnimeList()
	if err != nil {
		return nil, err
	}

	// Could add business rules here:
	// - Filter out certain anime
	// - Sort by criteria
	// - Validate data
	
	return animeList, nil
}
