package mapping

import (
	"encoding/json"
	"io/fs"
	"plex-ani-sync/filehandler"
	"plex-ani-sync/utils"
	"reflect"
	"testing"
)

func TestGetAnilistMapping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name, FileData, Input, Expected string
	}{
		{"Get existing mapping", `[{"PlexSeasonRatingKey": "123","AnilistId": "456"}]`, "123", "456"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			mapper := New(filehandler.NewMock[Mappings]([]byte(tc.FileData), nil))
			result, err := mapper.GetAnilistMapping(tc.Input)

			errored := err != nil
			expectedResultFound := result.AnilistId == tc.Expected

			if !errored && !expectedResultFound {
				t.Errorf("Got wrong AnilistId. Expected: '%s' found: '%s'", tc.Expected, result.AnilistId)
			} else if !errored && expectedResultFound {
				t.Log("Found correct AnilistId")
			} else {
				t.Errorf("Test failed unexpectedly. Error: %s", err)
			}
		})
	}
}

func TestGetAnilistErrorsWhenMappingDoesntExist(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name, FileData, Input string
		Expected              Mapping
		ExpectedError         error
	}{
		{"Get missing mapping", `[{"PlexSeasonRatingKey": "123","AnilistId": "456"}]`, "999", Mapping{}, utils.SliceItemNotFound{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			mapper := New(filehandler.NewMock[Mappings]([]byte(tc.FileData), nil))
			result, err := mapper.GetAnilistMapping(tc.Input)

			expectedErrorFound := reflect.TypeOf(err) == reflect.TypeOf(tc.ExpectedError)
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			expectedResultFound := result == tc.Expected

			if !expectedErrorFound {
				t.Errorf("Wrong error type. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if !expectedResultFound {
				t.Errorf("Got wrong result. Expected: '%#v' found: '%#v'", tc.Expected, result)
			} else if expectedErrorFound && expectedResultFound {
				t.Logf("Correct error: '%#v' and result: '%#v' found", errorType, result)
			} else {
				t.Errorf("Test failed unexpectedly. Error: %s", err)
			}
		})
	}
}

func TestGetAnilistIdErrorsWhenFileFailsToRead(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name, FileData, Input    string
		FileError, ExpectedError error
	}{
		{"Get mapping when file failed to load", `[{"PlexSeasonRatingKey": "123","AnilistId": "456"}]`, "123", &fs.PathError{}, &fs.PathError{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			mapper := New(filehandler.NewMock[Mappings]([]byte(tc.FileData), tc.FileError))
			_, err := mapper.GetAnilistMapping(tc.Input)

			errored := err != nil
			errorExpected := tc.ExpectedError != nil
			expectedErrorFound := reflect.TypeOf(err) == reflect.TypeOf(tc.ExpectedError)
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)

			if errored && errorExpected && expectedErrorFound {
				t.Log("Mapping file failing to load returns correct error")
			} else if errored && errorExpected && !expectedErrorFound {
				t.Errorf("Wrong error type. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else {
				t.Errorf("Test failed unexpectedly. Error: %s", err)
			}
		})
	}
}

func TestGetAnilistIdErrorsWhenMappingJsonIsInvalid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name, FileData, Input string
		ExpectedError         error
	}{
		{"Get mapping when JSON is invalid", `[{PlexSeasonRatingKey": "123","AnilistId": "456"}]`, "123", &json.SyntaxError{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			mapper := New(filehandler.NewMock[Mappings]([]byte(tc.FileData), nil))
			_, err := mapper.GetAnilistMapping(tc.Input)

			errored := err != nil
			errorExpected := tc.ExpectedError != nil
			expectedErrorFound := reflect.TypeOf(err) == reflect.TypeOf(tc.ExpectedError)
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)

			if errored && errorExpected && expectedErrorFound {
				t.Log("Invalid JSON returns correct error")
			} else if errored && errorExpected && !expectedErrorFound {
				t.Errorf("Wrong error type. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else {
				t.Errorf("Test failed unexpectedly. Error: %s", err)
			}
		})
	}
}

func TestLoadMapping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name, FileData, ExpectedValue string
		FindExpectedValue             func(data Mappings) string
		FileError, ExpectedError      error
	}{
		{"Load mapping file", `[{"PlexSeasonRatingKey": "123","AnilistId": "456"}]`, "456", func(data Mappings) string { return data[0].AnilistId }, nil, nil},
		{"Load invalid JSON", `[{PlexSeasonRatingKey": "123","AnilistId": "456"}]`, "456", nil, nil, &json.SyntaxError{}},
		{"Load mapping when failed to read file", `[{"PlexSeasonRatingKey": "123","AnilistId": "456"}]`, "", nil, &fs.PathError{}, &fs.PathError{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			mapper := New(filehandler.NewMock[Mappings]([]byte(tc.FileData), tc.FileError))
			result, err := mapper.LoadMapping()

			errored := err != nil
			errorExpected := tc.ExpectedError != nil
			expectedErrorFound := reflect.TypeOf(err) == reflect.TypeOf(tc.ExpectedError)
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)
			expectedValue := ""
			if tc.FindExpectedValue != nil {
				expectedValue = tc.FindExpectedValue(result)
			}
			expectedResultFound := expectedValue == tc.ExpectedValue

			if errored && errorExpected && expectedErrorFound {
				t.Log("Correct error found")
			} else if errored && errorExpected && !expectedErrorFound {
				t.Errorf("Wrong error type. Expected: '%s' found: '%s'", expectedErrorType, errorType)
			} else if err != nil && (!errorExpected || !expectedErrorFound) {
				t.Errorf("Unexpected error: '%s'", errorType)
			} else if !errorExpected && !expectedResultFound {
				t.Errorf("Got wrong AnilistId. Expected: '%s' found: '%s'", tc.ExpectedValue, expectedValue)
			} else if !errorExpected && expectedResultFound {
				t.Log("Found correct AnilistId")
			} else {
				t.Errorf("Test failed unexpectedly. Error: %s", err)
			}
		})
	}
}
