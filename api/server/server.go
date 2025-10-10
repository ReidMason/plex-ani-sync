package server

import (
	"net/http"

	"github.com/ReidMason/plex-ani-sync/server/common"
)

type ControllerRegistry struct {
	Controllers []common.Controller
	Router      *http.ServeMux
	Port        string
}

func New(controllers []common.Controller, router *http.ServeMux, port string) *ControllerRegistry {
	return &ControllerRegistry{
		Controllers: controllers,
		Router:      router,
		Port:        port,
	}
}

func (s *ControllerRegistry) Start() {
	s.Router.HandleFunc("/", getRoot)

	s.RegisterControllers(s.Controllers)

	http.ListenAndServe(":"+s.Port, nil)
}

func (s *ControllerRegistry) RegisterControllers(controller []common.Controller) {
	for _, c := range controller {
		c.RegisterRoutes(s.Router)
	}
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}
