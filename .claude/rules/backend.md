# Backend (Go) Best Practices

## Architecture & Code Organization

- **Clean Architecture:** Follow the layered pattern: `handler → service → repository → model`
- **Dependency Injection:** Use constructor functions (`New*Handler`, `New*Service`, `New*Repository`) with interface dependencies
- **Interface Segregation:** Define small, focused interfaces in the consuming package (e.g., `TransactionHandlerServiceInterface` in handler package)

```go
// Good: Interface defined where it's used
type TransactionHandlerServiceInterface interface {
    Create(ctx context.Context, userID uuid.UUID, input CreateTransactionInput) (*model.Transaction, error)
    Get(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
}

type TransactionHandler struct {
    service TransactionHandlerServiceInterface
}

func NewTransactionHandler(service TransactionHandlerServiceInterface) *TransactionHandler {
    return &TransactionHandler{service: service}
}
```

## Handler Patterns

- **Standard Flow:** Extract context → Decode request → Validate → Call service → Respond
- **Response Functions:** Use `respondJSON()`, `respondError()`, `respondAppError()` consistently
- **Swagger Documentation:** Every endpoint must have swagger annotations

```go
// @Summary Create a transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateTransactionInput true "Transaction data"
// @Success 201 {object} model.Transaction
// @Router /api/transactions [post]
func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(UserIDKey).(uuid.UUID)
    // ... validation and service call
}
```

## Error Handling

- **Sentinel Errors:** Define package-level errors for common cases
- **AppError Type:** Use `apperror.AppError` for HTTP-aware errors with status codes

```go
// Repository layer - sentinel errors
var ErrTransactionNotFound = errors.New("transaction not found")

// Handler layer - AppError responses
if input.Amount.IsZero() {
    respondAppError(w, apperror.ValidationError("amount", "amount must be greater than 0"))
    return
}
```

## Data Types & Validation

- **UUID for IDs:** Always use `uuid.UUID` for entity identifiers
- **Decimal for Money:** Use `decimal.Decimal` from shopspring/decimal for all monetary amounts
- **Custom Types:** Use typed enums for constrained values

```go
type TransactionType string
const (
    TransactionTypeIncome  TransactionType = "income"
    TransactionTypeExpense TransactionType = "expense"
)
```

## Database Patterns

- **Parameterized Queries:** Always use `$1, $2` placeholders, never string interpolation
- **NULL Handling:** Use PostgreSQL coalescing: `($2::text IS NULL OR type = $2)`
- **Timestamps:** Use `NOW()` and `RETURNING` clause for generated values

```go
query := `
    INSERT INTO transactions (id, user_id, amount, type, created_at)
    VALUES ($1, $2, $3, $4, NOW())
    RETURNING created_at`
```

## Testing

- **Mock Interfaces:** Use `stretchr/testify/mock` for service mocks
- **Table-Driven Tests:** Group related test cases in struct slices
- **HTTP Testing:** Use `httptest.NewRequest` and `httptest.NewRecorder`

```go
tests := []struct {
    name    string
    input   CreateTransactionInput
    wantErr bool
}{
    {name: "valid expense", input: validInput, wantErr: false},
    {name: "missing type", input: missingType, wantErr: true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Packages | lowercase, short | `handler`, `service`, `repository` |
| Interfaces | PascalCase + `Interface` suffix | `TransactionRepositoryInterface` |
| Constructors | `New` prefix | `NewTransactionHandler()` |
| Errors | `Err` prefix | `ErrTransactionNotFound` |
| Receivers | 1-2 letter abbreviation | `(h *Handler)`, `(s *Service)` |
