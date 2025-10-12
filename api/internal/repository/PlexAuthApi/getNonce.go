package plexAuthApi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/internal/application/plexAuth"
)

func (p *PlexAuthApi) GetNonce(clientIdentifier string) (string, error) {
	req, err := http.NewRequest("GET", "https://clients.plex.tv/api/v2/auth/nonce", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var getNonceResponse plexAuth.GetNonceResponse
	if err := json.Unmarshal(responseBody, &getNonceResponse); err != nil {
		return "", err
	}

	return getNonceResponse.Nonce, nil
}
