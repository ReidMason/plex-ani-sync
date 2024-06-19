package mapping

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/logger"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/ReidMason/plex-ani-sync/internal/utils"
	"golang.org/x/exp/slices"
)

type MappingFinder interface {
	CreateMappingsForSeasons(title string, seasons []Season) ([]storage.Mapping, error)
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

	firstSeason, err := m.findFirstSeason(title, results)
	if err != nil {
		return nil, err
	}

	if firstSeason.Id == "" {
		m.log.Info("No mappings found", slog.String("title", title))
		return mappings, nil
	}

	totalEpisodes := 0
	for _, season := range seasons {
		totalEpisodes += season.Episodes
	}

	return m.findSequelMappings(firstSeason, seasons, totalEpisodes, mappings)
}

func (m AnimeMappingFinder) findSequelMappings(anime animeList.Anime, seasons []Season, totalEpisodes int, mappings []storage.Mapping) ([]storage.Mapping, error) {
	if anime.Id == "" || len(seasons) == 0 {
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

	if anime.Sequel.AnimeId == "" {
		return mappings, nil
	}

	sequel, err := m.animeList.GetAnime(anime.Sequel.AnimeId)
	if err != nil {
		return nil, err
	}
	disallowedSequelFormats := []string{"ONA", "OVA", "Music", "Movie"}
	if slices.Contains(disallowedSequelFormats, sequel.Format) {
		sequel, err = m.animeList.GetAnime(sequel.Sequel.AnimeId)
		if err != nil {
			return nil, err
		}
	}

	return m.findSequelMappings(sequel, seasons, totalEpisodes, mappings)
}

func (m AnimeMappingFinder) findFirstSeason(title string, results []animeList.Anime) (animeList.Anime, error) {
	match := animeList.Anime{}
	closest := 1

	for _, result := range results {
		distance := scoreAnimeMatch(title, result)
		m.log.Info("Scored anime", slog.String("title", result.Title), slog.Int("score", distance))
		if distance < closest {
			match = result
			closest = distance
		}
	}

	if match.Id == "" {
		for match.Prequel.AnimeId != "" {
			sequel, err := m.animeList.GetAnime(match.Prequel.AnimeId)
			if err != nil {
				m.log.Error("Failed to get anime", slog.Any("error", err))
				return match, err
			}
			match = sequel
		}
	}

	return match, nil
}

func scoreAnimeMatch(targetTitle string, result animeList.Anime) int {
	distance := 0

	targetTitle = cleanTitle(targetTitle)
	closestTitleDistance := utils.ComputeDistance(targetTitle, cleanTitle(result.Title))
	for _, synonym := range result.Synonyms {
		synonymScore := utils.ComputeDistance(targetTitle, cleanTitle(synonym))
		closestTitleDistance = utils.Min(closestTitleDistance, synonymScore)
	}
	distance += closestTitleDistance

	return distance
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
