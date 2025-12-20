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

// Note: AuthServiceInterface is defined in auth_handler.go

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

	mockService.On("Register", mock.Anything, input).Return(expectedResp, nil)

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

	mockService.On("Register", mock.Anything, input).Return(nil, service.ErrEmailTaken)

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

	mockService.On("Register", mock.Anything, input).Return(nil, errors.New("db error"))

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

	mockService.On("Login", mock.Anything, input).Return(expectedResp, nil)

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

	mockService.On("Login", mock.Anything, input).Return(nil, service.ErrInvalidCredentials)

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

	mockService.On("Login", mock.Anything, input).Return(nil, errors.New("db error"))

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
