package plex

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func GenerateAuthCode(clientIdentifier, appName string) pinResponse {
	req, err := http.NewRequest("POST", "https://plex.tv/api/v2/pins?strong=true", nil)
	if err != nil {
		log.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("X-Plex-Client-Identifier", clientIdentifier)
	q.Add("X-Plex-Product", appName)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result pinResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatal(err)
	}

	return result
}

func CreateForwardUrl(code, clientIdentifier, appName string) string {
	req, err := http.NewRequest("GET", "https://app.plex.tv/auth/", nil)
	if err != nil {
		log.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("clientID", clientIdentifier)
	q.Add("code", code)
	q.Add("context[device][product]", appName)
	// q.Add("forwardUrl", "https://app.plex.tv/auth/forward")

	return req.URL.String() + "#?" + q.Encode()
}

func BuildPollingLink(pinId int64, pinCode, clientIdentifier string) string {

	req, err := http.NewRequest("GET", "https://plex.tv/api/v2/pins/"+fmt.Sprint(pinId), nil)
	if err != nil {
		log.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("code", pinCode)
	q.Add("X-Plex-Client-Identifier", clientIdentifier)

	req.URL.RawQuery = q.Encode()

	return req.URL.String()
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

type pinResponse struct {
	authToken        *string
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