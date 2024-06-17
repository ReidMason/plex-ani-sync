package mapping

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/ReidMason/plex-ani-sync/internal/utils"
)

type MappingFinder interface {
	CreateMappings(series mediaHost.Series) error
}

type AnimeMappingFinder struct {
	animeList        animeList.AnimeList
	log              *slog.Logger
	mappingStorage   storage.MappingStorage
	mediaHostService mediaHost.MediaHost
}

func NewMappingFinder(animeList animeList.AnimeList, mappingStorage storage.MappingStorage, mediaHostService mediaHost.MediaHost, logger *slog.Logger) *AnimeMappingFinder {
	return &AnimeMappingFinder{animeList: animeList, log: logger, mappingStorage: mappingStorage, mediaHostService: mediaHostService}
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

func cleanTitle(title string) string {
	removeChars := []string{"...", ":", "!", "?", "(TV)", "’", "'", " TV", "-"}
	for _, char := range removeChars {
		title = strings.ReplaceAll(title, char, "")
	}

	return strings.TrimSpace(title)
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

func (m AnimeMappingFinder) CreateMappings(series mediaHost.Series) error {
	matchedSeries := 0
	allSeasons, err := m.mediaHostService.GetSeasons(series.Id)
	if err != nil {
		m.log.Error("Failed to get seasons", slog.Any("error", err))
		return err
	}

	seasons := make([]mediaHost.Season, 0)
	for _, season := range allSeasons {
		if season.Index == 0 {
			continue
		}
		seasons = append(seasons, season)
	}

	total_non_special_episodes := 0
	for _, season := range seasons {
		total_non_special_episodes += season.Episodes
	}

	mappedEpisodes := 0
	for _, season := range seasons {
		mappings, err := m.mappingStorage.GetMappings(season.Id)
		if err != nil {
			m.log.Error("Failed to get mappings", slog.Any("error", err))
			return err
		}

		for _, mapping := range mappings {
			mappedEpisodes += mapping.SesasonEpisodeEnd - mapping.SeasonEpisodeStart + 1
		}
	}

	if mappedEpisodes == total_non_special_episodes {
		m.log.Info("All episodes already mapped", slog.String("series", series.Title))
		return nil
	}

	newMappings, err := m.findMappingsForSeries(series, seasons)
	if err == nil {
		matchedSeries += 1
	}

	err = m.mappingStorage.SetMappings(newMappings)
	if err != nil {
		m.log.Error("Failed to add mappings", slog.Any("error", err))
		return err
	}

	return nil
}
