package plex

import (
	"net/url"
	"strconv"
)

func (p *PlexAuth) GetPlexAuthUrl(forwardUrl string) (string, error) {
	pin, err := p.generatePin()
	if err != nil {
		return "", err
	}

	authURL, err := buildAuthURL(pin.Code, pin.ID, p.clientIdentifier, p.appName, forwardUrl)
	if err != nil {
		return "", err
	}

	return authURL, nil
}

func buildAuthURL(pin string, pinId int, clientIdentifier string, appName string, forwardUrl string) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "app.plex.tv",
		Path:   "/auth",
	}
	q := u.Query()
	q.Set("clientID", clientIdentifier)
	q.Set("code", pin)
	q.Set("context[device][product]", appName)

	forwardUrlPath, err := url.Parse(forwardUrl)
	if err != nil {
		return "", err
	}
	query := forwardUrlPath.Query()
	query.Set("pinId", strconv.Itoa(pinId))
	forwardUrlPath.RawQuery = query.Encode()

	q.Set("forwardUrl", forwardUrlPath.String())

	return u.String() + "#?" + q.Encode(), nil
}
