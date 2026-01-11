package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/service"
)

// TOTPServiceInterface defines the interface for TOTP operations.
type TOTPServiceInterface interface {
	Setup(ctx context.Context, userID uuid.UUID) (*service.TOTPSetupResponse, error)
	Verify(ctx context.Context, userID uuid.UUID, code string) ([]string, error)
	ValidateCode(ctx context.Context, userID uuid.UUID, code string) error
	ValidateBackupCode(ctx context.Context, userID uuid.UUID, code string) error
	Disable(ctx context.Context, userID uuid.UUID, code string) error
	RegenerateBackupCodes(ctx context.Context, userID uuid.UUID, code string) ([]string, error)
}

// TOTPHandler handles 2FA-related HTTP requests.
type TOTPHandler struct {
	totpService TOTPServiceInterface
}

// NewTOTPHandler creates a new TOTP handler.
func NewTOTPHandler(totpService TOTPServiceInterface) *TOTPHandler {
	return &TOTPHandler{totpService: totpService}
}

type totpCodeRequest struct {
	Code string `json:"code"`
}

// Setup godoc
// @Summary Set up 2FA
// @Description Generate a new TOTP secret and QR code for 2FA setup
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} service.TOTPSetupResponse
// @Failure 400 {object} ErrorResponse "2FA already enabled"
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/2fa/setup [post]
func (h *TOTPHandler) Setup(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.totpService.Setup(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrTOTPAlreadyEnabled) {
			respondError(w, http.StatusBadRequest, "2FA is already enabled")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to set up 2FA")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

type totpVerifyResponse struct {
	BackupCodes []string `json:"backupCodes"`
}

// Verify godoc
// @Summary Verify and enable 2FA
// @Description Verify the TOTP code and enable 2FA for the user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body totpCodeRequest true "TOTP code"
// @Success 200 {object} totpVerifyResponse "Returns backup codes"
// @Failure 400 {object} ErrorResponse "Invalid code or 2FA not set up"
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/2fa/verify [post]
func (h *TOTPHandler) Verify(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req totpCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	backupCodes, err := h.totpService.Verify(r.Context(), userID, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrTOTPAlreadyEnabled) {
			respondError(w, http.StatusBadRequest, "2FA is already enabled")
			return
		}
		if errors.Is(err, service.ErrTOTPNotSetup) {
			respondError(w, http.StatusBadRequest, "2FA has not been set up")
			return
		}
		if errors.Is(err, service.ErrInvalidTOTPCode) {
			respondError(w, http.StatusBadRequest, "invalid code")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to verify 2FA")
		return
	}

	respondJSON(w, http.StatusOK, totpVerifyResponse{BackupCodes: backupCodes})
}

// Disable godoc
// @Summary Disable 2FA
// @Description Disable 2FA for the user (requires current TOTP code)
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body totpCodeRequest true "TOTP code"
// @Success 200 {object} map[string]string "success message"
// @Failure 400 {object} ErrorResponse "Invalid code or 2FA not enabled"
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/2fa/disable [post]
func (h *TOTPHandler) Disable(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req totpCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	if err := h.totpService.Disable(r.Context(), userID, req.Code); err != nil {
		if errors.Is(err, service.ErrTOTPNotEnabled) {
			respondError(w, http.StatusBadRequest, "2FA is not enabled")
			return
		}
		if errors.Is(err, service.ErrInvalidTOTPCode) {
			respondError(w, http.StatusBadRequest, "invalid code")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to disable 2FA")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "2FA disabled successfully"})
}

// RegenerateBackupCodes godoc
// @Summary Regenerate backup codes
// @Description Generate new backup codes (requires current TOTP code)
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body totpCodeRequest true "TOTP code"
// @Success 200 {object} totpVerifyResponse "Returns new backup codes"
// @Failure 400 {object} ErrorResponse "Invalid code or 2FA not enabled"
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/2fa/backup-codes [post]
func (h *TOTPHandler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req totpCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	backupCodes, err := h.totpService.RegenerateBackupCodes(r.Context(), userID, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrTOTPNotEnabled) {
			respondError(w, http.StatusBadRequest, "2FA is not enabled")
			return
		}
		if errors.Is(err, service.ErrInvalidTOTPCode) {
			respondError(w, http.StatusBadRequest, "invalid code")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to regenerate backup codes")
		return
	}

	respondJSON(w, http.StatusOK, totpVerifyResponse{BackupCodes: backupCodes})
}
