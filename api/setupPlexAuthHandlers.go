package api

import (
	"net/http"
	"strconv"

	"github.com/ReidMason/plex-ani-sync/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/labstack/echo/v4"
)

func (s *Server) getSetupPlexAuth(c echo.Context) error {
	pinId, err := strconv.Atoi(c.Request().URL.Query().Get("pinid"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid pin id")
	}

	clientIdentifier := c.Request().URL.Query().Get("clientIdentifier")
	code := c.Request().URL.Query().Get("code")

	pollingLink, err := mediaHost.BuildAuthTokenPollingLink(pinId, code, clientIdentifier)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to build polling link")
	}

	authResponse, err := mediaHost.PollForAuthToken(pollingLink)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to poll for auth token")
	}

	user, err := s.store.GetUser()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to find user")
	}

	if authResponse.AuthToken == nil {
		return c.String(http.StatusInternalServerError, "Failed to authenticate with Plex, no auth token found")
	}

	user.PlexToken = authResponse.AuthToken
	s.store.UpdateUser(user)

	err = s.initialiseMediaHost()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to initialize media host")
	}

	return c.Redirect(http.StatusFound, routes.SETUP_LIBRARIES)
}
