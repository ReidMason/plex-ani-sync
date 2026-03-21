package plex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"myapp/internal/domain"
	"net/http"
	"strings"
)

type Repository struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func NewRepository(baseURL, token string) *Repository {
	return &Repository{
		httpClient: &http.Client{},
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
	}
}

// -- JSON response shapes ----------------------------------------------------

type sectionsResponse struct {
	MediaContainer struct {
		Directory []struct {
			Key   string `json:"key"`
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"Directory"`
	} `json:"MediaContainer"`
}

type showsResponse struct {
	MediaContainer struct {
		Metadata []struct {
			RatingKey string `json:"ratingKey"`
			Title     string `json:"title"`
			// LegacyGuid is the older single-string agent guid
			// e.g. "com.plexapp.agents.thetvdb://295222/1/1?lang=en".
			// Declaring it explicitly prevents Go's case-insensitive decoder
			// from matching this string field against the Guid slice below.
			LegacyGuid string `json:"guid"`
			// Guid is the modern per-provider array, populated when includeGuids=1.
			Guid []struct {
				ID string `json:"id"`
			} `json:"Guid"`
		} `json:"Metadata"`
	} `json:"MediaContainer"`
}

type episodesResponse struct {
	MediaContainer struct {
		Metadata []struct {
			Index                int    `json:"index"`
			ParentIndex          int    `json:"parentIndex"`
			GrandparentRatingKey string `json:"grandparentRatingKey"`
			ViewCount            int    `json:"viewCount"`
		} `json:"Metadata"`
	} `json:"MediaContainer"`
}

// ----------------------------------------------------------------------------

func (r *Repository) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Plex-Token", r.token)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %d", path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parsing response from %s: %w", path, err)
	}

	return nil
}

func (r *Repository) GetAnime(ctx context.Context) ([]domain.MediaHostAnime, error) {
	var sections sectionsResponse
	if err := r.get(ctx, "/library/sections/all", &sections); err != nil {
		return nil, fmt.Errorf("fetching library sections: %w", err)
	}

	var anime []domain.MediaHostAnime
	for _, section := range sections.MediaContainer.Directory {
		if section.Type != "show" {
			continue
		}

		log.Printf("plex: scanning section %q", section.Title)
		sectionAnime, err := r.getAnimeForSection(ctx, section.Key)
		if err != nil {
			return nil, fmt.Errorf("section %q (%s): %w", section.Title, section.Key, err)
		}
		log.Printf("plex: found %d shows in section %q", len(sectionAnime), section.Title)
		anime = append(anime, sectionAnime...)
	}

	return anime, nil
}

func (r *Repository) getAnimeForSection(ctx context.Context, sectionKey string) ([]domain.MediaHostAnime, error) {
	// includeGuids=1 is required for Plex to include external IDs (e.g. tvdb://) in the response.
	log.Printf("plex: fetching shows for section %s", sectionKey)
	var shows showsResponse
	if err := r.get(ctx, fmt.Sprintf("/library/sections/%s/all?includeGuids=1", sectionKey), &shows); err != nil {
		return nil, fmt.Errorf("fetching shows: %w", err)
	}

	// Map ratingKey → TvDB ID (skip shows with no TvDB GUID).
	tvdbByKey := make(map[string]domain.TvDbID)
	titleByKey := make(map[string]domain.MediaHostAnimeTitle)
	for _, show := range shows.MediaContainer.Metadata {
		if id, ok := tvdbGUID(show.Guid, show.LegacyGuid); ok {
			tvdbByKey[show.RatingKey] = id
			titleByKey[show.RatingKey] = domain.MediaHostAnimeTitle(show.Title)
		}
	}

	if len(tvdbByKey) == 0 {
		return nil, nil
	}

	log.Printf("plex: fetching all episodes for section %s (%d shows with TvDB IDs)", sectionKey, len(tvdbByKey))
	var episodes episodesResponse
	if err := r.get(ctx, fmt.Sprintf("/library/sections/%s/allLeaves", sectionKey), &episodes); err != nil {
		return nil, fmt.Errorf("fetching episodes: %w", err)
	}
	log.Printf("plex: fetched %d episodes", len(episodes.MediaContainer.Metadata))

	// Group episodes: showRatingKey → seasonNumber → []MediaHostEpisode
	type showSeasonKey struct {
		showKey   string
		seasonNum int
	}
	seasonEps := make(map[showSeasonKey][]domain.MediaHostEpisode)
	for _, ep := range episodes.MediaContainer.Metadata {
		if _, ok := tvdbByKey[ep.GrandparentRatingKey]; !ok {
			continue
		}
		key := showSeasonKey{ep.GrandparentRatingKey, ep.ParentIndex}
		seasonEps[key] = append(seasonEps[key], domain.MediaHostEpisode{
			Number:  domain.MediaHostEpisodeNumber(ep.Index),
			Watched: ep.ViewCount > 0,
		})
	}

	// Collect unique show keys seen in the episode list.
	showKeys := make(map[string]struct{})
	for k := range seasonEps {
		showKeys[k.showKey] = struct{}{}
	}

	anime := make([]domain.MediaHostAnime, 0, len(showKeys))
	for showKey := range showKeys {
		var seasons []domain.MediaHostSeason
		for key, eps := range seasonEps {
			if key.showKey != showKey {
				continue
			}
			seasons = append(seasons, domain.MediaHostSeason{
				Number:   domain.MediaHostSeasonNumber(key.seasonNum),
				Episodes: eps,
			})
		}
		anime = append(anime, domain.MediaHostAnime{
			ID:      tvdbByKey[showKey],
			Title:   titleByKey[showKey],
			Seasons: seasons,
		})
	}

	return anime, nil
}

// tvdbGUID extracts the TvDB ID, first from the modern Guid slice
// (e.g. "tvdb://295222") then falling back to the legacy agent string
// (e.g. "com.plexapp.agents.thetvdb://295222/1/1?lang=en").
func tvdbGUID(guids []struct{ ID string `json:"id"` }, legacy string) (domain.TvDbID, bool) {
	for _, g := range guids {
		if id, ok := strings.CutPrefix(g.ID, "tvdb://"); ok && id != "" {
			return domain.TvDbID(id), true
		}
	}
	// Legacy format: "com.plexapp.agents.thetvdb://295222/1/1?lang=en"
	if i := strings.Index(legacy, "thetvdb://"); i != -1 {
		rest := legacy[i+len("thetvdb://"):]
		if id, _, _ := strings.Cut(rest, "/"); id != "" {
			return domain.TvDbID(id), true
		}
	}
	return "", false
}
