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

func makeRequest[T any](client HttpClient, request *http.Request) (T, error) {
	var result T
	resp, err := client.Do(request)
	if err != nil {
		return result, err
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("Request failed: %s", resp.Status)
	}

	defer resp.Body.Close()
	return parseResponse[T](resp.Body)
}

func parseResponse[T any](responseBody io.ReadCloser) (T, error) {
	var result T
	body, err := io.ReadAll(responseBody)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}

	return result, nil
}

func (p Plex) GetCurrentUser() (PlexUser, error) {
	var plexUser PlexUser
	req, err := buildRequest("GET", "https://plex.tv/api/v2/user", p.token)
	if err != nil {
		return plexUser, err
	}

	return makeRequest[PlexUser](p.client, req)
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
