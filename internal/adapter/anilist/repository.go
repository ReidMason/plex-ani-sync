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
	"strings"
	"time"
)

const graphqlEndpoint = "https://graphql.anilist.co"

// To obtain a token:
//  1. Go to https://anilist.co/settings/developer and create a new client.
//  2. Visit https://anilist.co/api/v2/oauth/authorize?client_id={clientId}&response_type=token
//  3. Authorize, then copy the access_token value from the redirect URL fragment.
//  4. Set it as ANILIST_TOKEN in your .env file.
//
// For local runs without the API, set ANILIST_MOCK=1 (see mock_repository.go).
// Set saveListPath (e.g. via ANILIST_SAVE_LIST) to persist each successful list fetch as JSON for ANILIST_MOCK_FILE.
// Set ANILIST_APPLY=1 to run SaveMediaListEntry for rows that need updates (cmd/server).
// Safety helpers (same binary): ANILIST_BACKUP_BEFORE_APPLY=1 writes your full list to
// ANILIST_BACKUP_DIR (default data/) as anilist-backup-YYYYMMDD-HHMMSS.json before any save;
// ANILIST_SAVE_LIST=path saves the fetched list on each run (no writes). Default is dry-run until ANILIST_APPLY is set.

type Repository struct {
	httpClient   *http.Client
	token        string
	saveListPath string
}

func NewRepository(token, saveListPath string) *Repository {
	return &Repository{
		httpClient:   &http.Client{},
		token:        token,
		saveListPath: saveListPath,
	}
}

func (r *Repository) graphql(ctx context.Context, q string, variables map[string]any, dataInto any) error {
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
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AniList HTTP %d: %s", resp.StatusCode, summarizeAniListErrorBody(data))
	}

	return decodeAniListGraphQLResponse(data, dataInto)
}

func decodeAniListGraphQLResponse(raw []byte, dataInto any) error {
	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	if len(env.Errors) > 0 && env.Errors[0].Message != "" {
		return fmt.Errorf("graphql: %s", env.Errors[0].Message)
	}
	if dataInto == nil {
		return nil
	}
	s := strings.TrimSpace(string(env.Data))
	if s == "" || s == "null" {
		return fmt.Errorf("graphql: empty data")
	}
	if err := json.Unmarshal(env.Data, dataInto); err != nil {
		return fmt.Errorf("parsing data: %w", err)
	}
	return nil
}

// summarizeAniListErrorBody prefers the first GraphQL error message when present
// (e.g. API disabled notices), otherwise returns a trimmed raw body.
func summarizeAniListErrorBody(data []byte) string {
	const max = 500
	var payload struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(data, &payload) == nil && len(payload.Errors) > 0 && payload.Errors[0].Message != "" {
		return payload.Errors[0].Message
	}
	s := strings.TrimSpace(string(data))
	if len(s) > max {
		return s[:max] + "..."
	}
	if s == "" {
		return "(empty body)"
	}
	return s
}

func (r *Repository) viewerID(ctx context.Context) (int, error) {
	const q = `query { Viewer { id } }`

	var data struct {
		Viewer struct {
			ID int `json:"id"`
		} `json:"Viewer"`
	}

	if err := r.graphql(ctx, q, nil, &data); err != nil {
		return 0, fmt.Errorf("fetching viewer: %w", err)
	}
	return data.Viewer.ID, nil
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
						progress
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

	var data struct {
		MediaListCollection struct {
			Lists []struct {
				Entries []struct {
					MediaID  int    `json:"mediaId"`
					Status   string `json:"status"`
					Progress int    `json:"progress"`
					Media    struct {
						Title struct {
							English string `json:"english"`
							Romaji  string `json:"romaji"`
						} `json:"title"`
					} `json:"media"`
				} `json:"entries"`
			} `json:"lists"`
		} `json:"MediaListCollection"`
	}

	if err := r.graphql(ctx, q, map[string]any{"userId": userID}, &data); err != nil {
		return nil, fmt.Errorf("fetching anime list: %w", err)
	}

	var entries []domain.AnimeListEntry
	for _, list := range data.MediaListCollection.Lists {
		for _, e := range list.Entries {
			title := e.Media.Title.English
			if title == "" {
				title = e.Media.Title.Romaji
			}
			entries = append(entries, domain.AnimeListEntry{
				AnilistId: domain.AniListID(strconv.Itoa(e.MediaID)),
				Title:     title,
				Status:    anilistStatus(e.Status),
				Progress:  e.Progress,
			})
		}
	}

	if r.saveListPath != "" {
		if err := SaveAnimeListEntries(r.saveListPath, entries); err != nil {
			return nil, fmt.Errorf("saving anime list to %s: %w", r.saveListPath, err)
		}
		log.Printf("anilist: wrote %d entries to %s", len(entries), r.saveListPath)
	}

	log.Printf("anilist: fetched %d entries", len(entries))
	return entries, nil
}

// SaveAnimeListEntry calls SaveMediaListEntry (create or update). Sleeps briefly
// after success to stay under AniList's per-minute rate limit (~90/min).
func (r *Repository) SaveAnimeListEntry(ctx context.Context, mediaID domain.AniListID, status domain.WatchStatus, progress int) error {
	id, err := strconv.Atoi(string(mediaID))
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid media id %q", mediaID)
	}
	st, err := domainWatchStatusToMediaListStatus(status)
	if err != nil {
		return err
	}

	const q = `
mutation ($mediaId: Int!, $status: MediaListStatus, $progress: Int) {
  SaveMediaListEntry(mediaId: $mediaId, status: $status, progress: $progress) {
    id
    status
    progress
  }
}`

	var data struct {
		SaveMediaListEntry struct {
			ID       int    `json:"id"`
			Status   string `json:"status"`
			Progress int    `json:"progress"`
		} `json:"SaveMediaListEntry"`
	}

	if err := r.graphql(ctx, q, map[string]any{
		"mediaId":  id,
		"status":   st,
		"progress": progress,
	}, &data); err != nil {
		return err
	}

	log.Printf("anilist: saved mediaId=%d status=%s progress=%d (entry id=%d)",
		id, data.SaveMediaListEntry.Status, data.SaveMediaListEntry.Progress, data.SaveMediaListEntry.ID)
	time.Sleep(700 * time.Millisecond)
	return nil
}

func domainWatchStatusToMediaListStatus(s domain.WatchStatus) (string, error) {
	switch s {
	case domain.WatchStatusInProgress:
		return "CURRENT", nil
	case domain.WatchStatusCompleted:
		return "COMPLETED", nil
	case domain.WatchStatusPaused:
		return "PAUSED", nil
	case domain.WatchStatusDropped:
		return "DROPPED", nil
	case domain.WatchStatusNotStarted:
		return "PLANNING", nil
	default:
		return "", fmt.Errorf("unsupported watch status %q", s)
	}
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
