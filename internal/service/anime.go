package service

import "github.com/ReidMason/plex-ani-sync/internal/domain"

func GetAnime() domain.Anime {
	return domain.NewAnime("1", "Naruto")
}
