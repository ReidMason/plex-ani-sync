package port

import (
	"context"
	"myapp/internal/domain"
)

type MediaHostRepository interface {
	GetAnime(ctx context.Context) ([]domain.MediaHostAnime, error)
}
