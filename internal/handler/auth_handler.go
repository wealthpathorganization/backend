package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/service"
)

// AuthServiceInterface for handler testing
type AuthServiceInterface interface {
	Register(ctx context.Context, input service.RegisterInput) (*service.AuthResponse, error)
	Login(ctx context.Context, input service.LoginInput) (*service.AuthResponse, error)
	LoginWithTOTP(ctx context.Context, tempToken, code string) (*service.AuthResponse, error)
	LoginWithBackupCode(ctx context.Context, tempToken, backupCode string) (*service.AuthResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateSettings(ctx context.Context, userID uuid.UUID, input service.UpdateSettingsInput) (*model.User, error)
	RefreshToken(ctx context.Context, userID uuid.UUID) (*service.AuthResponse, error)
}

type AuthHandler struct {
	userService AuthServiceInterface
}

func NewAuthHandler(userService AuthServiceInterface) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param input body service.RegisterInput true "Registration data"
// @Success 201 {object} service.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Email already in use"
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Email == "" || input.Password == "" {
		respondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	resp, err := h.userService.Register(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			respondError(w, http.StatusConflict, "email already in use")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to register")
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

// Login godoc
// @Summary Login user
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param input body service.LoginInput true "Login credentials"
// @Success 200 {object} service.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.userService.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// Me godoc
// @Summary Get current user
// @Description Get the authenticated user's profile
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.User
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// UpdateSettings godoc
// @Summary Update user settings
// @Description Update the authenticated user's profile settings
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.UpdateSettingsInput true "Settings to update"
// @Success 200 {object} model.User
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/settings [put]
func (h *AuthHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input service.UpdateSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userService.UpdateSettings(r.Context(), userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update settings: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// RefreshToken godoc
// @Summary Refresh authentication token
// @Description Get a new JWT token using an existing valid token. Useful for mobile apps to maintain sessions.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} service.AuthResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.userService.RefreshToken(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to refresh token")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

type loginTOTPRequest struct {
	TempToken string `json:"tempToken"`
	Code      string `json:"code"`
}

// LoginWithTOTP godoc
// @Summary Complete login with 2FA
// @Description Complete authentication using a TOTP code after initial login
// @Tags auth
// @Accept json
// @Produce json
// @Param input body loginTOTPRequest true "Temp token and TOTP code"
// @Success 200 {object} service.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Invalid credentials or code"
// @Failure 500 {object} ErrorResponse
// @Router /auth/login/2fa [post]
func (h *AuthHandler) LoginWithTOTP(w http.ResponseWriter, r *http.Request) {
	var input loginTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.TempToken == "" || input.Code == "" {
		respondError(w, http.StatusBadRequest, "tempToken and code are required")
		return
	}

	resp, err := h.userService.LoginWithTOTP(r.Context(), input.TempToken, input.Code)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid credentials or code")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to complete login")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

type loginBackupCodeRequest struct {
	TempToken  string `json:"tempToken"`
	BackupCode string `json:"backupCode"`
}

// LoginWithBackupCode godoc
// @Summary Complete login with backup code
// @Description Complete authentication using a backup code after initial login
// @Tags auth
// @Accept json
// @Produce json
// @Param input body loginBackupCodeRequest true "Temp token and backup code"
// @Success 200 {object} service.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Invalid credentials or code"
// @Failure 500 {object} ErrorResponse
// @Router /auth/login/2fa/backup [post]
func (h *AuthHandler) LoginWithBackupCode(w http.ResponseWriter, r *http.Request) {
	var input loginBackupCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.TempToken == "" || input.BackupCode == "" {
		respondError(w, http.StatusBadRequest, "tempToken and backupCode are required")
		return
	}

	resp, err := h.userService.LoginWithBackupCode(r.Context(), input.TempToken, input.BackupCode)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid credentials or backup code")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to complete login")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}
