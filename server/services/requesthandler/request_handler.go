package requesthandler

import (
	"io"
	"net/http"
	"net/url"
	"path"
)

type IRequestHandler interface {
	MakeRequest(method, url string, headers http.Header) (string, error)
}

type RequestHandler struct{}

var _ IRequestHandler = (*RequestHandler)(nil)

func New() *RequestHandler {
	return &RequestHandler{}
}

func (rh RequestHandler) MakeRequest(method, url string, headers http.Header) (string, error) {
	client := http.Client{}

	req, err := BuildRequest(method, url, headers)
	if err != nil {
		return "", HttpRequestError{Err: err}
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// Extract response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	stringBody := string(body)
	return stringBody, nil
}

func BuildRequest(method, url string, headers http.Header) (*http.Request, error) {
	// Setup request
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header = headers

	return req, nil
}

func BuildUrl(baseUrl, endpoint string, queryParams []QueryParam) (string, error) {
	url, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}

	if len(queryParams) > 0 {
		AddQueryParams(url, queryParams)
	}

	// Add requested path
	url.Path = path.Join(url.Path, endpoint)

	return url.String(), nil
}

func AddQueryParams(url *url.URL, queryParams []QueryParam) {
	for _, queryParam := range queryParams {
		// Add token query param
		q := url.Query()
		q.Set(queryParam.Key, queryParam.Value)
		url.RawQuery = q.Encode()
	}
}

type HttpRequestError struct {
	Err error
}

func (e HttpRequestError) Error() string { return e.Err.Error() }

type QueryParam struct {
	Key   string
	Value string
}
