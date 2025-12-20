package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/wealthpath/backend/internal/model"
)

var ErrBudgetNotFound = errors.New("budget not found")

type BudgetRepository struct {
	db *sqlx.DB
}

func NewBudgetRepository(db *sqlx.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

func (r *BudgetRepository) Create(ctx context.Context, budget *model.Budget) error {
	query := `
		INSERT INTO budgets (id, user_id, category, amount, currency, period, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at`

	budget.ID = uuid.New()
	return r.db.QueryRowxContext(ctx, query,
		budget.ID, budget.UserID, budget.Category, budget.Amount, budget.Currency,
		budget.Period, budget.StartDate, budget.EndDate,
	).Scan(&budget.CreatedAt, &budget.UpdatedAt)
}

func (r *BudgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error) {
	var budget model.Budget
	query := `SELECT * FROM budgets WHERE id = $1`
	err := r.db.GetContext(ctx, &budget, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBudgetNotFound
	}
	return &budget, err
}

func (r *BudgetRepository) List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error) {
	var budgets []model.Budget
	query := `SELECT * FROM budgets WHERE user_id = $1 ORDER BY category`
	err := r.db.SelectContext(ctx, &budgets, query, userID)
	return budgets, err
}

func (r *BudgetRepository) Update(ctx context.Context, budget *model.Budget) error {
	query := `
		UPDATE budgets 
		SET category = $2, amount = $3, currency = $4, period = $5, start_date = $6, end_date = $7, updated_at = NOW()
		WHERE id = $1 AND user_id = $8
		RETURNING updated_at`
	result := r.db.QueryRowxContext(ctx, query,
		budget.ID, budget.Category, budget.Amount, budget.Currency,
		budget.Period, budget.StartDate, budget.EndDate, budget.UserID,
	)
	return result.Scan(&budget.UpdatedAt)
}

func (r *BudgetRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM budgets WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrBudgetNotFound
	}
	return nil
}

func (r *BudgetRepository) GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]model.Budget, error) {
	var budgets []model.Budget
	query := `
		SELECT * FROM budgets 
		WHERE user_id = $1 
		AND (end_date IS NULL OR end_date >= NOW())
		ORDER BY category`
	err := r.db.SelectContext(ctx, &budgets, query, userID)
	return budgets, err
}
