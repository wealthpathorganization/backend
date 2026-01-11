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

// SavingsGoalWithProjection extends SavingsGoal with calculated projection fields.
type SavingsGoalWithProjection struct {
	model.SavingsGoal
	MonthlyContributionRate decimal.Decimal `json:"monthlyContributionRate"`
	EstimatedCompletionDate *time.Time      `json:"estimatedCompletionDate,omitempty"`
	MonthsToCompletion      *int            `json:"monthsToCompletion,omitempty"`
	IsOnTrack               *bool           `json:"isOnTrack,omitempty"` // Only set if target date exists
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

// ListWithProjections retrieves all savings goals with calculated projections.
func (s *SavingsGoalService) ListWithProjections(ctx context.Context, userID uuid.UUID) ([]SavingsGoalWithProjection, error) {
	goals, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing savings goals for user %s: %w", userID, err)
	}

	result := make([]SavingsGoalWithProjection, len(goals))
	for i, goal := range goals {
		result[i] = s.calculateProjection(&goal)
	}
	return result, nil
}

// calculateProjection computes the ETA and monthly rate for a savings goal.
func (s *SavingsGoalService) calculateProjection(goal *model.SavingsGoal) SavingsGoalWithProjection {
	projection := SavingsGoalWithProjection{SavingsGoal: *goal}

	// If goal is completed, no projection needed
	if goal.CurrentAmount.GreaterThanOrEqual(goal.TargetAmount) {
		projection.MonthlyContributionRate = decimal.Zero
		return projection
	}

	// Calculate months since creation
	monthsSinceCreation := monthsBetween(goal.CreatedAt, time.Now())
	if monthsSinceCreation <= 0 {
		monthsSinceCreation = 1 // At least 1 month for new goals
	}

	// Calculate monthly contribution rate based on current progress
	if goal.CurrentAmount.GreaterThan(decimal.Zero) {
		monthlyRate := goal.CurrentAmount.Div(decimal.NewFromInt(int64(monthsSinceCreation)))
		projection.MonthlyContributionRate = monthlyRate.Round(2)

		// Calculate months to completion
		remaining := goal.TargetAmount.Sub(goal.CurrentAmount)
		if monthlyRate.GreaterThan(decimal.Zero) {
			monthsToGo := remaining.Div(monthlyRate).Ceil().IntPart()
			monthsToGoInt := int(monthsToGo)
			projection.MonthsToCompletion = &monthsToGoInt

			// Calculate estimated completion date
			completionDate := time.Now().AddDate(0, monthsToGoInt, 0)
			projection.EstimatedCompletionDate = &completionDate

			// Check if on track (only if target date is set)
			if goal.TargetDate != nil {
				isOnTrack := completionDate.Before(*goal.TargetDate) || completionDate.Equal(*goal.TargetDate)
				projection.IsOnTrack = &isOnTrack
			}
		}
	}

	return projection
}

// monthsBetween calculates the number of months between two dates.
func monthsBetween(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	days := end.Day() - start.Day()

	totalMonths := years*12 + months
	if days < 0 {
		totalMonths--
	}
	if totalMonths < 0 {
		totalMonths = 0
	}
	return totalMonths
}
