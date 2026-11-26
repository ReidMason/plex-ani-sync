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

// FribbEntry is one row of Fribb's anime-list-full.json; TMDBID is served as `{"tv": id}`.
type FribbEntry struct {
	AniDBID   int
	AniListID int
	TVDBID    int
	TMDBID    int
	Season    *Season
	Offset    *Season
	Type      string
}

func (e FribbEntry) MarshalJSON() ([]byte, error) {
	wire := struct {
		AniDBID   int     `json:"anidb_id"`
		AniListID int     `json:"anilist_id,omitempty"`
		TVDBID    int     `json:"tvdb_id,omitempty"`
		TMDB      *tmdbID `json:"themoviedb_id,omitempty"`
		Season    *Season `json:"season,omitempty"`
		Offset    *Season `json:"episode_offset,omitempty"`
		Type      string  `json:"type"`
	}{e.AniDBID, e.AniListID, e.TVDBID, nil, e.Season, e.Offset, e.Type}
	if e.TMDBID > 0 {
		wire.TMDB = &tmdbID{TV: e.TMDBID}
	}
	return json.Marshal(wire)
}

type tmdbID struct {
	TV int `json:"tv"`
}

type Host struct {
	FribbURL      string
	AnimeListsURL string

	mu          sync.Mutex
	fribb       []FribbEntry
	lists       string
	failed      bool
	listsFailed bool
	hits        int
}

func New(t *testing.T) *Host {
	t.Helper()
	h := &Host{lists: "<anime-list></anime-list>"}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /fribb.json", func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		entries, failed := h.fribb, h.failed
		h.hits++
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
		lists, failed := h.lists, h.failed || h.listsFailed
		h.hits++
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

func (h *Host) SetAnimeLists(xml string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lists = xml
}

func (h *Host) Requests() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hits
}

func (h *Host) FailAnimeLists() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.listsFailed = true
}

func (h *Host) Fail() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failed = true
}
