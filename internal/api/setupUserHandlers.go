package api

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ReidMason/plex-ani-sync/internal/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/ReidMason/plex-ani-sync/templates/components"
	"github.com/ReidMason/plex-ani-sync/templates/components/ui"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
)

func (s *Server) getSetupUser(c echo.Context) error {
	component := views.SetupUser(getDefaultSetupFormData())
	return component.Render(c.Request().Context(), c.Response())
}

func (s *Server) postSetupUser(c echo.Context) error {
	newFormData := extractSetupFormData(c)

	formData, validationPassed := validateSetupForm(newFormData)
	formData.FormSubmitted = "true"

	if !validationPassed {
		component := components.SetupUserFormContent(formData)
		return component.Render(c.Request().Context(), c.Response())
	}

	user, err := s.userManager.SetupUser(formData.Name.Value, formData.PlexUrl.Value, formData.HostUrl.Value)
	if err != nil {
		slog.Error("Failed to create user", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to create user")
	}

	forwardUrl, err := url.Parse(formData.HostUrl.Value)
	if err != nil {
		slog.Error("Failed to parse host url", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to parse host url")
	}

	forwardUrl.Path = routes.SETUP_PLEX_AUTH
	authUrl, err := mediaHost.GetPlexAuthUrl(forwardUrl.String(), user.ClientIdentifier, APP_NAME)
	if err != nil {
		slog.Error("Failed to authorize with Plex", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to authorize with Plex")
	}

	c.Response().Header().Set("HX-Redirect", authUrl)
	return c.String(http.StatusOK, authUrl)
}

func (s *Server) postSetupUserValidate(c echo.Context) error {
	newFormData := extractSetupFormData(c)
	formData, _ := validateSetupForm(newFormData)

	component := components.SetupUserFormContent(formData)
	return component.Render(c.Request().Context(), c.Response())
}

func getDefaultSetupFormData() components.SetupUserFormData {
	return components.SetupUserFormData{
		FormSubmitted: "false",
		Name: ui.Field{
			Name:          "name",
			Label:         "Name",
			Placeholder:   "Enter your name",
			Valid:         true,
			ValidateRoute: routes.SETUP_USER_VALIDATE,
		},
		HostUrl: ui.Field{
			Name:          "hostUrl",
			Label:         "Host url",
			Placeholder:   "Enter your PlexAnilistSync host url",
			Valid:         true,
			ValidateRoute: routes.SETUP_USER_VALIDATE,
		},
		PlexUrl: ui.Field{
			Name:          "plexUrl",
			Label:         "Plex URL",
			Placeholder:   "Enter your Plex URL",
			Valid:         true,
			ValidateRoute: routes.SETUP_USER_VALIDATE,
		},
	}
}

func validateSetupForm(formData components.SetupUserFormData) (components.SetupUserFormData, bool) {
	validationPassed := true

	if valid, msg := storage.ValidateName(formData.Name.Value); !valid {
		validationPassed = false
		formData.Name.Valid = false
		formData.Name.Error = msg
	}

	if valid, msg := storage.ValidatePlexUrl(formData.PlexUrl.Value); !valid {
		validationPassed = false
		formData.PlexUrl.Valid = false
		formData.PlexUrl.Error = msg
	}

	if valid, msg := storage.ValidateHostUrl(formData.HostUrl.Value); !valid {
		validationPassed = false
		formData.HostUrl.Valid = false
		formData.HostUrl.Error = msg
	}

	return formData, validationPassed
}

func extractSetupFormData(c echo.Context) components.SetupUserFormData {
	formData := getDefaultSetupFormData()

	formData.FormSubmitted = c.FormValue("formSubmitted")
	formData.Name.Value = c.FormValue(formData.Name.Name)
	formData.HostUrl.Value = c.FormValue(formData.HostUrl.Name)
	formData.PlexUrl.Value = c.FormValue(formData.PlexUrl.Name)

	return formData
}
