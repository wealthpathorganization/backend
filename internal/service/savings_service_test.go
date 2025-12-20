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

// MockSavingsGoalRepo implements SavingsGoalRepositoryInterface for testing
type MockSavingsGoalRepo struct {
	mock.Mock
}

func (m *MockSavingsGoalRepo) Create(ctx context.Context, goal *model.SavingsGoal) error {
	args := m.Called(ctx, goal)
	if goal.ID == uuid.Nil {
		goal.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockSavingsGoalRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.SavingsGoal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SavingsGoal), args.Error(1)
}

func (m *MockSavingsGoalRepo) List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.SavingsGoal), args.Error(1)
}

func (m *MockSavingsGoalRepo) Update(ctx context.Context, goal *model.SavingsGoal) error {
	args := m.Called(ctx, goal)
	return args.Error(0)
}

func (m *MockSavingsGoalRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockSavingsGoalRepo) AddContribution(ctx context.Context, id, userID uuid.UUID, amount decimal.Decimal) error {
	args := m.Called(ctx, id, userID, amount)
	return args.Error(0)
}

// Table-driven tests with parallel execution (following Go rules)
func TestSavingsGoalService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     CreateSavingsGoalInput
		setupMock func(*MockSavingsGoalRepo)
		wantErr   bool
		check     func(*testing.T, *model.SavingsGoal)
	}{
		{
			name: "success with all fields",
			input: CreateSavingsGoalInput{
				Name:         "Emergency Fund",
				TargetAmount: decimal.NewFromFloat(10000),
				Currency:     "USD",
				Color:        "#10B981",
				Icon:         "shield",
			},
			setupMock: func(m *MockSavingsGoalRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.SavingsGoal")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, g *model.SavingsGoal) {
				assert.Equal(t, "Emergency Fund", g.Name)
				assert.Equal(t, "#10B981", g.Color)
			},
		},
		{
			name: "default currency to USD",
			input: CreateSavingsGoalInput{
				Name:         "Vacation",
				TargetAmount: decimal.NewFromFloat(5000),
				Currency:     "",
			},
			setupMock: func(m *MockSavingsGoalRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(g *model.SavingsGoal) bool {
					return g.Currency == "USD"
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, g *model.SavingsGoal) {
				assert.Equal(t, "USD", g.Currency)
			},
		},
		{
			name: "default color",
			input: CreateSavingsGoalInput{
				Name:         "Test",
				TargetAmount: decimal.NewFromFloat(1000),
				Color:        "",
			},
			setupMock: func(m *MockSavingsGoalRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(g *model.SavingsGoal) bool {
					return g.Color == "#3B82F6"
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, g *model.SavingsGoal) {
				assert.Equal(t, "#3B82F6", g.Color)
			},
		},
		{
			name: "default icon",
			input: CreateSavingsGoalInput{
				Name:         "Test",
				TargetAmount: decimal.NewFromFloat(1000),
				Icon:         "",
			},
			setupMock: func(m *MockSavingsGoalRepo) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(g *model.SavingsGoal) bool {
					return g.Icon == "piggy-bank"
				})).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, g *model.SavingsGoal) {
				assert.Equal(t, "piggy-bank", g.Icon)
			},
		},
		{
			name: "repository error",
			input: CreateSavingsGoalInput{
				Name:         "Test",
				TargetAmount: decimal.NewFromFloat(1000),
			},
			setupMock: func(m *MockSavingsGoalRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.SavingsGoal")).Return(errors.New("db error"))
			},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockSavingsGoalRepo)
			service := NewSavingsGoalService(mockRepo)
			tt.setupMock(mockRepo)

			goal, err := service.Create(context.Background(), uuid.New(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, goal)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, goal)
				if tt.check != nil {
					tt.check(t, goal)
				}
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockSavingsGoalRepo, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockSavingsGoalRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(&model.SavingsGoal{ID: id, Name: "Test"}, nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setupMock: func(m *MockSavingsGoalRepo, id uuid.UUID) {
				m.On("GetByID", mock.Anything, id).Return(nil, repository.ErrSavingsGoalNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockSavingsGoalRepo)
			service := NewSavingsGoalService(mockRepo)
			goalID := uuid.New()
			tt.setupMock(mockRepo, goalID)

			goal, err := service.Get(context.Background(), goalID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, goal)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, goal)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalService_List(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockSavingsGoalRepo)
	service := NewSavingsGoalService(mockRepo)
	userID := uuid.New()

	expected := []model.SavingsGoal{
		{ID: uuid.New(), UserID: userID, Name: "Emergency Fund"},
		{ID: uuid.New(), UserID: userID, Name: "Vacation"},
	}

	mockRepo.On("List", mock.Anything, userID).Return(expected, nil)

	goals, err := service.List(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, goals, 2)
	mockRepo.AssertExpectations(t)
}

func TestSavingsGoalService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockSavingsGoalRepo, uuid.UUID, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockSavingsGoalRepo, goalID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, goalID).Return(&model.SavingsGoal{
					ID:     goalID,
					UserID: userID,
					Name:   "Old Name",
				}, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.SavingsGoal")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "not owner",
			setupMock: func(m *MockSavingsGoalRepo, goalID, userID uuid.UUID) {
				otherUserID := uuid.New()
				m.On("GetByID", mock.Anything, goalID).Return(&model.SavingsGoal{
					ID:     goalID,
					UserID: otherUserID,
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "not found",
			setupMock: func(m *MockSavingsGoalRepo, goalID, userID uuid.UUID) {
				m.On("GetByID", mock.Anything, goalID).Return(nil, repository.ErrSavingsGoalNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockSavingsGoalRepo)
			service := NewSavingsGoalService(mockRepo)
			goalID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mockRepo, goalID, userID)

			targetDate := time.Now().AddDate(1, 0, 0)
			goal, err := service.Update(context.Background(), goalID, userID, UpdateSavingsGoalInput{
				Name:       "New Name",
				TargetDate: &targetDate,
			})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, goal)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, goal)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSavingsGoalService_Delete(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockSavingsGoalRepo)
	service := NewSavingsGoalService(mockRepo)
	goalID := uuid.New()
	userID := uuid.New()

	mockRepo.On("Delete", mock.Anything, goalID, userID).Return(nil)

	err := service.Delete(context.Background(), goalID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSavingsGoalService_Contribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*MockSavingsGoalRepo, uuid.UUID, uuid.UUID)
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *MockSavingsGoalRepo, goalID, userID uuid.UUID) {
				m.On("AddContribution", mock.Anything, goalID, userID, decimal.NewFromFloat(500)).Return(nil)
				m.On("GetByID", mock.Anything, goalID).Return(&model.SavingsGoal{
					ID:            goalID,
					UserID:        userID,
					CurrentAmount: decimal.NewFromFloat(2500),
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "contribution error",
			setupMock: func(m *MockSavingsGoalRepo, goalID, userID uuid.UUID) {
				m.On("AddContribution", mock.Anything, goalID, userID, decimal.NewFromFloat(500)).Return(errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockSavingsGoalRepo)
			service := NewSavingsGoalService(mockRepo)
			goalID := uuid.New()
			userID := uuid.New()
			tt.setupMock(mockRepo, goalID, userID)

			goal, err := service.Contribute(context.Background(), goalID, userID, decimal.NewFromFloat(500))

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, goal)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, goal)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
