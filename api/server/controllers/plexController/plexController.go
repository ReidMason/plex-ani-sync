package plexController

import (
	"net/http"

	"github.com/ReidMason/plex-ani-sync/server/common"
)

type PlexController struct {
	AuthURLGenerator PlexAuthURLGenerator
}

type PlexAuthURLGenerator interface {
	GetPlexAuthUrl() (string, error)
}

func New(plexAuthUrlGenerator PlexAuthURLGenerator) *PlexController {
	return &PlexController{
		AuthURLGenerator: plexAuthUrlGenerator,
	}
}

func (p *PlexController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(common.BuildEndpointUrl("plex/authUrl"), p.GetAuthURL)
}

func (p *PlexController) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	authURL, err := p.AuthURLGenerator.GetPlexAuthUrl()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(authURL))
}
