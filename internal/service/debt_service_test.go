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

// MockDebtRepo implements DebtRepositoryInterface for testing
type MockDebtRepo struct {
	mock.Mock
}

func (m *MockDebtRepo) Create(ctx context.Context, debt *model.Debt) error {
	args := m.Called(ctx, debt)
	if debt.ID == uuid.Nil {
		debt.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockDebtRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Debt, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Debt), args.Error(1)
}

func (m *MockDebtRepo) List(ctx context.Context, userID uuid.UUID) ([]model.Debt, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Debt), args.Error(1)
}

func (m *MockDebtRepo) Update(ctx context.Context, debt *model.Debt) error {
	args := m.Called(ctx, debt)
	return args.Error(0)
}

func (m *MockDebtRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockDebtRepo) RecordPayment(ctx context.Context, payment *model.DebtPayment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

// Table-driven tests with parallel execution (following Go rules)
func TestDebtService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     CreateDebtInput
		setupMock func(*MockDebtRepo)
		wantErr   bool
		check     func(*testing.T, *model.Debt)
	}{
		{
			name: "success with all fields",
			input: CreateDebtInput{
				Name:           "Mortgage",
				Type:           model.DebtTypeMortgage,
				OriginalAmount: decimal.NewFromFloat(200000),
				CurrentBalance: decimal.NewFromFloat(180000),
				InterestRate:   decimal.NewFromFloat(4.5),
				MinimumPayment: decimal.NewFromFloat(1200),
				Currency:       "USD",
			},
			setupMock: func(m *MockDebtRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.Debt")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, d *model.Debt) {
				assert.Equal(t, "Mortgage", d.Name)
				assert.Equal(t, "USD", d.Currency)
			},
		},
		{
			name: "default currency to USD",
			input: CreateDebtInput{
				Name:           "Credit Card",
				Type:           model.DebtTypeCreditCard,
				OriginalAmount: decimal.NewFromFloat(5000),
				Currency:       "",
			},
			setupMock: func(m *MockDebtRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(d *model.Debt) bool {
					return d.Currency == "USD"
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, d *model.Debt) {
				assert.Equal(t, "USD", d.Currency)
			},
		},
		{
			name: "default current balance to original",
			input: CreateDebtInput{
				Name:           "Auto Loan",
				Type:           model.DebtTypeAutoLoan,
				OriginalAmount: decimal.NewFromFloat(25000),
				CurrentBalance: decimal.Zero,
			},
			setupMock: func(m *MockDebtRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(d *model.Debt) bool {
					return d.CurrentBalance.Equal(d.OriginalAmount)
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, d *model.Debt) {
				assert.True(t, d.CurrentBalance.Equal(d.OriginalAmount))
			},
		},
		{
			name: "repository error",
			input: CreateDebtInput{
				Name:           "Test",
				OriginalAmount: decimal.NewFromFloat(1000),
			},
			setupMock: func(m *MockDebtRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.Debt")).Return(errors.New("db error"))
			},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockDebtRepo)
			service := NewDebtService(mockRepo)
			tt.setupMock(mockRepo)

			debt, err := service.Create(context.Background(), uuid.New(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, debt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, debt)
				if tt.check != nil {
					tt.check(t, debt)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDebtService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockDebtRepo, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockDebtRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(&model.Debt{ID: id, Name: "Test"}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(m *MockDebtRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(nil, repository.ErrDebtNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockDebtRepo)
			service := NewDebtService(mockRepo)
			debtID := uuid.New()
			tt.setupMock(mockRepo, debtID)

			debt, err := service.Get(context.Background(), debtID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, debt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, debt)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDebtService_List(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockDebtRepo)
	service := NewDebtService(mockRepo)
	userID := uuid.New()

	expected := []model.Debt{
		{ID: uuid.New(), UserID: userID, Name: "Mortgage"},
		{ID: uuid.New(), UserID: userID, Name: "Credit Card"},
	}

	mockRepo.On("List", mock.Anything, userID).Return(expected, nil)

	debts, err := service.List(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, debts, 2)
	mockRepo.AssertExpectations(t)
}

func TestDebtService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockDebtRepo, uuid.UUID, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockDebtRepo, debtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, debtID).Return(&model.Debt{
					ID:     debtID,
					UserID: userID,
					Name:   "Old Name",
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.Debt")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "not owner",
			setupMock: func(m *MockDebtRepo, debtID, userID uuid.UUID) {
				otherUserID := uuid.New()
				m.On("GetByID", mock.Anything, debtID).Return(&model.Debt{
					ID:     debtID,
					UserID: otherUserID,
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "not found",
			setupMock: func(m *MockDebtRepo, debtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, debtID).Return(nil, repository.ErrDebtNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockDebtRepo)
			service := NewDebtService(mockRepo)
			debtID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mockRepo, debtID, userID)

			debt, err := service.Update(context.Background(), debtID, userID, UpdateDebtInput{Name: "New Name"})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, debt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, debt)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDebtService_Delete(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockDebtRepo)
	service := NewDebtService(mockRepo)
	debtID := uuid.New()
	userID := uuid.New()

	mockRepo.On("Delete", mock.Anything, debtID, userID).Return(nil)

	err := service.Delete(context.Background(), debtID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDebtService_MakePayment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockDebtRepo, uuid.UUID, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockDebtRepo, debtID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, debtID).Return(&model.Debt{
					ID:             debtID,
					UserID:         userID,
					CurrentBalance: decimal.NewFromFloat(10000),
					InterestRate:   decimal.NewFromFloat(12),
				}, nil).Once()
				m.On("RecordPayment", mock.Anything, mock.AnythingOfType("*model.DebtPayment")).Return(nil)
				m.On("GetByID", mock.Anything, debtID).Return(&model.Debt{
					ID:             debtID,
					UserID:         userID,
					CurrentBalance: decimal.NewFromFloat(9600),
				}, nil).Once()
			},
			wantErr: false,
		},
		{
			name: "not owner",
			setupMock: func(m *MockDebtRepo, debtID, userID uuid.UUID) {
				otherUserID := uuid.New()
				m.On("GetByID", mock.Anything, debtID).Return(&model.Debt{
					ID:     debtID,
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

			mockRepo := new(MockDebtRepo)
			service := NewDebtService(mockRepo)
			debtID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mockRepo, debtID, userID)

			input := MakePaymentInput{
				Amount: decimal.NewFromFloat(500),
				Date:   time.Now(),
			}

			debt, err := service.MakePayment(context.Background(), debtID, userID, input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, debt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, debt)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDebtService_GetPayoffPlan(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockDebtRepo)
	service := NewDebtService(mockRepo)
	debtID := uuid.New()

	mockRepo.On("GetByID", mock.Anything, debtID).Return(&model.Debt{
		ID:             debtID,
		CurrentBalance: decimal.NewFromFloat(10000),
		InterestRate:   decimal.NewFromFloat(12),
		MinimumPayment: decimal.NewFromFloat(200),
	}, nil)

	plan, err := service.GetPayoffPlan(context.Background(), debtID, decimal.NewFromFloat(500))

	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.True(t, plan.MonthlyPayment.Equal(decimal.NewFromFloat(500)))
	mockRepo.AssertExpectations(t)
}

func TestDebtService_GetPayoffPlan_DefaultPayment(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockDebtRepo)
	service := NewDebtService(mockRepo)
	debtID := uuid.New()

	mockRepo.On("GetByID", mock.Anything, debtID).Return(&model.Debt{
		ID:             debtID,
		CurrentBalance: decimal.NewFromFloat(5000),
		InterestRate:   decimal.NewFromFloat(18),
		MinimumPayment: decimal.NewFromFloat(100),
	}, nil)

	plan, err := service.GetPayoffPlan(context.Background(), debtID, decimal.Zero)

	assert.NoError(t, err)
	assert.True(t, plan.MonthlyPayment.Equal(decimal.NewFromFloat(100)))
	mockRepo.AssertExpectations(t)
}

func TestDebtService_CalculateInterest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input InterestCalculatorInput
		check func(*testing.T, *InterestCalculatorResult)
	}{
		{
			name: "zero interest rate",
			input: InterestCalculatorInput{
				Principal:    decimal.NewFromFloat(12000),
				InterestRate: decimal.Zero,
				TermMonths:   12,
			},
			check: func(t *testing.T, r *InterestCalculatorResult) {
				assert.True(t, r.MonthlyPayment.Equal(decimal.NewFromFloat(1000)))
				assert.True(t, r.TotalInterest.Equal(decimal.Zero))
			},
		},
		{
			name: "with interest rate",
			input: InterestCalculatorInput{
				Principal:    decimal.NewFromFloat(10000),
				InterestRate: decimal.NewFromFloat(12),
				TermMonths:   12,
			},
			check: func(t *testing.T, r *InterestCalculatorResult) {
				assert.True(t, r.MonthlyPayment.GreaterThan(decimal.NewFromFloat(800)))
				assert.True(t, r.TotalInterest.GreaterThan(decimal.Zero))
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewDebtService(nil)
			result, err := service.CalculateInterest(tt.input)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			tt.check(t, result)
		})
	}
}

func TestCalculatePayoffPlan(t *testing.T) {
	t.Parallel()

	debt := &model.Debt{
		ID:             uuid.New(),
		CurrentBalance: decimal.NewFromFloat(10000),
		InterestRate:   decimal.NewFromFloat(12),
	}

	plan := calculatePayoffPlan(debt, decimal.NewFromFloat(500))

	assert.NotNil(t, plan)
	assert.True(t, plan.MonthsToPayoff > 0)
	assert.True(t, plan.TotalInterest.GreaterThan(decimal.Zero))
	assert.NotEmpty(t, plan.AmortizationPlan)
}
