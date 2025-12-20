package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/service"
)

// MockRecurringService implements RecurringServiceInterface for testing
type MockRecurringService struct {
	mock.Mock
}

func (m *MockRecurringService) Create(ctx context.Context, userID uuid.UUID, input service.CreateRecurringInput) (*model.RecurringTransaction, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringService) GetByID(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringService) Update(ctx context.Context, userID, id uuid.UUID, input service.UpdateRecurringInput) (*model.RecurringTransaction, error) {
	args := m.Called(ctx, userID, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockRecurringService) Pause(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringService) Resume(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringService) GetUpcoming(ctx context.Context, userID uuid.UUID, limit int) ([]model.UpcomingBill, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UpcomingBill), args.Error(1)
}

func TestNewRecurringHandler(t *testing.T) {
	mockService := new(MockRecurringService)
	handler := NewRecurringHandler(mockService)
	assert.NotNil(t, handler)
}

func TestRecurringHandler_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		body       interface{}
		setupMock  func(*MockRecurringService, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			body: map[string]interface{}{
				"type":        "expense",
				"amount":      100,
				"category":    "Utilities",
				"description": "Electricity",
				"frequency":   "monthly",
			},
			setupMock: func(m *MockRecurringService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateRecurringInput")).Return(&model.RecurringTransaction{
					ID:          uuid.New(),
					UserID:      userID,
					Description: "Electricity",
				}, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "unauthorized - nil userID",
			userID:     uuid.Nil,
			body:       map[string]interface{}{},
			setupMock:  func(m *MockRecurringService, userID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid body",
			userID:     uuid.New(),
			body:       "invalid",
			setupMock:  func(m *MockRecurringService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: uuid.New(),
			body: map[string]interface{}{
				"type":      "expense",
				"amount":    100,
				"frequency": "monthly",
			},
			setupMock: func(m *MockRecurringService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateRecurringInput")).Return(nil, errors.New("validation error"))
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)

			tt.setupMock(mockService, tt.userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/recurring", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(ctxWithUserID(tt.userID))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringHandler_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMock  func(*MockRecurringService, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			setupMock: func(m *MockRecurringService, userID uuid.UUID) {
				m.On("GetByUserID", mock.Anything, userID).Return([]model.RecurringTransaction{
					{ID: uuid.New(), Description: "Electricity"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			userID:     uuid.Nil,
			setupMock:  func(m *MockRecurringService, userID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "service error",
			userID: uuid.New(),
			setupMock: func(m *MockRecurringService, userID uuid.UUID) {
				m.On("GetByUserID", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)

			tt.setupMock(mockService, tt.userID)

			req := httptest.NewRequest(http.MethodGet, "/api/recurring", nil)
			req = req.WithContext(ctxWithUserID(tt.userID))
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		rtID       string
		setupMock  func(*MockRecurringService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("GetByID", mock.Anything, userID, rtID).Return(&model.RecurringTransaction{
					ID:     rtID,
					UserID: userID,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			userID:     uuid.Nil,
			rtID:       uuid.New().String(),
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid uuid",
			userID:     uuid.New(),
			rtID:       "invalid",
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "not found",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("GetByID", mock.Anything, userID, rtID).Return(nil, errors.New("not found"))
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)
			rtID, _ := uuid.Parse(tt.rtID)

			tt.setupMock(mockService, tt.userID, rtID)

			req := httptest.NewRequest(http.MethodGet, "/api/recurring/"+tt.rtID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.rtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(tt.userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Get(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringHandler_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		rtID       string
		body       interface{}
		setupMock  func(*MockRecurringService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			body:   map[string]interface{}{"description": "Updated"},
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("Update", mock.Anything, userID, rtID, mock.AnythingOfType("service.UpdateRecurringInput")).Return(&model.RecurringTransaction{
					ID:          rtID,
					Description: "Updated",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			userID:     uuid.Nil,
			rtID:       uuid.New().String(),
			body:       map[string]interface{}{},
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid uuid",
			userID:     uuid.New(),
			rtID:       "invalid",
			body:       map[string]interface{}{},
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			userID:     uuid.New(),
			rtID:       uuid.New().String(),
			body:       "invalid",
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)
			rtID, _ := uuid.Parse(tt.rtID)

			tt.setupMock(mockService, tt.userID, rtID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/recurring/"+tt.rtID, bytes.NewReader(body))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.rtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(tt.userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Update(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringHandler_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		rtID       string
		setupMock  func(*MockRecurringService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("Delete", mock.Anything, userID, rtID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "unauthorized",
			userID:     uuid.Nil,
			rtID:       uuid.New().String(),
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "not found",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("Delete", mock.Anything, userID, rtID).Return(errors.New("not found"))
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)
			rtID, _ := uuid.Parse(tt.rtID)

			tt.setupMock(mockService, tt.userID, rtID)

			req := httptest.NewRequest(http.MethodDelete, "/api/recurring/"+tt.rtID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.rtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(tt.userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Delete(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringHandler_Pause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		rtID       string
		setupMock  func(*MockRecurringService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("Pause", mock.Anything, userID, rtID).Return(&model.RecurringTransaction{
					ID:       rtID,
					IsActive: false,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			userID:     uuid.Nil,
			rtID:       uuid.New().String(),
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid uuid",
			userID:     uuid.New(),
			rtID:       "invalid",
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("Pause", mock.Anything, userID, rtID).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)
			rtID, _ := uuid.Parse(tt.rtID)

			tt.setupMock(mockService, tt.userID, rtID)

			req := httptest.NewRequest(http.MethodPost, "/api/recurring/"+tt.rtID+"/pause", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.rtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(tt.userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Pause(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringHandler_Resume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		rtID       string
		setupMock  func(*MockRecurringService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("Resume", mock.Anything, userID, rtID).Return(&model.RecurringTransaction{
					ID:       rtID,
					IsActive: true,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			userID:     uuid.Nil,
			rtID:       uuid.New().String(),
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid uuid",
			userID:     uuid.New(),
			rtID:       "invalid",
			setupMock:  func(m *MockRecurringService, userID, rtID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: uuid.New(),
			rtID:   uuid.New().String(),
			setupMock: func(m *MockRecurringService, userID, rtID uuid.UUID) {
				m.On("Resume", mock.Anything, userID, rtID).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)
			rtID, _ := uuid.Parse(tt.rtID)

			tt.setupMock(mockService, tt.userID, rtID)

			req := httptest.NewRequest(http.MethodPost, "/api/recurring/"+tt.rtID+"/resume", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.rtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(tt.userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Resume(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringHandler_Upcoming(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMock  func(*MockRecurringService, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			userID: uuid.New(),
			setupMock: func(m *MockRecurringService, userID uuid.UUID) {
				m.On("GetUpcoming", mock.Anything, userID, 10).Return([]model.UpcomingBill{
					{ID: uuid.New(), Description: "Electricity", Amount: decimal.NewFromFloat(100)},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			userID:     uuid.Nil,
			setupMock:  func(m *MockRecurringService, userID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "service error",
			userID: uuid.New(),
			setupMock: func(m *MockRecurringService, userID uuid.UUID) {
				m.On("GetUpcoming", mock.Anything, userID, 10).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockRecurringService)
			handler := NewRecurringHandler(mockService)

			tt.setupMock(mockService, tt.userID)

			req := httptest.NewRequest(http.MethodGet, "/api/recurring/upcoming", nil)
			req = req.WithContext(ctxWithUserID(tt.userID))
			w := httptest.NewRecorder()

			handler.Upcoming(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
