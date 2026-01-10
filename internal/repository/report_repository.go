package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// CategoryTotal represents aggregated spending for a category.
type CategoryTotal struct {
	Category         string          `db:"category"`
	Amount           decimal.Decimal `db:"amount"`
	TransactionCount int             `db:"transaction_count"`
}

// MonthlyTotal represents income and expenses for a specific month.
type MonthlyTotal struct {
	Month    string          `db:"month"`
	Income   decimal.Decimal `db:"income"`
	Expenses decimal.Decimal `db:"expenses"`
}

// CategoryMonthlyAmount represents a category's spending for a specific month.
type CategoryMonthlyAmount struct {
	Category string          `db:"category"`
	Month    string          `db:"month"`
	Amount   decimal.Decimal `db:"amount"`
}

// ReportRepository provides data access for report generation.
type ReportRepository struct {
	db *sqlx.DB
}

// NewReportRepository creates a new ReportRepository with the given database connection.
func NewReportRepository(db *sqlx.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// GetMonthlyTotals retrieves total income and expenses for a specific month.
func (r *ReportRepository) GetMonthlyTotals(ctx context.Context, userID uuid.UUID, year, month int) (income, expenses decimal.Decimal, err error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expenses
		FROM transactions
		WHERE user_id = $1
		AND EXTRACT(YEAR FROM date) = $2
		AND EXTRACT(MONTH FROM date) = $3`

	var result struct {
		Income   decimal.Decimal `db:"income"`
		Expenses decimal.Decimal `db:"expenses"`
	}
	err = r.db.GetContext(ctx, &result, query, userID, year, month)
	return result.Income, result.Expenses, err
}

// GetTopExpenseCategories retrieves the top expense categories for a specific month.
func (r *ReportRepository) GetTopExpenseCategories(ctx context.Context, userID uuid.UUID, year, month int, limit int) ([]CategoryTotal, error) {
	query := `
		SELECT
			category,
			SUM(amount) as amount,
			COUNT(*) as transaction_count
		FROM transactions
		WHERE user_id = $1
			AND type = 'expense'
			AND EXTRACT(YEAR FROM date) = $2
			AND EXTRACT(MONTH FROM date) = $3
		GROUP BY category
		ORDER BY amount DESC
		LIMIT $4`

	var results []CategoryTotal
	err := r.db.SelectContext(ctx, &results, query, userID, year, month, limit)
	return results, err
}

// GetCategoryAverageForPeriod calculates the average spending for a category over a period.
func (r *ReportRepository) GetCategoryAverageForPeriod(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, int, error) {
	query := `
		SELECT
			COALESCE(SUM(amount), 0) as total,
			COUNT(DISTINCT TO_CHAR(date, 'YYYY-MM')) as month_count
		FROM transactions
		WHERE user_id = $1
			AND type = 'expense'
			AND category = $2
			AND date >= $3
			AND date < $4`

	var result struct {
		Total      decimal.Decimal `db:"total"`
		MonthCount int             `db:"month_count"`
	}
	err := r.db.GetContext(ctx, &result, query, userID, category, startDate, endDate)
	if err != nil {
		return decimal.Zero, 0, err
	}

	if result.MonthCount == 0 {
		return decimal.Zero, 0, nil
	}

	avg := result.Total.Div(decimal.NewFromInt(int64(result.MonthCount)))
	return avg, result.MonthCount, nil
}

// GetIncomeCategoryAverageForPeriod calculates the average income for a category over a period.
func (r *ReportRepository) GetIncomeCategoryAverageForPeriod(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, int, error) {
	query := `
		SELECT
			COALESCE(SUM(amount), 0) as total,
			COUNT(DISTINCT TO_CHAR(date, 'YYYY-MM')) as month_count
		FROM transactions
		WHERE user_id = $1
			AND type = 'income'
			AND category = $2
			AND date >= $3
			AND date < $4`

	var result struct {
		Total      decimal.Decimal `db:"total"`
		MonthCount int             `db:"month_count"`
	}
	err := r.db.GetContext(ctx, &result, query, userID, category, startDate, endDate)
	if err != nil {
		return decimal.Zero, 0, err
	}

	if result.MonthCount == 0 {
		return decimal.Zero, 0, nil
	}

	avg := result.Total.Div(decimal.NewFromInt(int64(result.MonthCount)))
	return avg, result.MonthCount, nil
}

// GetCategoryAmountForMonth retrieves the total spending for a category in a specific month.
func (r *ReportRepository) GetCategoryAmountForMonth(ctx context.Context, userID uuid.UUID, category string, year, month int) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1
			AND type = 'expense'
			AND category = $2
			AND EXTRACT(YEAR FROM date) = $3
			AND EXTRACT(MONTH FROM date) = $4`

	var amount decimal.Decimal
	err := r.db.GetContext(ctx, &amount, query, userID, category, year, month)
	return amount, err
}

// GetIncomeCategoryAmountForMonth retrieves the total income for a category in a specific month.
func (r *ReportRepository) GetIncomeCategoryAmountForMonth(ctx context.Context, userID uuid.UUID, category string, year, month int) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1
			AND type = 'income'
			AND category = $2
			AND EXTRACT(YEAR FROM date) = $3
			AND EXTRACT(MONTH FROM date) = $4`

	var amount decimal.Decimal
	err := r.db.GetContext(ctx, &amount, query, userID, category, year, month)
	return amount, err
}

// GetCategoryTrendsData retrieves monthly spending by category for trend analysis.
func (r *ReportRepository) GetCategoryTrendsData(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, categoryLimit int) ([]CategoryMonthlyAmount, error) {
	query := `
		WITH top_categories AS (
			SELECT category
			FROM transactions
			WHERE user_id = $1
				AND type = 'expense'
				AND date >= $2
				AND date < $3
			GROUP BY category
			ORDER BY SUM(amount) DESC
			LIMIT $4
		)
		SELECT
			t.category,
			TO_CHAR(t.date, 'YYYY-MM') as month,
			COALESCE(SUM(t.amount), 0) as amount
		FROM transactions t
		INNER JOIN top_categories tc ON t.category = tc.category
		WHERE t.user_id = $1
			AND t.type = 'expense'
			AND t.date >= $2
			AND t.date < $3
		GROUP BY t.category, TO_CHAR(t.date, 'YYYY-MM')
		ORDER BY t.category, month`

	var results []CategoryMonthlyAmount
	err := r.db.SelectContext(ctx, &results, query, userID, startDate, endDate, categoryLimit)
	return results, err
}

// GetDistinctExpenseCategories retrieves all distinct expense categories for a user in a period.
func (r *ReportRepository) GetDistinctExpenseCategories(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]string, error) {
	query := `
		SELECT DISTINCT category
		FROM transactions
		WHERE user_id = $1
			AND type = 'expense'
			AND date >= $2
			AND date < $3
		ORDER BY category`

	var categories []string
	err := r.db.SelectContext(ctx, &categories, query, userID, startDate, endDate)
	return categories, err
}

// GetDistinctIncomeCategories retrieves all distinct income categories for a user in a period.
func (r *ReportRepository) GetDistinctIncomeCategories(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]string, error) {
	query := `
		SELECT DISTINCT category
		FROM transactions
		WHERE user_id = $1
			AND type = 'income'
			AND date >= $2
			AND date < $3
		ORDER BY category`

	var categories []string
	err := r.db.SelectContext(ctx, &categories, query, userID, startDate, endDate)
	return categories, err
}

// GetUserCurrency retrieves the user's preferred currency.
func (r *ReportRepository) GetUserCurrency(ctx context.Context, userID uuid.UUID) (string, error) {
	query := `SELECT currency FROM users WHERE id = $1`
	var currency string
	err := r.db.GetContext(ctx, &currency, query, userID)
	if err != nil {
		return "USD", err
	}
	if currency == "" {
		return "USD", nil
	}
	return currency, nil
}
