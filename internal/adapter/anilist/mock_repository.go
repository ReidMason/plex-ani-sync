package anilist

import (
	"context"
	"encoding/json"
	"fmt"
	"myapp/internal/domain"
	"os"
	"path/filepath"
)

// MockRepository implements port.AnimeListRepository with a fixed list (for
// local runs when the AniList API is unavailable).
type MockRepository struct {
	entries []domain.AnimeListEntry
}

func NewMockRepository(entries []domain.AnimeListEntry) *MockRepository {
	return &MockRepository{entries: append([]domain.AnimeListEntry(nil), entries...)}
}

func (r *MockRepository) GetAnimeList(_ context.Context) ([]domain.AnimeListEntry, error) {
	return append([]domain.AnimeListEntry(nil), r.entries...), nil
}

// DefaultMockEntriesPlex295222 matches internal/adapter/plex.MockRepository
// (TvDB 295222, Gate) and Anime-Lists master + anime-offline-database IDs:
// AniDB 10982 → AniList 20994 (12 eps), AniDB 11602 → AniList 21364 (12 eps, offset 12).
// Plex mock marks 22/24 episodes watched → first cour completed, second in progress.
// This mock marks the second cour as not started on AniList so CompareWithAniList shows a diff.
func DefaultMockEntriesPlex295222() []domain.AnimeListEntry {
	return []domain.AnimeListEntry{
		{AnilistId: "20994", Title: "GATE: Jieitai Kanochi nite, Kaku Tatakaeri (mock)", Status: domain.WatchStatusCompleted, Progress: 12},
		{AnilistId: "21364", Title: "GATE Season 2 (mock)", Status: domain.WatchStatusNotStarted, Progress: 0},
	}
}

type mockFileEntry struct {
	AnilistID string             `json:"anilist_id"`
	Title     string             `json:"title"`
	Status    domain.WatchStatus `json:"status"`
	Progress  int                `json:"progress,omitempty"`
}

// LoadMockEntriesFile reads JSON array of {anilist_id, title, status} where
// status is a domain watch status (e.g. in_progress, completed).
func LoadMockEntriesFile(path string) ([]domain.AnimeListEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading mock file: %w", err)
	}
	var rows []mockFileEntry
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("parsing mock file: %w", err)
	}
	out := make([]domain.AnimeListEntry, len(rows))
	for i, r := range rows {
		if r.AnilistID == "" {
			return nil, fmt.Errorf("mock file row %d: missing anilist_id", i)
		}
		out[i] = domain.AnimeListEntry{AnilistId: domain.AniListID(r.AnilistID), Title: r.Title, Status: r.Status, Progress: r.Progress}
	}
	return out, nil
}

// SaveAnimeListEntries writes entries as JSON (same format as LoadMockEntriesFile / ANILIST_MOCK_FILE).
func SaveAnimeListEntries(path string, entries []domain.AnimeListEntry) error {
	rows := make([]mockFileEntry, len(entries))
	for i, e := range entries {
		rows[i] = mockFileEntry{AnilistID: string(e.AnilistId), Title: e.Title, Status: e.Status, Progress: e.Progress}
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding list: %w", err)
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}
