package plexController

import (
	"errors"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/server/responseFactory"
)

type GetPlexAuthUrlResponseDto struct {
	AuthURL string `json:"authUrl"`
}

func (p *PlexController) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	forwardUrl := r.URL.Query().Get("forwardUrl")
	if forwardUrl == "" {
		responseFactory.BadRequest(w, errors.New("forwardUrl is required"), "Forward URL is required")
		return
	}

	authURL, err := p.plexAuth.GetAuthUrl(forwardUrl)
	if err != nil {
		responseFactory.InternalServerError(w, err, "Failed to generate Plex authentication URL")
		return
	}

	responseFactory.Ok(w, GetPlexAuthUrlResponseDto{AuthURL: string(authURL)}, "Successfully generated Plex authentication URL")
}
