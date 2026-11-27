package server_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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
)

const usernameTTL = 5 * time.Minute

var testEncryptionKey = bytes.Repeat([]byte{9}, 32)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
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
}

type appOption func(*appSettings)

func withDatabase(dbURL string) appOption { return func(s *appSettings) { s.dbURL = dbURL } }

func withOIDC(p *oidctest.Provider) appOption { return func(s *appSettings) { s.provider = p } }

func withPlexTV(tv *plextest.TV) appOption { return func(s *appSettings) { s.tv = tv } }

func withPlexServer(srv *plexservertest.Server) appOption {
	return func(s *appSettings) { s.plexServer = srv }
}

func withDatasets(h *datasettest.Host) appOption { return func(s *appSettings) { s.datasets = h } }

func withAniList(h *anilisttest.Host) appOption { return func(s *appSettings) { s.anilist = h } }

func withEncryptionKey(key []byte) appOption { return func(s *appSettings) { s.key = key } }

func withCustomMappings(m []mappings.Mapping) appOption {
	return func(s *appSettings) { s.customMappings = m }
}

func withClock(c *fakeClock) appOption { return func(s *appSettings) { s.clock = c } }

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

// restartOf starts against prev's database, key, clock and fakes, unless other options replace them.
func restartOf(prev *app) appOption { return func(s *appSettings) { s.restart = prev } }

func startApp(t *testing.T, opts ...appOption) *app {
	t.Helper()
	var s appSettings
	for _, opt := range opts {
		opt(&s)
	}
	a := s.app
	if prev := s.restart; prev != nil {
		a.dbURL = or(a.dbURL, prev.dbURL)
		if a.key == nil {
			a.key = prev.key
		}
		a.provider = or(a.provider, prev.provider)
		a.tv = or(a.tv, prev.tv)
		a.plexServer = or(a.plexServer, prev.plexServer)
		a.datasets = or(a.datasets, prev.datasets)
		a.anilist = or(a.anilist, prev.anilist)
		a.clock = or(a.clock, prev.clock)
	}
	a.t = t
	a.dbURL = orElse(a.dbURL, func() string { return newDatabase(t) })
	if a.key == nil {
		a.key = testEncryptionKey
	}
	a.provider = orElse(a.provider, func() *oidctest.Provider { return oidctest.New(t) })
	a.tv = orElse(a.tv, func() *plextest.TV { return plextest.New(t) })
	a.plexServer = orElse(a.plexServer, func() *plexservertest.Server { return plexservertest.New(t) })
	a.datasets = orElse(a.datasets, func() *datasettest.Host { return datasettest.New(t) })
	a.anilist = orElse(a.anilist, func() *anilisttest.Host { return anilisttest.New(t) })
	a.clock = orElse(a.clock, newFakeClock)
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
		PlexTVURL:              a.tv.URL,
		FribbURL:               a.datasets.FribbURL,
		AnimeListsURL:          a.datasets.AnimeListsURL,
		CustomMappings:         s.customMappings,
		DatasetRefreshInterval: s.refreshInterval,
		AniListURL:             a.anilist.URL,
		AniListPollInterval:    5 * time.Millisecond,
		Clock:                  a.clock.Now,
		UsernameTTL:            usernameTTL,
	}
	h, closeFn, err := server.New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeFn)
	a.server = httptest.NewServer(h)
	t.Cleanup(a.server.Close)
	a.URL = a.server.URL
	return &a
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

func or[T comparable](v, fallback T) T {
	var zero T
	if v == zero {
		return fallback
	}
	return v
}

func orElse[T comparable](v T, fallback func() T) T {
	var zero T
	if v == zero {
		return fallback()
	}
	return v
}
