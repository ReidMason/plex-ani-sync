package mapping

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/logger"
	"github.com/ReidMason/plex-ani-sync/internal/request"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

func TestCreateMapping(t *testing.T) {
	testCases := []struct {
		name     string
		title    string
		seasons  []Season
		expected []storage.Mapping
	}{
		{
			name:     "No seasons",
			title:    "Fake show",
			seasons:  []Season{},
			expected: []storage.Mapping{},
		},
		{
			name:  "No title",
			title: "",
			seasons: []Season{
				{
					Id:       "12345",
					Episodes: 12,
				},
			},
			expected: []storage.Mapping{},
		},
		{
			name:  "One season",
			title: "No Game No Life",
			seasons: []Season{
				{
					Id:       "12345",
					Episodes: 12,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "19815",
					SeasonId:           "12345",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
			},
		},
		{
			name:  "Two seasons",
			title: "Jujutsu Kaisen",
			seasons: []Season{
				{
					Id:       "1",
					Episodes: 24,
				},
				{
					Id:       "2",
					Episodes: 23,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "113415",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    24,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  24,
				},
				{
					AnimeId:            "145064",
					SeasonId:           "2",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    23,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  23,
				},
			},
		},
		{
			name:  "Multiple seasons",
			title: "My Hero Academia",
			seasons: []Season{
				{
					Id:       "1",
					Episodes: 13,
				},
				{
					Id:       "2",
					Episodes: 25,
				},
				{
					Id:       "3",
					Episodes: 25,
				},
				// {
				//   Id:       "4",
				//   Episodes: 25,
				// },
				// {
				//   Id:       "5",
				//   Episodes: 25,
				// },
				// {
				//   Id:       "6",
				//   Episodes: 25,
				// },
				// {
				//   Id:       "7",
				//   Episodes: 25,
				// },
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "21459",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    13,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  13,
				},
				{
					AnimeId:            "21856",
					SeasonId:           "2",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    25,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  25,
				},
				{
					AnimeId:            "100166",
					SeasonId:           "3",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    25,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  25,
				},
			},
		},
	}

	mockLogger := logger.MockLogger{}
	baseClient := http.DefaultClient
	client := request.NewStaggeredHttpClient(baseClient, mockLogger)
	dbLocation := "../../data/data.db"
	cache, err := storage.NewSqliteStorage(dbLocation, mockLogger)
	if err != nil {
		panic(err)
	}
	anilist := animeList.NewAnilist(client, cache, mockLogger)

	m := NewMappingFinder(anilist, mockLogger)

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.title), func(t *testing.T) {
			t.Parallel()

			result, err := m.CreateMappingsForSeasons(tc.title, tc.seasons)
			if err != nil {
				t.Errorf("createMapping(%v, %v, %v) = %v; want %v", tc.title, tc.seasons, nil, result, tc.expected)
			}

			if len(result) != len(tc.expected) {
				t.Errorf("createMapping(%v, %v, %v) = %v; want %v", tc.title, tc.seasons, nil, result, tc.expected)
			}

			for i, mapping := range result {
				if mapping != tc.expected[i] {
					fmt.Printf("result  : %v\nexpected: %v\n", result, tc.expected)
					t.Errorf("createMapping(%v, %v, %v) = %v; want %v", tc.title, tc.seasons, nil, result, tc.expected)
				}
			}
		})
	}
}
