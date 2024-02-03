package plex

import (
	"fmt"
	"testing"
)

func TestCreateForwardUrl(t *testing.T) {
	code := "testCode"
	clientIdentifier := "testClientIdentifier"
	appName := "testAppName"
	expected := "https://app.plex.tv/auth/#?clientID=" + clientIdentifier + "&code=" + code + "&context%5Bdevice%5D%5Bproduct%5D=" + appName

	result := CreateForwardUrl(code, clientIdentifier, appName)

	if result != expected {
		t.Errorf("CreateForwardUrl(%s) = %s; want %s", code, result, expected)
		fmt.Printf("result  : %v\n", result)
		fmt.Printf("expected: %v\n", expected)
	}
}

func TestBuildPollingLink(t *testing.T) {
	pinId := int64(123456)
	pinCode := "testCode"
	clientIdentifier := "testClientIdentifier"
	expected := "https://plex.tv/api/v2/pins/123456?X-Plex-Client-Identifier=testClientIdentifier&code=testCode"

	result := BuildPollingLink(pinId, pinCode, clientIdentifier)

	if result != expected {
		t.Errorf("BuildPollingLink(%d, %s, %s) = %s; want %s", pinId, pinCode, clientIdentifier, result, expected)
		fmt.Printf("result  : %v\n", result)
		fmt.Printf("expected: %v\n", expected)
	}
}
