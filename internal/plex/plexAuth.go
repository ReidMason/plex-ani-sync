package plex

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const PLEX_BASE_URL = "https://plex.tv"
const PLEX_APP_BASE_URL = "https://app.plex.tv"

func GetPlexAuthUrl(forwardUrl, client_identifier, app_name string) (string, error) {
	requestUrl, err := buildAuthRequestUrl(client_identifier, app_name)
	if err != nil {
		return "", err
	}

	authData, err := getAuthData(requestUrl)
	if err != nil {
		return "", err
	}

	authUrl, err := buildAuthUrl(forwardUrl, authData.Id, authData.Code, client_identifier, app_name)
	if err != nil {
		return "", err
	}

	return authUrl, nil
}

func buildAuthRequestUrl(clientIdentifier, appName string) (string, error) {
	req, err := http.NewRequest("POST", PLEX_BASE_URL+"/api/v2/pins", nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("X-Plex-Client-Identifier", clientIdentifier)
	q.Add("X-Plex-Product", appName)
	q.Add("strong", "true")
	req.URL.RawQuery = q.Encode()

	return req.URL.String(), nil
}

func getAuthData(authRequestUrl string) (authResponse, error) {
	var result authResponse

	req, err := http.NewRequest("POST", authRequestUrl, nil)
	if err != nil {
		return result, err
	}

	req.Header.Add("accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}

	return result, nil
}

func buildAuthUrl(forwardUrl string, pinId int64, code, clientIdentifier, appName string) (string, error) {
	req, err := http.NewRequest("GET", PLEX_APP_BASE_URL+"/auth/", nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("clientID", clientIdentifier)
	q.Add("code", code)
	q.Add("context[device][product]", appName)

	u, err := url.Parse(forwardUrl)
	if err != nil {
		return "", err
	}
	forwardUrlQuery := u.Query()
	forwardUrlQuery.Add("pinid", fmt.Sprint(pinId))
	forwardUrlQuery.Add("code", code)
	forwardUrlQuery.Add("clientIdentifier", clientIdentifier)
	u.RawQuery = forwardUrlQuery.Encode()
	q.Add("forwardUrl", u.String())

	return req.URL.String() + "#?" + q.Encode(), nil
}

func BuildAuthTokenPollingLink(pinId int, pinCode, clientIdentifier string) (string, error) {
	req, err := http.NewRequest("GET", PLEX_BASE_URL+"/api/v2/pins/"+fmt.Sprint(pinId), nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("code", pinCode)
	q.Add("X-Plex-Client-Identifier", clientIdentifier)

	req.URL.RawQuery = q.Encode()

	return req.URL.String(), nil
}

func PollForAuthToken(pollingLink string) (authResponse, error) {
	var result authResponse

	req, err := http.NewRequest("GET", pollingLink, nil)
	if err != nil {
		return result, err
	}

	req.Header.Add("accept", "application/json")

	iterLimit := 60
	for i := 0; i < iterLimit; i++ {
		time.Sleep(1 * time.Second)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		if result.AuthToken != nil {
			return result, nil
		}
	}

	return result, errors.New("Failed to get auth token")
}

type location struct {
	code                       string
	continentCode              string
	country                    string
	city                       string
	timeZone                   string
	postalCode                 string
	subdivisions               string
	coordinates                string
	europeanUnionMember        bool
	inPrivacyRestrictedCountry bool
}

type authResponse struct {
	AuthToken        *string
	newRegistration  *string
	Code             string
	product          string
	qr               string
	clientIdentifier string
	createdAt        string
	expiresAt        string
	location         location
	Id               int64
	expiresIn        int
	trusted          bool
}
