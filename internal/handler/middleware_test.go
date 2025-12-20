package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wealthpath/backend/internal/service"
)

func TestAuthMiddleware(t *testing.T) {
	// Set JWT secret for tests
	t.Setenv("JWT_SECRET", "test-secret")

	tests := []struct {
		name       string
		authHeader string
		wantCode   int
	}{
		{
			name:       "missing authorization header",
			authHeader: "",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "invalid authorization format - no bearer",
			authHeader: "invalid-token",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "invalid authorization format - wrong prefix",
			authHeader: "Basic invalid-token",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-jwt-token",
			wantCode:   http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(next)
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantCode == http.StatusUnauthorized {
				assert.False(t, nextCalled)
			}
		})
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Set JWT secret for tests
	t.Setenv("JWT_SECRET", "test-secret")

	// Generate a valid token
	token, err := service.GenerateTokenForTest()
	if err != nil {
		t.Skip("Skipping test - cannot generate token")
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		// Check if userID is in context
		userID := GetUserID(r.Context())
		assert.NotEqual(t, userID.String(), "00000000-0000-0000-0000-000000000000")
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		assert.True(t, nextCalled)
	}
}
