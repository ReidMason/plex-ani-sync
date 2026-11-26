package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/anilisttest"
	"github.com/ReidMason/plex-ani-sync/internal/datasettest"
	"github.com/ReidMason/plex-ani-sync/internal/mappings"
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
	Reason      string            `json:"reason"`
	ExternalIDs map[string]string `json:"externalIds"`
	Mappings    []mappingJSON     `json:"mappings"`
	Uncovered   []struct {
		Season   int   `json:"season"`
		Episodes []int `json:"episodes"`
	} `json:"uncoveredEpisodes"`
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
	restarted, _ := startAppWith(t, p.dbURL, p.provider, p.tv, func(c *server.Config) { c.FribbURL, c.AnimeListsURL = changed.FribbURL, changed.AnimeListsURL })
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

func withAnimeLists(xml string) func(*datasettest.Host) {
	return func(h *datasettest.Host) { h.SetAnimeLists(xml) }
}

func TestTwoSeasonsFoldingIntoOneEntryResolveToTwoMappings(t *testing.T) {
	p := startPlexApp(t,
		withFribb(datasettest.FribbEntry{AniDBID: 10, AniListID: 1001, TVDBID: 77, Season: tvdb(1), Type: "TV"}),
		withAnimeLists(`<?xml version="1.0" encoding="utf-8"?>
<anime-list>
  <anime anidbid="10" tvdbid="77" defaulttvdbseason="1">
    <name>Merged</name>
    <mapping-list>
      <mapping anidbseason="0" tvdbseason="0">;1-1;</mapping>
      <mapping anidbseason="1" tvdbseason="1" start="1" end="12"/>
      <mapping anidbseason="1" tvdbseason="2" start="13" end="24" offset="-12"/>
    </mapping-list>
  </anime>
</anime-list>`))
	b, srv := animeSetup(t, p)
	for s := 1; s <= 2; s++ {
		for n := 1; n <= 12; n++ {
			srv.AddEpisode("2", plexservertest.Episode{Show: "Merged", Season: s, Number: n, Title: "x"})
		}
	}
	srv.SetShowGUIDs("2", "Merged", "tvdb://77")
	selectAnime(b, srv)

	got := b.mappedShows()["Merged"]

	assertMappings(t, got, "mapped",
		mappingJSON{Season: 1, FirstEpisode: 1, LastEpisode: ptr(12), AniListID: 1001, EntryFirst: 1, EntryLast: ptr(12)},
		mappingJSON{Season: 2, FirstEpisode: 1, LastEpisode: ptr(12), AniListID: 1001, EntryFirst: 13, EntryLast: ptr(24)},
	)
}

func TestFribbOnlyShowsAreUnaffectedByAnimeLists(t *testing.T) {
	p := startPlexApp(t,
		withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 5, Season: tvdb(1), Type: "TV"}),
		withAnimeLists(`<anime-list><anime anidbid="1" tvdbid="5" defaulttvdbseason="1"><name>Plain</name></anime></anime-list>`))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Plain", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Plain", "tvdb://5")
	selectAnime(b, srv)

	assertMappings(t, b.mappedShows()["Plain"], "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 10, EntryFirst: 1})
}

func TestShowWithEpisodesOutsideEveryMappingNeedsReview(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Gappy", Season: 1, Number: 1, Title: "x"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Gappy", Season: 2, Number: 1, Title: "x"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Gappy", Season: 2, Number: 2, Title: "x"})
	srv.SetShowGUIDs("2", "Gappy", "tvdb://77")
	selectAnime(b, srv)

	got := b.mappedShows()["Gappy"]

	if got.Status != "needs_review" || got.Reason == "" {
		t.Fatalf("status = %q reason = %q", got.Status, got.Reason)
	}
	if len(got.Uncovered) != 1 || got.Uncovered[0].Season != 2 || len(got.Uncovered[0].Episodes) != 2 {
		t.Errorf("uncovered = %+v", got.Uncovered)
	}
}

func TestGapBetweenMappedRangesNeedsReviewAndClearsWhenEpisodesChange(t *testing.T) {
	p := startPlexApp(t, withFribb(
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 77, Season: tvdb(1), Offset: tvdb(12), Type: "TV"},
	))
	b, srv := animeSetup(t, p)
	for _, n := range []int{1, 2, 12, 13} {
		srv.AddEpisode("2", plexservertest.Episode{Show: "Cour", Season: 1, Number: n, Title: "x"})
	}
	srv.SetShowGUIDs("2", "Cour", "tvdb://77")
	selectAnime(b, srv)
	if got := b.mappedShows()["Cour"]; got.Status != "mapped" {
		t.Fatalf("status = %q", got.Status)
	}

	srv.AddEpisode("2", plexservertest.Episode{Show: "Cour", Season: 3, Number: 1, Title: "x"})
	p.clock.Advance(time.Hour)

	if got := b.mappedShows()["Cour"]; got.Status != "needs_review" {
		t.Errorf("status after new season = %q", got.Status)
	}
}

func mismatchSetup(t *testing.T, episodes int) (plexApp, *browser) {
	t.Helper()
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	for n := 1; n <= episodes; n++ {
		srv.AddEpisode("2", plexservertest.Episode{Show: "Counted", Season: 1, Number: n, Title: "x"})
	}
	srv.SetShowGUIDs("2", "Counted", "tvdb://77")
	selectAnime(b, srv)
	return p, b
}

func TestEpisodeCountDisagreeingWithEntryTotalNeedsReview(t *testing.T) {
	p, b := mismatchSetup(t, 13)
	p.anilist.Set(anilisttest.Media{ID: 10, Episodes: ptr(12), Title: anilisttest.Title{Romaji: "Counted"}})

	got := b.mappedShows()["Counted"]

	if got.Status != "needs_review" || !strings.Contains(got.Reason, "13") || !strings.Contains(got.Reason, "12") {
		t.Errorf("status = %q reason = %q", got.Status, got.Reason)
	}
}

func TestEpisodeCountMatchingEntryTotalStaysMapped(t *testing.T) {
	p, b := mismatchSetup(t, 12)
	p.anilist.Set(anilisttest.Media{ID: 10, Episodes: ptr(12)})

	if got := b.mappedShows()["Counted"]; got.Status != "mapped" {
		t.Errorf("status = %q reason = %q", got.Status, got.Reason)
	}
}

func TestEntryWithUnknownTotalIsNotCompared(t *testing.T) {
	p, b := mismatchSetup(t, 13)
	p.anilist.Set(anilisttest.Media{ID: 10})

	if got := b.mappedShows()["Counted"]; got.Status != "mapped" {
		t.Errorf("status = %q", got.Status)
	}
}

func TestAniListMetadataIsCachedBetweenReads(t *testing.T) {
	p, b := mismatchSetup(t, 13)
	p.anilist.Set(anilisttest.Media{ID: 10, Episodes: ptr(12)})
	b.mappedShows()
	before := p.anilist.Requests()

	got := b.mappedShows()["Counted"]

	if p.anilist.Requests() != before || got.Status != "needs_review" {
		t.Errorf("requests %d -> %d, status = %q", before, p.anilist.Requests(), got.Status)
	}
}

func TestAniListOutageDegradesToMissingSignal(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests} {
		p, b := mismatchSetup(t, 13)
		p.anilist.Set(anilisttest.Media{ID: 10, Episodes: ptr(12)})
		p.anilist.FailWith(status)

		got := b.mappedShows()["Counted"]

		if got.Status != "mapped" {
			t.Errorf("status %d: got %q", status, got.Status)
		}
		p.anilist.FailWith(0)
		if got := b.mappedShows()["Counted"]; got.Status != "needs_review" {
			t.Errorf("status %d after recovery: got %q", status, got.Status)
		}
	}
}

func titleSetup(t *testing.T, plexTitle string, anilistTitle anilisttest.Title) *browser {
	t.Helper()
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: plexTitle, Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", plexTitle, "tvdb://77")
	selectAnime(b, srv)
	p.anilist.Set(anilisttest.Media{ID: 10, Title: anilistTitle})
	return b
}

func TestWildlyDifferentTitleNeedsReviewWithoutChangingMappings(t *testing.T) {
	b := titleSetup(t, "Cooking Show", anilisttest.Title{Romaji: "Sousou no Frieren", English: "Frieren: Beyond Journey's End"})

	got := b.mappedShows()["Cooking Show"]

	if got.Status != "needs_review" || !strings.Contains(got.Reason, "title") || len(got.Mappings) != 1 {
		t.Errorf("status = %q reason = %q mappings = %+v", got.Status, got.Reason, got.Mappings)
	}
}

func TestSimilarTitlesAreNotFlagged(t *testing.T) {
	cases := []struct {
		plex  string
		title anilisttest.Title
	}{
		{"Frieren", anilisttest.Title{Romaji: "Sousou no Frieren", English: "Frieren: Beyond Journey's End"}},
		{"Sousou no Frieren", anilisttest.Title{Romaji: "Sousou no Frieren"}},
		{"Attack on Titan", anilisttest.Title{Romaji: "Shingeki no Kyojin", English: "attack on titan"}},
		{"Re:Zero", anilisttest.Title{Romaji: "Re:Zero kara Hajimeru Isekai Seikatsu"}},
	}
	for _, c := range cases {
		b := titleSetup(t, c.plex, c.title)

		if got := b.mappedShows()[c.plex]; got.Status != "mapped" {
			t.Errorf("%q vs %+v: status = %q reason = %q", c.plex, c.title, got.Status, got.Reason)
		}
	}
}

func TestMissingAniListTitleNeverFlagsAShow(t *testing.T) {
	b := titleSetup(t, "Cooking Show", anilisttest.Title{})

	if got := b.mappedShows()["Cooking Show"]; got.Status != "mapped" {
		t.Errorf("status = %q reason = %q", got.Status, got.Reason)
	}
}

func TestUnmappedShowsExplainWhy(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "NoGuid", Season: 1, Number: 1, Title: "x"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "TMDBOnly", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "TMDBOnly", "tmdb://5")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Unknown", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Unknown", "tvdb://999")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Fine", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Fine", "tvdb://77")
	selectAnime(b, srv)

	shows := b.mappedShows()

	for title, want := range map[string]string{
		"NoGuid":   "Plex has no TVDB ID for this show",
		"TMDBOnly": "No TVDB ID found for TMDB ID 5",
		"Unknown":  "No mappings found for TVDB ID 999",
		"Fine":     "",
	} {
		if shows[title].Reason != want {
			t.Errorf("%s reason = %q, want %q", title, shows[title].Reason, want)
		}
	}
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

func startPlexAppWithCustom(t *testing.T, custom []mappings.Mapping, entries ...datasettest.FribbEntry) plexApp {
	t.Helper()
	p := startPlexApp(t)
	host := datasettest.New(t)
	host.SetFribb(entries...)
	app, _ := startAppWith(t, newDatabase(t), p.provider, p.tv, func(c *server.Config) {
		c.FribbURL = host.FribbURL
		c.CustomMappings = custom
	})
	p.app = app
	return p
}

func TestCustomMappingAppearsAlongsideDatasetRowsAndDisplacesOverlappingOnes(t *testing.T) {
	custom := []mappings.Mapping{{TVDBID: 77, Season: 1, Episodes: mappings.Range{First: 13, Last: ptr(24)}, AniListID: 3003, Entry: mappings.Range{First: 1, Last: ptr(12)}}}
	p := startPlexAppWithCustom(t, custom,
		datasettest.FribbEntry{AniDBID: 100, AniListID: 1001, TVDBID: 77, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 200, AniListID: 2002, TVDBID: 77, Season: tvdb(1), Offset: tvdb(12), Type: "TV"},
	)
	b, srv := animeSetup(t, p)
	for n := 1; n <= 24; n++ {
		srv.AddEpisode("2", plexservertest.Episode{Show: "Split", Season: 1, Number: n, Title: "x"})
	}
	srv.SetShowGUIDs("2", "Split", "tvdb://77")
	selectAnime(b, srv)

	got := b.mappedShows()["Split"]

	assertMappings(t, got, "mapped",
		mappingJSON{Season: 1, FirstEpisode: 1, LastEpisode: ptr(12), AniListID: 1001, EntryFirst: 1, EntryLast: ptr(12)},
		mappingJSON{Season: 1, FirstEpisode: 13, LastEpisode: ptr(24), AniListID: 3003, EntryFirst: 1, EntryLast: ptr(12)},
	)
}

func startRefreshingApp(t *testing.T, custom []mappings.Mapping, host *datasettest.Host) plexApp {
	t.Helper()
	p := startPlexApp(t)
	app, _ := startAppWith(t, newDatabase(t), p.provider, p.tv, func(c *server.Config) {
		c.FribbURL, c.AnimeListsURL = host.FribbURL, host.AnimeListsURL
		c.CustomMappings = custom
		c.DatasetRefreshInterval = 20 * time.Millisecond
	})
	p.app = app
	return p
}

func waitForRefreshes(t *testing.T, host *datasettest.Host, n int) {
	t.Helper()
	start := host.Requests()
	deadline := time.Now().Add(5 * time.Second)
	for host.Requests() < start+n {
		if time.Now().After(deadline) {
			t.Fatal("no dataset refresh happened")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForMapping(t *testing.T, b *browser, show string, anilistID int) mappedShow {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := b.mappedShows()[show]
		if len(got.Mappings) > 0 && got.Mappings[0].AniListID == anilistID {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never resolved to %d: %+v", show, anilistID, got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRefreshReplacesDatasetRowsAndKeepsCustomMappings(t *testing.T) {
	host := datasettest.New(t)
	host.SetFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"})
	custom := []mappings.Mapping{{TVDBID: 88, Season: 1, Episodes: mappings.Range{First: 1}, AniListID: 5005, Entry: mappings.Range{First: 1}}}
	p := startRefreshingApp(t, custom, host)
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Moving", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Moving", "tvdb://77")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Pinned", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Pinned", "tvdb://88")
	selectAnime(b, srv)
	waitForMapping(t, b, "Moving", 10)

	host.SetFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 99, TVDBID: 77, Season: tvdb(1), Type: "TV"})

	waitForMapping(t, b, "Moving", 99)
	assertMappings(t, b.mappedShows()["Pinned"], "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 5005, EntryFirst: 1})
}

func TestRefreshWithOverlappingCustomMappingKeepsCustomWinning(t *testing.T) {
	host := datasettest.New(t)
	custom := []mappings.Mapping{{TVDBID: 77, Season: 1, Episodes: mappings.Range{First: 1}, AniListID: 5005, Entry: mappings.Range{First: 1}}}
	p := startRefreshingApp(t, custom, host)
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Pinned", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Pinned", "tvdb://77")
	selectAnime(b, srv)

	host.SetFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"})
	waitForRefreshes(t, host, 4)

	assertMappings(t, b.mappedShows()["Pinned"], "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 5005, EntryFirst: 1})
}

func TestFailedRefreshLeavesPreviousRowsInPlace(t *testing.T) {
	for name, fail := range map[string]func(*datasettest.Host){
		"download":              (*datasettest.Host).Fail,
		"second dataset failed": (*datasettest.Host).FailAnimeLists,
	} {
		t.Run(name, func(t *testing.T) {
			host := datasettest.New(t)
			host.SetFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"})
			p := startRefreshingApp(t, nil, host)
			b, srv := animeSetup(t, p)
			srv.AddEpisode("2", plexservertest.Episode{Show: "Stable", Season: 1, Number: 1, Title: "x"})
			srv.SetShowGUIDs("2", "Stable", "tvdb://77")
			selectAnime(b, srv)
			waitForMapping(t, b, "Stable", 10)

			host.SetFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 99, TVDBID: 77, Season: tvdb(1), Type: "TV"})
			fail(host)
			waitForRefreshes(t, host, 4)

			assertMappings(t, b.mappedShows()["Stable"], "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 10, EntryFirst: 1})
		})
	}
}

func TestCustomMappingAddedBetweenRunsDisplacesStoredOverlappingDatasetRows(t *testing.T) {
	row := datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}
	p := startPlexApp(t, withFribb(row))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Pinned", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "Pinned", "tvdb://77")
	selectAnime(b, srv)
	host := datasettest.New(t)
	host.SetFribb(row)
	custom := []mappings.Mapping{{TVDBID: 77, Season: 1, Episodes: mappings.Range{First: 1}, AniListID: 5005, Entry: mappings.Range{First: 1}}}

	p.app, _ = startAppWith(t, p.dbURL, p.provider, p.tv, func(c *server.Config) {
		c.FribbURL, c.AnimeListsURL, c.CustomMappings = host.FribbURL, host.AnimeListsURL, custom
	})

	assertMappings(t, p.signedIn(t, "sub-alice", "Alice").mappedShows()["Pinned"], "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 5005, EntryFirst: 1})
}

func TestSpecialsEpisodesDoNotCountAsUncoveredByMappings(t *testing.T) {
	p := startPlexApp(t, withFribb(datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"}))
	b, srv := animeSetup(t, p)
	srv.AddEpisode("2", plexservertest.Episode{Show: "WithSpecials", Season: 0, Number: 1, Title: "x"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "WithSpecials", Season: 1, Number: 1, Title: "x"})
	srv.SetShowGUIDs("2", "WithSpecials", "tvdb://77")
	selectAnime(b, srv)

	got := b.mappedShows()["WithSpecials"]

	assertMappings(t, got, "mapped", mappingJSON{Season: 1, FirstEpisode: 1, AniListID: 10, EntryFirst: 1})
	if len(got.Uncovered) != 0 {
		t.Errorf("uncovered = %+v", got.Uncovered)
	}
}
