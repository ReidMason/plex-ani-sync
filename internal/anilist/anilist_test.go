package anilist_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/anilist"
)

func serve(t *testing.T, status int, fixture string) *anilist.Client {
	t.Helper()
	body, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			http.Error(w, "no", status)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return anilist.New(anilist.Options{URL: srv.URL, HTTP: srv.Client()})
}

func TestFetchParsesTitlesAndEpisodeCounts(t *testing.T) {
	t.Parallel()
	got, err := serve(t, http.StatusOK, "media.json").Fetch(context.Background(), []int{154587, 999001})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0].ID != 154587 || got[0].Episodes == nil || *got[0].Episodes != 28 ||
		got[0].TitleRomaji != "Sousou no Frieren" || got[0].TitleEnglish != "Frieren: Beyond Journey's End" {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].Episodes != nil || got[1].TitleEnglish != "" {
		t.Errorf("second = %+v", got[1])
	}
}

func TestFetchReportsRateLimitAndServerErrors(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError} {
		if _, err := serve(t, status, "media.json").Fetch(context.Background(), []int{1}); err == nil {
			t.Errorf("status %d: want error", status)
		}
	}
}

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func newPaced(t *testing.T, rpm int, respond http.HandlerFunc) (*anilist.Client, *fakeClock) {
	t.Helper()
	srv := httptest.NewServer(respond)
	t.Cleanup(srv.Close)
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return anilist.New(anilist.Options{URL: srv.URL, HTTP: srv.Client(), Now: clock.Now, RequestsPerMinute: rpm}), clock
}

func okResponse(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(`{"data":{"Page":{"media":[]}}}`))
}

func TestRequestsAreSpacedByTheConfiguredRate(t *testing.T) {
	t.Parallel()
	c, clock := newPaced(t, 30, okResponse)
	if _, err := c.Fetch(context.Background(), []int{1}); err != nil {
		t.Fatal(err)
	}

	_, early := c.Fetch(context.Background(), []int{1})
	clock.now = clock.now.Add(2 * time.Second)
	_, onTime := c.Fetch(context.Background(), []int{1})

	var throttled *anilist.ThrottledError
	if !errors.As(early, &throttled) || onTime != nil {
		t.Errorf("early = %v, onTime = %v", early, onTime)
	}
}

func TestRateLimitWithoutRetryAfterPausesForAMinute(t *testing.T) {
	t.Parallel()
	c, clock := newPaced(t, 0, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) })
	_, _ = c.Fetch(context.Background(), []int{1})

	clock.now = clock.now.Add(59 * time.Second)
	_, during := c.Fetch(context.Background(), []int{1})
	clock.now = clock.now.Add(2 * time.Second)
	_, after := c.Fetch(context.Background(), []int{1})

	var throttled *anilist.ThrottledError
	if !errors.As(during, &throttled) || errors.As(after, &throttled) {
		t.Errorf("during = %v, after = %v", during, after)
	}
}

func TestFailureBackoffIsCappedAndResetsAfterSuccess(t *testing.T) {
	t.Parallel()
	fail := true
	c, clock := newPaced(t, 0, func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		okResponse(w, r)
	})
	for range 12 {
		_, _ = c.Fetch(context.Background(), []int{1})
		clock.now = clock.now.Add(15*time.Minute + time.Second)
	}
	_, _ = c.Fetch(context.Background(), []int{1})
	clock.now = clock.now.Add(15*time.Minute - time.Second)
	_, capped := c.Fetch(context.Background(), []int{1})

	fail = false
	clock.now = clock.now.Add(2 * time.Second)
	if _, err := c.Fetch(context.Background(), []int{1}); err != nil {
		t.Fatalf("fetch after the cap: %v", err)
	}
	fail = true
	_, _ = c.Fetch(context.Background(), []int{1})
	fail = false
	clock.now = clock.now.Add(31 * time.Second)
	_, reset := c.Fetch(context.Background(), []int{1})

	var throttled *anilist.ThrottledError
	if !errors.As(capped, &throttled) || reset != nil {
		t.Errorf("capped = %v, reset = %v", capped, reset)
	}
}

func TestRetryAfterHeaderIsHonoured(t *testing.T) {
	t.Parallel()
	c, clock := newPaced(t, 0, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", strconv.Itoa(5))
		w.WriteHeader(http.StatusTooManyRequests)
	})
	_, _ = c.Fetch(context.Background(), []int{1})

	clock.now = clock.now.Add(6 * time.Second)
	_, err := c.Fetch(context.Background(), []int{1})

	var throttled *anilist.ThrottledError
	if errors.As(err, &throttled) {
		t.Errorf("still throttled after Retry-After: %v", err)
	}
}

func TestCancelledRequestDoesNotBackOff(t *testing.T) {
	t.Parallel()
	c, _ := newPaced(t, 0, okResponse)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = c.Fetch(ctx, []int{1})

	_, err := c.Fetch(context.Background(), []int{1})

	if err != nil {
		t.Errorf("fetch after a cancelled one: %v", err)
	}
}
