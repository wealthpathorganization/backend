package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/wealthpath/backend/internal/model"
)

type RecurringRepository struct {
	db *sqlx.DB
}

func NewRecurringRepository(db *sqlx.DB) *RecurringRepository {
	return &RecurringRepository{db: db}
}

func (r *RecurringRepository) Create(ctx context.Context, rt *model.RecurringTransaction) error {
	query := `
		INSERT INTO recurring_transactions (id, user_id, type, amount, currency, category, description, 
			frequency, start_date, end_date, next_occurrence, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		RETURNING created_at, updated_at`

	rt.ID = uuid.New()
	return r.db.QueryRowxContext(ctx, query,
		rt.ID, rt.UserID, rt.Type, rt.Amount, rt.Currency, rt.Category, rt.Description,
		rt.Frequency, rt.StartDate, rt.EndDate, rt.NextOccurrence, rt.IsActive,
	).Scan(&rt.CreatedAt, &rt.UpdatedAt)
}

func (r *RecurringRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.RecurringTransaction, error) {
	var rt model.RecurringTransaction
	query := `SELECT * FROM recurring_transactions WHERE id = $1`
	err := r.db.GetContext(ctx, &rt, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("recurring transaction not found")
	}
	return &rt, err
}

func (r *RecurringRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error) {
	var items []model.RecurringTransaction
	query := `SELECT * FROM recurring_transactions WHERE user_id = $1 ORDER BY next_occurrence ASC`
	err := r.db.SelectContext(ctx, &items, query, userID)
	return items, err
}

func (r *RecurringRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error) {
	var items []model.RecurringTransaction
	query := `SELECT * FROM recurring_transactions WHERE user_id = $1 AND is_active = true ORDER BY next_occurrence ASC`
	err := r.db.SelectContext(ctx, &items, query, userID)
	return items, err
}

func (r *RecurringRepository) Update(ctx context.Context, rt *model.RecurringTransaction) error {
	query := `
		UPDATE recurring_transactions 
		SET type = $2, amount = $3, currency = $4, category = $5, description = $6,
			frequency = $7, start_date = $8, end_date = $9, next_occurrence = $10, 
			is_active = $11, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowxContext(ctx, query,
		rt.ID, rt.Type, rt.Amount, rt.Currency, rt.Category, rt.Description,
		rt.Frequency, rt.StartDate, rt.EndDate, rt.NextOccurrence, rt.IsActive,
	).Scan(&rt.UpdatedAt)
}

func (r *RecurringRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM recurring_transactions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetDueTransactions returns all active recurring transactions that are due
func (r *RecurringRepository) GetDueTransactions(ctx context.Context, before time.Time) ([]model.RecurringTransaction, error) {
	var items []model.RecurringTransaction
	query := `
		SELECT * FROM recurring_transactions 
		WHERE is_active = true 
			AND next_occurrence <= $1
			AND (end_date IS NULL OR end_date >= $1)
		ORDER BY next_occurrence ASC`
	err := r.db.SelectContext(ctx, &items, query, before)
	return items, err
}

// UpdateLastGenerated updates the last_generated and next_occurrence after generating a transaction
func (r *RecurringRepository) UpdateLastGenerated(ctx context.Context, id uuid.UUID, lastGenerated time.Time, nextOccurrence time.Time) error {
	query := `
		UPDATE recurring_transactions 
		SET last_generated = $2, next_occurrence = $3, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, lastGenerated, nextOccurrence)
	return err
}

// GetUpcoming returns upcoming bills/income for dashboard widget
func (r *RecurringRepository) GetUpcoming(ctx context.Context, userID uuid.UUID, limit int) ([]model.UpcomingBill, error) {
	var items []model.UpcomingBill
	query := `
		SELECT id, description, amount, currency, category, next_occurrence as due_date, type
		FROM recurring_transactions 
		WHERE user_id = $1 
			AND is_active = true 
			AND next_occurrence >= CURRENT_DATE
		ORDER BY next_occurrence ASC
		LIMIT $2`
	err := r.db.SelectContext(ctx, &items, query, userID, limit)
	return items, err
}
