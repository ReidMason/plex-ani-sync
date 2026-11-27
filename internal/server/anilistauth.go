package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
)

const (
	aniListStateCookie = "plex_ani_sync_anilist_state"
	cookieFieldSep     = "~"
)

var returnPages = map[string]string{"settings": "/settings", "setup": "/setup/anilist"}

// signState binds a random nonce to the Session, so a state issued to one Session is useless to another.
func (p *plexLink) signState(r *http.Request, nonce string) string {
	c, _ := r.Cookie(sessionCookie)
	mac := hmac.New(sha256.New, p.stateKey)
	mac.Write([]byte("anilist-state\x00"))
	mac.Write(hashID(cookieValue(c)))
	mac.Write([]byte(nonce))
	return nonce + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (p *plexLink) startAniListAuthorisation(w http.ResponseWriter, r *http.Request, _ dbgen.GetSessionUserRow) {
	if p.aniListOAuth == nil {
		http.NotFound(w, r)
		return
	}
	returnTo := r.URL.Query().Get("return")
	if _, ok := returnPages[returnTo]; !ok {
		returnTo = "settings"
	}
	state := p.signState(r, randomID())
	cookie := secureCookie(aniListStateCookie, state+cookieFieldSep+returnTo, "/auth/anilist")
	cookie.MaxAge = int(loginStateTTL.Seconds())
	http.SetCookie(w, cookie)
	http.Redirect(w, r, p.aniListOAuth.AuthoriseURL(state), http.StatusFound)
}

func (p *plexLink) finishAniListAuthorisation(w http.ResponseWriter, r *http.Request, s dbgen.GetSessionUserRow) {
	if p.aniListOAuth == nil {
		http.NotFound(w, r)
		return
	}
	c, _ := r.Cookie(aniListStateCookie)
	state, returnTo, _ := strings.Cut(cookieValue(c), cookieFieldSep)
	query := r.URL.Query()
	nonce, _, _ := strings.Cut(state, ".")
	if state == "" || nonce == "" || !hmac.Equal([]byte(query.Get("state")), []byte(state)) || !hmac.Equal([]byte(state), []byte(p.signState(r, nonce))) {
		http.Error(w, "invalid authorisation state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, clearedCookie(aniListStateCookie, "/auth/anilist"))
	page, ok := returnPages[returnTo]
	if !ok {
		page = returnPages["settings"]
	}
	redirect := func(result string, extra url.Values) {
		if extra == nil {
			extra = url.Values{}
		}
		extra.Set("anilist", result)
		http.Redirect(w, r, page+"?"+extra.Encode(), http.StatusFound)
	}
	if query.Get("error") != "" || query.Get("code") == "" {
		redirect("denied", nil)
		return
	}
	previous, err := p.completeAniListAuthorisation(r.Context(), s.UserID, query.Get("code"))
	if err != nil {
		p.logger.Warn("anilist authorisation failed", "error", err)
		redirect("failed", nil)
		return
	}
	extra := url.Values{}
	if previous != "" {
		extra.Set("previous", previous)
	}
	redirect("authorised", extra)
}

// completeAniListAuthorisation stores the token and the account's username, and returns the username it replaced, if different.
func (p *plexLink) completeAniListAuthorisation(ctx context.Context, userID int64, code string) (string, error) {
	token, err := p.aniListOAuth.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	username, err := p.anilistClient.Viewer(ctx, token)
	if err != nil {
		return "", err
	}
	sealed, err := p.box.Seal([]byte(token))
	if err != nil {
		return "", err
	}
	previous, err := p.queries.GetTrackerUsername(ctx, dbgen.GetTrackerUsernameParams{UserID: userID, Kind: trackerKindAniList})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if err := p.queries.UpsertAuthorisedTracker(ctx, dbgen.UpsertAuthorisedTrackerParams{UserID: userID, Kind: trackerKindAniList, Username: username, TokenEncrypted: sealed}); err != nil {
		return "", err
	}
	if previous == username {
		return "", nil
	}
	return previous, nil
}
