package app

import (
	"context"
	"errors"
	"myapp/internal/domain"
	"testing"
	"time"
)

type mockMediaHostRepo struct {
	anime    []domain.MediaHostAnime
	animeErr error
}

func (m *mockMediaHostRepo) GetAnime(_ context.Context) ([]domain.MediaHostAnime, error) {
	return m.anime, m.animeErr
}

// mockAnimeListRepo implements port.AnimeListRepository for testing.
type mockAnimeListRepo struct {
	entries   []domain.AnimeListEntry
	err       error
	saveErr   error
	saveCalls []anilistSaveCall
}

type anilistSaveCall struct {
	id       domain.AniListID
	status   domain.WatchStatus
	progress int
}

func (m *mockAnimeListRepo) GetAnimeList(_ context.Context) ([]domain.AnimeListEntry, error) {
	return m.entries, m.err
}

func (m *mockAnimeListRepo) SaveAnimeListEntry(_ context.Context, id domain.AniListID, status domain.WatchStatus, progress int) error {
	m.saveCalls = append(m.saveCalls, anilistSaveCall{id: id, status: status, progress: progress})
	return m.saveErr
}

// makeSyncService builds a SyncService backed by the provided mocks.
func makeSyncService(anime []domain.MediaHostAnime, animeErr error, mappingRepo *mockMappingSourceRepo) *SyncService {
	return NewSyncService(
		nil,
		&mockMediaHostRepo{anime: anime, animeErr: animeErr},
		NewMappingService(mappingRepo),
		DefaultSyncConfig(),
	)
}

// watched returns n watched episodes numbered sequentially from 1, with no timestamps.
func watched(n int) []domain.MediaHostEpisode {
	eps := make([]domain.MediaHostEpisode, n)
	for i := range eps {
		eps[i] = domain.MediaHostEpisode{Number: domain.MediaHostEpisodeNumber(i + 1), Watched: true}
	}
	return eps
}

// watchedAt returns n watched episodes all with the given last-watched timestamp.
func watchedAt(n int, t time.Time) []domain.MediaHostEpisode {
	eps := make([]domain.MediaHostEpisode, n)
	ts := t
	for i := range eps {
		eps[i] = domain.MediaHostEpisode{
			Number:        domain.MediaHostEpisodeNumber(i + 1),
			Watched:       true,
			LastWatchedAt: &ts,
		}
	}
	return eps
}

// unwatched returns n unwatched episodes numbered sequentially from 1.
func unwatched(n int) []domain.MediaHostEpisode {
	eps := make([]domain.MediaHostEpisode, n)
	for i := range eps {
		eps[i] = domain.MediaHostEpisode{Number: domain.MediaHostEpisodeNumber(i + 1), Watched: false}
	}
	return eps
}

func TestMergeAnimeStatusesForAnilist_duplicatePlexMatchesCollapse(t *testing.T) {
	id := domain.AniListID("1")
	parts := []domain.AnimeStatus{
		{AnilistId: id, PlexTitle: "Naruto", Status: domain.WatchStatusCompleted, WatchedEpisodes: 3, TotalEpisodes: 3},
		{AnilistId: id, PlexTitle: "Naruto", Status: domain.WatchStatusCompleted, WatchedEpisodes: 3, TotalEpisodes: 3},
		{AnilistId: id, PlexTitle: "Naruto", Status: domain.WatchStatusCompleted, WatchedEpisodes: 3, TotalEpisodes: 3},
	}
	got := mergeAnimeStatusesForAnilist(id, parts)
	want := parts[0]
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestCollapseDuplicatePlexTitles_fiveNarutoOneRow(t *testing.T) {
	in := []domain.AnimeStatus{
		{AnilistId: "300", PlexTitle: "Naruto", Status: domain.WatchStatusCompleted, WatchedEpisodes: 3, TotalEpisodes: 3},
		{AnilistId: "100", PlexTitle: "Naruto", Status: domain.WatchStatusCompleted, WatchedEpisodes: 3, TotalEpisodes: 3},
		{AnilistId: "200", PlexTitle: "naruto ", Status: domain.WatchStatusCompleted, WatchedEpisodes: 3, TotalEpisodes: 3},
	}
	got := collapseDuplicatePlexTitles(in)
	if len(got) != 1 {
		t.Fatalf("len %d want 1: %+v", len(got), got)
	}
	if got[0].AnilistId != "100" {
		t.Fatalf("expected smallest id when tied, got %+v", got[0])
	}
}

func TestCollapseDuplicatePlexTitles_prefersLargerScope(t *testing.T) {
	in := []domain.AnimeStatus{
		{AnilistId: "1", PlexTitle: "Naruto", WatchedEpisodes: 3, TotalEpisodes: 3},
		{AnilistId: "2", PlexTitle: "Naruto", WatchedEpisodes: 50, TotalEpisodes: 220},
	}
	got := collapseDuplicatePlexTitles(in)
	if len(got) != 1 || got[0].AnilistId != "2" || got[0].TotalEpisodes != 220 {
		t.Fatalf("got %+v", got)
	}
}

func TestCollapseDuplicatePlexTitles_prefersMoreWatchedOverLargerTotal(t *testing.T) {
	in := []domain.AnimeStatus{
		{AnilistId: "1", PlexTitle: "Love, Chunibyo & Other Delusions!", WatchedEpisodes: 2, TotalEpisodes: 16, Status: domain.WatchStatusDropped},
		{AnilistId: "2", PlexTitle: "Love, Chunibyo & Other Delusions!", WatchedEpisodes: 12, TotalEpisodes: 12, Status: domain.WatchStatusCompleted},
	}
	got := collapseDuplicatePlexTitles(in)
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].AnilistId != "2" || got[0].WatchedEpisodes != 12 || got[0].TotalEpisodes != 12 {
		t.Fatalf("want 12/12 complete copy, got %+v", got[0])
	}
}

func TestCollapseDuplicatePlexTitles_doesNotMergeEmptyTitlesAcrossIds(t *testing.T) {
	in := []domain.AnimeStatus{
		{AnilistId: "1", PlexTitle: "", WatchedEpisodes: 1, TotalEpisodes: 2},
		{AnilistId: "2", PlexTitle: "", WatchedEpisodes: 1, TotalEpisodes: 2},
	}
	got := collapseDuplicatePlexTitles(in)
	if len(got) != 2 {
		t.Fatalf("len %d want 2", len(got))
	}
}

func TestPlexTitleMergeKey_isLowercase(t *testing.T) {
	k := plexTitleMergeKey(domain.AnimeStatus{PlexTitle: "ABC", AnilistId: "1"})
	if k != "abc" {
		t.Fatalf("got %q", k)
	}
}

func TestMergeAnimeStatusesForAnilist_disjointSplitsSum(t *testing.T) {
	id := domain.AniListID("1")
	parts := []domain.AnimeStatus{
		{AnilistId: id, PlexTitle: "X", Status: domain.WatchStatusCompleted, WatchedEpisodes: 2, TotalEpisodes: 2},
		{AnilistId: id, PlexTitle: "X", Status: domain.WatchStatusInProgress, WatchedEpisodes: 1, TotalEpisodes: 3},
	}
	got := mergeAnimeStatusesForAnilist(id, parts)
	want := domain.AnimeStatus{
		AnilistId: id, PlexTitle: "X", Status: domain.WatchStatusInProgress,
		WatchedEpisodes: 3, TotalEpisodes: 5,
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestSyncAnime_MediaHostError(t *testing.T) {
	repoErr := errors.New("plex unavailable")
	svc := makeSyncService(nil, repoErr, &mockMappingSourceRepo{})

	_, err := svc.SyncAnime(context.Background())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected media host error, got %v", err)
	}
}

func TestSyncAnime_MappingError(t *testing.T) {
	repoErr := errors.New("mapping repo unavailable")
	svc := makeSyncService(
		[]domain.MediaHostAnime{{ID: "tvdb-1"}},
		nil,
		&mockMappingSourceRepo{tvDbToAniDbMappingErr: repoErr},
	)

	_, err := svc.SyncAnime(context.Background())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected mapping error, got %v", err)
	}
}

func TestSyncAnime_AnimeWithNoMappingIsSkipped(t *testing.T) {
	svc := makeSyncService(
		[]domain.MediaHostAnime{{ID: "tvdb-unknown"}},
		nil,
		&mockMappingSourceRepo{tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{}},
	)

	statuses, err := svc.SyncAnime(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected no statuses, got %d", len(statuses))
	}
}

func TestSyncAnime_NegativeOffsetIsSkipped(t *testing.T) {
	svc := makeSyncService(
		[]domain.MediaHostAnime{{
			ID:      "tvdb-1",
			Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: watched(12)}},
		}},
		nil,
		&mockMappingSourceRepo{
			tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
				"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: -1}},
			},
			aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
				"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
			},
		},
	)

	statuses, err := svc.SyncAnime(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected no statuses, got %d", len(statuses))
	}
}

func TestSyncAnime_InsufficientEpisodesIsSkipped(t *testing.T) {
	// Mapping claims 13 episodes starting at offset 12, but the season only has 12.
	svc := makeSyncService(
		[]domain.MediaHostAnime{{
			ID:      "tvdb-1",
			Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: watched(12)}},
		}},
		nil,
		&mockMappingSourceRepo{
			tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
				"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 12}},
			},
			aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
				"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 13},
			},
		},
	)

	statuses, err := svc.SyncAnime(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected no statuses, got %d", len(statuses))
	}
}

func TestSyncAnime_CompletionStatus(t *testing.T) {
	tests := []struct {
		name    string
		anime   []domain.MediaHostAnime
		mapping *mockMappingSourceRepo
		want    []domain.AnimeStatus
	}{
		{
			name: "all episodes watched",
			anime: []domain.MediaHostAnime{{
				ID:      "tvdb-1",
				Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: watched(12)}},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12}},
		},
		{
			name: "no episodes watched",
			anime: []domain.MediaHostAnime{{
				ID:      "tvdb-1",
				Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: unwatched(12)}},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusNotStarted, WatchedEpisodes: 0, TotalEpisodes: 12}},
		},
		{
			name: "some episodes watched without timestamps is in_progress",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{{
					Number:   1,
					Episodes: append(watched(6), unwatched(6)...),
				}},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusInProgress, WatchedEpisodes: 6, TotalEpisodes: 12}},
		},
		{
			// Split-cour show: TvDB holds all episodes in one season; two AniList
			// entries cover the first and second halves via episodeoffset.
			name: "split cour mapped via episode offset within one season",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{
					{Number: 1, Episodes: append(watched(13), watched(11)...)},
				},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {
						{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0},
						{TvDbID: "tvdb-1", AniDbID: "anidb-2", TvDbSeason: "1", EpisodeOffset: 13},
					},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 13},
					"anidb-2": {AniDbID: "anidb-2", AnilistId: "anilist-101", EpisodeCount: 11},
				},
			},
			want: []domain.AnimeStatus{
				{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 13, TotalEpisodes: 13},
				{AnilistId: "anilist-101", Status: domain.WatchStatusCompleted, WatchedEpisodes: 11, TotalEpisodes: 11},
			},
		},
		{
			// Shield Hero style: TvDB has a separate season per cours; each maps
			// to a distinct AniList entry with offset 0. Only S1 is watched.
			name: "each tvdb season maps to a separate anilist entry",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{
					{Number: 1, Episodes: watched(13)},
					{Number: 2, Episodes: unwatched(13)},
				},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {
						{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0},
						{TvDbID: "tvdb-1", AniDbID: "anidb-2", TvDbSeason: "2", EpisodeOffset: 0},
					},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 13},
					"anidb-2": {AniDbID: "anidb-2", AnilistId: "anilist-101", EpisodeCount: 13},
				},
			},
			want: []domain.AnimeStatus{
				{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 13, TotalEpisodes: 13},
				{AnilistId: "anilist-101", Status: domain.WatchStatusNotStarted, WatchedEpisodes: 0, TotalEpisodes: 13},
			},
		},
		{
			// Season list on MediaHostAnime may arrive in any order; we should
			// look up the season by number so order doesn't matter.
			name: "seasons listed out of order are resolved correctly by number",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{
					{Number: 2, Episodes: unwatched(12)},
					{Number: 1, Episodes: watched(12)},
				},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12}},
		},
		{
			name: "last watched over paused threshold is paused",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{{
					Number:   1,
					Episodes: append(watchedAt(6, time.Now().Add(-20*24*time.Hour)), unwatched(6)...),
				}},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusPaused, WatchedEpisodes: 6, TotalEpisodes: 12}},
		},
		{
			name: "last watched over dropped threshold is dropped",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{{
					Number:   1,
					Episodes: append(watchedAt(6, time.Now().Add(-45*24*time.Hour)), unwatched(6)...),
				}},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusDropped, WatchedEpisodes: 6, TotalEpisodes: 12}},
		},
		{
			name: "multiple anime each produce a status",
			anime: []domain.MediaHostAnime{
				{ID: "tvdb-1", Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: watched(12)}}},
				{ID: "tvdb-2", Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: unwatched(12)}}},
			},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
					"tvdb-2": {{TvDbID: "tvdb-2", AniDbID: "anidb-2", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
					"anidb-2": {AniDbID: "anidb-2", AnilistId: "anilist-200", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{
				{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12},
				{AnilistId: "anilist-200", Status: domain.WatchStatusNotStarted, WatchedEpisodes: 0, TotalEpisodes: 12},
			},
		},
		{
			name: "single season span overrides anime offline DB episodes set to one",
			anime: []domain.MediaHostAnime{{
				ID:      "tvdb-1",
				Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: watched(12)}},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 1},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12}},
		},
		{
			name: "same anilist id across tvdb seasons merges into one status",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{
					{Number: 1, Episodes: watched(3)},
					{Number: 2, Episodes: watched(3)},
				},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {
						{TvDbID: "tvdb-1", AniDbID: "anidb-1", TvDbSeason: "1", EpisodeOffset: 0},
						{TvDbID: "tvdb-1", AniDbID: "anidb-2", TvDbSeason: "2", EpisodeOffset: 0},
					},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-900", EpisodeCount: 3},
					"anidb-2": {AniDbID: "anidb-2", AnilistId: "anilist-900", EpisodeCount: 3},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-900", Status: domain.WatchStatusCompleted, WatchedEpisodes: 6, TotalEpisodes: 6}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := makeSyncService(tt.anime, nil, tt.mapping)

			got, err := svc.SyncAnime(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d statuses, got %d", len(tt.want), len(got))
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("status[%d]: expected %+v, got %+v", i, w, got[i])
				}
			}
		})
	}
}

func TestCompareWithAniList_AniListError(t *testing.T) {
	repoErr := errors.New("anilist unavailable")
	svc := NewSyncService(&mockAnimeListRepo{err: repoErr}, nil, nil, DefaultSyncConfig())

	_, err := svc.CompareWithAniList(context.Background(), []domain.AnimeStatus{})
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected anilist error, got %v", err)
	}
}

func TestCompareWithAniList(t *testing.T) {
	tests := []struct {
		name         string
		plexStatuses []domain.AnimeStatus
		anilistList  []domain.AnimeListEntry
		want         []domain.SyncResult
	}{
		{
			name: "not_started entries are excluded",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusNotStarted},
			},
			anilistList: []domain.AnimeListEntry{},
			want:        nil,
		},
		{
			name: "in_progress included with matching anilist status",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusInProgress, WatchedEpisodes: 5, TotalEpisodes: 12},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Status: domain.WatchStatusInProgress, Progress: 5},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress, AnilistProgress: 5, PlexWatchedEpisodes: 5, TotalEpisodes: 12},
			},
		},
		{
			name: "completed included with differing anilist status",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Status: domain.WatchStatusInProgress, Progress: 8},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: domain.WatchStatusInProgress, AnilistProgress: 8, PlexWatchedEpisodes: 12, TotalEpisodes: 12},
			},
		},
		{
			name: "not yet on anilist has empty anilist status",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12},
			},
			anilistList: []domain.AnimeListEntry{},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: "", AnilistProgress: 0, PlexWatchedEpisodes: 12, TotalEpisodes: 12},
			},
		},
		{
			name: "not_started excluded; in_progress, paused, dropped, completed included",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12},
				{AnilistId: "101", Status: domain.WatchStatusInProgress, WatchedEpisodes: 3, TotalEpisodes: 12},
				{AnilistId: "102", Status: domain.WatchStatusNotStarted},
				{AnilistId: "103", Status: domain.WatchStatusPaused, WatchedEpisodes: 4, TotalEpisodes: 12},
				{AnilistId: "104", Status: domain.WatchStatusDropped, WatchedEpisodes: 2, TotalEpisodes: 12},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Status: domain.WatchStatusInProgress, Progress: 6},
				{AnilistId: "101", Status: domain.WatchStatusInProgress, Progress: 3},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: domain.WatchStatusInProgress, AnilistProgress: 6, PlexWatchedEpisodes: 12, TotalEpisodes: 12},
				{AnilistId: "101", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress, AnilistProgress: 3, PlexWatchedEpisodes: 3, TotalEpisodes: 12},
				{AnilistId: "103", PlexStatus: domain.WatchStatusPaused, AnilistStatus: "", AnilistProgress: 0, PlexWatchedEpisodes: 4, TotalEpisodes: 12},
				{AnilistId: "104", PlexStatus: domain.WatchStatusDropped, AnilistStatus: "", AnilistProgress: 0, PlexWatchedEpisodes: 2, TotalEpisodes: 12},
			},
		},
		{
			name: "entry already completed on anilist is skipped",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusCompleted, WatchedEpisodes: 12, TotalEpisodes: 12},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Status: domain.WatchStatusCompleted, Progress: 12},
			},
			want: nil,
		},
		{
			name: "title falls back to plex title when anilist title is empty",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", PlexTitle: "Plex Show Name", Status: domain.WatchStatusInProgress, WatchedEpisodes: 1, TotalEpisodes: 12},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Title: "", Status: domain.WatchStatusInProgress, Progress: 1},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", Title: "Plex Show Name", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress, AnilistProgress: 1, PlexWatchedEpisodes: 1, TotalEpisodes: 12},
			},
		},
		{
			name: "anilist title takes priority over plex title",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", PlexTitle: "Plex Show Name", Status: domain.WatchStatusInProgress, WatchedEpisodes: 7, TotalEpisodes: 12},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Title: "AniList Title", Status: domain.WatchStatusInProgress, Progress: 7},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", Title: "AniList Title", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress, AnilistProgress: 7, PlexWatchedEpisodes: 7, TotalEpisodes: 12},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSyncService(&mockAnimeListRepo{entries: tt.anilistList}, nil, nil, DefaultSyncConfig())

			got, err := svc.CompareWithAniList(context.Background(), tt.plexStatuses)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d results, got %d", len(tt.want), len(got))
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("result[%d]: expected %+v, got %+v", i, w, got[i])
				}
			}
		})
	}
}

func TestApplyAniListUpdates(t *testing.T) {
	tests := []struct {
		name      string
		results   []domain.SyncResult
		saveErr   error
		wantCalls []anilistSaveCall
		wantCount int
		wantErr   bool
	}{
		{
			name:      "no rows need change",
			results:   []domain.SyncResult{{AnilistId: "1", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress, PlexWatchedEpisodes: 3, AnilistProgress: 3}},
			wantCalls: nil,
			wantCount: 0,
		},
		{
			name: "status mismatch triggers save",
			results: []domain.SyncResult{
				{AnilistId: "10", Title: "A", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: domain.WatchStatusInProgress, PlexWatchedEpisodes: 12, AnilistProgress: 8, TotalEpisodes: 12},
			},
			wantCalls: []anilistSaveCall{{id: "10", status: domain.WatchStatusCompleted, progress: 12}},
			wantCount: 1,
		},
		{
			name: "plex ahead on episodes only",
			results: []domain.SyncResult{
				{AnilistId: "11", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress, PlexWatchedEpisodes: 7, AnilistProgress: 5, TotalEpisodes: 12},
			},
			wantCalls: []anilistSaveCall{{id: "11", status: domain.WatchStatusInProgress, progress: 7}},
			wantCount: 1,
		},
		{
			name: "save error is returned",
			results: []domain.SyncResult{
				{AnilistId: "9", Title: "ErrShow", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: domain.WatchStatusInProgress, PlexWatchedEpisodes: 1, AnilistProgress: 0},
			},
			saveErr:   errors.New("api down"),
			wantCalls: []anilistSaveCall{{id: "9", status: domain.WatchStatusCompleted, progress: 1}},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAnimeListRepo{saveErr: tt.saveErr}
			svc := NewSyncService(repo, nil, nil, DefaultSyncConfig())
			gotCount, err := svc.ApplyAniListUpdates(context.Background(), tt.results)
			if gotCount != tt.wantCount {
				t.Fatalf("applied count: want %d, got %d", tt.wantCount, gotCount)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(repo.saveCalls) != len(tt.wantCalls) {
				t.Fatalf("save calls: want %d, got %d (%+v)", len(tt.wantCalls), len(repo.saveCalls), repo.saveCalls)
			}
			for i, w := range tt.wantCalls {
				if repo.saveCalls[i] != w {
					t.Errorf("saveCalls[%d]: want %+v, got %+v", i, w, repo.saveCalls[i])
				}
			}
		})
	}
}
