package plexController

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/plex"
)

// MockPlexAuthURLGenerator is a mock implementation for testing
type MockPlexAuthURLGenerator struct {
	GetPlexAuthUrlFunc func() (string, error)
}

func (m *MockPlexAuthURLGenerator) GetPlexAuthUrl() (string, error) {
	return m.GetPlexAuthUrlFunc()
}

// MockHTTPClient is a mock implementation of HTTPClient for integration testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestGetAuthURL_FullIntegration_Success(t *testing.T) {
	// Arrange
	expectedPinCode := "integration-test-pin-123"
	clientID := "test-client-id"
	appName := "TestApp"

	// Mock the Plex API response
	mockPinResponse := plex.PinResponse{
		ID:               999,
		Code:             expectedPinCode,
		Product:          appName,
		Trusted:          false,
		ClientIdentifier: clientID,
		ExpiresIn:        300,
	}

	// Create a mock HTTP client that simulates the Plex API
	mockHTTPClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify the request to the "Plex API" is correct
			if req.Method != "POST" {
				t.Errorf("Expected POST request to Plex API, got %s", req.Method)
			}
			if req.URL.String() != "https://plex.tv/api/v2/pins?strong=true" {
				t.Errorf("Unexpected Plex API URL: %s", req.URL.String())
			}

			// Verify headers
			if req.Header.Get("X-Plex-Product") != appName {
				t.Errorf("Expected X-Plex-Product header to be %s, got %s", appName, req.Header.Get("X-Plex-Product"))
			}
			if req.Header.Get("X-Plex-Client-Identifier") != clientID {
				t.Errorf("Expected X-Plex-Client-Identifier header to be %s, got %s", clientID, req.Header.Get("X-Plex-Client-Identifier"))
			}
			if req.Header.Get("Accept") != "application/json" {
				t.Errorf("Expected Accept header to be application/json, got %s", req.Header.Get("Accept"))
			}

			// Return mock Plex API response
			body, _ := json.Marshal(mockPinResponse)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		},
	}

	// Wire up the real components with the mock HTTP client
	plexAuth := plex.NewPlexAuth(clientID, appName, mockHTTPClient)
	controller := New(plexAuth)

	// Create HTTP request to our controller
	req := httptest.NewRequest("GET", "/api/plex/authUrl", nil)
	w := httptest.NewRecorder()

	// Act
	controller.GetAuthURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	actualURL := string(bodyBytes)

	// Verify the URL contains expected components
	expectedURLPattern := "https://app.plex.tv/auth#?clientID=" + clientID + "&code=" + expectedPinCode
	if len(actualURL) < len(expectedURLPattern) || actualURL[:len(expectedURLPattern)] != expectedURLPattern {
		t.Errorf("Expected URL to start with %s, got %s", expectedURLPattern, actualURL)
	}

	// Verify the URL contains the app name (URL encoded)
	if !contains(actualURL, "context") {
		t.Error("Expected URL to contain context parameter")
	}
}

func TestGetAuthURL_FullIntegration_PlexAPIError(t *testing.T) {
	// Arrange
	clientID := "test-client-id"
	appName := "TestApp"
	expectedError := errors.New("plex API connection failed")

	// Create a mock HTTP client that simulates a Plex API failure
	mockHTTPClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, expectedError
		},
	}

	// Wire up the real components
	plexAuth := plex.NewPlexAuth(clientID, appName, mockHTTPClient)
	controller := New(plexAuth)

	// Create HTTP request to our controller
	req := httptest.NewRequest("GET", "/api/plex/authUrl", nil)
	w := httptest.NewRecorder()

	// Act
	controller.GetAuthURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	errorMessage := string(bodyBytes)

	if errorMessage != expectedError.Error() {
		t.Errorf("Expected error message %s, got %s", expectedError.Error(), errorMessage)
	}
}

func TestGetAuthURL_FullIntegration_InvalidJSON(t *testing.T) {
	// This integration test verifies error handling when the "Plex API" returns invalid JSON

	// Arrange
	clientID := "test-client-id"
	appName := "TestApp"

	// Create a mock HTTP client that returns invalid JSON
	mockHTTPClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte("invalid json response"))),
			}, nil
		},
	}

	// Wire up the real components
	plexAuth := plex.NewPlexAuth(clientID, appName, mockHTTPClient)
	controller := New(plexAuth)

	// Create HTTP request to our controller
	req := httptest.NewRequest("GET", "/api/plex/authUrl", nil)
	w := httptest.NewRecorder()

	// Act
	controller.GetAuthURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d due to invalid JSON, got %d", http.StatusInternalServerError, resp.StatusCode)
	}
}

func TestGetAuthURL_FullIntegration_WithRouting(t *testing.T) {
	// This integration test verifies the full routing and request handling

	// Arrange
	expectedPinCode := "routing-test-pin-456"
	clientID := "routing-client"
	appName := "RoutingApp"

	mockPinResponse := plex.PinResponse{
		ID:               777,
		Code:             expectedPinCode,
		Product:          appName,
		ClientIdentifier: clientID,
		ExpiresIn:        600,
	}

	mockHTTPClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			body, _ := json.Marshal(mockPinResponse)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		},
	}

	// Wire up everything including the mux
	plexAuth := plex.NewPlexAuth(clientID, appName, mockHTTPClient)
	controller := New(plexAuth)
	mux := http.NewServeMux()
	controller.RegisterRoutes(mux)

	// Create HTTP request through the mux
	req := httptest.NewRequest("GET", "/api/plex/authUrl", nil)
	w := httptest.NewRecorder()

	// Act
	mux.ServeHTTP(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	actualURL := string(bodyBytes)

	// Verify it generated a valid URL
	if !contains(actualURL, "https://app.plex.tv/auth") {
		t.Errorf("Expected URL to contain auth endpoint, got %s", actualURL)
	}
	if !contains(actualURL, expectedPinCode) {
		t.Errorf("Expected URL to contain pin code %s, got %s", expectedPinCode, actualURL)
	}
	if !contains(actualURL, clientID) {
		t.Errorf("Expected URL to contain client ID %s, got %s", clientID, actualURL)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
