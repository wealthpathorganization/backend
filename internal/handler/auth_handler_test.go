package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/config"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/service"
)

// MockUserService implements UserServiceInterface for handler tests
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(ctx context.Context, input service.RegisterInput) (*service.AuthResponse, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) Login(ctx context.Context, input service.LoginInput) (*service.AuthResponse, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) UpdateSettings(ctx context.Context, id uuid.UUID, input service.UpdateSettingsInput) (*model.User, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) RefreshToken(ctx context.Context, userID uuid.UUID) (*service.AuthResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) LoginWithTOTP(ctx context.Context, tempToken, code string) (*service.AuthResponse, error) {
	args := m.Called(ctx, tempToken, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) LoginWithBackupCode(ctx context.Context, tempToken, backupCode string) (*service.AuthResponse, error) {
	args := m.Called(ctx, tempToken, backupCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) RegisterWithDeviceInfo(ctx context.Context, input service.RegisterInput, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error) {
	args := m.Called(ctx, input, deviceInfo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) LoginWithDeviceInfo(ctx context.Context, input service.LoginInput, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error) {
	args := m.Called(ctx, input, deviceInfo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) LoginWithTOTPAndDeviceInfo(ctx context.Context, tempToken, code string, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error) {
	args := m.Called(ctx, tempToken, code, deviceInfo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) LoginWithBackupCodeAndDeviceInfo(ctx context.Context, tempToken, backupCode string, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error) {
	args := m.Called(ctx, tempToken, backupCode, deviceInfo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) RefreshAccessToken(ctx context.Context, refreshTokenString string, deviceInfo *model.DeviceInfo) (*service.AuthResponse, error) {
	args := m.Called(ctx, refreshTokenString, deviceInfo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthResponse), args.Error(1)
}

func (m *MockUserService) RevokeRefreshTokenByString(ctx context.Context, refreshTokenString, reason string) error {
	args := m.Called(ctx, refreshTokenString, reason)
	return args.Error(0)
}

func (m *MockUserService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*model.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Session), args.Error(1)
}

func (m *MockUserService) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, reason string) error {
	args := m.Called(ctx, userID, sessionID, reason)
	return args.Error(0)
}

func (m *MockUserService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) (int64, error) {
	args := m.Called(ctx, userID, reason)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserService) GetSessionIDFromRefreshToken(ctx context.Context, refreshTokenString string) (uuid.UUID, error) {
	args := m.Called(ctx, refreshTokenString)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// Note: AuthServiceInterface is defined in auth_handler.go

// testConfig returns a config for testing
func testConfig() *config.Config {
	return &config.Config{
		Env: "development",
		Cookie: config.CookieConfig{
			Domain:   "",
			Secure:   false,
			SameSite: "Strict",
			Path:     "/",
		},
	}
}

func TestAuthHandler_Register_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := service.RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	expectedResp := &service.AuthResponse{
		User: &model.User{
			ID:    uuid.New(),
			Email: input.Email,
			Name:  input.Name,
		},
		Token: "jwt_token",
	}

	mockService.On("RegisterWithDeviceInfo", mock.Anything, input, mock.Anything).Return(expectedResp, nil)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAuthHandler_Register_MissingEmail(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := map[string]string{
		"password": "password123",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "email and password are required")
}

func TestAuthHandler_Register_MissingPassword(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := map[string]string{
		"email": "test@example.com",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "email and password are required")
}

func TestAuthHandler_Register_EmailTaken(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := service.RegisterInput{
		Email:    "existing@example.com",
		Password: "password123",
	}

	mockService.On("RegisterWithDeviceInfo", mock.Anything, input, mock.Anything).Return(nil, service.ErrEmailTaken)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Register_ServiceError(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := service.RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("RegisterWithDeviceInfo", mock.Anything, input, mock.Anything).Return(nil, errors.New("db error"))

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := service.LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	}

	expectedResp := &service.AuthResponse{
		User: &model.User{
			ID:    uuid.New(),
			Email: input.Email,
		},
		Token: "jwt_token",
	}

	mockService.On("LoginWithDeviceInfo", mock.Anything, input, mock.Anything).Return(expectedResp, nil)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTestableAuthHandler_Login_InvalidBody(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := service.LoginInput{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	mockService.On("LoginWithDeviceInfo", mock.Anything, input, mock.Anything).Return(nil, service.ErrInvalidCredentials)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_ServiceError(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	input := service.LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("LoginWithDeviceInfo", mock.Anything, input, mock.Anything).Return(nil, errors.New("db error"))

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Me_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	userID := uuid.New()
	expectedUser := &model.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	mockService.On("GetByID", mock.Anything, userID).Return(expectedUser, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.Me(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	// No userID in context

	rr := httptest.NewRecorder()
	handler.Me(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthHandler_Me_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	userID := uuid.New()
	mockService.On("GetByID", mock.Anything, userID).Return(nil, errors.New("not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.Me(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_UpdateSettings_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	userID := uuid.New()
	name := "New Name"
	currency := "EUR"
	input := service.UpdateSettingsInput{
		Name:     &name,
		Currency: &currency,
	}

	expectedUser := &model.User{
		ID:       userID,
		Email:    "test@example.com",
		Name:     name,
		Currency: currency,
	}

	mockService.On("UpdateSettings", mock.Anything, userID, mock.AnythingOfType("service.UpdateSettingsInput")).Return(expectedUser, nil)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPut, "/api/auth/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.UpdateSettings(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_UpdateSettings_Unauthorized(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	body, _ := json.Marshal(map[string]string{"name": "Test"})
	req := httptest.NewRequest(http.MethodPut, "/api/auth/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No userID in context

	rr := httptest.NewRecorder()
	handler.UpdateSettings(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthHandler_UpdateSettings_InvalidBody(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/api/auth/settings", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.UpdateSettings(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAuthHandler_UpdateSettings_ServiceError(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	userID := uuid.New()
	name := "New Name"
	input := service.UpdateSettingsInput{Name: &name}

	mockService.On("UpdateSettings", mock.Anything, userID, mock.AnythingOfType("service.UpdateSettingsInput")).Return(nil, errors.New("db error"))

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPut, "/api/auth/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.UpdateSettings(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	userID := uuid.New()
	expectedResp := &service.AuthResponse{
		User: &model.User{
			ID:    userID,
			Email: "test@example.com",
			Name:  "Test User",
		},
		Token: "new_jwt_token",
	}

	mockService.On("RefreshToken", mock.Anything, userID).Return(expectedResp, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.RefreshToken(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken_Unauthorized(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	// No userID in context

	rr := httptest.NewRecorder()
	handler.RefreshToken(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthHandler_RefreshToken_ServiceError(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	userID := uuid.New()
	mockService.On("RefreshToken", mock.Anything, userID).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.RefreshToken(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

// ============ Cookie Handling Tests ============

func TestAuthHandler_Login_SetsCookie(t *testing.T) {
	mockService := new(MockUserService)
	cfg := testConfig()
	handler := NewAuthHandlerWithConfig(mockService, cfg)

	input := service.LoginInput{
		Email:      "test@example.com",
		Password:   "password123",
		RememberMe: true,
	}

	expectedResp := &service.AuthResponse{
		User: &model.User{
			ID:    uuid.New(),
			Email: input.Email,
		},
		Token:        "jwt_token",
		RefreshToken: "refresh_token_123",
	}

	mockService.On("LoginWithDeviceInfo", mock.Anything, input, mock.Anything).Return(expectedResp, nil)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that cookie was set
	cookies := rr.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == RefreshTokenCookieName {
			refreshCookie = c
			break
		}
	}
	assert.NotNil(t, refreshCookie)
	assert.Equal(t, "refresh_token_123", refreshCookie.Value)
	assert.True(t, refreshCookie.HttpOnly)

	// Verify refresh token is NOT in response body
	var respBody service.AuthResponse
	json.NewDecoder(rr.Body).Decode(&respBody)
	assert.Empty(t, respBody.RefreshToken)

	mockService.AssertExpectations(t)
}

func TestAuthHandler_RefreshAccessToken_Success(t *testing.T) {
	mockService := new(MockUserService)
	cfg := testConfig()
	handler := NewAuthHandlerWithConfig(mockService, cfg)

	expectedResp := &service.AuthResponse{
		User: &model.User{
			ID:    uuid.New(),
			Email: "test@example.com",
		},
		Token:        "new_access_token",
		RefreshToken: "new_refresh_token",
		ExpiresIn:    900,
	}

	mockService.On("RefreshAccessToken", mock.Anything, "old_refresh_token", mock.Anything).Return(expectedResp, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: "old_refresh_token",
	})

	rr := httptest.NewRecorder()
	handler.RefreshAccessToken(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that new cookie was set
	cookies := rr.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == RefreshTokenCookieName {
			refreshCookie = c
			break
		}
	}
	assert.NotNil(t, refreshCookie)
	assert.Equal(t, "new_refresh_token", refreshCookie.Value)

	mockService.AssertExpectations(t)
}

func TestAuthHandler_RefreshAccessToken_NoCookie(t *testing.T) {
	mockService := new(MockUserService)
	cfg := testConfig()
	handler := NewAuthHandlerWithConfig(mockService, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	// No cookie

	rr := httptest.NewRecorder()
	handler.RefreshAccessToken(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthHandler_RefreshAccessToken_InvalidToken(t *testing.T) {
	mockService := new(MockUserService)
	cfg := testConfig()
	handler := NewAuthHandlerWithConfig(mockService, cfg)

	mockService.On("RefreshAccessToken", mock.Anything, "invalid_token", mock.Anything).Return(nil, service.ErrRefreshTokenInvalid)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: "invalid_token",
	})

	rr := httptest.NewRecorder()
	handler.RefreshAccessToken(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// Check that cookie was cleared
	cookies := rr.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == RefreshTokenCookieName {
			refreshCookie = c
			break
		}
	}
	assert.NotNil(t, refreshCookie)
	assert.Equal(t, "", refreshCookie.Value)
	assert.Equal(t, -1, refreshCookie.MaxAge)

	mockService.AssertExpectations(t)
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	mockService := new(MockUserService)
	cfg := testConfig()
	handler := NewAuthHandlerWithConfig(mockService, cfg)

	mockService.On("RevokeRefreshTokenByString", mock.Anything, "my_refresh_token", "logout").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: "my_refresh_token",
	})

	rr := httptest.NewRecorder()
	handler.Logout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that cookie was cleared
	cookies := rr.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == RefreshTokenCookieName {
			refreshCookie = c
			break
		}
	}
	assert.NotNil(t, refreshCookie)
	assert.Equal(t, "", refreshCookie.Value)
	assert.Equal(t, -1, refreshCookie.MaxAge)

	mockService.AssertExpectations(t)
}

func TestAuthHandler_Logout_NoCookie(t *testing.T) {
	mockService := new(MockUserService)
	cfg := testConfig()
	handler := NewAuthHandlerWithConfig(mockService, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	// No cookie

	rr := httptest.NewRecorder()
	handler.Logout(rr, req)

	// Should still succeed
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestParseUserAgent(t *testing.T) {
	tests := []struct {
		name       string
		userAgent  string
		expBrowser string
		expOS      string
		expDevice  string
	}{
		{
			name:       "Chrome on macOS",
			userAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expBrowser: "Chrome",
			expOS:      "macOS",
			expDevice:  "desktop",
		},
		{
			name:       "Firefox on Windows",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			expBrowser: "Firefox",
			expOS:      "Windows",
			expDevice:  "desktop",
		},
		{
			name:       "Safari on iPhone",
			userAgent:  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			expBrowser: "Safari",
			expOS:      "iOS",
			expDevice:  "mobile",
		},
		{
			name:       "Chrome on Android",
			userAgent:  "Mozilla/5.0 (Linux; Android 10; SM-G981B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			expBrowser: "Chrome",
			expOS:      "Android",
			expDevice:  "mobile",
		},
		{
			name:       "Edge on Windows",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			expBrowser: "Edge",
			expOS:      "Windows",
			expDevice:  "desktop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			browser, os, deviceType := parseUserAgent(tt.userAgent)
			assert.Equal(t, tt.expBrowser, browser)
			assert.Equal(t, tt.expOS, os)
			assert.Equal(t, tt.expDevice, deviceType)
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name      string
		xff       string
		xri       string
		remoteIP  string
		expected  string
	}{
		{
			name:      "X-Forwarded-For single",
			xff:       "192.168.1.1",
			expected:  "192.168.1.1",
		},
		{
			name:      "X-Forwarded-For multiple",
			xff:       "192.168.1.1, 10.0.0.1, 172.16.0.1",
			expected:  "192.168.1.1",
		},
		{
			name:      "X-Real-IP",
			xri:       "192.168.1.2",
			expected:  "192.168.1.2",
		},
		{
			name:      "RemoteAddr",
			remoteIP:  "192.168.1.3:12345",
			expected:  "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			if tt.remoteIP != "" {
				req.RemoteAddr = tt.remoteIP
			}

			ip := getClientIP(req)
			assert.Equal(t, tt.expected, ip)
		})
	}
}
