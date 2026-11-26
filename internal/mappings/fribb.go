package mappings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
)

type fribbEntry struct {
	AniDBID   int `json:"anidb_id"`
	AniListID int `json:"anilist_id"`
	TVDBID    int `json:"thetvdb_id"`
	TMDBID    int `json:"themoviedb_id"`
	Season    *struct {
		TVDB int `json:"tvdb"`
	} `json:"season"`
	EpisodeOffset *struct {
		TVDB int `json:"tvdb"`
	} `json:"episode_offset"`
}

// IDLink says a TMDB show ID and a TVDB show ID identify the same Show.
type IDLink struct {
	TMDBID int
	TVDBID int
}

// Dataset is the parsed content of Fribb's list.
type Dataset struct {
	Rows  []Row
	Links []IDLink
}

// FetchFribb downloads and parses Fribb's anime-list-full.json.
func FetchFribb(ctx context.Context, client *http.Client, url string) (Dataset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Dataset{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Dataset{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Dataset{}, fmt.Errorf("fribb dataset: unexpected status %d", resp.StatusCode)
	}
	var entries []fribbEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return Dataset{}, fmt.Errorf("fribb dataset: %w", err)
	}
	slices.SortStableFunc(entries, func(a, b fribbEntry) int { return a.AniDBID - b.AniDBID })
	var data Dataset
	linked := map[int]bool{}
	for _, e := range entries {
		if e.TMDBID > 0 && e.TVDBID > 0 && !linked[e.TMDBID] {
			linked[e.TMDBID] = true
			data.Links = append(data.Links, IDLink{TMDBID: e.TMDBID, TVDBID: e.TVDBID})
		}
		if e.Season == nil {
			continue
		}
		offset := 0
		if e.EpisodeOffset != nil {
			offset = e.EpisodeOffset.TVDB
		}
		data.Rows = append(data.Rows, Row{AniDBID: e.AniDBID, AniListID: e.AniListID, TVDBID: e.TVDBID, Season: e.Season.TVDB, EpisodeOffset: offset})
	}
	return data, nil
}
