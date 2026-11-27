package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/anilisttest"
)

type trackerResponse struct {
	Username *string `json:"username"`
	Error    string  `json:"error"`
	Until    string  `json:"until"`
}

func (b *browser) trackerRequest(method, username string) (int, trackerResponse) {
	b.t.Helper()
	var body *bytes.Reader
	if method == http.MethodPut {
		payload, _ := json.Marshal(map[string]string{"username": username})
		body = bytes.NewReader(payload)
	} else {
		body = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, b.app.URL+"/api/trackers/anilist", body)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range b.cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out trackerResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (b *browser) linkAniList(username string) (int, trackerResponse) {
	b.app.clock.Advance(time.Minute)
	return b.trackerRequest(http.MethodPut, username)
}

func (b *browser) aniListUsername() string {
	b.t.Helper()
	status, got := b.trackerRequest(http.MethodGet, "")
	if status != http.StatusOK {
		b.t.Fatalf("get tracker: %d", status)
	}
	if got.Username == nil {
		return ""
	}
	return *got.Username
}

func publicUser(p *app, name string) {
	p.anilist.SetUserList(name, anilisttest.ListEntry{MediaID: 1, Status: "CURRENT", Progress: 3})
}

func TestLinkingAPublicAniListUsernameSavesIt(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	publicUser(p, "alice_anime")
	b := p.signedIn("sub-alice", "Alice")

	status, got := b.linkAniList("alice_anime")

	if status != http.StatusOK || got.Username == nil || *got.Username != "alice_anime" {
		t.Fatalf("status %d got %+v", status, got)
	}
	if name := b.aniListUsername(); name != "alice_anime" {
		t.Errorf("stored username = %q", name)
	}
}

func TestAniListUsernameStartsUnlinked(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")

	if name := b.aniListUsername(); name != "" {
		t.Errorf("username = %q", name)
	}
}

func TestUnknownAndPrivateUsernamesReturnDistinctErrorsAndSaveNothing(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	p.anilist.SetPrivateUser("hidden")
	b := p.signedIn("sub-alice", "Alice")

	cases := []struct {
		name     string
		username string
		status   int
		code     string
	}{
		{"unknown", "nobody", http.StatusNotFound, "user_not_found"},
		{"private", "hidden", http.StatusUnprocessableEntity, "private_list"},
		{"empty", "  ", http.StatusBadRequest, "invalid_username"},
	}
	for _, c := range cases {
		status, got := b.linkAniList(c.username)
		if status != c.status || got.Error != c.code {
			t.Errorf("%s: got %d %q, want %d %q", c.name, status, got.Error, c.status, c.code)
		}
	}
	if name := b.aniListUsername(); name != "" {
		t.Errorf("saved %q", name)
	}
}

func TestAFailedChangeKeepsThePreviousUsername(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	publicUser(p, "alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	b.linkAniList("alice_anime")

	b.linkAniList("nobody")

	if name := b.aniListUsername(); name != "alice_anime" {
		t.Errorf("username = %q", name)
	}
}

func TestAniListUsernameCanBeChangedAndRemoved(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	publicUser(p, "first")
	publicUser(p, "second")
	b := p.signedIn("sub-alice", "Alice")
	b.linkAniList("first")

	b.linkAniList("second")
	changed := b.aniListUsername()
	status, _ := b.trackerRequest(http.MethodDelete, "")
	removed := b.aniListUsername()

	if changed != "second" || status != http.StatusNoContent || removed != "" {
		t.Errorf("changed %q, delete %d, removed %q", changed, status, removed)
	}
}

func TestUsersCannotSeeOrChangeEachOthersTracker(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	publicUser(p, "alice_anime")
	publicUser(p, "bob_anime")
	alice := p.signedIn("sub-alice", "Alice")
	bob := p.signedIn("sub-bob", "Bob")
	alice.linkAniList("alice_anime")

	seenByBob := bob.aniListUsername()
	bob.trackerRequest(http.MethodDelete, "")
	bob.linkAniList("bob_anime")

	if seenByBob != "" || alice.aniListUsername() != "alice_anime" || bob.aniListUsername() != "bob_anime" {
		t.Errorf("bob saw %q, alice has %q, bob has %q", seenByBob, alice.aniListUsername(), bob.aniListUsername())
	}
}

func TestTrackerEndpointsRequireASession(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	b := p.browser()

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		if status, _ := b.trackerRequest(method, "x"); status != http.StatusUnauthorized {
			t.Errorf("%s status %d", method, status)
		}
	}
}

func TestLinkingReportsAniListThrottleWithRetryTime(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	publicUser(p, "alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	p.anilist.FailWithRetryAfter(http.StatusTooManyRequests, 90)
	b.linkAniList("alice_anime")

	status, got := b.trackerRequest(http.MethodPut, "alice_anime")

	want := p.clock.Now().Add(90 * time.Second).UTC().Format(time.RFC3339)
	if status != http.StatusTooManyRequests || got.Error != "throttled" || got.Until != want {
		t.Errorf("status %d got %+v want until %s", status, got, want)
	}
}

func TestAniListFailuresAreBadGateway(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	p.anilist.FailWith(http.StatusInternalServerError)

	status, got := b.linkAniList("alice_anime")

	if status != http.StatusBadGateway || got.Error != "anilist_unavailable" {
		t.Errorf("status %d got %+v", status, got)
	}
}
