package mediaHost

import "github.com/ReidMason/plex-ani-sync/internal/request"

type MediaHost interface {
	Initialize(token string, host string, client request.HttpClient) error
	GetLibraries() ([]Library, error)
	GetCurrentUser() (User, error)
	GetSeasons(seriesKey string) ([]PlexSeason, error)
	GetEpisodes(seasonKey string) ([]PlexEpisode, error)
	GetSeries(libraryKey string) ([]Series, error)
}

type Library struct {
	Key   string
	Title string
	Type  string
}

type User struct {
	Username string
}

type Series struct {
	Id string
}
