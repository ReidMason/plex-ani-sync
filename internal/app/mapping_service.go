package app

import (
	"context"
	"errors"
	"myapp/internal/domain"
	"myapp/internal/port"
)

var ErrNoMappingFound = errors.New("no mapping found")

type MappingService struct {
	mappingSourceRepo port.MappingSourceRepository
}

func NewMappingService(mappingSourceRepo port.MappingSourceRepository) *MappingService {
	return &MappingService{mappingSourceRepo: mappingSourceRepo}
}

func (s *MappingService) GetMapping(ctx context.Context, tvDbID domain.TvDbID) ([]domain.Mapping, error) {
	aniDbMappings, err := s.mappingSourceRepo.GetTvDbToAniDbMapping(ctx)
	if err != nil {
		return nil, err
	}

	aniDbMapping := aniDbMappings[tvDbID]
	if len(aniDbMapping) == 0 {
		return nil, ErrNoMappingFound
	}

	aniDbToListIdMappings, err := s.mappingSourceRepo.GetAniDbToListIdMapping(ctx)
	if err != nil {
		return nil, err
	}

	animeMappings := make([]domain.Mapping, 0)
	for _, aniDbMapping := range aniDbMapping {
		aniDbToListIdMapping, ok := aniDbToListIdMappings[aniDbMapping.AniDbID]
		if !ok {
			continue
		}
		animeMappings = append(animeMappings, domain.Mapping{
			TvDbID:        tvDbID,
			AnilistId:     aniDbToListIdMapping.AnilistId,
			TvDbSeason:    aniDbMapping.TvDbSeason,
			EpisodeOffset: aniDbMapping.EpisodeOffset,
			EpisodeCount:  aniDbToListIdMapping.EpisodeCount,
		})
	}

	return animeMappings, nil
}
