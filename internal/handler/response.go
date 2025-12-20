package handler

import (
	"encoding/json"
	"net/http"

	"github.com/wealthpath/backend/internal/apperror"
)

// ErrorResponse represents a JSON error response body.
type ErrorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

// respondJSON writes a JSON response with the given status code.
// It sets the Content-Type header to application/json.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// respondError writes a JSON error response with the given status code and message.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}

// respondAppError writes a JSON error response from an AppError.
// It extracts the status code and message from the error.
func respondAppError(w http.ResponseWriter, err *apperror.AppError) {
	resp := ErrorResponse{
		Error: err.Message,
		Field: err.Field,
	}
	respondJSON(w, err.StatusCode, resp)
}
