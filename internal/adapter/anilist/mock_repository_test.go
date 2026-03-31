package anilist

import (
	"path/filepath"
	"testing"

	"myapp/internal/domain"
)

func TestSaveLoadAnimeListEntriesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "list.json")
	want := []domain.AnimeListEntry{
		{AnilistId: "123", Title: "Test Anime", Status: domain.WatchStatusInProgress, Progress: 9},
		{AnilistId: "456", Title: "", Status: domain.WatchStatusCompleted, Progress: 24},
	}
	if err := SaveAnimeListEntries(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadMockEntriesFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}
