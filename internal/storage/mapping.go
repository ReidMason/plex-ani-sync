package storage

import (
	"context"
	"database/sql"
	"errors"

	sqlite3Storage "github.com/ReidMason/plex-ani-sync/internal/storage/sqlite3"
	"golang.org/x/exp/slices"
)

type MappingStorage interface {
	GetMappings(seasonId string) ([]Mapping, error)
	SetMappings(newMappings []Mapping) error
}

func (s Sqlite) GetMappings(seasonId string) ([]Mapping, error) {
	ctx := context.Background()
	results, err := s.queries.GetMappings(ctx, seasonId)
	if err == sql.ErrNoRows {
		return make([]Mapping, 0), nil
	} else if err != nil {
		return nil, err
	}

	mappings := make([]Mapping, len(results))
	for i, result := range results {
		mappings[i] = Mapping{
			Id:                 int(result.ID),
			AnimeId:            result.AnimeID,
			SeasonId:           result.SeasonID,
			AnimeEpisodeStart:  int(result.AnimeEpisodeStart),
			AnimeEpisodeEnd:    int(result.AnimeEpisodeEnd),
			SeasonEpisodeStart: int(result.SeasonEpisodeStart),
			SesasonEpisodeEnd:  int(result.SeasonEpisodeEnd),
		}
	}

	return mappings, nil
}

func (s Sqlite) SetMappings(newMappings []Mapping) error {
	ctx := context.Background()

	if len(newMappings) == 0 {
		return nil
	}

	mappingSeason := newMappings[0].SeasonId
	for _, mapping := range newMappings {
		if mapping.SeasonId != mappingSeason {
			return errors.New("All mappings must be for the same season")
		}
	}

	existingMappings, err := s.GetMappings(mappingSeason)
	if err != nil {
		return err
	}

	mappingsIdsToRemove := make([]int, 0)
	for _, existingMapping := range existingMappings {
		mappingExistsInNewMappings := slices.ContainsFunc(newMappings, func(newMapping Mapping) bool {
			return existingMapping.AnimeId == newMapping.AnimeId && existingMapping.SeasonId == newMapping.SeasonId
		})

		if !mappingExistsInNewMappings {
			mappingsIdsToRemove = append(mappingsIdsToRemove, existingMapping.Id)
		}
	}

	for _, mappingId := range mappingsIdsToRemove {
		err := s.queries.DeleteMapping(ctx, int64(mappingId))
		if err != nil {
			return err
		}
	}

	for _, mapping := range newMappings {
		err := s.queries.AddMapping(ctx, sqlite3Storage.AddMappingParams{
			AnimeID:            mapping.AnimeId,
			AnimeEpisodeStart:  int64(mapping.AnimeEpisodeStart),
			AnimeEpisodeEnd:    int64(mapping.AnimeEpisodeEnd),
			SeasonID:           mapping.SeasonId,
			SeasonEpisodeStart: int64(mapping.SeasonEpisodeStart),
			SeasonEpisodeEnd:   int64(mapping.SesasonEpisodeEnd),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

type Mapping struct {
	Id                 int
	AnimeId            string
	SeasonId           string
	AnimeEpisodeStart  int
	AnimeEpisodeEnd    int
	SeasonEpisodeStart int
	SesasonEpisodeEnd  int
}
