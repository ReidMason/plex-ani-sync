package plex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Plex struct {
	client HttpClient
	token  string
	host   string
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func New(token string, host string, client HttpClient) Plex {
	return Plex{token: token, host: host, client: client}
}

func buildRequest(method, url, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("X-Plex-Token", token)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("accept", "application/json")

	return req, nil
}

func (p Plex) GetCurrentUser() (PlexUser, error) {
	var plexUser PlexUser
	req, err := buildRequest("GET", "https://plex.tv/api/v2/user", p.token)
	if err != nil {
		return plexUser, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return plexUser, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return plexUser, fmt.Errorf("Failed to get current Plex user: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return plexUser, err
	}

	if err := json.Unmarshal(body, &plexUser); err != nil {
		return plexUser, err
	}

	return plexUser, nil
}

type PlexUser struct {
	Locale            *string      `json:"locale"`
	Thumb             string       `json:"thumb"`
	Title             string       `json:"title"`
	Country           string       `json:"country"`
	ScrobbleTypes     string       `json:"scrobbleTypes"`
	FriendlyName      string       `json:"friendlyName"`
	UUID              string       `json:"uuid"`
	MailingListStatus string       `json:"mailingListStatus"`
	AuthToken         string       `json:"authToken"`
	Email             string       `json:"email"`
	Username          string       `json:"username"`
	Subscription      Subscription `json:"subscription"`
	ID                int          `json:"id"`
	JoinedAt          int          `json:"joinedAt"`
	Confirmed         bool         `json:"confirmed"`
	MailingListActive bool         `json:"mailingListActive"`
	Protected         bool         `json:"protected"`
	HasPassword       bool         `json:"hasPassword"`
	EmailOnlyAuth     bool         `json:"emailOnlyAuth"`
}

type Subscription struct {
	SubscribedAt   string `json:"subscribedAt"`
	Status         string `json:"status"`
	PaymentService string `json:"paymentService"`
	Plan           string `json:"plan"`
	Active         bool   `json:"active"`
}
