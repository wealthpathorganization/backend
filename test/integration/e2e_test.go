//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/wealthpath/backend/internal/handler"
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/internal/service"
)

// Schema for test database
const testSchema = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),
    avatar_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense')),
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    category VARCHAR(100) NOT NULL,
    description TEXT,
    date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    period VARCHAR(20) DEFAULT 'monthly',
    start_date DATE NOT NULL,
    end_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS savings_goals (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    target_amount DECIMAL(15, 2) NOT NULL,
    current_amount DECIMAL(15, 2) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    target_date DATE,
    color VARCHAR(7) DEFAULT '#3B82F6',
    icon VARCHAR(50) DEFAULT 'piggy-bank',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS debts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    original_amount DECIMAL(15, 2) NOT NULL,
    current_balance DECIMAL(15, 2) NOT NULL,
    interest_rate DECIMAL(5, 2) NOT NULL,
    minimum_payment DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    due_day INTEGER,
    start_date DATE NOT NULL,
    expected_payoff DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS recurring_transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense')),
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    category VARCHAR(100) NOT NULL,
    description TEXT,
    frequency VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    next_occurrence DATE,
    is_active BOOLEAN DEFAULT true,
    last_generated DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

// TestEnv holds the test environment
type TestEnv struct {
	DB        *sqlx.DB
	Container testcontainers.Container
	Server    *httptest.Server
	Router    *chi.Mux
	Token     string // JWT token for authenticated requests
}

// SetupTestEnv creates a test environment with a real PostgreSQL database
func SetupTestEnv(t *testing.T) *TestEnv {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	db, err := sqlx.Connect("postgres", connStr)
	require.NoError(t, err)

	// Run migrations
	_, err = db.Exec(testSchema)
	require.NoError(t, err)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)
	savingsRepo := repository.NewSavingsGoalRepository(db)
	debtRepo := repository.NewDebtRepository(db)
	recurringRepo := repository.NewRecurringRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	budgetService := service.NewBudgetService(budgetRepo)
	savingsService := service.NewSavingsGoalService(savingsRepo)
	debtService := service.NewDebtService(debtRepo)
	recurringService := service.NewRecurringService(recurringRepo, transactionRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	budgetHandler := handler.NewBudgetHandler(budgetService)
	savingsHandler := handler.NewSavingsGoalHandler(savingsService)
	debtHandler := handler.NewDebtHandler(debtService)
	recurringHandler := handler.NewRecurringHandler(recurringService)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Public routes
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware)

		r.Get("/api/auth/me", authHandler.Me)
		r.Put("/api/auth/settings", authHandler.UpdateSettings)

		r.Get("/api/transactions", transactionHandler.List)
		r.Post("/api/transactions", transactionHandler.Create)
		r.Get("/api/transactions/{id}", transactionHandler.Get)
		r.Put("/api/transactions/{id}", transactionHandler.Update)
		r.Delete("/api/transactions/{id}", transactionHandler.Delete)

		r.Get("/api/budgets", budgetHandler.List)
		r.Post("/api/budgets", budgetHandler.Create)
		r.Get("/api/budgets/{id}", budgetHandler.Get)
		r.Put("/api/budgets/{id}", budgetHandler.Update)
		r.Delete("/api/budgets/{id}", budgetHandler.Delete)

		r.Get("/api/savings-goals", savingsHandler.List)
		r.Post("/api/savings-goals", savingsHandler.Create)
		r.Get("/api/savings-goals/{id}", savingsHandler.Get)
		r.Put("/api/savings-goals/{id}", savingsHandler.Update)
		r.Delete("/api/savings-goals/{id}", savingsHandler.Delete)
		r.Post("/api/savings-goals/{id}/contribute", savingsHandler.Contribute)

		r.Get("/api/debts", debtHandler.List)
		r.Post("/api/debts", debtHandler.Create)
		r.Get("/api/debts/{id}", debtHandler.Get)
		r.Put("/api/debts/{id}", debtHandler.Update)
		r.Delete("/api/debts/{id}", debtHandler.Delete)
		r.Post("/api/debts/{id}/payment", debtHandler.MakePayment)

		r.Get("/api/recurring", recurringHandler.List)
		r.Post("/api/recurring", recurringHandler.Create)
		r.Get("/api/recurring/{id}", recurringHandler.Get)
		r.Put("/api/recurring/{id}", recurringHandler.Update)
		r.Delete("/api/recurring/{id}", recurringHandler.Delete)
	})

	server := httptest.NewServer(r)

	return &TestEnv{
		DB:        db,
		Container: pgContainer,
		Server:    server,
		Router:    r,
	}
}

// Cleanup tears down the test environment
func (e *TestEnv) Cleanup(t *testing.T) {
	e.Server.Close()
	e.DB.Close()
	if err := e.Container.Terminate(context.Background()); err != nil {
		t.Logf("Failed to terminate container: %v", err)
	}
}

// Helper: Make HTTP request
func (e *TestEnv) Request(method, path string, body interface{}) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, e.Server.URL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.Token != "" {
		req.Header.Set("Authorization", "Bearer "+e.Token)
	}
	return http.DefaultClient.Do(req)
}

// Helper: Register and get token
func (e *TestEnv) RegisterUser(t *testing.T, email, password, name string) string {
	resp, err := e.Request("POST", "/api/auth/register", map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["token"].(string)
}

// ============ E2E Tests ============

func TestE2E_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	resp, err := env.Request("GET", "/api/health", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestE2E_AuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// 1. Register
	resp, err := env.Request("POST", "/api/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var registerResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&registerResult)
	assert.NotEmpty(t, registerResult["token"])
	assert.NotNil(t, registerResult["user"])

	// 2. Login
	resp, err = env.Request("POST", "/api/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResult)
	env.Token = loginResult["token"].(string)
	assert.NotEmpty(t, env.Token)

	// 3. Get current user
	resp, err = env.Request("GET", "/api/auth/me", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var user map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&user)
	assert.Equal(t, "test@example.com", user["email"])
}

func TestE2E_TransactionCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// Register and login
	env.Token = env.RegisterUser(t, "txtest@example.com", "password123", "TX User")

	// 1. Create transaction
	resp, err := env.Request("POST", "/api/transactions", map[string]interface{}{
		"type":        "expense",
		"amount":      50,
		"category":    "Food",
		"description": "Lunch",
		"date":        time.Now().Format("2006-01-02"),
		"currency":    "USD",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	txID := created["id"].(string)
	assert.NotEmpty(t, txID)

	// 2. Get transaction
	resp, err = env.Request("GET", fmt.Sprintf("/api/transactions/%s", txID), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var fetched map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&fetched)
	assert.Equal(t, "Food", fetched["category"])

	// 3. Update transaction (need all fields)
	resp, err = env.Request("PUT", fmt.Sprintf("/api/transactions/%s", txID), map[string]interface{}{
		"type":        "expense",
		"amount":      55,
		"currency":    "USD",
		"category":    "Dining",
		"description": "Updated lunch",
		"date":        time.Now().Format("2006-01-02"),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. Delete transaction
	resp, err = env.Request("DELETE", fmt.Sprintf("/api/transactions/%s", txID), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify deleted - should return 404
	resp, err = env.Request("GET", fmt.Sprintf("/api/transactions/%s", txID), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_BudgetCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	env.Token = env.RegisterUser(t, "budget@example.com", "password123", "Budget User")

	// Create budget
	resp, err := env.Request("POST", "/api/budgets", map[string]interface{}{
		"category":   "Food",
		"amount":     500,
		"period":     "monthly",
		"start_date": time.Now().Format("2006-01-02"),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var budget map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&budget)
	budgetID := budget["id"].(string)

	// List budgets
	resp, err = env.Request("GET", "/api/budgets", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Delete budget
	resp, err = env.Request("DELETE", fmt.Sprintf("/api/budgets/%s", budgetID), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestE2E_SavingsGoalCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	env.Token = env.RegisterUser(t, "savings@example.com", "password123", "Savings User")

	// Create savings goal
	resp, err := env.Request("POST", "/api/savings-goals", map[string]interface{}{
		"name":         "Emergency Fund",
		"targetAmount": 10000,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var goal map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&goal)
	goalID := goal["id"].(string)

	// Contribute to goal
	resp, err = env.Request("POST", fmt.Sprintf("/api/savings-goals/%s/contribute", goalID), map[string]interface{}{
		"amount": 500,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Get updated goal
	resp, err = env.Request("GET", fmt.Sprintf("/api/savings-goals/%s", goalID), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	json.NewDecoder(resp.Body).Decode(&goal)
	assert.Equal(t, "500", goal["currentAmount"])

	// Delete goal
	resp, err = env.Request("DELETE", fmt.Sprintf("/api/savings-goals/%s", goalID), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestE2E_DebtCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	env.Token = env.RegisterUser(t, "debt@example.com", "password123", "Debt User")

	// Create debt
	resp, err := env.Request("POST", "/api/debts", map[string]interface{}{
		"name":           "Car Loan",
		"type":           "auto_loan",
		"originalAmount": 20000,
		"currentBalance": 18000,
		"interestRate":   5.5,
		"minimumPayment": 400,
		"currency":       "USD",
		"dueDay":         15,
		"startDate":      time.Now().AddDate(-1, 0, 0).Format(time.RFC3339),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var debt map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&debt)
	debtID := debt["id"].(string)

	// Make payment
	resp, err = env.Request("POST", fmt.Sprintf("/api/debts/%s/payment", debtID), map[string]interface{}{
		"amount": 500,
		"date":   time.Now().Format(time.RFC3339),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// List debts
	resp, err = env.Request("GET", "/api/debts", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Delete debt
	resp, err = env.Request("DELETE", fmt.Sprintf("/api/debts/%s", debtID), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestE2E_UnauthorizedAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// No token set - should fail
	resp, err := env.Request("GET", "/api/transactions", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp, err = env.Request("GET", "/api/auth/me", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_InvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	env.Token = "invalid-jwt-token"

	resp, err := env.Request("GET", "/api/transactions", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_DuplicateEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// Register first user
	env.RegisterUser(t, "duplicate@example.com", "password123", "First User")

	// Try to register with same email
	resp, err := env.Request("POST", "/api/auth/register", map[string]string{
		"email":    "duplicate@example.com",
		"password": "password456",
		"name":     "Second User",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestE2E_InvalidLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	// Register user
	env.RegisterUser(t, "login@example.com", "password123", "Login User")

	// Try wrong password
	resp, err := env.Request("POST", "/api/auth/login", map[string]string{
		"email":    "login@example.com",
		"password": "wrongpassword",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Try non-existent email
	resp, err = env.Request("POST", "/api/auth/login", map[string]string{
		"email":    "nonexistent@example.com",
		"password": "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
