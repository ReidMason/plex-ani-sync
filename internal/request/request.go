package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"golang.org/x/exp/slog"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func MakeRequest[T any](client HttpClient, request *http.Request) (T, error) {
	var result T

	if client == nil {
		return result, errors.New("No client provided for request")
	}

	slog.Info("Making request", slog.String("url", request.URL.String()))
	resp, err := client.Do(request)
	if err != nil {
		slog.Error("Failed to make request", slog.Any("error", err))
		return result, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Request failed",
			slog.String("status", resp.Status),
			slog.String("URL", request.URL.RequestURI()),
			slog.String("Response body", string(body)),
		)
		return result, errors.New("Request failed with status: " + resp.Status)
	}

	defer resp.Body.Close()
	return parseResponse[T](resp.Body)
}

func parseResponse[T any](responseBody io.ReadCloser) (T, error) {
	var result T
	body, err := io.ReadAll(responseBody)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}

	return result, nil
}
