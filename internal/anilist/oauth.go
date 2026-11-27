package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const viewerQuery = `query { Viewer { name } }`

// ErrUnauthorised means AniList rejected the access token.
var ErrUnauthorised = errors.New("anilist: token rejected")

// OAuth runs the authorization-code grant against AniList.
type OAuth struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HTTP         *http.Client
}

func (o OAuth) AuthoriseURL(state string) string {
	q := url.Values{"client_id": {o.ClientID}, "redirect_uri": {o.RedirectURI}, "response_type": {"code"}, "state": {state}}
	return o.BaseURL + "/authorize?" + q.Encode()
}

// Exchange trades an authorisation code for an access token.
func (o OAuth) Exchange(ctx context.Context, code string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"grant_type": "authorization_code", "client_id": o.ClientID, "client_secret": o.ClientSecret, "redirect_uri": o.RedirectURI, "code": code,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/token", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
	}
	decodeErr := json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode != http.StatusOK || decodeErr != nil || body.AccessToken == "" {
		return "", fmt.Errorf("anilist: token exchange failed with status %d", resp.StatusCode)
	}
	return body.AccessToken, nil
}

// Viewer returns the username of the account token belongs to, or ErrUnauthorised.
func (c *Client) Viewer(ctx context.Context, token string) (string, error) {
	env, err := c.sendAs(ctx, token, viewerQuery, nil)
	if err != nil {
		return "", err
	}
	for _, e := range env.Errors {
		if e.Status == http.StatusUnauthorized {
			return "", ErrUnauthorised
		}
	}
	if len(env.Errors) > 0 {
		return "", fmt.Errorf("anilist: %s", env.Errors[0].Message)
	}
	var data struct {
		Viewer struct {
			Name string `json:"name"`
		} `json:"Viewer"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil || data.Viewer.Name == "" {
		return "", fmt.Errorf("anilist: viewer response unreadable")
	}
	return data.Viewer.Name, nil
}
