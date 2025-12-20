package apperror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name: "without field",
			appErr: &AppError{
				Message: "something went wrong",
			},
			expected: "something went wrong",
		},
		{
			name: "with field",
			appErr: &AppError{
				Message: "is required",
				Field:   "email",
			},
			expected: "email: is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.appErr.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error")
	appErr := &AppError{
		Err:     originalErr,
		Message: "wrapped error",
	}

	assert.Equal(t, originalErr, appErr.Unwrap())
	assert.True(t, errors.Is(appErr, originalErr))
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	err := NotFound("user")

	assert.Equal(t, "user not found", err.Message)
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestBadRequest(t *testing.T) {
	t.Parallel()

	err := BadRequest("invalid input")

	assert.Equal(t, "invalid input", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.True(t, errors.Is(err, ErrBadRequest))
}

func TestValidationError(t *testing.T) {
	t.Parallel()

	err := ValidationError("email", "must be a valid email")

	assert.Equal(t, "must be a valid email", err.Message)
	assert.Equal(t, "email", err.Field)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.True(t, errors.Is(err, ErrValidation))
}

func TestUnauthorized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "with message",
			message:  "custom message",
			expected: "custom message",
		},
		{
			name:     "empty message defaults",
			message:  "",
			expected: "unauthorized",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Unauthorized(tt.message)
			assert.Equal(t, tt.expected, err.Message)
			assert.Equal(t, http.StatusUnauthorized, err.StatusCode)
		})
	}
}

func TestForbidden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "with message",
			message:  "access denied",
			expected: "access denied",
		},
		{
			name:     "empty message defaults",
			message:  "",
			expected: "forbidden",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Forbidden(tt.message)
			assert.Equal(t, tt.expected, err.Message)
			assert.Equal(t, http.StatusForbidden, err.StatusCode)
		})
	}
}

func TestConflict(t *testing.T) {
	t.Parallel()

	err := Conflict("resource already exists")

	assert.Equal(t, "resource already exists", err.Message)
	assert.Equal(t, http.StatusConflict, err.StatusCode)
	assert.True(t, errors.Is(err, ErrConflict))
}

func TestInternal(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("database connection failed")
	err := Internal(originalErr)

	assert.Equal(t, "an internal error occurred", err.Message)
	assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
	assert.True(t, errors.Is(err, originalErr))
}

func TestWrap(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original")
	err := Wrap(originalErr, "custom message")

	assert.Equal(t, "custom message", err.Message)
	assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
	assert.True(t, errors.Is(err, originalErr))
}

func TestGetStatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "AppError",
			err:      &AppError{StatusCode: http.StatusTeapot},
			expected: http.StatusTeapot,
		},
		{
			name:     "ErrNotFound",
			err:      ErrNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "ErrUnauthorized",
			err:      ErrUnauthorized,
			expected: http.StatusUnauthorized,
		},
		{
			name:     "ErrForbidden",
			err:      ErrForbidden,
			expected: http.StatusForbidden,
		},
		{
			name:     "ErrBadRequest",
			err:      ErrBadRequest,
			expected: http.StatusBadRequest,
		},
		{
			name:     "ErrValidation",
			err:      ErrValidation,
			expected: http.StatusBadRequest,
		},
		{
			name:     "ErrConflict",
			err:      ErrConflict,
			expected: http.StatusConflict,
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown"),
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, GetStatusCode(tt.err))
		})
	}
}

func TestGetMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "AppError",
			err:      &AppError{Message: "custom message"},
			expected: "custom message",
		},
		{
			name:     "regular error",
			err:      errors.New("regular error message"),
			expected: "regular error message",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, GetMessage(tt.err))
		})
	}
}

// Test sentinel errors exist
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, ErrNotFound)
	assert.NotNil(t, ErrUnauthorized)
	assert.NotNil(t, ErrForbidden)
	assert.NotNil(t, ErrBadRequest)
	assert.NotNil(t, ErrConflict)
	assert.NotNil(t, ErrInternal)
	assert.NotNil(t, ErrValidation)
	assert.NotNil(t, ErrInvalidCredentials)
}
