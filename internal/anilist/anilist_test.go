package anilist_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
)

func serve(t *testing.T, status int, fixture string) anilist.Client {
	t.Helper()
	body, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			http.Error(w, "no", status)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return anilist.Client{URL: srv.URL, HTTP: srv.Client()}
}

func TestFetchParsesTitlesAndEpisodeCounts(t *testing.T) {
	got, err := serve(t, http.StatusOK, "media.json").Fetch(context.Background(), []int{154587, 999001})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0].ID != 154587 || got[0].Episodes == nil || *got[0].Episodes != 28 ||
		got[0].TitleRomaji != "Sousou no Frieren" || got[0].TitleEnglish != "Frieren: Beyond Journey's End" {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].Episodes != nil || got[1].TitleEnglish != "" {
		t.Errorf("second = %+v", got[1])
	}
}

func TestFetchReportsRateLimitAndServerErrors(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError} {
		if _, err := serve(t, status, "media.json").Fetch(context.Background(), []int{1}); err == nil {
			t.Errorf("status %d: want error", status)
		}
	}
}
