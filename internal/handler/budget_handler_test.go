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

// MockBudgetService implements BudgetServiceInterface for testing
type MockBudgetService struct {
	mock.Mock
}

func (m *MockBudgetService) Create(ctx context.Context, userID uuid.UUID, input service.CreateBudgetInput) (*model.Budget, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Budget), args.Error(1)
}

func (m *MockBudgetService) Get(ctx context.Context, id uuid.UUID) (*model.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Budget), args.Error(1)
}

func (m *MockBudgetService) List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Budget), args.Error(1)
}

func (m *MockBudgetService) ListWithSpent(ctx context.Context, userID uuid.UUID) ([]model.BudgetWithSpent, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.BudgetWithSpent), args.Error(1)
}

func (m *MockBudgetService) Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateBudgetInput) (*model.Budget, error) {
	args := m.Called(ctx, id, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Budget), args.Error(1)
}

func (m *MockBudgetService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

// Helper to create context with userID
func ctxWithUserID(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), UserIDKey, userID)
}

func TestNewBudgetHandler(t *testing.T) {
	mockService := new(MockBudgetService)
	handler := NewBudgetHandler(mockService)
	assert.NotNil(t, handler)
}

func TestBudgetHandler_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(*MockBudgetService, uuid.UUID)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]interface{}{
				"category": "Food",
				"amount":   500,
				"period":   "monthly",
			},
			setupMock: func(m *MockBudgetService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateBudgetInput")).Return(&model.Budget{
					ID:       uuid.New(),
					UserID:   userID,
					Category: "Food",
					Amount:   decimal.NewFromFloat(500),
				}, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid body",
			body:       "invalid json",
			setupMock:  func(m *MockBudgetService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: map[string]interface{}{
				"category": "Food",
				"amount":   500,
			},
			setupMock: func(m *MockBudgetService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateBudgetInput")).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockBudgetService)
			handler := NewBudgetHandler(mockService)
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(ctxWithUserID(userID))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestBudgetHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		budgetID   string
		setupMock  func(*MockBudgetService, uuid.UUID)
		wantStatus int
	}{
		{
			name:     "success",
			budgetID: uuid.New().String(),
			setupMock: func(m *MockBudgetService, id uuid.UUID) {
				m.On("Get", mock.Anything, id).Return(&model.Budget{
					ID:       id,
					Category: "Food",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			budgetID:   "invalid-uuid",
			setupMock:  func(m *MockBudgetService, id uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "not found",
			budgetID: uuid.New().String(),
			setupMock: func(m *MockBudgetService, id uuid.UUID) {
				m.On("Get", mock.Anything, id).Return(nil, errors.New("not found"))
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockBudgetService)
			handler := NewBudgetHandler(mockService)

			budgetID, _ := uuid.Parse(tt.budgetID)
			tt.setupMock(mockService, budgetID)

			req := httptest.NewRequest(http.MethodGet, "/api/budgets/"+tt.budgetID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.budgetID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Get(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestBudgetHandler_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(*MockBudgetService, uuid.UUID)
		wantStatus int
	}{
		{
			name: "success",
			setupMock: func(m *MockBudgetService, userID uuid.UUID) {
				m.On("ListWithSpent", mock.Anything, userID).Return([]model.BudgetWithSpent{
					{Budget: model.Budget{ID: uuid.New(), Category: "Food"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "service error",
			setupMock: func(m *MockBudgetService, userID uuid.UUID) {
				m.On("ListWithSpent", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockBudgetService)
			handler := NewBudgetHandler(mockService)
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			req := httptest.NewRequest(http.MethodGet, "/api/budgets", nil)
			req = req.WithContext(ctxWithUserID(userID))
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestBudgetHandler_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		budgetID   string
		body       interface{}
		setupMock  func(*MockBudgetService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:     "success",
			budgetID: uuid.New().String(),
			body:     map[string]interface{}{"category": "Shopping", "amount": 600},
			setupMock: func(m *MockBudgetService, budgetID, userID uuid.UUID) {
				m.On("Update", mock.Anything, budgetID, userID, mock.AnythingOfType("service.UpdateBudgetInput")).Return(&model.Budget{
					ID:       budgetID,
					Category: "Shopping",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			budgetID:   "invalid-uuid",
			body:       map[string]interface{}{"category": "Shopping"},
			setupMock:  func(m *MockBudgetService, budgetID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			budgetID:   uuid.New().String(),
			body:       "invalid",
			setupMock:  func(m *MockBudgetService, budgetID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "service error",
			budgetID: uuid.New().String(),
			body:     map[string]interface{}{"category": "Shopping"},
			setupMock: func(m *MockBudgetService, budgetID, userID uuid.UUID) {
				m.On("Update", mock.Anything, budgetID, userID, mock.AnythingOfType("service.UpdateBudgetInput")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockBudgetService)
			handler := NewBudgetHandler(mockService)
			userID := uuid.New()
			budgetID, _ := uuid.Parse(tt.budgetID)

			tt.setupMock(mockService, budgetID, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/budgets/"+tt.budgetID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.budgetID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Update(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestBudgetHandler_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		budgetID   string
		setupMock  func(*MockBudgetService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:     "success",
			budgetID: uuid.New().String(),
			setupMock: func(m *MockBudgetService, budgetID, userID uuid.UUID) {
				m.On("Delete", mock.Anything, budgetID, userID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			budgetID:   "invalid-uuid",
			setupMock:  func(m *MockBudgetService, budgetID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "service error",
			budgetID: uuid.New().String(),
			setupMock: func(m *MockBudgetService, budgetID, userID uuid.UUID) {
				m.On("Delete", mock.Anything, budgetID, userID).Return(errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockBudgetService)
			handler := NewBudgetHandler(mockService)
			userID := uuid.New()
			budgetID, _ := uuid.Parse(tt.budgetID)

			tt.setupMock(mockService, budgetID, userID)

			req := httptest.NewRequest(http.MethodDelete, "/api/budgets/"+tt.budgetID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.budgetID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Delete(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
