package api

import (
	"errors"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

const APP_NAME = "Plex-anilist-sync"

type Server struct {
	store      storage.Storage
	mediaHost  mediaHost.MediaHost
	listenAddr string
}

func NewServer(listenAddr string, store storage.Storage, mediaHost mediaHost.MediaHost) *Server {
	return &Server{listenAddr: listenAddr, store: store, mediaHost: mediaHost}
}

func (s *Server) Start() error {
	e := echo.New()

	e.Static(routes.PUBLIC, "public")

	err := s.initialiseMediaHost()
	if err != nil {
		log.Warnf("Failed to initialise media host: %v", err)
	}

	e.GET(routes.HOME, s.handleGetRoot)
	e.GET(routes.SETUP_USER, s.handleGetSetupUser)
	e.GET(routes.SETUP_PLEX_AUTH, s.handlePlexAuth)
	e.POST(routes.SETUP_VALIDATE, s.handleValidateSetupForm)

	e.GET(routes.SETUP_LIBRARIES, s.handleSetupLibraries)
	e.POST(routes.LIBRARIES, s.postLibraries)

	e.POST(routes.USER, s.handlePostUser)

	e.Logger.Fatal(e.Start(":8000"))

	return nil
}

func (s *Server) initialiseMediaHost() error {
	user, err := s.store.GetUser()
	if err != nil {
		return err
	}

	client := http.Client{}
	if user.PlexToken == nil {
		return errors.New("Failed to initialise media host: user has no plex token")
	}

	return s.mediaHost.Initialize(*user.PlexToken, user.PlexUrl, &client)
}
