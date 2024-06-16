package mediaHost

import "github.com/ReidMason/plex-ani-sync/internal/request"

type MediaHost interface {
	Initialize(token string, host string, client request.HttpClient) (MediaHost, error)
	GetLibraries() ([]Library, error)
	GetCurrentUser() (User, error)
	GetSeries(libraryKey string) ([]Series, error)
	GetSeasons(seriesKey string) ([]Season, error)
	GetEpisodes(seasonKey string) ([]Episode, error)
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
	Id              string
	Title           string
	WatchedEpisodes int
	TotalEpisodes   int
}

type Season struct {
	Id       string
	Title    string
	Index    int
	Episodes int
	Year     int
}

type Episode struct {
	Id string
}
