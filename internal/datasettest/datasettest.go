package datasettest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type Season struct {
	TVDB int `json:"tvdb"`
}

// FribbEntry is one row of Fribb's anime-list-full.json.
type FribbEntry struct {
	AniDBID   int     `json:"anidb_id"`
	AniListID int     `json:"anilist_id,omitempty"`
	TVDBID    int     `json:"thetvdb_id,omitempty"`
	Season    *Season `json:"season,omitempty"`
	Offset    *Season `json:"episode_offset,omitempty"`
	Type      string  `json:"type"`
}

type Host struct {
	FribbURL      string
	AnimeListsURL string

	mu     sync.Mutex
	fribb  []FribbEntry
	lists  string
	failed bool
}

// New starts a fake dataset host serving an empty Fribb list.
func New(t *testing.T) *Host {
	t.Helper()
	h := &Host{lists: "<anime-list></anime-list>"}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /fribb.json", func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		entries, failed := h.fribb, h.failed
		h.mu.Unlock()
		if failed {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entries)
	})
	mux.HandleFunc("GET /anime-list.xml", func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		lists, failed := h.lists, h.failed
		h.mu.Unlock()
		if failed {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(lists))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	h.FribbURL = srv.URL + "/fribb.json"
	h.AnimeListsURL = srv.URL + "/anime-list.xml"
	return h
}

func (h *Host) SetFribb(entries ...FribbEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fribb = entries
}

// SetAnimeLists serves the given Anime-Lists anime-list.xml document.
func (h *Host) SetAnimeLists(xml string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lists = xml
}

// Fail makes the dataset endpoints respond with a server error.
func (h *Host) Fail() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failed = true
}
