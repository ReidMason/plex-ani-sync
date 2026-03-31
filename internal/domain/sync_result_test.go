package domain

import "testing"

func TestSyncResult_TargetWatchedForAniList(t *testing.T) {
	r := SyncResult{AnilistProgress: 10, PlexWatchedEpisodes: 5}
	if got := r.TargetWatchedForAniList(); got != 10 {
		t.Fatalf("anilist ahead: got %d want 10", got)
	}
	r = SyncResult{AnilistProgress: 3, PlexWatchedEpisodes: 8}
	if got := r.TargetWatchedForAniList(); got != 8 {
		t.Fatalf("plex ahead: got %d want 8", got)
	}
}

func TestSyncResult_NeedsAniListChange(t *testing.T) {
	r := SyncResult{PlexStatus: WatchStatusCompleted, AnilistStatus: WatchStatusInProgress, PlexWatchedEpisodes: 5, AnilistProgress: 10}
	if !r.NeedsAniListChange() {
		t.Fatal("status diff should need change")
	}
	r = SyncResult{PlexStatus: WatchStatusInProgress, AnilistStatus: WatchStatusInProgress, PlexWatchedEpisodes: 5, AnilistProgress: 10}
	if r.NeedsAniListChange() {
		t.Fatal("anilist ahead on eps, same status: should not need change")
	}
	r = SyncResult{PlexStatus: WatchStatusInProgress, AnilistStatus: WatchStatusInProgress, PlexWatchedEpisodes: 10, AnilistProgress: 5}
	if !r.NeedsAniListChange() {
		t.Fatal("plex ahead on eps should need change")
	}
}
