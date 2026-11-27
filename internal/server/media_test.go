package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/plexservertest"
)

type mediaSource struct {
	ID   int64  `json:"id"`
	URL  string `json:"url"`
	Kind string `json:"kind"`
}

type mediaError struct {
	Error string `json:"error"`
}

func (b *browser) addMediaSource(serverURL string) (int, mediaSource, mediaError) {
	b.t.Helper()
	payload, _ := json.Marshal(map[string]string{"url": serverURL})
	req, _ := http.NewRequest(http.MethodPost, b.app.URL+"/api/media-sources", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range b.cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	var src mediaSource
	var e mediaError
	raw := new(bytes.Buffer)
	_, _ = raw.ReadFrom(resp.Body)
	_ = json.Unmarshal(raw.Bytes(), &src)
	_ = json.Unmarshal(raw.Bytes(), &e)
	return resp.StatusCode, src, e
}

func (b *browser) mediaSources() []mediaSource {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/media-sources")
	defer resp.Body.Close()
	var out []mediaSource
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out
}

func (b *browser) mustAdd(serverURL string) mediaSource {
	b.t.Helper()
	status, src, e := b.addMediaSource(serverURL)
	if status != http.StatusCreated {
		b.t.Fatalf("add %s: %d %+v", serverURL, status, e)
	}
	return src
}

func TestAddingAReachablePlexServerSavesAMediaSource(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")

	status, src, _ := b.addMediaSource(srv.URL + "/")

	if status != http.StatusCreated || src.URL != srv.URL || src.ID == 0 || src.Kind != "plex" {
		t.Fatalf("status %d src %+v", status, src)
	}
	if got := b.mediaSources(); len(got) != 1 || got[0] != src {
		t.Errorf("list = %+v", got)
	}
}

func TestMediaSourcesStartEmpty(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")

	if got := b.mediaSources(); got == nil || len(got) != 0 {
		t.Errorf("list = %#v, want empty array", got)
	}
}

func TestAccessCheckFailuresReturnDistinctErrorsAndSaveNothing(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	noAccess := plexservertest.New(t)
	forbidden := plexservertest.New(t)
	forbidden.Forbid("tok-a")

	cases := []struct {
		name   string
		url    string
		status int
		code   string
	}{
		{"unreachable", plexservertest.NewUnreachable(t), http.StatusBadGateway, "unreachable"},
		{"not plex", plexservertest.NewNotPlex(t), http.StatusUnprocessableEntity, "not_plex_server"},
		{"401", noAccess.URL, http.StatusForbidden, "no_access"},
		{"403", forbidden.URL, http.StatusForbidden, "no_access"},
		{"bad scheme", "ftp://example.com", http.StatusBadRequest, "invalid_url"},
		{"empty", "", http.StatusBadRequest, "invalid_url"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, _, e := b.addMediaSource(c.url)
			if status != c.status || e.Error != c.code {
				t.Errorf("got %d %q, want %d %q", status, e.Error, c.status, c.code)
			}
		})
	}
	if got := b.mediaSources(); len(got) != 0 {
		t.Errorf("media sources = %+v, want none", got)
	}
}

func TestAddingWithoutALinkedPlexAccountIsRejectedWithoutContactingTheServer(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	srv := plexservertest.New(t)

	status, _, e := b.addMediaSource(srv.URL)

	if status != http.StatusConflict || e.Error != "plex_not_linked" {
		t.Errorf("got %d %q", status, e.Error)
	}
	if srv.Requests() != 0 {
		t.Error("server was contacted")
	}
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	target := plexservertest.New(t)
	target.Allow("tok-a")

	status, _, e := b.addMediaSource(plexservertest.NewRedirecting(t, target.URL))

	if status != http.StatusUnprocessableEntity || e.Error != "not_plex_server" {
		t.Errorf("got %d %q", status, e.Error)
	}
	if target.Requests() != 0 {
		t.Error("redirect was followed")
	}
}

func TestAUserCanAddSeveralServersButNotTheSameOneTwice(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	one, two := plexservertest.New(t), plexservertest.New(t)
	one.Allow("tok-a")
	two.Allow("tok-a")

	b.mustAdd(one.URL)
	b.mustAdd(two.URL)
	status, _, e := b.addMediaSource(one.URL)

	if status != http.StatusConflict || e.Error != "already_added" {
		t.Errorf("duplicate: %d %q", status, e.Error)
	}
	if got := b.mediaSources(); len(got) != 2 {
		t.Errorf("list = %+v", got)
	}
}

func TestMediaSourcesAreIsolatedBetweenUsers(t *testing.T) {
	p := startApp(t)
	alice := p.signedIn("sub-alice", "Alice")
	bob := p.signedIn("sub-bob", "Bob")
	alice.link(p, "tok-a", "alice_plex")
	bob.link(p, "tok-b", "bob_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.Allow("tok-b")

	alice.mustAdd(srv.URL)

	if got := bob.mediaSources(); len(got) != 0 {
		t.Errorf("bob sees %+v", got)
	}
	bob.mustAdd(srv.URL)
	if got := alice.mediaSources(); len(got) != 1 {
		t.Errorf("alice sees %+v", got)
	}
}

func TestMediaSourceUsesThePlexAccountTokenAndKeepsNoneOfItsOwn(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	src := b.mustAdd(srv.URL)

	b.link(p, "tok-new", "alice_plex")
	srv.Allow("tok-new")
	srv.Forbid("tok-a")

	if got := b.mediaSources(); len(got) != 1 {
		t.Errorf("relinking dropped media sources: %+v", got)
	}
	if status, _ := b.libraries(src.ID); status != http.StatusOK {
		t.Errorf("libraries after relink = %d, want 200 using the new account token", status)
	}
}

func TestMediaSourceEndpointsRequireSession(t *testing.T) {
	p := startApp(t)
	b := p.browser()

	if status, _, _ := b.addMediaSource("http://x"); status != http.StatusUnauthorized {
		t.Errorf("add status %d", status)
	}
	if resp := b.do(http.MethodGet, p.URL+"/api/media-sources"); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("list status %d", resp.StatusCode)
	}
}

func (b *browser) removeMediaSource(id int64) int {
	b.t.Helper()
	resp := b.do(http.MethodDelete, b.app.URL+"/api/media-sources/"+strconv.FormatInt(id, 10))
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestRemovingAMediaSourceKeepsThePlexAccountAndOtherSources(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	one, two := plexservertest.New(t), plexservertest.New(t)
	one.Allow("tok-a")
	two.Allow("tok-a")
	removed := b.mustAdd(one.URL)
	kept := b.mustAdd(two.URL)

	if status := b.removeMediaSource(removed.ID); status != http.StatusNoContent {
		t.Fatalf("status %d", status)
	}

	if got := b.mediaSources(); len(got) != 1 || got[0] != kept {
		t.Errorf("list = %+v", got)
	}
	if got := b.plexUsername(); got != "alice_plex" {
		t.Errorf("plex account lost: %q", got)
	}
}

func TestRemovingAnotherUsersMediaSourceDoesNothing(t *testing.T) {
	p := startApp(t)
	alice := p.signedIn("sub-alice", "Alice")
	bob := p.signedIn("sub-bob", "Bob")
	alice.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	src := alice.mustAdd(srv.URL)

	if status := bob.removeMediaSource(src.ID); status != http.StatusNotFound {
		t.Errorf("status %d, want 404", status)
	}
	if got := alice.mediaSources(); len(got) != 1 {
		t.Errorf("alice list = %+v", got)
	}
}

func TestRemovingRequiresSession(t *testing.T) {
	p := startApp(t)
	b := p.browser()

	if status := b.removeMediaSource(1); status != http.StatusUnauthorized {
		t.Errorf("status %d", status)
	}
}

type library struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Selected bool   `json:"selected"`
}

func (b *browser) libraries(id int64) (int, []library) {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/media-sources/"+strconv.FormatInt(id, 10)+"/libraries")
	defer resp.Body.Close()
	var out []library
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestLibrariesListsOnlyShowLibrariesUnselectedByDefault(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("1", "movie", "Movies")
	srv.AddLibrary("2", "show", "Anime")
	srv.AddLibrary("3", "show", "TV Shows")
	srv.AddLibrary("4", "artist", "Music")
	src := b.mustAdd(srv.URL)

	status, got := b.libraries(src.ID)

	want := []library{{Key: "2", Title: "Anime"}, {Key: "3", Title: "TV Shows"}}
	if status != http.StatusOK || !slices.Equal(got, want) {
		t.Errorf("got %d %+v, want %+v", status, got, want)
	}
}

func (b *browser) selectAnimeLibraries(id int64, keys []string) int {
	b.t.Helper()
	payload, _ := json.Marshal(map[string][]string{"keys": keys})
	req, _ := http.NewRequest(http.MethodPut, b.app.URL+"/api/media-sources/"+strconv.FormatInt(id, 10)+"/anime-libraries", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range b.cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestSelectedAnimeLibrariesAreRememberedPerMediaSource(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	srv.AddLibrary("3", "show", "TV Shows")
	other := plexservertest.New(t)
	other.Allow("tok-a")
	other.AddLibrary("2", "show", "Anime")
	src, otherSrc := b.mustAdd(srv.URL), b.mustAdd(other.URL)

	if status := b.selectAnimeLibraries(src.ID, []string{"2"}); status != http.StatusNoContent {
		t.Fatalf("select status %d", status)
	}

	_, got := b.libraries(src.ID)
	want := []library{{Key: "2", Title: "Anime", Selected: true}, {Key: "3", Title: "TV Shows"}}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if _, got := b.libraries(otherSrc.ID); got[0].Selected {
		t.Errorf("selection leaked to another media source: %+v", got)
	}
}

func TestSelectingAnUnknownLibraryIsRejectedAndSavesNothing(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("1", "movie", "Movies")
	srv.AddLibrary("2", "show", "Anime")
	src := b.mustAdd(srv.URL)

	for _, keys := range [][]string{{"9"}, {"1"}, {"2", "9"}} {
		if status := b.selectAnimeLibraries(src.ID, keys); status != http.StatusUnprocessableEntity {
			t.Errorf("keys %v: status %d, want 422", keys, status)
		}
	}
	if _, got := b.libraries(src.ID); got[0].Selected {
		t.Errorf("selection saved despite rejection: %+v", got)
	}
}

type animeShow struct {
	Title   string `json:"title"`
	Seasons []struct {
		Number   int `json:"number"`
		Episodes []struct {
			Number        int     `json:"number"`
			Title         string  `json:"title"`
			Watched       bool    `json:"watched"`
			LastWatchedAt *string `json:"lastWatchedAt"`
		} `json:"episodes"`
	} `json:"seasons"`
}

func (b *browser) anime() (int, []animeShow) {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/anime")
	defer resp.Body.Close()
	var out []animeShow
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestAnimePullsShowsSeasonsAndWatchedEpisodesFromSelectedLibrariesOnly(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	srv.AddLibrary("3", "show", "TV Shows")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 0, Number: 1, Title: "Special", ViewCount: 1})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 1, Title: "The Journey's End", ViewCount: 2, LastViewedAt: 1700000000})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 2, Title: "It Didn't Have to Be Magic"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 2, Number: 1, Title: "Season Two Opener"})
	srv.AddEpisode("2", plexservertest.Episode{Show: "Bleach", Season: 1, Number: 1, Title: "Ichigo"})
	srv.AddEpisode("3", plexservertest.Episode{Show: "The Office", Season: 1, Number: 1, Title: "Pilot", ViewCount: 1})
	src := b.mustAdd(srv.URL)
	b.selectAnimeLibraries(src.ID, []string{"2"})

	status, shows := b.anime()

	if status != http.StatusOK || len(shows) != 2 {
		t.Fatalf("status %d shows %+v", status, shows)
	}
	titles := []string{shows[0].Title, shows[1].Title}
	slices.Sort(titles)
	if !slices.Equal(titles, []string{"Bleach", "Frieren"}) {
		t.Fatalf("titles = %v", titles)
	}
	frieren := shows[0]
	if frieren.Title != "Frieren" {
		frieren = shows[1]
	}
	if len(frieren.Seasons) != 2 || frieren.Seasons[0].Number != 1 || frieren.Seasons[1].Number != 2 {
		t.Fatalf("seasons (specials must be excluded) = %+v", frieren.Seasons)
	}
	e1, e2 := frieren.Seasons[0].Episodes[0], frieren.Seasons[0].Episodes[1]
	if !e1.Watched || e1.Title != "The Journey's End" || e1.LastWatchedAt == nil || *e1.LastWatchedAt != "2023-11-14T22:13:20Z" {
		t.Errorf("episode 1 = %+v", e1)
	}
	if e2.Watched || e2.LastWatchedAt != nil {
		t.Errorf("episode 2 = %+v", e2)
	}
}

func TestAnimeIsEmptyUntilALibraryIsSelected(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 1, Title: "x"})
	b.mustAdd(srv.URL)

	if status, shows := b.anime(); status != http.StatusOK || shows == nil || len(shows) != 0 {
		t.Errorf("got %d %#v, want empty array", status, shows)
	}
}

func TestAnimeReturnsEveryEpisodeOfALargeLibrary(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	for n := 1; n <= 450; n++ {
		srv.AddEpisode("2", plexservertest.Episode{Show: "One Piece", Season: 1, Number: n, Title: "ep"})
	}
	src := b.mustAdd(srv.URL)
	b.selectAnimeLibraries(src.ID, []string{"2"})

	_, shows := b.anime()

	if len(shows) != 1 || len(shows[0].Seasons) != 1 || len(shows[0].Seasons[0].Episodes) != 450 {
		t.Errorf("got %d shows, want 1 show with 450 episodes", len(shows))
	}
}

func TestAnimeSkipsSelectedLibrariesRemovedFromPlex(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	srv.AddLibrary("3", "show", "Old Anime")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 1, Title: "x"})
	srv.AddEpisode("3", plexservertest.Episode{Show: "Bleach", Season: 1, Number: 1, Title: "x"})
	src := b.mustAdd(srv.URL)
	b.selectAnimeLibraries(src.ID, []string{"2", "3"})
	srv.RemoveLibrary("3")

	status, shows := b.anime()

	if status != http.StatusOK || len(shows) != 1 || shows[0].Title != "Frieren" {
		t.Errorf("got %d %+v", status, shows)
	}
}

func TestAnimeEndpointsRequireSession(t *testing.T) {
	p := startApp(t)
	b := p.browser()
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/anime"},
		{http.MethodGet, "/api/media-sources/1/libraries"},
		{http.MethodPut, "/api/media-sources/1/anime-libraries"},
	} {
		resp := b.do(c.method, p.URL+c.path)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", c.method, c.path, resp.StatusCode)
		}
	}
}

func TestAnotherUsersMediaSourceIsNotAccessible(t *testing.T) {
	p := startApp(t)
	alice := p.signedIn("sub-alice", "Alice")
	alice.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 1, Title: "x"})
	src := alice.mustAdd(srv.URL)
	alice.selectAnimeLibraries(src.ID, []string{"2"})
	bob := p.signedIn("sub-bob", "Bob")
	bob.link(p, "tok-b", "bob_plex")

	if status, _ := bob.libraries(src.ID); status != http.StatusNotFound {
		t.Errorf("libraries status %d, want 404", status)
	}
	if status := bob.selectAnimeLibraries(src.ID, []string{"2"}); status != http.StatusNotFound {
		t.Errorf("select status %d, want 404", status)
	}
	if status, shows := bob.anime(); status != http.StatusOK || len(shows) != 0 {
		t.Errorf("anime = %d %+v, want empty", status, shows)
	}
}

func TestPlexFailuresAreReportedOnLibraryAndAnimeEndpoints(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	src := b.mustAdd(srv.URL)
	b.selectAnimeLibraries(src.ID, []string{"2"})

	srv.Forbid("tok-a")
	if status, _ := b.libraries(src.ID); status != http.StatusForbidden {
		t.Errorf("libraries with rejected token = %d, want 403", status)
	}
	if status, _ := b.anime(); status != http.StatusForbidden {
		t.Errorf("anime with rejected token = %d, want 403", status)
	}

	srv.Shutdown()
	if status, _ := b.libraries(src.ID); status != http.StatusBadGateway {
		t.Errorf("libraries when down = %d, want 502", status)
	}
	if status, _ := b.anime(); status != http.StatusBadGateway {
		t.Errorf("anime when down = %d, want 502", status)
	}
}

func TestAnimeReusesCachedShowsUntilTheyExpire(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	srv.AddLibrary("2", "show", "Anime")
	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 1, Title: "One"})
	src := b.mustAdd(srv.URL)
	b.selectAnimeLibraries(src.ID, []string{"2"})
	b.anime()
	afterFirst := srv.Requests()

	srv.AddEpisode("2", plexservertest.Episode{Show: "Frieren", Season: 1, Number: 2, Title: "Two"})
	_, cached := b.anime()
	if srv.Requests() != afterFirst+1 || len(cached[0].Seasons[0].Episodes) != 1 {
		t.Errorf("expected cached shows, got %d episodes and %d extra requests", len(cached[0].Seasons[0].Episodes), srv.Requests()-afterFirst)
	}

	p.clock.Advance(time.Hour)
	_, fresh := b.anime()
	if len(fresh[0].Seasons[0].Episodes) != 2 {
		t.Errorf("expected expired cache to reload, got %d episodes", len(fresh[0].Seasons[0].Episodes))
	}
}
