package animelists

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"myapp/internal/domain"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const (
	animeListURL  = "https://raw.githubusercontent.com/Anime-Lists/anime-lists/refs/heads/master/anime-list-master.xml"
	cacheFileName = "anime-list.xml"
)

type XMLRepository struct {
	httpClient *http.Client
	cacheDir   string
}

func NewXMLRepository(cacheDir string) *XMLRepository {
	return &XMLRepository{
		httpClient: &http.Client{},
		cacheDir:   cacheDir,
	}
}

type animeListXML struct {
	XMLName xml.Name   `xml:"anime-list"`
	Anime   []animeXML `xml:"anime"`
}

type animeXML struct {
	AniDbID       string `xml:"anidbid,attr"`
	TvDbID        string `xml:"tvdbid,attr"`
	EpisodeOffset string `xml:"episodeoffset,attr"`
}

func (r *XMLRepository) fetchOrLoad(ctx context.Context) ([]byte, error) {
	cachePath := filepath.Join(r.cacheDir, cacheFileName)

	data, err := os.ReadFile(cachePath)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading cache file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, animeListURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading anime list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d downloading anime list", resp.StatusCode)
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

	return data, nil
}

func (r *XMLRepository) GetTvDbToAniDbMapping(ctx context.Context) (map[domain.TvDbID][]domain.TvDbToAniDbMapping, error) {
	data, err := r.fetchOrLoad(ctx)
	if err != nil {
		return nil, err
	}

	var list animeListXML
	if err := xml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing anime list XML: %w", err)
	}

	mappings := make(map[domain.TvDbID][]domain.TvDbToAniDbMapping)
	for _, a := range list.Anime {
		if a.TvDbID == "" {
			continue
		}

		var offset int
		if a.EpisodeOffset != "" {
			offset, err = strconv.Atoi(a.EpisodeOffset)
			if err != nil {
				return nil, fmt.Errorf("parsing episode offset %q for anidbid %s: %w", a.EpisodeOffset, a.AniDbID, err)
			}
		}

		tvDbID := domain.TvDbID(a.TvDbID)
		mappings[tvDbID] = append(mappings[tvDbID], domain.TvDbToAniDbMapping{
			TvDbID:        tvDbID,
			AniDbID:       domain.AniDbID(a.AniDbID),
			EpisodeOffset: domain.EpisodeOffset(offset),
		})
	}

	return mappings, nil
}

