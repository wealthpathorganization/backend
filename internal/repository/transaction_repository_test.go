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

// Helper to create a mock DB
func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestNewTransactionRepository(t *testing.T) {
	t.Parallel()

	db, _ := newMockDB(t)
	defer func() { _ = db.Close() }()

	repo := NewTransactionRepository(db)
	assert.NotNil(t, repo)
}

func TestTransactionRepository_Create(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()
	repo := NewTransactionRepository(db)

	ctx := context.Background()
	tx := &model.Transaction{
		UserID:      uuid.New(),
		Type:        model.TransactionTypeExpense,
		Amount:      decimal.NewFromFloat(50),
		Currency:    "USD",
		Category:    "Food",
		Description: "Lunch",
		Date:        time.Now(),
	}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now)

	mock.ExpectQuery(`INSERT INTO transactions`).
		WithArgs(sqlmock.AnyArg(), tx.UserID, tx.Type, tx.Amount, tx.Currency, tx.Category, tx.Description, tx.Date).
		WillReturnRows(rows)

	err := repo.Create(ctx, tx)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tx.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_GetByID(t *testing.T) {
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
				rows := sqlmock.NewRows([]string{"id", "user_id", "type", "amount", "currency", "category", "description", "date", "created_at", "updated_at"}).
					AddRow(id, uuid.New(), "expense", decimal.NewFromFloat(50), "USD", "Food", "Lunch", time.Now(), time.Now(), time.Now())
				mock.ExpectQuery(`SELECT \* FROM transactions WHERE id = \$1`).
					WithArgs(id).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				mock.ExpectQuery(`SELECT \* FROM transactions WHERE id = \$1`).
					WithArgs(id).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: ErrTransactionNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newMockDB(t)
			defer func() { _ = db.Close() }()
			repo := NewTransactionRepository(db)

			ctx := context.Background()
			txID := uuid.New()
			tt.setupMock(mock, txID)

			tx, err := repo.GetByID(ctx, txID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tx)
				assert.Equal(t, txID, tx.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTransactionRepository_List(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()
	repo := NewTransactionRepository(db)

	ctx := context.Background()
	userID := uuid.New()
	filters := TransactionFilters{
		Limit:  20,
		Offset: 0,
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "amount", "currency", "category", "description", "date", "created_at", "updated_at"}).
		AddRow(uuid.New(), userID, "expense", decimal.NewFromFloat(50), "USD", "Food", "Lunch", time.Now(), time.Now(), time.Now()).
		AddRow(uuid.New(), userID, "income", decimal.NewFromFloat(5000), "USD", "Salary", "Monthly", time.Now(), time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM transactions`).
		WithArgs(userID, nil, nil, nil, nil, 20, 0).
		WillReturnRows(rows)

	txs, err := repo.List(ctx, userID, filters)

	assert.NoError(t, err)
	assert.Len(t, txs, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_Update(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()
	repo := NewTransactionRepository(db)

	ctx := context.Background()
	tx := &model.Transaction{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Type:        model.TransactionTypeExpense,
		Amount:      decimal.NewFromFloat(75),
		Currency:    "USD",
		Category:    "Shopping",
		Description: "Clothes",
		Date:        time.Now(),
	}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"updated_at"}).AddRow(now)

	mock.ExpectQuery(`UPDATE transactions`).
		WithArgs(tx.ID, tx.Type, tx.Amount, tx.Currency, tx.Category, tx.Description, tx.Date, tx.UserID).
		WillReturnRows(rows)

	err := repo.Update(ctx, tx)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_Delete(t *testing.T) {
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
				mock.ExpectExec(`DELETE FROM transactions WHERE id = \$1 AND user_id = \$2`).
					WithArgs(id, userID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(mock sqlmock.Sqlmock, id, userID uuid.UUID) {
				mock.ExpectExec(`DELETE FROM transactions WHERE id = \$1 AND user_id = \$2`).
					WithArgs(id, userID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errType: ErrTransactionNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newMockDB(t)
			defer func() { _ = db.Close() }()
			repo := NewTransactionRepository(db)

			ctx := context.Background()
			txID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mock, txID, userID)

			err := repo.Delete(ctx, txID, userID)

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

func TestTransactionRepository_GetMonthlyTotals(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()
	repo := NewTransactionRepository(db)

	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"income", "expenses"}).
		AddRow(decimal.NewFromFloat(5000), decimal.NewFromFloat(2000))

	mock.ExpectQuery(`SELECT`).
		WithArgs(userID, 2024, 6).
		WillReturnRows(rows)

	income, expenses, err := repo.GetMonthlyTotals(ctx, userID, 2024, 6)

	assert.NoError(t, err)
	assert.True(t, income.Equal(decimal.NewFromFloat(5000)))
	assert.True(t, expenses.Equal(decimal.NewFromFloat(2000)))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_GetExpensesByCategory(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()
	repo := NewTransactionRepository(db)

	ctx := context.Background()
	userID := uuid.New()
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	rows := sqlmock.NewRows([]string{"category", "total"}).
		AddRow("Food", decimal.NewFromFloat(500)).
		AddRow("Transport", decimal.NewFromFloat(200))

	mock.ExpectQuery(`SELECT category, SUM`).
		WithArgs(userID, startDate, endDate).
		WillReturnRows(rows)

	result, err := repo.GetExpensesByCategory(ctx, userID, startDate, endDate)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.True(t, result["Food"].Equal(decimal.NewFromFloat(500)))
	assert.True(t, result["Transport"].Equal(decimal.NewFromFloat(200)))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_GetSpentByCategory(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()
	repo := NewTransactionRepository(db)

	ctx := context.Background()
	userID := uuid.New()
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	rows := sqlmock.NewRows([]string{"coalesce"}).AddRow(decimal.NewFromFloat(500))

	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(userID, "Food", startDate, endDate).
		WillReturnRows(rows)

	spent, err := repo.GetSpentByCategory(ctx, userID, "Food", startDate, endDate)

	assert.NoError(t, err)
	assert.True(t, spent.Equal(decimal.NewFromFloat(500)))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_GetRecentTransactions(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()
	repo := NewTransactionRepository(db)

	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "amount", "currency", "category", "description", "date", "created_at", "updated_at"}).
		AddRow(uuid.New(), userID, "expense", decimal.NewFromFloat(50), "USD", "Food", "Lunch", time.Now(), time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM transactions WHERE user_id = \$1 ORDER BY date DESC`).
		WithArgs(userID, 5).
		WillReturnRows(rows)

	txs, err := repo.GetRecentTransactions(ctx, userID, 5)

	assert.NoError(t, err)
	assert.Len(t, txs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionFilters_Struct(t *testing.T) {
	t.Parallel()

	txType := "expense"
	category := "Food"
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	filters := TransactionFilters{
		Type:      &txType,
		Category:  &category,
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     20,
		Offset:    0,
	}

	assert.Equal(t, "expense", *filters.Type)
	assert.Equal(t, "Food", *filters.Category)
	assert.Equal(t, 20, filters.Limit)
	assert.Equal(t, 0, filters.Offset)
}

func TestErrTransactionNotFound(t *testing.T) {
	t.Parallel()

	assert.Error(t, ErrTransactionNotFound)
	assert.Equal(t, "transaction not found", ErrTransactionNotFound.Error())
}
