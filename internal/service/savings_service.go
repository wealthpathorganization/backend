package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
)

// SavingsGoalRepositoryInterface defines the contract for savings goal data access.
// Implementations must be safe for concurrent use.
type SavingsGoalRepositoryInterface interface {
	Create(ctx context.Context, goal *model.SavingsGoal) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.SavingsGoal, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error)
	Update(ctx context.Context, goal *model.SavingsGoal) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	AddContribution(ctx context.Context, id, userID uuid.UUID, amount decimal.Decimal) error
}

// SavingsGoalService handles business logic for savings goals and contributions.
type SavingsGoalService struct {
	repo SavingsGoalRepositoryInterface
}

// NewSavingsGoalService creates a new SavingsGoalService with the given repository.
func NewSavingsGoalService(repo SavingsGoalRepositoryInterface) *SavingsGoalService {
	return &SavingsGoalService{repo: repo}
}

type CreateSavingsGoalInput struct {
	Name         string          `json:"name"`
	TargetAmount decimal.Decimal `json:"targetAmount"`
	Currency     string          `json:"currency"`
	TargetDate   *time.Time      `json:"targetDate"`
	Color        string          `json:"color"`
	Icon         string          `json:"icon"`
}

type UpdateSavingsGoalInput struct {
	Name          string          `json:"name"`
	TargetAmount  decimal.Decimal `json:"targetAmount"`
	CurrentAmount decimal.Decimal `json:"currentAmount"`
	Currency      string          `json:"currency"`
	TargetDate    *time.Time      `json:"targetDate"`
	Color         string          `json:"color"`
	Icon          string          `json:"icon"`
}

type ContributeInput struct {
	Amount decimal.Decimal `json:"amount"`
}

// Create creates a new savings goal for the given user.
// Defaults currency to USD, color to blue, and icon to piggy-bank if not specified.
func (s *SavingsGoalService) Create(ctx context.Context, userID uuid.UUID, input CreateSavingsGoalInput) (*model.SavingsGoal, error) {
	goal := &model.SavingsGoal{
		UserID:       userID,
		Name:         input.Name,
		TargetAmount: input.TargetAmount,
		Currency:     input.Currency,
		TargetDate:   input.TargetDate,
		Color:        input.Color,
		Icon:         input.Icon,
	}

	if goal.Currency == "" {
		goal.Currency = "USD"
	}
	if goal.Color == "" {
		goal.Color = "#3B82F6"
	}
	if goal.Icon == "" {
		goal.Icon = "piggy-bank"
	}

	if err := s.repo.Create(ctx, goal); err != nil {
		return nil, fmt.Errorf("creating savings goal: %w", err)
	}

	return goal, nil
}

// Get retrieves a savings goal by its ID.
func (s *SavingsGoalService) Get(ctx context.Context, id uuid.UUID) (*model.SavingsGoal, error) {
	goal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting savings goal %s: %w", id, err)
	}
	return goal, nil
}

// List retrieves all savings goals for a user.
func (s *SavingsGoalService) List(ctx context.Context, userID uuid.UUID) ([]model.SavingsGoal, error) {
	goals, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing savings goals for user %s: %w", userID, err)
	}
	return goals, nil
}

// Update modifies an existing savings goal.
// Returns ErrSavingsGoalNotFound if the goal does not exist or belongs to another user.
func (s *SavingsGoalService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input UpdateSavingsGoalInput) (*model.SavingsGoal, error) {
	goal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching savings goal %s for update: %w", id, err)
	}

	if goal.UserID != userID {
		return nil, repository.ErrSavingsGoalNotFound
	}

	goal.Name = input.Name
	goal.TargetAmount = input.TargetAmount
	goal.CurrentAmount = input.CurrentAmount
	goal.Currency = input.Currency
	goal.TargetDate = input.TargetDate
	goal.Color = input.Color
	goal.Icon = input.Icon

	if err := s.repo.Update(ctx, goal); err != nil {
		return nil, fmt.Errorf("updating savings goal %s: %w", id, err)
	}

	return goal, nil
}

// Delete removes a savings goal by ID for the given user.
func (s *SavingsGoalService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("deleting savings goal %s: %w", id, err)
	}
	return nil
}

// Contribute adds a contribution amount to a savings goal.
func (s *SavingsGoalService) Contribute(ctx context.Context, id uuid.UUID, userID uuid.UUID, amount decimal.Decimal) (*model.SavingsGoal, error) {
	if err := s.repo.AddContribution(ctx, id, userID, amount); err != nil {
		return nil, fmt.Errorf("adding contribution to savings goal %s: %w", id, err)
	}
	goal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching updated savings goal %s: %w", id, err)
	}
	return goal, nil
}
