package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
)

var ErrSavingsGoalNotFound = errors.New("savings goal not found")

type SavingsGoalRepository struct {
	db *sqlx.DB
}

func NewSavingsGoalRepository(db *sqlx.DB) *SavingsGoalRepository {
	return &SavingsGoalRepository{db: db}
}

func (r *SavingsGoalRepository) Create(ctx context.Context, goal *model.SavingsGoal) error {
	query := `
		INSERT INTO savings_goals (id, user_id, name, target_amount, current_amount, currency, target_date, color, icon, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING created_at, updated_at`

	goal.ID = uuid.New()
	if goal.CurrentAmount.IsZero() {
		goal.CurrentAmount = decimal.Zero
	}
	return r.db.QueryRowxContext(ctx, query,
		goal.ID, goal.UserID, goal.Name, goal.TargetAmount, goal.CurrentAmount,
		goal.Currency, goal.TargetDate, goal.Color, goal.Icon,
	).Scan(&goal.CreatedAt, &goal.UpdatedAt)
}

func (r *SavingsGoalRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SavingsGoal, error) {
	var goal model.SavingsGoal
	query := `SELECT * FROM savings_goals WHERE id = $1`
	err := r.db.GetContext(ctx, &goal, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSavingsGoalNotFound
	}
	return &goal, err
}

func (r *SavingsGoalRepository) List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error) {
	var goals []model.SavingsGoal
	query := `SELECT * FROM savings_goals WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &goals, query, userID)
	return goals, err
}

func (r *SavingsGoalRepository) Update(ctx context.Context, goal *model.SavingsGoal) error {
	query := `
		UPDATE savings_goals 
		SET name = $2, target_amount = $3, current_amount = $4, currency = $5, target_date = $6, color = $7, icon = $8, updated_at = NOW()
		WHERE id = $1 AND user_id = $9
		RETURNING updated_at`
	result := r.db.QueryRowxContext(ctx, query,
		goal.ID, goal.Name, goal.TargetAmount, goal.CurrentAmount,
		goal.Currency, goal.TargetDate, goal.Color, goal.Icon, goal.UserID,
	)
	return result.Scan(&goal.UpdatedAt)
}

func (r *SavingsGoalRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM savings_goals WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSavingsGoalNotFound
	}
	return nil
}

func (r *SavingsGoalRepository) AddContribution(ctx context.Context, id uuid.UUID, userID uuid.UUID, amount decimal.Decimal) error {
	query := `
		UPDATE savings_goals 
		SET current_amount = current_amount + $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID, amount)
	return err
}

func (r *SavingsGoalRepository) GetTotalSavings(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	query := `SELECT COALESCE(SUM(current_amount), 0) FROM savings_goals WHERE user_id = $1`
	err := r.db.GetContext(ctx, &total, query, userID)
	return total, err
}
