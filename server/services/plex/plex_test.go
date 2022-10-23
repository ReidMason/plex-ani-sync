package plex

import (
	"encoding/json"
	"net/url"
	"plex-ani-sync/services/config"
	"plex-ani-sync/services/requesthandler"
	"reflect"
	"testing"
)

func getBaseMockConfig() config.IConfigHandler {
	mockConfig := config.Config{
		Plex: config.PlexConfig{
			BaseUrl: "",
			Token:   "",
		},
		Sync: config.SyncConfig{
			DaysUntilPaused:  14,
			DaysUntilDropped: 31,
		},
	}

	return config.NewMock(mockConfig)
}

func TestGetAllSeries(t *testing.T) {
	t.Parallel()

	plexConnection := New(getBaseMockConfig(), requesthandler.NewMock(requesthandler.GetMockResponseData().SeriesResponse, nil))
	allSeries, err := plexConnection.GetSeries("1")

	expectedTitle := "Test anime title"
	gotExpectedTitle := allSeries[0].Title == expectedTitle
	errored := err != nil

	if errored {
		t.Errorf("Unexpected error: '%s'", err)
	} else if len(allSeries) <= 0 {
		t.Error("Found no series found")
	} else if !gotExpectedTitle {
		t.Errorf("Got wrong series title. Expected: '%s' found: '%s'", allSeries[0].Title, expectedTitle)
	} else if !errored && gotExpectedTitle {
		t.Logf("Found %d series", len(allSeries))
	} else {
		t.Error("Test failed unexpectedly")
	}
}

func TestGetAllLibraries(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, ResponseData, ExpectedResult string
		GetResult                          func(x []Library) string
	}{
		{"Get all libraries", requesthandler.GetMockResponseData().LibrariesResponse, "Test library", func(x []Library) string { return x[0].Title }},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexConnection := New(getBaseMockConfig(), requesthandler.NewMock(tc.ResponseData, nil))
			libraries, err := plexConnection.GetAllLibraries()

			result := tc.GetResult(libraries)
			gotExpectedResult := result == tc.ExpectedResult
			errored := err != nil

			if errored {
				t.Errorf("Unexpected error: '%s'", err)
			} else if len(libraries) <= 0 {
				t.Error("Found no libraries")
			} else if !gotExpectedResult {
				t.Errorf("Got wrong library title. Expected: '%s' found: '%s'", libraries[0].Title, tc.ExpectedResult)
			} else if !errored && gotExpectedResult {
				t.Logf("Found %d series", len(libraries))
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}
}

func TestGetAllSeriesWhenResponseErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name          string
		ResponseError error
		ExpectedError error
	}{
		{"Get all libraries with response error", &url.Error{}, &url.Error{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexConnection := New(getBaseMockConfig(), requesthandler.NewMock("", tc.ResponseError))
			libraries, err := plexConnection.GetSeries("1")

			gotExpectedResult := libraries == nil
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			if !gotExpectedError {
				t.Errorf("Wrong error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Wrong response. Expected: '%#v' found: '%#v'", nil, libraries)
			} else if gotExpectedError && gotExpectedResult {
				t.Log("Got expected error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}
}

func TestGetAllSeriesWhenJSONIsInvalid(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, ResponseData string
		ExpectedError      error
	}{
		{"Get all libraries with invalid JSON", `"data": "invalid"}`, &json.SyntaxError{}},
		{"Get all libraries with no JSON", ``, &json.SyntaxError{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexConnection := New(getBaseMockConfig(), requesthandler.NewMock(tc.ResponseData, nil))
			libraries, err := plexConnection.GetSeries("1")

			gotExpectedResult := libraries == nil
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			if !gotExpectedError {
				t.Errorf("Wrong error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Wrong response. Expected: '%#v' found: '%#v'", nil, libraries)
			} else if gotExpectedError && gotExpectedResult {
				t.Log("Got expected error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}
}

func TestGetAllLibrariesWhenResponseErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name          string
		ResponseError error
		ExpectedError error
	}{
		{"Get all libraries with response error", &url.Error{}, &url.Error{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexConnection := New(getBaseMockConfig(), requesthandler.NewMock("", tc.ResponseError))
			libraries, err := plexConnection.GetAllLibraries()

			gotExpectedResult := libraries == nil
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			if !gotExpectedError {
				t.Errorf("Wrong error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Wrong response. Expected: '%#v' found: '%#v'", nil, libraries)
			} else if gotExpectedError && gotExpectedResult {
				t.Log("Got expected error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}
}

func TestGetAllLibrariesWhenJSONIsInvalid(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, ResponseData string
		ExpectedError      error
	}{
		{"Get all libraries with invalid JSON", `"data": "invalid"}`, &json.SyntaxError{}},
		{"Get all libraries with no JSON", ``, &json.SyntaxError{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexConnection := New(getBaseMockConfig(), requesthandler.NewMock(tc.ResponseData, nil))
			libraries, err := plexConnection.GetAllLibraries()

			gotExpectedResult := libraries == nil
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			if !gotExpectedError {
				t.Errorf("Wrong error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Wrong response. Expected: '%#v' found: '%#v'", nil, libraries)
			} else if gotExpectedError && gotExpectedResult {
				t.Log("Got expected error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}
}

func TestGetSeasons(t *testing.T) {
	t.Parallel()

	plexConnection := New(getBaseMockConfig(), requesthandler.NewMock(requesthandler.GetMockResponseData().SeasonsResponse, nil))
	seasons, err := plexConnection.GetSeasons("1")

	expectedTitle := "Season 1"
	gotExpectedTitle := seasons[0].Title == expectedTitle
	errored := err != nil

	if errored {
		t.Errorf("Unexpected error: '%s'", err)
	} else if len(seasons) <= 0 {
		t.Error("Found no seasons found")
	} else if !gotExpectedTitle {
		t.Errorf("Got wrong season title. Expected: '%s' found: '%s'", seasons[0].Title, expectedTitle)
	} else if !errored && gotExpectedTitle {
		t.Logf("Found %d seasons", len(seasons))
	} else {
		t.Error("Test failed unexpectedly")
	}
}

func TestGetSeasonsWhenResponseErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name          string
		ResponseError error
		ExpectedError error
	}{
		{"Get seasons with response error", &url.Error{}, &url.Error{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexConnection := New(getBaseMockConfig(), requesthandler.NewMock("", tc.ResponseError))
			seasons, err := plexConnection.GetSeasons("1")

			gotExpectedResult := seasons == nil
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			if !gotExpectedError {
				t.Errorf("Wrong error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Wrong response. Expected: '%#v' found: '%#v'", nil, seasons)
			} else if gotExpectedError && gotExpectedResult {
				t.Log("Got expected error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}
}

func TestGetSeasonsWhenJSONIsInvalid(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, ResponseData string
		ExpectedError      error
	}{
		{"Get seasons with invalid JSON", `"data": "invalid"}`, &json.SyntaxError{}},
		{"Get seasons with no JSON", ``, &json.SyntaxError{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			plexConnection := New(getBaseMockConfig(), requesthandler.NewMock(tc.ResponseData, nil))
			seasons, err := plexConnection.GetSeasons("1")

			gotExpectedResult := seasons == nil
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			gotExpectedError := errorType == expectedErrorType

			if !gotExpectedError {
				t.Errorf("Wrong error. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !gotExpectedResult {
				t.Errorf("Wrong response. Expected: '%#v' found: '%#v'", nil, seasons)
			} else if gotExpectedError && gotExpectedResult {
				t.Log("Got expected error")
			} else {
				t.Error("Test failed unexpectedly")
			}
		})
	}
}
