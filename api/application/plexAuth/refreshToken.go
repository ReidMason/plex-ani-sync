package plexAuth

import (
	"github.com/ReidMason/plex-ani-sync/internal/domain/common"
	plexAuthDomain "github.com/ReidMason/plex-ani-sync/internal/domain/plexAuth"
)

func (p *PlexAuth) RefreshToken() error {
	now := p.time.Now()
	nonce, err := p.PlexAuthApiRegistry.GetNonce(common.ClientIdentifier)
	if err != nil {
		return err
	}

	privateKey, err := p.StorageRepository.getPrivateKey()
	if err != nil {
		return err
	}

	signedPlexJwt, err := plexAuthDomain.GetPlexJwt(nonce, now, privateKey)
	if err != nil {
		return err
	}

	plexToken, err := p.PlexAuthApiRegistry.GetToken(common.ClientIdentifier, signedPlexJwt)
	if err != nil {
		return err
	}

	err = p.StorageRepository.setToken(plexToken)
	if err != nil {
		return err
	}

	return nil
}
