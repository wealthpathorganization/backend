# Backend (Go) Development Rules

## Architecture: Clean Architecture

The backend follows a strict layered architecture: `handler -> service -> repository -> model`.

### Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Handler | `internal/handler/` | HTTP request handling, validation, response formatting |
| Service | `internal/service/` | Business logic, orchestration, domain rules |
| Repository | `internal/repository/` | Data persistence, SQL queries, database operations |
| Model | `internal/model/` | Domain types, constants, struct definitions |

### Dependency Rule

Dependencies flow inward only. Each layer depends only on the layer below it:
- Handlers depend on Services (via interfaces)
- Services depend on Repositories (via interfaces)
- Repositories depend on Models
- Models have no internal dependencies

## Interface Patterns

### Interface Segregation (Consumer-Defined Interfaces)

Define interfaces in the **consuming package**, not the implementing package. This keeps interfaces small and focused on what the consumer actually needs.

```go
// handler/transaction_handler.go - Interface defined where it's USED
type TransactionHandlerServiceInterface interface {
    Create(ctx context.Context, userID uuid.UUID, input service.CreateTransactionInput) (*model.Transaction, error)
    Get(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
    List(ctx context.Context, userID uuid.UUID, input service.ListTransactionsInput) ([]model.Transaction, error)
    Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateTransactionInput) (*model.Transaction, error)
    Delete(ctx context.Context, id, userID uuid.UUID) error
}

// service/budget_service.go - Service defines interface for what IT needs from repository
type BudgetRepositoryInterface interface {
    Create(ctx context.Context, budget *model.Budget) error
    GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error)
    List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
    GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
    Update(ctx context.Context, budget *model.Budget) error
    Delete(ctx context.Context, id, userID uuid.UUID) error
}
```

### Naming Convention

- Interface names: `{Entity}{Layer}Interface` (e.g., `BudgetServiceInterface`, `TransactionRepositoryInterface`)
- Implementation structs: `{Entity}{Layer}` (e.g., `BudgetService`, `TransactionRepository`)
- Constructors: `New{Entity}{Layer}` (e.g., `NewBudgetService`)

## Handler Patterns

### Standard Handler Structure

Every handler follows this pattern:

```go
// Package comment describing the handler package
package handler

// 1. Define interface for the service this handler needs
type BudgetServiceInterface interface {
    Create(ctx context.Context, userID uuid.UUID, input service.CreateBudgetInput) (*model.Budget, error)
    // ... other methods
}

// 2. Handler struct with service dependency
type BudgetHandler struct {
    service BudgetServiceInterface
}

// 3. Constructor for dependency injection
func NewBudgetHandler(service BudgetServiceInterface) *BudgetHandler {
    return &BudgetHandler{service: service}
}

// 4. Handler methods with Swagger annotations
```

### Handler Method Flow

Every handler method follows this flow:
1. Extract user ID from context (for authenticated routes)
2. Parse path parameters (using `chi.URLParam`)
3. Parse query parameters or decode request body
4. Validate input
5. Call service method
6. Handle errors with appropriate response
7. Return JSON response

```go
// Create godoc
// @Summary Create a budget
// @Description Create a new budget for tracking spending
// @Tags budgets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateBudgetInput true "Budget data"
// @Success 201 {object} model.Budget
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /budgets [post]
func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. Get user from context
    userID := GetUserID(r.Context())

    // 2. Decode request body
    var input service.CreateBudgetInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        respondAppError(w, apperror.BadRequest("invalid request body: "+err.Error()))
        return
    }

    // 3. Validate required fields
    if input.Category == "" {
        respondAppError(w, apperror.ValidationError("category", "category is required"))
        return
    }
    if input.Amount.IsZero() {
        respondAppError(w, apperror.ValidationError("amount", "amount must be greater than 0"))
        return
    }

    // 4. Call service
    budget, err := h.service.Create(r.Context(), userID, input)
    if err != nil {
        respondAppError(w, apperror.Internal(err))
        return
    }

    // 5. Return response
    respondJSON(w, http.StatusCreated, budget)
}
```

### Swagger Annotations

Every endpoint MUST have complete Swagger documentation:

```go
// @Summary Brief description (imperative mood)
// @Description Longer description with details
// @Tags resource-name (lowercase, plural)
// @Accept json (for POST/PUT/PATCH)
// @Produce json
// @Security BearerAuth (for authenticated routes)
// @Param id path string true "Resource ID"
// @Param input body service.InputType true "Description"
// @Param page query int false "Page number" default(0)
// @Success 200 {object} model.Resource
// @Success 201 {object} model.Resource (for create)
// @Success 204 "No Content" (for delete)
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /path [method]
```

### Response Functions

Use the standard response helpers:

```go
// Success responses
respondJSON(w, http.StatusOK, data)              // GET, PUT
respondJSON(w, http.StatusCreated, data)         // POST
w.WriteHeader(http.StatusNoContent)              // DELETE

// Error responses
respondAppError(w, apperror.BadRequest("message"))
respondAppError(w, apperror.ValidationError("field", "message"))
respondAppError(w, apperror.NotFound("resource"))
respondAppError(w, apperror.Internal(err))
respondAppError(w, apperror.Unauthorized("message"))
```

### Path Parameter Parsing

Use Chi router for path parameters:

```go
// Parse UUID path parameter
id, err := uuid.Parse(chi.URLParam(r, "id"))
if err != nil {
    respondAppError(w, apperror.BadRequest("invalid ID format"))
    return
}
```

## Service Layer Patterns

### Service Structure

```go
// Service struct with repository dependencies
type BudgetService struct {
    repo            BudgetRepositoryInterface
    transactionRepo TransactionRepoForBudget  // Optional: for cross-entity calculations
}

// Constructor
func NewBudgetService(repo BudgetRepositoryInterface) *BudgetService {
    return &BudgetService{repo: repo}
}

// Optional setter for additional dependencies
func (s *BudgetService) SetTransactionRepo(repo TransactionRepoForBudget) {
    s.transactionRepo = repo
}
```

### Input/Output Types

Define input structs in the service package with JSON tags for Swagger documentation:

```go
type CreateBudgetInput struct {
    Category          string           `json:"category"`
    Amount            decimal.Decimal  `json:"amount"`
    Currency          string           `json:"currency"`
    Period            string           `json:"period"`
    StartDate         time.Time        `json:"startDate"`
    EndDate           *time.Time       `json:"endDate"`
    EnableRollover    bool             `json:"enableRollover"`
    MaxRolloverAmount *decimal.Decimal `json:"maxRolloverAmount,omitempty"`
}
```

### Error Handling in Services

Wrap errors with context using `fmt.Errorf`:

```go
func (s *BudgetService) Create(ctx context.Context, userID uuid.UUID, input CreateBudgetInput) (*model.Budget, error) {
    budget := &model.Budget{
        UserID:   userID,
        Category: input.Category,
        // ...
    }

    if err := s.repo.Create(ctx, budget); err != nil {
        return nil, fmt.Errorf("creating budget: %w", err)
    }

    return budget, nil
}
```

### Authorization Checks

Services should verify resource ownership:

```go
func (s *BudgetService) Update(ctx context.Context, id, userID uuid.UUID, input UpdateBudgetInput) (*model.Budget, error) {
    budget, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("fetching budget %s for update: %w", id, err)
    }

    // Authorization check
    if budget.UserID != userID {
        return nil, repository.ErrBudgetNotFound  // Return not found to avoid leaking info
    }

    // ... update logic
}
```

## Repository Patterns

### Repository Structure

```go
type TransactionRepository struct {
    db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) *TransactionRepository {
    return &TransactionRepository{db: db}
}
```

### Sentinel Errors

Define sentinel errors at package level for common cases:

```go
var (
    ErrTransactionNotFound = errors.New("transaction not found")
    ErrBudgetNotFound      = errors.New("budget not found")
    ErrUserNotFound        = errors.New("user not found")
)
```

### SQL Query Patterns

**Always use parameterized queries:**

```go
query := `
    INSERT INTO transactions (id, user_id, type, amount, currency, category, description, date, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
    RETURNING created_at, updated_at`

tx.ID = uuid.New()
return r.db.QueryRowxContext(ctx, query,
    tx.ID, tx.UserID, tx.Type, tx.Amount, tx.Currency, tx.Category, tx.Description, tx.Date,
).Scan(&tx.CreatedAt, &tx.UpdatedAt)
```

**Handle NULL in WHERE clauses with type casting:**

```go
query := `
    SELECT * FROM transactions
    WHERE user_id = $1
    AND ($2::text IS NULL OR type = $2)
    AND ($3::text IS NULL OR category = $3)
    AND ($4::text[] IS NULL OR category = ANY($4))
    AND ($5::timestamp IS NULL OR date >= $5)
    AND ($6::timestamp IS NULL OR date <= $6)
    ORDER BY date DESC
    LIMIT $7 OFFSET $8`
```

**Use RETURNING for generated values:**

```go
query := `
    UPDATE transactions
    SET type = $2, amount = $3, updated_at = NOW()
    WHERE id = $1 AND user_id = $4
    RETURNING updated_at`
```

**Check sql.ErrNoRows for not found:**

```go
func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
    var tx model.Transaction
    query := `SELECT * FROM transactions WHERE id = $1`
    err := r.db.GetContext(ctx, &tx, query, id)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrTransactionNotFound
    }
    return &tx, err
}
```

**Check RowsAffected for delete operations:**

```go
func (r *TransactionRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
    query := `DELETE FROM transactions WHERE id = $1 AND user_id = $2`
    result, err := r.db.ExecContext(ctx, query, id, userID)
    if err != nil {
        return err
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return ErrTransactionNotFound
    }
    return nil
}
```

### Filter Structs

Define filter structs for complex queries:

```go
type TransactionFilters struct {
    Type       *string
    Category   *string
    Categories []string
    Search     *string
    MinAmount  *decimal.Decimal
    MaxAmount  *decimal.Decimal
    StartDate  *time.Time
    EndDate    *time.Time
    Limit      int
    Offset     int
}
```

## Model Patterns

### Struct Definition

Use struct tags for database mapping and JSON serialization:

```go
type Transaction struct {
    ID          uuid.UUID       `db:"id" json:"id"`
    UserID      uuid.UUID       `db:"user_id" json:"userId"`
    Type        TransactionType `db:"type" json:"type"`
    Amount      decimal.Decimal `db:"amount" json:"amount"`
    Currency    string          `db:"currency" json:"currency"`
    Category    string          `db:"category" json:"category"`
    Description string          `db:"description" json:"description"`
    Date        time.Time       `db:"date" json:"date"`
    CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
    UpdatedAt   time.Time       `db:"updated_at" json:"updatedAt"`
}
```

### Type Safety for Enums

Use typed string constants for constrained values:

```go
type TransactionType string

const (
    TransactionTypeIncome  TransactionType = "income"
    TransactionTypeExpense TransactionType = "expense"
)

type DebtType string

const (
    DebtTypeMortgage     DebtType = "mortgage"
    DebtTypeAutoLoan     DebtType = "auto_loan"
    DebtTypeStudentLoan  DebtType = "student_loan"
    DebtTypeCreditCard   DebtType = "credit_card"
    DebtTypePersonalLoan DebtType = "personal_loan"
    DebtTypeOther        DebtType = "other"
)
```

### Required Types

- **IDs**: Always use `uuid.UUID` from `github.com/google/uuid`
- **Money**: Always use `decimal.Decimal` from `github.com/shopspring/decimal`
- **Timestamps**: Use `time.Time`, store as `TIMESTAMP WITH TIME ZONE` in PostgreSQL
- **Optional fields**: Use pointers (`*string`, `*time.Time`, `*decimal.Decimal`)
- **JSON omit empty**: Use `json:"field,omitempty"` for optional fields

## Error Handling

### AppError Package

Use the `apperror` package for HTTP-aware errors:

```go
// Sentinel errors
var (
    ErrNotFound           = errors.New("resource not found")
    ErrUnauthorized       = errors.New("unauthorized")
    ErrBadRequest         = errors.New("bad request")
    ErrValidation         = errors.New("validation error")
)

// AppError wraps errors with HTTP status
type AppError struct {
    Err        error  // Original error (for logging)
    Message    string // User-friendly message
    StatusCode int    // HTTP status code
    Field      string // Optional field name for validation errors
}

// Constructor functions
apperror.NotFound("transaction")              // 404
apperror.BadRequest("invalid request body")   // 400
apperror.ValidationError("amount", "amount must be greater than 0")  // 400 with field
apperror.Unauthorized("invalid token")        // 401
apperror.Forbidden("access denied")           // 403
apperror.Conflict("email already exists")     // 409
apperror.Internal(err)                        // 500
```

### Error Handling Pattern

```go
// In handlers: convert repository errors to HTTP responses
if errors.Is(err, repository.ErrTransactionNotFound) {
    respondAppError(w, apperror.NotFound("transaction"))
    return
}
respondAppError(w, apperror.Internal(err))
```

## Testing Patterns

### Table-Driven Tests

Use table-driven tests with parallel execution:

```go
func TestBudgetService_Create(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        input     CreateBudgetInput
        setupMock func(*MockBudgetRepo)
        wantErr   bool
        check     func(*testing.T, *model.Budget)
    }{
        {
            name: "success with all fields",
            input: CreateBudgetInput{
                Category: "Food",
                Amount:   decimal.NewFromFloat(500),
            },
            setupMock: func(m *MockBudgetRepo) {
                m.On("Create", mock.Anything, mock.AnythingOfType("*model.Budget")).Return(nil)
            },
            wantErr: false,
            check: func(t *testing.T, b *model.Budget) {
                assert.Equal(t, "Food", b.Category)
            },
        },
        {
            name: "repository error",
            input: CreateBudgetInput{Category: "Food"},
            setupMock: func(m *MockBudgetRepo) {
                m.On("Create", mock.Anything, mock.AnythingOfType("*model.Budget")).Return(errors.New("db error"))
            },
            wantErr: true,
            check:   nil,
        },
    }

    for _, tt := range tests {
        tt := tt  // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            mockRepo := new(MockBudgetRepo)
            service := NewBudgetService(mockRepo)
            tt.setupMock(mockRepo)

            result, err := service.Create(context.Background(), uuid.New(), tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                if tt.check != nil {
                    tt.check(t, result)
                }
            }
            mockRepo.AssertExpectations(t)
        })
    }
}
```

### Mock Patterns

Define mocks in test files or `internal/mocks/`:

```go
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
```

### Handler Tests

Use `httptest` for handler testing:

```go
func TestTransactionHandler_Create_Success(t *testing.T) {
    mockService := new(MockTransactionService)
    handler := NewTransactionHandler(mockService)

    userID := uuid.New()
    expectedTx := &model.Transaction{ID: uuid.New(), UserID: userID}

    mockService.On("Create", mock.Anything, userID, mock.Anything).Return(expectedTx, nil)

    body := []byte(`{"type":"expense","amount":"100","category":"Food"}`)
    req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))

    rr := httptest.NewRecorder()
    handler.Create(rr, req)

    assert.Equal(t, http.StatusCreated, rr.Code)
    mockService.AssertExpectations(t)
}
```

### Chi Route Context in Tests

For handlers that use Chi URL parameters:

```go
req := httptest.NewRequest(http.MethodGet, "/api/transactions/"+txID.String(), nil)
rctx := chi.NewRouteContext()
rctx.URLParams.Add("id", txID.String())
req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
```

## Middleware Patterns

### Auth Middleware

```go
type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "missing authorization header", http.StatusUnauthorized)
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            http.Error(w, "invalid authorization header", http.StatusUnauthorized)
            return
        }

        userID, err := service.ValidateToken(parts[1])
        if err != nil {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), UserIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetUserID(ctx context.Context) uuid.UUID {
    userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
    if !ok {
        return uuid.Nil
    }
    return userID
}
```

## Logging Patterns

Use structured logging with `log/slog`:

```go
import "log/slog"

// In production: JSON format
// In development: Text format

logger.Info("Request processed",
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
    slog.Duration("duration", time.Since(start)),
)

logger.Error("Failed to create budget",
    slog.String("error", err.Error()),
    slog.String("user_id", userID.String()),
)
```

## Database Migrations

### Flyway Migration Files

Location: `migrations/db/migration/V{number}__{description}.sql`

Naming convention:
- `V1__initial_schema.sql`
- `V2__oauth_support.sql`
- `V3__recurring_transactions.sql`

Example migration:

```sql
-- V1__initial_schema.sql
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

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
```

### Migration Best Practices

- Always use `IF NOT EXISTS` for CREATE TABLE
- Define proper constraints (NOT NULL, CHECK, REFERENCES)
- Create indexes for commonly queried columns
- Use `ON DELETE CASCADE` for child tables
- Use `DECIMAL(15, 2)` for monetary values
- Use `TIMESTAMP WITH TIME ZONE` for timestamps
- Default timestamps with `DEFAULT NOW()`

## Naming Conventions Summary

| Element | Convention | Example |
|---------|------------|---------|
| Packages | lowercase, short | `handler`, `service`, `repository` |
| Interfaces | PascalCase + `Interface` suffix | `BudgetServiceInterface` |
| Structs | PascalCase | `TransactionHandler` |
| Constructors | `New` + struct name | `NewTransactionHandler()` |
| Errors | `Err` prefix | `ErrTransactionNotFound` |
| Context keys | unexported type | `type contextKey string` |
| JSON fields | camelCase | `json:"userId"` |
| DB columns | snake_case | `db:"user_id"` |
| URL params | kebab-case | `/savings-goals/{id}` |
| Receivers | 1-2 letter abbreviation | `(h *Handler)`, `(s *Service)` |
