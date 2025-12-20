package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/wealthpath/backend/internal/model"
)

func TestNewBudgetRepository(t *testing.T) {
	t.Parallel()

	mockDB, _, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")

	repo := NewBudgetRepository(db)
	assert.NotNil(t, repo)
}

func TestBudgetRepository_Create(t *testing.T) {
	t.Parallel()

	mockDB, mock, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")
	repo := NewBudgetRepository(db)

	ctx := context.Background()
	budget := &model.Budget{
		UserID:    uuid.New(),
		Category:  "Food",
		Amount:    decimal.NewFromFloat(500),
		Currency:  "USD",
		Period:    "monthly",
		StartDate: time.Now(),
	}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now)

	mock.ExpectQuery(`INSERT INTO budgets`).
		WithArgs(sqlmock.AnyArg(), budget.UserID, budget.Category, budget.Amount, budget.Currency, budget.Period, budget.StartDate, nil).
		WillReturnRows(rows)

	err := repo.Create(ctx, budget)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, budget.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBudgetRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock, uuid.UUID)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "category", "amount", "currency", "period", "start_date", "end_date", "created_at", "updated_at"}).
					AddRow(id, uuid.New(), "Food", decimal.NewFromFloat(500), "USD", "monthly", time.Now(), nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT \* FROM budgets WHERE id = \$1`).
					WithArgs(id).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				mock.ExpectQuery(`SELECT \* FROM budgets WHERE id = \$1`).
					WithArgs(id).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: ErrBudgetNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, _ := sqlmock.New()
			defer func() { _ = mockDB.Close() }()
			db := sqlx.NewDb(mockDB, "sqlmock")
			repo := NewBudgetRepository(db)

			ctx := context.Background()
			budgetID := uuid.New()
			tt.setupMock(mock, budgetID)

			budget, err := repo.GetByID(ctx, budgetID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, budget)
				assert.Equal(t, budgetID, budget.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBudgetRepository_List(t *testing.T) {
	t.Parallel()

	mockDB, mock, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")
	repo := NewBudgetRepository(db)

	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "user_id", "category", "amount", "currency", "period", "start_date", "end_date", "created_at", "updated_at"}).
		AddRow(uuid.New(), userID, "Food", decimal.NewFromFloat(500), "USD", "monthly", time.Now(), nil, time.Now(), time.Now()).
		AddRow(uuid.New(), userID, "Transport", decimal.NewFromFloat(200), "USD", "monthly", time.Now(), nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM budgets WHERE user_id = \$1 ORDER BY category`).
		WithArgs(userID).
		WillReturnRows(rows)

	budgets, err := repo.List(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, budgets, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBudgetRepository_Update(t *testing.T) {
	t.Parallel()

	mockDB, mock, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")
	repo := NewBudgetRepository(db)

	ctx := context.Background()
	budget := &model.Budget{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Category:  "Shopping",
		Amount:    decimal.NewFromFloat(600),
		Currency:  "USD",
		Period:    "monthly",
		StartDate: time.Now(),
	}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"updated_at"}).AddRow(now)

	mock.ExpectQuery(`UPDATE budgets`).
		WithArgs(budget.ID, budget.Category, budget.Amount, budget.Currency, budget.Period, budget.StartDate, nil, budget.UserID).
		WillReturnRows(rows)

	err := repo.Update(ctx, budget)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBudgetRepository_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock, uuid.UUID, uuid.UUID)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			setupMock: func(mock sqlmock.Sqlmock, id, userID uuid.UUID) {
				mock.ExpectExec(`DELETE FROM budgets WHERE id = \$1 AND user_id = \$2`).
					WithArgs(id, userID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(mock sqlmock.Sqlmock, id, userID uuid.UUID) {
				mock.ExpectExec(`DELETE FROM budgets WHERE id = \$1 AND user_id = \$2`).
					WithArgs(id, userID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errType: ErrBudgetNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockDB, mock, _ := sqlmock.New()
			defer func() { _ = mockDB.Close() }()
			db := sqlx.NewDb(mockDB, "sqlmock")
			repo := NewBudgetRepository(db)

			ctx := context.Background()
			budgetID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mock, budgetID, userID)

			err := repo.Delete(ctx, budgetID, userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBudgetRepository_GetActiveForUser(t *testing.T) {
	t.Parallel()

	mockDB, mock, _ := sqlmock.New()
	defer func() { _ = mockDB.Close() }()
	db := sqlx.NewDb(mockDB, "sqlmock")
	repo := NewBudgetRepository(db)

	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "user_id", "category", "amount", "currency", "period", "start_date", "end_date", "created_at", "updated_at"}).
		AddRow(uuid.New(), userID, "Food", decimal.NewFromFloat(500), "USD", "monthly", time.Now(), nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM budgets`).
		WithArgs(userID).
		WillReturnRows(rows)

	budgets, err := repo.GetActiveForUser(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, budgets, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestErrBudgetNotFound(t *testing.T) {
	t.Parallel()

	assert.Error(t, ErrBudgetNotFound)
	assert.Equal(t, "budget not found", ErrBudgetNotFound.Error())
}
