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
	"github.com/wealthpath/backend/internal/repository"
)

// MockReportRepository implements ReportRepositoryInterface for testing
type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) GetMonthlyTotals(ctx context.Context, userID uuid.UUID, year, month int) (decimal.Decimal, decimal.Decimal, error) {
	args := m.Called(ctx, userID, year, month)
	return args.Get(0).(decimal.Decimal), args.Get(1).(decimal.Decimal), args.Error(2)
}

func (m *MockReportRepository) GetTopExpenseCategories(ctx context.Context, userID uuid.UUID, year, month int, limit int) ([]repository.CategoryTotal, error) {
	args := m.Called(ctx, userID, year, month, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.CategoryTotal), args.Error(1)
}

func (m *MockReportRepository) GetCategoryAverageForPeriod(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, int, error) {
	args := m.Called(ctx, userID, category, startDate, endDate)
	return args.Get(0).(decimal.Decimal), args.Int(1), args.Error(2)
}

func (m *MockReportRepository) GetIncomeCategoryAverageForPeriod(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, int, error) {
	args := m.Called(ctx, userID, category, startDate, endDate)
	return args.Get(0).(decimal.Decimal), args.Int(1), args.Error(2)
}

func (m *MockReportRepository) GetCategoryAmountForMonth(ctx context.Context, userID uuid.UUID, category string, year, month int) (decimal.Decimal, error) {
	args := m.Called(ctx, userID, category, year, month)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockReportRepository) GetIncomeCategoryAmountForMonth(ctx context.Context, userID uuid.UUID, category string, year, month int) (decimal.Decimal, error) {
	args := m.Called(ctx, userID, category, year, month)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockReportRepository) GetCategoryTrendsData(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, categoryLimit int) ([]repository.CategoryMonthlyAmount, error) {
	args := m.Called(ctx, userID, startDate, endDate, categoryLimit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.CategoryMonthlyAmount), args.Error(1)
}

func (m *MockReportRepository) GetDistinctExpenseCategories(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]string, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockReportRepository) GetDistinctIncomeCategories(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]string, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockReportRepository) GetUserCurrency(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func TestNewReportService(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockReportRepository)
	svc := NewReportService(mockRepo)

	assert.NotNil(t, svc)
}

func TestReportService_GetMonthlyReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		year      int
		month     int
		setupMock func(*MockReportRepository, uuid.UUID)
		wantErr   bool
		check     func(*testing.T, *MonthlyReport)
	}{
		{
			name:  "success - with transactions",
			year:  2026,
			month: 1,
			setupMock: func(repo *MockReportRepository, userID uuid.UUID) {
				repo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
				repo.On("GetMonthlyTotals", mock.Anything, userID, 2026, 1).Return(
					decimal.NewFromFloat(8500),
					decimal.NewFromFloat(5200),
					nil,
				)
				repo.On("GetTopExpenseCategories", mock.Anything, userID, 2026, 1, 5).Return(
					[]repository.CategoryTotal{
						{Category: "Housing", Amount: decimal.NewFromFloat(1800), TransactionCount: 2},
						{Category: "Food & Dining", Amount: decimal.NewFromFloat(850), TransactionCount: 24},
					},
					nil,
				)
				// Mock for anomaly detection
				repo.On("GetDistinctExpenseCategories", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					[]string{"Housing", "Food & Dining"},
					nil,
				)
				repo.On("GetCategoryAverageForPeriod", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(500),
					3,
					nil,
				).Maybe()
				repo.On("GetCategoryAmountForMonth", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(800),
					nil,
				).Maybe()
				// Mock for income anomalies
				repo.On("GetDistinctIncomeCategories", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					[]string{"Salary"},
					nil,
				)
				repo.On("GetIncomeCategoryAverageForPeriod", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(8000),
					3,
					nil,
				).Maybe()
				repo.On("GetIncomeCategoryAmountForMonth", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything).Return(
					decimal.NewFromFloat(8500),
					nil,
				).Maybe()
				// For comparison
				repo.On("GetMonthlyTotals", mock.Anything, userID, 2025, 12).Return(
					decimal.NewFromFloat(8000),
					decimal.NewFromFloat(5500),
					nil,
				).Maybe()
			},
			wantErr: false,
			check: func(t *testing.T, report *MonthlyReport) {
				assert.Equal(t, 2026, report.Year)
				assert.Equal(t, 1, report.Month)
				assert.Equal(t, "USD", report.Currency)
				assert.Equal(t, "8500.00", report.TotalIncome)
				assert.Equal(t, "5200.00", report.TotalExpenses)
				assert.Equal(t, "3300.00", report.NetSavings)
				assert.True(t, report.SavingsRate > 0)
				assert.NotEmpty(t, report.TopCategories)
			},
		},
		{
			name:  "success - no transactions (empty state)",
			year:  2026,
			month: 1,
			setupMock: func(repo *MockReportRepository, userID uuid.UUID) {
				repo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
				repo.On("GetMonthlyTotals", mock.Anything, userID, 2026, 1).Return(
					decimal.Zero,
					decimal.Zero,
					nil,
				)
				repo.On("GetTopExpenseCategories", mock.Anything, userID, 2026, 1, 5).Return(
					[]repository.CategoryTotal{},
					nil,
				)
				repo.On("GetDistinctExpenseCategories", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					[]string{},
					nil,
				)
				repo.On("GetDistinctIncomeCategories", mock.Anything, userID, mock.Anything, mock.Anything).Return(
					[]string{},
					nil,
				)
				repo.On("GetMonthlyTotals", mock.Anything, userID, 2025, 12).Return(
					decimal.Zero,
					decimal.Zero,
					nil,
				).Maybe()
			},
			wantErr: false,
			check: func(t *testing.T, report *MonthlyReport) {
				assert.Equal(t, 2026, report.Year)
				assert.Equal(t, 1, report.Month)
				assert.Equal(t, "0.00", report.TotalIncome)
				assert.Equal(t, "0.00", report.TotalExpenses)
				assert.Equal(t, "0.00", report.NetSavings)
				assert.Equal(t, float64(0), report.SavingsRate)
			},
		},
		{
			name:  "error - repository failure",
			year:  2026,
			month: 1,
			setupMock: func(repo *MockReportRepository, userID uuid.UUID) {
				repo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
				repo.On("GetMonthlyTotals", mock.Anything, userID, 2026, 1).Return(
					decimal.Zero,
					decimal.Zero,
					errors.New("database error"),
				)
			},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockReportRepository)
			svc := NewReportService(mockRepo)
			userID := uuid.New()

			tt.setupMock(mockRepo, userID)

			report, err := svc.GetMonthlyReport(context.Background(), userID, tt.year, tt.month)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, report)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, report)
				if tt.check != nil {
					tt.check(t, report)
				}
			}
		})
	}
}

func TestReportService_GetCategoryTrends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		months    int
		limit     int
		setupMock func(*MockReportRepository, uuid.UUID)
		wantErr   bool
		check     func(*testing.T, *CategoryTrendsResponse)
	}{
		{
			name:   "success - with data",
			months: 6,
			limit:  5,
			setupMock: func(repo *MockReportRepository, userID uuid.UUID) {
				repo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
				repo.On("GetCategoryTrendsData", mock.Anything, userID, mock.Anything, mock.Anything, 5).Return(
					[]repository.CategoryMonthlyAmount{
						{Category: "Housing", Month: "2025-08", Amount: decimal.NewFromFloat(1800)},
						{Category: "Housing", Month: "2025-09", Amount: decimal.NewFromFloat(1800)},
						{Category: "Food", Month: "2025-08", Amount: decimal.NewFromFloat(500)},
						{Category: "Food", Month: "2025-09", Amount: decimal.NewFromFloat(600)},
					},
					nil,
				)
			},
			wantErr: false,
			check: func(t *testing.T, result *CategoryTrendsResponse) {
				assert.Equal(t, "USD", result.Currency)
				assert.NotEmpty(t, result.PeriodStart)
				assert.NotEmpty(t, result.PeriodEnd)
				assert.NotEmpty(t, result.Trends)
			},
		},
		{
			name:   "success - empty data",
			months: 6,
			limit:  10,
			setupMock: func(repo *MockReportRepository, userID uuid.UUID) {
				repo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
				repo.On("GetCategoryTrendsData", mock.Anything, userID, mock.Anything, mock.Anything, 10).Return(
					[]repository.CategoryMonthlyAmount{},
					nil,
				)
			},
			wantErr: false,
			check: func(t *testing.T, result *CategoryTrendsResponse) {
				assert.Empty(t, result.Trends)
			},
		},
		{
			name:   "error - repository failure",
			months: 6,
			limit:  10,
			setupMock: func(repo *MockReportRepository, userID uuid.UUID) {
				repo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
				repo.On("GetCategoryTrendsData", mock.Anything, userID, mock.Anything, mock.Anything, 10).Return(
					nil,
					errors.New("database error"),
				)
			},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockReportRepository)
			svc := NewReportService(mockRepo)
			userID := uuid.New()

			tt.setupMock(mockRepo, userID)

			result, err := svc.GetCategoryTrends(context.Background(), userID, tt.months, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.check != nil {
					tt.check(t, result)
				}
			}
		})
	}
}

func TestDetermineTrendFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		incomeChange      float64
		expenseChange     float64
		savingsChange     float64
		expectedDirection string
	}{
		{
			name:              "improving - savings up significantly (>10%)",
			incomeChange:      0,
			expenseChange:     0,
			savingsChange:     15,
			expectedDirection: "improving",
		},
		{
			name:              "improving - expenses down significantly with stable income",
			incomeChange:      5,
			expenseChange:     -10, // < -5 triggers improving
			savingsChange:     0,
			expectedDirection: "improving",
		},
		{
			name:              "declining - savings down significantly (<-10%)",
			incomeChange:      0,
			expenseChange:     0,
			savingsChange:     -15,
			expectedDirection: "declining",
		},
		{
			name:              "declining - expenses up significantly with income down",
			incomeChange:      -5,
			expenseChange:     20, // > 15 with income <= 0 triggers declining
			savingsChange:     0,
			expectedDirection: "declining",
		},
		{
			name:              "stable - minor changes",
			incomeChange:      2,
			expenseChange:     2,
			savingsChange:     2,
			expectedDirection: "stable",
		},
		{
			name:              "stable - borderline savings change",
			incomeChange:      0,
			expenseChange:     0,
			savingsChange:     10, // exactly 10 is not > 10, so stable
			expectedDirection: "stable",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			direction := determineTrend(tt.incomeChange, tt.expenseChange, tt.savingsChange)
			assert.Equal(t, tt.expectedDirection, direction)
		})
	}
}

func TestSavingsRateCalculation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		income       decimal.Decimal
		expenses     decimal.Decimal
		expectedRate float64
	}{
		{
			name:         "positive savings rate",
			income:       decimal.NewFromFloat(8500),
			expenses:     decimal.NewFromFloat(5200),
			expectedRate: 38.82,
		},
		{
			name:         "zero income - zero rate",
			income:       decimal.Zero,
			expenses:     decimal.NewFromFloat(100),
			expectedRate: 0,
		},
		{
			name:         "100% savings rate",
			income:       decimal.NewFromFloat(1000),
			expenses:     decimal.Zero,
			expectedRate: 100,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			netSavings := tt.income.Sub(tt.expenses)
			var savingsRate float64
			if !tt.income.IsZero() {
				savingsRate = netSavings.Div(tt.income).Mul(decimal.NewFromInt(100)).InexactFloat64()
			}

			if tt.expectedRate == 0 {
				assert.Equal(t, float64(0), savingsRate)
			} else {
				assert.InDelta(t, tt.expectedRate, savingsRate, 1.0)
			}
		})
	}
}

// Benchmark tests
func BenchmarkReportService_GetMonthlyReport(b *testing.B) {
	mockRepo := new(MockReportRepository)
	svc := NewReportService(mockRepo)
	userID := uuid.New()

	mockRepo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
	mockRepo.On("GetMonthlyTotals", mock.Anything, userID, mock.Anything, mock.Anything).Return(
		decimal.NewFromFloat(8500),
		decimal.NewFromFloat(5200),
		nil,
	)
	mockRepo.On("GetTopExpenseCategories", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything).Return(
		[]repository.CategoryTotal{{Category: "Housing", Amount: decimal.NewFromFloat(1800), TransactionCount: 2}},
		nil,
	)
	mockRepo.On("GetDistinctExpenseCategories", mock.Anything, userID, mock.Anything, mock.Anything).Return([]string{}, nil)
	mockRepo.On("GetDistinctIncomeCategories", mock.Anything, userID, mock.Anything, mock.Anything).Return([]string{}, nil)
	mockRepo.On("GetMonthlyTotals", mock.Anything, userID, 2025, 12).Return(decimal.Zero, decimal.Zero, nil).Maybe()

	for i := 0; i < b.N; i++ {
		_, _ = svc.GetMonthlyReport(context.Background(), userID, 2026, 1)
	}
}

func BenchmarkReportService_GetCategoryTrends(b *testing.B) {
	mockRepo := new(MockReportRepository)
	svc := NewReportService(mockRepo)
	userID := uuid.New()

	mockRepo.On("GetUserCurrency", mock.Anything, userID).Return("USD", nil)
	mockRepo.On("GetCategoryTrendsData", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything).Return(
		[]repository.CategoryMonthlyAmount{},
		nil,
	)

	for i := 0; i < b.N; i++ {
		_, _ = svc.GetCategoryTrends(context.Background(), userID, 6, 10)
	}
}
