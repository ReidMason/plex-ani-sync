package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/anilisttest"
	"github.com/ReidMason/plex-ani-sync/internal/datasettest"
	"github.com/ReidMason/plex-ani-sync/internal/plexservertest"
)

type planState struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

type planChange struct {
	AniListID int        `json:"anilistId"`
	Title     string     `json:"title"`
	Current   *planState `json:"current"`
	Proposed  planState  `json:"proposed"`
	Reason    string     `json:"reason"`
}

type planSkip struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type syncPlan struct {
	Changes []planChange `json:"changes"`
	Skipped []planSkip   `json:"skipped"`
	Error   string       `json:"error"`
}

func (b *browser) syncNow() (int, syncPlan) {
	b.t.Helper()
	b.app.clock.Advance(time.Minute)
	resp := b.do(http.MethodPost, b.app.URL+"/api/sync")
	defer resp.Body.Close()
	var out syncPlan
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (p syncPlan) change(anilistID int) *planChange {
	for i := range p.Changes {
		if p.Changes[i].AniListID == anilistID {
			return &p.Changes[i]
		}
	}
	return nil
}

var testStart = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func watch(srv *plexservertest.Server, show string, season int, watched ...bool) {
	watchedDaysAgo(srv, show, season, 0, watched...)
}

func watchedDaysAgo(srv *plexservertest.Server, show string, season int, days int, watched ...bool) {
	for i, w := range watched {
		e := plexservertest.Episode{Show: show, Season: season, Number: i + 1, Title: "x"}
		if w {
			e.ViewCount = 1
			e.LastViewedAt = testStart.AddDate(0, 0, -days).Unix()
		}
		srv.AddEpisode("2", e)
	}
}

func syncSetup(t *testing.T, entries ...datasettest.FribbEntry) (*app, *browser, *plexservertest.Server) {
	t.Helper()
	p := startApp(t, withFribb(entries...))
	b, srv := animeSetup(t, p)
	return p, b, srv
}

func linkedToList(p *app, b *browser, list ...anilisttest.ListEntry) {
	p.anilist.SetUserList("alice_anime", list...)
	b.linkAniList("alice_anime")
}

func TestShowWatchedInPlexButNotOnTheListIsProposedAsAnAddition(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t, datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 77, Season: tvdb(1), Type: "TV"})
	watch(srv, "Frieren", 1, true, true, false)
	srv.SetShowGUIDs("2", "Frieren", "tvdb://77")
	selectAnime(b, srv)
	linkedToList(p, b)

	status, plan := b.syncNow()

	got := plan.change(10)
	if status != http.StatusOK || got == nil || got.Current != nil || got.Proposed != (planState{"CURRENT", 2}) || got.Title != "Frieren" || got.Reason == "" {
		t.Fatalf("status %d plan %+v", status, plan)
	}
}

func TestProgressCountsOnlyWatchedEpisodesInsideEachEntrysMappingsAndNeedNotBeConsecutive(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 100, AniListID: 1001, TVDBID: 77, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 200, AniListID: 2002, TVDBID: 77, Season: tvdb(1), Offset: tvdb(3), Type: "TV"},
	)
	watch(srv, "Split", 1, true, false, true, false, true, true)
	srv.AddEpisode("2", plexservertest.Episode{Show: "Split", Season: 0, Number: 1, Title: "x", ViewCount: 1})
	srv.SetShowGUIDs("2", "Split", "tvdb://77")
	selectAnime(b, srv)
	linkedToList(p, b)

	_, plan := b.syncNow()

	first, second := plan.change(1001), plan.change(2002)
	if first == nil || first.Proposed.Progress != 2 || second == nil || second.Proposed.Progress != 2 {
		t.Fatalf("plan %+v", plan)
	}
}

func TestProgressNeverDecreasesAndCompletedAndRepeatingEntriesAreUntouched(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 3, AniListID: 30, TVDBID: 3, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 4, AniListID: 40, TVDBID: 4, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 5, AniListID: 50, TVDBID: 5, Season: tvdb(1), Type: "TV"},
	)
	for i, title := range []string{"Behind", "Ahead", "Equal", "Done", "Again"} {
		watch(srv, title, 1, true, true, true)
		srv.SetShowGUIDs("2", title, "tvdb://"+string(rune('1'+i)))
	}
	selectAnime(b, srv)
	linkedToList(p, b,
		anilisttest.ListEntry{MediaID: 10, Status: "PLANNING", Progress: 1},
		anilisttest.ListEntry{MediaID: 20, Status: "CURRENT", Progress: 8},
		anilisttest.ListEntry{MediaID: 30, Status: "CURRENT", Progress: 3},
		anilisttest.ListEntry{MediaID: 40, Status: "COMPLETED", Progress: 1},
		anilisttest.ListEntry{MediaID: 50, Status: "REPEATING", Progress: 1},
	)

	_, plan := b.syncNow()

	got := plan.change(10)
	if len(plan.Changes) != 1 || got == nil || got.Current == nil || *got.Current != (planState{"PLANNING", 1}) || got.Proposed.Progress != 3 || got.Reason == "" {
		t.Fatalf("plan %+v", plan.Changes)
	}
}

func TestEntriesWithNoWatchedEpisodesGetNoProposalAndUntouchedListEntriesAreIgnored(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
	)
	watch(srv, "Unwatched", 1, false, false)
	srv.SetShowGUIDs("2", "Unwatched", "tvdb://1")
	watch(srv, "UnwatchedListed", 1, false)
	srv.SetShowGUIDs("2", "UnwatchedListed", "tvdb://2")
	selectAnime(b, srv)
	linkedToList(p, b,
		anilisttest.ListEntry{MediaID: 20, Status: "PLANNING"},
		anilisttest.ListEntry{MediaID: 999, Status: "CURRENT", Progress: 4},
	)

	status, plan := b.syncNow()

	if status != http.StatusOK || len(plan.Changes) != 0 {
		t.Fatalf("status %d plan %+v", status, plan)
	}
}

func TestUnmappedAndNeedsReviewShowsAreSkippedWithReasons(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t, datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"})
	watch(srv, "Unmapped", 1, true)
	watch(srv, "Partial", 1, true, true)
	srv.SetShowGUIDs("2", "Partial", "tvdb://1")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Partial", Season: 2, Number: 1, Title: "x", ViewCount: 1})
	selectAnime(b, srv)
	linkedToList(p, b)

	_, plan := b.syncNow()

	want := map[string]string{"Unmapped": "unmapped", "Partial": "needs_review"}
	if len(plan.Skipped) != 2 || len(plan.Changes) != 0 {
		t.Fatalf("plan %+v", plan)
	}
	for _, s := range plan.Skipped {
		if want[s.Title] != s.Status || s.Reason == "" {
			t.Errorf("skipped %+v", s)
		}
	}
}

func TestSyncNeedsALinkedAniListAndNeverUsesAnotherUsersList(t *testing.T) {
	t.Parallel()
	p, alice, _ := syncSetup(t)
	linkedToList(p, alice)
	bob := p.signedIn("sub-bob", "Bob")

	status, plan := bob.syncNow()
	unauthenticated, _ := p.browser().syncNow()

	if status != http.StatusConflict || plan.Error != "anilist_not_linked" || unauthenticated != http.StatusUnauthorized {
		t.Errorf("status %d %+v, anonymous %d", status, plan, unauthenticated)
	}
}

func TestSyncReportsAniListThrottle(t *testing.T) {
	t.Parallel()
	p, b, _ := syncSetup(t)
	linkedToList(p, b)
	p.anilist.FailWithRetryAfter(http.StatusTooManyRequests, 90)
	b.syncNow()

	status, plan := b.syncNow()

	if status != http.StatusTooManyRequests || plan.Error != "throttled" {
		t.Errorf("status %d %+v", status, plan)
	}
}

func TestOnlyOneSyncRunsPerUserAtATime(t *testing.T) {
	t.Parallel()
	p, alice, _ := syncSetup(t)
	linkedToList(p, alice)
	arrived, release := p.anilist.HoldUserLists()
	first := make(chan int)
	go func() {
		status, _ := alice.syncNow()
		first <- status
	}()
	<-arrived

	status, plan := alice.syncNow()
	release()

	if status != http.StatusConflict || plan.Error != "sync_in_progress" || <-first != http.StatusOK {
		t.Errorf("second sync: %d %+v", status, plan)
	}
	if after, _ := alice.syncNow(); after != http.StatusOK {
		t.Errorf("sync after finishing: %d", after)
	}
}

func TestSyncNeverWritesToAniList(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t, datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"})
	watch(srv, "Show", 1, true)
	srv.SetShowGUIDs("2", "Show", "tvdb://1")
	selectAnime(b, srv)
	linkedToList(p, b)

	_, plan := b.syncNow()

	if len(plan.Changes) != 1 || p.anilist.Writes() != 0 {
		t.Errorf("changes %d, writes %d", len(plan.Changes), p.anilist.Writes())
	}
}

func TestKnownTotalMatchingProgressProposesCompletedAndUnknownTotalStaysWatching(t *testing.T) {
	t.Parallel()
	three := 3
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 3, AniListID: 30, TVDBID: 3, Season: tvdb(1), Type: "TV"},
	)
	for i, title := range []string{"Finished", "Ongoing", "Partway"} {
		watch(srv, title, 1, true, true, true)
		srv.SetShowGUIDs("2", title, "tvdb://"+string(rune('1'+i)))
	}
	selectAnime(b, srv)
	linkedToList(p, b,
		anilisttest.ListEntry{MediaID: 10, Status: "CURRENT", Progress: 1, Episodes: &three},
		anilisttest.ListEntry{MediaID: 20, Status: "CURRENT", Progress: 1},
		anilisttest.ListEntry{MediaID: 30, Status: "CURRENT", Progress: 1, Episodes: ptr(12)},
	)

	_, plan := b.syncNow()

	if got := plan.change(10); got == nil || got.Proposed != (planState{"COMPLETED", 3}) {
		t.Errorf("finished: %+v", got)
	}
	if got := plan.change(20); got == nil || got.Proposed != (planState{"CURRENT", 3}) {
		t.Errorf("ongoing: %+v", got)
	}
	if got := plan.change(30); got == nil || got.Proposed != (planState{"CURRENT", 3}) {
		t.Errorf("partway: %+v", got)
	}
}

