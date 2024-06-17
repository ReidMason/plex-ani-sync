package mapping

import (
	"fmt"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

func TestCreateMapping(t *testing.T) {
	testCases := []struct {
		name                   string
		series                 mediaHost.Series
		selectedSeasons        []mediaHost.Season
		selectedAnilistEntries []animeList.Anime
		expected               []storage.Mapping
	}{
		{
			name: "Test 1",
			series: mediaHost.Series{
				Id:    "1",
				Title: "Test Series",
			},
			selectedSeasons: []mediaHost.Season{
				{
					Id:       "123",
					Title:    "Season 1",
					Index:    1,
					Episodes: 12,
				},
			},
			selectedAnilistEntries: []animeList.Anime{
				{
					Id:       456,
					Title:    "Test Anime",
					Episodes: 12,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "456",
					SeasonId:           "123",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
			},
		},
		{
			name: "Test 2",
			series: mediaHost.Series{
				Id:    "1",
				Title: "Test Series",
			},
			selectedSeasons: []mediaHost.Season{
				{
					Id:       "123",
					Title:    "Season 1",
					Index:    1,
					Episodes: 24,
				},
			},
			selectedAnilistEntries: []animeList.Anime{
				{
					Id:       456,
					Title:    "Test Anime 1",
					Episodes: 12,
				},
				{
					Id:       789,
					Title:    "Test Anime 2",
					Episodes: 12,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "456",
					SeasonId:           "123",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
				{
					AnimeId:            "789",
					SeasonId:           "123",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 13,
					SesasonEpisodeEnd:  24,
				},
			},
		},
		{
			name: "Test 3",
			series: mediaHost.Series{
				Id:    "1",
				Title: "Test Series",
			},
			selectedSeasons: []mediaHost.Season{
				{
					Id:       "123",
					Title:    "Season 1",
					Index:    1,
					Episodes: 12,
				},
				{
					Id:       "456",
					Title:    "Season 2",
					Index:    1,
					Episodes: 12,
				},
			},
			selectedAnilistEntries: []animeList.Anime{
				{
					Id:       789,
					Title:    "Test Anime 1",
					Episodes: 24,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "789",
					SeasonId:           "123",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
				{
					AnimeId:            "789",
					SeasonId:           "456",
					AnimeEpisodeStart:  13,
					AnimeEpisodeEnd:    24,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			t.Parallel()

			result := createMapping(tc.selectedSeasons, tc.selectedAnilistEntries)

			for i, m := range result {
				if m != tc.expected[i] {
					fmt.Printf("result  : %v\nexpected: %v\n", result, tc.expected)
					t.Errorf("createMapping(%v, %v, %v) = %v; want %v", tc.series, tc.selectedSeasons, tc.selectedAnilistEntries, result, tc.expected)
				}
			}
		})
	}
}
