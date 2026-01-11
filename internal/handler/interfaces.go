package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/service"
)

// BudgetServiceInterface for handler testing
type BudgetServiceInterface interface {
	Create(ctx context.Context, userID uuid.UUID, input service.CreateBudgetInput) (*model.Budget, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Budget, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
	ListWithSpent(ctx context.Context, userID uuid.UUID) ([]model.BudgetWithSpent, error)
	Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateBudgetInput) (*model.Budget, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// DebtServiceInterface for handler testing
type DebtServiceInterface interface {
	Create(ctx context.Context, userID uuid.UUID, input service.CreateDebtInput) (*model.Debt, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Debt, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.Debt, error)
	Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateDebtInput) (*model.Debt, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	MakePayment(ctx context.Context, id, userID uuid.UUID, input service.MakePaymentInput) (*model.Debt, error)
	GetPayoffPlan(ctx context.Context, id uuid.UUID, monthlyPayment decimal.Decimal) (*model.PayoffPlan, error)
	GetDebtSummary(ctx context.Context, userID uuid.UUID) (*service.DebtSummary, error)
	CalculateInterest(input service.InterestCalculatorInput) (*service.InterestCalculatorResult, error)
}

// SavingsGoalServiceInterface for handler testing
type SavingsGoalServiceInterface interface {
	Create(ctx context.Context, userID uuid.UUID, input service.CreateSavingsGoalInput) (*model.SavingsGoal, error)
	Get(ctx context.Context, id uuid.UUID) (*model.SavingsGoal, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error)
	ListWithProjections(ctx context.Context, userID uuid.UUID) ([]service.SavingsGoalWithProjection, error)
	Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateSavingsGoalInput) (*model.SavingsGoal, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	Contribute(ctx context.Context, id, userID uuid.UUID, amount decimal.Decimal) (*model.SavingsGoal, error)
}

// RecurringServiceInterface for handler testing
type RecurringServiceInterface interface {
	Create(ctx context.Context, userID uuid.UUID, input service.CreateRecurringInput) (*model.RecurringTransaction, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error)
	GetByID(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error)
	Update(ctx context.Context, userID, id uuid.UUID, input service.UpdateRecurringInput) (*model.RecurringTransaction, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	Pause(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error)
	Resume(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error)
	GetUpcoming(ctx context.Context, userID uuid.UUID, limit int) ([]model.UpcomingBill, error)
}

// Note: TransactionServiceInterface and UserServiceInterface are defined in their respective test files
