package plex

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ReidMason/plex-ani-sync/internal/httpClient"
)

type PlexAuth struct {
	clientIdentifier string
	appName          string
	httpClient       httpClient.HTTPClient
}

func NewPlexAuth(clientIdentifier string, appName string, httpClient httpClient.HTTPClient) *PlexAuth {
	return &PlexAuth{
		clientIdentifier: clientIdentifier,
		appName:          appName,
		httpClient:       httpClient,
	}
}

func (p *PlexAuth) GetPlexAuthUrl() (string, error) {
	pin, err := p.generatePin()
	if err != nil {
		return "", err
	}

	authURL := buildAuthURL(pin.Code, p.clientIdentifier, p.appName)
	return authURL, nil
}

type PinResponse struct {
	ID               int      `json:"id"`
	Code             string   `json:"code"`
	Product          string   `json:"product"`
	Trusted          bool     `json:"trusted"`
	QR               string   `json:"qr"`
	ClientIdentifier string   `json:"clientIdentifier"`
	Location         Location `json:"location"`
	ExpiresIn        int      `json:"expiresIn"`
	CreatedAt        string   `json:"createdAt"`
	ExpiresAt        string   `json:"expiresAt"`
	AuthToken        *string  `json:"authToken"`
	NewRegistration  *bool    `json:"newRegistration"`
}

type Location struct {
	Code                       string `json:"code"`
	EuropeanUnionMember        bool   `json:"european_union_member"`
	ContinentCode              string `json:"continent_code"`
	Country                    string `json:"country"`
	City                       string `json:"city"`
	TimeZone                   string `json:"time_zone"`
	PostalCode                 string `json:"postal_code"`
	InPrivacyRestrictedCountry bool   `json:"in_privacy_restricted_country"`
	InPrivacyRestrictedRegion  bool   `json:"in_privacy_restricted_region"`
	Subdivisions               string `json:"subdivisions"`
	Coordinates                string `json:"coordinates"`
}

func (p *PlexAuth) generatePin() (PinResponse, error) {
	req, err := http.NewRequest("POST", "https://plex.tv/api/v2/pins?strong=true", nil)
	if err != nil {
		return PinResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Product", p.appName)
	req.Header.Set("X-Plex-Client-Identifier", p.clientIdentifier)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return PinResponse{}, err
	}
	defer resp.Body.Close()

	var pinResponse PinResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinResponse); err != nil {
		return PinResponse{}, err
	}

	return pinResponse, nil
}

func buildAuthURL(pin string, clientIdentifier string, appName string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "app.plex.tv",
		Path:   "/auth",
	}
	q := u.Query()
	q.Set("clientID", clientIdentifier)
	q.Set("code", pin)
	q.Set("context[device][product]", appName)

	return u.String() + "#?" + q.Encode()
}
