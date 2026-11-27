package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	sessionCookie   = "plex_ani_sync_session"
	sessionLifetime = 30 * 24 * time.Hour
)

type browser struct {
	t       *testing.T
	app     *app
	cookies map[string]*http.Cookie
}

func (b *browser) do(method, rawURL string) *http.Response {
	b.t.Helper()
	req, _ := http.NewRequest(method, rawURL, nil)
	for _, c := range b.cookies {
		req.AddCookie(c)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	for _, c := range resp.Cookies() {
		b.cookies[c.Name] = c
	}
	return resp
}

func (b *browser) get(rawURL string) *http.Response {
	b.t.Helper()
	resp := b.do(http.MethodGet, rawURL)
	resp.Body.Close()
	return resp
}

func (b *browser) signIn() *http.Response {
	b.t.Helper()
	login := b.get(b.app.URL + "/auth/login")
	authorized := b.get(login.Header.Get("Location"))
	callback := mustParse(b.t, authorized.Header.Get("Location"))
	return b.get(b.app.URL + callback.Path + "?" + callback.RawQuery)
}

func (b *browser) logout() *http.Response {
	b.t.Helper()
	resp := b.do(http.MethodPost, b.app.URL+"/auth/logout")
	resp.Body.Close()
	return resp
}

func (b *browser) me() (int, string) {
	b.t.Helper()
	resp := b.do(http.MethodGet, b.app.URL+"/api/me")
	defer resp.Body.Close()
	var body struct {
		DisplayName string `json:"displayName"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.DisplayName
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func queryInt(t *testing.T, dbURL, sql string, args ...any) int {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var n int
	if err := conn.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLoginRedirectsToProviderWithPKCE(t *testing.T) {
	app := startApp(t)

	resp := app.browser().get(app.URL + "/auth/login")

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	loc := mustParse(t, resp.Header.Get("Location"))
	if !strings.HasPrefix(loc.String(), app.provider.URL+"/authorize?") {
		t.Fatalf("Location = %q, want provider authorize endpoint", loc)
	}
	q := loc.Query()
	if q.Get("client_id") != "plex-ani-sync" || q.Get("response_type") != "code" ||
		q.Get("redirect_uri") != "https://plex-ani-sync.example.com/auth/callback" ||
		q.Get("state") == "" || q.Get("nonce") == "" ||
		q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("unexpected authorize query %v", q)
	}
}

func TestFirstLoginCreatesUserAndSessionCarryingAuthentikName(t *testing.T) {
	app := startApp(t)
	app.provider.SignInAs("sub-alice", "Alice")
	b := app.browser()

	resp := b.signIn()

	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/" {
		t.Fatalf("callback = %d -> %q, want 302 -> /", resp.StatusCode, resp.Header.Get("Location"))
	}
	c := b.cookies[sessionCookie]
	if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %+v, want HttpOnly Secure SameSite=Lax", c)
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM users WHERE oidc_issuer = $1 AND oidc_subject = 'sub-alice'", app.provider.URL); got != 1 {
		t.Errorf("users = %d, want 1", got)
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions WHERE display_name = 'Alice'"); got != 1 {
		t.Errorf("sessions carrying name = %d, want 1", got)
	}
	if status, name := b.me(); status != 200 || name != "Alice" {
		t.Errorf("/api/me = %d %q, want 200 Alice", status, name)
	}
}

func TestRepeatLoginReusesUserAndShowsLatestName(t *testing.T) {
	app := startApp(t)
	app.provider.SignInAs("sub-alice", "Alice")
	app.browser().signIn()

	app.provider.SignInAs("sub-alice", "Alice Renamed")
	b := app.browser()
	b.signIn()

	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM users"); got != 1 {
		t.Errorf("users = %d, want 1", got)
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions"); got != 2 {
		t.Errorf("sessions = %d, want 2", got)
	}
	if _, name := b.me(); name != "Alice Renamed" {
		t.Errorf("name = %q, want Alice Renamed", name)
	}
}

func TestDifferentSubjectsAreDifferentUsers(t *testing.T) {
	app := startApp(t)
	app.provider.SignInAs("sub-alice", "Alice")
	app.browser().signIn()
	app.provider.SignInAs("sub-bob", "Bob")
	app.browser().signIn()

	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM users"); got != 2 {
		t.Errorf("users = %d, want 2", got)
	}
}

func TestMeWithoutValidSessionIs401(t *testing.T) {
	app := startApp(t)
	b := app.browser()
	if status, _ := b.me(); status != http.StatusUnauthorized {
		t.Errorf("no cookie: status = %d, want 401", status)
	}
	b.cookies[sessionCookie] = &http.Cookie{Name: sessionCookie, Value: "not-a-real-session"}
	if status, _ := b.me(); status != http.StatusUnauthorized {
		t.Errorf("unknown cookie: status = %d, want 401", status)
	}
}

func TestSessionIDIsStoredOnlyAsHash(t *testing.T) {
	app := startApp(t)
	b := app.browser()
	b.signIn()

	id := b.cookies[sessionCookie].Value
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions WHERE id_hash = $1::bytea", []byte(id)); got != 0 {
		t.Error("raw session id found in sessions table")
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions WHERE id_hash = sha256($1::bytea)", []byte(id)); got != 1 {
		t.Errorf("sessions with sha256(id) hash = %d, want 1", got)
	}
}

func TestLogoutRevokesSessionAndClearsCookie(t *testing.T) {
	app := startApp(t)
	b := app.browser()
	b.signIn()
	old := *b.cookies[sessionCookie]

	if resp := b.logout(); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	if c := b.cookies[sessionCookie]; c.Value != "" || c.MaxAge >= 0 {
		t.Errorf("cookie = %+v, want cleared", c)
	}
	replay := app.browser()
	replay.cookies[sessionCookie] = &old
	if status, _ := replay.me(); status != http.StatusUnauthorized {
		t.Errorf("replayed cookie: status = %d, want 401", status)
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions WHERE revoked_at IS NOT NULL"); got != 1 {
		t.Errorf("revoked sessions = %d, want 1", got)
	}
}

func TestSessionSurvivesAppRestart(t *testing.T) {
	app := startApp(t)
	app.provider.SignInAs("sub-alice", "Alice")
	b := app.browser()
	b.signIn()
	app.server.Close()

	restarted := startApp(t, restartOf(app))
	b.app = restarted

	if status, name := b.me(); status != 200 || name != "Alice" {
		t.Errorf("/api/me after restart = %d %q, want 200 Alice", status, name)
	}
}

func TestExpiredSessionIs401(t *testing.T) {
	app := startApp(t)
	b := app.browser()
	b.signIn()

	app.clock.Advance(sessionLifetime - time.Second)
	if status, _ := b.me(); status != http.StatusOK {
		t.Fatalf("status just before expiry = %d, want 200", status)
	}

	app.clock.Advance(time.Second)
	if status, _ := b.me(); status != http.StatusUnauthorized {
		t.Errorf("status at expiry = %d, want 401", status)
	}
}

func TestCallbackRejectsBadState(t *testing.T) {
	app := startApp(t)
	b := app.browser()
	b.get(app.URL + "/auth/login")

	resp := b.get(app.URL + "/auth/callback?code=x&state=forged")

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions"); got != 0 {
		t.Errorf("sessions = %d, want 0", got)
	}
}

func TestCallbackWithProviderErrorCreatesNoSession(t *testing.T) {
	app := startApp(t)
	b := app.browser()
	login := b.get(app.URL + "/auth/login")
	state := mustParse(t, login.Header.Get("Location")).Query().Get("state")

	resp := b.get(app.URL + "/auth/callback?error=access_denied&state=" + url.QueryEscape(state))

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions"); got != 0 {
		t.Errorf("sessions = %d, want 0", got)
	}
	if app.provider.TokenRequests() != 0 {
		t.Error("code exchange attempted despite provider error")
	}
}

func TestCallbackRejectsNonceMismatch(t *testing.T) {
	app := startApp(t)
	b := app.browser()
	login := b.get(app.URL + "/auth/login")
	authorize := mustParse(t, login.Header.Get("Location"))
	q := authorize.Query()
	q.Set("nonce", "tampered")
	authorize.RawQuery = q.Encode()
	callback := mustParse(t, b.get(authorize.String()).Header.Get("Location"))

	resp := b.get(app.URL + callback.Path + "?" + callback.RawQuery)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	if got := queryInt(t, app.dbURL, "SELECT count(*) FROM sessions"); got != 0 {
		t.Errorf("sessions = %d, want 0", got)
	}
}
