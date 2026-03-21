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

func TestSyncAnime_InsufficientEpisodesIsSkipped(t *testing.T) {
	// Mapping claims 13 episodes starting at offset 12, but the anime only has 12.
	svc := makeSyncService(
		[]domain.MediaHostAnime{{
			ID:      "tvdb-1",
			Seasons: []domain.MediaHostSeason{{Number: 1, Episodes: watched(12)}},
		}},
		nil,
		&mockMappingSourceRepo{
			tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
				"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 12}},
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
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0}},
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
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0}},
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
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0}},
				},
				aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
					"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				},
			},
			want: []domain.AnimeStatus{{AnilistId: "anilist-100", Status: domain.WatchStatusInProgress}},
		},
		{
			name: "split cour mapped via episode offset",
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{
					{Number: 1, Episodes: watched(13)},
					{Number: 2, Episodes: watched(11)},
				},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {
						{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0},
						{TvDbID: "tvdb-1", AniDbID: "anidb-2", EpisodeOffset: 13},
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
			name: "seasons provided out of order are flattened correctly",
			// Season 2 (unwatched) is listed first; season 1 (watched) second.
			// The mapping covers absolute episodes 1-12, which should resolve to
			// season 1 after sorting — so the result should be completed.
			anime: []domain.MediaHostAnime{{
				ID: "tvdb-1",
				Seasons: []domain.MediaHostSeason{
					{Number: 2, Episodes: unwatched(12)},
					{Number: 1, Episodes: watched(12)},
				},
			}},
			mapping: &mockMappingSourceRepo{
				tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0}},
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
					"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0}},
					"tvdb-2": {{TvDbID: "tvdb-2", AniDbID: "anidb-2", EpisodeOffset: 0}},
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
