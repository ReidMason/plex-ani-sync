package mediaHost

import "github.com/ReidMason/plex-ani-sync/internal/request"

type MediaHost interface {
	Initialize(token string, host string, client request.HttpClient) error
	GetLibraries() ([]Library, error)
	GetCurrentUser() (PlexUser, error)
	GetSeries(libraryKey string) ([]PlexSeries, error)
	GetSeasons(seriesKey string) ([]PlexSeason, error)
	GetEpisodes(seasonKey string) ([]PlexEpisode, error)
}

type Library struct {
	Key   string
	Title string
	Type  string
}
