package mappings

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
)

// SeasonSpan is one Anime-Lists `mapping`: Entry episodes First..Last are TVDB Season episodes shifted by Offset.
type SeasonSpan struct {
	Season int
	First  int
	Last   int
	Offset int
}

// AnimeLists holds the usable episode-level mappings from Anime-Lists, keyed by AniDB ID.
type AnimeLists map[int][]SeasonSpan

type xmlList struct {
	Anime []struct {
		AniDBID  string `xml:"anidbid,attr"`
		Mappings []struct {
			AniDBSeason string `xml:"anidbseason,attr"`
			TVDBSeason  string `xml:"tvdbseason,attr"`
			Start       string `xml:"start,attr"`
			End         string `xml:"end,attr"`
			Offset      string `xml:"offset,attr"`
		} `xml:"mapping-list>mapping"`
	} `xml:"anime"`
}

// FetchAnimeLists downloads anime-list.xml, keeping only range mappings from AniDB season 1 into a TVDB season.
func FetchAnimeLists(ctx context.Context, client *http.Client, url string) (AnimeLists, error) {
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
		return nil, fmt.Errorf("anime-lists dataset: unexpected status %d", resp.StatusCode)
	}
	var doc xmlList
	if err := xml.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("anime-lists dataset: %w", err)
	}
	out := AnimeLists{}
	for _, a := range doc.Anime {
		anidb, err := strconv.Atoi(a.AniDBID)
		if err != nil {
			continue
		}
		for _, m := range a.Mappings {
			season, errS := strconv.Atoi(m.TVDBSeason)
			first, errF := strconv.Atoi(m.Start)
			last, errL := strconv.Atoi(m.End)
			if m.AniDBSeason != "1" || errS != nil || errF != nil || errL != nil || season <= 0 || first <= 0 || last < first {
				continue
			}
			offset := 0
			if m.Offset != "" {
				if offset, err = strconv.Atoi(m.Offset); err != nil {
					continue
				}
			}
			out[anidb] = append(out[anidb], SeasonSpan{Season: season, First: first, Last: last, Offset: offset})
		}
	}
	return out, nil
}
