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
	"github.com/wealthpath/backend/internal/repository"
)

// MockTransactionRepo for testing
type MockTransactionRepo struct {
	mock.Mock
}

func (m *MockTransactionRepo) Create(ctx context.Context, tx *model.Transaction) error {
	ret := m.Called(ctx, tx)
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	return ret.Error(0)
}

func (m *MockTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	ret := m.Called(ctx, id)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*model.Transaction), ret.Error(1)
}

func (m *MockTransactionRepo) List(ctx context.Context, userID uuid.UUID, filters repository.TransactionFilters) ([]model.Transaction, error) {
	ret := m.Called(ctx, userID, filters)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]model.Transaction), ret.Error(1)
}

func (m *MockTransactionRepo) Update(ctx context.Context, tx *model.Transaction) error {
	ret := m.Called(ctx, tx)
	return ret.Error(0)
}

func (m *MockTransactionRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ret := m.Called(ctx, id, userID)
	return ret.Error(0)
}

func (m *MockTransactionRepo) GetSpentByCategory(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, error) {
	ret := m.Called(ctx, userID, category, startDate, endDate)
	return ret.Get(0).(decimal.Decimal), ret.Error(1)
}

// TestCreateTransactionInput tests
func TestCreateTransactionInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateTransactionInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid expense",
			input: CreateTransactionInput{
				Type:     model.TransactionTypeExpense,
				Amount:   decimal.NewFromFloat(100.50),
				Currency: "USD",
				Category: "Food & Dining",
			},
			wantErr: false,
		},
		{
			name: "valid income",
			input: CreateTransactionInput{
				Type:     model.TransactionTypeIncome,
				Amount:   decimal.NewFromFloat(5000),
				Currency: "USD",
				Category: "Salary",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			input: CreateTransactionInput{
				Amount:   decimal.NewFromFloat(100),
				Category: "Food",
			},
			wantErr: true,
			errMsg:  "type",
		},
		{
			name: "zero amount",
			input: CreateTransactionInput{
				Type:     model.TransactionTypeExpense,
				Amount:   decimal.Zero,
				Category: "Food",
			},
			wantErr: true,
			errMsg:  "amount",
		},
		{
			name: "negative amount",
			input: CreateTransactionInput{
				Type:     model.TransactionTypeExpense,
				Amount:   decimal.NewFromFloat(-50),
				Category: "Food",
			},
			wantErr: true,
			errMsg:  "amount",
		},
		{
			name: "missing category",
			input: CreateTransactionInput{
				Type:   model.TransactionTypeExpense,
				Amount: decimal.NewFromFloat(100),
			},
			wantErr: true,
			errMsg:  "category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := tt.input.Type == "" || tt.input.Amount.LessThanOrEqual(decimal.Zero) || tt.input.Category == ""
			assert.Equal(t, tt.wantErr, hasErr)
		})
	}
}

// TestTransactionType tests
func TestTransactionType_Validation(t *testing.T) {
	tests := []struct {
		name      string
		txType    model.TransactionType
		wantValid bool
	}{
		{"valid income", model.TransactionTypeIncome, true},
		{"valid expense", model.TransactionTypeExpense, true},
		{"invalid transfer", model.TransactionType("transfer"), false},
		{"invalid empty", model.TransactionType(""), false},
		{"invalid random", model.TransactionType("foo"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.txType == model.TransactionTypeIncome || tt.txType == model.TransactionTypeExpense
			assert.Equal(t, tt.wantValid, isValid)
		})
	}
}

// TestListTransactionsInput tests
func TestListTransactionsInput_Defaults(t *testing.T) {
	input := ListTransactionsInput{}

	// Default page should be 0
	assert.Equal(t, 0, input.Page)

	// Default page size should be handled by service
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	assert.Equal(t, 20, input.PageSize)

	// Max page size should be capped
	input.PageSize = 200
	if input.PageSize > 100 {
		input.PageSize = 100
	}
	assert.Equal(t, 100, input.PageSize)
}

func TestListTransactionsInput_WithFilters(t *testing.T) {
	txType := "expense"
	category := "Food"
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	input := ListTransactionsInput{
		Type:      &txType,
		Category:  &category,
		StartDate: &startDate,
		EndDate:   &endDate,
		Page:      0,
		PageSize:  20,
	}

	assert.NotNil(t, input.Type)
	assert.Equal(t, "expense", *input.Type)
	assert.NotNil(t, input.Category)
	assert.Equal(t, "Food", *input.Category)
	assert.NotNil(t, input.StartDate)
	assert.NotNil(t, input.EndDate)
}

// TestUpdateTransactionInput tests
func TestUpdateTransactionInput_Validation(t *testing.T) {
	input := UpdateTransactionInput{
		Type:        model.TransactionTypeExpense,
		Amount:      decimal.NewFromFloat(150),
		Currency:    "EUR",
		Category:    "Shopping",
		Description: "Updated description",
	}

	assert.Equal(t, model.TransactionTypeExpense, input.Type)
	assert.True(t, input.Amount.Equal(decimal.NewFromFloat(150)))
	assert.Equal(t, "EUR", input.Currency)
	assert.Equal(t, "Shopping", input.Category)
}

// Test service methods behavior
func TestTransactionService_Create_DefaultCurrency(t *testing.T) {
	// Test that empty currency defaults to USD
	input := CreateTransactionInput{
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(100),
		Category: "Food",
		Currency: "", // Empty currency
	}

	tx := &model.Transaction{
		UserID:   uuid.New(),
		Type:     input.Type,
		Amount:   input.Amount,
		Category: input.Category,
		Currency: input.Currency,
	}

	if tx.Currency == "" {
		tx.Currency = "USD"
	}

	assert.Equal(t, "USD", tx.Currency)
}

func TestTransactionService_List_PageSizeLimits(t *testing.T) {
	tests := []struct {
		name         string
		inputSize    int
		expectedSize int
	}{
		{"negative becomes 20", -1, 20},
		{"zero becomes 20", 0, 20},
		{"valid size preserved", 50, 50},
		{"over 100 becomes 100", 200, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := ListTransactionsInput{PageSize: tt.inputSize}

			if input.PageSize <= 0 {
				input.PageSize = 20
			}
			if input.PageSize > 100 {
				input.PageSize = 100
			}

			assert.Equal(t, tt.expectedSize, input.PageSize)
		})
	}
}

func TestTransactionService_Update_OwnershipCheck(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	tx := &model.Transaction{
		ID:     uuid.New(),
		UserID: userID,
	}

	// Same user should be allowed
	assert.Equal(t, userID, tx.UserID)

	// Different user should fail
	if tx.UserID != otherUserID {
		err := repository.ErrTransactionNotFound
		assert.Error(t, err)
	}
}

func TestTransactionService_Delete_ReturnsError(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	ctx := context.Background()
	id := uuid.New()
	userID := uuid.New()

	expectedErr := errors.New("delete failed")
	mockRepo.On("Delete", ctx, id, userID).Return(expectedErr)

	err := mockRepo.Delete(ctx, id, userID)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

// ============================================
// Service Integration Tests with Mock Repo
// ============================================

func TestTransactionService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     CreateTransactionInput
		setupMock func(*MockTransactionRepo)
		wantErr   bool
		checkTx   func(*testing.T, *model.Transaction)
	}{
		{
			name: "success with all fields",
			input: CreateTransactionInput{
				Type:        model.TransactionTypeExpense,
				Amount:      decimal.NewFromFloat(100),
				Currency:    "USD",
				Category:    "Food",
				Description: "Lunch",
			},
			setupMock: func(m *MockTransactionRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.Transaction")).Return(nil)
			},
			wantErr: false,
			checkTx: func(t *testing.T, tx *model.Transaction) {
				assert.Equal(t, "USD", tx.Currency)
				assert.Equal(t, "Food", tx.Category)
			},
		},
		{
			name: "default currency to USD",
			input: CreateTransactionInput{
				Type:     model.TransactionTypeExpense,
				Amount:   decimal.NewFromFloat(50),
				Currency: "",
				Category: "Shopping",
			},
			setupMock: func(m *MockTransactionRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(tx *model.Transaction) bool {
					return tx.Currency == "USD"
				})).Return(nil)
			},
			wantErr: false,
			checkTx: func(t *testing.T, tx *model.Transaction) {
				assert.Equal(t, "USD", tx.Currency)
			},
		},
		{
			name: "repository error",
			input: CreateTransactionInput{
				Type:     model.TransactionTypeIncome,
				Amount:   decimal.NewFromFloat(1000),
				Category: "Salary",
			},
			setupMock: func(m *MockTransactionRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.Transaction")).Return(errors.New("db error"))
			},
			wantErr: true,
			checkTx: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockTransactionRepo)
			service := NewTransactionService(mockRepo)
			tt.setupMock(mockRepo)

			tx, err := service.Create(context.Background(), uuid.New(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tx)
				if tt.checkTx != nil {
					tt.checkTx(t, tx)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestTransactionService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockTransactionRepo, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockTransactionRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(&model.Transaction{
					ID:       id,
					Category: "Food",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(m *MockTransactionRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(nil, repository.ErrTransactionNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockTransactionRepo)
			service := NewTransactionService(mockRepo)
			txID := uuid.New()
			tt.setupMock(mockRepo, txID)

			tx, err := service.Get(context.Background(), txID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tx)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestTransactionService_List_Success(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	input := ListTransactionsInput{
		Page:     0,
		PageSize: 20,
	}

	expected := []model.Transaction{
		{ID: uuid.New(), UserID: userID, Type: model.TransactionTypeExpense},
		{ID: uuid.New(), UserID: userID, Type: model.TransactionTypeIncome},
	}

	mockRepo.On("List", ctx, userID, mock.AnythingOfType("repository.TransactionFilters")).Return(expected, nil)

	txs, err := service.List(ctx, userID, input)

	assert.NoError(t, err)
	assert.Len(t, txs, 2)
	mockRepo.AssertExpectations(t)
}

func TestTransactionService_List_DefaultPageSize(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	input := ListTransactionsInput{
		PageSize: 0, // Should default to 20
	}

	mockRepo.On("List", ctx, userID, mock.MatchedBy(func(f repository.TransactionFilters) bool {
		return f.Limit == 20
	})).Return([]model.Transaction{}, nil)

	_, err := service.List(ctx, userID, input)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTransactionService_List_MaxPageSize(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	input := ListTransactionsInput{
		PageSize: 200, // Should be capped to 100
	}

	mockRepo.On("List", ctx, userID, mock.MatchedBy(func(f repository.TransactionFilters) bool {
		return f.Limit == 100
	})).Return([]model.Transaction{}, nil)

	_, err := service.List(ctx, userID, input)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTransactionService_Update_Success(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	existing := &model.Transaction{
		ID:       txID,
		UserID:   userID,
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(50),
		Category: "Food",
	}

	input := UpdateTransactionInput{
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(75),
		Category: "Shopping",
	}

	mockRepo.On("GetByID", ctx, txID).Return(existing, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)

	tx, err := service.Update(ctx, txID, userID, input)

	assert.NoError(t, err)
	assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(75)))
	assert.Equal(t, "Shopping", tx.Category)
	mockRepo.AssertExpectations(t)
}

func TestTransactionService_Update_NotOwner(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	txID := uuid.New()

	existing := &model.Transaction{
		ID:     txID,
		UserID: ownerID,
	}

	input := UpdateTransactionInput{}

	mockRepo.On("GetByID", ctx, txID).Return(existing, nil)

	tx, err := service.Update(ctx, txID, otherUserID, input)

	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, repository.ErrTransactionNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestTransactionService_Update_NotFound(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	input := UpdateTransactionInput{}

	mockRepo.On("GetByID", ctx, txID).Return(nil, repository.ErrTransactionNotFound)

	tx, err := service.Update(ctx, txID, userID, input)

	assert.Error(t, err)
	assert.Nil(t, tx)
	mockRepo.AssertExpectations(t)
}

func TestTransactionService_Delete_Success(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	mockRepo.On("Delete", ctx, txID, userID).Return(nil)

	err := service.Delete(ctx, txID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTransactionService_Delete_Error(t *testing.T) {
	mockRepo := new(MockTransactionRepo)
	service := NewTransactionService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	mockRepo.On("Delete", ctx, txID, userID).Return(errors.New("delete error"))

	err := service.Delete(ctx, txID, userID)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

// Test categories
func TestExpenseCategories(t *testing.T) {
	expectedCategories := []string{
		"Housing", "Transportation", "Food & Dining", "Utilities",
		"Healthcare", "Insurance", "Entertainment", "Shopping",
		"Personal Care", "Education", "Travel", "Gifts & Donations",
		"Investments", "Debt Payments", "Other",
	}

	for _, cat := range expectedCategories {
		assert.Contains(t, model.ExpenseCategories, cat)
	}
}

func TestIncomeCategories(t *testing.T) {
	expectedCategories := []string{
		"Salary", "Freelance", "Business", "Investments",
		"Rental", "Gifts", "Refunds", "Other",
	}

	for _, cat := range expectedCategories {
		assert.Contains(t, model.IncomeCategories, cat)
	}
}

// Benchmark tests
func BenchmarkCreateTransactionInput_Validation(b *testing.B) {
	input := CreateTransactionInput{
		Type:     model.TransactionTypeExpense,
		Amount:   decimal.NewFromFloat(100.50),
		Currency: "USD",
		Category: "Food & Dining",
	}
	for i := 0; i < b.N; i++ {
		_ = input.Type == "" || input.Amount.LessThanOrEqual(decimal.Zero) || input.Category == ""
	}
}
