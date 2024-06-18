package api

import (
	"log/slog"
	"net/http"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	api "github.com/ReidMason/plex-ani-sync/internal/api/index"
	"github.com/ReidMason/plex-ani-sync/internal/api/routes"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/templates/views"
	"github.com/labstack/echo/v4"
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
				indexData.FullList = watchList
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
