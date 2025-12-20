package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/wealthpath/backend/internal/model"
)

// DashboardTransactionRepo provides transaction data needed for dashboard aggregations.
type DashboardTransactionRepo interface {
	GetMonthlyTotals(ctx context.Context, userID uuid.UUID, year, month int) (decimal.Decimal, decimal.Decimal, error)
	GetExpensesByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (map[string]decimal.Decimal, error)
	GetSpentByCategory(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, error)
	GetRecentTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.Transaction, error)
}

// DashboardBudgetRepo provides budget data needed for dashboard.
type DashboardBudgetRepo interface {
	GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
}

// DashboardSavingsRepo provides savings data needed for dashboard.
type DashboardSavingsRepo interface {
	List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error)
	GetTotalSavings(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
}

// DashboardDebtRepo provides debt data needed for dashboard.
type DashboardDebtRepo interface {
	GetTotalDebt(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
}

// DashboardService aggregates financial data from multiple sources for dashboard display.
type DashboardService struct {
	transactionRepo DashboardTransactionRepo
	budgetRepo      DashboardBudgetRepo
	savingsRepo     DashboardSavingsRepo
	debtRepo        DashboardDebtRepo
}

// NewDashboardService creates a new DashboardService with the required repository dependencies.
func NewDashboardService(
	transactionRepo DashboardTransactionRepo,
	budgetRepo DashboardBudgetRepo,
	savingsRepo DashboardSavingsRepo,
	debtRepo DashboardDebtRepo,
) *DashboardService {
	return &DashboardService{
		transactionRepo: transactionRepo,
		budgetRepo:      budgetRepo,
		savingsRepo:     savingsRepo,
		debtRepo:        debtRepo,
	}
}

// GetDashboard retrieves dashboard data for the current month.
func (s *DashboardService) GetDashboard(ctx context.Context, userID uuid.UUID) (*model.DashboardData, error) {
	now := time.Now()
	return s.GetMonthlyDashboard(ctx, userID, now.Year(), int(now.Month()))
}

// GetMonthlyDashboard retrieves aggregated financial data for a specific month.
// Includes income/expenses, budget progress, savings goals, debt totals, and trends.
func (s *DashboardService) GetMonthlyDashboard(ctx context.Context, userID uuid.UUID, year, month int) (*model.DashboardData, error) {
	income, expenses, err := s.transactionRepo.GetMonthlyTotals(ctx, userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("getting monthly totals: %w", err)
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	expensesByCategory, err := s.transactionRepo.GetExpensesByCategory(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("getting expenses by category: %w", err)
	}

	budgets, err := s.budgetRepo.GetActiveForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting active budgets: %w", err)
	}

	budgetSummary := make([]model.BudgetWithSpent, len(budgets))
	for i, budget := range budgets {
		spent, err := s.transactionRepo.GetSpentByCategory(ctx, userID, budget.Category, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("getting spent for budget %s: %w", budget.Category, err)
		}

		remaining := budget.Amount.Sub(spent)
		percentage := float64(0)
		if !budget.Amount.IsZero() {
			percentage = spent.Div(budget.Amount).Mul(decimal.NewFromInt(100)).InexactFloat64()
		}

		budgetSummary[i] = model.BudgetWithSpent{
			Budget:     budget,
			Spent:      spent,
			Remaining:  remaining,
			Percentage: percentage,
		}
	}

	savingsGoals, err := s.savingsRepo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting savings goals: %w", err)
	}

	totalSavings, err := s.savingsRepo.GetTotalSavings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting total savings: %w", err)
	}

	totalDebt, err := s.debtRepo.GetTotalDebt(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting total debt: %w", err)
	}

	recentTransactions, err := s.transactionRepo.GetRecentTransactions(ctx, userID, 10)
	if err != nil {
		return nil, fmt.Errorf("getting recent transactions: %w", err)
	}

	incomeVsExpenses := make([]model.MonthlyComparison, 6)
	for i := 5; i >= 0; i-- {
		m := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -i, 0)
		inc, exp, err := s.transactionRepo.GetMonthlyTotals(ctx, userID, m.Year(), int(m.Month()))
		if err != nil {
			return nil, fmt.Errorf("getting monthly comparison for %s: %w", m.Format("Jan 2006"), err)
		}
		incomeVsExpenses[5-i] = model.MonthlyComparison{
			Month:    m.Format("Jan 2006"),
			Income:   inc,
			Expenses: exp,
		}
	}

	return &model.DashboardData{
		TotalIncome:        income,
		TotalExpenses:      expenses,
		NetCashFlow:        income.Sub(expenses),
		TotalSavings:       totalSavings,
		TotalDebt:          totalDebt,
		BudgetSummary:      budgetSummary,
		SavingsGoals:       savingsGoals,
		RecentTransactions: recentTransactions,
		ExpensesByCategory: expensesByCategory,
		IncomeVsExpenses:   incomeVsExpenses,
	}, nil
}
