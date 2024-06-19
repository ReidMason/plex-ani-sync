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
					Id:          "12345",
					Episodes:    12,
					ReleaseYear: 2014,
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
					Id:          "1",
					Episodes:    24,
					ReleaseYear: 2020,
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
					Id:          "1",
					Episodes:    13,
					ReleaseYear: 2016,
				},
				{
					Id:       "2",
					Episodes: 25,
				},
				{
					Id:       "3",
					Episodes: 25,
				},
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
		{
			name:  "Split cour animelist",
			title: "Attack on Titan",
			seasons: []Season{
				{
					Id:          "1",
					Episodes:    25,
					ReleaseYear: 2013,
				},
				{
					Id:       "2",
					Episodes: 12,
				},
				{
					Id:       "3",
					Episodes: 22,
				},
				{
					Id:       "4",
					Episodes: 30,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "16498",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    25,
					SeasonEpisodeStart: 1, SesasonEpisodeEnd: 25,
				},
				{
					AnimeId:            "20958",
					SeasonId:           "2",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
				{
					AnimeId:            "99147",
					SeasonId:           "3",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
				{
					AnimeId:            "104578",
					SeasonId:           "3",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    10,
					SeasonEpisodeStart: 13,
					SesasonEpisodeEnd:  22,
				},
				{
					AnimeId:            "110277",
					SeasonId:           "4",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    16,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  16,
				},
				{
					AnimeId:            "131681",
					SeasonId:           "4",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 17,
					SesasonEpisodeEnd:  28,
				},
				{
					AnimeId:            "146984",
					SeasonId:           "4",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    1,
					SeasonEpisodeStart: 29,
					SesasonEpisodeEnd:  29,
				},
				{
					AnimeId:            "162314",
					SeasonId:           "4",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    1,
					SeasonEpisodeStart: 30,
					SesasonEpisodeEnd:  30,
				},
			},
		},
		{
			name:  "Split cour Plex",
			title: "Naruto",
			seasons: []Season{
				{
					Id:          "1",
					Episodes:    35,
					ReleaseYear: 2002,
				},
				{
					Id:       "2",
					Episodes: 48,
				},
				{
					Id:       "3",
					Episodes: 48,
				},
				{
					Id:       "4",
					Episodes: 48,
				},
				{
					Id:       "5",
					Episodes: 41,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "20",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    35,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  35,
				},
				{
					AnimeId:            "20",
					SeasonId:           "2",
					AnimeEpisodeStart:  36,
					AnimeEpisodeEnd:    83,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  48,
				},
				{
					AnimeId:            "20",
					SeasonId:           "3",
					AnimeEpisodeStart:  84,
					AnimeEpisodeEnd:    131,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  48,
				},
				{
					AnimeId:            "20",
					SeasonId:           "4",
					AnimeEpisodeStart:  132,
					AnimeEpisodeEnd:    179,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  48,
				}, {
					AnimeId:            "20",
					SeasonId:           "5",
					AnimeEpisodeStart:  180,
					AnimeEpisodeEnd:    220,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  41,
				},
			},
		},
		{
			name:  "Sequel is an OVA",
			title: "Maken-Ki! Battling Venus",
			seasons: []Season{
				{
					Id:          "1",
					Episodes:    12,
					ReleaseYear: 2011,
				},
				{
					Id:       "2",
					Episodes: 10,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "9936",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    12,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
				},
				{
					AnimeId:            "15565",
					SeasonId:           "2",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    10,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  10,
				},
			},
		},
		{
			name:  "An earlier remake exists",
			title: "Hunter x Hunter",
			seasons: []Season{
				{
					Id:          "1",
					Episodes:    58,
					ReleaseYear: 2011,
				},
				{
					Id:       "2",
					Episodes: 78,
				},
				{
					Id:       "3",
					Episodes: 12,
				},
			},
			expected: []storage.Mapping{
				{
					AnimeId:            "11061",
					SeasonId:           "1",
					AnimeEpisodeStart:  1,
					AnimeEpisodeEnd:    58,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  58,
				},
				{
					AnimeId:            "11061",
					SeasonId:           "2",
					AnimeEpisodeStart:  59,
					AnimeEpisodeEnd:    136,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  78,
				},
				{
					AnimeId:            "11061",
					SeasonId:           "3",
					AnimeEpisodeStart:  137,
					AnimeEpisodeEnd:    148,
					SeasonEpisodeStart: 1,
					SesasonEpisodeEnd:  12,
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
				t.Errorf("createMapping(%v, %v, %v) \nfound %v;\nwant  %v", tc.title, tc.seasons, nil, result, tc.expected)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("createMapping(%v, %v, %v) \nfound %v;\nwant  %v", tc.title, tc.seasons, nil, result, tc.expected)
				return
			}

			for i, mapping := range result {
				if mapping != tc.expected[i] {
					t.Errorf("createMapping(%v, %v, %v) \nfound %v;\nwant  %v", tc.title, tc.seasons, nil, result, tc.expected)
					return
				}
			}
		})
	}
}
