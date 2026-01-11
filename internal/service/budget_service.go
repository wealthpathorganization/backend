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

// BudgetRepositoryInterface defines the contract for budget data access.
// Implementations must be safe for concurrent use.
type BudgetRepositoryInterface interface {
	Create(ctx context.Context, budget *model.Budget) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
	GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]model.Budget, error)
	Update(ctx context.Context, budget *model.Budget) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// TransactionRepoForBudget provides transaction data needed for budget calculations.
type TransactionRepoForBudget interface {
	GetSpentByCategory(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, error)
}

// BudgetService handles business logic for budget management.
// It tracks spending against budget limits and calculates remaining amounts.
type BudgetService struct {
	repo            BudgetRepositoryInterface
	transactionRepo TransactionRepoForBudget
}

// NewBudgetService creates a new BudgetService with the given repository.
func NewBudgetService(repo BudgetRepositoryInterface) *BudgetService {
	return &BudgetService{repo: repo}
}

// SetTransactionRepo sets the transaction repository for spent calculations.
func (s *BudgetService) SetTransactionRepo(repo TransactionRepoForBudget) {
	s.transactionRepo = repo
}

type CreateBudgetInput struct {
	Category          string           `json:"category"`
	Amount            decimal.Decimal  `json:"amount"`
	Currency          string           `json:"currency"`
	Period            string           `json:"period"` // monthly, weekly, yearly
	StartDate         time.Time        `json:"startDate"`
	EndDate           *time.Time       `json:"endDate"`
	EnableRollover    bool             `json:"enableRollover"`
	MaxRolloverAmount *decimal.Decimal `json:"maxRolloverAmount,omitempty"`
}

type UpdateBudgetInput struct {
	Category          string           `json:"category"`
	Amount            decimal.Decimal  `json:"amount"`
	Currency          string           `json:"currency"`
	Period            string           `json:"period"`
	StartDate         time.Time        `json:"startDate"`
	EndDate           *time.Time       `json:"endDate"`
	EnableRollover    bool             `json:"enableRollover"`
	MaxRolloverAmount *decimal.Decimal `json:"maxRolloverAmount,omitempty"`
}

// Create creates a new budget for the given user.
// Defaults currency to USD and period to monthly if not specified.
func (s *BudgetService) Create(ctx context.Context, userID uuid.UUID, input CreateBudgetInput) (*model.Budget, error) {
	budget := &model.Budget{
		UserID:            userID,
		Category:          input.Category,
		Amount:            input.Amount,
		Currency:          input.Currency,
		Period:            input.Period,
		StartDate:         input.StartDate,
		EndDate:           input.EndDate,
		EnableRollover:    input.EnableRollover,
		MaxRolloverAmount: input.MaxRolloverAmount,
		RolloverAmount:    decimal.Zero,
	}

	if budget.Currency == "" {
		budget.Currency = "USD"
	}
	if budget.Period == "" {
		budget.Period = "monthly"
	}

	if err := s.repo.Create(ctx, budget); err != nil {
		return nil, fmt.Errorf("creating budget: %w", err)
	}

	return budget, nil
}

// Get retrieves a budget by its ID.
func (s *BudgetService) Get(ctx context.Context, id uuid.UUID) (*model.Budget, error) {
	budget, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting budget %s: %w", id, err)
	}
	return budget, nil
}

// List retrieves all budgets for a user.
func (s *BudgetService) List(ctx context.Context, userID uuid.UUID) ([]model.Budget, error) {
	budgets, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing budgets for user %s: %w", userID, err)
	}
	return budgets, nil
}

// ListWithSpent retrieves active budgets with calculated spending data.
// It calculates spent amount, remaining amount, and percentage used for each budget.
func (s *BudgetService) ListWithSpent(ctx context.Context, userID uuid.UUID) ([]model.BudgetWithSpent, error) {
	budgets, err := s.repo.GetActiveForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting active budgets for user %s: %w", userID, err)
	}

	if s.transactionRepo == nil {
		result := make([]model.BudgetWithSpent, len(budgets))
		for i, b := range budgets {
			result[i] = model.BudgetWithSpent{Budget: b}
		}
		return result, nil
	}

	result := make([]model.BudgetWithSpent, len(budgets))
	now := time.Now()

	for i, budget := range budgets {
		startDate, endDate := getPeriodDates(budget.Period, now)

		spent, err := s.transactionRepo.GetSpentByCategory(ctx, userID, budget.Category, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("calculating spent for budget %s: %w", budget.ID, err)
		}

		// Effective budget includes rollover amount
		effectiveBudget := budget.Amount.Add(budget.RolloverAmount)
		remaining := effectiveBudget.Sub(spent)
		percentage := float64(0)
		if !effectiveBudget.IsZero() {
			percentage = spent.Div(effectiveBudget).Mul(decimal.NewFromInt(100)).InexactFloat64()
		}

		result[i] = model.BudgetWithSpent{
			Budget:     budget,
			Spent:      spent,
			Remaining:  remaining,
			Percentage: percentage,
		}
	}

	return result, nil
}

// Update modifies an existing budget.
// Returns ErrBudgetNotFound if the budget does not exist or belongs to another user.
func (s *BudgetService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input UpdateBudgetInput) (*model.Budget, error) {
	budget, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching budget %s for update: %w", id, err)
	}

	if budget.UserID != userID {
		return nil, repository.ErrBudgetNotFound
	}

	budget.Category = input.Category
	budget.Amount = input.Amount
	budget.Currency = input.Currency
	budget.Period = input.Period
	budget.StartDate = input.StartDate
	budget.EndDate = input.EndDate
	budget.EnableRollover = input.EnableRollover
	budget.MaxRolloverAmount = input.MaxRolloverAmount

	if err := s.repo.Update(ctx, budget); err != nil {
		return nil, fmt.Errorf("updating budget %s: %w", id, err)
	}

	return budget, nil
}

// Delete removes a budget by ID for the given user.
func (s *BudgetService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("deleting budget %s: %w", id, err)
	}
	return nil
}

// getPeriodDates calculates the start and end dates for a budget period.
func getPeriodDates(period string, now time.Time) (start, end time.Time) {
	switch period {
	case "weekly":
		weekday := int(now.Weekday())
		start = now.AddDate(0, 0, -weekday)
		end = start.AddDate(0, 0, 7).Add(-time.Second)
	case "yearly":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
	default: // monthly
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0).Add(-time.Second)
	}
	return start, end
}
