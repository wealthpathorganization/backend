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

// MockDebtService implements DebtServiceInterface for testing
type MockDebtService struct {
	mock.Mock
}

func (m *MockDebtService) Create(ctx context.Context, userID uuid.UUID, input service.CreateDebtInput) (*model.Debt, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Debt), args.Error(1)
}

func (m *MockDebtService) Get(ctx context.Context, id uuid.UUID) (*model.Debt, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Debt), args.Error(1)
}

func (m *MockDebtService) List(ctx context.Context, userID uuid.UUID) ([]model.Debt, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Debt), args.Error(1)
}

func (m *MockDebtService) Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateDebtInput) (*model.Debt, error) {
	args := m.Called(ctx, id, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Debt), args.Error(1)
}

func (m *MockDebtService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockDebtService) MakePayment(ctx context.Context, id, userID uuid.UUID, input service.MakePaymentInput) (*model.Debt, error) {
	args := m.Called(ctx, id, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Debt), args.Error(1)
}

func (m *MockDebtService) GetPayoffPlan(ctx context.Context, id uuid.UUID, monthlyPayment decimal.Decimal) (*model.PayoffPlan, error) {
	args := m.Called(ctx, id, monthlyPayment)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PayoffPlan), args.Error(1)
}

func (m *MockDebtService) CalculateInterest(input service.InterestCalculatorInput) (*service.InterestCalculatorResult, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.InterestCalculatorResult), args.Error(1)
}

func (m *MockDebtService) GetDebtSummary(ctx context.Context, userID uuid.UUID) (*service.DebtSummary, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.DebtSummary), args.Error(1)
}

func TestNewDebtHandler(t *testing.T) {
	mockService := new(MockDebtService)
	handler := NewDebtHandler(mockService)
	assert.NotNil(t, handler)
}

func TestDebtHandler_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(*MockDebtService, uuid.UUID)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]interface{}{
				"name":           "Mortgage",
				"type":           "mortgage",
				"originalAmount": 200000,
				"interestRate":   4.5,
			},
			setupMock: func(m *MockDebtService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateDebtInput")).Return(&model.Debt{
					ID:     uuid.New(),
					UserID: userID,
					Name:   "Mortgage",
				}, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid body",
			body:       "invalid",
			setupMock:  func(m *MockDebtService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: map[string]interface{}{
				"name": "Test",
			},
			setupMock: func(m *MockDebtService, userID uuid.UUID) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateDebtInput")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDebtService)
			handler := NewDebtHandler(mockService)
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/debts", bytes.NewReader(body))
			req = req.WithContext(ctxWithUserID(userID))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDebtHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		debtID     string
		setupMock  func(*MockDebtService, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			debtID: uuid.New().String(),
			setupMock: func(m *MockDebtService, id uuid.UUID) {
				m.On("Get", mock.Anything, id).Return(&model.Debt{ID: id}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			debtID:     "invalid",
			setupMock:  func(m *MockDebtService, id uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "not found",
			debtID: uuid.New().String(),
			setupMock: func(m *MockDebtService, id uuid.UUID) {
				m.On("Get", mock.Anything, id).Return(nil, errors.New("not found"))
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDebtService)
			handler := NewDebtHandler(mockService)
			debtID, _ := uuid.Parse(tt.debtID)

			tt.setupMock(mockService, debtID)

			req := httptest.NewRequest(http.MethodGet, "/api/debts/"+tt.debtID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.debtID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Get(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDebtHandler_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(*MockDebtService, uuid.UUID)
		wantStatus int
	}{
		{
			name: "success",
			setupMock: func(m *MockDebtService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID).Return([]model.Debt{
					{ID: uuid.New(), Name: "Mortgage"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "service error",
			setupMock: func(m *MockDebtService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDebtService)
			handler := NewDebtHandler(mockService)
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			req := httptest.NewRequest(http.MethodGet, "/api/debts", nil)
			req = req.WithContext(ctxWithUserID(userID))
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDebtHandler_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		debtID     string
		body       interface{}
		setupMock  func(*MockDebtService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			debtID: uuid.New().String(),
			body:   map[string]interface{}{"name": "Updated"},
			setupMock: func(m *MockDebtService, debtID, userID uuid.UUID) {
				m.On("Update", mock.Anything, debtID, userID, mock.AnythingOfType("service.UpdateDebtInput")).Return(&model.Debt{ID: debtID}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			debtID:     "invalid",
			body:       map[string]interface{}{},
			setupMock:  func(m *MockDebtService, debtID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			debtID:     uuid.New().String(),
			body:       "invalid",
			setupMock:  func(m *MockDebtService, debtID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			debtID: uuid.New().String(),
			body:   map[string]interface{}{"name": "Updated"},
			setupMock: func(m *MockDebtService, debtID, userID uuid.UUID) {
				m.On("Update", mock.Anything, debtID, userID, mock.AnythingOfType("service.UpdateDebtInput")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDebtService)
			handler := NewDebtHandler(mockService)
			userID := uuid.New()
			debtID, _ := uuid.Parse(tt.debtID)

			tt.setupMock(mockService, debtID, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/debts/"+tt.debtID, bytes.NewReader(body))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.debtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Update(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDebtHandler_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		debtID     string
		setupMock  func(*MockDebtService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			debtID: uuid.New().String(),
			setupMock: func(m *MockDebtService, debtID, userID uuid.UUID) {
				m.On("Delete", mock.Anything, debtID, userID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			debtID:     "invalid",
			setupMock:  func(m *MockDebtService, debtID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			debtID: uuid.New().String(),
			setupMock: func(m *MockDebtService, debtID, userID uuid.UUID) {
				m.On("Delete", mock.Anything, debtID, userID).Return(errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDebtService)
			handler := NewDebtHandler(mockService)
			userID := uuid.New()
			debtID, _ := uuid.Parse(tt.debtID)

			tt.setupMock(mockService, debtID, userID)

			req := httptest.NewRequest(http.MethodDelete, "/api/debts/"+tt.debtID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.debtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.Delete(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDebtHandler_MakePayment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		debtID     string
		body       interface{}
		setupMock  func(*MockDebtService, uuid.UUID, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			debtID: uuid.New().String(),
			body:   map[string]interface{}{"amount": 500},
			setupMock: func(m *MockDebtService, debtID, userID uuid.UUID) {
				m.On("MakePayment", mock.Anything, debtID, userID, mock.AnythingOfType("service.MakePaymentInput")).Return(&model.Debt{ID: debtID}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			debtID:     "invalid",
			body:       map[string]interface{}{"amount": 500},
			setupMock:  func(m *MockDebtService, debtID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			debtID:     uuid.New().String(),
			body:       "invalid",
			setupMock:  func(m *MockDebtService, debtID, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			debtID: uuid.New().String(),
			body:   map[string]interface{}{"amount": 500},
			setupMock: func(m *MockDebtService, debtID, userID uuid.UUID) {
				m.On("MakePayment", mock.Anything, debtID, userID, mock.AnythingOfType("service.MakePaymentInput")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDebtService)
			handler := NewDebtHandler(mockService)
			userID := uuid.New()
			debtID, _ := uuid.Parse(tt.debtID)

			tt.setupMock(mockService, debtID, userID)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/debts/"+tt.debtID+"/payments", bytes.NewReader(body))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.debtID)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.MakePayment(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDebtHandler_GetPayoffPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		debtID     string
		query      string
		setupMock  func(*MockDebtService, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success",
			debtID: uuid.New().String(),
			query:  "?monthlyPayment=500",
			setupMock: func(m *MockDebtService, debtID uuid.UUID) {
				m.On("GetPayoffPlan", mock.Anything, debtID, mock.AnythingOfType("decimal.Decimal")).Return(&model.PayoffPlan{
					DebtID:         debtID,
					MonthsToPayoff: 24,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			debtID:     "invalid",
			query:      "",
			setupMock:  func(m *MockDebtService, debtID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			debtID: uuid.New().String(),
			query:  "",
			setupMock: func(m *MockDebtService, debtID uuid.UUID) {
				m.On("GetPayoffPlan", mock.Anything, debtID, mock.AnythingOfType("decimal.Decimal")).Return(nil, errors.New("error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDebtService)
			handler := NewDebtHandler(mockService)
			debtID, _ := uuid.Parse(tt.debtID)

			tt.setupMock(mockService, debtID)

			req := httptest.NewRequest(http.MethodGet, "/api/debts/"+tt.debtID+"/payoff-plan"+tt.query, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.debtID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.GetPayoffPlan(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDebtHandler_InterestCalculator(t *testing.T) {
	t.Parallel()

	mockService := new(MockDebtService)
	handler := NewDebtHandler(mockService)

	mockService.On("CalculateInterest", mock.AnythingOfType("service.InterestCalculatorInput")).Return(&service.InterestCalculatorResult{
		MonthlyPayment: decimal.NewFromFloat(500),
		TotalPayment:   decimal.NewFromFloat(6000),
		TotalInterest:  decimal.NewFromFloat(1000),
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/debts/calculator?principal=5000&interestRate=12&termMonths=12", nil)
	w := httptest.NewRecorder()

	handler.InterestCalculator(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}
