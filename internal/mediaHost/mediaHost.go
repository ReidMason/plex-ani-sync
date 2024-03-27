package mediaHost

import "net/http"

type MediaHost interface {
	Initialize(token string, host string, client HttpClient) error
	GetLibraries() ([]Library, error)
	GetCurrentUser() (PlexUser, error)
	GetSeries(libraryKey string) ([]PlexSeries, error)
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
