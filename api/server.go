package api

import (
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

	e.Static("/public", "public")

	e.GET("/", s.handleGetRoot)
	e.GET("/setup/user", s.handleGetSetupUser)
	e.GET("/setup/plex-auth", s.handlePlexAuth)
	e.POST("/setup/validate", s.handleValidateSetupForm)

	e.POST("/user", s.handlePostUser)

	e.Logger.Fatal(e.Start(":8000"))

	return nil
}
