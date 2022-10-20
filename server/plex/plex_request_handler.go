package plex

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"plex-ani-sync/config"
	"plex-ani-sync/requesthandler"
)

type RequestHandler struct {
	ConfigHandler config.IConfigHandler
}

func NewRequestHandler(configHandler config.IConfigHandler) *RequestHandler {
	return &RequestHandler{ConfigHandler: configHandler}
}

var _ requesthandler.IRequestHandler = (*RequestHandler)(nil)

func (prh RequestHandler) MakeRequest(method, endpoint string) (string, error) {
	client := http.Client{}
	plexUrl, err := prh.getPlexUrl(endpoint)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(method, plexUrl, nil)
	if err != nil {
		return "", requesthandler.HttpRequestError{Err: err}
	}

	// Add headers
	req.Header = http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// Extract response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (prh RequestHandler) getPlexUrl(endpoint string) (string, error) {
	cfg, _ := prh.ConfigHandler.GetConfig()
	plexUrl, err := url.Parse(cfg.Plex.BaseUrl)
	if err != nil {
		return "", err
	}

	// Add token query param
	q := plexUrl.Query()
	q.Set("X-Plex-Token", cfg.Plex.Token)
	plexUrl.RawQuery = q.Encode()

	// Add requested path
	plexUrl.Path = path.Join(plexUrl.Path, endpoint)

	return plexUrl.String(), nil
}
