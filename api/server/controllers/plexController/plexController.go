package plexController

import (
	"net/http"

	"github.com/ReidMason/plex-ani-sync/application/plexAuth"
	"github.com/ReidMason/plex-ani-sync/server/common"
)

type PlexController struct {
	plexAuth *plexAuth.PlexAuth
}

func New(plexAuth *plexAuth.PlexAuth) *PlexController {
	return &PlexController{
		plexAuth: plexAuth,
	}
}

func (p *PlexController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(common.BuildEndpointUrl("plex/authUrl"), p.GetAuthURL)
}
