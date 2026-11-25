package plex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidURL    = errors.New("invalid plex server url")
	ErrUnreachable   = errors.New("plex server unreachable")
	ErrNotPlexServer = errors.New("not a plex server")
	ErrNoAccess      = errors.New("plex account has no access to server")
)

type ServerClient interface {
	// CheckAccess returns ErrUnreachable, ErrNotPlexServer or ErrNoAccess when the server cannot be used with token.
	CheckAccess(ctx context.Context, clientID, serverURL, token string) error
}

type HTTPServerClient struct {
	http *http.Client
}

func NewServerClient() *HTTPServerClient {
	return &HTTPServerClient{http: &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// NormalizeServerURL accepts a bare host or an http/https URL and returns it without a trailing slash.
func NormalizeServerURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrInvalidURL
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return "", ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", ErrInvalidURL
	}
	return u.Scheme + "://" + u.Host + strings.TrimRight(u.Path, "/"), nil
}

func (c *HTTPServerClient) CheckAccess(ctx context.Context, clientID, serverURL, token string) error {
	serverURL, err := NormalizeServerURL(serverURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/library/sections", nil)
	if err != nil {
		return ErrInvalidURL
	}
	setPlexHeaders(req, clientID, token)
	resp, err := c.http.Do(req)
	if err != nil {
		return ErrUnreachable
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return ErrNoAccess
	case resp.StatusCode != http.StatusOK:
		return ErrNotPlexServer
	}
	var body struct {
		MediaContainer *json.RawMessage `json:"MediaContainer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.MediaContainer == nil {
		return ErrNotPlexServer
	}
	return nil
}
