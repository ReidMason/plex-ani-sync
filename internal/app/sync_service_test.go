package app

import (
	"context"
	"errors"
	"myapp/internal/domain"
	"testing"
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
	entries []domain.AnimeListEntry
	err     error
}

func (m *mockAnimeListRepo) GetAnimeList(_ context.Context) ([]domain.AnimeListEntry, error) {
	return m.entries, m.err
}

// makeSyncService builds a SyncService backed by the provided mocks.
// animeListRepo is not yet used in SyncAnime so nil is passed.
func makeSyncService(anime []domain.MediaHostAnime, animeErr error, mappingRepo *mockMappingSourceRepo) *SyncService {
	return NewSyncService(
		nil,
		&mockMediaHostRepo{anime: anime, animeErr: animeErr},
		NewMappingService(mappingRepo),
	)
}

// watched returns n watched episodes numbered sequentially from 1.
func watched(n int) []domain.MediaHostEpisode {
	eps := make([]domain.MediaHostEpisode, n)
	for i := range eps {
		eps[i] = domain.MediaHostEpisode{Number: domain.MediaHostEpisodeNumber(i + 1), Watched: true}
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
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted}},
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
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusNotStarted}},
		},
		{
			name: "some episodes watched",
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
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusInProgress}},
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
				{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted},
				{AnilistId: "anilist-101", Status: domain.WatchStatusCompleted},
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
				{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted},
				{AnilistId: "anilist-101", Status: domain.WatchStatusNotStarted},
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
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted}},
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
				{AnilistId: "anilist-100", Status: domain.WatchStatusCompleted},
				{AnilistId: "anilist-200", Status: domain.WatchStatusNotStarted},
			},
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
	svc := NewSyncService(&mockAnimeListRepo{err: repoErr}, nil, nil)

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
				{AnilistId: "100", Status: domain.WatchStatusInProgress},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Status: domain.WatchStatusInProgress},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress},
			},
		},
		{
			name: "completed included with differing anilist status",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusCompleted},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Status: domain.WatchStatusInProgress},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: domain.WatchStatusInProgress},
			},
		},
		{
			name: "not yet on anilist has empty anilist status",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusCompleted},
			},
			anilistList: []domain.AnimeListEntry{},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: ""},
			},
		},
		{
			name: "mixed statuses only include in_progress and completed",
			plexStatuses: []domain.AnimeStatus{
				{AnilistId: "100", Status: domain.WatchStatusCompleted},
				{AnilistId: "101", Status: domain.WatchStatusInProgress},
				{AnilistId: "102", Status: domain.WatchStatusNotStarted},
			},
			anilistList: []domain.AnimeListEntry{
				{AnilistId: "100", Status: domain.WatchStatusInProgress},
				{AnilistId: "101", Status: domain.WatchStatusInProgress},
			},
			want: []domain.SyncResult{
				{AnilistId: "100", PlexStatus: domain.WatchStatusCompleted, AnilistStatus: domain.WatchStatusInProgress},
				{AnilistId: "101", PlexStatus: domain.WatchStatusInProgress, AnilistStatus: domain.WatchStatusInProgress},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSyncService(&mockAnimeListRepo{entries: tt.anilistList}, nil, nil)

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
