package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/model"
)

// Mock implementations for dashboard repos
type MockDashboardTxRepo struct {
	mock.Mock
}

func (m *MockDashboardTxRepo) GetMonthlyTotals(ctx context.Context, userID uuid.UUID, year, month int) (decimal.Decimal, decimal.Decimal, error) {
	args := m.Called(ctx, userID, year, month)
	return args.Get(0).(decimal.Decimal), args.Get(1).(decimal.Decimal), args.Error(2)
}

func (m *MockDashboardTxRepo) GetExpensesByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (map[string]decimal.Decimal, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]decimal.Decimal), args.Error(1)
}

func (m *MockDashboardTxRepo) GetSpentByCategory(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, error) {
	args := m.Called(ctx, userID, category, startDate, endDate)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockDashboardTxRepo) GetRecentTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.Transaction, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Transaction), args.Error(1)
}

type MockDashboardBudgetRepo struct {
	mock.Mock
}

func (m *MockDashboardBudgetRepo) GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]model.Budget, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Budget), args.Error(1)
}

type MockDashboardSavingsRepo struct {
	mock.Mock
}

func (m *MockDashboardSavingsRepo) List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.SavingsGoal), args.Error(1)
}

func (m *MockDashboardSavingsRepo) GetTotalSavings(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

type MockDashboardDebtRepo struct {
	mock.Mock
}

func (m *MockDashboardDebtRepo) GetTotalDebt(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func TestNewDashboardService(t *testing.T) {
	t.Parallel()

	txRepo := new(MockDashboardTxRepo)
	budgetRepo := new(MockDashboardBudgetRepo)
	savingsRepo := new(MockDashboardSavingsRepo)
	debtRepo := new(MockDashboardDebtRepo)

	service := NewDashboardService(txRepo, budgetRepo, savingsRepo, debtRepo)

	assert.NotNil(t, service)
}

func TestDashboardService_GetDashboard(t *testing.T) {
	t.Parallel()

	txRepo := new(MockDashboardTxRepo)
	budgetRepo := new(MockDashboardBudgetRepo)
	savingsRepo := new(MockDashboardSavingsRepo)
	debtRepo := new(MockDashboardDebtRepo)

	service := NewDashboardService(txRepo, budgetRepo, savingsRepo, debtRepo)
	userID := uuid.New()

	// Setup mocks for all calls in GetMonthlyDashboard
	txRepo.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
		decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
	)
	txRepo.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
		map[string]decimal.Decimal{"Food": decimal.NewFromFloat(500)}, nil,
	)
	budgetRepo.On("GetActiveForUser", mock.Anything, userID).Return([]model.Budget{}, nil)
	savingsRepo.On("List", mock.Anything, userID).Return([]model.SavingsGoal{}, nil)
	savingsRepo.On("GetTotalSavings", mock.Anything, userID).Return(decimal.NewFromFloat(10000), nil)
	debtRepo.On("GetTotalDebt", mock.Anything, userID).Return(decimal.NewFromFloat(5000), nil)
	txRepo.On("GetRecentTransactions", mock.Anything, userID, 10).Return([]model.Transaction{}, nil)

	dashboard, err := service.GetDashboard(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, dashboard)
}

func TestDashboardService_GetMonthlyDashboard_Success(t *testing.T) {
	t.Parallel()

	txRepo := new(MockDashboardTxRepo)
	budgetRepo := new(MockDashboardBudgetRepo)
	savingsRepo := new(MockDashboardSavingsRepo)
	debtRepo := new(MockDashboardDebtRepo)

	service := NewDashboardService(txRepo, budgetRepo, savingsRepo, debtRepo)
	userID := uuid.New()

	// Setup comprehensive mocks
	txRepo.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
		decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
	)
	txRepo.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
		map[string]decimal.Decimal{
			"Food":      decimal.NewFromFloat(500),
			"Transport": decimal.NewFromFloat(300),
		}, nil,
	)
	budgetRepo.On("GetActiveForUser", mock.Anything, userID).Return([]model.Budget{
		{
			ID:       uuid.New(),
			UserID:   userID,
			Category: "Food",
			Amount:   decimal.NewFromFloat(600),
		},
	}, nil)
	txRepo.On("GetSpentByCategory", mock.Anything, userID, "Food", mock.Anything, mock.Anything).Return(
		decimal.NewFromFloat(500), nil,
	)
	savingsRepo.On("List", mock.Anything, userID).Return([]model.SavingsGoal{
		{ID: uuid.New(), Name: "Emergency Fund", TargetAmount: decimal.NewFromFloat(10000)},
	}, nil)
	savingsRepo.On("GetTotalSavings", mock.Anything, userID).Return(decimal.NewFromFloat(10000), nil)
	debtRepo.On("GetTotalDebt", mock.Anything, userID).Return(decimal.NewFromFloat(5000), nil)
	txRepo.On("GetRecentTransactions", mock.Anything, userID, 10).Return([]model.Transaction{
		{ID: uuid.New(), Description: "Groceries"},
	}, nil)

	dashboard, err := service.GetMonthlyDashboard(context.Background(), userID, 2024, 6)

	assert.NoError(t, err)
	assert.NotNil(t, dashboard)
	assert.Equal(t, decimal.NewFromFloat(5000), dashboard.TotalIncome)
	assert.Equal(t, decimal.NewFromFloat(3000), dashboard.TotalExpenses)
	assert.Equal(t, decimal.NewFromFloat(2000), dashboard.NetCashFlow)
	assert.Len(t, dashboard.BudgetSummary, 1)
	assert.Len(t, dashboard.SavingsGoals, 1)
}

func TestDashboardService_GetMonthlyDashboard_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*MockDashboardTxRepo, *MockDashboardBudgetRepo, *MockDashboardSavingsRepo, *MockDashboardDebtRepo, uuid.UUID)
	}{
		{
			name: "monthly totals error",
			setupMocks: func(tx *MockDashboardTxRepo, b *MockDashboardBudgetRepo, s *MockDashboardSavingsRepo, d *MockDashboardDebtRepo, userID uuid.UUID) {
				tx.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					decimal.Zero, decimal.Zero, errors.New("db error"),
				)
			},
		},
		{
			name: "expenses by category error",
			setupMocks: func(tx *MockDashboardTxRepo, b *MockDashboardBudgetRepo, s *MockDashboardSavingsRepo, d *MockDashboardDebtRepo, userID uuid.UUID) {
				tx.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
				)
				tx.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					(map[string]decimal.Decimal)(nil), errors.New("db error"),
				)
			},
		},
		{
			name: "budget repo error",
			setupMocks: func(tx *MockDashboardTxRepo, b *MockDashboardBudgetRepo, s *MockDashboardSavingsRepo, d *MockDashboardDebtRepo, userID uuid.UUID) {
				tx.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
				)
				tx.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					map[string]decimal.Decimal{}, nil,
				)
				b.On("GetActiveForUser", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
		},
		{
			name: "savings list error",
			setupMocks: func(tx *MockDashboardTxRepo, b *MockDashboardBudgetRepo, s *MockDashboardSavingsRepo, d *MockDashboardDebtRepo, userID uuid.UUID) {
				tx.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
				)
				tx.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					map[string]decimal.Decimal{}, nil,
				)
				b.On("GetActiveForUser", mock.Anything, userID).Return([]model.Budget{}, nil)
				s.On("List", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
		},
		{
			name: "total savings error",
			setupMocks: func(tx *MockDashboardTxRepo, b *MockDashboardBudgetRepo, s *MockDashboardSavingsRepo, d *MockDashboardDebtRepo, userID uuid.UUID) {
				tx.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
				)
				tx.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					map[string]decimal.Decimal{}, nil,
				)
				b.On("GetActiveForUser", mock.Anything, userID).Return([]model.Budget{}, nil)
				s.On("List", mock.Anything, userID).Return([]model.SavingsGoal{}, nil)
				s.On("GetTotalSavings", mock.Anything, userID).Return(decimal.Zero, errors.New("db error"))
			},
		},
		{
			name: "total debt error",
			setupMocks: func(tx *MockDashboardTxRepo, b *MockDashboardBudgetRepo, s *MockDashboardSavingsRepo, d *MockDashboardDebtRepo, userID uuid.UUID) {
				tx.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
				)
				tx.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					map[string]decimal.Decimal{}, nil,
				)
				b.On("GetActiveForUser", mock.Anything, userID).Return([]model.Budget{}, nil)
				s.On("List", mock.Anything, userID).Return([]model.SavingsGoal{}, nil)
				s.On("GetTotalSavings", mock.Anything, userID).Return(decimal.NewFromFloat(10000), nil)
				d.On("GetTotalDebt", mock.Anything, userID).Return(decimal.Zero, errors.New("db error"))
			},
		},
		{
			name: "recent transactions error",
			setupMocks: func(tx *MockDashboardTxRepo, b *MockDashboardBudgetRepo, s *MockDashboardSavingsRepo, d *MockDashboardDebtRepo, userID uuid.UUID) {
				tx.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
				)
				tx.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					map[string]decimal.Decimal{}, nil,
				)
				b.On("GetActiveForUser", mock.Anything, userID).Return([]model.Budget{}, nil)
				s.On("List", mock.Anything, userID).Return([]model.SavingsGoal{}, nil)
				s.On("GetTotalSavings", mock.Anything, userID).Return(decimal.NewFromFloat(10000), nil)
				d.On("GetTotalDebt", mock.Anything, userID).Return(decimal.NewFromFloat(5000), nil)
				tx.On("GetRecentTransactions", mock.Anything, userID, 10).Return(nil, errors.New("db error"))
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txRepo := new(MockDashboardTxRepo)
			budgetRepo := new(MockDashboardBudgetRepo)
			savingsRepo := new(MockDashboardSavingsRepo)
			debtRepo := new(MockDashboardDebtRepo)

			service := NewDashboardService(txRepo, budgetRepo, savingsRepo, debtRepo)
			userID := uuid.New()
			tt.setupMocks(txRepo, budgetRepo, savingsRepo, debtRepo, userID)

			dashboard, err := service.GetMonthlyDashboard(context.Background(), userID, 2024, 6)

			assert.Error(t, err)
			assert.Nil(t, dashboard)
		})
	}
}

func TestDashboardService_BudgetPercentageCalculation(t *testing.T) {
	t.Parallel()

	txRepo := new(MockDashboardTxRepo)
	budgetRepo := new(MockDashboardBudgetRepo)
	savingsRepo := new(MockDashboardSavingsRepo)
	debtRepo := new(MockDashboardDebtRepo)

	service := NewDashboardService(txRepo, budgetRepo, savingsRepo, debtRepo)
	userID := uuid.New()

	// Test with zero budget amount to ensure no division by zero
	txRepo.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
		decimal.NewFromFloat(5000), decimal.NewFromFloat(3000), nil,
	)
	txRepo.On("GetExpensesByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(
		map[string]decimal.Decimal{}, nil,
	)
	budgetRepo.On("GetActiveForUser", mock.Anything, userID).Return([]model.Budget{
		{
			ID:       uuid.New(),
			UserID:   userID,
			Category: "Food",
			Amount:   decimal.Zero, // Zero budget
		},
	}, nil)
	txRepo.On("GetSpentByCategory", mock.Anything, userID, "Food", mock.Anything, mock.Anything).Return(
		decimal.NewFromFloat(100), nil,
	)
	savingsRepo.On("List", mock.Anything, userID).Return([]model.SavingsGoal{}, nil)
	savingsRepo.On("GetTotalSavings", mock.Anything, userID).Return(decimal.Zero, nil)
	debtRepo.On("GetTotalDebt", mock.Anything, userID).Return(decimal.Zero, nil)
	txRepo.On("GetRecentTransactions", mock.Anything, userID, 10).Return([]model.Transaction{}, nil)

	dashboard, err := service.GetMonthlyDashboard(context.Background(), userID, 2024, 6)

	assert.NoError(t, err)
	assert.NotNil(t, dashboard)
	assert.Len(t, dashboard.BudgetSummary, 1)
	assert.Equal(t, float64(0), dashboard.BudgetSummary[0].Percentage) // No division by zero
}
