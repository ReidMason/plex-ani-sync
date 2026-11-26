package plexservertest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"testing"
)

type Server struct {
	URL string

	close     func()
	mu        sync.Mutex
	allowed   map[string]bool
	forbidden map[string]bool
	requests  int
	libraries []library
	episodes  map[string][]episode
	guids     map[string]map[string][]string
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

func (s *Server) RemoveLibrary(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.libraries = slices.DeleteFunc(s.libraries, func(l library) bool { return l.Key == key })
	delete(s.episodes, key)
}

// Shutdown stops the server so later requests fail to connect.
func (s *Server) Shutdown() { s.close() }

func (s *Server) AddEpisode(libraryKey string, e Episode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.episodes[libraryKey] = append(s.episodes[libraryKey], episode{
		GrandparentRatingKey: "show-" + e.Show, GrandparentTitle: e.Show, ParentIndex: e.Season,
		Index: e.Number, Title: e.Title, ViewCount: e.ViewCount, LastViewedAt: e.LastViewedAt,
	})
}

func (s *Server) SetShowGUIDs(libraryKey, show string, guids ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.guids[libraryKey] == nil {
		s.guids[libraryKey] = map[string][]string{}
	}
	s.guids[libraryKey]["show-"+show] = guids
}

func (s *Server) shows(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Plex-Token")
	s.mu.Lock()
	allowed := s.allowed[token]
	key := r.PathValue("key")
	var out []map[string]any
	seen := map[string]bool{}
	for _, e := range s.episodes[key] {
		if seen[e.GrandparentRatingKey] {
			continue
		}
		seen[e.GrandparentRatingKey] = true
		var guids []map[string]string
		for _, g := range s.guids[key][e.GrandparentRatingKey] {
			guids = append(guids, map[string]string{"id": g})
		}
		out = append(out, map[string]any{"ratingKey": e.GrandparentRatingKey, "title": e.GrandparentTitle, "Guid": guids})
	}
	s.mu.Unlock()
	if !allowed {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"MediaContainer": map[string]any{"size": len(out), "totalSize": len(out), "Metadata": out}})
}

func (s *Server) allLeaves(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Plex-Token")
	s.mu.Lock()
	allowed, forbidden := s.allowed[token], s.forbidden[token]
	key := r.PathValue("key")
	known := slices.ContainsFunc(s.libraries, func(l library) bool { return l.Key == key })
	all := append([]episode{}, s.episodes[key]...)
	s.mu.Unlock()
	switch {
	case forbidden:
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	case !allowed:
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	case !known:
		http.Error(w, "not found", http.StatusNotFound)
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
	s := &Server{allowed: map[string]bool{}, forbidden: map[string]bool{}, episodes: map[string][]episode{}, guids: map[string]map[string][]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /library/sections", s.sections)
	mux.HandleFunc("GET /library/sections/{key}/allLeaves", s.allLeaves)
	mux.HandleFunc("GET /library/sections/{key}/all", s.shows)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	s.close = srv.Close
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
