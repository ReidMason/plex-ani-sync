package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ReidMason/plex-ani-sync/internal/loopbacktest"
	"github.com/ReidMason/plex-ani-sync/internal/server"
)

const templateDB = "app_template"

var (
	adminURL string
	dbSeq    atomic.Int64
)

func TestMain(m *testing.M) {
	loopbacktest.Install()
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
		testcontainers.WithCmdArgs("-c", "synchronous_commit=off", "-c", "full_page_writes=off"),
		testcontainers.WithTmpfs(map[string]string{"/var/lib/postgresql/data": "rw"}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres container:", err)
		os.Exit(1)
	}
	adminURL, err = pg.ConnectionString(ctx, "sslmode=disable")
	if err == nil {
		err = createTemplate(ctx)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		_ = testcontainers.TerminateContainer(pg)
		os.Exit(1)
	}
	code := m.Run()
	_ = testcontainers.TerminateContainer(pg)
	os.Exit(code)
}

func createTemplate(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+templateDB); err != nil {
		return err
	}
	pool, err := db.Open(ctx, databaseURL(templateDB))
	if err != nil {
		return fmt.Errorf("migrate template database: %w", err)
	}
	pool.Close()
	return nil
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
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+name+" TEMPLATE "+templateDB); err != nil {
		t.Fatal(err)
	}
	return databaseURL(name)
}

func databaseURL(name string) string {
	u, err := url.Parse(adminURL)
	if err != nil {
		panic(err)
	}
	u.Path = "/" + name
	return u.String()
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
	a := startApp(t)

	for _, path := range []string{"/", "/some/client/route"} {
		resp, body := get(t, a.URL+path)
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
	a := startApp(t)

	resp, _ := get(t, a.URL+"/assets/index-abc.js")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if got, want := resp.Header.Get("Cache-Control"), "public, max-age=31536000, immutable"; got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

func TestUnknownAPIPathIsNotTheSPA(t *testing.T) {
	a := startApp(t)

	resp, _ := get(t, a.URL+"/api/nope")
	if resp.StatusCode != 404 {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

func TestHealthCheckReportsDatabaseReachable(t *testing.T) {
	a := startApp(t)

	resp, _ := get(t, a.URL+"/api/health")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
}

func TestRefusesToStartWithoutExternalURL(t *testing.T) {
	cases := map[string]func(*server.Config){
		"PlexTVURL":     func(c *server.Config) { c.PlexTVURL = "" },
		"AniListURL":    func(c *server.Config) { c.AniListURL = "" },
		"FribbURL":      func(c *server.Config) { c.FribbURL = "" },
		"AnimeListsURL": func(c *server.Config) { c.AnimeListsURL = "" },
	}
	for field, clear := range cases {
		t.Run(field, func(t *testing.T) {
			_, err := tryStartApp(t, withConfigEdit(clear))

			if err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("err = %v, want error naming %s", err, field)
			}
		})
	}
}
