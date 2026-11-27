package anilisttest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const (
	ClientID     = "anilist-client"
	ClientSecret = "anilist-secret"
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

type ListEntry struct {
	MediaID  int
	Status   string
	Progress int
	Episodes *int
}

type Host struct {
	URL string

	mu       sync.Mutex
	media    map[int]Media
	lists    map[string][]ListEntry
	private  map[string]bool
	status   int
	retry    string
	requests int
	writes   int
	hold     chan struct{}
	arrived  chan struct{}

	approver     string
	deny         bool
	failExchange bool
	rejectTokens bool
	codes        map[string]string
	tokens       map[string]string
	issued       []string
}

func New(t *testing.T) *Host {
	t.Helper()
	h := &Host{media: map[int]Media{}, lists: map[string][]ListEntry{}, private: map[string]bool{}, codes: map[string]string{}, tokens: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth/authorize", h.authorize)
	mux.HandleFunc("POST /oauth/token", h.token)
	mux.HandleFunc("/", h.serve)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	h.URL = srv.URL
	return h
}

// OAuthURL is the base URL of the fake's OAuth endpoints.
func (h *Host) OAuthURL() string { return h.URL + "/oauth" }

// ApproveAs makes the AniList account that approves authorisation requests username.
func (h *Host) ApproveAs(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.approver = username
	h.deny = false
}

// DenyAuthorisation makes the User deny authorisation requests.
func (h *Host) DenyAuthorisation() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deny = true
}

// FailTokenExchange makes the token endpoint reject every code.
func (h *Host) FailTokenExchange() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failExchange = true
}

// RejectTokens makes every authorised request fail with 401, as after a revoked or rotated client.
func (h *Host) RejectTokens() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rejectTokens = true
}

// IssuedTokens lists every access token the fake has handed out.
func (h *Host) IssuedTokens() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.issued...)
}

func (h *Host) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirect, err := url.Parse(q.Get("redirect_uri"))
	if q.Get("client_id") != ClientID || q.Get("response_type") != "code" || err != nil || redirect.Host == "" {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	deny, approver := h.deny, h.approver
	var code string
	if !deny {
		code = fmt.Sprintf("code-%d", len(h.codes)+1)
		h.codes[code] = approver
	}
	h.mu.Unlock()
	out := redirect.Query()
	out.Set("state", q.Get("state"))
	if deny {
		out.Set("error", "access_denied")
	} else {
		out.Set("code", code)
	}
	redirect.RawQuery = out.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (h *Host) token(w http.ResponseWriter, r *http.Request) {
	var body struct {
		GrantType    string `json:"grant_type"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
		Code         string `json:"code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	h.mu.Lock()
	defer h.mu.Unlock()
	account, known := h.codes[body.Code]
	delete(h.codes, body.Code)
	if h.failExchange || !known || body.GrantType != "authorization_code" || body.ClientID != ClientID || body.ClientSecret != ClientSecret || body.RedirectURI == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
		return
	}
	token := fmt.Sprintf("anilist-access-token-%d", len(h.issued)+1)
	h.tokens[token] = account
	h.issued = append(h.issued, token)
	_ = json.NewEncoder(w).Encode(map[string]any{"token_type": "Bearer", "expires_in": 31536000, "access_token": token})
}

func (h *Host) serveViewer(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	h.mu.Lock()
	h.requests++
	name, ok := h.tokens[token]
	rejected := h.rejectTokens
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if !ok || rejected {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": "Invalid token", "status": 401}}, "data": map[string]any{"Viewer": nil}})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"Viewer": map[string]any{"name": name}}})
}

func (h *Host) Set(m Media) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.media[m.ID] = m
}

// SetUserList makes username a public user with the given list.
func (h *Host) SetUserList(username string, entries ...ListEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lists[username] = entries
	delete(h.private, username)
}

// SetPrivateUser makes username a user whose list can't be read.
func (h *Host) SetPrivateUser(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.private[username] = true
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

// HoldUserLists makes user list requests block once they arrive, until release is called; arrived receives one value per held request.
func (h *Host) HoldUserLists() (arrived <-chan struct{}, release func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hold = make(chan struct{})
	h.arrived = make(chan struct{}, 16)
	var once sync.Once
	hold := h.hold
	return h.arrived, func() { once.Do(func() { close(hold) }) }
}

// Writes counts requests that tried to change data on AniList.
func (h *Host) Writes() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.writes
}

func (h *Host) Requests() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.requests
}

func (h *Host) serve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query     string `json:"query"`
		Variables struct {
			IDs  []int   `json:"ids"`
			Name *string `json:"name"`
		} `json:"variables"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.Contains(body.Query, "Viewer") {
		h.serveViewer(w, r)
		return
	}
	h.mu.Lock()
	h.requests++
	if strings.Contains(strings.ToLower(body.Query), "mutation") {
		h.writes++
	}
	hold, arrived := h.hold, h.arrived
	status, retry := h.status, h.retry
	found := []Media{}
	for _, id := range body.Variables.IDs {
		if m, ok := h.media[id]; ok {
			found = append(found, m)
		}
	}
	name := body.Variables.Name
	var list []ListEntry
	var known, private bool
	if name != nil {
		list, known = h.lists[*name]
		private = h.private[*name]
	}
	h.mu.Unlock()
	if hold != nil && name != nil {
		arrived <- struct{}{}
		<-hold
	}
	if status != 0 {
		if retry != "" {
			w.Header().Set("Retry-After", retry)
		}
		http.Error(w, "failing", status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if name != nil {
		h.serveUserList(w, list, known, private)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"Page": map[string]any{"media": found}}})
}

func (h *Host) serveUserList(w http.ResponseWriter, list []ListEntry, known, private bool) {
	failure := func(status int, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": message, "status": status}}, "data": map[string]any{"MediaListCollection": nil}})
	}
	switch {
	case private:
		failure(http.StatusForbidden, "Private User")
		return
	case !known:
		failure(http.StatusNotFound, "User not found")
		return
	}
	entries := make([]map[string]any, 0, len(list))
	for _, e := range list {
		entries = append(entries, map[string]any{"status": e.Status, "progress": e.Progress, "media": map[string]any{"id": e.MediaID, "episodes": e.Episodes}})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"MediaListCollection": map[string]any{"lists": []map[string]any{{"entries": entries}}}}})
}
