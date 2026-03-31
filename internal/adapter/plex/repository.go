package plex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"myapp/internal/domain"
	"net/http"
	"sort"
	"strings"
	"time"
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
			// LastViewedAt is a Unix timestamp (seconds). Only present when watched.
			LastViewedAt int64 `json:"lastViewedAt"`
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

	return mergeAnimeByTvDbID(anime), nil
}

// mergeAnimeByTvDbID combines duplicate series that appear in more than one
// Plex library section (same TheTVDB id). Episodes are merged per season with
// de-duplication by episode number; watched if any copy is watched.
func mergeAnimeByTvDbID(in []domain.MediaHostAnime) []domain.MediaHostAnime {
	if len(in) == 0 {
		return nil
	}

	type agg struct {
		title   domain.MediaHostAnimeTitle
		seasons map[int]map[int]domain.MediaHostEpisode // seasonNum -> epNum -> ep
	}
	byID := make(map[domain.TvDbID]*agg)

	for _, a := range in {
		ac := byID[a.ID]
		if ac == nil {
			ac = &agg{seasons: make(map[int]map[int]domain.MediaHostEpisode)}
			byID[a.ID] = ac
		}
		if a.Title != "" {
			ac.title = a.Title
		}
		for _, s := range a.Seasons {
			sn := int(s.Number)
			if ac.seasons[sn] == nil {
				ac.seasons[sn] = make(map[int]domain.MediaHostEpisode)
			}
			for _, ep := range s.Episodes {
				en := int(ep.Number)
				if prev, ok := ac.seasons[sn][en]; ok {
					ac.seasons[sn][en] = mergeMediaHostEpisode(prev, ep)
				} else {
					ac.seasons[sn][en] = ep
				}
			}
		}
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)

	out := make([]domain.MediaHostAnime, 0, len(byID))
	for _, sid := range ids {
		id := domain.TvDbID(sid)
		a := byID[id]
		var seasonNums []int
		for sn := range a.seasons {
			seasonNums = append(seasonNums, sn)
		}
		sort.Ints(seasonNums)
		seasons := make([]domain.MediaHostSeason, 0, len(seasonNums))
		for _, sn := range seasonNums {
			epMap := a.seasons[sn]
			nums := make([]int, 0, len(epMap))
			for n := range epMap {
				nums = append(nums, n)
			}
			sort.Ints(nums)
			eps := make([]domain.MediaHostEpisode, 0, len(nums))
			for _, n := range nums {
				eps = append(eps, epMap[n])
			}
			seasons = append(seasons, domain.MediaHostSeason{
				Number:   domain.MediaHostSeasonNumber(sn),
				Episodes: eps,
			})
		}
		out = append(out, domain.MediaHostAnime{
			ID:      id,
			Title:   a.title,
			Seasons: seasons,
		})
	}
	return out
}

func mergeMediaHostEpisode(a, b domain.MediaHostEpisode) domain.MediaHostEpisode {
	watched := a.Watched || b.Watched
	var last *time.Time
	switch {
	case a.LastWatchedAt != nil && b.LastWatchedAt != nil:
		if a.LastWatchedAt.After(*b.LastWatchedAt) {
			last = a.LastWatchedAt
		} else {
			last = b.LastWatchedAt
		}
	case a.LastWatchedAt != nil:
		last = a.LastWatchedAt
	default:
		last = b.LastWatchedAt
	}
	return domain.MediaHostEpisode{
		Number:        a.Number,
		Watched:       watched,
		LastWatchedAt: last,
	}
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
		episode := domain.MediaHostEpisode{
			Number:  domain.MediaHostEpisodeNumber(ep.Index),
			Watched: ep.ViewCount > 0,
		}
		if ep.LastViewedAt > 0 {
			t := time.Unix(ep.LastViewedAt, 0)
			episode.LastWatchedAt = &t
		}
		seasonEps[key] = append(seasonEps[key], episode)
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
func tvdbGUID(guids []struct {
	ID string `json:"id"`
}, legacy string) (domain.TvDbID, bool) {
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
