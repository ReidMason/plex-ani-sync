package service

import "github.com/ReidMason/plex-ani-sync/internal/domain"

// PlexService implements the MediaProvider interface
type PlexService struct {
	// Will hold Plex client configuration
}

func NewPlexService() *PlexService {
	return &PlexService{}
}

// FetchAnimeList retrieves anime from Plex
// TODO: Replace with actual Plex API integration
func (p *PlexService) FetchAnimeList() ([]domain.Anime, error) {
	// Hardcoded for now - will be replaced with actual Plex API calls
	return []domain.Anime{
		domain.NewAnime("1", "Naruto"),
		domain.NewAnime("2", "One Piece"),
		domain.NewAnime("3", "Attack on Titan"),
		domain.NewAnime("4", "My Hero Academia"),
		domain.NewAnime("5", "Death Note"),
	}, nil
}
