package api

import (
	"log/slog"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
)

func (s *Server) getSetupLibraries(c echo.Context) error {
	user, err := s.userManager.GetUser()
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

	selectedLibraries, err := s.userManager.GetSelectedLibraries(user.Id)
	if err != nil {
		slog.Error("Failed to get selected libraries", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get selected libraries")
	}

	selectedLibraryKeys := make([]string, 0, len(selectedLibraries))
	for _, library := range selectedLibraries {
		selectedLibraryKeys = append(selectedLibraryKeys, library.LibraryKey)
	}

	formData := views.SetupLibrariesFormData{
		SelectedLibraries: selectedLibraryKeys,
	}

	view := views.SetupLibraries(formData, filteredLibraries)
	return view.Render(c.Request().Context(), c.Response())
}

func (s *Server) postSetupLibraries(c echo.Context) error {
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

	user, err := s.userManager.GetUser()
	if err != nil {
		slog.Error("Failed to get user", slog.Any("error", err))
		c.Redirect(http.StatusFound, routes.SETUP_USER)
		return nil
	}

	err = s.userManager.AddSelectedLibraries(user.Id, selectedLibraries)
	if err != nil {
		slog.Error("Failed to add libraries", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to add libraries")
	}

	c.Response().Header().Set("HX-Redirect", routes.INDEX)
	return c.String(http.StatusOK, routes.INDEX)
}
