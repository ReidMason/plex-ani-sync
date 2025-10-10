package responseFactory

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Ok(w http.ResponseWriter, data interface{}, message string) {
	if message == "" {
		message = "Success"
	}

	response := Response{
		Success: true,
		Message: message,
		Data:    data,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func InternalServerError(w http.ResponseWriter, err error, message string) {
	if message == "" {
		message = "Internal Server Error"
	}

	response := Response{
		Success: false,
		Message: message,
		Data:    err.Error(),
	}

	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(response)
}

func BadRequest(w http.ResponseWriter, err error, message string) {
	if message == "" {
		message = "Bad Request"
	}

	response := Response{
		Success: false,
		Message: message,
		Data:    err.Error(),
	}

	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}
