package plexAuth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const PLEX_BASE_URL = "https://plex.tv"
const PLEX_APP_BASE_URL = "https://app.plex.tv"

func AuthPlex(client_identifier, app_name string) (authResponse, error) {
	var response authResponse

	log.Print("Authenticating with Plex")
	requestUrl, err := buildAuthRequestUrl(client_identifier, app_name)
	if err != nil {
		return response, err
	}

	authData, err := getAuthData(requestUrl)
	if err != nil {
		return response, err
	}

	authUrl, err := buildAuthUrl(authData.Code, client_identifier, app_name)
	if err != nil {
		return response, err
	}

	log.Printf("Visit this URL to authenticate: %v", authUrl)

	pollingUrl, err := buildPollingLink(authData.Id, authData.Code, client_identifier)
	if err != nil {
		return response, err
	}

	log.Print("Polling for authentication")
	return pollForAuthToken(pollingUrl)
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

func buildAuthUrl(code, clientIdentifier, appName string) (string, error) {
	req, err := http.NewRequest("GET", PLEX_APP_BASE_URL+"/auth/", nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("clientID", clientIdentifier)
	q.Add("code", code)
	q.Add("context[device][product]", appName)
	// q.Add("forwardUrl", "https://app.plex.tv/auth/forward")

	return req.URL.String() + "#?" + q.Encode(), nil
}

func buildPollingLink(pinId int64, pinCode, clientIdentifier string) (string, error) {
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

func pollForAuthToken(pollingLink string) (authResponse, error) {
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
