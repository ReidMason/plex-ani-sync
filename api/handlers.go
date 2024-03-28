package api

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/api/routes"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
)

func (s *Server) getIndex(c echo.Context) error {
	// Redirect to setup if user doesn't exist
	_, err := s.store.GetUser()
	if err != nil {
		log.Println("Failed to find existing user redirecting to setup")
		c.Redirect(http.StatusFound, routes.SETUP_USER)
		return nil
	}

	user, err := s.mediaHost.GetCurrentUser()
	if err != nil {
		log.Println("Failed to get current user from media host: ", err)
		return c.String(http.StatusInternalServerError, "Failed to get current user from media host")
	}

	slog.Info("Got user", slog.Any("user", user))

	series, err := s.mediaHost.GetSeries("1")
	if err != nil {
		slog.Error("Failed to get series from media host", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get series from media host")
	}

	slog.Info("Got series", slog.Any("series", series))

	component := views.Index(user)
	return component.Render(c.Request().Context(), c.Response())
}
