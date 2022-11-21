package mapping

import (
	"errors"
	"plex-ani-sync/services/anilist"
	"plex-ani-sync/services/database"
	"plex-ani-sync/services/plex"
	"plex-ani-sync/services/requesthandler"
	"plex-ani-sync/testUtils"
	"strings"
	"testing"
)

func TestGetSeriesMappings(t *testing.T) {
	testCases := []struct {
		Name, SeriesName string
		ExpectedMappings []anilist.AnilistMedia
	}{
		{
			Name:       "One to one mapping",
			SeriesName: "Mysterious Girlfriend X",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 12467,
				},
			},
		},
		{
			Name:       "Continuous mapping",
			SeriesName: "One Piece",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 21,
				},
			},
		},
		{
			Name:       "Many Plex seasons to multiple anilist season",
			SeriesName: "Bleach",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 269,
				},
				{
					ID: 116674,
				},
			},
		},
		{
			Name:       "Multiple mappings with one Plex season into two Anilist entries",
			SeriesName: "Attack on Titan",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 16498,
				},
				{
					ID: 20958,
				},
				{
					ID: 99147,
				},
				{
					ID: 104578,
				},
				{
					ID: 110277,
				},
				{
					ID: 131681,
				},
			},
		},
		{
			Name:       "One to one mapping where sequel is a different format from prequel",
			SeriesName: "B: The Beginning",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 21665,
				},
				{
					ID: 102498,
				},
			},
		},
		{
			Name:       "Series with sequels with unrelated titles",
			SeriesName: "Aria",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 477,
				},
				{
					ID: 962,
				},
				{
					ID: 3297,
				},
			},
		},
		{
			Name:       "Series where Plex seasons are made up on TV and Specials",
			SeriesName: "Ah! My Goddess",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 50,
				},
				{
					ID: 1003,
				},
				{
					ID: 880,
				},
				{
					ID: 2198,
				},
			},
		},
		{
			Name:       "Ensure release year is factored in",
			SeriesName: "The Ancient Magus' Bride",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 98436,
				},
			},
		},
		{
			Name:       "Anilist entries with multiple sequels",
			SeriesName: "Demon Slayer: Kimetsu no Yaiba",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 101922,
				},
				{
					ID: 129874,
				},
				{
					ID: 142329,
				},
			},
		},
		{
			Name:       "Sequel of OVA can be TV",
			SeriesName: "FLCL",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 227,
				},
				{
					ID: 21746,
				},
				{
					ID: 21748,
				},
			},
		},
		{
			Name:       "Sequel might actually be a side story",
			SeriesName: "Full Metal Panic!",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 71,
				},
				{
					ID: 72,
				},
				{
					ID: 73,
				},
				{
					ID: 21451,
				},
			},
		},
		{
			Name:       "Idek anymore",
			SeriesName: "Food Wars! Shokugeki no Soma",
			ExpectedMappings: []anilist.AnilistMedia{
				{
					ID: 20923,
				},
				{
					ID: 21518,
				},
				{
					ID: 99255,
				},
				{
					ID: 100773,
				},
				{
					ID: 109963,
				},
				{
					ID: 114043,
				},
			},
		},
	}

	databaseService := database.Connect()
	requestHandler := requesthandler.New()
	anilistToken := "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6ImI5ZWUxZTgxYTA5MTc5MzAzYzc1YjY0ODUwODc1Yzk0MmY2MjYzOTAzZGQ5MmZkZDdmZjMyNTdlYTQ5ZjRlZGM0YzIxMmVkMTdjMWYyMmU5In0.eyJhdWQiOiIzMDU0IiwianRpIjoiYjllZTFlODFhMDkxNzkzMDNjNzViNjQ4NTA4NzVjOTQyZjYyNjM5MDNkZDkyZmRkN2ZmMzI1N2VhNDlmNGVkYzRjMjEyZWQxN2MxZjIyZTkiLCJpYXQiOjE2NDE0MTk0MTcsIm5iZiI6MTY0MTQxOTQxNywiZXhwIjoxNjcyOTU1NDE3LCJzdWIiOiIzOTI3NTQiLCJzY29wZXMiOltdfQ.iEoBzUvDu2PBdwTjLkZjHHmYZW472EMefUFnoJD1iZkaVoUUblfwG6OJonXkJf8owrNRbPdp5lS2ohJCKK-M0fwJND1GE1rDs3FhsmGBnH79y0Jgyn3ikhdoT98vnqoCBz0vZWrHWxH_wcOwnDySHci_DZrMWh9t5aH0Qf8fDVJwT_JDsXRQYbVRDlfx-w6JlxKtanV_i7ygasFQEs429N8-4s87E5nHlCC5Y-Wa04rfwAqRCl6O1DBTapoXEoqdk3FD6ZaCsHCZEDm0ojmy2EJXBeePETeg6Yw5r2BscuOK_wF_yF-ajN-Jh-ug2aBIyMcj9illTKqV9kROQdq1EVLrHMcf6vRsH5xB_XhiY7wM_MRhRobCSEnFWEnFk_yoq3w8kST66XrECaXKbHNwROE0B0ux2Id-1v1fvqlxdmP2RwG7I-tv7Ade3rm5wuqN40DRvw2JyfJ0pBiJE2h8NIzrbHoCf4qghsRATFdizRdUpqBxocDotMxCsgF6whKWne94J0A_An2rUrPkON-knBwKoz6ZbalIGZiaUx3FR0pow0Kp-GVp-rGRvQIpNLJtmXYobtDH2FUfwksAZv0BMzXp7D1_h3Ej42mRGsbaEvahrXynXkkYAOxCB2R5FcMHybDGJ8y58lxx8V5rYCxa9Q4jFrZBh_fsIVRd2pflx2w"
	// anilistService := anilist.NewMockAnilistService(databaseService)
	anilistService := anilist.New(requestHandler, anilistToken, databaseService)
	anilistSeriesFinder := NewAnilistMappingFinder(anilistService)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			series, err := findPlexSeries(tc.SeriesName)
			if err != nil {
				t.Fatalf("Failed to find series test data for '%s'", tc.SeriesName)
			}

			result, err := anilistSeriesFinder.GetSeriesAnilistEntries(series)

			if err != nil {
				t.Fatal("Got unexpected error:", err)
			}

			if len(result) != len(tc.ExpectedMappings) {
				t.Fatalf("Got wrong number of anilist entries. Expected: '%d' found '%d'", len(tc.ExpectedMappings), len(result))
			}

			for i, entry := range result {
				expectedEntry := tc.ExpectedMappings[i]
				if entry.ID != expectedEntry.ID {
					t.Errorf("Found wrong anilist entry. Expected '%d' found '%d'", expectedEntry.ID, entry.ID)
				}
			}
		})
	}
}

const plexSeriesFilename = "plex_series.json"

func findPlexSeries(seriesTitle string) (plex.Series, error) {
	allSeries, err := testUtils.GetSavedTestData[[]plex.Series](plexSeriesFilename)
	if err != nil {
		return plex.Series{}, err
	}

	for _, series := range allSeries {
		if strings.ToLower(series.Title) == strings.ToLower(seriesTitle) {
			return series, nil
		}
	}

	return plex.Series{}, errors.New("Didn't find Plex series")
}
