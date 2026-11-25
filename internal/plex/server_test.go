package plex_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/internal/plexservertest"
)

func TestServerClientCheckAccessOutcomes(t *testing.T) {
	srv := plexservertest.New(t)
	srv.Allow("good")
	srv.Forbid("banned")
	cases := []struct {
		name  string
		url   string
		token string
		want  error
	}{
		{"ok", srv.URL, "good", nil},
		{"rejected token", srv.URL, "bad", plex.ErrNoAccess},
		{"forbidden token", srv.URL, "banned", plex.ErrNoAccess},
		{"not plex", plexservertest.NewNotPlex(t), "good", plex.ErrNotPlexServer},
		{"redirect not followed", plexservertest.NewRedirecting(t, srv.URL), "good", plex.ErrNotPlexServer},
		{"unreachable", plexservertest.NewUnreachable(t), "good", plex.ErrUnreachable},
	}
	client := plex.NewServerClient()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := client.CheckAccess(context.Background(), "client-1", c.url, c.token)
			if !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestNormalizeServerURL(t *testing.T) {
	cases := map[string]string{
		"192.168.1.5:32400":          "http://192.168.1.5:32400",
		"plex.example.com":           "http://plex.example.com",
		" https://plex.example.com/": "https://plex.example.com",
		"HTTP://10.0.0.2:32400":      "http://10.0.0.2:32400",
	}
	for in, want := range cases {
		got, err := plex.NormalizeServerURL(in)
		if err != nil || got != want {
			t.Errorf("%q -> %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "ftp://x.com", "file:///etc/passwd", "http://", "javascript:alert(1)", "http://a b"} {
		if _, err := plex.NormalizeServerURL(in); !errors.Is(err, plex.ErrInvalidURL) {
			t.Errorf("%q err = %v, want ErrInvalidURL", in, err)
		}
	}
}
