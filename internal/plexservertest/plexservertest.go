package plexservertest

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Server is a fake Plex media server answering GET /library/sections.
type Server struct {
	URL string

	mu        sync.Mutex
	allowed   map[string]bool
	forbidden map[string]bool
	requests  int
}

// New starts a fake Plex server. Tokens are rejected with 401 until allowed.
func New(t *testing.T) *Server {
	t.Helper()
	s := &Server{allowed: map[string]bool{}, forbidden: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /library/sections", s.sections)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	s.URL = srv.URL
	return s
}

// NewNotPlex starts an HTTP server that is not a Plex server.
func NewNotPlex(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>hello</body></html>"))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// NewRedirecting starts a server that redirects every request to target.
func NewRedirecting(t *testing.T, target string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// NewUnreachable returns the URL of an address nothing is listening on.
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

// Forbid makes the server answer the token with 403.
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
	s.mu.Unlock()
	switch {
	case forbidden:
		http.Error(w, "forbidden", http.StatusForbidden)
	case !allowed:
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	default:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"Directory":[]}}`))
	}
}
