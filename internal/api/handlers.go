package api

import (
	"log/slog"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	api "github.com/ReidMason/plex-ani-sync/internal/api/index"
	"github.com/ReidMason/plex-ani-sync/internal/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mapping"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
)

func (s *Server) getIndex(c echo.Context) error {
	indexData := api.IndexData{
		Libraries: make([]string, 0),
	}

	// Redirect to setup if user doesn't exist
	user, err := s.userManager.GetUser()
	if err != nil {
		s.log.Warn("Failed to find existing user redirecting to setup", slog.Any("error", err))
		c.Redirect(http.StatusFound, routes.SETUP_USER)
		return nil
	}

	indexData.Name = user.Name

	if user.AnimeListToken != nil {
		animeListUser, err := s.animeList.GetCurrentUser(*user.AnimeListToken)
		if err != nil {
			s.log.Error("Failed to get anime list user", slog.Any("error", err))
		} else {
			watchList, err := s.animeList.GetAnimeList(animeListUser.Id)
			if err != nil {
				s.log.Error("Failed to get anime list", slog.Any("error", err))
			} else {
				for _, entry := range watchList {
					switch entry.Status {
					case animeList.Current:
						indexData.ListData.Watching++
					case animeList.Planning:
						indexData.ListData.Planning++
					case animeList.Completed:
						indexData.ListData.Completed++
					case animeList.Dropped:
						indexData.ListData.Dropped++
					case animeList.Paused:
						indexData.ListData.Paused++
					}
				}
			}
		}
	}

	// user, err := s.mediaHost.GetCurrentUser()
	// if err != nil {
	// 	s.log.Error("Failed to get current user from media host", slog.Any("error", err))
	// 	return c.String(http.StatusInternalServerError, "Failed to get current user from media host")
	// }

	libraries, err := s.mediaHost.GetLibraries()
	if err != nil {
		libraries = make([]mediaHost.Library, 0)
	}

	for _, library := range libraries {
		for _, userLibrary := range user.Libraries {
			if library.Key == userLibrary.LibraryKey {
				indexData.Libraries = append(indexData.Libraries, library.Title)
			}
		}
	}

	component := views.Index(indexData)
	return component.Render(c.Request().Context(), c.Response())
}

func (s *Server) startMapping(c echo.Context) error {
	s.log.Info("Starting mapping")

	user, err := s.userManager.GetUser()
	if err != nil {
		s.log.Error("Failed to get user", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get user")
	}

	libraries, err := s.mediaHost.GetLibraries()
	if err != nil {
		s.log.Error("Failed to get libraries from media host", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "Failed to get libraries from media host")
	}

	selectedLibraries := make([]mediaHost.Library, 0)
	for _, library := range libraries {
		if slices.ContainsFunc(user.Libraries, func(userLibrary storage.Library) bool {
			return userLibrary.LibraryKey == library.Key
		}) {
			selectedLibraries = append(selectedLibraries, library)
		}
	}

	mappingsFound := 0
	mappingsFailed := 0

	for _, library := range selectedLibraries {
		s.log.Info("Mapping library", slog.String("library", library.Title))
		allSeries, err := s.mediaHost.GetSeries(library.Key)
		if err != nil {
			s.log.Error("Failed to get series from media host", slog.Any("error", err))
			continue
		}

		for _, series := range allSeries {
			s.log.Info("Mapping series", slog.String("series", series.Title))
			seriesSeasons, err := s.mediaHost.GetSeasons(series.Id)
			if err != nil {
				s.log.Error("Failed to get seasons from media host", slog.Any("error", err))
				continue
			}

			seasons := make([]mapping.Season, 0)
			for _, season := range seriesSeasons {
				seasons = append(seasons, mapping.Season{
					Id:          season.Id,
					Episodes:    season.Episodes,
					ReleaseYear: season.Year,
				})
			}

			mappings, err := s.mappingFinder.CreateMappingsForSeasons(series.Title, seasons)
			if err != nil {
				s.log.Error("Failed to find mapping", slog.Any("error", err))
			}

			if len(mappings) > 0 {
				mappingsFound++
			} else {
				mappingsFailed++
			}

			s.mappingStorage.SetMappings(mappings)
		}
	}

	s.log.Info("Mapping complete", slog.Int("mappingsFound", mappingsFound), slog.Int("mappingsFailed", mappingsFailed))

	return nil
}

func (s *Server) startSyncDryRun(c echo.Context) error {
	s.log.Info("Starting sync dry run")
	s.SyncService.Sync()
	s.log.Info("Sync dry run complete")

	return nil
}
