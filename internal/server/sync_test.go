package server_test

import (
	"encoding/json"
	"net/http"
	"slices"
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
	Seasons   []int      `json:"seasons"`
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
	Until   time.Time    `json:"until"`
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

	if status != http.StatusTooManyRequests || plan.Error != "throttled" || !plan.Until.After(p.clock.Now()) {
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

func TestUnfinishedEntriesStalledFor14DaysArePausedAndFor30DaysDropped(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 3, AniListID: 30, TVDBID: 3, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 4, AniListID: 40, TVDBID: 4, Season: tvdb(1), Type: "TV"},
	)
	for i, c := range []struct {
		title string
		days  int
	}{{"Recent", 13}, {"Stalled", 15}, {"Abandoned", 31}, {"Finished", 60}} {
		watchedDaysAgo(srv, c.title, 1, c.days, true, true)
		srv.SetShowGUIDs("2", c.title, "tvdb://"+string(rune('1'+i)))
	}
	selectAnime(b, srv)
	linkedToList(p, b,
		anilisttest.ListEntry{MediaID: 10, Status: "CURRENT", Progress: 0, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 20, Status: "CURRENT", Progress: 0, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 30, Status: "CURRENT", Progress: 0, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 40, Status: "CURRENT", Progress: 0, Episodes: ptr(2)},
	)

	_, plan := b.syncNow()

	for id, want := range map[int]string{10: "CURRENT", 20: "PAUSED", 30: "DROPPED", 40: "COMPLETED"} {
		if got := plan.change(id); got == nil || got.Proposed.Status != want {
			t.Errorf("entry %d: %+v, want %s", id, got, want)
		}
	}
}

func TestDroppedEntryIsNeverProposedAsPausedAndPlexActivityRevivesPlanningPausedAndDropped(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 3, AniListID: 30, TVDBID: 3, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 4, AniListID: 40, TVDBID: 4, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 5, AniListID: 50, TVDBID: 5, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 6, AniListID: 60, TVDBID: 6, Season: tvdb(1), Type: "TV"},
	)
	for i, c := range []struct {
		title string
		days  int
	}{{"DroppedStalled", 20}, {"DroppedRecent", 1}, {"PausedRecent", 1}, {"PlanningRecent", 1}, {"PausedLong", 40}, {"DroppedFinished", 1}} {
		watchedDaysAgo(srv, c.title, 1, c.days, true, true)
		srv.SetShowGUIDs("2", c.title, "tvdb://"+string(rune('1'+i)))
	}
	selectAnime(b, srv)
	linkedToList(p, b,
		anilisttest.ListEntry{MediaID: 10, Status: "DROPPED", Progress: 1, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 20, Status: "DROPPED", Progress: 1, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 30, Status: "PAUSED", Progress: 1, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 40, Status: "PLANNING", Progress: 0, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 50, Status: "PAUSED", Progress: 1, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 60, Status: "DROPPED", Progress: 1, Episodes: ptr(2)},
	)

	_, plan := b.syncNow()

	for id, want := range map[int]string{10: "DROPPED", 20: "CURRENT", 30: "CURRENT", 40: "CURRENT", 50: "DROPPED", 60: "COMPLETED"} {
		if got := plan.change(id); got == nil || got.Proposed.Status != want {
			t.Errorf("entry %d: %+v, want %s", id, got, want)
		}
	}
}

func TestStatusIsEvaluatedWithTheHigherProgressWhenAniListIsAhead(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 3, AniListID: 30, TVDBID: 3, Season: tvdb(1), Type: "TV"},
	)
	for i, c := range []struct {
		title string
		days  int
	}{{"AheadFinished", 1}, {"AheadStalled", 40}, {"AheadFine", 1}} {
		watchedDaysAgo(srv, c.title, 1, c.days, true, true)
		srv.SetShowGUIDs("2", c.title, "tvdb://"+string(rune('1'+i)))
	}
	selectAnime(b, srv)
	linkedToList(p, b,
		anilisttest.ListEntry{MediaID: 10, Status: "CURRENT", Progress: 8, Episodes: ptr(8)},
		anilisttest.ListEntry{MediaID: 20, Status: "CURRENT", Progress: 8, Episodes: ptr(12)},
		anilisttest.ListEntry{MediaID: 30, Status: "CURRENT", Progress: 8, Episodes: ptr(12)},
	)

	_, plan := b.syncNow()

	if got := plan.change(10); got == nil || got.Proposed != (planState{"COMPLETED", 8}) || got.Reason == "" {
		t.Errorf("finished: %+v", got)
	}
	if got := plan.change(20); got == nil || got.Proposed != (planState{"DROPPED", 8}) || got.Reason == "" {
		t.Errorf("stalled: %+v", got)
	}
	if got := plan.change(30); got != nil || len(plan.Changes) != 2 {
		t.Errorf("fine: %+v", plan.Changes)
	}
}

func TestSecondSyncDoesNotRefetchTheAniListList(t *testing.T) {
	t.Parallel()
	p, b, _ := syncSetup(t)
	linkedToList(p, b, anilisttest.ListEntry{MediaID: 10, Status: "CURRENT", Progress: 1})
	b.syncNow()
	before := p.anilist.Requests()

	status, _ := b.syncNow()

	if status != http.StatusOK || p.anilist.Requests() != before {
		t.Errorf("status %d, anilist requests %d -> %d", status, before, p.anilist.Requests())
	}
}

func TestSyncRefetchesTheAniListListOnceTheCacheExpires(t *testing.T) {
	t.Parallel()
	p, b, _ := syncSetup(t)
	linkedToList(p, b)
	b.syncNow()
	before := p.anilist.Requests()
	p.clock.Advance(10 * time.Minute)

	status, _ := b.syncNow()

	if status != http.StatusOK || p.anilist.Requests() != before+1 {
		t.Errorf("status %d, anilist requests %d -> %d", status, before, p.anilist.Requests())
	}
}

func TestSecondSyncDoesNotRefetchPlexShows(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t, datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"})
	watch(srv, "Show", 1, true)
	srv.SetShowGUIDs("2", "Show", "tvdb://1")
	selectAnime(b, srv)
	linkedToList(p, b)
	b.syncNow()
	before := srv.Requests()

	_, plan := b.syncNow()

	if srv.Requests() != before+1 || len(plan.Changes) != 1 {
		t.Errorf("plex requests %d -> %d, plan %+v", before, srv.Requests(), plan)
	}
}

func TestAdditionIsProposedAsCompletedWhenAniListMediaTotalMatchesProgress(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
	)
	p.anilist.Set(anilisttest.Media{ID: 10, Episodes: ptr(3)})
	p.anilist.Set(anilisttest.Media{ID: 20})
	for i, title := range []string{"Finished", "Ongoing"} {
		watch(srv, title, 1, true, true, true)
		srv.SetShowGUIDs("2", title, "tvdb://"+string(rune('1'+i)))
	}
	selectAnime(b, srv)
	p.settledShows(t, b)
	p.clock.Advance(time.Minute)
	linkedToList(p, b)

	_, plan := b.syncNow()

	if got := plan.change(10); got == nil || got.Proposed != (planState{"COMPLETED", 3}) {
		t.Errorf("finished: %+v", got)
	}
	if got := plan.change(20); got == nil || got.Proposed != (planState{"CURRENT", 3}) {
		t.Errorf("ongoing: %+v", got)
	}
}

func TestAdditionsFollowTheIdleRulesPausedAfter14DaysAndDroppedAfter30(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 1, AniListID: 10, TVDBID: 1, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 2, AniListID: 20, TVDBID: 2, Season: tvdb(1), Type: "TV"},
	)
	for i, c := range []struct {
		title string
		days  int
	}{{"Stalled", 20}, {"Abandoned", 60}} {
		watchedDaysAgo(srv, c.title, 1, c.days, true, true)
		srv.SetShowGUIDs("2", c.title, "tvdb://"+string(rune('1'+i)))
	}
	selectAnime(b, srv)
	linkedToList(p, b)

	_, plan := b.syncNow()

	if got := plan.change(10); got == nil || got.Proposed != (planState{"PAUSED", 2}) {
		t.Errorf("stalled: %+v", got)
	}
	if got := plan.change(20); got == nil || got.Proposed != (planState{"DROPPED", 2}) {
		t.Errorf("abandoned: %+v", got)
	}
}

func TestChangesNameThePlexSeasonsEachEntryCovers(t *testing.T) {
	t.Parallel()
	p, b, srv := syncSetup(t,
		datasettest.FribbEntry{AniDBID: 100, AniListID: 1001, TVDBID: 77, Season: tvdb(1), Type: "TV"},
		datasettest.FribbEntry{AniDBID: 200, AniListID: 2002, TVDBID: 77, Season: tvdb(2), Type: "TV"},
	)
	watch(srv, "Multi", 1, true, true)
	watch(srv, "Multi", 2, true)
	srv.SetShowGUIDs("2", "Multi", "tvdb://77")
	selectAnime(b, srv)
	linkedToList(p, b)

	_, plan := b.syncNow()

	first, second := plan.change(1001), plan.change(2002)
	if first == nil || !slices.Equal(first.Seasons, []int{1}) || second == nil || !slices.Equal(second.Seasons, []int{2}) {
		t.Fatalf("plan %+v", plan)
	}
}
