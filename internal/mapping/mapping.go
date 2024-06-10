package mapping

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

type MappingFinder interface {
	FindMapping(series mediaHost.Series, seasons []mediaHost.Season) (animeList.Anime, error)
}

type AnimeMappingFinder struct {
	animeList animeList.AnimeList
}

func NewMapping(animeList animeList.AnimeList) *AnimeMappingFinder {
	return &AnimeMappingFinder{animeList: animeList}
}

func (m AnimeMappingFinder) findAnime(totalEpisodes int, anime []animeList.Anime) []animeList.Anime {
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
		return m.findAnime(totalEpisodes, anime)
	}

	return nil
}

func (m AnimeMappingFinder) FindMapping(series mediaHost.Series, seasons []mediaHost.Season) ([]storage.Mapping, error) {
	cleanedTitle := cleanTitle(series.Title)
	results, err := m.animeList.SearchAnime(cleanedTitle)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		slog.Error("No anime found", slog.String("title", cleanedTitle))
		return nil, errors.New("No anime found")
	}

	// This could result in some false positives
	// for _, result := range results {
	// 	if len(seasons) == 1 && result.Year == seasons[0].Year {
	// 		return results[0], nil
	// 	}
	// }

	matchedResults, err := findMatchingTitleResults(results, cleanedTitle)
	if err != nil {
		slog.Error("Failed to find matching title results", slog.Any("error", err), slog.String("title", cleanedTitle))
		return nil, err
	}
	if len(matchedResults) == 0 {
		slog.Error("Titles filtered out all anime", slog.String("title", cleanedTitle))
		return nil, errors.New("No anime found")
	}

	totalEpisodes := getTotalEpisodes(seasons)
	for _, result := range matchedResults {
		anime := make([]animeList.Anime, 0)
		anime = append(anime, result)
		anime = m.findAnime(totalEpisodes, anime)
		if anime != nil {
			return createMapping(seasons, anime), nil
		}
	}

	slog.Error("No matching anime found", slog.String("title", cleanedTitle), slog.Any("results", matchedResults))
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

			fmt.Println("Plex end: ", plexEpisodeStart+seasonLength)
			fmt.Println("Anilist end: ", anilistEpisodeStart+seasonLength)

			mapping := storage.Mapping{
				AnimeId:           fmt.Sprint(anilistEntry.Id),
				MediaId:           season.Id,
				AnimeEpisodeStart: anilistEpisodeStart,
				AnimeEpisodeEnd:   anilistEpisodeStart + seasonLength,
				MediaEpisodeStart: plexEpisodeStart,
				MediaEpisodeEnd:   plexEpisodeStart + seasonLength,
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
