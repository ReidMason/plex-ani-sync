package utils

import (
	"reflect"
	"testing"
)

func TestGetSliceIndex(t *testing.T) {
	t.Parallel()
	testSlice := []string{"test1", "test2", "test3", "test4"}

	testCases := []struct {
		Name, SliceValue string
		Expected         int
	}{
		{"Find item in first index", "test1", 0},
		{"Find item in third index", "test3", 2},
		{"Find item not in slice", "test5", -1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result := GetSliceIndex(testSlice, func(x string) bool { return x == tc.SliceValue })
			if result != tc.Expected {
				t.Errorf("Got index '%d' expected '%d'", result, tc.Expected)
			} else {
				t.Log("Correct index found")
			}
		})
	}
}

func TestGetSliceItem(t *testing.T) {
	t.Parallel()
	testSlice := []string{"test1", "test2", "test3", "test4"}

	testCases := []struct {
		Name, SliceValue, Expected string
	}{
		{"First slice value", "test1", "test1"},
		{"Third slice value", "test3", "test3"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result, err := GetSliceItem(testSlice, func(x string) bool { return x == tc.SliceValue })

			errored := err != nil
			expectedResultFound := result == tc.Expected

			if errored {
				t.Errorf("Unexpected error: '%s'", err)
			} else if !expectedResultFound {
				t.Errorf("Wrong result. Expected: '%s' found: '%s'", tc.Expected, result)
			} else if expectedResultFound {
				t.Log("Correct item found")
			} else {
				t.Errorf("Test failed unexpectedly")
			}
		})
	}
}

func TestGetSliceItemWhenItDoesntExist(t *testing.T) {
	t.Parallel()
	testSlice := []string{"test1", "test2", "test3", "test4"}

	testCases := []struct {
		Name, SliceValue, Expected string
		ExpectedError              error
	}{
		{"Missing slice value", "test5", "", SliceItemNotFound{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result, err := GetSliceItem(testSlice, func(x string) bool { return x == tc.SliceValue })

			expectedResultFound := result == tc.Expected
			expectedErrorFound := reflect.TypeOf(err) == reflect.TypeOf(tc.ExpectedError)
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)

			if !expectedErrorFound {
				t.Errorf("Wrong error. Found: '%s' expected: '%s'", errorType, expectedErrorType)
			} else if !expectedResultFound {
				t.Errorf("Wrong result. Found: '%s' expected: '%s'", tc.Expected, result)
			} else if expectedErrorFound && expectedResultFound {
				t.Log("Correct error and result found")
			} else {
				t.Errorf("Test failed unexpectedly")
			}
		})
	}
}

func TestGetSliceItemWhenItDoesntExistForStruct(t *testing.T) {
	t.Parallel()
	testSlice := []Person{
		{
			Name: "Joe",
		},
		{
			Name: "Samuel",
		},
		{
			Name: "Samantha",
		},
	}

	testCases := []struct {
		Name, PersonName string
		Expected         Person
		ExpectedError    error
	}{
		{"Missing slice item", "Ralf", Person{}, SliceItemNotFound{}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result, err := GetSliceItem(testSlice, func(x Person) bool { return x.Name == tc.PersonName })

			expectedResultFound := result == tc.Expected
			expectedErrorFound := reflect.TypeOf(err) == reflect.TypeOf(tc.ExpectedError)
			errorType := reflect.TypeOf(err)
			expectedErrorType := reflect.TypeOf(tc.ExpectedError)

			if !expectedErrorFound {
				t.Errorf("Wrong error. Found: '%s' expected: '%s'", errorType, expectedErrorType)
			} else if !expectedResultFound {
				t.Errorf("Wrong result. Found: '%s' expected: '%s'", tc.Expected, result)
			} else if expectedErrorFound && expectedResultFound {
				t.Logf("Correct error: '%s' and result: '%s' found", errorType, result)
			} else {
				t.Errorf("Test failed unexpectedly")
			}
		})
	}
}

func TestParseJson(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name, StringData  string
		FindExpectedValue func(x Data) bool
		ExpectError       bool
	}{
		{"Valid JSON", `{"data": {"name": "joe"}}`, func(x Data) bool { return x.Data.Name == "joe" }, false},
		{"Invalid JSON", `{data": {"nametc"joe"}}`, nil, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseJson[Data](tc.StringData)

			if err != nil && !tc.ExpectError {
				t.Errorf("Error parsing JSON '%s' error: '%s'", tc.StringData, err)
			} else if tc.FindExpectedValue != nil && !tc.FindExpectedValue(result) {
				t.Errorf("Incorrectly parsed JSON '%s'", tc.StringData)
			} else if tc.ExpectError && err == nil {
				t.Errorf("Expected parsing error for JSON '%s'", tc.StringData)
			} else {
				t.Log("JSON data parsed correctly")
			}
		})
	}
}

type Data struct {
	Data Person
}

type Person struct {
	Name string
}
