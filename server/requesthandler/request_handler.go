package requesthandler

type IRequestHandler interface {
	MakeRequest(method, endpoint string) (string, error)
}

type HttpRequestError struct {
	Err error
}

func (e HttpRequestError) Error() string { return e.Err.Error() }
