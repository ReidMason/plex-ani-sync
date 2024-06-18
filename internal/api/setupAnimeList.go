package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ReidMason/plex-ani-sync/internal/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/ReidMason/plex-ani-sync/templates/components"
	"github.com/ReidMason/plex-ani-sync/templates/components/ui"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
)

func (s Server) getSetupAnimeList(c echo.Context) error {
	formData := components.SetupAnimeListFormData{
		ClientId: ui.Field{
			Name:        "ClientId",
			Value:       "",
			Valid:       true,
			Placeholder: "Enter your Anilist client id",
		},
		Secret: ui.Field{
			Name:        "Secret",
			Value:       "",
			Valid:       true,
			Placeholder: "Enter your Anilist client secret",
		},
	}

	user, err := s.userManager.GetUser()
	if err != nil {
		s.log.Error("Failed to find user", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to find user")
	}

	if user.AnimeListClientId != nil {
		formData.ClientId.Value = *user.AnimeListClientId
	}
	if user.AnimeListClientSecret != nil {
		formData.Secret.Value = *user.AnimeListClientSecret
	}

	redirectUrl, err := buildRedirectUrl(user.HostUrl)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to build redirect url")
	}

	view := views.SetupAnimelist(formData, redirectUrl)
	return view.Render(c.Request().Context(), c.Response())
}

func (s Server) postSetupAnimeList(c echo.Context) error {
	clientId := c.FormValue("ClientId")
	secret := c.FormValue("Secret")

	user, err := s.userManager.GetUser()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to find user")
	}

	user.AnimeListClientId = &clientId
	user.AnimeListClientSecret = &secret
	err = s.userManager.UpdateUser(user.Id, storage.UserUpdate{User: &user})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update user")
	}

	redirectUrl, err := buildRedirectUrl(user.HostUrl)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to build redirect url")
	}

	authUrl := fmt.Sprintf("https://anilist.co/api/v2/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code", clientId, redirectUrl)
	slog.Info("authUrl", slog.String("authUrl", authUrl))

	c.Response().Header().Set("HX-Redirect", authUrl)
	return c.String(http.StatusOK, authUrl)
}

func (s Server) getSetupAnimeListValidate(c echo.Context) error {
	code := c.Request().URL.Query().Get("code")

	slog.Info("code", slog.String("code", code))

	user, err := s.userManager.GetUser()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to find user")
	}

	if user.AnimeListClientId == nil || user.AnimeListClientSecret == nil {
		return c.String(http.StatusInternalServerError, "Client ID or Client Secret not set")
	}

	redirectUrl, err := buildRedirectUrl(user.HostUrl)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to build redirect url")
	}

	tokenResponse, err := s.animeList.GetAuthToken(*user.AnimeListClientId, *user.AnimeListClientSecret, redirectUrl, code)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get token")
	}

	user.AnimeListToken = &tokenResponse.AccessToken
	user.AnimeListRefreshToken = &tokenResponse.RefreshToken
	err = s.userManager.UpdateUser(user.Id, storage.UserUpdate{User: &user})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update user")
	}

	return c.Redirect(http.StatusFound, routes.INDEX)
}

func buildRedirectUrl(hostUrl string) (string, error) {
	url, err := url.Parse(hostUrl)
	if err != nil {
		return "", err
	}

	return url.JoinPath(routes.SETUP_ANIMELIST_VALIDATE).String(), nil
}
