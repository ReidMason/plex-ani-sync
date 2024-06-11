package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/internal/api/routes"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
)

func (s *Server) getIndex(c echo.Context) error {
	// Redirect to setup if user doesn't exist
	_, err := s.userManager.GetUser()
	if err != nil {
		s.log.Warn("Failed to find existing user redirecting to setup", slog.Any("error", err))
		c.Redirect(http.StatusFound, routes.SETUP_USER)
		return nil
	}

	user, err := s.mediaHost.GetCurrentUser()
	if err != nil {
		s.log.Error("Failed to get current user from media host", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get current user from media host")
	}

	s.log.Info("Got user", slog.Any("user", user))

	series, err := s.mediaHost.GetSeries("1")
	if err != nil {
		s.log.Error("Failed to get series from media host", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get series from media host")
	}

	s.log.Info(fmt.Sprintf("Found %d series", len(series)))

	seasons, err := s.mediaHost.GetSeasons(series[0].Id)
	if err != nil {
		s.log.Error("Failed to get seasons from media host", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get seasons from media host")
	}

	s.log.Info(fmt.Sprintf("Found %d seasons", len(seasons)))

	episodes, err := s.mediaHost.GetEpisodes(seasons[0].Id)
	if err != nil {
		s.log.Error("Failed to get episodes from media host", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get episodes from media host")
	}

	s.log.Info(fmt.Sprintf("Found %d episodes", len(episodes)))

	component := views.Index(user)
	return component.Render(c.Request().Context(), c.Response())
}
