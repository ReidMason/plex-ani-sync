package app

import (
	"context"
	"errors"
	"myapp/internal/domain"
	"testing"
)

// mockMappingSourceRepo implements port.MappingSourceRepository for testing.
type mockMappingSourceRepo struct {
	tvDbToAniDbMapping    map[domain.TvDbID][]domain.TvDbToAniDbMapping
	tvDbToAniDbMappingErr error

	aniDbToListIdMapping    map[domain.AniDbID]domain.AniDbToListIdMapping
	aniDbToListIdMappingErr error
}

func (m *mockMappingSourceRepo) GetTvDbToAniDbMapping(_ context.Context) (map[domain.TvDbID][]domain.TvDbToAniDbMapping, error) {
	return m.tvDbToAniDbMapping, m.tvDbToAniDbMappingErr
}

func (m *mockMappingSourceRepo) GetAniDbToListIdMapping(_ context.Context) (map[domain.AniDbID]domain.AniDbToListIdMapping, error) {
	return m.aniDbToListIdMapping, m.aniDbToListIdMappingErr
}

func TestGetMapping_TvDbToAniDbMappingError(t *testing.T) {
	repoErr := errors.New("repo unavailable")
	svc := NewMappingService(&mockMappingSourceRepo{
		tvDbToAniDbMappingErr: repoErr,
	})

	_, err := svc.GetMapping(context.Background(), "tvdb-1")
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestGetMapping_NoMappingFound(t *testing.T) {
	svc := NewMappingService(&mockMappingSourceRepo{
		tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{},
	})

	_, err := svc.GetMapping(context.Background(), "tvdb-unknown")
	if !errors.Is(err, ErrNoMappingFound) {
		t.Fatalf("expected ErrNoMappingFound, got %v", err)
	}
}

func TestGetMapping_AniDbToListIdMappingError(t *testing.T) {
	repoErr := errors.New("list id repo unavailable")
	svc := NewMappingService(&mockMappingSourceRepo{
		tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
			"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0}},
		},
		aniDbToListIdMappingErr: repoErr,
	})

	_, err := svc.GetMapping(context.Background(), "tvdb-1")
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestGetMapping_ReturnsCorrectMappings(t *testing.T) {
	tests := []struct {
		name                 string
		tvDbToAniDbMapping   map[domain.TvDbID][]domain.TvDbToAniDbMapping
		aniDbToListIdMapping map[domain.AniDbID]domain.AniDbToListIdMapping
		tvDbID               domain.TvDbID
		want                 []domain.Mapping
	}{
		{
			name: "single anidb mapping",
			tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
				"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 3}},
			},
			aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
				"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
			},
			tvDbID: "tvdb-1",
			want: []domain.Mapping{
				{TvDbID: "tvdb-1", AnilistId: "anilist-100", EpisodeOffset: 3, EpisodeCount: 12},
			},
		},
		{
			name: "anidb id missing from list id mapping is skipped",
			tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
				"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-missing", EpisodeOffset: 0}},
			},
			aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{},
			tvDbID:               "tvdb-1",
			want:                 []domain.Mapping{},
		},
		{
			name: "only returns mappings for the requested tvdb id",
			tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
				"tvdb-1": {{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0}},
				"tvdb-2": {{TvDbID: "tvdb-2", AniDbID: "anidb-2", EpisodeOffset: 0}},
			},
			aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
				"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 12},
				"anidb-2": {AniDbID: "anidb-2", AnilistId: "anilist-200", EpisodeCount: 24},
			},
			tvDbID: "tvdb-1",
			want: []domain.Mapping{
				{TvDbID: "tvdb-1", AnilistId: "anilist-100", EpisodeOffset: 0, EpisodeCount: 12},
			},
		},
		{
			name: "multiple anidb mappings",
			tvDbToAniDbMapping: map[domain.TvDbID][]domain.TvDbToAniDbMapping{
				"tvdb-1": {
					{TvDbID: "tvdb-1", AniDbID: "anidb-1", EpisodeOffset: 0},
					{TvDbID: "tvdb-1", AniDbID: "anidb-2", EpisodeOffset: 13},
				},
			},
			aniDbToListIdMapping: map[domain.AniDbID]domain.AniDbToListIdMapping{
				"anidb-1": {AniDbID: "anidb-1", AnilistId: "anilist-100", EpisodeCount: 13},
				"anidb-2": {AniDbID: "anidb-2", AnilistId: "anilist-101", EpisodeCount: 12},
			},
			tvDbID: "tvdb-1",
			want: []domain.Mapping{
				{TvDbID: "tvdb-1", AnilistId: "anilist-100", EpisodeOffset: 0, EpisodeCount: 13},
				{TvDbID: "tvdb-1", AnilistId: "anilist-101", EpisodeOffset: 13, EpisodeCount: 12},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMappingService(&mockMappingSourceRepo{
				tvDbToAniDbMapping:   tt.tvDbToAniDbMapping,
				aniDbToListIdMapping: tt.aniDbToListIdMapping,
			})

			got, err := svc.GetMapping(context.Background(), tt.tvDbID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d mappings, got %d", len(tt.want), len(got))
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("mapping[%d]: expected %+v, got %+v", i, w, got[i])
				}
			}
		})
	}
}
