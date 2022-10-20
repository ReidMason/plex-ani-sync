package plex

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"plex-ani-sync/config"
	"plex-ani-sync/requesthandler"
	"reflect"
	"testing"
)

func TestGetPlexUrl(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, Endpoint, ExpectedUrl string
		MockConfig                  config.Config
	}{
		{"No trailing or leading slashes", "endpoint", "http://testing/endpoint?X-Plex-Token=testToken123",
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing",
					Token:   "testToken123",
				},
			},
		},
		{"Leading slashes", "/endpoint", "http://testing/endpoint?X-Plex-Token=testToken123",
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing",
					Token:   "testToken123",
				},
			},
		},
		{"Trailing slashes", "endpoint/", "http://testing/endpoint?X-Plex-Token=testToken123",
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing/",
					Token:   "testToken123",
				},
			},
		},
		{"Trailing and leading slashes", "/endpoint/", "http://testing/endpoint?X-Plex-Token=testToken123",
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing/",
					Token:   "testToken123",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			plexRequestHandler := NewRequestHandler(config.NewMock(tc.MockConfig))

			plexUrl, err := plexRequestHandler.getPlexUrl(tc.Endpoint)
			errored := err != nil

			gotExpectedResult := plexUrl == tc.ExpectedUrl

			if errored {
				t.Errorf("Unexpected error: '%s'", err)
			} else if !gotExpectedResult {
				t.Errorf("Got wrong URL. Expected: '%s' found: '%s'", tc.ExpectedUrl, plexUrl)
			} else if gotExpectedResult {
				t.Log("Got expected URL")
			}

		})
	}
}

func TestGetPlexUrlWithInvalidBaseUrl(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, Endpoint string
		ExpectedError  error
		MockConfig     config.Config
	}{
		{"Leading space", "endpoint", &url.Error{},
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: " http://testing",
					Token:   "testToken123",
				},
			},
		},
		{"Invalid character '`'", "endpoint", &url.Error{},
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing`",
					Token:   "testToken123",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexRequestHandler := NewRequestHandler(config.NewMock(tc.MockConfig))

			plexUrl, err := plexRequestHandler.getPlexUrl(tc.Endpoint)
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType
			expectedResult := ""
			gotExpectedResult := plexUrl == expectedResult

			if !gotExpectedError {
				t.Errorf("Got unexpected error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Got wrong URL. Expected: '%s' found: '%s'", expectedResult, plexUrl)
			} else if gotExpectedResult && gotExpectedError {
				t.Log("Got expected URL and error")
			} else {
				t.Error("Test failed unexpectedly")
			}

		})
	}
}

func TestMakeRequestWithPlexUrlParsingError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name          string
		ExpectedError error
		MockConfig    config.Config
	}{
		{"Invlaid plex base url", &url.Error{},
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: " http://testing",
					Token:   "testToken123",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexRequestHandler := NewRequestHandler(config.NewMock(tc.MockConfig))

			response, err := plexRequestHandler.MakeRequest("GET", "/test")
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			expectedResult := ""
			gotExpectedResult := response == expectedResult

			if !gotExpectedError {
				t.Errorf("Got unexpected error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Got wrong response. Expected: '%s' found: '%s'", expectedResult, response)
			} else if gotExpectedResult && gotExpectedError {
				t.Log("Got expected response and error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}

}

func TestMakeRequestWithRequestBuilderError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, RequestMethod string
		ExpectedError       error
		MockConfig          config.Config
	}{
		{"Invalid request method", "bad method", requesthandler.HttpRequestError{},
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing",
					Token:   "testToken123",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexRequestHandler := NewRequestHandler(config.NewMock(tc.MockConfig))

			response, err := plexRequestHandler.MakeRequest(tc.RequestMethod, "/test")
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			expectedResult := ""
			gotExpectedResult := response == expectedResult

			if !gotExpectedError {
				t.Errorf("Got unexpected error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Got wrong response. Expected: '%s' found: '%s'", expectedResult, response)
			} else if gotExpectedResult && gotExpectedError {
				t.Log("Got expected response and error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}

}

func TestMakeRequestWithRequestError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, RequestMethod string
		ExpectedError       error
		MockConfig          config.Config
	}{
		{"Invalid endpoint", "GET", &url.Error{},
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing",
					Token:   "testToken123",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexRequestHandler := NewRequestHandler(config.NewMock(tc.MockConfig))

			response, err := plexRequestHandler.MakeRequest(tc.RequestMethod, "/test")
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			expectedResult := ""
			gotExpectedResult := response == expectedResult

			if !gotExpectedError {
				t.Errorf("Got unexpected error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Got wrong response. Expected: '%s' found: '%s'", expectedResult, response)
			} else if gotExpectedResult && gotExpectedError {
				t.Log("Got expected response and error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}

}

func TestMakeRequest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, RequestMethod string
		MockConfig          config.Config
	}{
		{"Valid endpoint", "GET",
			config.Config{
				Plex: config.PlexConfig{
					BaseUrl: "http://testing",
					Token:   "testToken123",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value":"fixed"}`))
	}))

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			// Update the conection URL for testing
			tc.MockConfig.Plex.BaseUrl = server.URL

			plexRequestHandler := NewRequestHandler(config.NewMock(tc.MockConfig))

			response, err := plexRequestHandler.MakeRequest(tc.RequestMethod, "/test")
			errored := err != nil
			errorType := reflect.TypeOf(err)

			expectedResult := `{"value":"fixed"}`
			gotExpectedResult := response == expectedResult

			if errored {
				t.Errorf("Got unexpected error. Found: '%s'", errorType)
			} else if !gotExpectedResult {
				t.Errorf("Got wrong response. Expected: '%s' found: '%s'", expectedResult, response)
			} else if gotExpectedResult && !errored {
				t.Log("Got expected response")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}

}
