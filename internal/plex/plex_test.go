package plex

import (
	"fmt"
	"testing"
)

func TestBuildAuthRequestUrl(t *testing.T) {
	clientIdentifier := "testClientIdentifier"
	appName := "testAppName"
	expected := "https://plex.tv/api/v2/pins?X-Plex-Client-Identifier=testClientIdentifier&X-Plex-Product=testAppName&strong=true"

	result, err := BuildAuthRequestUrl(clientIdentifier, appName)
	if err != nil {
		t.Errorf("BuildAuthRequestUrl(%s, %s) threw an error", clientIdentifier, appName)
	}

	if result != expected {
		t.Errorf("BuildAuthRequestUrl(%s, %s) = %s; want %s", clientIdentifier, appName, result, expected)
		fmt.Printf("result  : %v\n", result)
		fmt.Printf("expected: %v\n", expected)
	}
}

func TestCreateForwardUrl(t *testing.T) {
	code := "testCode"
	clientIdentifier := "testClientIdentifier"
	appName := "testAppName"
	expected := "https://app.plex.tv/auth/#?clientID=" + clientIdentifier + "&code=" + code + "&context%5Bdevice%5D%5Bproduct%5D=" + appName

	result, err := BuildAuthUrl(code, clientIdentifier, appName)

	if err != nil {
		t.Errorf("BuildAuthUrl(%s, %s, %s) threw an error", code, clientIdentifier, appName)
	}

	if result != expected {
		t.Errorf("BuildAuthUrl(%s, %s, %s) = %s; want %s", code, clientIdentifier, appName, result, expected)
		fmt.Printf("result  : %v\n", result)
		fmt.Printf("expected: %v\n", expected)
	}
}

func TestBuildPollingLink(t *testing.T) {
	pinId := int64(123456)
	pinCode := "testCode"
	clientIdentifier := "testClientIdentifier"
	expected := "https://plex.tv/api/v2/pins/123456?X-Plex-Client-Identifier=testClientIdentifier&code=testCode"

	result, err := BuildPollingLink(pinId, pinCode, clientIdentifier)

	if err != nil {
		t.Errorf("BuildPollingLink(%d, %s, %s) threw an error", pinId, pinCode, clientIdentifier)
	}

	if result != expected {
		t.Errorf("BuildPollingLink(%d, %s, %s) = %s; want %s", pinId, pinCode, clientIdentifier, result, expected)
		fmt.Printf("result  : %v\n", result)
		fmt.Printf("expected: %v\n", expected)
	}
}
