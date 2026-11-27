package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/plexservertest"
)

type pinResponse struct {
	ID      int64  `json:"id"`
	Code    string `json:"code"`
	AuthURL string `json:"authUrl"`
}

type pollResponse struct {
	Status   string `json:"status"`
	Username string `json:"username"`
}

func (b *browser) post(rawURL string) (*http.Response, []byte) {
	b.t.Helper()
	resp := b.do(http.MethodPost, rawURL)
	defer resp.Body.Close()
	var buf []byte
	buf, _ = io.ReadAll(resp.Body)
	return resp, buf
}

func (b *browser) createPIN() pinResponse {
	b.t.Helper()
	resp, body := b.post(b.app.URL + "/api/plex/pins")
	if resp.StatusCode != http.StatusOK {
		b.t.Fatalf("create pin: status %d: %s", resp.StatusCode, body)
	}
	var pin pinResponse
	if err := json.Unmarshal(body, &pin); err != nil {
		b.t.Fatal(err)
	}
	return pin
}

func (b *browser) pollPIN(pin pinResponse) (int, pollResponse) {
	b.t.Helper()
	resp, body := b.post(b.app.URL + "/api/plex/pins/" + strconv.FormatInt(pin.ID, 10) + "/poll?code=" + pin.Code)
	var out pollResponse
	_ = json.Unmarshal(body, &out)
	return resp.StatusCode, out
}

func (b *browser) plexUsername() string {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/me")
	defer resp.Body.Close()
	var body struct {
		Plex *struct {
			Username string `json:"username"`
		} `json:"plex"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Plex == nil {
		return ""
	}
	return body.Plex.Username
}

func TestPlexPINFlowLinksAccountAndShowsUsername(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")

	pin := b.createPIN()
	if pin.AuthURL == "" {
		t.Error("no auth url for the popup")
	}
	if status, got := b.pollPIN(pin); status != 200 || got.Status != "pending" {
		t.Fatalf("before approval: %d %+v", status, got)
	}
	if got := b.plexUsername(); got != "" {
		t.Fatalf("username before linking = %q", got)
	}

	p.tv.Approve(pin.Code, "plex-token-1", "alice_plex")
	status, got := b.pollPIN(pin)
	if status != 200 || got.Status != "linked" || got.Username != "alice_plex" {
		t.Fatalf("after approval: %d %+v", status, got)
	}
	if got := b.plexUsername(); got != "alice_plex" {
		t.Errorf("me.plex.username = %q", got)
	}
}

func (b *browser) link(p *app, token, username string) {
	b.t.Helper()
	pin := b.createPIN()
	p.tv.Approve(pin.Code, token, username)
	if _, got := b.pollPIN(pin); got.Status != "linked" {
		b.t.Fatalf("link failed: %+v", got)
	}
}

func TestPlexEndpointsRequireSession(t *testing.T) {
	p := startApp(t)
	b := p.browser()

	if resp, _ := b.post(p.URL + "/api/plex/pins"); resp.StatusCode != 401 {
		t.Errorf("create pin status %d", resp.StatusCode)
	}
	if resp, _ := b.post(p.URL + "/api/plex/pins/1/poll?code=x"); resp.StatusCode != 401 {
		t.Errorf("poll status %d", resp.StatusCode)
	}
}

func TestPlexTokenIsNotStoredInPlaintext(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")

	b.link(p, "plex-token-secret", "alice_plex")

	assertNotStored(t, p.dbURL, []byte("plex-token-secret"))
	if got := b.plexUsername(); got != "alice_plex" {
		t.Errorf("username = %q, want alice_plex", got)
	}
}

func TestRelinkingReplacesTheUsersSinglePlexAccount(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")

	b.link(p, "tok-old", "alice_old")
	b.link(p, "tok-new", "alice_new")

	if got := b.plexUsername(); got != "alice_new" {
		t.Errorf("username = %q", got)
	}
}

func TestPlexAccountsAreSeparatePerUser(t *testing.T) {
	p := startApp(t)
	alice := p.signedIn("sub-alice", "Alice")
	bob := p.signedIn("sub-bob", "Bob")

	alice.link(p, "tok-a", "alice_plex")

	if got := bob.plexUsername(); got != "" {
		t.Errorf("bob sees %q", got)
	}
	bob.link(p, "tok-b", "bob_plex")
	if got := alice.plexUsername(); got != "alice_plex" {
		t.Errorf("alice sees %q", got)
	}
}

func TestPlexClientIdentifierIsGeneratedOnceAndSurvivesRestart(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.createPIN()

	app2 := startApp(t, restartOf(p))
	b2 := app2.browser()
	b2.cookies = b.cookies
	b2.createPIN()

	ids := p.tv.ClientIdentifiers()
	if len(ids) != 1 || len(ids[0]) < 32 {
		t.Errorf("client identifiers = %v, want one uuid", ids)
	}
}

func TestPollingWhilePlexTVIsDownFailsWithoutLinking(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	pin := b.createPIN()
	p.tv.Approve(pin.Code, "tok", "alice_plex")
	p.tv.SetDown(true)

	if status, _ := b.pollPIN(pin); status != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", status)
	}
	if b.plexLinked() {
		t.Error("plex account linked despite plex.tv being down")
	}
}

func TestDashboardLoadRefreshesThePlexUsernameFromPlexTV(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_old")

	p.tv.SetUsername("tok-a", "alice_renamed")
	p.clock.Advance(usernameTTL + time.Second)

	if got := b.plexUsername(); got != "alice_renamed" {
		t.Errorf("username = %q, want alice_renamed", got)
	}
	p.tv.SetDown(true)
	if got := b.plexUsername(); got != "alice_renamed" {
		t.Errorf("cached username = %q, want alice_renamed", got)
	}
}

func TestDashboardLoadsWithCachedUsernameWhenPlexTVIsDown(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	b.plexUsername()
	p.clock.Advance(usernameTTL + time.Second)
	p.tv.SetDown(true)

	resp := b.do(http.MethodGet, p.URL+"/api/me")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if got := b.plexUsername(); got != "alice_plex" {
		t.Errorf("username = %q", got)
	}
}

func TestDashboardOmitsUsernameWhenPlexTVIsDownAndNothingIsCached(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	p.tv.SetDown(true)

	if got := b.plexUsername(); got != "" {
		t.Errorf("username = %q, want none", got)
	}
	if !b.plexLinked() || b.plexNeedsReSignIn() {
		t.Error("account should show as linked and healthy without a name")
	}
}

func TestRepeatedDashboardLoadsHitPlexTVOncePerTTL(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	before := p.tv.UserLookups()

	for range 3 {
		b.plexUsername()
	}
	if got := p.tv.UserLookups() - before; got != 1 {
		t.Fatalf("plex.tv lookups within ttl = %d, want 1", got)
	}

	p.clock.Advance(usernameTTL + time.Second)
	b.plexUsername()
	b.plexUsername()
	if got := p.tv.UserLookups() - before; got != 2 {
		t.Errorf("plex.tv lookups after ttl = %d, want 2", got)
	}
}

func TestRelinkingInvalidatesTheCachedUsername(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-old", "alice_old")
	if got := b.plexUsername(); got != "alice_old" {
		t.Fatalf("username = %q", got)
	}

	b.link(p, "tok-new", "alice_new")

	if got := b.plexUsername(); got != "alice_new" {
		t.Errorf("username = %q, want alice_new", got)
	}
}

func (b *browser) plexLinked() bool {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/me")
	defer resp.Body.Close()
	var body struct {
		Plex *struct{} `json:"plex"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return body.Plex != nil
}

func (b *browser) plexNeedsReSignIn() bool {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/me")
	defer resp.Body.Close()
	var body struct {
		Plex *struct {
			NeedsReSignIn bool `json:"needsReSignIn"`
		} `json:"plex"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return body.Plex != nil && body.Plex.NeedsReSignIn
}

func TestRevokedTokenShowsNeedsReSignInAndReSigningInKeepsMediaSources(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	srv := plexservertest.New(t)
	srv.Allow("tok-a")
	b.mustAdd(srv.URL)
	if b.plexNeedsReSignIn() {
		t.Fatal("needs re-sign-in before revocation")
	}

	p.tv.RevokeToken("tok-a")
	p.clock.Advance(usernameTTL + time.Second)
	if !b.plexNeedsReSignIn() {
		t.Fatal("revoked token not flagged")
	}
	if got := b.plexUsername(); got != "" {
		t.Errorf("username after revocation = %q, want none", got)
	}

	b.link(p, "tok-b", "alice_plex")
	if b.plexNeedsReSignIn() {
		t.Error("still needs re-sign-in after re-linking")
	}
	if got := b.mediaSources(); len(got) != 1 {
		t.Errorf("media sources = %+v, want 1 kept", got)
	}
}

func TestPlexTVOutageDoesNotFlagReSignIn(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")
	p.tv.SetDown(true)

	if b.plexNeedsReSignIn() {
		t.Error("outage flagged as needs re-sign-in")
	}
}

func TestUndecryptableTokenAfterKeyRotationFlagsReSignIn(t *testing.T) {
	p := startApp(t)
	b := p.signedIn("sub-alice", "Alice")
	b.link(p, "tok-a", "alice_plex")

	rotated := startApp(t, restartOf(p), withEncryptionKey(bytes.Repeat([]byte{7}, 32)))
	b2 := rotated.browser()
	b2.cookies = b.cookies

	if !b2.plexNeedsReSignIn() {
		t.Error("undecryptable token not flagged")
	}
	if got := b2.plexUsername(); got != "" {
		t.Errorf("username = %q, want none", got)
	}
}
