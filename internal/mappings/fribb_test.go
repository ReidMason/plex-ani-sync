package mappings_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/mappings"
)

func serveFile(t *testing.T, path string) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchFribbParsesRowsRecordedFromTheRealDataset(t *testing.T) {
	t.Parallel()
	srv := serveFile(t, "testdata/fribb-sample.json")

	got, err := mappings.FetchFribb(context.Background(), srv.Client(), srv.URL)

	wantRows := []mappings.Row{
		{AniDBID: 1, AniListID: 290, TVDBID: 72025, Season: 1},
		{AniDBID: 7, AniListID: 164},
		{AniDBID: 11, AniListID: 821, TVDBID: 70900, EpisodeOffset: 2},
		{AniDBID: 19, AniListID: 197, TVDBID: 83692},
		{AniDBID: 102, AniListID: 232, TVDBID: 70668},
		{AniDBID: 391, AniListID: 690, TVDBID: 70862, Season: 1, EpisodeOffset: 1},
	}
	wantLinks := []mappings.IDLink{{TMDBID: 26209, TVDBID: 72025}, {TMDBID: 34684, TVDBID: 83692}, {TMDBID: 35790, TVDBID: 70668}, {TMDBID: 42460, TVDBID: 70862}}
	if err != nil || !reflect.DeepEqual(got.Rows, wantRows) || !reflect.DeepEqual(got.Links, wantLinks) {
		t.Errorf("got rows %+v links %+v err %v", got.Rows, got.Links, err)
	}
}
