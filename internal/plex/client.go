package plex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultTVURL = "https://plex.tv"
	productName  = "Plex Ani Sync"
)

var ErrUnauthorized = errors.New("plex token rejected")

type PIN struct {
	ID   int64
	Code string
}

// Client is the subset of plex.tv this app uses: the PIN sign-in flow and the username lookup.
type Client interface {
	CreatePIN(ctx context.Context, clientID string) (PIN, error)
	// CheckPIN returns the account token once the PIN is approved, or "" while it is pending.
	CheckPIN(ctx context.Context, clientID string, pin PIN) (string, error)
	Username(ctx context.Context, clientID, token string) (string, error)
}

type HTTPClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 10 * time.Second}}
}

// AuthURL is the plex.tv page where the user approves the PIN.
func AuthURL(clientID string, pin PIN) string {
	q := url.Values{}
	q.Set("clientID", clientID)
	q.Set("code", pin.Code)
	q.Set("context[device][product]", productName)
	return "https://app.plex.tv/auth#?" + q.Encode()
}

func (c *HTTPClient) CreatePIN(ctx context.Context, clientID string) (PIN, error) {
	var body struct {
		ID   int64  `json:"id"`
		Code string `json:"code"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v2/pins?strong=true", clientID, "", &body); err != nil {
		return PIN{}, err
	}
	return PIN{ID: body.ID, Code: body.Code}, nil
}

func (c *HTTPClient) CheckPIN(ctx context.Context, clientID string, pin PIN) (string, error) {
	var body struct {
		AuthToken *string `json:"authToken"`
	}
	path := "/api/v2/pins/" + strconv.FormatInt(pin.ID, 10) + "?code=" + url.QueryEscape(pin.Code)
	if err := c.do(ctx, http.MethodGet, path, clientID, "", &body); err != nil {
		return "", err
	}
	if body.AuthToken == nil {
		return "", nil
	}
	return *body.AuthToken, nil
}

func (c *HTTPClient) Username(ctx context.Context, clientID, token string) (string, error) {
	var body struct {
		Username string `json:"username"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v2/user", clientID, token, &body); err != nil {
		return "", err
	}
	return body.Username, nil
}

func (c *HTTPClient) do(ctx context.Context, method, path, clientID, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", clientID)
	req.Header.Set("X-Plex-Product", productName)
	if token != "" {
		req.Header.Set("X-Plex-Token", token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return fmt.Errorf("plex.tv %s %s: status %d", method, strings.SplitN(path, "?", 2)[0], resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
