package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

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
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")
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
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")

	if got := b.mediaSources(); got == nil || len(got) != 0 {
		t.Errorf("list = %#v, want empty array", got)
	}
}

func TestAccessCheckFailuresReturnDistinctErrorsAndSaveNothing(t *testing.T) {
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")
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
	if n := queryInt(t, p.dbURL, "SELECT count(*) FROM media_sources"); n != 0 {
		t.Errorf("media sources = %d, want 0", n)
	}
}

func TestAddingWithoutALinkedPlexAccountIsRejectedWithoutContactingTheServer(t *testing.T) {
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")
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
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")
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
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")
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
	p := startPlexApp(t)
	alice := p.signedIn(t, "sub-alice", "Alice")
	bob := p.signedIn(t, "sub-bob", "Bob")
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
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	b.mustAdd(srv.URL)

	b.link(p, "tok-new", "alice_plex")
	srv.Allow("tok-new")

	if got := b.mediaSources(); len(got) != 1 {
		t.Errorf("relinking dropped media sources: %+v", got)
	}
	if n := queryInt(t, p.dbURL, "SELECT count(*) FROM information_schema.columns WHERE table_name = 'media_sources' AND column_name ILIKE '%token%'"); n != 0 {
		t.Error("media_sources has a token column")
	}
}

func TestMediaSourceEndpointsRequireSession(t *testing.T) {
	p := startPlexApp(t)
	b := newBrowser(t, p.app)

	if status, _, _ := b.addMediaSource("http://x"); status != http.StatusUnauthorized {
		t.Errorf("add status %d", status)
	}
	if resp := b.do(http.MethodGet, p.app.URL+"/api/media-sources"); resp.StatusCode != http.StatusUnauthorized {
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
	p := startPlexApp(t)
	b := p.signedIn(t, "sub-alice", "Alice")
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
	p := startPlexApp(t)
	alice := p.signedIn(t, "sub-alice", "Alice")
	bob := p.signedIn(t, "sub-bob", "Bob")
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
	p := startPlexApp(t)
	b := newBrowser(t, p.app)

	if status := b.removeMediaSource(1); status != http.StatusUnauthorized {
		t.Errorf("status %d", status)
	}
}
