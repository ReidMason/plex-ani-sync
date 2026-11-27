package server_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const stateCookie = "plex_ani_sync_anilist_state"

func (b *browser) startAuthorisation() *http.Response {
	b.t.Helper()
	return b.get(b.app.URL + "/auth/anilist/start")
}

// approveOnAniList follows the start redirect to the fake AniList and returns the callback URL it sends the browser back to.
func (b *browser) approveOnAniList(start *http.Response) string {
	b.t.Helper()
	resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Get(start.Header.Get("Location"))
	if err != nil {
		b.t.Fatal(err)
	}
	resp.Body.Close()
	back := mustParse(b.t, resp.Header.Get("Location"))
	return b.app.URL + back.Path + "?" + back.RawQuery
}

func (b *browser) authorise() *http.Response {
	b.t.Helper()
	b.app.clock.Advance(time.Minute)
	return b.get(b.approveOnAniList(b.startAuthorisation()))
}

func (b *browser) authorisedAs() (string, bool) {
	b.t.Helper()
	got := b.trackerStatus()
	name := ""
	if got.Username != nil {
		name = *got.Username
	}
	return name, got.Authorised != nil && *got.Authorised
}

func TestStartingAuthorisationRedirectsToAniListWithState(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	b := p.signedIn("sub-alice", "Alice")

	resp := b.startAuthorisation()

	loc := mustParse(t, resp.Header.Get("Location"))
	q := loc.Query()
	if resp.StatusCode != http.StatusFound || !strings.HasPrefix(resp.Header.Get("Location"), p.anilist.OAuthURL()+"/authorize") {
		t.Fatalf("status %d location %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if q.Get("client_id") != "anilist-client" || q.Get("response_type") != "code" || q.Get("state") == "" ||
		q.Get("redirect_uri") != "https://plex-ani-sync.example.com/auth/anilist/callback" {
		t.Errorf("query %v", q)
	}
}

func TestAuthorisingStoresTheTokenAndTakesTheUsernameFromAniList(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")

	resp := b.authorise()

	name, authorised := b.authorisedAs()
	if resp.Header.Get("Location") != "/settings?anilist=authorised" || name != "alice_anime" || !authorised {
		t.Errorf("location %q, username %q, authorised %v", resp.Header.Get("Location"), name, authorised)
	}
	tokens := p.anilist.IssuedTokens()
	if len(tokens) != 1 {
		t.Fatalf("tokens issued: %v", tokens)
	}
	assertNotStored(t, p.dbURL, []byte(tokens[0]))
}

func TestAuthorisationSurvivesARestart(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	b.authorise()

	restarted := startApp(t, restartOf(p), withAniListCredentials())
	b2 := restarted.signedIn("sub-alice", "Alice")

	if name, authorised := b2.authorisedAs(); name != "alice_anime" || !authorised {
		t.Errorf("username %q authorised %v", name, authorised)
	}
}

func TestAuthorisingReturnsToTheSetupWizardWhenStartedThere(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	p.clock.Advance(time.Minute)

	start := b.get(b.app.URL + "/auth/anilist/start?return=setup")
	resp := b.get(b.approveOnAniList(start))

	if resp.Header.Get("Location") != "/setup/anilist?anilist=authorised" {
		t.Errorf("location %q", resp.Header.Get("Location"))
	}
}

func TestAuthorisingAsAnotherAccountReplacesTheUsernameAndSaysSo(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	publicUser(p, "alice_anime")
	p.anilist.ApproveAs("bob_anime")
	b := p.signedIn("sub-alice", "Alice")
	b.linkAniList("alice_anime")

	resp := b.authorise()

	name, authorised := b.authorisedAs()
	if resp.Header.Get("Location") != "/settings?anilist=authorised" || name != "bob_anime" || !authorised {
		t.Errorf("location %q, username %q, authorised %v", resp.Header.Get("Location"), name, authorised)
	}
	first, second := b.statusKeepingCookies(), b.statusKeepingCookies()
	if first.PreviousUsername == nil || *first.PreviousUsername != "alice_anime" || second.PreviousUsername != nil {
		t.Errorf("previous username: first %v, second %v", first.PreviousUsername, second.PreviousUsername)
	}
}

func TestAuthorisingAsTheLinkedAccountDoesNotReportAChange(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	publicUser(p, "alice_anime")
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	b.linkAniList("alice_anime")

	resp := b.authorise()

	if resp.Header.Get("Location") != "/settings?anilist=authorised" || b.statusKeepingCookies().PreviousUsername != nil {
		t.Errorf("location %q", resp.Header.Get("Location"))
	}
}

func TestACallbackWithoutAMatchingStateIsRejectedAndChangesNothing(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	publicUser(p, "alice_anime")
	p.anilist.ApproveAs("mallory")
	b := p.signedIn("sub-alice", "Alice")
	b.linkAniList("alice_anime")
	p.clock.Advance(time.Minute)
	callback := mustParse(t, b.approveOnAniList(b.startAuthorisation()))

	forged := url.Values{"code": {callback.Query().Get("code")}, "state": {"forged"}}
	missing := url.Values{"code": {callback.Query().Get("code")}}
	for name, q := range map[string]url.Values{"forged": forged, "missing": missing} {
		if loc := b.get(p.URL + "/auth/anilist/callback?" + q.Encode()).Header.Get("Location"); loc != "/settings?anilist=invalid" {
			t.Errorf("%s state: location %q", name, loc)
		}
	}

	if name, authorised := b.authorisedAs(); name != "alice_anime" || authorised {
		t.Errorf("username %q authorised %v", name, authorised)
	}
}

func TestAReplayedCallbackIsRejected(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	p.clock.Advance(time.Minute)
	callback := b.approveOnAniList(b.startAuthorisation())
	b.get(callback)
	b.trackerRequest(http.MethodDelete, "")

	loc := b.get(callback).Header.Get("Location")

	if name, authorised := b.authorisedAs(); loc != "/settings?anilist=invalid" || name != "" || authorised {
		t.Errorf("location %q username %q authorised %v", loc, name, authorised)
	}
}

func TestAStateIssuedToAnotherSessionIsRejected(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	alice := p.signedIn("sub-alice", "Alice")
	bob := p.signedIn("sub-bob", "Bob")
	p.clock.Advance(time.Minute)
	callback := alice.approveOnAniList(alice.startAuthorisation())
	bob.cookies[stateCookie] = alice.cookies[stateCookie]

	loc := bob.get(callback).Header.Get("Location")

	if name, authorised := bob.authorisedAs(); loc != "/settings?anilist=invalid" || name != "" || authorised {
		t.Errorf("location %q username %q authorised %v", loc, name, authorised)
	}
}

func TestAnExpiredStateIsRejectedEvenIfTheCookieSurvives(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	p.clock.Advance(time.Minute)
	callback := b.approveOnAniList(b.startAuthorisation())
	p.clock.Advance(11 * time.Minute)

	loc := b.get(callback).Header.Get("Location")

	if name, authorised := b.authorisedAs(); loc != "/settings?anilist=invalid" || name != "" || authorised {
		t.Errorf("location %q username %q authorised %v", loc, name, authorised)
	}
}

func (b *browser) statusKeepingCookies() trackerStatus {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/trackers/anilist")
	defer resp.Body.Close()
	var out trackerStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		b.t.Fatal(err)
	}
	return out
}

func TestACallbackWhileSignedOutIsRejected(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	p.clock.Advance(time.Minute)
	callback := b.approveOnAniList(b.startAuthorisation())
	b.logout()

	status := b.get(callback).StatusCode
	startStatus := p.browser().get(p.URL + "/auth/anilist/start").StatusCode

	if status != http.StatusUnauthorized || startStatus != http.StatusUnauthorized {
		t.Errorf("callback %d start %d", status, startStatus)
	}
	again := p.signedIn("sub-alice", "Alice")
	if name, authorised := again.authorisedAs(); name != "" || authorised {
		t.Errorf("username %q authorised %v", name, authorised)
	}
}

func TestDenyingAuthorisationLeavesTheTrackerUnchanged(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	publicUser(p, "alice_anime")
	p.anilist.DenyAuthorisation()
	b := p.signedIn("sub-alice", "Alice")
	b.linkAniList("alice_anime")

	resp := b.authorise()

	name, authorised := b.authorisedAs()
	if resp.Header.Get("Location") != "/settings?anilist=denied" || name != "alice_anime" || authorised {
		t.Errorf("location %q, username %q, authorised %v", resp.Header.Get("Location"), name, authorised)
	}
}

func TestAFailedTokenExchangeLeavesTheTrackerUnchanged(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	publicUser(p, "alice_anime")
	p.anilist.ApproveAs("bob_anime")
	p.anilist.FailTokenExchange()
	b := p.signedIn("sub-alice", "Alice")
	b.linkAniList("alice_anime")

	resp := b.authorise()

	name, authorised := b.authorisedAs()
	if resp.Header.Get("Location") != "/settings?anilist=failed" || name != "alice_anime" || authorised {
		t.Errorf("location %q, username %q, authorised %v", resp.Header.Get("Location"), name, authorised)
	}
}

func TestAuthorisationIsUnavailableWithoutCredentials(t *testing.T) {
	t.Parallel()
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")

	start := b.startAuthorisation().StatusCode
	callback := b.get(p.URL + "/auth/anilist/callback?code=x&state=y").StatusCode

	if start != http.StatusNotFound || callback != http.StatusNotFound {
		t.Errorf("start %d callback %d", start, callback)
	}
}

func TestRemovingTheTrackerClearsAuthorisation(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	b.authorise()

	b.trackerRequest(http.MethodDelete, "")

	if name, authorised := b.authorisedAs(); name != "" || authorised {
		t.Errorf("username %q authorised %v", name, authorised)
	}
	assertNotStored(t, p.dbURL, []byte(p.anilist.IssuedTokens()[0]))
	assertNotStored(t, p.dbURL, []byte("alice_anime"))
}

func TestRelinkingADifferentUsernameDropsAuthorisation(t *testing.T) {
	t.Parallel()
	p := startApp(t, withAniListCredentials())
	publicUser(p, "other")
	p.anilist.ApproveAs("alice_anime")
	b := p.signedIn("sub-alice", "Alice")
	b.authorise()

	b.linkAniList("other")

	if name, authorised := b.authorisedAs(); name != "other" || authorised {
		t.Errorf("username %q authorised %v", name, authorised)
	}
}
