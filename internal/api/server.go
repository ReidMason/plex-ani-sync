package api

import (
	"errors"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/internal/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/userManager"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

const APP_NAME = "Plex-anilist-sync"

type Server struct {
	mediaHost   mediaHost.MediaHost
	userManager userManager.UserManager
	listenAddr  string
}

func NewServer(listenAddr string, mediaHost mediaHost.MediaHost, userManager userManager.UserManager) *Server {
	return &Server{listenAddr: listenAddr, mediaHost: mediaHost, userManager: userManager}
}

func (s *Server) Start() error {
	e := echo.New()

	e.Static(routes.PUBLIC, "public")

	err := s.initialiseMediaHost()
	if err != nil {
		log.Warnf("Failed to initialise media host: %v", err)
	}

	e.GET(routes.INDEX, s.getIndex)

	e.GET(routes.SETUP_USER, s.getSetupUser)
	e.POST(routes.SETUP_USER, s.postSetupUser)
	e.POST(routes.SETUP_USER_VALIDATE, s.postSetupUserValidate)

	e.GET(routes.SETUP_PLEX_AUTH, s.getSetupPlexAuth)

	e.GET(routes.SETUP_LIBRARIES, s.getSetupLibraries)
	e.POST(routes.SETUP_LIBRARIES, s.postSetupLibraries)

	e.Logger.Fatal(e.Start(":8000"))

	return nil
}

func (s *Server) initialiseMediaHost() error {
	user, err := s.userManager.GetUser()
	if err != nil {
		return err
	}

	client := http.Client{}
	if user.PlexToken == nil {
		return errors.New("Failed to initialise media host: user has no plex token")
	}

	return s.mediaHost.Initialize(*user.PlexToken, user.PlexUrl, &client)
}
