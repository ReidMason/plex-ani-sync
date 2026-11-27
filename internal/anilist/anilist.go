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
	userListQuery     = `query ($name: String) { MediaListCollection(userName: $name, type: ANIME) { lists { entries { status progress media { id episodes } } } } }`
	query             = `query ($ids: [Int], $perPage: Int) { Page(perPage: $perPage) { media(id_in: $ids, type: ANIME) { id episodes title { romaji english } } } }`
)

type Media struct {
	ID int
	// Episodes is nil while AniList doesn't know the total.
	Episodes     *int
	TitleRomaji  string
	TitleEnglish string
}

var (
	ErrUserNotFound = errors.New("anilist: user not found")
	ErrPrivateList  = errors.New("anilist: list is private")
)

// ListEntry is one item on a User's AniList list.
type ListEntry struct {
	MediaID  int
	Status   string
	Progress int
	// Episodes is nil while AniList doesn't know the total.
	Episodes *int
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
	env, err := c.send(ctx, query, map[string]any{"ids": ids, "perPage": MaxBatch})
	if err != nil {
		return nil, err
	}
	if len(env.Errors) > 0 {
		return nil, fmt.Errorf("anilist: %s", env.Errors[0].Message)
	}
	var data struct {
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
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("anilist: %w", err)
	}
	out := make([]Media, 0, len(data.Page.Media))
	for _, m := range data.Page.Media {
		out = append(out, Media{ID: m.ID, Episodes: m.Episodes, TitleRomaji: m.Title.Romaji, TitleEnglish: m.Title.English})
	}
	return out, nil
}

// UserList reads a public user's anime list; it returns ErrUserNotFound or ErrPrivateList when it can't.
func (c *Client) UserList(ctx context.Context, username string) ([]ListEntry, error) {
	env, err := c.send(ctx, userListQuery, map[string]any{"name": username})
	if err != nil {
		return nil, err
	}
	for _, e := range env.Errors {
		switch e.Status {
		case http.StatusNotFound:
			return nil, ErrUserNotFound
		case http.StatusForbidden:
			return nil, ErrPrivateList
		}
	}
	if len(env.Errors) > 0 {
		return nil, fmt.Errorf("anilist: %s", env.Errors[0].Message)
	}
	var data struct {
		Collection struct {
			Lists []struct {
				Entries []struct {
					Status   string `json:"status"`
					Progress int    `json:"progress"`
					Media    struct {
						ID       int  `json:"id"`
						Episodes *int `json:"episodes"`
					} `json:"media"`
				} `json:"entries"`
			} `json:"lists"`
		} `json:"MediaListCollection"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("anilist: %w", err)
	}
	out := []ListEntry{}
	for _, l := range data.Collection.Lists {
		for _, e := range l.Entries {
			out = append(out, ListEntry{MediaID: e.Media.ID, Status: e.Status, Progress: e.Progress, Episodes: e.Media.Episodes})
		}
	}
	return out, nil
}

type rateLimitedError struct{ retryAfter time.Duration }

func (e *rateLimitedError) Error() string { return "anilist: unexpected status 429" }

type envelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"errors"`
}

// GraphQL errors come back in the envelope, not as an error.
func (c *Client) send(ctx context.Context, q string, vars map[string]any) (*envelope, error) {
	return c.sendAs(ctx, "", q, vars)
}

func (c *Client) sendAs(ctx context.Context, token, q string, vars map[string]any) (*envelope, error) {
	if err := c.reserve(); err != nil {
		return nil, err
	}
	env, err := c.post(ctx, token, q, vars)
	var limited *rateLimitedError
	switch {
	case err == nil, errors.Is(err, ErrUnauthorised):
		c.recordSuccess()
	case errors.As(err, &limited):
		c.pauseFor(limited.retryAfter)
	case ctx.Err() == nil:
		c.recordFailure()
	}
	return env, err
}

func (c *Client) post(ctx context.Context, token, q string, vars map[string]any) (*envelope, error) {
	payload, err := json.Marshal(map[string]any{"query": q, "variables": vars})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &rateLimitedError{retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), c.now())}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorised
	}
	var env envelope
	decodeErr := json.NewDecoder(resp.Body).Decode(&env)
	if resp.StatusCode != http.StatusOK && (decodeErr != nil || len(env.Errors) == 0) {
		return nil, fmt.Errorf("anilist: unexpected status %d", resp.StatusCode)
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("anilist: %w", decodeErr)
	}
	return &env, nil
}
