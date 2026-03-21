package animelists

import (
	"context"
	"myapp/internal/domain"
)

// MappingRepository composes XMLRepository and OfflineDBRepository to satisfy
// port.MappingSourceRepository. XMLRepository provides the TvDB→AniDB mapping
// and OfflineDBRepository provides the AniDB→AniList mapping.
type MappingRepository struct {
	xml       *XMLRepository
	offlineDB *OfflineDBRepository
}

func NewMappingRepository(cacheDir string) *MappingRepository {
	return &MappingRepository{
		xml:       NewXMLRepository(cacheDir),
		offlineDB: NewOfflineDBRepository(cacheDir),
	}
}

func (r *MappingRepository) GetTvDbToAniDbMapping(ctx context.Context) (map[domain.TvDbID][]domain.TvDbToAniDbMapping, error) {
	return r.xml.GetTvDbToAniDbMapping(ctx)
}

func (r *MappingRepository) GetAniDbToListIdMapping(ctx context.Context) (map[domain.AniDbID]domain.AniDbToListIdMapping, error) {
	return r.offlineDB.GetAniDbToListIdMapping(ctx)
}
