package mapping

import (
	"plex-ani-sync/filehandler"
	"plex-ani-sync/utils"
)

type Mappings = []Mapping

type IMapper interface {
	GetAnilistMapping(seasonRatingKey string) (Mapping, error)
}

type mapper struct {
	FileHandler filehandler.FileSystem[Mappings]
}

var _ IMapper = (*mapper)(nil)

func New(fh filehandler.FileSystem[Mappings]) *mapper {
	return &mapper{FileHandler: fh}
}

func (m mapper) GetAnilistMapping(seasonRatingKey string) (Mapping, error) {
	allMappings, err := m.LoadMapping()

	if err != nil {
		return Mapping{}, err
	}

	return utils.GetSliceItem(allMappings, func(x Mapping) bool { return x.PlexSeasonRatingKey == seasonRatingKey })
}

func (m mapper) LoadMapping() (Mappings, error) {
	filePath := "data/mapping.json"
	_, err := m.FileHandler.EnsureFileExists(filePath, Mappings{})
	if err != nil {
		return []Mapping{}, nil
	}

	return m.FileHandler.LoadJsonFile(filePath)
}

func (m mapper) SaveMapping(data Mappings) error {
	return m.FileHandler.SaveJson("data/mapping.json", data)
}

type Mapping struct {
	PlexSeasonRatingKey string
	AnilistId           string
	TvdbId              string
	Ignored             bool
	EpisodeStart        int
	SeasonLength        int
}
