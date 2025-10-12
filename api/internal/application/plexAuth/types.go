package plexAuth

import plexAuthDomain "github.com/ReidMason/plex-ani-sync/internal/domain/plexAuth"

type PlexAuthApiRepository interface {
	GetToken(clientIdentifier plexAuthDomain.PlexToken, signedPlexJwt plexAuthDomain.SignedPlexJwt) (plexAuthDomain.PlexToken, error)
	GeneratePin(appName string, clientIdentifier string) (PlexPin, error)
	GetNonce(clientIdentifier string) (plexAuthDomain.PlexNonce, error)
}

type StorageRepository interface {
	setToken(token plexAuthDomain.PlexToken) error
	getPrivateKey() ([]byte, error)
}

type PlexPin struct {
	Pin              string
	PinId            int
	ClientIdentifier string
	AppName          string
	AuthToken        *string
}

type GetNonceResponse struct {
	Nonce string `json:"nonce"`
}
