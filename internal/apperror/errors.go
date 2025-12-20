package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for common cases
var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrBadRequest         = errors.New("bad request")
	ErrConflict           = errors.New("conflict")
	ErrInternal           = errors.New("internal server error")
	ErrValidation         = errors.New("validation error")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// AppError wraps errors with HTTP status and user-friendly message
type AppError struct {
	Err        error  // Original error (for logging)
	Message    string // User-friendly message
	StatusCode int    // HTTP status code
	Field      string // Optional field name for validation errors
}

func (e *AppError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Constructor functions for common errors

func NotFound(resource string) *AppError {
	return &AppError{
		Err:        ErrNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
	}
}

func BadRequest(message string) *AppError {
	return &AppError{
		Err:        ErrBadRequest,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func ValidationError(field, message string) *AppError {
	return &AppError{
		Err:        ErrValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Field:      field,
	}
}

func Unauthorized(message string) *AppError {
	if message == "" {
		message = "unauthorized"
	}
	return &AppError{
		Err:        ErrUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func Forbidden(message string) *AppError {
	if message == "" {
		message = "forbidden"
	}
	return &AppError{
		Err:        ErrForbidden,
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

func Conflict(message string) *AppError {
	return &AppError{
		Err:        ErrConflict,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

func Internal(err error) *AppError {
	return &AppError{
		Err:        err,
		Message:    "an internal error occurred",
		StatusCode: http.StatusInternalServerError,
	}
}

func Wrap(err error, message string) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}

// GetStatusCode extracts HTTP status from error, defaults to 500
func GetStatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	// Check sentinel errors
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrBadRequest), errors.Is(err, ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// GetMessage extracts user message from error
func GetMessage(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return err.Error()
}
