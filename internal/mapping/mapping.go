package mapping

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/logger"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/ReidMason/plex-ani-sync/internal/utils"
)

type MappingFinder interface {
	CreateMappings(series mediaHost.Series) error
}

type AnimeMappingFinder struct {
	animeList animeList.AnimeList
	log       logger.Logger
}

func NewMappingFinder(animeList animeList.AnimeList, logger logger.Logger) *AnimeMappingFinder {
	if animeList == nil {
		panic("animeList is required")
	}

	if logger == nil {
		panic("logger is required")
	}

	return &AnimeMappingFinder{animeList: animeList, log: logger}
}

func (m AnimeMappingFinder) CreateMappingsForSeasons(title string, seasons []Season) ([]storage.Mapping, error) {
	mappings := make([]storage.Mapping, 0)
	title = cleanTitle(title)
	if len(seasons) == 0 || title == "" {
		return mappings, nil
	}

	results, err := m.animeList.SearchAnime(title)
	if err != nil {
		return nil, err
	}

	firstSeason := findFirstSeason(title, results)

	if firstSeason.Id == 0 {
		m.log.Info("No anime found", slog.String("title", title))
		return mappings, nil
	}

	totalEpisodes := 0
	for _, season := range seasons {
		totalEpisodes += season.Episodes
	}

	return m.findSequelMappings(firstSeason, seasons, totalEpisodes, mappings)
}

func (m AnimeMappingFinder) findSequelMappings(anime animeList.Anime, seasons []Season, totalEpisodes int, mappings []storage.Mapping) ([]storage.Mapping, error) {
	if anime.Id == 0 || len(seasons) == 0 {
		return mappings, nil
	}

	unmappedEpisodes := totalEpisodes
	for _, mapping := range mappings {
		unmappedEpisodes -= mapping.SesasonEpisodeEnd - mapping.SeasonEpisodeStart + 1
	}

	if unmappedEpisodes <= 0 {
		return mappings, nil
	}

	season, seasons := seasons[0], seasons[1:]

	animeEpisodeStart := season.Offset + 1
	animeEpisodeEnd := utils.Min(season.Offset+season.Episodes, anime.Episodes)

	seasonEpisodeStart := 1
	if season.partial {
		previousMapping := mappings[len(mappings)-1]
		seasonEpisodeStart = previousMapping.SesasonEpisodeEnd + 1
	}

	seasonEpisodeEnd := utils.Min(animeEpisodeEnd, season.Episodes)
	if season.partial {
		previousMapping := mappings[len(mappings)-1]
		seasonEpisodeEnd = previousMapping.SesasonEpisodeEnd + (animeEpisodeEnd - animeEpisodeStart) + 1
	}

	mappings = append(mappings, storage.Mapping{
		AnimeId:            fmt.Sprint(anime.Id),
		SeasonId:           season.Id,
		AnimeEpisodeStart:  animeEpisodeStart,
		AnimeEpisodeEnd:    animeEpisodeEnd,
		SeasonEpisodeStart: seasonEpisodeStart,
		SesasonEpisodeEnd:  seasonEpisodeEnd,
	})

	if season.Episodes-animeEpisodeEnd > 0 {
		season.Episodes = season.Episodes - animeEpisodeEnd
		season.partial = true
		seasons = append([]Season{season}, seasons...)
	}

	// There are more episdes in the anime than the season
	if anime.Episodes > animeEpisodeEnd {
		if len(seasons) > 0 {
			seasons[0].Offset = animeEpisodeEnd
		}
		return m.findSequelMappings(anime, seasons, totalEpisodes, mappings)
	}

	if anime.Sequel.Id == "" {
		return mappings, nil
	}

	sequel, err := m.animeList.GetAnime(fmt.Sprint(anime.Sequel.Id))
	if err != nil {
		return nil, err
	}

	return m.findSequelMappings(sequel, seasons, totalEpisodes, mappings)
}

func findFirstSeason(title string, results []animeList.Anime) animeList.Anime {
	for _, result := range results {
		if titlesMatch(result.Title, title) {
			return result
		}
	}

	for _, result := range results {
		for _, synonym := range result.Synonyms {
			if titlesMatch(synonym, title) {
				return result
			}
		}
	}

	return animeList.Anime{}
}

func titlesMatch(title1, title2 string) bool {
	return cleanTitle(title1) == cleanTitle(title2)
}

func cleanTitle(title string) string {
	removeChars := []string{"...", ":", "!", "?", "(TV)", ",", "’", "'", " TV", "-"}
	for _, char := range removeChars {
		title = strings.ReplaceAll(title, char, "")
	}

	return strings.ToLower(strings.TrimSpace(title))
}

type Season struct {
	Id       string
	Episodes int
	partial  bool
	Offset   int
}
