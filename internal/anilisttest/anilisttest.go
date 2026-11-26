package anilisttest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

type Title struct {
	Romaji  string `json:"romaji,omitempty"`
	English string `json:"english,omitempty"`
}

type Media struct {
	ID       int   `json:"id"`
	Episodes *int  `json:"episodes"`
	Title    Title `json:"title"`
}

type Host struct {
	URL string

	mu       sync.Mutex
	media    map[int]Media
	status   int
	retry    string
	requests int
}

func New(t *testing.T) *Host {
	t.Helper()
	h := &Host{media: map[int]Media{}}
	srv := httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(srv.Close)
	h.URL = srv.URL
	return h
}

func (h *Host) Set(m Media) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.media[m.ID] = m
}

func (h *Host) FailWith(status int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = status
	h.retry = ""
}

// FailWithRetryAfter makes every request fail with status and a Retry-After header of seconds.
func (h *Host) FailWithRetryAfter(status, seconds int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = status
	h.retry = strconv.Itoa(seconds)
}

func (h *Host) Requests() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.requests
}

func (h *Host) serve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Variables struct {
			IDs []int `json:"ids"`
		} `json:"variables"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	h.mu.Lock()
	h.requests++
	status, retry := h.status, h.retry
	found := []Media{}
	for _, id := range body.Variables.IDs {
		if m, ok := h.media[id]; ok {
			found = append(found, m)
		}
	}
	h.mu.Unlock()
	if status != 0 {
		if retry != "" {
			w.Header().Set("Retry-After", retry)
		}
		http.Error(w, "failing", status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"Page": map[string]any{"media": found}}})
}
