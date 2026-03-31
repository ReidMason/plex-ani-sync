package plex

import (
	"testing"

	"myapp/internal/domain"
)

func TestMergeAnimeByTvDbID(t *testing.T) {
	got := mergeAnimeByTvDbID([]domain.MediaHostAnime{
		{
			ID:    "123",
			Title: "Naruto",
			Seasons: []domain.MediaHostSeason{
				{Number: 1, Episodes: []domain.MediaHostEpisode{{Number: 1, Watched: true}, {Number: 2, Watched: false}}},
			},
		},
		{
			ID:    "123",
			Title: "Naruto",
			Seasons: []domain.MediaHostSeason{
				{Number: 1, Episodes: []domain.MediaHostEpisode{{Number: 2, Watched: true}, {Number: 3, Watched: false}}},
			},
		},
	})
	if len(got) != 1 {
		t.Fatalf("len %d, want 1 merged show", len(got))
	}
	if got[0].ID != "123" || string(got[0].Title) != "Naruto" {
		t.Fatalf("got %+v", got[0])
	}
	if len(got[0].Seasons) != 1 || len(got[0].Seasons[0].Episodes) != 3 {
		t.Fatalf("seasons: %+v", got[0].Seasons)
	}
	eps := got[0].Seasons[0].Episodes
	if !eps[0].Watched || !eps[1].Watched || eps[2].Watched {
		t.Fatalf("watched flags: %+v", eps)
	}
}
