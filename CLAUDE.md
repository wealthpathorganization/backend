# CLAUDE.md - WealthPath Backend

This file provides guidance to Claude Code when working with the WealthPath Go backend.

## Project Overview

WealthPath Backend is a REST API server for personal finance management built with Go. It provides endpoints for tracking transactions, budgets, savings goals, debts, and recurring transactions.

### Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| Router | Chi v5 |
| Database | PostgreSQL (via pgx/sqlx) |
| Auth | JWT (access tokens) + HttpOnly cookies (refresh tokens) |
| Migrations | Flyway |
| API Docs | Swagger (swaggo) |
| Logging | log/slog |

### Project Structure

```
backend/
├── cmd/
│   ├── api/            # Main API server entry point
│   └── scraper/        # Interest rate scraper service
├── internal/
│   ├── apperror/       # HTTP-aware error types
│   ├── config/         # Configuration loading
│   ├── handler/        # HTTP handlers (controllers)
│   ├── logger/         # Structured logging setup
│   ├── mocks/          # Test mocks
│   ├── model/          # Domain models and types
│   ├── repository/     # Database access layer
│   ├── scheduler/      # Background job scheduler
│   ├── scraper/        # Bank interest rate scraper
│   └── service/        # Business logic layer
├── migrations/
│   └── db/migration/   # Flyway SQL migrations (V*.sql)
├── pkg/
│   ├── currency/       # Currency utilities
│   └── datetime/       # Date/time utilities
├── docs/               # Generated Swagger documentation
└── test/
    └── integration/    # E2E integration tests
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Docker (optional, for local development)

### Local Development

```bash
# Start PostgreSQL with Docker Compose (from project root)
docker compose -f docker-compose.local.yaml up -d postgres

# Run database migrations
make migrate-local

# Start the API server
make run
# Server runs at http://localhost:8080

# Or build and run binary
make build
./bin/api
```

### Environment Variables

```bash
# Required
DATABASE_URL=postgres://wealthpath:localdev@localhost:5432/wealthpath?sslmode=disable
JWT_SECRET=your-secret-key-min-32-chars

# Optional
PORT=8080
ALLOWED_ORIGINS=http://localhost:3000
ENV=development  # or "production" for JSON logging

# OAuth (optional)
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URI=http://localhost:8080/api/auth/google/callback

# AI Features (optional)
OPENAI_API_KEY=...

# Push Notifications (optional)
VAPID_PUBLIC_KEY=...
VAPID_PRIVATE_KEY=...
```

## Build & Test Commands

```bash
# Run the server (development)
make run

# Build binary
make build

# Build for CI/CD (CGO disabled, explicit cache dirs)
make build-ci

# Run all tests
make test

# Run specific package tests
go test -v ./internal/handler/...
go test -v ./internal/service/...

# Run single test by name
go test -v -run TestTransactionHandler_Create ./internal/handler/...

# Run tests with coverage
go test -cover ./...

# Format code
make fmt

# Lint code (requires golangci-lint)
make lint

# Generate Swagger docs (after adding/modifying endpoints)
make swagger

# Run migrations against local Docker PostgreSQL
make migrate-local

# Clean build artifacts
make clean
```

## Architecture

### Clean Architecture Layers

The backend follows clean architecture with strict dependency rules:

```
HTTP Request
    ↓
Handler (internal/handler/)
    ↓ calls interface
Service (internal/service/)
    ↓ calls interface
Repository (internal/repository/)
    ↓ uses
Model (internal/model/)
    ↓
PostgreSQL
```

**Key principle**: Dependencies flow inward. Interfaces are defined in the consuming package (handler defines ServiceInterface, service defines RepositoryInterface).

### Request Flow Example

```
POST /api/transactions
    ↓
AuthMiddleware → validates JWT, extracts userID
    ↓
TransactionHandler.Create
    ↓ json.Decode → validate → call service
TransactionService.Create
    ↓ business logic → call repo
TransactionRepository.Create
    ↓ parameterized SQL
PostgreSQL INSERT
    ↓
RETURNING → populate model
    ↓
respondJSON(201, transaction)
```

### Key Packages

| Package | Purpose |
|---------|---------|
| `handler` | HTTP handlers, request validation, response formatting |
| `service` | Business logic, domain rules, orchestration |
| `repository` | SQL queries, database operations |
| `model` | Domain types, enums, struct definitions |
| `apperror` | HTTP-aware errors with status codes |

## API Endpoints

### Authentication
- `POST /api/auth/register` - Create account
- `POST /api/auth/login` - Login, returns JWT + sets refresh cookie
- `POST /api/auth/refresh` - Refresh access token (cookie-based)
- `POST /api/auth/logout` - Clear refresh token
- `GET /api/auth/me` - Get current user (protected)
- `GET /api/auth/{provider}` - OAuth login (google, facebook)

### Transactions
- `GET /api/transactions` - List with filters (type, category, date range, search)
- `POST /api/transactions` - Create transaction
- `GET /api/transactions/{id}` - Get single transaction
- `PUT /api/transactions/{id}` - Update transaction
- `DELETE /api/transactions/{id}` - Delete transaction

### Budgets
- `GET /api/budgets` - List budgets with spent amounts
- `POST /api/budgets` - Create budget
- `GET /api/budgets/{id}` - Get budget
- `PUT /api/budgets/{id}` - Update budget
- `DELETE /api/budgets/{id}` - Delete budget

### Savings Goals
- `GET /api/savings-goals` - List savings goals
- `POST /api/savings-goals` - Create goal
- `POST /api/savings-goals/{id}/contribute` - Add contribution

### Debts
- `GET /api/debts` - List debts
- `POST /api/debts` - Create debt
- `GET /api/debts/summary` - Get debt summary
- `POST /api/debts/{id}/payment` - Record payment
- `GET /api/debts/{id}/payoff-plan` - Calculate payoff plan

### Recurring Transactions
- `GET /api/recurring` - List recurring transactions
- `POST /api/recurring` - Create recurring transaction
- `GET /api/recurring/upcoming` - Get upcoming bills
- `POST /api/recurring/{id}/pause` - Pause recurring
- `POST /api/recurring/{id}/resume` - Resume recurring

### Dashboard & Reports
- `GET /api/dashboard` - Aggregated financial overview
- `GET /api/reports/monthly` - Monthly income/expense report
- `GET /api/reports/category-trends` - Category spending trends

### Interest Rates (Public)
- `GET /api/interest-rates` - List bank interest rates
- `GET /api/interest-rates/best` - Get best rates by term

## Database

### Local PostgreSQL

```bash
# Connection string for local Docker
postgres://wealthpath:localdev@localhost:5432/wealthpath?sslmode=disable

# Connect via psql in Docker
docker exec -it wealthpathorganization-postgres-1 psql -U wealthpath -d wealthpath
```

### Migrations

Migrations use Flyway naming convention: `V{number}__{description}.sql`

Location: `migrations/db/migration/`

```sql
-- Example: V1__initial_schema.sql
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense')),
    amount DECIMAL(15, 2) NOT NULL,
    ...
);
```

## Testing

### Test Structure

Tests follow Go conventions with table-driven tests:

```go
func TestBudgetService_Create(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        input     CreateBudgetInput
        setupMock func(*MockBudgetRepo)
        wantErr   bool
    }{
        // test cases...
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // test implementation
        })
    }
}
```

### Mocking

Mocks are defined in test files or `internal/mocks/` using testify/mock:

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

### HTTP Handler Tests

Use httptest for handler testing:

```go
req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req = req.WithContext(context.WithValue(req.Context(), handler.UserIDKey, userID))

rr := httptest.NewRecorder()
handler.Create(rr, req)

assert.Equal(t, http.StatusCreated, rr.Code)
```

## Swagger Documentation

API documentation is auto-generated from handler annotations.

### Regenerate Docs

```bash
# After modifying handler annotations
make swagger

# Docs generated to docs/swagger.json
```

### View Documentation

Start the server and visit: http://localhost:8080/swagger/index.html

### Annotation Format

```go
// Create godoc
// @Summary Create a transaction
// @Description Create a new income or expense transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateTransactionInput true "Transaction data"
// @Success 201 {object} model.Transaction
// @Failure 400 {object} handler.ErrorResponse
// @Failure 401 {object} handler.ErrorResponse
// @Router /transactions [post]
func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
```

## Error Handling

### AppError Pattern

Use `internal/apperror` for HTTP-aware errors:

```go
// In handlers
respondAppError(w, apperror.BadRequest("invalid input"))
respondAppError(w, apperror.ValidationError("amount", "must be positive"))
respondAppError(w, apperror.NotFound("transaction"))
respondAppError(w, apperror.Internal(err))
```

### Repository Errors

Define sentinel errors for not-found cases:

```go
var ErrTransactionNotFound = errors.New("transaction not found")

// Check in handlers
if errors.Is(err, repository.ErrTransactionNotFound) {
    respondAppError(w, apperror.NotFound("transaction"))
    return
}
```

## Logging

Uses structured logging with `log/slog`:

```go
// Development: text format
// Production (ENV=production): JSON format

logger.Info("Transaction created",
    slog.String("user_id", userID.String()),
    slog.String("type", string(tx.Type)),
    slog.String("amount", tx.Amount.String()),
)
```

## Rules

Detailed development rules and patterns are documented in:
- `.claude/rules/backend.md` - Comprehensive Go patterns for this codebase

Key rules:
- **Interface Segregation**: Define interfaces in the consuming package
- **Dependency Injection**: Use constructors (NewXxxHandler, NewXxxService)
- **Parameterized Queries**: Never use string interpolation for SQL
- **Error Wrapping**: Use `fmt.Errorf("context: %w", err)`
- **Table-Driven Tests**: Use parallel execution with `t.Parallel()`
- **Swagger Required**: Every endpoint must have complete annotations
- **UUID for IDs**: Use `github.com/google/uuid`
- **Decimal for Money**: Use `github.com/shopspring/decimal`

## CI/CD

### GitHub Actions

Tests run on push to main via GitHub Actions:
1. Run `make test`
2. On success, build Docker image
3. Push to GHCR: `ghcr.io/wealthpathorganization/backend`

### Docker Build

```bash
# Build image
docker build -t wealthpath-backend .

# Build with make (CI-friendly)
make build-ci
```

## Common Tasks

### Add New Endpoint

1. Define service interface method in handler file
2. Implement method in service
3. Implement repository method if needed
4. Add handler method with Swagger annotations
5. Register route in `cmd/api/main.go`
6. Regenerate Swagger: `make swagger`
7. Add tests for handler, service, and repository

### Add Database Migration

1. Create `migrations/db/migration/V{next}__{description}.sql`
2. Run `make migrate-local` to test
3. Commit migration file

### Add New Entity

1. Define model in `internal/model/models.go`
2. Create repository with interface
3. Create service with interface
4. Create handler with interface
5. Register routes
6. Add migration for table

## Related Services

- **Frontend**: Next.js app consuming this API
- **Mobile**: Expo/React Native app consuming this API
- **Infrastructure**: See `infra-ansible/` and `wealthpath-k8s/` repos
