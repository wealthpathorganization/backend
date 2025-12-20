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

var ErrDebtNotFound = errors.New("debt not found")

type DebtRepository struct {
	db *sqlx.DB
}

func NewDebtRepository(db *sqlx.DB) *DebtRepository {
	return &DebtRepository{db: db}
}

func (r *DebtRepository) Create(ctx context.Context, debt *model.Debt) error {
	query := `
		INSERT INTO debts (id, user_id, name, type, original_amount, current_balance, interest_rate, minimum_payment, currency, due_day, start_date, expected_payoff, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		RETURNING created_at, updated_at`

	debt.ID = uuid.New()
	return r.db.QueryRowxContext(ctx, query,
		debt.ID, debt.UserID, debt.Name, debt.Type, debt.OriginalAmount, debt.CurrentBalance,
		debt.InterestRate, debt.MinimumPayment, debt.Currency, debt.DueDay, debt.StartDate, debt.ExpectedPayoff,
	).Scan(&debt.CreatedAt, &debt.UpdatedAt)
}

func (r *DebtRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Debt, error) {
	var debt model.Debt
	query := `SELECT * FROM debts WHERE id = $1`
	err := r.db.GetContext(ctx, &debt, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDebtNotFound
	}
	return &debt, err
}

func (r *DebtRepository) List(ctx context.Context, userID uuid.UUID) ([]model.Debt, error) {
	var debts []model.Debt
	query := `SELECT * FROM debts WHERE user_id = $1 ORDER BY interest_rate DESC`
	err := r.db.SelectContext(ctx, &debts, query, userID)
	return debts, err
}

func (r *DebtRepository) Update(ctx context.Context, debt *model.Debt) error {
	query := `
		UPDATE debts 
		SET name = $2, type = $3, original_amount = $4, current_balance = $5, interest_rate = $6, 
			minimum_payment = $7, currency = $8, due_day = $9, start_date = $10, expected_payoff = $11, updated_at = NOW()
		WHERE id = $1 AND user_id = $12
		RETURNING updated_at`
	result := r.db.QueryRowxContext(ctx, query,
		debt.ID, debt.Name, debt.Type, debt.OriginalAmount, debt.CurrentBalance,
		debt.InterestRate, debt.MinimumPayment, debt.Currency, debt.DueDay,
		debt.StartDate, debt.ExpectedPayoff, debt.UserID,
	)
	return result.Scan(&debt.UpdatedAt)
}

func (r *DebtRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM debts WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDebtNotFound
	}
	return nil
}

func (r *DebtRepository) RecordPayment(ctx context.Context, payment *model.DebtPayment) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Record the payment
	paymentQuery := `
		INSERT INTO debt_payments (id, debt_id, amount, principal, interest, date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`

	payment.ID = uuid.New()
	_, err = tx.ExecContext(ctx, paymentQuery,
		payment.ID, payment.DebtID, payment.Amount, payment.Principal, payment.Interest, payment.Date,
	)
	if err != nil {
		return err
	}

	// Update the debt balance
	updateQuery := `UPDATE debts SET current_balance = current_balance - $2, updated_at = NOW() WHERE id = $1`
	_, err = tx.ExecContext(ctx, updateQuery, payment.DebtID, payment.Principal)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *DebtRepository) GetPayments(ctx context.Context, debtID uuid.UUID) ([]model.DebtPayment, error) {
	var payments []model.DebtPayment
	query := `SELECT * FROM debt_payments WHERE debt_id = $1 ORDER BY date DESC`
	err := r.db.SelectContext(ctx, &payments, query, debtID)
	return payments, err
}

func (r *DebtRepository) GetTotalDebt(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	query := `SELECT COALESCE(SUM(current_balance), 0) FROM debts WHERE user_id = $1`
	err := r.db.GetContext(ctx, &total, query, userID)
	return total, err
}
