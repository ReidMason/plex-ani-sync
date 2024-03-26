package api

import (
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ReidMason/plex-ani-sync/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/templates/components"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
)

func (s *Server) handlePostUser(c echo.Context) error {
	_, err := s.store.GetUser()
	if err == nil {
		s.store.DeleteUser()
	}

	newFormData := extractSetupFormData(c)
	formData, validationPassed := validateSetupForm(newFormData)
	formData.FormSubmitted = "true"

	if !validationPassed {
		component := views.SetupFormContent(formData)
		return component.Render(c.Request().Context(), c.Response())
	}

	user, err := s.store.CreateUser(formData.Name.Value, formData.PlexUrl.Value, formData.HostUrl.Value)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create user")
	}

	forwardUrl, err := url.Parse(formData.HostUrl.Value)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to parse host url")
	}

	forwardUrl.Path = routes.SETUP_PLEX_AUTH
	authUrl, err := mediaHost.GetPlexAuthUrl(forwardUrl.String(), user.ClientIdentifier, APP_NAME)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to authorize with Plex")
	}

	c.Response().Header().Set("HX-Redirect", authUrl)
	return c.String(http.StatusOK, authUrl)
}

func (s *Server) handlePlexAuth(c echo.Context) error {
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

func (s *Server) handleGetRoot(c echo.Context) error {
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

	component := views.Home(user)
	return component.Render(c.Request().Context(), c.Response())
}

func (s *Server) handleGetSetupUser(c echo.Context) error {
	component := views.Setup(getDefaultSetupFormData())
	return component.Render(c.Request().Context(), c.Response())
}

func (s *Server) handleValidateSetupForm(c echo.Context) error {
	newFormData := extractSetupFormData(c)
	formData, _ := validateSetupForm(newFormData)

	component := views.SetupFormContent(formData)
	return component.Render(c.Request().Context(), c.Response())
}

func (s *Server) handleSetupLibraries(c echo.Context) error {
	_, err := s.store.GetUser()
	if err != nil {
		slog.Error("Failed to get user", slog.Any("error", err))
		c.Redirect(http.StatusFound, routes.SETUP_USER)
		return nil
	}

	libraries, err := s.mediaHost.GetLibraries()
	if err != nil {
		slog.Error("Failed to get libraries", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get libraries")
	}

	filteredLibraries := make([]mediaHost.Library, 0)
	for _, library := range libraries {
		if library.Type == "show" {
			filteredLibraries = append(filteredLibraries, library)
		}
	}

	formData := views.SetupLibrariesFormData{
		SelectedLibraries: []string{},
	}
	view := views.LibrarySelector(formData, filteredLibraries)
	return view.Render(c.Request().Context(), c.Response())
}

func (s *Server) postLibraries(c echo.Context) error {
	data, err := c.FormParams()
	if err != nil {
		slog.Error("Failed to get form params", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get form params")
	}

	selectedLibraries := make([]string, 0, len(data))
	for key := range data {
		selectedLibraries = append(selectedLibraries, key)
	}

	slog.Info("Selected libraries", slog.Any("libraries", selectedLibraries))

	user, err := s.store.GetUser()
	if err != nil {
		slog.Error("Failed to get user", slog.Any("error", err))
		c.Redirect(http.StatusFound, routes.SETUP_USER)
		return nil
	}

	err = s.store.AddLibraries(user.Id, selectedLibraries)
	if err != nil {
		slog.Error("Failed to add libraries", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to add libraries")
	}

	c.Response().Header().Set("HX-Redirect", routes.HOME)
	return c.String(http.StatusOK, routes.HOME)
}

func getDefaultSetupFormData() views.FormData {
	return views.FormData{
		FormSubmitted: "false",
		Name: components.Field{
			Name:          "name",
			Label:         "Name",
			Placeholder:   "Enter your name",
			Valid:         true,
			ValidateRoute: routes.SETUP_VALIDATE,
		},
		HostUrl: components.Field{
			Name:          "hostUrl",
			Label:         "Host url",
			Placeholder:   "Enter your PlexAnilistSync host url",
			Valid:         true,
			ValidateRoute: routes.SETUP_VALIDATE,
		},
		PlexUrl: components.Field{
			Name:          "plexUrl",
			Label:         "Plex URL",
			Placeholder:   "Enter your Plex URL",
			Valid:         true,
			ValidateRoute: routes.SETUP_VALIDATE,
		},
	}
}

func validateName(name string) (bool, string) {
	if name == "" {
		return false, "Name is required"
	}

	return true, ""
}

func validateHostUrl(hostUrl string) (bool, string) {
	if hostUrl == "" {
		return false, "A host URL is required"
	}

	_, err := url.ParseRequestURI(hostUrl)
	if err != nil {
		return false, "Host URL is invalid"
	}

	return true, ""
}

func validatePlexUrl(plexUrl string) (bool, string) {
	if plexUrl == "" {
		return false, "Plex URL is required"
	}

	_, err := url.ParseRequestURI(plexUrl)
	if err != nil {
		return false, "Plex URL is invalid"
	}

	return true, ""
}

func validateSetupForm(formData views.FormData) (views.FormData, bool) {
	validationPassed := true

	if valid, msg := validateName(formData.Name.Value); !valid {
		validationPassed = false
		formData.Name.Valid = false
		formData.Name.Error = msg
	}

	if valid, msg := validatePlexUrl(formData.PlexUrl.Value); !valid {
		validationPassed = false
		formData.PlexUrl.Valid = false
		formData.PlexUrl.Error = msg
	}

	if valid, msg := validateHostUrl(formData.HostUrl.Value); !valid {
		validationPassed = false
		formData.HostUrl.Valid = false
		formData.HostUrl.Error = msg
	}

	return formData, validationPassed
}

func extractSetupFormData(c echo.Context) views.FormData {
	formData := getDefaultSetupFormData()

	formData.FormSubmitted = c.FormValue("formSubmitted")
	formData.Name.Value = c.FormValue(formData.Name.Name)
	formData.HostUrl.Value = c.FormValue(formData.HostUrl.Name)
	formData.PlexUrl.Value = c.FormValue(formData.PlexUrl.Name)

	return formData
}
