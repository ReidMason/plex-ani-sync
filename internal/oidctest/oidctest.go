package oidctest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const keyID = "test-key"

type identity struct{ sub, name string }

type grant struct {
	identity
	nonce string
}

// Provider is a stub OIDC provider; its URL is the issuer.
type Provider struct {
	*httptest.Server
	key *rsa.PrivateKey

	mu       sync.Mutex
	identity identity
	codes    map[string]grant
	tokens   int
}

func New(t *testing.T) *Provider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	p := &Provider{key: key, identity: identity{"default-sub", "Default User"}, codes: map[string]grant{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                p.URL,
			"authorization_endpoint":                p.URL + "/authorize",
			"token_endpoint":                        p.URL + "/token",
			"end_session_endpoint":                  p.URL + "/end-session",
			"jwks_uri":                              p.URL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &key.PublicKey, KeyID: keyID, Algorithm: string(jose.RS256), Use: "sig"},
		}})
	})
	mux.HandleFunc("/authorize", p.authorize)
	mux.HandleFunc("/token", p.token)
	p.Server = httptest.NewServer(mux)
	t.Cleanup(p.Close)
	return p
}

func (p *Provider) SignInAs(sub, name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.identity = identity{sub, name}
}

func (p *Provider) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	code := hex.EncodeToString(b)
	p.mu.Lock()
	p.codes[code] = grant{identity: p.identity, nonce: q.Get("nonce")}
	p.mu.Unlock()

	redirect, err := url.Parse(q.Get("redirect_uri"))
	if err != nil {
		http.Error(w, "bad redirect_uri", http.StatusBadRequest)
		return
	}
	rq := redirect.Query()
	rq.Set("code", code)
	rq.Set("state", q.Get("state"))
	redirect.RawQuery = rq.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (p *Provider) TokenRequests() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tokens
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.tokens++
	g, ok := p.codes[r.FormValue("code")]
	delete(p.codes, r.FormValue("code"))
	p.mu.Unlock()
	if !ok {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: p.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	idToken, err := jwt.Signed(signer).Claims(map[string]any{
		"iss":   p.URL,
		"sub":   g.sub,
		"aud":   "plex-ani-sync",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"nonce": g.nonce,
		"name":  g.name,
	}).Serialize()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": "stub-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     idToken,
	})
}
