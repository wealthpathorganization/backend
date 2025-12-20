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
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/internal/service"
)

// MockTransactionService implements a mock transaction service for handler tests
type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) Create(ctx context.Context, userID uuid.UUID, input service.CreateTransactionInput) (*model.Transaction, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Transaction), args.Error(1)
}

func (m *MockTransactionService) Get(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Transaction), args.Error(1)
}

func (m *MockTransactionService) List(ctx context.Context, userID uuid.UUID, input service.ListTransactionsInput) ([]model.Transaction, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (m *MockTransactionService) Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateTransactionInput) (*model.Transaction, error) {
	args := m.Called(ctx, id, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Transaction), args.Error(1)
}

func (m *MockTransactionService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

// Note: TransactionHandlerServiceInterface is defined in transaction_handler.go

func TestTransactionHandler_Create_Success(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	userID := uuid.New()
	expectedTx := &model.Transaction{
		ID:       uuid.New(),
		UserID:   userID,
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(100),
		Currency: "USD",
		Category: "Food",
	}

	mockService.On("Create", mock.Anything, userID, mock.Anything).Return(expectedTx, nil)

	body := []byte(`{"type":"expense","amount":"100","currency":"USD","category":"Food","description":"Lunch"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_Create_InvalidBody(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTransactionHandler_Create_MissingType(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	input := map[string]interface{}{
		"amount":   100,
		"category": "Food",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "type is required")
}

func TestTransactionHandler_Create_MissingAmount(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	input := map[string]interface{}{
		"type":     "expense",
		"category": "Food",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "amount is required")
}

func TestTransactionHandler_Create_MissingCategory(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	input := map[string]interface{}{
		"type":   "expense",
		"amount": 100,
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "category is required")
}

func TestTransactionHandler_Create_ServiceError(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	userID := uuid.New()
	mockService.On("Create", mock.Anything, userID, mock.Anything).Return(nil, errors.New("service error"))

	body := []byte(`{"type":"expense","amount":"100","category":"Food"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_Get_Success(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	txID := uuid.New()
	expectedTx := &model.Transaction{
		ID:       txID,
		UserID:   uuid.New(),
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(100),
		Category: "Food",
	}

	mockService.On("Get", mock.Anything, txID).Return(expectedTx, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/"+txID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Get(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTestableTransactionHandler_Get_InvalidID(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/invalid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTransactionHandler_Get_NotFound(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	txID := uuid.New()
	mockService.On("Get", mock.Anything, txID).Return(nil, repository.ErrTransactionNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/"+txID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Get(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_List_Success(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	userID := uuid.New()
	expectedTxs := []model.Transaction{
		{ID: uuid.New(), UserID: userID, Type: model.TransactionTypeExpense},
		{ID: uuid.New(), UserID: userID, Type: model.TransactionTypeIncome},
	}

	mockService.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return(expectedTxs, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_List_Error(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	userID := uuid.New()
	mockService.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

	rr := httptest.NewRecorder()
	handler.List(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_Update_Success(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	userID := uuid.New()
	txID := uuid.New()
	expectedTx := &model.Transaction{
		ID:       txID,
		UserID:   userID,
		Category: "Shopping",
	}

	mockService.On("Update", mock.Anything, txID, userID, mock.Anything).Return(expectedTx, nil)

	body := []byte(`{"category":"Shopping"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/transactions/"+txID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set both UserID context and route context
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_Update_InvalidID(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	body, _ := json.Marshal(map[string]interface{}{"category": "Test"})
	req := httptest.NewRequest(http.MethodPut, "/api/transactions/invalid", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTransactionHandler_Update_InvalidBody(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	txID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/api/transactions/"+txID.String(), bytes.NewReader([]byte("invalid")))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTransactionHandler_Delete_Success(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	userID := uuid.New()
	txID := uuid.New()

	mockService.On("Delete", mock.Anything, txID, userID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/transactions/"+txID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_Delete_InvalidID(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/api/transactions/invalid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTransactionHandler_Delete_Error(t *testing.T) {
	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	userID := uuid.New()
	txID := uuid.New()

	mockService.On("Delete", mock.Anything, txID, userID).Return(errors.New("delete error"))

	req := httptest.NewRequest(http.MethodDelete, "/api/transactions/"+txID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}

// Test List with query parameters
func TestTransactionHandler_List_WithQueryParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		queryParams string
		setupMock   func(*MockTransactionService, uuid.UUID)
		wantStatus  int
	}{
		{
			name:        "with page and pageSize",
			queryParams: "?page=2&pageSize=50",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with type filter",
			queryParams: "?type=expense",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with category filter",
			queryParams: "?category=Food",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with date range",
			queryParams: "?startDate=2024-01-01&endDate=2024-12-31",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with invalid page (uses default)",
			queryParams: "?page=invalid",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with invalid pageSize (uses default)",
			queryParams: "?pageSize=invalid",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with invalid startDate (ignored)",
			queryParams: "?startDate=invalid-date",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with invalid endDate (ignored)",
			queryParams: "?endDate=invalid-date",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "with all filters",
			queryParams: "?page=1&pageSize=25&type=income&category=Salary&startDate=2024-01-01&endDate=2024-12-31",
			setupMock: func(m *MockTransactionService, userID uuid.UUID) {
				m.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{
					{ID: uuid.New(), Category: "Salary"},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockTransactionService)
			handler := NewTransactionHandler(mockService)
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			req := httptest.NewRequest(http.MethodGet, "/api/transactions"+tt.queryParams, nil)
			req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
