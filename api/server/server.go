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

	// Wrap the router with CORS middleware
	handler := corsMiddleware(s.Router)

	http.ListenAndServe(":"+s.Port, handler)
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
