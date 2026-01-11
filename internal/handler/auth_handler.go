package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/config"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/service"
)

// RefreshTokenCookieName is the name of the refresh token cookie.
const RefreshTokenCookieName = "refresh_token"

// AuthServiceInterface for handler testing
type AuthServiceInterface interface {
	Register(ctx context.Context, input service.RegisterInput) (*service.AuthResponse, error)
	RegisterWithDeviceInfo(ctx context.Context, input service.RegisterInput, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error)
	Login(ctx context.Context, input service.LoginInput) (*service.AuthResponse, error)
	LoginWithDeviceInfo(ctx context.Context, input service.LoginInput, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error)
	LoginWithTOTP(ctx context.Context, tempToken, code string) (*service.AuthResponse, error)
	LoginWithTOTPAndDeviceInfo(ctx context.Context, tempToken, code string, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error)
	LoginWithBackupCode(ctx context.Context, tempToken, backupCode string) (*service.AuthResponse, error)
	LoginWithBackupCodeAndDeviceInfo(ctx context.Context, tempToken, backupCode string, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateSettings(ctx context.Context, userID uuid.UUID, input service.UpdateSettingsInput) (*model.User, error)
	RefreshToken(ctx context.Context, userID uuid.UUID) (*service.AuthResponse, error)
	RefreshAccessToken(ctx context.Context, refreshTokenString string, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error)
	RevokeRefreshTokenByString(ctx context.Context, refreshTokenString, reason string) error
	GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*model.Session, error)
	RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, reason string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) (int64, error)
	GetSessionIDFromRefreshToken(ctx context.Context, refreshTokenString string) (uuid.UUID, error)
}

type AuthHandler struct {
	userService AuthServiceInterface
	cfg         *config.Config
}

// NewAuthHandler creates a new AuthHandler (backward compatible, no cookie support).
func NewAuthHandler(userService AuthServiceInterface) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// NewAuthHandlerWithConfig creates a new AuthHandler with config for cookie support.
func NewAuthHandlerWithConfig(userService AuthServiceInterface, cfg *config.Config) *AuthHandler {
	return &AuthHandler{userService: userService, cfg: cfg}
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

	// Extract device info from request
	deviceInfo := h.extractDeviceInfo(r)

	resp, err := h.userService.RegisterWithDeviceInfo(r.Context(), input, deviceInfo)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			respondError(w, http.StatusConflict, "email already in use")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to register")
		return
	}

	// Set refresh token cookie if available
	if resp.RefreshToken != "" && h.cfg != nil {
		h.setRefreshTokenCookie(w, resp.RefreshToken, input.RememberMe)
		// Don't send refresh token in response body (it's in the cookie)
		resp.RefreshToken = ""
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

	// Extract device info from request
	deviceInfo := h.extractDeviceInfo(r)

	resp, err := h.userService.LoginWithDeviceInfo(r.Context(), input, deviceInfo)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		// Log the actual error for debugging
		fmt.Printf("[ERROR] Login failed: %v\n", err)
		respondError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	// Set refresh token cookie if available (not for 2FA flow)
	if resp.RefreshToken != "" && h.cfg != nil {
		h.setRefreshTokenCookie(w, resp.RefreshToken, input.RememberMe)
		// Don't send refresh token in response body (it's in the cookie)
		resp.RefreshToken = ""
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
// @Summary Refresh authentication token (legacy)
// @Description Get a new JWT token using an existing valid token. Useful for mobile apps to maintain sessions.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} service.AuthResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh-legacy [post]
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

// RefreshAccessToken godoc
// @Summary Refresh access token using refresh token cookie
// @Description Exchange a refresh token (from HttpOnly cookie) for a new access token
// @Tags auth
// @Produce json
// @Success 200 {object} service.AuthResponse
// @Failure 401 {object} ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshToken, err := h.getRefreshTokenFromCookie(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "refresh token not found")
		return
	}

	// Extract device info
	deviceInfo := h.extractDeviceInfo(r)

	resp, err := h.userService.RefreshAccessToken(r.Context(), refreshToken, deviceInfo)
	if err != nil {
		if errors.Is(err, service.ErrRefreshTokenInvalid) {
			h.clearRefreshTokenCookie(w)
			respondError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		if errors.Is(err, service.ErrRefreshTokenExpired) {
			h.clearRefreshTokenCookie(w)
			respondError(w, http.StatusUnauthorized, "refresh token expired")
			return
		}
		if errors.Is(err, service.ErrRefreshTokenRevoked) {
			h.clearRefreshTokenCookie(w)
			respondError(w, http.StatusUnauthorized, "refresh token revoked")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to refresh token")
		return
	}

	// Set new refresh token cookie (token rotation)
	if resp.RefreshToken != "" && h.cfg != nil {
		// Preserve the remember me setting based on the new token expiry
		h.setRefreshTokenCookie(w, resp.RefreshToken, true) // Default to long expiry for rotated tokens
		resp.RefreshToken = ""
	}

	respondJSON(w, http.StatusOK, resp)
}

// Logout godoc
// @Summary Logout user
// @Description Revoke the refresh token and clear the cookie
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshToken, err := h.getRefreshTokenFromCookie(r)
	if err == nil && refreshToken != "" {
		// Revoke the refresh token (ignore errors, we're logging out anyway)
		_ = h.userService.RevokeRefreshTokenByString(r.Context(), refreshToken, "logout")
	}

	// Clear the cookie
	h.clearRefreshTokenCookie(w)

	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
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

	// Extract device info
	deviceInfo := h.extractDeviceInfo(r)

	resp, err := h.userService.LoginWithTOTPAndDeviceInfo(r.Context(), input.TempToken, input.Code, deviceInfo)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid credentials or code")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to complete login")
		return
	}

	// Set refresh token cookie if available
	if resp.RefreshToken != "" && h.cfg != nil {
		h.setRefreshTokenCookie(w, resp.RefreshToken, true) // RememberMe preference is in temp token
		resp.RefreshToken = ""
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

	// Extract device info
	deviceInfo := h.extractDeviceInfo(r)

	resp, err := h.userService.LoginWithBackupCodeAndDeviceInfo(r.Context(), input.TempToken, input.BackupCode, deviceInfo)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid credentials or backup code")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to complete login")
		return
	}

	// Set refresh token cookie if available
	if resp.RefreshToken != "" && h.cfg != nil {
		h.setRefreshTokenCookie(w, resp.RefreshToken, true) // RememberMe preference is in temp token
		resp.RefreshToken = ""
	}

	respondJSON(w, http.StatusOK, resp)
}

// ============ Cookie Helper Functions ============

// setRefreshTokenCookie sets the refresh token as an HttpOnly cookie.
func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, refreshToken string, rememberMe bool) {
	if h.cfg == nil {
		return
	}

	// Determine expiry based on rememberMe
	maxAge := 7 * 24 * 60 * 60 // 7 days in seconds
	if rememberMe {
		maxAge = 30 * 24 * 60 * 60 // 30 days in seconds
	}

	sameSite := http.SameSiteStrictMode
	switch strings.ToLower(h.cfg.Cookie.SameSite) {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    refreshToken,
		Path:     h.cfg.Cookie.Path,
		Domain:   h.cfg.Cookie.Domain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cfg.Cookie.Secure,
		SameSite: sameSite,
	}

	http.SetCookie(w, cookie)
}

// clearRefreshTokenCookie clears the refresh token cookie.
func (h *AuthHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	if h.cfg == nil {
		// Fallback for tests without config
		http.SetCookie(w, &http.Cookie{
			Name:     RefreshTokenCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		return
	}

	sameSite := http.SameSiteStrictMode
	switch strings.ToLower(h.cfg.Cookie.SameSite) {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     h.cfg.Cookie.Path,
		Domain:   h.cfg.Cookie.Domain,
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
		Secure:   h.cfg.Cookie.Secure,
		SameSite: sameSite,
	}

	http.SetCookie(w, cookie)
}

// getRefreshTokenFromCookie extracts the refresh token from the cookie.
func (h *AuthHandler) getRefreshTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		return "", err
	}
	if cookie.Value == "" {
		return "", http.ErrNoCookie
	}
	return cookie.Value, nil
}

// extractDeviceInfo extracts device information from the request.
func (h *AuthHandler) extractDeviceInfo(r *http.Request) *model.DeviceInfo {
	userAgent := r.UserAgent()
	if userAgent == "" {
		return nil
	}

	// Parse User-Agent to extract browser and OS
	browser, os, deviceType := parseUserAgent(userAgent)

	// Get client IP
	ip := getClientIP(r)

	return &model.DeviceInfo{
		Browser:    browser,
		OS:         os,
		DeviceType: deviceType,
		IP:         ip,
	}
}

// parseUserAgent extracts browser, OS, and device type from User-Agent string.
func parseUserAgent(ua string) (browser, os, deviceType string) {
	ua = strings.ToLower(ua)

	// Detect browser
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome/") && !strings.Contains(ua, "chromium/"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome/"):
		browser = "Safari"
	case strings.Contains(ua, "opera/") || strings.Contains(ua, "opr/"):
		browser = "Opera"
	default:
		browser = "Unknown"
	}

	// Detect OS (order matters - check mobile OS before desktop equivalents)
	switch {
	case strings.Contains(ua, "android"):
		os = "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		os = "iOS"
	case strings.Contains(ua, "windows"):
		os = "Windows"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		os = "macOS"
	case strings.Contains(ua, "linux"):
		os = "Linux"
	default:
		os = "Unknown"
	}

	// Detect device type
	switch {
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone"):
		deviceType = "mobile"
	case strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad"):
		deviceType = "tablet"
	default:
		deviceType = "desktop"
	}

	return browser, os, deviceType
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
