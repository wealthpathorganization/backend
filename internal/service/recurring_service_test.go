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
)

// MockRecurringRepo implements RecurringRepositoryInterface for testing
type MockRecurringRepo struct {
	mock.Mock
}

func (m *MockRecurringRepo) Create(ctx context.Context, rt *model.RecurringTransaction) error {
	args := m.Called(ctx, rt)
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockRecurringRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.RecurringTransaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringRepo) Update(ctx context.Context, rt *model.RecurringTransaction) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRecurringRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRecurringRepo) GetUpcoming(ctx context.Context, userID uuid.UUID, limit int) ([]model.UpcomingBill, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UpcomingBill), args.Error(1)
}

func (m *MockRecurringRepo) GetDueTransactions(ctx context.Context, now time.Time) ([]model.RecurringTransaction, error) {
	args := m.Called(ctx, now)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.RecurringTransaction), args.Error(1)
}

func (m *MockRecurringRepo) UpdateLastGenerated(ctx context.Context, id uuid.UUID, lastGenerated, nextOccurrence time.Time) error {
	args := m.Called(ctx, id, lastGenerated, nextOccurrence)
	return args.Error(0)
}

// MockTransactionCreator implements TransactionCreator for testing
type MockTransactionCreator struct {
	mock.Mock
}

func (m *MockTransactionCreator) Create(ctx context.Context, tx *model.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// Table-driven tests with parallel execution (following Go rules)
func TestRecurringService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     CreateRecurringInput
		setupMock func(*MockRecurringRepo)
		wantErr   bool
		check     func(*testing.T, *model.RecurringTransaction)
	}{
		{
			name: "success",
			input: CreateRecurringInput{
				Type:        model.TransactionTypeExpense,
				Amount:      decimal.NewFromFloat(100),
				Currency:    "USD",
				Category:    "Utilities",
				Description: "Electricity",
				Frequency:   model.FrequencyMonthly,
				StartDate:   time.Now(),
			},
			setupMock: func(m *MockRecurringRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, rt *model.RecurringTransaction) {
				assert.Equal(t, "Electricity", rt.Description)
				assert.True(t, rt.IsActive)
			},
		},
		{
			name: "default currency to USD",
			input: CreateRecurringInput{
				Type:      model.TransactionTypeExpense,
				Amount:    decimal.NewFromFloat(50),
				Currency:  "",
				Category:  "Utilities",
				Frequency: model.FrequencyMonthly,
				StartDate: time.Now(),
			},
			setupMock: func(m *MockRecurringRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(rt *model.RecurringTransaction) bool {
					return rt.Currency == "USD"
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, rt *model.RecurringTransaction) {
				assert.Equal(t, "USD", rt.Currency)
			},
		},
		{
			name: "invalid amount",
			input: CreateRecurringInput{
				Type:      model.TransactionTypeExpense,
				Amount:    decimal.NewFromFloat(-100),
				Category:  "Utilities",
				Frequency: model.FrequencyMonthly,
			},
			setupMock: func(m *MockRecurringRepo) {},
			wantErr:   true,
			check:     nil,
		},
		{
			name: "invalid type",
			input: CreateRecurringInput{
				Type:      model.TransactionType("invalid"),
				Amount:    decimal.NewFromFloat(100),
				Category:  "Utilities",
				Frequency: model.FrequencyMonthly,
			},
			setupMock: func(m *MockRecurringRepo) {},
			wantErr:   true,
			check:     nil,
		},
		{
			name: "invalid frequency",
			input: CreateRecurringInput{
				Type:      model.TransactionTypeExpense,
				Amount:    decimal.NewFromFloat(100),
				Category:  "Utilities",
				Frequency: model.RecurringFrequency("invalid"),
			},
			setupMock: func(m *MockRecurringRepo) {},
			wantErr:   true,
			check:     nil,
		},
		{
			name: "repository error",
			input: CreateRecurringInput{
				Type:      model.TransactionTypeExpense,
				Amount:    decimal.NewFromFloat(100),
				Category:  "Utilities",
				Frequency: model.FrequencyMonthly,
				StartDate: time.Now(),
			},
			setupMock: func(m *MockRecurringRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(errors.New("db error"))
			},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRecurringRepo)
			mockTxRepo := new(MockTransactionCreator)
			service := NewRecurringService(mockRepo, mockTxRepo)
			tt.setupMock(mockRepo)

			rt, err := service.Create(context.Background(), uuid.New(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rt)
				if tt.check != nil {
					tt.check(t, rt)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRecurringService_GetByUserID(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockRecurringRepo)
	mockTxRepo := new(MockTransactionCreator)
	service := NewRecurringService(mockRepo, mockTxRepo)
	userID := uuid.New()

	expected := []model.RecurringTransaction{
		{ID: uuid.New(), UserID: userID, Description: "Electricity"},
		{ID: uuid.New(), UserID: userID, Description: "Internet"},
	}

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(expected, nil)

	rts, err := service.GetByUserID(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, rts, 2)
	mockRepo.AssertExpectations(t)
}

func TestRecurringService_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockRecurringRepo, uuid.UUID, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:     rtID,
					UserID: userID,
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "not owner",
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				otherUserID := uuid.New()
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:     rtID,
					UserID: otherUserID,
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "not found",
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRecurringRepo)
			mockTxRepo := new(MockTransactionCreator)
			service := NewRecurringService(mockRepo, mockTxRepo)
			rtID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mockRepo, rtID, userID)

			rt, err := service.GetByID(context.Background(), userID, rtID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rt)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRecurringService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     UpdateRecurringInput
		setupMock func(*MockRecurringRepo, uuid.UUID, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success - update amount",
			input: func() UpdateRecurringInput {
				amount := decimal.NewFromFloat(150)
				return UpdateRecurringInput{Amount: &amount}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Amount:    decimal.NewFromFloat(100),
					Frequency: model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update type",
			input: func() UpdateRecurringInput {
				txType := model.TransactionTypeIncome
				return UpdateRecurringInput{Type: &txType}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Type:      model.TransactionTypeExpense,
					Frequency: model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update currency",
			input: func() UpdateRecurringInput {
				currency := "EUR"
				return UpdateRecurringInput{Currency: &currency}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Currency:  "USD",
					Frequency: model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update category",
			input: func() UpdateRecurringInput {
				category := "Shopping"
				return UpdateRecurringInput{Category: &category}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Category:  "Utilities",
					Frequency: model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update description",
			input: func() UpdateRecurringInput {
				desc := "Updated description"
				return UpdateRecurringInput{Description: &desc}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:          rtID,
					UserID:      userID,
					Description: "Old description",
					Frequency:   model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update frequency",
			input: func() UpdateRecurringInput {
				freq := model.FrequencyWeekly
				return UpdateRecurringInput{Frequency: &freq}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Frequency: model.FrequencyMonthly,
					StartDate: time.Now(),
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update start date",
			input: func() UpdateRecurringInput {
				startDate := time.Now().AddDate(0, 1, 0)
				return UpdateRecurringInput{StartDate: &startDate}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Frequency: model.FrequencyMonthly,
					StartDate: time.Now(),
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update end date",
			input: func() UpdateRecurringInput {
				endDate := time.Now().AddDate(1, 0, 0)
				return UpdateRecurringInput{EndDate: &endDate}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Frequency: model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update isActive",
			input: func() UpdateRecurringInput {
				isActive := false
				return UpdateRecurringInput{IsActive: &isActive}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					IsActive:  true,
					Frequency: model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "invalid amount",
			input: func() UpdateRecurringInput {
				amount := decimal.NewFromFloat(-50)
				return UpdateRecurringInput{Amount: &amount}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:     rtID,
					UserID: userID,
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "invalid frequency",
			input: func() UpdateRecurringInput {
				freq := model.RecurringFrequency("invalid")
				return UpdateRecurringInput{Frequency: &freq}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Frequency: model.FrequencyMonthly,
				}, nil)
			},
			wantErr: true,
		},
		{
			name:  "not owner",
			input: UpdateRecurringInput{},
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				otherUserID := uuid.New()
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:     rtID,
					UserID: otherUserID,
				}, nil)
			},
			wantErr: true,
		},
		{
			name:  "get error",
			input: UpdateRecurringInput{},
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "update error",
			input: func() UpdateRecurringInput {
				amount := decimal.NewFromFloat(150)
				return UpdateRecurringInput{Amount: &amount}
			}(),
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:        rtID,
					UserID:    userID,
					Frequency: model.FrequencyMonthly,
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.RecurringTransaction")).Return(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRecurringRepo)
			mockTxRepo := new(MockTransactionCreator)
			service := NewRecurringService(mockRepo, mockTxRepo)
			rtID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mockRepo, rtID, userID)

			rt, err := service.Update(context.Background(), userID, rtID, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rt)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRecurringService_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockRecurringRepo, uuid.UUID, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:     rtID,
					UserID: userID,
				}, nil)
				m.On("Delete", mock.Anything, rtID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "not owner",
			setupMock: func(m *MockRecurringRepo, rtID, userID uuid.UUID) {
				otherUserID := uuid.New()
				m.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
					ID:     rtID,
					UserID: otherUserID,
				}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRecurringRepo)
			mockTxRepo := new(MockTransactionCreator)
			service := NewRecurringService(mockRepo, mockTxRepo)
			rtID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mockRepo, rtID, userID)

			err := service.Delete(context.Background(), userID, rtID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRecurringService_Pause(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockRecurringRepo)
	mockTxRepo := new(MockTransactionCreator)
	service := NewRecurringService(mockRepo, mockTxRepo)
	rtID := uuid.New()
	userID := uuid.New()

	mockRepo.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
		ID:       rtID,
		UserID:   userID,
		IsActive: true,
	}, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(rt *model.RecurringTransaction) bool {
		return !rt.IsActive
	})).Return(nil)

	rt, err := service.Pause(context.Background(), userID, rtID)

	assert.NoError(t, err)
	assert.False(t, rt.IsActive)
	mockRepo.AssertExpectations(t)
}

func TestRecurringService_Resume(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockRecurringRepo)
	mockTxRepo := new(MockTransactionCreator)
	service := NewRecurringService(mockRepo, mockTxRepo)
	rtID := uuid.New()
	userID := uuid.New()

	mockRepo.On("GetByID", mock.Anything, rtID).Return(&model.RecurringTransaction{
		ID:       rtID,
		UserID:   userID,
		IsActive: false,
	}, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(rt *model.RecurringTransaction) bool {
		return rt.IsActive
	})).Return(nil)

	rt, err := service.Resume(context.Background(), userID, rtID)

	assert.NoError(t, err)
	assert.True(t, rt.IsActive)
	mockRepo.AssertExpectations(t)
}

func TestRecurringService_GetUpcoming(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockRecurringRepo)
	mockTxRepo := new(MockTransactionCreator)
	service := NewRecurringService(mockRepo, mockTxRepo)
	userID := uuid.New()

	expected := []model.UpcomingBill{
		{ID: uuid.New(), Description: "Electricity"},
		{ID: uuid.New(), Description: "Internet"},
	}

	mockRepo.On("GetUpcoming", mock.Anything, userID, 5).Return(expected, nil)

	bills, err := service.GetUpcoming(context.Background(), userID, 5)

	assert.NoError(t, err)
	assert.Len(t, bills, 2)
	mockRepo.AssertExpectations(t)
}

func TestRecurringService_GetUpcoming_DefaultLimit(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockRecurringRepo)
	mockTxRepo := new(MockTransactionCreator)
	service := NewRecurringService(mockRepo, mockTxRepo)
	userID := uuid.New()

	mockRepo.On("GetUpcoming", mock.Anything, userID, 5).Return([]model.UpcomingBill{}, nil)

	_, err := service.GetUpcoming(context.Background(), userID, 0)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRecurringService_ProcessDueTransactions(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockRecurringRepo)
	mockTxRepo := new(MockTransactionCreator)
	service := NewRecurringService(mockRepo, mockTxRepo)

	dueItems := []model.RecurringTransaction{
		{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			Type:           model.TransactionTypeExpense,
			Amount:         decimal.NewFromFloat(100),
			Currency:       "USD",
			Category:       "Utilities",
			Description:    "Electricity",
			Frequency:      model.FrequencyMonthly,
			NextOccurrence: time.Now(),
		},
	}

	mockRepo.On("GetDueTransactions", mock.Anything, mock.Anything).Return(dueItems, nil)
	mockTxRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Transaction")).Return(nil)
	mockRepo.On("UpdateLastGenerated", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	count, err := service.ProcessDueTransactions(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	mockRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestRecurringService_ProcessDueTransactions_Error(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockRecurringRepo)
	mockTxRepo := new(MockTransactionCreator)
	service := NewRecurringService(mockRepo, mockTxRepo)

	mockRepo.On("GetDueTransactions", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	count, err := service.ProcessDueTransactions(context.Background())

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	mockRepo.AssertExpectations(t)
}

// Test helper functions
func TestIsValidFrequency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		frequency model.RecurringFrequency
		want      bool
	}{
		{model.FrequencyDaily, true},
		{model.FrequencyWeekly, true},
		{model.FrequencyBiweekly, true},
		{model.FrequencyMonthly, true},
		{model.FrequencyYearly, true},
		{model.RecurringFrequency("invalid"), false},
		{model.RecurringFrequency(""), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.frequency), func(t *testing.T) {
			t.Parallel()
			result := isValidFrequency(tt.frequency)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCalculateNextOccurrence(t *testing.T) {
	t.Parallel()

	baseDate := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		frequency model.RecurringFrequency
		expected  time.Time
	}{
		{model.FrequencyDaily, baseDate.AddDate(0, 0, 1)},
		{model.FrequencyWeekly, baseDate.AddDate(0, 0, 7)},
		{model.FrequencyBiweekly, baseDate.AddDate(0, 0, 14)},
		{model.FrequencyMonthly, baseDate.AddDate(0, 1, 0)},
		{model.FrequencyYearly, baseDate.AddDate(1, 0, 0)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.frequency), func(t *testing.T) {
			t.Parallel()
			result := calculateNextOccurrence(baseDate, tt.frequency)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateNextOccurrence_UnknownFrequency(t *testing.T) {
	t.Parallel()

	baseDate := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	result := calculateNextOccurrence(baseDate, model.RecurringFrequency("unknown"))

	// Unknown frequency defaults to monthly
	assert.True(t, result.After(baseDate))
}
