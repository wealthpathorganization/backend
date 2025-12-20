package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/handler"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/service"
)

// ============ Mock Services ============

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

func (m *MockUserService) UpdateSettings(ctx context.Context, userID uuid.UUID, input service.UpdateSettingsInput) (*model.User, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

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

// ============ Test Server Setup ============

func setupTestRouter(
	authHandler *handler.AuthHandler,
	transactionHandler *handler.TransactionHandler,
	budgetHandler *handler.BudgetHandler,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Public routes
	if authHandler != nil {
		r.Post("/api/auth/register", authHandler.Register)
		r.Post("/api/auth/login", authHandler.Login)
	}

	// Protected routes (simplified for testing - no real auth)
	r.Group(func(r chi.Router) {
		// For testing, we'll inject userID directly
		if transactionHandler != nil {
			r.Get("/api/transactions", transactionHandler.List)
			r.Post("/api/transactions", transactionHandler.Create)
			r.Get("/api/transactions/{id}", transactionHandler.Get)
			r.Put("/api/transactions/{id}", transactionHandler.Update)
			r.Delete("/api/transactions/{id}", transactionHandler.Delete)
		}

		if budgetHandler != nil {
			r.Get("/api/budgets", budgetHandler.List)
			r.Post("/api/budgets", budgetHandler.Create)
			r.Get("/api/budgets/{id}", budgetHandler.Get)
			r.Put("/api/budgets/{id}", budgetHandler.Update)
			r.Delete("/api/budgets/{id}", budgetHandler.Delete)
		}
	})

	return r
}

// Helper to add userID to request context
func withUserID(req *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), handler.UserIDKey, userID)
	return req.WithContext(ctx)
}

// ============ API Integration Tests ============

func TestAPI_HealthCheck(t *testing.T) {
	t.Parallel()

	router := setupTestRouter(nil, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/health")

	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "ok", body["status"])
}

func TestAPI_Auth_Register(t *testing.T) {
	t.Parallel()

	mockUserService := new(MockUserService)
	authHandler := handler.NewAuthHandler(mockUserService)

	userID := uuid.New()
	mockUserService.On("Register", mock.Anything, mock.AnythingOfType("service.RegisterInput")).Return(&service.AuthResponse{
		User: &model.User{
			ID:    userID,
			Email: "test@example.com",
			Name:  "Test User",
		},
		Token: "jwt-token-here",
	}, nil)

	router := setupTestRouter(authHandler, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()

	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewReader(body))

	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var respBody map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&respBody)
	assert.NotEmpty(t, respBody["token"])
	mockUserService.AssertExpectations(t)
}

func TestAPI_Auth_Register_MissingFields(t *testing.T) {
	t.Parallel()

	mockUserService := new(MockUserService)
	authHandler := handler.NewAuthHandler(mockUserService)

	router := setupTestRouter(authHandler, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()

	// Missing email
	reqBody := map[string]string{
		"password": "password123",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewReader(body))

	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAPI_Auth_Login(t *testing.T) {
	t.Parallel()

	mockUserService := new(MockUserService)
	authHandler := handler.NewAuthHandler(mockUserService)

	userID := uuid.New()
	mockUserService.On("Login", mock.Anything, mock.AnythingOfType("service.LoginInput")).Return(&service.AuthResponse{
		User: &model.User{
			ID:    userID,
			Email: "test@example.com",
		},
		Token: "jwt-token-here",
	}, nil)

	router := setupTestRouter(authHandler, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()

	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewReader(body))

	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&respBody)
	assert.NotEmpty(t, respBody["token"])
	mockUserService.AssertExpectations(t)
}

func TestAPI_Auth_Login_InvalidCredentials(t *testing.T) {
	t.Parallel()

	mockUserService := new(MockUserService)
	authHandler := handler.NewAuthHandler(mockUserService)

	mockUserService.On("Login", mock.Anything, mock.AnythingOfType("service.LoginInput")).Return(nil, service.ErrInvalidCredentials)

	router := setupTestRouter(authHandler, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()

	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewReader(body))

	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUserService.AssertExpectations(t)
}

func TestAPI_Transactions_Create(t *testing.T) {
	t.Parallel()

	mockTxService := new(MockTransactionService)
	txHandler := handler.NewTransactionHandler(mockTxService)

	userID := uuid.New()
	txID := uuid.New()

	mockTxService.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateTransactionInput")).Return(&model.Transaction{
		ID:       txID,
		UserID:   userID,
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(50),
		Category: "Food",
	}, nil)

	router := setupTestRouter(nil, txHandler, nil)

	reqBody := map[string]interface{}{
		"type":        "expense",
		"amount":      "50",
		"category":    "Food",
		"description": "Lunch",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, userID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var respBody map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&respBody)
	assert.Equal(t, txID.String(), respBody["id"])
	mockTxService.AssertExpectations(t)
}

func TestAPI_Transactions_List(t *testing.T) {
	t.Parallel()

	mockTxService := new(MockTransactionService)
	txHandler := handler.NewTransactionHandler(mockTxService)

	userID := uuid.New()

	mockTxService.On("List", mock.Anything, userID, mock.AnythingOfType("service.ListTransactionsInput")).Return([]model.Transaction{
		{ID: uuid.New(), UserID: userID, Type: model.TransactionTypeExpense, Amount: decimal.NewFromFloat(50), Category: "Food"},
		{ID: uuid.New(), UserID: userID, Type: model.TransactionTypeIncome, Amount: decimal.NewFromFloat(5000), Category: "Salary"},
	}, nil)

	router := setupTestRouter(nil, txHandler, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req = withUserID(req, userID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody []map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&respBody)
	assert.Len(t, respBody, 2)
	mockTxService.AssertExpectations(t)
}

func TestAPI_Transactions_Get(t *testing.T) {
	t.Parallel()

	mockTxService := new(MockTransactionService)
	txHandler := handler.NewTransactionHandler(mockTxService)

	txID := uuid.New()
	userID := uuid.New()

	mockTxService.On("Get", mock.Anything, txID).Return(&model.Transaction{
		ID:       txID,
		UserID:   userID,
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(50),
		Category: "Food",
	}, nil)

	router := setupTestRouter(nil, txHandler, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/"+txID.String(), nil)
	req = withUserID(req, userID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockTxService.AssertExpectations(t)
}

func TestAPI_Transactions_Delete(t *testing.T) {
	t.Parallel()

	mockTxService := new(MockTransactionService)
	txHandler := handler.NewTransactionHandler(mockTxService)

	txID := uuid.New()
	userID := uuid.New()

	mockTxService.On("Delete", mock.Anything, txID, userID).Return(nil)

	router := setupTestRouter(nil, txHandler, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/transactions/"+txID.String(), nil)
	req = withUserID(req, userID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockTxService.AssertExpectations(t)
}

func TestAPI_Budgets_Create(t *testing.T) {
	t.Parallel()

	mockBudgetService := new(MockBudgetService)
	budgetHandler := handler.NewBudgetHandler(mockBudgetService)

	userID := uuid.New()
	budgetID := uuid.New()

	mockBudgetService.On("Create", mock.Anything, userID, mock.AnythingOfType("service.CreateBudgetInput")).Return(&model.Budget{
		ID:       budgetID,
		UserID:   userID,
		Category: "Food",
		Amount:   decimal.NewFromFloat(500),
		Period:   "monthly",
	}, nil)

	router := setupTestRouter(nil, nil, budgetHandler)

	reqBody := map[string]interface{}{
		"category": "Food",
		"amount":   500,
		"period":   "monthly",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, userID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockBudgetService.AssertExpectations(t)
}

func TestAPI_Budgets_List(t *testing.T) {
	t.Parallel()

	mockBudgetService := new(MockBudgetService)
	budgetHandler := handler.NewBudgetHandler(mockBudgetService)

	userID := uuid.New()

	mockBudgetService.On("ListWithSpent", mock.Anything, userID).Return([]model.BudgetWithSpent{
		{Budget: model.Budget{ID: uuid.New(), Category: "Food", Amount: decimal.NewFromFloat(500)}},
	}, nil)

	router := setupTestRouter(nil, nil, budgetHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/budgets", nil)
	req = withUserID(req, userID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockBudgetService.AssertExpectations(t)
}

func TestAPI_Budgets_Delete(t *testing.T) {
	t.Parallel()

	mockBudgetService := new(MockBudgetService)
	budgetHandler := handler.NewBudgetHandler(mockBudgetService)

	budgetID := uuid.New()
	userID := uuid.New()

	mockBudgetService.On("Delete", mock.Anything, budgetID, userID).Return(nil)

	router := setupTestRouter(nil, nil, budgetHandler)

	req := httptest.NewRequest(http.MethodDelete, "/api/budgets/"+budgetID.String(), nil)
	req = withUserID(req, userID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockBudgetService.AssertExpectations(t)
}

// ============ Error Cases ============

func TestAPI_InvalidJSON(t *testing.T) {
	t.Parallel()

	mockUserService := new(MockUserService)
	authHandler := handler.NewAuthHandler(mockUserService)

	router := setupTestRouter(authHandler, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewReader([]byte("invalid json")))

	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAPI_NotFound(t *testing.T) {
	t.Parallel()

	router := setupTestRouter(nil, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/nonexistent")

	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
