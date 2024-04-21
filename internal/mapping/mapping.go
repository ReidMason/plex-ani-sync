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

func (m Mapping) FindMapping(series mediaHost.Series, seasons []mediaHost.Season) (animeList.Anime, error) {
	slog.Info("Searching for anime: ", slog.String("title", series.Title))

	results, err := m.animeList.SearchAnime(series.Title)
	if err != nil {
		return animeList.Anime{}, err
	}

	matchedResult, err := findMatchingResult(results, series.Title)
	if err != nil {
		return animeList.Anime{}, err
	}

	totalEpisodes := getTotalEpisodes(seasons)
	if totalEpisodes != matchedResult.Episodes {
		slog.Info("Total episodes mismatch", slog.Int("expected", matchedResult.Episodes), slog.Int("actual", totalEpisodes))
		return animeList.Anime{}, errors.New("Total episodes mismatch")
	}

	slog.Info("Found matching anime: ", slog.String("title", matchedResult.Title))

	return animeList.Anime{}, nil
}

func getTotalEpisodes(seasons []mediaHost.Season) int {
	totalEpisodes := 0
	for _, season := range seasons {
		slog.Info("Season: ", slog.String("title", season.Title), slog.Int("index", season.Index))
		totalEpisodes += season.Episodes
	}

	return totalEpisodes
}

func findMatchingResult(results []animeList.Anime, title string) (animeList.Anime, error) {
	for _, result := range results {
		if strings.ToLower(result.Title) == strings.ToLower(title) {
			return result, nil
		}

		for _, synonym := range result.Synonyms {
			if strings.ToLower(synonym) == strings.ToLower(title) {
				return result, nil
			}
		}
	}

	return animeList.Anime{}, errors.New("no matching anime found")
}
