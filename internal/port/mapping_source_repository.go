package port

import (
	"context"
	"myapp/internal/domain"
)

type MappingSourceRepository interface {
	GetTvDbToAniDbMapping(ctx context.Context) (map[domain.TvDbID][]domain.TvDbToAniDbMapping, error)
	GetAniDbToListIdMapping(ctx context.Context) (map[domain.AniDbID]domain.AniDbToListIdMapping, error)
}
