package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	DefaultURL = "https://graphql.anilist.co"
	pageSize   = 50
	query      = `query ($ids: [Int], $perPage: Int) { Page(perPage: $perPage) { media(id_in: $ids, type: ANIME) { id episodes title { romaji english } } } }`
)

type Media struct {
	ID int
	// Episodes is nil while AniList doesn't know the total.
	Episodes     *int
	TitleRomaji  string
	TitleEnglish string
}

type Client struct {
	URL  string
	HTTP *http.Client
}

// Fetch looks up Entries by AniList ID; IDs AniList doesn't know are absent from the result.
func (c Client) Fetch(ctx context.Context, ids []int) ([]Media, error) {
	var out []Media
	for start := 0; start < len(ids); start += pageSize {
		page, err := c.fetchPage(ctx, ids[start:min(start+pageSize, len(ids))])
		if err != nil {
			return nil, err
		}
		out = append(out, page...)
	}
	return out, nil
}

func (c Client) fetchPage(ctx context.Context, ids []int) ([]Media, error) {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": map[string]any{"ids": ids, "perPage": pageSize}})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anilist: unexpected status %d", resp.StatusCode)
	}
	var body struct {
		Data struct {
			Page struct {
				Media []struct {
					ID       int  `json:"id"`
					Episodes *int `json:"episodes"`
					Title    struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
					} `json:"title"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("anilist: %w", err)
	}
	if len(body.Errors) > 0 {
		return nil, fmt.Errorf("anilist: %s", body.Errors[0].Message)
	}
	out := make([]Media, 0, len(body.Data.Page.Media))
	for _, m := range body.Data.Page.Media {
		out = append(out, Media{ID: m.ID, Episodes: m.Episodes, TitleRomaji: m.Title.Romaji, TitleEnglish: m.Title.English})
	}
	return out, nil
}
