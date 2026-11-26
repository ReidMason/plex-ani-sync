package mappings

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed custom.json
var embeddedCustom []byte

type customEntry struct {
	TVDBID            int  `json:"tvdbId"`
	Season            int  `json:"season"`
	FirstEpisode      int  `json:"firstEpisode"`
	LastEpisode       *int `json:"lastEpisode"`
	AniListID         int  `json:"anilistId"`
	EntryFirstEpisode int  `json:"entryFirstEpisode"`
	EntryLastEpisode  *int `json:"entryLastEpisode"`
}

// Custom returns the Custom Mappings embedded from custom.json.
func Custom() ([]Mapping, error) {
	return ParseCustom(embeddedCustom)
}

// ParseCustom reads Custom Mappings from JSON, rejecting entries that are incomplete or have inverted ranges.
func ParseCustom(data []byte) ([]Mapping, error) {
	var entries []customEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("custom mappings: %w", err)
	}
	out := make([]Mapping, 0, len(entries))
	for i, e := range entries {
		if e.TVDBID <= 0 || e.Season <= 0 || e.AniListID <= 0 || e.FirstEpisode <= 0 || e.EntryFirstEpisode <= 0 ||
			(e.LastEpisode != nil && *e.LastEpisode < e.FirstEpisode) ||
			(e.EntryLastEpisode != nil && *e.EntryLastEpisode < e.EntryFirstEpisode) {
			return nil, fmt.Errorf("custom mappings: entry %d is invalid", i)
		}
		out = append(out, Mapping{
			TVDBID: e.TVDBID, Season: e.Season, Episodes: Range{First: e.FirstEpisode, Last: e.LastEpisode},
			AniListID: e.AniListID, Entry: Range{First: e.EntryFirstEpisode, Last: e.EntryLastEpisode},
		})
	}
	return out, nil
}
