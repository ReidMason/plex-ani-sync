package api

import (
	"github.com/ReidMason/plex-ani-sync/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/labstack/echo/v4"
)

const APP_NAME = "Plex-anilist-sync"

type Server struct {
	store      storage.Storage
	listenAddr string
}

func NewServer(listenAddr string, store storage.Storage) *Server {
	return &Server{listenAddr: listenAddr, store: store}
}

func (s *Server) Start() error {
	e := echo.New()

	e.Static(routes.PUBLIC, "public")

	e.GET(routes.HOME, s.handleGetRoot)
	e.GET(routes.SETUP_USER, s.handleGetSetupUser)
	e.GET(routes.SETUP_PLEX_AUTH, s.handlePlexAuth)
	e.POST(routes.SETUP_VALIDATE, s.handleValidateSetupForm)

	e.POST(routes.USER, s.handlePostUser)

	e.Logger.Fatal(e.Start(":8000"))

	return nil
}
