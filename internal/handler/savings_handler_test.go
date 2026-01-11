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

// MockSavingsGoalService implements SavingsGoalServiceInterface for testing
type MockSavingsGoalService struct {
	mock.Mock
}

func (m *MockSavingsGoalService) Create(ctx context.Context, userID uuid.UUID, input service.CreateSavingsGoalInput) (*model.SavingsGoal, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SavingsGoal), args.Error(1)
}

func (m *MockSavingsGoalService) Get(ctx context.Context, id uuid.UUID) (*model.SavingsGoal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SavingsGoal), args.Error(1)
}

func (m *MockSavingsGoalService) List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.SavingsGoal), args.Error(1)
}

func (m *MockSavingsGoalService) ListWithProjections(ctx context.Context, userID uuid.UUID) ([]service.SavingsGoalWithProjection, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]service.SavingsGoalWithProjection), args.Error(1)
}

func (m *MockSavingsGoalService) Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateSavingsGoalInput) (*model.SavingsGoal, error) {
	args := m.Called(ctx, id, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SavingsGoal), args.Error(1)
}

func (m *MockSavingsGoalService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockSavingsGoalService) Contribute(ctx context.Context, id, userID uuid.UUID, amount decimal.Decimal) (*model.SavingsGoal, error) {
	args := m.Called(ctx, id, userID, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SavingsGoal), args.Error(1)
}

func TestNewSavingsGoalHandler(t *testing.T) {
	mockService := new(MockSavingsGoalService)
	handler := NewSavingsGoalHandler(mockService)
	assert.NotNil(t, handler)
}

func TestSavingsGoalHandler_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(*MockSavingsGoalService, uuid.UUID)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]interface{}{
				"name":         "Emergency Fund",
				"targetAmount": 10000,
			},
			setupMock: func(m *MockSavingsGoalService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateSavingsGoalInput")).Return(&model.SavingsGoal{
					ID:     uuid.New(),
					UserID: userID,
					Name:   "Emergency Fund",
				}, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid body",
			body:       "invalid",
			setupMock:  func(m *MockSavingsGoalService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: map[string]interface{}{
				"name": "Test",
			},
			setupMock: func(m *MockSavingsGoalService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateSavingsGoalInput")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSavingsGoalService)
			handler := NewSavingsGoalHandler(mockService)
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/savings", bytes.NewReader(body))
			req = req.WithContext(ctxWithUserID(userID))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		goalID     string
		setupMock  func(*MockSavingsGoalService, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			goalID: uuid.New().String(),
			setupMock: func(m *MockSavingsGoalService, id uuid.UUID) {
				m.On("Get", mock.Anything, id).Return(&model.SavingsGoal{ID: id}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			goalID:     "invalid",
			setupMock:  func(m *MockSavingsGoalService, id uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "not found",
			goalID: uuid.New().String(),
			setupMock: func(m *MockSavingsGoalService, id uuid.UUID) {
				m.On("Get", mock.Anything, id).Return(nil, errors.New("not found"))
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSavingsGoalService)
			handler := NewSavingsGoalHandler(mockService)
			goalID, _ := uuid.Parse(tt.goalID)

			tt.setupMock(mockService, goalID)

			req := httptest.NewRequest(http.MethodGet, "/api/savings/"+tt.goalID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.goalID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Get(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalHandler_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(*MockSavingsGoalService, uuid.UUID)
		wantStatus int
	}{
		{
			name: "success",
			setupMock: func(m *MockSavingsGoalService, userID uuid.UUID) {
				m.On("ListWithProjections", mock.Anything, userID).Return([]service.SavingsGoalWithProjection{
					{SavingsGoal: model.SavingsGoal{ID: uuid.New(), Name: "Emergency Fund"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "service error",
			setupMock: func(m *MockSavingsGoalService, userID uuid.UUID) {
				m.On("ListWithProjections", mock.Anything, userID).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSavingsGoalService)
			handler := NewSavingsGoalHandler(mockService)
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			req := httptest.NewRequest(http.MethodGet, "/api/savings", nil)
			req = req.WithContext(ctxWithUserID(userID))
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalHandler_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		goalID     string
		body       interface{}
		setupMock  func(*MockSavingsGoalService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			goalID: uuid.New().String(),
			body:   map[string]interface{}{"name": "Updated"},
			setupMock: func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {
				m.On("Update", mock.Anything, goalID, userID, mock.AnythingOfType("service.UpdateSavingsGoalInput")).Return(&model.SavingsGoal{ID: goalID}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			goalID:     "invalid",
			body:       map[string]interface{}{},
			setupMock:  func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			goalID:     uuid.New().String(),
			body:       "invalid",
			setupMock:  func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			goalID: uuid.New().String(),
			body:   map[string]interface{}{"name": "Updated"},
			setupMock: func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {
				m.On("Update", mock.Anything, goalID, userID, mock.AnythingOfType("service.UpdateSavingsGoalInput")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSavingsGoalService)
			handler := NewSavingsGoalHandler(mockService)
			userID := uuid.New()
			goalID, _ := uuid.Parse(tt.goalID)

			tt.setupMock(mockService, goalID, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/savings/"+tt.goalID, bytes.NewReader(body))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.goalID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Update(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalHandler_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		goalID     string
		setupMock  func(*MockSavingsGoalService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			goalID: uuid.New().String(),
			setupMock: func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {
				m.On("Delete", mock.Anything, goalID, userID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			goalID:     "invalid",
			setupMock:  func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			goalID: uuid.New().String(),
			setupMock: func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {
				m.On("Delete", mock.Anything, goalID, userID).Return(errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSavingsGoalService)
			handler := NewSavingsGoalHandler(mockService)
			userID := uuid.New()
			goalID, _ := uuid.Parse(tt.goalID)

			tt.setupMock(mockService, goalID, userID)

			req := httptest.NewRequest(http.MethodDelete, "/api/savings/"+tt.goalID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.goalID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Delete(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalHandler_Contribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		goalID     string
		body       interface{}
		setupMock  func(*MockSavingsGoalService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			goalID: uuid.New().String(),
			body:   map[string]interface{}{"amount": 500},
			setupMock: func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {
				m.On("Contribute", mock.Anything, goalID, userID, mock.AnythingOfType("decimal.Decimal")).Return(&model.SavingsGoal{ID: goalID}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			goalID:     "invalid",
			body:       map[string]interface{}{"amount": 500},
			setupMock:  func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			goalID:     uuid.New().String(),
			body:       "invalid",
			setupMock:  func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			goalID: uuid.New().String(),
			body:   map[string]interface{}{"amount": 500},
			setupMock: func(m *MockSavingsGoalService, goalID, userID uuid.UUID) {
				m.On("Contribute", mock.Anything, goalID, userID, mock.AnythingOfType("decimal.Decimal")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockSavingsGoalService)
			handler := NewSavingsGoalHandler(mockService)
			userID := uuid.New()
			goalID, _ := uuid.Parse(tt.goalID)

			tt.setupMock(mockService, goalID, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/savings/"+tt.goalID+"/contribute", bytes.NewReader(body))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.goalID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Contribute(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
