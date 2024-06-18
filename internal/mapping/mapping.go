package mapping

import (
	"errors"
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
	animeEpisodeEnd := season.Offset + season.Episodes
	if animeEpisodeEnd > anime.Episodes {
		animeEpisodeEnd = anime.Episodes
	}

	seasonEpisodeStart := 1
	if season.partial {
		previousMapping := mappings[len(mappings)-1]
		seasonEpisodeStart = previousMapping.SesasonEpisodeEnd + 1
	}

	seasonEpisodeEnd := animeEpisodeEnd
	if season.Episodes < animeEpisodeEnd {
		seasonEpisodeEnd = season.Episodes
	}
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

func (m AnimeMappingFinder) findAnime(targetTitle string, totalEpisodes int, anime []animeList.Anime) []animeList.Anime {
	if anime == nil || len(anime) == 0 {
		return nil
	}

	animeEpisodeCount := 0
	for _, a := range anime {
		animeEpisodeCount += a.Episodes
	}

	if animeEpisodeCount == totalEpisodes {
		return anime
	}

	latestAnime := anime[len(anime)-1]

	differenceThreshold := 10
	if utils.ComputeDistance(latestAnime.Title, targetTitle) > differenceThreshold {
		return nil
	}

	if latestAnime.Sequel.Id == "" {
		return nil
	}

	result, err := m.animeList.GetAnime(latestAnime.Sequel.Id)
	if err != nil {
		return anime
	}

	anime = append(anime, result)
	animeEpisodeCount = 0
	for _, a := range anime {
		animeEpisodeCount += a.Episodes
	}

	if animeEpisodeCount == totalEpisodes {
		return anime
	} else if animeEpisodeCount < totalEpisodes {
		return m.findAnime(targetTitle, totalEpisodes, anime)
	}

	return nil
}

func (m AnimeMappingFinder) findMappingsForSeries(series mediaHost.Series, seasons []mediaHost.Season) ([]storage.Mapping, error) {
	cleanedTitle := cleanTitle(series.Title)
	results, err := m.animeList.SearchAnime(cleanedTitle)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		m.log.Warn("No anime found", slog.String("title", cleanedTitle))
		return nil, errors.New("No anime found")
	}

	matchedResults, err := findMatchingTitleResults(results, cleanedTitle)
	if err != nil {
		m.log.Error("Failed to find matching title results", slog.Any("error", err), slog.String("title", cleanedTitle))
		return nil, err
	}
	if len(matchedResults) == 0 {
		m.log.Warn("Titles filtered out all anime", slog.String("title", cleanedTitle))
		return nil, errors.New("No anime found")
	}

	totalEpisodes := getTotalEpisodes(seasons)
	for _, result := range matchedResults {
		anime := make([]animeList.Anime, 0)
		anime = append(anime, result)
		anime = m.findAnime(series.Title, totalEpisodes, anime)
		if anime != nil {
			return createMapping(seasons, anime), nil
		}
	}

	m.log.Warn("No matching anime found", slog.String("title", cleanedTitle), slog.Any("results", matchedResults))
	return nil, errors.New("No matching anime found")
}

func createMapping(selectedSeasons []mediaHost.Season, selectedAnilistEntries []animeList.Anime) []storage.Mapping {
	mappings := make([]storage.Mapping, 0)

	anilistEntryIndex := 0
	anilistEpisodeStart := 1
	for _, season := range selectedSeasons {
		plexEpisodeStart := 1

		// Run while there are unmapped Plex episodes and we haven't used all our anilist entries
		for plexEpisodeStart < season.Episodes && len(selectedAnilistEntries) > anilistEntryIndex {
			anilistEntry := selectedAnilistEntries[anilistEntryIndex]

			seasonEpisodesRemainingToMap := season.Episodes - plexEpisodeStart
			seasonLength := seasonEpisodesRemainingToMap

			anilistEntryUnmappedEpisodes := anilistEntry.Episodes - anilistEpisodeStart
			// If anilist entry doesn't have enough to map the rest of the season we use all the episodes the anilist entry has
			if anilistEntryUnmappedEpisodes < seasonEpisodesRemainingToMap {
				seasonLength = anilistEntryUnmappedEpisodes
			}

			mapping := storage.Mapping{
				AnimeId:            fmt.Sprint(anilistEntry.Id),
				SeasonId:           season.Id,
				AnimeEpisodeStart:  anilistEpisodeStart,
				AnimeEpisodeEnd:    anilistEpisodeStart + seasonLength,
				SeasonEpisodeStart: plexEpisodeStart,
				SesasonEpisodeEnd:  plexEpisodeStart + seasonLength,
			}

			plexEpisodeStart += seasonLength + 1
			anilistEpisodeStart += seasonLength + 1

			// If we have used all episodes in the anilist entry we can move on to the next one
			if anilistEntry.Episodes <= anilistEpisodeStart || anilistEntryUnmappedEpisodes <= 0 {
				anilistEntryIndex++
				anilistEpisodeStart = 1
			}

			mappings = append(mappings, mapping)
		}
	}

	return mappings
}

func getMatchingEpisodeResults(results []animeList.Anime, totalEpisodes int) ([]animeList.Anime, error) {
	matchedResults := make([]animeList.Anime, 0)
	for _, result := range results {
		if result.Episodes == totalEpisodes {
			matchedResults = append(matchedResults, result)
		}
	}

	return matchedResults, nil
}

func getTotalEpisodes(seasons []mediaHost.Season) int {
	totalEpisodes := 0
	for _, season := range seasons {
		if season.Index == 0 {
			continue
		}
		totalEpisodes += season.Episodes
	}

	return totalEpisodes
}

func findMatchingTitleResults(results []animeList.Anime, title string) ([]animeList.Anime, error) {
	matchedResults := make([]animeList.Anime, 0)
	for _, result := range results {
		resultTitle := strings.ToLower(cleanTitle(result.Title))
		if resultTitle == strings.ToLower(title) || synonymsMatch(title, result.Synonyms) {
			matchedResults = append(matchedResults, result)
		}
	}

	return matchedResults, nil
}

func synonymsMatch(title string, synonyms []string) bool {
	for _, synonym := range synonyms {
		synonym = strings.ToLower(cleanTitle(synonym))
		if synonym == strings.ToLower(title) {
			return true
		}
	}

	return false
}
