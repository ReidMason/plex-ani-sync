package plex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/media"
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

// Source adapts the client to media.Source for one install-wide client identifier.
type Source struct {
	client   *HTTPServerClient
	clientID string
}

func (c *HTTPServerClient) Source(clientID string) *Source {
	return &Source{client: c, clientID: clientID}
}

func (s *Source) Libraries(ctx context.Context, conn media.Connection) ([]media.Library, error) {
	var body struct {
		MediaContainer struct {
			Directory []struct {
				Key   string `json:"key"`
				Type  string `json:"type"`
				Title string `json:"title"`
			} `json:"Directory"`
		} `json:"MediaContainer"`
	}
	if err := s.getJSON(ctx, conn, "/library/sections", nil, &body); err != nil {
		return nil, err
	}
	var out []media.Library
	for _, d := range body.MediaContainer.Directory {
		if d.Type == "show" {
			out = append(out, media.Library{Key: d.Key, Title: d.Title})
		}
	}
	return out, nil
}

const leavesPageSize = 200

type leaf struct {
	GrandparentRatingKey string `json:"grandparentRatingKey"`
	GrandparentTitle     string `json:"grandparentTitle"`
	ParentIndex          int    `json:"parentIndex"`
	Index                int    `json:"index"`
	Title                string `json:"title"`
	ViewCount            int    `json:"viewCount"`
	LastViewedAt         int64  `json:"lastViewedAt"`
}

func (s *Source) Shows(ctx context.Context, conn media.Connection, libraryKey string) ([]media.Show, error) {
	var leaves []leaf
	for start := 0; ; start += leavesPageSize {
		var body struct {
			MediaContainer struct {
				TotalSize int    `json:"totalSize"`
				Metadata  []leaf `json:"Metadata"`
			} `json:"MediaContainer"`
		}
		page := http.Header{
			"X-Plex-Container-Start": {strconv.Itoa(start)},
			"X-Plex-Container-Size":  {strconv.Itoa(leavesPageSize)},
		}
		if err := s.getJSON(ctx, conn, "/library/sections/"+url.PathEscape(libraryKey)+"/allLeaves", page, &body); err != nil {
			return nil, err
		}
		leaves = append(leaves, body.MediaContainer.Metadata...)
		if len(body.MediaContainer.Metadata) == 0 || start+len(body.MediaContainer.Metadata) >= body.MediaContainer.TotalSize {
			break
		}
	}
	return groupLeaves(leaves), nil
}

func (s *Source) getJSON(ctx context.Context, conn media.Connection, path string, header http.Header, into any) error {
	serverURL, err := NormalizeServerURL(conn.URL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+path, nil)
	if err != nil {
		return ErrInvalidURL
	}
	setPlexHeaders(req, s.clientID, conn.Token)
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := s.client.http.Do(req)
	if err != nil {
		return media.ErrUnreachable
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return media.ErrNoAccess
	case resp.StatusCode != http.StatusOK:
		return ErrNotPlexServer
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		return ErrNotPlexServer
	}
	return nil
}

func groupLeaves(leaves []leaf) []media.Show {
	var shows []media.Show
	showAt := map[string]int{}
	for _, l := range leaves {
		i, ok := showAt[l.GrandparentRatingKey]
		if !ok {
			i = len(shows)
			showAt[l.GrandparentRatingKey] = i
			shows = append(shows, media.Show{Title: l.GrandparentTitle})
		}
		ep := media.Episode{Number: l.Index, Title: l.Title, Watched: l.ViewCount > 0}
		if ep.Watched && l.LastViewedAt > 0 {
			t := time.Unix(l.LastViewedAt, 0).UTC()
			ep.LastWatchedAt = &t
		}
		show := &shows[i]
		j := slices.IndexFunc(show.Seasons, func(s media.Season) bool { return s.Number == l.ParentIndex })
		if j < 0 {
			show.Seasons = append(show.Seasons, media.Season{Number: l.ParentIndex})
			j = len(show.Seasons) - 1
		}
		show.Seasons[j].Episodes = append(show.Seasons[j].Episodes, ep)
	}
	for i := range shows {
		slices.SortFunc(shows[i].Seasons, func(a, b media.Season) int { return a.Number - b.Number })
		for j := range shows[i].Seasons {
			slices.SortFunc(shows[i].Seasons[j].Episodes, func(a, b media.Episode) int { return a.Number - b.Number })
		}
	}
	return shows
}
