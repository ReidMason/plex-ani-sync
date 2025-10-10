package plexController

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/server/responseFactory"
)

type PlexPinIdValidator interface {
	GetPin(pinId int) (plex.GetPinResponse, error)
}

func (p *PlexController) GetPin(w http.ResponseWriter, r *http.Request) {
	pinId := r.PathValue("pinId")
	pinIdInt, err := strconv.Atoi(pinId)
	if err != nil {
		responseFactory.BadRequest(w, err, "Invalid pinId")
		return
	}

	pin, err := p.PinIdValidator.GetPin(pinIdInt)
	if err != nil {
		responseFactory.InternalServerError(w, err, "Failed to check pinId")
		return
	}

	fmt.Println(pin.AuthToken)
	responseFactory.Ok(w, "Valid", "Successfully checked pinId")
}
