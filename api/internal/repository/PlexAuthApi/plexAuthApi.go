package plexAuthApi

import (
	"github.com/ReidMason/plex-ani-sync/internal/httpClient"
)

type PlexAuthApi struct {
	httpClient httpClient.HTTPClient
}

func NewPlexAuthApi(httpClient httpClient.HTTPClient) *PlexAuthApi {
	return &PlexAuthApi{
		httpClient: httpClient,
	}
}
