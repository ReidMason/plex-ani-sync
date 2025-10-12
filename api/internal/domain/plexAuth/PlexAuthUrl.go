package plexAuth

import (
	"net/url"
	"strconv"
)

func BuildPlexAuthUrl(pin string, pinId int, clientIdentifier string, appName string, forwardUrl string) (PlexAuthURl, error) {
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
		return PlexAuthURl(""), err
	}
	query := forwardUrlPath.Query()
	query.Set("pinId", strconv.Itoa(pinId))
	forwardUrlPath.RawQuery = query.Encode()

	q.Set("forwardUrl", forwardUrlPath.String())

	return PlexAuthURl(u.String() + "#?" + q.Encode()), nil
}
