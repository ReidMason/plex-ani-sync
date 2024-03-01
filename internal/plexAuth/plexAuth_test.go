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
		{
			clientIdentifier: "testClientIdentifier1",
			appName:          "testAppName1",
			expected:         "https://plex.tv/api/v2/pins?X-Plex-Client-Identifier=testClientIdentifier1&X-Plex-Product=testAppName1&strong=true",
		},
		{
			clientIdentifier: "testClientIdentifier2",
			appName:          "testAppName2",
			expected:         "https://plex.tv/api/v2/pins?X-Plex-Client-Identifier=testClientIdentifier2&X-Plex-Product=testAppName2&strong=true",
		},
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
		hostUrl          string
		code             string
		clientIdentifier string
		appName          string
		expected         string
		pinId            int64
	}{
		{
			pinId:            123,
			hostUrl:          "http://site/",
			code:             "testCode1",
			clientIdentifier: "ci1",
			appName:          "testAppName1",
			expected:         "https://app.plex.tv/auth/#?clientID=ci1&code=testCode1&context%5Bdevice%5D%5Bproduct%5D=testAppName1&forwardUrl=http%3A%2F%2Fsite%2F%3FclientIdentifier%3Dci1%26code%3DtestCode1%26pinid%3D123",
		},
		{
			pinId:            123,
			hostUrl:          "http://site",
			code:             "testCode2",
			clientIdentifier: "ci2",
			appName:          "testAppName2",
			expected:         "https://app.plex.tv/auth/#?clientID=ci2&code=testCode2&context%5Bdevice%5D%5Bproduct%5D=testAppName2&forwardUrl=http%3A%2F%2Fsite%3FclientIdentifier%3Dci2%26code%3DtestCode2%26pinid%3D123",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("BuildAuthUrl(%s, %s, %s)", tc.code, tc.clientIdentifier, tc.appName), func(t *testing.T) {
			t.Parallel()

			result, err := buildAuthUrl(tc.hostUrl, tc.pinId, tc.code, tc.clientIdentifier, tc.appName)

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
		pinId            int
	}{
		{
			pinCode:          "testCode1",
			clientIdentifier: "testClientIdentifier1",
			expected:         "https://plex.tv/api/v2/pins/123456?X-Plex-Client-Identifier=testClientIdentifier1&code=testCode1",
			pinId:            123456,
		},
		{
			pinCode:          "testCode2",
			clientIdentifier: "testClientIdentifier2",
			expected:         "https://plex.tv/api/v2/pins/123456?X-Plex-Client-Identifier=testClientIdentifier2&code=testCode2",
			pinId:            123456,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("BuildPollingLink(%d, %s, %s)", tc.pinId, tc.pinCode, tc.clientIdentifier), func(t *testing.T) {
			t.Parallel()

			result, err := BuildAuthTokenPollingLink(tc.pinId, tc.pinCode, tc.clientIdentifier)

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
