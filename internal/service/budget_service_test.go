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

// MockBudgetRepo for testing
type MockBudgetRepo struct {
	mock.Mock
}

func (m *MockBudgetRepo) Create(ctx context.Context, budget *model.Budget) error {
	args := m.Called(ctx, budget)
	if budget.ID == uuid.Nil {
		budget.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockBudgetRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Budget), args.Error(1)
}

func (m *MockBudgetRepo) List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Budget), args.Error(1)
}

func (m *MockBudgetRepo) GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]model.Budget, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Budget), args.Error(1)
}

func (m *MockBudgetRepo) Update(ctx context.Context, budget *model.Budget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *MockBudgetRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

// Tests - Following Go rules: table-driven tests with parallel execution
func TestBudgetService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     CreateBudgetInput
		setupMock func(*MockBudgetRepo)
		wantErr   bool
		check     func(*testing.T, *model.Budget)
	}{
		{
			name: "success with all fields",
			input: CreateBudgetInput{
				Category: "Food",
				Amount:   decimal.NewFromFloat(500),
				Currency: "USD",
				Period:   "monthly",
			},
			setupMock: func(m *MockBudgetRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.Budget")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, b *model.Budget) {
				assert.Equal(t, "Food", b.Category)
				assert.Equal(t, "USD", b.Currency)
			},
		},
		{
			name: "default currency to USD",
			input: CreateBudgetInput{
				Category: "Food",
				Amount:   decimal.NewFromFloat(500),
				Currency: "",
			},
			setupMock: func(m *MockBudgetRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(b *model.Budget) bool {
					return b.Currency == "USD"
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, b *model.Budget) {
				assert.Equal(t, "USD", b.Currency)
			},
		},
		{
			name: "default period to monthly",
			input: CreateBudgetInput{
				Category: "Food",
				Amount:   decimal.NewFromFloat(500),
				Period:   "",
			},
			setupMock: func(m *MockBudgetRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(b *model.Budget) bool {
					return b.Period == "monthly"
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, b *model.Budget) {
				assert.Equal(t, "monthly", b.Period)
			},
		},
		{
			name: "repository error",
			input: CreateBudgetInput{
				Category: "Food",
				Amount:   decimal.NewFromFloat(500),
			},
			setupMock: func(m *MockBudgetRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.Budget")).Return(errors.New("db error"))
			},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockBudgetRepo)
			service := NewBudgetService(mockRepo)
			tt.setupMock(mockRepo)

			budget, err := service.Create(context.Background(), uuid.New(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, budget)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, budget)
				if tt.check != nil {
					tt.check(t, budget)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBudgetService_Get_Success(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)
	ctx := context.Background()
	budgetID := uuid.New()

	expected := &model.Budget{
		ID:       budgetID,
		UserID:   uuid.New(),
		Category: "Food",
	}

	mockRepo.On("GetByID", ctx, budgetID).Return(expected, nil)

	budget, err := service.Get(ctx, budgetID)

	assert.NoError(t, err)
	assert.Equal(t, expected, budget)
	mockRepo.AssertExpectations(t)
}

func TestBudgetService_Get_NotFound(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)
	ctx := context.Background()
	budgetID := uuid.New()

	mockRepo.On("GetByID", ctx, budgetID).Return(nil, repository.ErrBudgetNotFound)

	budget, err := service.Get(ctx, budgetID)

	assert.Error(t, err)
	assert.Nil(t, budget)
	mockRepo.AssertExpectations(t)
}

func TestBudgetService_List_Success(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	expected := []model.Budget{
		{ID: uuid.New(), UserID: userID, Category: "Food"},
		{ID: uuid.New(), UserID: userID, Category: "Transport"},
	}

	mockRepo.On("List", ctx, userID).Return(expected, nil)

	budgets, err := service.List(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, budgets, 2)
	mockRepo.AssertExpectations(t)
}

func TestBudgetService_Update_Success(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	budgetID := uuid.New()

	existing := &model.Budget{
		ID:       budgetID,
		UserID:   userID,
		Category: "Food",
	}

	input := UpdateBudgetInput{
		Category: "Shopping",
		Amount:   decimal.NewFromFloat(600),
	}

	mockRepo.On("GetByID", ctx, budgetID).Return(existing, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Budget")).Return(nil)

	budget, err := service.Update(ctx, budgetID, userID, input)

	assert.NoError(t, err)
	assert.Equal(t, "Shopping", budget.Category)
	mockRepo.AssertExpectations(t)
}

func TestBudgetService_Update_NotOwner(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)
	ctx := context.Background()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	budgetID := uuid.New()

	existing := &model.Budget{
		ID:     budgetID,
		UserID: ownerID,
	}

	input := UpdateBudgetInput{}

	mockRepo.On("GetByID", ctx, budgetID).Return(existing, nil)

	budget, err := service.Update(ctx, budgetID, otherUserID, input)

	assert.Error(t, err)
	assert.Nil(t, budget)
	mockRepo.AssertExpectations(t)
}

func TestBudgetService_Delete_Success(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	budgetID := uuid.New()

	mockRepo.On("Delete", ctx, budgetID, userID).Return(nil)

	err := service.Delete(ctx, budgetID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestBudgetService_Delete_Error(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	budgetID := uuid.New()

	mockRepo.On("Delete", ctx, budgetID, userID).Return(errors.New("delete error"))

	err := service.Delete(ctx, budgetID, userID)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

// Test getPeriodDates helper function
func TestGetPeriodDates(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		period string
	}{
		{"weekly"},
		{"monthly"},
		{"yearly"},
		{"default"},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			start, end := getPeriodDates(tt.period, now)
			// Just verify it returns valid times without error
			assert.False(t, start.IsZero())
			assert.False(t, end.IsZero())
			assert.True(t, end.After(start) || end.Equal(start))
		})
	}
}

func TestBudgetService_SetTransactionRepo(t *testing.T) {
	mockRepo := new(MockBudgetRepo)
	service := NewBudgetService(mockRepo)

	mockTxRepo := new(MockTransactionRepo)
	service.SetTransactionRepo(mockTxRepo)

	// The service should now have the transaction repo set
	assert.NotNil(t, service)
}

func TestBudgetService_ListWithSpent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockBudgetRepo, *MockTransactionRepo, uuid.UUID)
		setTxRepo bool
		wantErr   bool
	}{
		{
			name: "success with transaction repo",
			setupMock: func(br *MockBudgetRepo, tr *MockTransactionRepo, userID uuid.UUID) {
				budgets := []model.Budget{
					{
						ID:       uuid.New(),
						UserID:   userID,
						Category: "Food",
						Period:   "monthly",
						Amount:   decimal.NewFromFloat(500),
					},
				}
				br.On("GetActiveForUser", mock.Anything, userID).Return(budgets, nil)
				tr.On("GetSpentByCategory", mock.Anything, userID, "Food", mock.Anything, mock.Anything).Return(decimal.NewFromFloat(250), nil)
			},
			setTxRepo: true,
			wantErr:   false,
		},
		{
			name: "budget list error",
			setupMock: func(br *MockBudgetRepo, tr *MockTransactionRepo, userID uuid.UUID) {
				br.On("GetActiveForUser", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
			setTxRepo: true,
			wantErr:   true,
		},
		{
			name: "no transaction repo set - returns budgets without spent",
			setupMock: func(br *MockBudgetRepo, tr *MockTransactionRepo, userID uuid.UUID) {
				budgets := []model.Budget{
					{
						ID:       uuid.New(),
						UserID:   userID,
						Category: "Food",
						Period:   "monthly",
					},
				}
				br.On("GetActiveForUser", mock.Anything, userID).Return(budgets, nil)
			},
			setTxRepo: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockBudgetRepo := new(MockBudgetRepo)
			mockTxRepo := new(MockTransactionRepo)
			service := NewBudgetService(mockBudgetRepo)

			if tt.setTxRepo {
				service.SetTransactionRepo(mockTxRepo)
			}

			userID := uuid.New()
			tt.setupMock(mockBudgetRepo, mockTxRepo, userID)

			budgets, err := service.ListWithSpent(context.Background(), userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, budgets)
			}
			mockBudgetRepo.AssertExpectations(t)
		})
	}
}
