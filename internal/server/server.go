package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
	"github.com/ReidMason/plex-ani-sync/internal/cache"
	"github.com/ReidMason/plex-ani-sync/internal/config"
	"github.com/ReidMason/plex-ani-sync/internal/db"
	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
	"github.com/ReidMason/plex-ani-sync/internal/mappings"
	"github.com/ReidMason/plex-ani-sync/internal/media"
	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/internal/secretbox"
)

type Config struct {
	Env    config.Config
	Assets fs.FS
	Logger *slog.Logger
	// PlexTVURL, FribbURL, AnimeListsURL and AniListURL locate the external services; all are required.
	PlexTVURL     string
	FribbURL      string
	AnimeListsURL string
	AniListURL    string
	// CustomMappings replaces the embedded Custom Mappings, for tests.
	CustomMappings []mappings.Mapping
	// DatasetRefreshInterval is how often the mapping datasets are re-downloaded; it defaults to 24 hours.
	DatasetRefreshInterval time.Duration
	// AniListPollInterval is how often the AniList worker rechecks its queue; it defaults to 1 second.
	AniListPollInterval time.Duration
	// Cache stores shows pulled from media sources; it defaults to an in-memory cache.
	Cache cache.Cache
	// ShowsTTL is how long a library's shows are cached; it defaults to 5 minutes.
	ShowsTTL time.Duration
	// Clock drives sessions, caches and dataset refresh; it defaults to the real clock.
	Clock Clock
	// UsernameTTL is how long a plex.tv username is cached; it defaults to 5 minutes.
	UsernameTTL time.Duration
}

// New opens the database, applies migrations and returns the HTTP handler plus a cleanup func.
func New(ctx context.Context, cfg Config) (http.Handler, func(), error) {
	if err := cfg.requireExternalURLs(); err != nil {
		return nil, nil, err
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	pool, err := db.Open(ctx, cfg.Env.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	discoverCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(discoverCtx, cfg.Env.OIDCIssuerURL)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("OIDC discovery failed for OIDC_ISSUER_URL %q: %w", cfg.Env.OIDCIssuerURL, err)
	}
	clock := cfg.Clock
	if clock == nil {
		clock = realClock{}
	}
	queries := dbgen.New(pool)
	a := &auth{
		oauth: &oauth2.Config{
			ClientID:     cfg.Env.OIDCClientID,
			ClientSecret: cfg.Env.OIDCClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.Env.RedirectURI(),
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
		},
		issuer:   cfg.Env.OIDCIssuerURL,
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.Env.OIDCClientID}),
		queries:  queries,
		now:      clock.Now,
		logger:   logger,
	}
	box, err := secretbox.New(cfg.Env.EncryptionKey)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	clientID, err := loadClientIdentifier(ctx, queries)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("load plex client identifier: %w", err)
	}
	custom := cfg.CustomMappings
	if custom == nil {
		if custom, err = mappings.Custom(); err != nil {
			pool.Close()
			return nil, nil, err
		}
	}
	datasets := datasetSync{pool: pool, queries: queries, fribbURL: cfg.FribbURL, animeListsURL: cfg.AnimeListsURL, custom: custom, logger: logger}
	if err := datasets.loadAtStartup(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("load mapping dataset: %w", err)
	}
	refreshEvery := cfg.DatasetRefreshInterval
	if refreshEvery == 0 {
		refreshEvery = defaultDatasetRefreshInterval
	}
	refreshCtx, cancelRefresh := context.WithCancel(ctx)
	stopTicks := clock.Every(refreshEvery, func() { datasets.refreshOnTick(refreshCtx) })
	ttl := cfg.UsernameTTL
	if ttl == 0 {
		ttl = defaultUsernameTTL
	}
	aniListRate := cfg.Env.AniListRequestsPerMinute
	if aniListRate == 0 {
		aniListRate = config.DefaultAniListRequestsPerMinute
	}
	aniListPoll := cfg.AniListPollInterval
	if aniListPoll == 0 {
		aniListPoll = time.Second
	}
	aniListWorker := newAniListCache(anilist.New(anilist.Options{URL: cfg.AniListURL, HTTP: http.DefaultClient, Now: clock.Now, RequestsPerMinute: aniListRate}), queries, clock, defaultAniListMetadataTTL, aniListPoll, logger)
	aniListCtx, stopAniList := context.WithCancel(ctx)
	aniListDone := make(chan struct{})
	go func() {
		defer close(aniListDone)
		aniListWorker.run(aniListCtx)
	}()
	serverClient := plex.NewServerClient()
	store := cfg.Cache
	if store == nil {
		store = cache.NewMemory(clock.Now)
	}
	showsTTL := cfg.ShowsTTL
	if showsTTL == 0 {
		showsTTL = defaultShowsTTL
	}
	p := &plexLink{shows: &showsCache{store: store, ttl: showsTTL, logger: logger}, usernames: newUsernameCache(clock.Now, ttl), client: plex.NewHTTPClient(cfg.PlexTVURL), server: serverClient, sources: map[mediaKind]media.Source{mediaKindPlex: serverClient.Source(clientID)}, clientID: clientID, box: box, queries: queries, pool: pool, logger: logger,
		anilist: aniListWorker}
	mux := http.NewServeMux()
	a.register(mux)
	p.register(mux, a)
	p.registerMediaSources(mux, a)
	p.registerAnime(mux, a)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := queries.Ping(r.Context()); err != nil {
			logger.Error("health check failed", "error", err)
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", staticHandler(cfg.Assets))
	return mux, func() {
		cancelRefresh()
		stopTicks()
		stopAniList()
		<-aniListDone
		pool.Close()
	}, nil
}

func (c Config) requireExternalURLs() error {
	var missing []string
	for _, u := range []struct{ name, value string }{
		{"PlexTVURL", c.PlexTVURL},
		{"FribbURL", c.FribbURL},
		{"AnimeListsURL", c.AnimeListsURL},
		{"AniListURL", c.AniListURL},
	} {
		if u.value == "" {
			missing = append(missing, u.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("server config missing required %s", strings.Join(missing, ", "))
	}
	return nil
}

func staticHandler(assets fs.FS) http.Handler {
	hashed := http.FileServerFS(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isServerPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			hashed.ServeHTTP(w, r)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if info, err := fs.Stat(assets, name); err != nil || info.IsDir() {
			name = "index.html"
		}
		data, err := fs.ReadFile(assets, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		sum := sha256.Sum256(data)
		w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:16])+`"`)
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	})
}

func isServerPath(p string) bool {
	for _, prefix := range []string{"/api", "/auth"} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}
