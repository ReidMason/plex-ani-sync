package plexservertest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

type Server struct {
	URL string

	mu        sync.Mutex
	allowed   map[string]bool
	forbidden map[string]bool
	requests  int
	libraries []library
	episodes  map[string][]episode
}

type episode struct {
	GrandparentRatingKey string `json:"grandparentRatingKey"`
	GrandparentTitle     string `json:"grandparentTitle"`
	ParentIndex          int    `json:"parentIndex"`
	Index                int    `json:"index"`
	Title                string `json:"title"`
	ViewCount            int    `json:"viewCount,omitempty"`
	LastViewedAt         int64  `json:"lastViewedAt,omitempty"`
}

// Episode describes one episode in a show library. A zero ViewCount means unwatched and omits the field, as Plex does.
type Episode struct {
	Show         string
	Season       int
	Number       int
	Title        string
	ViewCount    int
	LastViewedAt int64
}

func (s *Server) AddEpisode(libraryKey string, e Episode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.episodes[libraryKey] = append(s.episodes[libraryKey], episode{
		GrandparentRatingKey: "show-" + e.Show, GrandparentTitle: e.Show, ParentIndex: e.Season,
		Index: e.Number, Title: e.Title, ViewCount: e.ViewCount, LastViewedAt: e.LastViewedAt,
	})
}

func (s *Server) allLeaves(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Plex-Token")
	s.mu.Lock()
	allowed := s.allowed[token]
	all := append([]episode{}, s.episodes[r.PathValue("key")]...)
	s.mu.Unlock()
	if !allowed {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	start, _ := strconv.Atoi(r.Header.Get("X-Plex-Container-Start"))
	size, err := strconv.Atoi(r.Header.Get("X-Plex-Container-Size"))
	if err != nil || size <= 0 {
		size = len(all)
	}
	start = min(start, len(all))
	page := all[start:min(start+size, len(all))]
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"MediaContainer": map[string]any{
		"size": len(page), "totalSize": len(all), "offset": start, "Metadata": page,
	}})
}

type library struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// AddLibrary adds a library section of the given Plex type, such as "show" or "movie".
func (s *Server) AddLibrary(key, kind, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.libraries = append(s.libraries, library{Key: key, Type: kind, Title: title})
}

// New starts a fake Plex server. Tokens are rejected with 401 until allowed.
func New(t *testing.T) *Server {
	t.Helper()
	s := &Server{allowed: map[string]bool{}, forbidden: map[string]bool{}, episodes: map[string][]episode{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /library/sections", s.sections)
	mux.HandleFunc("GET /library/sections/{key}/allLeaves", s.allLeaves)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	s.URL = srv.URL
	return s
}

func NewNotPlex(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>hello</body></html>"))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func NewRedirecting(t *testing.T, target string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func NewUnreachable(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	return url
}

func (s *Server) Allow(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowed[token] = true
}

func (s *Server) Forbid(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forbidden[token] = true
}

func (s *Server) Requests() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests
}

func (s *Server) sections(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Plex-Token")
	s.mu.Lock()
	s.requests++
	allowed, forbidden := s.allowed[token], s.forbidden[token]
	libs := append([]library{}, s.libraries...)
	s.mu.Unlock()
	switch {
	case forbidden:
		http.Error(w, "forbidden", http.StatusForbidden)
	case !allowed:
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"MediaContainer": map[string]any{"size": len(libs), "Directory": libs}})
	}
}
