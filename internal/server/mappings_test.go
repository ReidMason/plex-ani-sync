package server_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/datasettest"
	"github.com/ReidMason/plex-ani-sync/internal/plexservertest"
	"github.com/ReidMason/plex-ani-sync/internal/server"
)

type mappingJSON struct {
	Season       int  `json:"season"`
	FirstEpisode int  `json:"firstEpisode"`
	LastEpisode  *int `json:"lastEpisode"`
	AniListID    int  `json:"anilistId"`
	EntryFirst   int  `json:"entryFirstEpisode"`
	EntryLast    *int `json:"entryLastEpisode"`
}

type mappedShow struct {
	Title       string            `json:"title"`
	Status      string            `json:"status"`
	ExternalIDs map[string]string `json:"externalIds"`
	Mappings    []mappingJSON     `json:"mappings"`
}

func (b *browser) mappedShows() map[string]mappedShow {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/anime")
	defer resp.Body.Close()
	var list []mappedShow
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		b.t.Fatal(err)
	}
	out := map[string]mappedShow{}
	for _, s := range list {
		out[s.Title] = s
	}
	return out
}

func withFribb(entries ...datasettest.FribbEntry) func(*datasettest.Host) {
	return func(h *datasettest.Host) { h.SetFribb(entries...) }
}

func tvdb(season int) *datasettest.Season { return &datasettest.Season{TVDB: season} }

func animeSetup(t *testing.T, p plexApp) (*browser, *plexservertest.Server) {
	t.Helper()
	b := p.signedIn(t, "sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	return b, srv
}

func selectAnime(b *browser, srv *plexservertest.Server) {
	src := b.mustAdd(srv.URL)
	b.selectAnimeLibraries(src.ID, []string{"2"})
}

func ptr(n int) *int { return &n }

func equalMapping(a, b mappingJSON) bool {
	same := func(x, y *int) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) }
	return a.Season == b.Season && a.FirstEpisode == b.FirstEpisode && a.AniListID == b.AniListID && a.EntryFirst == b.EntryFirst &&
		same(a.LastEpisode, b.LastEpisode) && same(a.EntryLast, b.EntryLast)
}

func assertMappings(t *testing.T, got mappedShow, status string, want ...mappingJSON) {
	t.Helper()
	if got.Status != status || len(got.Mappings) != len(want) {
		t.Fatalf("got %s %+v, want %s %+v", got.Status, got.Mappings, status, want)
	}
	for i := range want {
		if !equalMapping(got.Mappings[i], want[i]) {
			t.Errorf("mapping %d = %+v, want %+v", i, got.Mappings[i], want[i])
		}
	}
}

func TestShowWithTVDBIDIsMappedFromFribb(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 17617, AniListID: 154587, TVDBID: 424536, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Frieren", "tvdb://424536", "tmdb://209867", "imdb://tt22248376")
	selectAnime(b, srv)

	got := b.mappedShows()["Frieren"]

	assertMappings(t, got, "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 154587, EntryFirst: 1})
	if got.ExternalIDs["tvdb"] != "424536" || got.ExternalIDs["tmdb"] != "209867" || got.ExternalIDs["imdb"] != "tt22248376" {
		t.Errorf("external ids = %v", got.ExternalIDs)
	}
}

func TestSplitCourSeasonResolvesToTwoMappingsWithOffsetRanges(t *testing.T) {
	p := startPlexApp(t, withFribb(
		datasettest.FribbEntry{AniDBID: 200, AniListID: 2002, TVDBID: 77, Season: tvdb(1), Offset: tvdb(12), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 100, AniListID: 1001, TVDBID: 77, Season: tvdb(1), Type: "TV"},
	))
	b, srv := animeSetup(t, p)
	for n := 1; n <= 24; n++ {
		srv.AddEpisode("2", plexservertest.Episode{Show: "Split", Season: 1, Number: n, Title: "x"})
	}
	srv.SetShowGUIDs("2", "Split", "tvdb://77")
	selectAnime(b, srv)

	got := b.mappedShows()["Split"]

	assertMappings(t, got, "mapped",
		mappingJSON{Season: 1, FirstEpisode: 1, LastEpisode: ptr(12), AniListID: 1001, EntryFirst: 1, EntryLast: ptr(12)},
		mappingJSON{Season: 1, FirstEpisode: 13, AniListID: 2002, EntryFirst: 1},
	)
}

func TestShowsWithoutAUsableTVDBIDAreUnmapped(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "NoGuid", Season: 1, Number: 1, Title: "x"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "TMDBOnly", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "TMDBOnly", "tmdb://5")
	srv.AddEpisode("2", plexservertest.Episode{Show: "UnknownToDataset", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "UnknownToDataset", "tvdb://999")
	selectAnime(b, srv)

	shows := b.mappedShows()

	for _, title := range []string{"NoGuid", "TMDBOnly", "UnknownToDataset"} {
		assertMappings(t, shows[title], "unmapped")
	}
}

func TestOverlappingDatasetRowsResolveByLowestAniDBID(t *testing.T) {
	rows := []datasettest.FribbEntry{
		{AniDBID: 30, AniListID: 3003, TVDBID: 77, Season: tvdb(1), Type: "TV"},
		{AniDBID: 10, AniListID: 1001, TVDBID: 77, Season: tvdb(1), Type: "TV"},
		{AniDBID: 20, AniListID: 2002, TVDBID: 77, Season: tvdb(1), Type: "TV"},
	}
	p := startPlexApp(t, withFribb(rows...))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Overlap", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Overlap", "tvdb://77")
	selectAnime(b, srv)

	got := b.mappedShows()["Overlap"]

	assertMappings(t, got, "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 1001, EntryFirst: 1})
}

func TestSpecialsAreNotMapped(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(0), Type: "OVA"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Specials", Season: 0, Number: 1, Title: "x"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Specials", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Specials", "tvdb://77")
	selectAnime(b, srv)

	assertMappings(t, b.mappedShows()["Specials"], "unmapped")
}

func TestDatasetIsLoadedOnlyWhenMappingsAreEmpty(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}))
	changed := datasettest.New(t)
	changed.SetFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 99, TVDBID: 77, Season: tvdb(1), Type: "TV"})
	restarted, _ := startAppWith(t, p.dbURL, p.provider, p.tv, func(c *server.Config) { c.FribbURL = changed.FribbURL })
	p.app = restarted
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Stable", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Stable", "tvdb://77")
	selectAnime(b, srv)

	assertMappings(t, b.mappedShows()["Stable"], "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 10, EntryFirst: 1})
}

func TestUnavailableDatasetLeavesShowsUnmappedWithoutBreakingStartup(t *testing.T) {
	p := startPlexApp(t, func(h *datasettest.Host) { h.Fail() })
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Down", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Down", "tvdb://77")
	selectAnime(b, srv)

	assertMappings(t, b.mappedShows()["Down"], "unmapped")
}

func TestTMDBOnlyShowResolvesThroughDatasetIDLinks(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, TMDBID: 5, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Linked", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Linked", "tmdb://5")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Absent", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Absent", "tmdb://404")
	selectAnime(b, srv)

	shows := b.mappedShows()

	assertMappings(t, shows["Linked"], "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 10, EntryFirst: 1})
	assertMappings(t, shows["Absent"], "unmapped")
}
