package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	MaxBatch = 50

	defaultRetryAfter = time.Minute
	initialBackoff    = 30 * time.Second
	maxBackoff        = 15 * time.Minute
	query             = `query ($ids: [Int], $perPage: Int) { Page(perPage: $perPage) { media(id_in: $ids, type: ANIME) { id episodes title { romaji english } } } }`
)

type Media struct {
	ID int
	// Episodes is nil while AniList doesn't know the total.
	Episodes     *int
	TitleRomaji  string
	TitleEnglish string
}

// ThrottledError means no request was sent because AniList must not be called before Until.
type ThrottledError struct{ Until time.Time }

func (e *ThrottledError) Error() string {
	return "anilist: requests paused until " + e.Until.Format(time.RFC3339)
}

type Options struct {
	URL  string
	HTTP *http.Client
	// Now defaults to time.Now.
	Now func() time.Time
	// RequestsPerMinute spaces requests evenly; zero means no spacing.
	RequestsPerMinute int
}

// Client shares one pause gate across every request, so a 429 holds back all callers.
type Client struct {
	url     string
	http    *http.Client
	now     func() time.Time
	spacing time.Duration

	mu            sync.Mutex
	nextAllowedAt time.Time
	failures      int
}

func New(o Options) *Client {
	c := &Client{url: o.URL, http: o.HTTP, now: o.Now}
	if c.now == nil {
		c.now = time.Now
	}
	if o.RequestsPerMinute > 0 {
		c.spacing = time.Minute / time.Duration(o.RequestsPerMinute)
	}
	return c
}

// reserve claims the next request slot, or reports when AniList may be called again.
func (c *Client) reserve() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if now.Before(c.nextAllowedAt) {
		return &ThrottledError{Until: c.nextAllowedAt}
	}
	c.nextAllowedAt = now.Add(c.spacing)
	return nil
}

func (c *Client) pauseFor(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if until := c.now().Add(d); until.After(c.nextAllowedAt) {
		c.nextAllowedAt = until
	}
}

func (c *Client) recordFailure() {
	c.mu.Lock()
	backoff := min(initialBackoff<<min(c.failures, 10), maxBackoff)
	c.failures++
	c.mu.Unlock()
	c.pauseFor(backoff)
}

func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
}

func parseRetryAfter(h string, now time.Time) time.Duration {
	if secs, err := strconv.Atoi(h); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if at, err := http.ParseTime(h); err == nil {
		return max(at.Sub(now), 0)
	}
	return defaultRetryAfter
}

// Fetch looks up Entries by AniList ID; IDs AniList doesn't know are absent from the result.
func (c *Client) Fetch(ctx context.Context, ids []int) ([]Media, error) {
	if len(ids) > MaxBatch {
		return nil, fmt.Errorf("anilist: %d ids exceeds batch size %d", len(ids), MaxBatch)
	}
	if err := c.reserve(); err != nil {
		return nil, err
	}
	media, err := c.fetchPage(ctx, ids)
	var limited *rateLimitedError
	switch {
	case err == nil:
		c.recordSuccess()
	case errors.As(err, &limited):
		c.pauseFor(limited.retryAfter)
	case ctx.Err() == nil:
		c.recordFailure()
	}
	return media, err
}

type rateLimitedError struct{ retryAfter time.Duration }

func (e *rateLimitedError) Error() string { return "anilist: unexpected status 429" }

func (c *Client) fetchPage(ctx context.Context, ids []int) ([]Media, error) {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": map[string]any{"ids": ids, "perPage": MaxBatch}})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &rateLimitedError{retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), c.now())}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anilist: unexpected status %d", resp.StatusCode)
	}
	var body struct {
		Data struct {
			Page struct {
				Media []struct {
					ID       int  `json:"id"`
					Episodes *int `json:"episodes"`
					Title    struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
					} `json:"title"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("anilist: %w", err)
	}
	if len(body.Errors) > 0 {
		return nil, fmt.Errorf("anilist: %s", body.Errors[0].Message)
	}
	out := make([]Media, 0, len(body.Data.Page.Media))
	for _, m := range body.Data.Page.Media {
		out = append(out, Media{ID: m.ID, Episodes: m.Episodes, TitleRomaji: m.Title.Romaji, TitleEnglish: m.Title.English})
	}
	return out, nil
}
