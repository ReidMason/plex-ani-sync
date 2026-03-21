package port

import (
	"context"
	"myapp/internal/domain"
)

type AnimeListRepository interface {
	GetAnimeList(ctx context.Context) ([]domain.MediaHostAnime, error)
}
