package plexController

import (
	"net/http"

	"github.com/ReidMason/plex-ani-sync/server/common"
)

type PlexController struct {
	AuthURLGenerator PlexAuthURLGenerator
	PinIdValidator   PlexPinIdValidator
}

type PlexAuthURLGenerator interface {
	GetPlexAuthUrl(forwardUrl string) (string, error)
}

func New(plexAuthUrlGenerator PlexAuthURLGenerator, pinIdValidator PlexPinIdValidator) *PlexController {
	return &PlexController{
		AuthURLGenerator: plexAuthUrlGenerator,
		PinIdValidator:   pinIdValidator,
	}
}

func (p *PlexController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(common.BuildEndpointUrl("plex/authUrl"), p.GetAuthURL)
	mux.HandleFunc(common.BuildEndpointUrl("plex/pins/{pinId}"), p.GetPin)
}
