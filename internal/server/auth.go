package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
)

const (
	sessionCookie  = "plex_ani_sync_session"
	loginCookie    = "plex_ani_sync_login"
	sessionTTL     = 30 * 24 * time.Hour
	loginStateTTL  = 10 * time.Minute
	randomIDLength = 32
)

type auth struct {
	oauth    *oauth2.Config
	issuer   string
	verifier *oidc.IDTokenVerifier
	queries  *dbgen.Queries
	logger   *slog.Logger
}

func (a *auth) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/login", a.login)
	mux.HandleFunc("GET /auth/callback", a.callback)
	mux.HandleFunc("POST /auth/logout", a.logout)
}

func (a *auth) login(w http.ResponseWriter, r *http.Request) {
	state, nonce, verifier := randomID(), randomID(), oauth2.GenerateVerifier()
	loginState := secureCookie(loginCookie, strings.Join([]string{state, nonce, verifier}, "."), "/auth")
	loginState.MaxAge = int(loginStateTTL.Seconds())
	http.SetCookie(w, loginState)
	http.Redirect(w, r, a.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

func (a *auth) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(loginCookie)
	parts := strings.Split(cookieValue(c), ".")
	if err != nil || len(parts) != 3 || parts[0] == "" ||
		subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("state")), []byte(parts[0])) != 1 {
		http.Error(w, "invalid login state", http.StatusBadRequest)
		return
	}
	nonce, verifier := parts[1], parts[2]
	http.SetCookie(w, clearedCookie(loginCookie, "/auth"))
	if providerErr := r.URL.Query().Get("error"); providerErr != "" {
		a.logger.Warn("provider returned error", "error", providerErr, "description", r.URL.Query().Get("error_description"))
		http.Error(w, "sign-in was not completed by the identity provider", http.StatusBadRequest)
		return
	}

	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		a.logger.Warn("code exchange failed", "error", err)
		http.Error(w, "sign-in failed", http.StatusBadGateway)
		return
	}
	rawID, _ := token.Extra("id_token").(string)
	idToken, err := a.verifier.Verify(r.Context(), rawID)
	if err != nil || idToken.Nonce != nonce {
		a.logger.Warn("id token rejected", "error", err)
		http.Error(w, "sign-in failed", http.StatusUnauthorized)
		return
	}
	var claims struct {
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "sign-in failed", http.StatusUnauthorized)
		return
	}

	userID, err := a.queries.UpsertUser(r.Context(), dbgen.UpsertUserParams{OidcIssuer: a.issuer, OidcSubject: idToken.Subject})
	if err != nil {
		a.internalError(w, err)
		return
	}
	sessionID := randomID()
	expires := time.Now().Add(sessionTTL)
	if err := a.queries.CreateSession(r.Context(), dbgen.CreateSessionParams{
		IDHash:      hashID(sessionID),
		UserID:      userID,
		DisplayName: firstNonEmpty(claims.Name, claims.PreferredUsername, idToken.Subject),
		ExpiresAt:   pgtype.Timestamptz{Time: expires, Valid: true},
	}); err != nil {
		a.internalError(w, err)
		return
	}
	session := secureCookie(sessionCookie, sessionID, "/")
	session.Expires = expires
	http.SetCookie(w, session)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *auth) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := a.queries.RevokeSession(r.Context(), hashID(c.Value)); err != nil {
			a.internalError(w, err)
			return
		}
	}
	http.SetCookie(w, clearedCookie(sessionCookie, "/"))
	w.WriteHeader(http.StatusNoContent)
}

// authenticated runs next only for requests carrying a live Session cookie.
func (a *auth) authenticated(next func(http.ResponseWriter, *http.Request, dbgen.GetSessionUserRow)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || c.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		session, err := a.queries.GetSessionUser(r.Context(), hashID(c.Value))
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		next(w, r, session)
	}
}

func (a *auth) internalError(w http.ResponseWriter, err error) {
	a.logger.Error("request failed", "error", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func secureCookie(name, value, path string) *http.Cookie {
	return &http.Cookie{Name: name, Value: value, Path: path, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode}
}

func clearedCookie(name, path string) *http.Cookie {
	c := secureCookie(name, "", path)
	c.MaxAge = -1
	return c
}

func randomID() string {
	b := make([]byte, randomIDLength)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashID(id string) []byte {
	sum := sha256.Sum256([]byte(id))
	return sum[:]
}

func cookieValue(c *http.Cookie) string {
	if c == nil {
		return ""
	}
	return c.Value
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
