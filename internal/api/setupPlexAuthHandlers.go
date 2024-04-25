package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ReidMason/plex-ani-sync/internal/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/labstack/echo/v4"
)

func (s *Server) getSetupPlexAuth(c echo.Context) error {
	pinId, err := strconv.Atoi(c.Request().URL.Query().Get("pinid"))
	if err != nil {
		slog.Error("Failed to parse pin id", slog.Any("error", err))
		return c.String(http.StatusBadRequest, "Invalid pin id")
	}

	clientIdentifier := c.Request().URL.Query().Get("clientIdentifier")
	code := c.Request().URL.Query().Get("code")

	authResponse, err := mediaHost.GetAuthResponse(pinId, code, clientIdentifier)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get auth response")
	}

	user, err := s.userManager.GetUser()
	if err != nil {
		slog.Error("Failed to find user", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to find user")
	}

	user.PlexToken = authResponse.AuthToken
	s.userManager.UpdateUser(user)

	err = s.InitialiseMediaHost()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to initialize media host")
	}

	return c.Redirect(http.StatusFound, routes.SETUP_LIBRARIES)
}
