package plexAuthApi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	plexAuthDomain "github.com/ReidMason/plex-ani-sync/internal/domain/plexAuth"
)

type GetTokenResponse struct {
	AuthToken string `json:"auth_token"`
}

func (p *PlexAuthApi) GetToken(clientIdentifier plexAuthDomain.PlexToken, signedPlexJwt plexAuthDomain.SignedPlexJwt) (plexAuthDomain.PlexToken, error) {
	body := []byte(fmt.Sprintf(`{"jwt": "%s"}`, signedPlexJwt))
	req, err := http.NewRequest("POST", "https://clients.plex.tv/api/v2/auth/token", bytes.NewBuffer(body))
	if err != nil {
		return plexAuthDomain.PlexToken(""), err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", string(clientIdentifier))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return plexAuthDomain.PlexToken(""), err
	}
	defer resp.Body.Close()

	var getTokenResponse GetTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&getTokenResponse); err != nil {
		return plexAuthDomain.PlexToken(""), err
	}

	return plexAuthDomain.PlexToken(getTokenResponse.AuthToken), nil
}
