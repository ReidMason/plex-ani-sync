package plex

import (
	"github.com/ReidMason/plex-ani-sync/internal/httpClient"
)

type PlexAuth struct {
	clientIdentifier string
	appName          string
	httpClient       httpClient.HTTPClient
}

func NewPlexAuth(clientIdentifier string, appName string, httpClient httpClient.HTTPClient) *PlexAuth {
	return &PlexAuth{
		clientIdentifier: clientIdentifier,
		appName:          appName,
		httpClient:       httpClient,
	}
}
