package loopbacktest_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/loopbacktest"
)

func TestMain(m *testing.M) {
	loopbacktest.Install()
	os.Exit(m.Run())
}

func TestRejectsNonLoopbackHostByName(t *testing.T) {
	t.Parallel()
	_, err := http.Get("http://example.invalid/")

	if !errors.Is(err, loopbacktest.ErrNonLoopback) || !strings.Contains(err.Error(), "example.invalid") {
		t.Fatalf("err = %v, want rejection naming example.invalid", err)
	}
}

func TestRejectsNonLoopbackIP(t *testing.T) {
	t.Parallel()
	_, err := http.Get("http://192.0.2.1/")

	if !errors.Is(err, loopbacktest.ErrNonLoopback) || !strings.Contains(err.Error(), "192.0.2.1") {
		t.Fatalf("err = %v, want rejection naming 192.0.2.1", err)
	}
}

func TestAllowsLoopback(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(srv.Close)

	for _, url := range []string{srv.URL, strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)} {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET %s: %v", url, err)
		}
		resp.Body.Close()
	}
}
