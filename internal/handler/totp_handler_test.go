package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/service"
)

// MockTOTPService is a mock implementation of TOTPServiceInterface
type MockTOTPService struct {
	mock.Mock
}

func (m *MockTOTPService) Setup(ctx context.Context, userID uuid.UUID) (*service.TOTPSetupResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TOTPSetupResponse), args.Error(1)
}

func (m *MockTOTPService) Verify(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	args := m.Called(ctx, userID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockTOTPService) ValidateCode(ctx context.Context, userID uuid.UUID, code string) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

func (m *MockTOTPService) ValidateBackupCode(ctx context.Context, userID uuid.UUID, code string) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

func (m *MockTOTPService) Disable(ctx context.Context, userID uuid.UUID, code string) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

func (m *MockTOTPService) RegenerateBackupCodes(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
	args := m.Called(ctx, userID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestTOTPHandler_Setup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         uuid.UUID
		mockSetup      func(*MockTOTPService, uuid.UUID)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful setup returns QR code URL and secret",
			userID: uuid.New(),
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Setup", mock.Anything, userID).Return(&service.TOTPSetupResponse{
					Secret:      "JBSWY3DPEHPK3PXP",
					QRCodeURL:   "otpauth://totp/WealthPath:test@example.com?secret=JBSWY3DPEHPK3PXP&issuer=WealthPath",
					ManualEntry: "JBSWY3DPEHPK3PXP (test@example.com)",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp service.TOTPSetupResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.Secret)
				assert.Contains(t, resp.QRCodeURL, "otpauth://totp/")
				assert.NotEmpty(t, resp.ManualEntry)
			},
		},
		{
			name:   "returns error when 2FA already enabled",
			userID: uuid.New(),
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Setup", mock.Anything, userID).Return(nil, service.ErrTOTPAlreadyEnabled)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "already enabled")
			},
		},
		{
			name:           "returns unauthorized when no user ID",
			userID:         uuid.Nil,
			mockSetup:      func(m *MockTOTPService, userID uuid.UUID) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTOTPService)
			tt.mockSetup(mockService, tt.userID)

			handler := NewTOTPHandler(mockService)

			req := httptest.NewRequest(http.MethodPost, "/auth/2fa/setup", nil)
			if tt.userID != uuid.Nil {
				ctx := context.WithValue(req.Context(), UserIDKey, tt.userID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.Setup(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
			mockService.AssertExpectations(t)
		})
	}
}

func TestTOTPHandler_Verify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         uuid.UUID
		requestBody    map[string]string
		mockSetup      func(*MockTOTPService, uuid.UUID)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "successful verify returns backup codes",
			userID:      uuid.New(),
			requestBody: map[string]string{"code": "123456"},
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Verify", mock.Anything, userID, "123456").Return([]string{
					"BACKUP1", "BACKUP2", "BACKUP3", "BACKUP4",
					"BACKUP5", "BACKUP6", "BACKUP7", "BACKUP8",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp totpVerifyResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Len(t, resp.BackupCodes, 8)
			},
		},
		{
			name:        "returns error for invalid code",
			userID:      uuid.New(),
			requestBody: map[string]string{"code": "000000"},
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Verify", mock.Anything, userID, "000000").Return(nil, service.ErrInvalidTOTPCode)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "invalid code")
			},
		},
		{
			name:        "returns error when 2FA not set up",
			userID:      uuid.New(),
			requestBody: map[string]string{"code": "123456"},
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Verify", mock.Anything, userID, "123456").Return(nil, service.ErrTOTPNotSetup)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "not been set up")
			},
		},
		{
			name:           "returns error when code is empty",
			userID:         uuid.New(),
			requestBody:    map[string]string{"code": ""},
			mockSetup:      func(m *MockTOTPService, userID uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "code is required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTOTPService)
			tt.mockSetup(mockService, tt.userID)

			handler := NewTOTPHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != uuid.Nil {
				ctx := context.WithValue(req.Context(), UserIDKey, tt.userID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.Verify(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
			mockService.AssertExpectations(t)
		})
	}
}

func TestTOTPHandler_Disable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         uuid.UUID
		requestBody    map[string]string
		mockSetup      func(*MockTOTPService, uuid.UUID)
		expectedStatus int
	}{
		{
			name:        "successful disable",
			userID:      uuid.New(),
			requestBody: map[string]string{"code": "123456"},
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Disable", mock.Anything, userID, "123456").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "returns error for invalid code",
			userID:      uuid.New(),
			requestBody: map[string]string{"code": "000000"},
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Disable", mock.Anything, userID, "000000").Return(service.ErrInvalidTOTPCode)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "returns error when 2FA not enabled",
			userID:      uuid.New(),
			requestBody: map[string]string{"code": "123456"},
			mockSetup: func(m *MockTOTPService, userID uuid.UUID) {
				m.On("Disable", mock.Anything, userID, "123456").Return(service.ErrTOTPNotEnabled)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTOTPService)
			tt.mockSetup(mockService, tt.userID)

			handler := NewTOTPHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/2fa/disable", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.Disable(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestTOTPHandler_RegenerateBackupCodes(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	mockService := new(MockTOTPService)
	mockService.On("RegenerateBackupCodes", mock.Anything, userID, "123456").Return([]string{
		"NEWCODE1", "NEWCODE2", "NEWCODE3", "NEWCODE4",
		"NEWCODE5", "NEWCODE6", "NEWCODE7", "NEWCODE8",
	}, nil)

	handler := NewTOTPHandler(mockService)

	body, _ := json.Marshal(map[string]string{"code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/backup-codes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RegenerateBackupCodes(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp totpVerifyResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Len(t, resp.BackupCodes, 8)
	mockService.AssertExpectations(t)
}
