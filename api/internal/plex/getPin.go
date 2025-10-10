package plex

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type GetPinResponse struct {
	AuthToken        string   `json:"authToken"`
	ClientIdentifier string   `json:"clientIdentifier"`
	Code             string   `json:"code"`
	CreatedAt        string   `json:"createdAt"`
	ExpiresAt        string   `json:"expiresAt"`
	ExpiresIn        int      `json:"expiresIn"`
	ID               int      `json:"id"`
	Location         Location `json:"location"`
	NewRegistration  bool     `json:"newRegistration"`
	Product          string   `json:"product"`
	QR               string   `json:"qr"`
	Trusted          bool     `json:"trusted"`
}

func (p *PlexAuth) GetPin(pinId int) (GetPinResponse, error) {
	req, err := http.NewRequest("GET", "https://plex.tv/api/v2/pins/"+strconv.Itoa(pinId), nil)
	if err != nil {
		return GetPinResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", p.clientIdentifier)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return GetPinResponse{}, err
	}
	defer resp.Body.Close()

	// Read the response body
	var pinResponse GetPinResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinResponse); err != nil {
		return GetPinResponse{}, err
	}

	return pinResponse, nil
}
