package mapping

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
)

type Mapping struct {
	animeList animeList.AnimeList
}

func NewMapping(animeList animeList.AnimeList) *Mapping {
	return &Mapping{animeList: animeList}
}

func (m Mapping) findAnime(totalEpisodes int, anime []animeList.Anime) []animeList.Anime {
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

func (m Mapping) FindMapping(series mediaHost.Series, seasons []mediaHost.Season) (animeList.Anime, error) {
	cleanedTitle := cleanTitle(series.Title)
	results, err := m.animeList.SearchAnime(cleanedTitle)
	if err != nil {
		return animeList.Anime{}, err
	}
	if len(results) == 0 {
		slog.Error("No anime found", slog.String("title", cleanedTitle))
		return animeList.Anime{}, errors.New("No anime found")
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
		return animeList.Anime{}, err
	}
	if len(matchedResults) == 0 {
		slog.Error("Titles filtered out all anime", slog.String("title", cleanedTitle))
		return animeList.Anime{}, errors.New("No anime found")
	}

	totalEpisodes := getTotalEpisodes(seasons)
	for _, result := range matchedResults {
		anime := make([]animeList.Anime, 0)
		anime = append(anime, result)
		anime = m.findAnime(totalEpisodes, anime)
		if anime != nil {
			return anime[len(anime)-1], nil
		}
	}

	slog.Error("No matching anime found", slog.String("title", cleanedTitle), slog.Any("results", matchedResults))
	return animeList.Anime{}, errors.New("No matching anime found")
}

func cleanTitle(title string) string {
	removeChars := []string{"...", ":", "!", "?", "(TV)", "’", "'", " TV", "-"}
	for _, char := range removeChars {
		title = strings.ReplaceAll(title, char, "")
	}

	return strings.TrimSpace(title)
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
