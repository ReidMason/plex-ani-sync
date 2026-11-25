package plextest

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

type pin struct {
	id    int64
	code  string
	token string
}

type TV struct {
	URL string

	mu        sync.Mutex
	pins      map[int64]*pin
	usernames map[string]string
	clientIDs []string
	nextID    int64
	down      bool
}

func New(t *testing.T) *TV {
	t.Helper()
	tv := &TV{pins: map[int64]*pin{}, usernames: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/pins", tv.createPIN)
	mux.HandleFunc("GET /api/v2/pins/{id}", tv.getPIN)
	mux.HandleFunc("GET /api/v2/user", tv.user)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tv.mu.Lock()
		down := tv.down
		if id := r.Header.Get("X-Plex-Client-Identifier"); id != "" {
			tv.clientIDs = append(tv.clientIDs, id)
		}
		tv.mu.Unlock()
		if down {
			http.Error(w, "plex.tv is down", http.StatusServiceUnavailable)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	tv.URL = srv.URL
	return tv
}

// Approve simulates the user approving the PIN at plex.tv, issuing token for username.
func (tv *TV) Approve(code, token, username string) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	for _, p := range tv.pins {
		if p.code == code {
			p.token = token
		}
	}
	tv.usernames[token] = username
}

func (tv *TV) SetUsername(token, username string) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	tv.usernames[token] = username
}

func (tv *TV) RevokeToken(token string) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	delete(tv.usernames, token)
}

func (tv *TV) SetDown(down bool) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	tv.down = down
}

func (tv *TV) ClientIdentifiers() []string {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, id := range tv.clientIDs {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (tv *TV) createPIN(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Plex-Client-Identifier") == "" {
		http.Error(w, "missing client identifier", http.StatusBadRequest)
		return
	}
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	tv.mu.Lock()
	tv.nextID++
	p := &pin{id: tv.nextID, code: hex.EncodeToString(buf)}
	tv.pins[p.id] = p
	tv.mu.Unlock()
	writeJSON(w, map[string]any{"id": p.id, "code": p.code, "authToken": nil})
}

func (tv *TV) getPIN(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	tv.mu.Lock()
	p, ok := tv.pins[id]
	var token string
	if ok {
		token = p.token
	}
	tv.mu.Unlock()
	if !ok || p.code != r.URL.Query().Get("code") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var authToken any
	if token != "" {
		authToken = token
	}
	writeJSON(w, map[string]any{"id": p.id, "code": p.code, "authToken": authToken})
}

func (tv *TV) user(w http.ResponseWriter, r *http.Request) {
	tv.mu.Lock()
	username, ok := tv.usernames[r.Header.Get("X-Plex-Token")]
	tv.mu.Unlock()
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]any{"username": username})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
