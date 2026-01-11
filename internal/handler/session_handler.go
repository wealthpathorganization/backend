package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/model"
)

// SessionServiceInterface defines the contract for session management.
type SessionServiceInterface interface {
	GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*model.Session, error)
	RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, reason string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) (int64, error)
	GetSessionIDFromRefreshToken(ctx context.Context, refreshTokenString string) (uuid.UUID, error)
}

// SessionHandler handles session management endpoints.
type SessionHandler struct {
	sessionService SessionServiceInterface
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(sessionService SessionServiceInterface) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

// ListSessions godoc
// @Summary List active sessions
// @Description Get all active sessions for the current user
// @Tags sessions
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.Session
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/sessions [get]
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessions, err := h.sessionService.GetActiveSessions(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get sessions")
		return
	}

	// Try to mark the current session
	if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil && cookie.Value != "" {
		currentSessionID, err := h.sessionService.GetSessionIDFromRefreshToken(r.Context(), cookie.Value)
		if err == nil {
			for _, session := range sessions {
				if session.ID == currentSessionID {
					session.IsCurrent = true
					break
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, sessions)
}

// RevokeSession godoc
// @Summary Revoke a specific session
// @Description Revoke a specific session by its ID
// @Tags sessions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/sessions/{id} [delete]
func (h *SessionHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	// Don't allow revoking the current session via this endpoint
	// (use logout instead)
	if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil && cookie.Value != "" {
		currentSessionID, err := h.sessionService.GetSessionIDFromRefreshToken(r.Context(), cookie.Value)
		if err == nil && currentSessionID == sessionID {
			respondError(w, http.StatusBadRequest, "cannot revoke current session, use logout instead")
			return
		}
	}

	err = h.sessionService.RevokeSession(r.Context(), userID, sessionID, "user_revoked")
	if err != nil {
		if err.Error() == "session does not belong to user" {
			respondError(w, http.StatusNotFound, "session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "session revoked"})
}

// RevokeAllSessions godoc
// @Summary Revoke all sessions (sign out everywhere)
// @Description Revoke all sessions for the current user except the current one
// @Tags sessions
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/sessions [delete]
func (h *SessionHandler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.sessionService.RevokeAllUserTokens(r.Context(), userID, "sign_out_everywhere")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to revoke sessions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "all sessions revoked",
		"sessionsRevoked": count,
	})
}
