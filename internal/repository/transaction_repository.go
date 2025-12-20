package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
)

var ErrTransactionNotFound = errors.New("transaction not found")

type TransactionRepository struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *model.Transaction) error {
	query := `
		INSERT INTO transactions (id, user_id, type, amount, currency, category, description, date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at`

	tx.ID = uuid.New()
	return r.db.QueryRowxContext(ctx, query,
		tx.ID, tx.UserID, tx.Type, tx.Amount, tx.Currency, tx.Category, tx.Description, tx.Date,
	).Scan(&tx.CreatedAt, &tx.UpdatedAt)
}

func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	var tx model.Transaction
	query := `SELECT * FROM transactions WHERE id = $1`
	err := r.db.GetContext(ctx, &tx, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTransactionNotFound
	}
	return &tx, err
}

func (r *TransactionRepository) List(ctx context.Context, userID uuid.UUID, filters TransactionFilters) ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := `
		SELECT * FROM transactions 
		WHERE user_id = $1
		AND ($2::text IS NULL OR type = $2)
		AND ($3::text IS NULL OR category = $3)
		AND ($4::timestamp IS NULL OR date >= $4)
		AND ($5::timestamp IS NULL OR date <= $5)
		ORDER BY date DESC, created_at DESC
		LIMIT $6 OFFSET $7`

	err := r.db.SelectContext(ctx, &transactions, query,
		userID, filters.Type, filters.Category, filters.StartDate, filters.EndDate, filters.Limit, filters.Offset,
	)
	return transactions, err
}

func (r *TransactionRepository) Update(ctx context.Context, tx *model.Transaction) error {
	query := `
		UPDATE transactions 
		SET type = $2, amount = $3, currency = $4, category = $5, description = $6, date = $7, updated_at = NOW()
		WHERE id = $1 AND user_id = $8
		RETURNING updated_at`
	result := r.db.QueryRowxContext(ctx, query,
		tx.ID, tx.Type, tx.Amount, tx.Currency, tx.Category, tx.Description, tx.Date, tx.UserID,
	)
	return result.Scan(&tx.UpdatedAt)
}

func (r *TransactionRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
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

func (r *TransactionRepository) GetMonthlyTotals(ctx context.Context, userID uuid.UUID, year int, month int) (income, expenses decimal.Decimal, err error) {
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

func (r *TransactionRepository) GetExpensesByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (map[string]decimal.Decimal, error) {
	query := `
		SELECT category, SUM(amount) as total
		FROM transactions
		WHERE user_id = $1 AND type = 'expense' AND date >= $2 AND date <= $3
		GROUP BY category`

	var results []struct {
		Category string          `db:"category"`
		Total    decimal.Decimal `db:"total"`
	}
	err := r.db.SelectContext(ctx, &results, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	categories := make(map[string]decimal.Decimal)
	for _, r := range results {
		categories[r.Category] = r.Total
	}
	return categories, nil
}

func (r *TransactionRepository) GetSpentByCategory(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1 AND type = 'expense' AND category = $2 AND date >= $3 AND date <= $4`

	var spent decimal.Decimal
	err := r.db.GetContext(ctx, &spent, query, userID, category, startDate, endDate)
	return spent, err
}

func (r *TransactionRepository) GetRecentTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := `SELECT * FROM transactions WHERE user_id = $1 ORDER BY date DESC, created_at DESC LIMIT $2`
	err := r.db.SelectContext(ctx, &transactions, query, userID, limit)
	return transactions, err
}

func (r *TransactionRepository) GetMonthlyComparison(ctx context.Context, userID uuid.UUID, months int) ([]model.MonthlyComparison, error) {
	query := `
		SELECT 
			TO_CHAR(date, 'YYYY-MM') as month,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expenses
		FROM transactions
		WHERE user_id = $1 AND date >= NOW() - INTERVAL '%d months'
		GROUP BY TO_CHAR(date, 'YYYY-MM')
		ORDER BY month`

	var results []model.MonthlyComparison
	err := r.db.SelectContext(ctx, &results, query, userID)
	return results, err
}

type TransactionFilters struct {
	Type      *string
	Category  *string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}
