package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"log/slog"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type StaggeredHttpClient struct {
	lastRequest time.Time
	client      HttpClient
	log         *slog.Logger
	mutex       sync.Mutex
}

func NewStaggeredHttpClient(client HttpClient, logger *slog.Logger) *StaggeredHttpClient {
	return &StaggeredHttpClient{client: client, lastRequest: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), mutex: sync.Mutex{}, log: logger}
}

func (r *StaggeredHttpClient) Do(request *http.Request) (*http.Response, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if time.Since(r.lastRequest) < 2*time.Second {
		r.log.Info("Sleeping for 2 seconds")
		time.Sleep(2 * time.Second)
	}
	r.lastRequest = time.Now()

	return r.client.Do(request)
}

func MakeRequest[T any](client HttpClient, request *http.Request) (T, error) {
	var result T

	if client == nil {
		return result, errors.New("No client provided for request")
	}

	resp, err := client.Do(request)
	if err != nil {
		return result, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return result, errors.New("Request failed with status: " + resp.Status + " URL: " + request.URL.RequestURI() + " Response body: " + string(body))
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

	slog.Info("response", slog.String("response", string(body)))
	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}

	return result, nil
}
