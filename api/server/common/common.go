package common

import (
	"net/http"
	"path"
)

const API_PREFIX = "/api"

func BuildEndpointUrl(endpoint string) string {
	return path.Join(API_PREFIX, endpoint)
}

type Controller interface {
	RegisterRoutes(mux *http.ServeMux)
}
