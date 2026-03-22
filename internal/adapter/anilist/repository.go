package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"myapp/internal/domain"
	"net/http"
	"strconv"
)

const graphqlEndpoint = "https://graphql.anilist.co"

// To obtain a token:
//  1. Go to https://anilist.co/settings/developer and create a new client.
//  2. Visit https://anilist.co/api/v2/oauth/authorize?client_id={clientId}&response_type=token
//  3. Authorize, then copy the access_token value from the redirect URL fragment.
//  4. Set it as ANILIST_TOKEN in your .env file.

type Repository struct {
	httpClient *http.Client
	token      string
}

func NewRepository(token string) *Repository {
	return &Repository{
		httpClient: &http.Client{},
		token:      token,
	}
}

func (r *Repository) query(ctx context.Context, q string, variables map[string]any, result any) error {
	body, err := json.Marshal(map[string]any{"query": q, "variables": variables})
	if err != nil {
		return fmt.Errorf("marshalling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from AniList", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	return nil
}

func (r *Repository) viewerID(ctx context.Context) (int, error) {
	const q = `query { Viewer { id } }`

	var resp struct {
		Data struct {
			Viewer struct {
				ID int `json:"id"`
			} `json:"Viewer"`
		} `json:"data"`
	}

	if err := r.query(ctx, q, nil, &resp); err != nil {
		return 0, fmt.Errorf("fetching viewer: %w", err)
	}
	return resp.Data.Viewer.ID, nil
}

func (r *Repository) GetAnimeList(ctx context.Context) ([]domain.AnimeListEntry, error) {
	userID, err := r.viewerID(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("anilist: fetching anime list for user %d", userID)

	const q = `
		query ($userId: Int) {
			MediaListCollection(userId: $userId, type: ANIME) {
				lists {
					entries {
						mediaId
						status
						media {
							title {
								english
								romaji
							}
						}
					}
				}
			}
		}`

	var resp struct {
		Data struct {
			MediaListCollection struct {
				Lists []struct {
					Entries []struct {
						MediaID int    `json:"mediaId"`
						Status  string `json:"status"`
						Media   struct {
							Title struct {
								English string `json:"english"`
								Romaji  string `json:"romaji"`
							} `json:"title"`
						} `json:"media"`
					} `json:"entries"`
				} `json:"lists"`
			} `json:"MediaListCollection"`
		} `json:"data"`
	}

	if err := r.query(ctx, q, map[string]any{"userId": userID}, &resp); err != nil {
		return nil, fmt.Errorf("fetching anime list: %w", err)
	}

	var entries []domain.AnimeListEntry
	for _, list := range resp.Data.MediaListCollection.Lists {
		for _, e := range list.Entries {
			title := e.Media.Title.English
			if title == "" {
				title = e.Media.Title.Romaji
			}
			entries = append(entries, domain.AnimeListEntry{
				AnilistId: domain.AniListID(strconv.Itoa(e.MediaID)),
				Title:     title,
				Status:    anilistStatus(e.Status),
			})
		}
	}

	log.Printf("anilist: fetched %d entries", len(entries))
	return entries, nil
}

// anilistStatus maps AniList's status strings to our domain WatchStatus.
func anilistStatus(s string) domain.WatchStatus {
	switch s {
	case "CURRENT":
		return domain.WatchStatusInProgress
	case "COMPLETED", "REPEATING":
		return domain.WatchStatusCompleted
	case "PAUSED":
		return domain.WatchStatusPaused
	case "DROPPED":
		return domain.WatchStatusDropped
	default:
		return domain.WatchStatusNotStarted
	}
}
