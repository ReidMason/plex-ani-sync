package plexAuth

import (
	"github.com/ReidMason/plex-ani-sync/internal/domain/common"
	plexAuthDomain "github.com/ReidMason/plex-ani-sync/internal/domain/plexAuth"
)

func (p *PlexAuth) GetAuthUrl(forwardUrl string) (plexAuthDomain.PlexAuthURl, error) {
	pin, err := p.PlexAuthApiRegistry.GeneratePin(common.AppName, common.ClientIdentifier)
	if err != nil {
		return plexAuthDomain.PlexAuthURl(""), err
	}

	authURL, err := plexAuthDomain.BuildPlexAuthUrl(pin.Pin, pin.PinId, pin.ClientIdentifier, pin.AppName, forwardUrl)
	if err != nil {
		return plexAuthDomain.PlexAuthURl(""), err
	}

	return authURL, nil
}
