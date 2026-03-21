package animelists

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"myapp/internal/domain"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	offlineDBURL      = "https://github.com/manami-project/anime-offline-database/releases/download/latest/anime-offline-database-minified.json"
	offlineDBFileName = "anime-offline-database.json"

	anidbSourcePrefix   = "https://anidb.net/anime/"
	anilistSourcePrefix = "https://anilist.co/anime/"
)

type OfflineDBRepository struct {
	httpClient *http.Client
	cacheDir   string

	once     sync.Once
	mappings map[domain.AniDbID]domain.AniDbToListIdMapping
	loadErr  error
}

func NewOfflineDBRepository(cacheDir string) *OfflineDBRepository {
	return &OfflineDBRepository{
		httpClient: &http.Client{},
		cacheDir:   cacheDir,
	}
}

type offlineDBJSON struct {
	Data []offlineDBEntry `json:"data"`
}

type offlineDBEntry struct {
	Sources  []string `json:"sources"`
	Episodes int      `json:"episodes"`
}

func (r *OfflineDBRepository) fetchOrLoad(ctx context.Context) ([]byte, error) {
	cachePath := filepath.Join(r.cacheDir, offlineDBFileName)

	data, err := os.ReadFile(cachePath)
	if err == nil {
		log.Printf("animelists: loaded AniDB→AniList mapping from cache (%s)", cachePath)
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading cache file: %w", err)
	}

	log.Printf("animelists: downloading AniDB→AniList mapping from %s", offlineDBURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, offlineDBURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading offline database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d downloading offline database", resp.StatusCode)
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if err := os.MkdirAll(r.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return nil, fmt.Errorf("writing cache file: %w", err)
	}

	log.Printf("animelists: cached AniDB→AniList mapping to %s (%d bytes)", cachePath, len(data))
	return data, nil
}

func (r *OfflineDBRepository) GetAniDbToListIdMapping(ctx context.Context) (map[domain.AniDbID]domain.AniDbToListIdMapping, error) {
	r.once.Do(func() { r.mappings, r.loadErr = r.load(ctx) })
	return r.mappings, r.loadErr
}

func (r *OfflineDBRepository) load(ctx context.Context) (map[domain.AniDbID]domain.AniDbToListIdMapping, error) {
	data, err := r.fetchOrLoad(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("animelists: parsing AniDB→AniList mapping (%d bytes)", len(data))
	var db offlineDBJSON
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("parsing offline database JSON: %w", err)
	}
	log.Printf("animelists: loaded %d AniDB→AniList entries", len(db.Data))

	mappings := make(map[domain.AniDbID]domain.AniDbToListIdMapping)
	for _, entry := range db.Data {
		anidbID, anilistID := extractSourceIDs(entry.Sources)
		if anidbID == "" || anilistID == "" {
			continue
		}

		mappings[anidbID] = domain.AniDbToListIdMapping{
			AniDbID:      anidbID,
			AnilistId:    anilistID,
			EpisodeCount: entry.Episodes,
		}
	}

	return mappings, nil
}

// extractSourceIDs pulls the AniDB and AniList IDs out of a sources URL list.
func extractSourceIDs(sources []string) (anidbID domain.AniDbID, anilistID domain.AniListID) {
	for _, source := range sources {
		if id, ok := trimPrefix(source, anidbSourcePrefix); ok {
			anidbID = domain.AniDbID(id)
		}
		if id, ok := trimPrefix(source, anilistSourcePrefix); ok {
			anilistID = domain.AniListID(id)
		}
	}
	return
}

func trimPrefix(s, prefix string) (string, bool) {
	trimmed := strings.TrimPrefix(s, prefix)
	if trimmed == s || trimmed == "" {
		return "", false
	}
	return trimmed, true
}
