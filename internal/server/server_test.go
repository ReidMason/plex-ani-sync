package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ReidMason/plex-ani-sync/internal/config"
	"github.com/ReidMason/plex-ani-sync/internal/oidctest"
	"github.com/ReidMason/plex-ani-sync/internal/server"
)

var (
	adminURL string
	dbSeq    atomic.Int64
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres container:", err)
		os.Exit(1)
	}
	adminURL, err = pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	_ = testcontainers.TerminateContainer(pg)
	os.Exit(code)
}

func newDatabase(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	name := fmt.Sprintf("test_%d", dbSeq.Add(1))
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	return u.String()
}

func testConfig(dbURL, issuer string) config.Config {
	return config.Config{
		DatabaseURL:      dbURL,
		PublicBaseURL:    "https://plex-ani-sync.example.com",
		OIDCIssuerURL:    issuer,
		OIDCClientID:     "plex-ani-sync",
		OIDCClientSecret: "super-secret-value",
	}
}

func startApp(t *testing.T) (*httptest.Server, string, *oidctest.Provider) {
	t.Helper()
	dbURL := newDatabase(t)
	issuer := oidctest.New(t)
	assets := fstest.MapFS{
		"index.html":          {Data: []byte("<!doctype html><title>plex-ani-sync</title>")},
		"assets/index-abc.js": {Data: []byte("console.log(1)")},
	}
	h, closeFn, err := server.New(context.Background(), server.Config{Env: testConfig(dbURL, issuer.URL), Assets: assets})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeFn)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts, dbURL, issuer
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(body)
}

func TestServesSPAShell(t *testing.T) {
	ts, _, _ := startApp(t)

	for _, path := range []string{"/", "/some/client/route"} {
		resp, body := get(t, ts.URL+path)
		if resp.StatusCode != 200 {
			t.Fatalf("%s: status %d", path, resp.StatusCode)
		}
		if body != "<!doctype html><title>plex-ani-sync</title>" {
			t.Errorf("%s: body = %q", path, body)
		}
		if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
			t.Errorf("%s: Cache-Control = %q, want no-cache", path, got)
		}
	}
}

func TestHashedAssetsCacheLong(t *testing.T) {
	ts, _, _ := startApp(t)

	resp, _ := get(t, ts.URL+"/assets/index-abc.js")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if got, want := resp.Header.Get("Cache-Control"), "public, max-age=31536000, immutable"; got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

func TestUnknownAPIPathIsNotTheSPA(t *testing.T) {
	ts, _, _ := startApp(t)

	resp, _ := get(t, ts.URL+"/api/nope")
	if resp.StatusCode != 404 {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

func TestHealthCheckReportsDatabaseReachable(t *testing.T) {
	ts, _, _ := startApp(t)

	resp, _ := get(t, ts.URL+"/api/health")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
}

func TestMigrationsRunOnStartup(t *testing.T) {
	_, dbURL, _ := startApp(t)

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	var applied int
	if err := conn.QueryRow(context.Background(), "SELECT count(*) FROM goose_db_version WHERE is_applied AND version_id > 0").Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied == 0 {
		t.Error("no migrations applied")
	}
}
