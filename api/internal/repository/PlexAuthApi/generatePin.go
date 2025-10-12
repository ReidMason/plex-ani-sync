package plexAuthApi

import (
	"encoding/json"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/internal/application/plexAuth"
)

type PlexApi struct{}

type GeneratePinResponse struct {
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

func (p *PlexAuthApi) GeneratePin(appName string, clientIdentifier string) (plexAuth.PlexPin, error) {
	req, err := http.NewRequest("POST", "https://plex.tv/api/v2/pins?strong=true", nil)
	if err != nil {
		return plexAuth.PlexPin{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Product", appName)
	req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return plexAuth.PlexPin{}, err
	}
	defer resp.Body.Close()

	var pinResponse GeneratePinResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinResponse); err != nil {
		return plexAuth.PlexPin{}, err
	}

	return plexAuth.PlexPin{
		Pin:              pinResponse.Code,
		PinId:            pinResponse.ID,
		ClientIdentifier: pinResponse.ClientIdentifier,
		AppName:          pinResponse.Product,
		AuthToken:        pinResponse.AuthToken,
	}, nil
}
