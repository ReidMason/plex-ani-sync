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
	// PlexTVURL overrides https://plex.tv, for tests.
	PlexTVURL string
	// FribbURL overrides the Fribb anime-list-full.json location, for tests.
	FribbURL string
	// CustomMappings replaces the embedded Custom Mappings, for tests.
	CustomMappings []mappings.Mapping
	// AnimeListsURL overrides the Anime-Lists anime-list.xml location, for tests.
	AnimeListsURL string
	// Clock and UsernameTTL control the plex.tv username cache; they default to time.Now and 5 minutes.
	Clock       func() time.Time
	UsernameTTL time.Duration
}

// New opens the database, applies migrations and returns the HTTP handler plus a cleanup func.
func New(ctx context.Context, cfg Config) (http.Handler, func(), error) {
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
	plexTV := cfg.PlexTVURL
	if plexTV == "" {
		plexTV = plex.DefaultTVURL
	}
	fribbURL := cfg.FribbURL
	if fribbURL == "" {
		fribbURL = defaultFribbURL
	}
	custom := cfg.CustomMappings
	if custom == nil {
		if custom, err = mappings.Custom(); err != nil {
			pool.Close()
			return nil, nil, err
		}
	}
	animeListsURL := cfg.AnimeListsURL
	if animeListsURL == "" {
		animeListsURL = defaultAnimeListsURL
	}
	if err := loadDatasetIfEmpty(ctx, pool, queries, fribbURL, animeListsURL, custom, logger); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("load mapping dataset: %w", err)
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	ttl := cfg.UsernameTTL
	if ttl == 0 {
		ttl = defaultUsernameTTL
	}
	serverClient := plex.NewServerClient()
	p := &plexLink{usernames: newUsernameCache(clock, ttl), client: plex.NewHTTPClient(plexTV), server: serverClient, sources: map[mediaKind]media.Source{mediaKindPlex: serverClient.Source(clientID)}, clientID: clientID, box: box, queries: queries, pool: pool, logger: logger}
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
	return mux, pool.Close, nil
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
