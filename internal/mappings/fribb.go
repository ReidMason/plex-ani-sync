package mappings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type fribbEntry struct {
	AniDBID   int `json:"anidb_id"`
	AniListID int `json:"anilist_id"`
	TVDBID    int `json:"thetvdb_id"`
	Season    *struct {
		TVDB int `json:"tvdb"`
	} `json:"season"`
	EpisodeOffset *struct {
		TVDB int `json:"tvdb"`
	} `json:"episode_offset"`
}

// FetchFribb downloads and parses Fribb's anime-list-full.json.
func FetchFribb(ctx context.Context, client *http.Client, url string) ([]Row, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fribb dataset: unexpected status %d", resp.StatusCode)
	}
	var entries []fribbEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("fribb dataset: %w", err)
	}
	rows := make([]Row, 0, len(entries))
	for _, e := range entries {
		if e.Season == nil {
			continue
		}
		offset := 0
		if e.EpisodeOffset != nil {
			offset = e.EpisodeOffset.TVDB
		}
		rows = append(rows, Row{AniDBID: e.AniDBID, AniListID: e.AniListID, TVDBID: e.TVDBID, Season: e.Season.TVDB, EpisodeOffset: offset})
	}
	return rows, nil
}
