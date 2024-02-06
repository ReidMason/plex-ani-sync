package plexAuth

import (
	"fmt"
	"testing"
)

func TestBuildAuthRequestUrl(t *testing.T) {
	testCases := []struct {
		clientIdentifier string
		appName          string
		expected         string
	}{
		{"testClientIdentifier1", "testAppName1", "https://plex.tv/api/v2/pins?X-Plex-Client-Identifier=testClientIdentifier1&X-Plex-Product=testAppName1&strong=true"},
		{"testClientIdentifier2", "testAppName2", "https://plex.tv/api/v2/pins?X-Plex-Client-Identifier=testClientIdentifier2&X-Plex-Product=testAppName2&strong=true"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("BuildAuthRequestUrl(%s, %s)", tc.clientIdentifier, tc.appName), func(t *testing.T) {
			t.Parallel()

			result, err := buildAuthRequestUrl(tc.clientIdentifier, tc.appName)
			if err != nil {
				t.Errorf("BuildAuthRequestUrl(%s, %s) threw an error", tc.clientIdentifier, tc.appName)
			}

			if result != tc.expected {
				t.Errorf("BuildAuthRequestUrl(%s, %s) = %s; want %s", tc.clientIdentifier, tc.appName, result, tc.expected)
				fmt.Printf("result  : %v\n", result)
				fmt.Printf("expected: %v\n", tc.expected)
			}
		})
	}
}

func TestBuildAuthUrl(t *testing.T) {
	testCases := []struct {
		code             string
		clientIdentifier string
		appName          string
		expected         string
	}{
		{"testCode1", "testClientIdentifier1", "testAppName1", "https://app.plex.tv/auth/#?clientID=testClientIdentifier1&code=testCode1&context%5Bdevice%5D%5Bproduct%5D=testAppName1"},
		{"testCode2", "testClientIdentifier2", "testAppName2", "https://app.plex.tv/auth/#?clientID=testClientIdentifier2&code=testCode2&context%5Bdevice%5D%5Bproduct%5D=testAppName2"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("BuildAuthUrl(%s, %s, %s)", tc.code, tc.clientIdentifier, tc.appName), func(t *testing.T) {
			t.Parallel()

			result, err := buildAuthUrl(tc.code, tc.clientIdentifier, tc.appName)

			if err != nil {
				t.Errorf("BuildAuthUrl(%s, %s, %s) threw an error", tc.code, tc.clientIdentifier, tc.appName)
			}

			if result != tc.expected {
				t.Errorf("BuildAuthUrl(%s, %s, %s) = %s; want %s", tc.code, tc.clientIdentifier, tc.appName, result, tc.expected)
				fmt.Printf("result  : %v\n", result)
				fmt.Printf("expected: %v\n", tc.expected)
			}
		})
	}
}

func TestBuildPollingLink(t *testing.T) {
	testCases := []struct {
		pinCode          string
		clientIdentifier string
		expected         string
		pinId            int64
	}{
		{"testCode1", "testClientIdentifier1", "https://plex.tv/api/v2/pins/123456?X-Plex-Client-Identifier=testClientIdentifier1&code=testCode1", 123456},
		{"testCode2", "testClientIdentifier2", "https://plex.tv/api/v2/pins/123456?X-Plex-Client-Identifier=testClientIdentifier2&code=testCode2", 123456},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("BuildPollingLink(%d, %s, %s)", tc.pinId, tc.pinCode, tc.clientIdentifier), func(t *testing.T) {
			t.Parallel()

			result, err := buildPollingLink(tc.pinId, tc.pinCode, tc.clientIdentifier)

			if err != nil {
				t.Errorf("BuildPollingLink(%d, %s, %s) threw an error", tc.pinId, tc.pinCode, tc.clientIdentifier)
			}

			if result != tc.expected {
				t.Errorf("BuildPollingLink(%d, %s, %s) = %s; want %s", tc.pinId, tc.pinCode, tc.clientIdentifier, result, tc.expected)
				fmt.Printf("result  : %v\n", result)
				fmt.Printf("expected: %v\n", tc.expected)
			}
		})
	}
}
