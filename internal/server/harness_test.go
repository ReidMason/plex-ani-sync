package server_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/anilisttest"
	"github.com/ReidMason/plex-ani-sync/internal/config"
	"github.com/ReidMason/plex-ani-sync/internal/datasettest"
	"github.com/ReidMason/plex-ani-sync/internal/mappings"
	"github.com/ReidMason/plex-ani-sync/internal/oidctest"
	"github.com/ReidMason/plex-ani-sync/internal/plexservertest"
	"github.com/ReidMason/plex-ani-sync/internal/plextest"
	"github.com/ReidMason/plex-ani-sync/internal/server"
	"github.com/jackc/pgx/v5"
)

const (
	usernameTTL         = 5 * time.Minute
	aniListPollInterval = 5 * time.Millisecond
)

var testEncryptionKey = bytes.Repeat([]byte{9}, 32)

// fakeClock only moves on Advance, which runs every tick that falls due before returning.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers map[*fakeTicker]struct{}
}

type fakeTicker struct {
	period time.Duration
	next   time.Time
	fn     func()
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), tickers: map[*fakeTicker]struct{}{}}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Every(d time.Duration, fn func()) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	tk := &fakeTicker{period: d, next: c.now.Add(d), fn: fn}
	c.tickers[tk] = struct{}{}
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.tickers, tk)
	}
}

// Advance moves time forward by d; like time.Ticker, a ticker that fell several periods behind fires once.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	var due []func()
	for tk := range c.tickers {
		if c.now.Before(tk.next) {
			continue
		}
		due = append(due, tk.fn)
		tk.next = tk.next.Add((c.now.Sub(tk.next)/tk.period + 1) * tk.period)
	}
	c.mu.Unlock()
	for _, fn := range due {
		fn()
	}
}

// app is a running server wired to fakes for every external dependency.
type app struct {
	t          *testing.T
	URL        string
	server     *httptest.Server
	dbURL      string
	key        []byte
	provider   *oidctest.Provider
	tv         *plextest.TV
	plexServer *plexservertest.Server
	datasets   *datasettest.Host
	anilist    *anilisttest.Host
	clock      *fakeClock
}

type appSettings struct {
	app
	restart          *app
	datasetSetup     []func(*datasettest.Host)
	customMappings   []mappings.Mapping
	refreshInterval  time.Duration
	aniListPerMinute int
	configEdits      []func(*server.Config)
}

type appOption func(*appSettings)

func withDatasets(h *datasettest.Host) appOption { return func(s *appSettings) { s.datasets = h } }

func withEncryptionKey(key []byte) appOption { return func(s *appSettings) { s.key = key } }

func withCustomMappings(m []mappings.Mapping) appOption {
	return func(s *appSettings) { s.customMappings = m }
}

func withDatasetRefreshInterval(d time.Duration) appOption {
	return func(s *appSettings) { s.refreshInterval = d }
}

func withAniListRequestsPerMinute(n int) appOption {
	return func(s *appSettings) { s.aniListPerMinute = n }
}

func withFribb(entries ...datasettest.FribbEntry) appOption {
	return withDatasetSetup(func(h *datasettest.Host) { h.SetFribb(entries...) })
}

func withAnimeLists(xml string) appOption {
	return withDatasetSetup(func(h *datasettest.Host) { h.SetAnimeLists(xml) })
}

func withFailingDatasets() appOption {
	return withDatasetSetup((*datasettest.Host).Fail)
}

func withDatasetSetup(fn func(*datasettest.Host)) appOption {
	return func(s *appSettings) { s.datasetSetup = append(s.datasetSetup, fn) }
}

func withConfigEdit(fn func(*server.Config)) appOption {
	return func(s *appSettings) { s.configEdits = append(s.configEdits, fn) }
}

// restartOf starts against prev's database, key, clock and fakes, unless other options replace them.
func restartOf(prev *app) appOption { return func(s *appSettings) { s.restart = prev } }

func startApp(t *testing.T, opts ...appOption) *app {
	t.Helper()
	a, err := tryStartApp(t, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func tryStartApp(t *testing.T, opts ...appOption) (*app, error) {
	t.Helper()
	var s appSettings
	for _, opt := range opts {
		opt(&s)
	}
	a := s.app
	a.t = t
	prev := s.restart
	if prev == nil {
		prev = &app{}
	}
	a.dbURL = firstSet(a.dbURL, prev.dbURL, func() string { return newDatabase(t) })
	a.key = firstSet(a.key, prev.key, func() []byte { return testEncryptionKey })
	a.provider = firstSet(a.provider, prev.provider, func() *oidctest.Provider { return oidctest.New(t) })
	a.tv = firstSet(a.tv, prev.tv, func() *plextest.TV { return plextest.New(t) })
	a.plexServer = firstSet(a.plexServer, prev.plexServer, func() *plexservertest.Server { return plexservertest.New(t) })
	a.datasets = firstSet(a.datasets, prev.datasets, func() *datasettest.Host { return datasettest.New(t) })
	a.anilist = firstSet(a.anilist, prev.anilist, func() *anilisttest.Host { return anilisttest.New(t) })
	a.clock = firstSet(a.clock, prev.clock, newFakeClock)
	for _, fn := range s.datasetSetup {
		fn(a.datasets)
	}

	cfg := server.Config{
		Env: config.Config{
			DatabaseURL:              a.dbURL,
			PublicBaseURL:            "https://plex-ani-sync.example.com",
			OIDCIssuerURL:            a.provider.URL,
			OIDCClientID:             "plex-ani-sync",
			OIDCClientSecret:         "super-secret-value",
			EncryptionKey:            a.key,
			AniListRequestsPerMinute: s.aniListPerMinute,
		},
		Assets: fstest.MapFS{
			"index.html":          {Data: []byte("<!doctype html><title>plex-ani-sync</title>")},
			"assets/index-abc.js": {Data: []byte("console.log(1)")},
		},
		ExternalURLs: server.ExternalURLs{
			PlexTV:       a.tv.URL,
			Fribb:        a.datasets.FribbURL,
			AnimeLists:   a.datasets.AnimeListsURL,
			AniList:      a.anilist.URL,
			AniListOAuth: a.anilist.OAuthURL(),
		},
		CustomMappings:         s.customMappings,
		DatasetRefreshInterval: s.refreshInterval,
		AniListPollInterval:    aniListPollInterval,
		Clock:                  a.clock,
		UsernameTTL:            usernameTTL,
	}
	for _, edit := range s.configEdits {
		edit(&cfg)
	}
	h, closeFn, err := server.New(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	t.Cleanup(closeFn)
	a.server = httptest.NewServer(h)
	t.Cleanup(a.server.Close)
	a.URL = a.server.URL
	return &a, nil
}

func (a *app) browser() *browser {
	return &browser{t: a.t, app: a, cookies: map[string]*http.Cookie{}}
}

func (a *app) signedIn(sub, name string) *browser {
	a.t.Helper()
	a.provider.SignInAs(sub, name)
	b := a.browser()
	b.signIn()
	return b
}

// firstSet returns the first non-zero of v and inherited, building a default only when both are zero.
func firstSet[T any](v, inherited T, build func() T) T {
	switch {
	case !reflect.ValueOf(&v).Elem().IsZero():
		return v
	case !reflect.ValueOf(&inherited).Elem().IsZero():
		return inherited
	}
	return build()
}

func locationsHolding(t *testing.T, dbURL string, needle []byte) []string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `
		SELECT table_schema, table_name, column_name, data_type = 'bytea'
		FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		AND data_type IN ('text', 'character varying', 'character', 'bytea', 'json', 'jsonb')`)
	if err != nil {
		t.Fatal(err)
	}
	type column struct {
		schema, table, name string
		isBytea             bool
	}
	columns, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (column, error) {
		var c column
		err := r.Scan(&c.schema, &c.table, &c.name, &c.isBytea)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	var hits []string
	for _, c := range columns {
		col := pgx.Identifier{c.name}.Sanitize()
		value := col
		if !c.isBytea {
			value = "convert_to(" + col + "::text, 'UTF8')"
		}
		sql := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s WHERE position($1::bytea in %s) > 0)",
			pgx.Identifier{c.schema, c.table}.Sanitize(), value)
		var found bool
		if err := conn.QueryRow(ctx, sql, needle).Scan(&found); err != nil {
			t.Fatal(err)
		}
		if found {
			hits = append(hits, c.table+"."+c.name)
		}
	}
	return hits
}

func assertNotStored(t *testing.T, dbURL string, secret []byte) {
	t.Helper()
	if hits := locationsHolding(t, dbURL, secret); len(hits) != 0 {
		t.Errorf("secret stored in plaintext at %v", hits)
	}
}

func assertStored(t *testing.T, dbURL string, value []byte) {
	t.Helper()
	if hits := locationsHolding(t, dbURL, value); len(hits) == 0 {
		t.Error("value not found anywhere in the database")
	}
}
