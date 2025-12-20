package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
)

//go:generate mockery --name=UserRepositoryInterface --output=../mocks --outpkg=mocks
type UserRepositoryInterface interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByOAuth(ctx context.Context, provider, oauthID string) (*model.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, user *model.User) error
}

//go:generate mockery --name=TransactionRepositoryInterface --output=../mocks --outpkg=mocks
type TransactionRepositoryInterface interface {
	Create(ctx context.Context, tx *model.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
	List(ctx context.Context, userID uuid.UUID, filters TransactionFilters) ([]model.Transaction, error)
	Update(ctx context.Context, tx *model.Transaction) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	GetMonthlyTotals(ctx context.Context, userID uuid.UUID, year, month int) (decimal.Decimal, decimal.Decimal, error)
	GetExpensesByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (map[string]decimal.Decimal, error)
	GetSpentByCategory(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, error)
	GetRecentTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.Transaction, error)
	GetMonthlyComparison(ctx context.Context, userID uuid.UUID, months int) ([]model.MonthlyComparison, error)
}

//go:generate mockery --name=BudgetRepositoryInterface --output=../mocks --outpkg=mocks
type BudgetRepositoryInterface interface {
	Create(ctx context.Context, budget *model.Budget) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
	GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
	Update(ctx context.Context, budget *model.Budget) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

//go:generate mockery --name=DebtRepositoryInterface --output=../mocks --outpkg=mocks
type DebtRepositoryInterface interface {
	Create(ctx context.Context, debt *model.Debt) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Debt, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.Debt, error)
	Update(ctx context.Context, debt *model.Debt) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	RecordPayment(ctx context.Context, payment *model.DebtPayment) error
	GetTotalDebt(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
}

//go:generate mockery --name=SavingsGoalRepositoryInterface --output=../mocks --outpkg=mocks
type SavingsGoalRepositoryInterface interface {
	Create(ctx context.Context, goal *model.SavingsGoal) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.SavingsGoal, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error)
	Update(ctx context.Context, goal *model.SavingsGoal) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	AddContribution(ctx context.Context, id, userID uuid.UUID, amount decimal.Decimal) error
	GetTotalSavings(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
}

//go:generate mockery --name=RecurringRepositoryInterface --output=../mocks --outpkg=mocks
type RecurringRepositoryInterface interface {
	Create(ctx context.Context, rt *model.RecurringTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.RecurringTransaction, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error)
	Update(ctx context.Context, rt *model.RecurringTransaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUpcoming(ctx context.Context, userID uuid.UUID, limit int) ([]model.UpcomingBill, error)
	GetDueTransactions(ctx context.Context, now time.Time) ([]model.RecurringTransaction, error)
	UpdateLastGenerated(ctx context.Context, id uuid.UUID, lastGenerated, nextOccurrence time.Time) error
}
