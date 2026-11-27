package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ReidMason/plex-ani-sync/internal/db/dbgen"
)

func (p *plexLink) registerAniListAuthorisation(mux *http.ServeMux, a *auth) {
	mux.HandleFunc("GET /auth/anilist/start", a.authenticated(p.startAniListAuthorisation))
	mux.HandleFunc("GET /auth/anilist/callback", a.authenticated(p.finishAniListAuthorisation))
}

const (
	aniListStateCookie = "plex_ani_sync_anilist_state"
	cookieFieldSep     = "~"

	aniListPreviousCookie = "plex_ani_sync_anilist_previous"
)

var returnPages = map[string]string{"settings": "/settings", "setup": "/setup/anilist"}

// newState binds a random nonce and issue time to the Session, so a state issued to one Session is useless to another.
func (p *plexLink) newState(r *http.Request) string {
	payload := randomID() + "." + strconv.FormatInt(p.now().Unix(), 10)
	return payload + "." + p.stateMAC(r, payload)
}

func (p *plexLink) stateMAC(r *http.Request, payload string) string {
	c, _ := r.Cookie(sessionCookie)
	mac := hmac.New(sha256.New, p.stateKey)
	mac.Write([]byte("anilist-state\x00"))
	mac.Write(hashID(cookieValue(c)))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (p *plexLink) validState(r *http.Request, state string) bool {
	parts := strings.Split(state, ".")
	if len(parts) != 3 || parts[0] == "" {
		return false
	}
	issued, err := strconv.ParseInt(parts[1], 10, 64)
	age := p.now().Sub(time.Unix(issued, 0))
	if err != nil || age < 0 || age > loginStateTTL {
		return false
	}
	return hmac.Equal([]byte(parts[2]), []byte(p.stateMAC(r, parts[0]+"."+parts[1])))
}

func returnPage(returnTo string) string {
	if page, ok := returnPages[returnTo]; ok {
		return page
	}
	return returnPages["settings"]
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
	state := p.newState(r)
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
	http.SetCookie(w, clearedCookie(aniListStateCookie, "/auth/anilist"))
	redirect := func(result string) {
		http.Redirect(w, r, returnPage(returnTo)+"?anilist="+result, http.StatusFound)
	}
	if !hmac.Equal([]byte(query.Get("state")), []byte(state)) || !p.validState(r, state) {
		redirect("invalid")
		return
	}
	if query.Get("error") != "" || query.Get("code") == "" {
		redirect("denied")
		return
	}
	previous, err := p.completeAniListAuthorisation(r.Context(), s.UserID, query.Get("code"))
	if err != nil {
		p.logger.Warn("anilist authorisation failed", "error", err)
		redirect("failed")
		return
	}
	if previous != "" {
		flash := secureCookie(aniListPreviousCookie, url.QueryEscape(previous), "/api/trackers/anilist")
		flash.MaxAge = int(loginStateTTL.Seconds())
		http.SetCookie(w, flash)
	}
	redirect("authorised")
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
