package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
)

// Service-level errors for recurring transactions.
var (
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrInvalidType       = errors.New("type must be 'income' or 'expense'")
	ErrInvalidFrequency  = errors.New("invalid frequency")
	ErrRecurringNotFound = errors.New("recurring transaction not found")
)

// RecurringRepositoryInterface defines the contract for recurring transaction data access.
// Implementations must be safe for concurrent use.
type RecurringRepositoryInterface interface {
	Create(ctx context.Context, rt *model.RecurringTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.RecurringTransaction, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error)
	Update(ctx context.Context, rt *model.RecurringTransaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUpcoming(ctx context.Context, userID uuid.UUID, limit int) ([]model.UpcomingBill, error)
	GetDueTransactions(ctx context.Context, now time.Time) ([]model.RecurringTransaction, error)
	UpdateLastGenerated(ctx context.Context, id uuid.UUID, lastGenerated, nextOccurrence time.Time) error
}

// TransactionCreator provides transaction creation capability for recurring processing.
type TransactionCreator interface {
	Create(ctx context.Context, tx *model.Transaction) error
}

// RecurringService handles business logic for recurring transactions and bill scheduling.
type RecurringService struct {
	recurringRepo   RecurringRepositoryInterface
	transactionRepo TransactionCreator
}

// NewRecurringService creates a new RecurringService with the given repositories.
func NewRecurringService(recurringRepo RecurringRepositoryInterface, transactionRepo TransactionCreator) *RecurringService {
	return &RecurringService{
		recurringRepo:   recurringRepo,
		transactionRepo: transactionRepo,
	}
}

type CreateRecurringInput struct {
	Type        model.TransactionType    `json:"type"`
	Amount      decimal.Decimal          `json:"amount"`
	Currency    string                   `json:"currency"`
	Category    string                   `json:"category"`
	Description string                   `json:"description"`
	Frequency   model.RecurringFrequency `json:"frequency"`
	StartDate   time.Time                `json:"startDate"`
	EndDate     *time.Time               `json:"endDate"`
}

type UpdateRecurringInput struct {
	Type        *model.TransactionType    `json:"type"`
	Amount      *decimal.Decimal          `json:"amount"`
	Currency    *string                   `json:"currency"`
	Category    *string                   `json:"category"`
	Description *string                   `json:"description"`
	Frequency   *model.RecurringFrequency `json:"frequency"`
	StartDate   *time.Time                `json:"startDate"`
	EndDate     *time.Time                `json:"endDate"`
	IsActive    *bool                     `json:"isActive"`
}

// Create creates a new recurring transaction for the given user.
// Validates amount, type, and frequency before creation.
func (s *RecurringService) Create(ctx context.Context, userID uuid.UUID, input CreateRecurringInput) (*model.RecurringTransaction, error) {
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidAmount
	}

	if input.Type != model.TransactionTypeIncome && input.Type != model.TransactionTypeExpense {
		return nil, ErrInvalidType
	}

	if !isValidFrequency(input.Frequency) {
		return nil, ErrInvalidFrequency
	}

	rt := &model.RecurringTransaction{
		UserID:         userID,
		Type:           input.Type,
		Amount:         input.Amount,
		Currency:       input.Currency,
		Category:       input.Category,
		Description:    input.Description,
		Frequency:      input.Frequency,
		StartDate:      input.StartDate,
		EndDate:        input.EndDate,
		NextOccurrence: input.StartDate,
		IsActive:       true,
	}

	if rt.Currency == "" {
		rt.Currency = "USD"
	}

	if err := s.recurringRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("creating recurring transaction: %w", err)
	}

	return rt, nil
}

// GetByUserID retrieves all recurring transactions for a user.
func (s *RecurringService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.RecurringTransaction, error) {
	rts, err := s.recurringRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing recurring transactions for user %s: %w", userID, err)
	}
	return rts, nil
}

// GetByID retrieves a recurring transaction by ID, ensuring it belongs to the user.
func (s *RecurringService) GetByID(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error) {
	rt, err := s.recurringRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting recurring transaction %s: %w", id, err)
	}
	if rt.UserID != userID {
		return nil, ErrRecurringNotFound
	}
	return rt, nil
}

// Update modifies an existing recurring transaction.
// Returns ErrRecurringNotFound if the transaction does not exist or belongs to another user.
func (s *RecurringService) Update(ctx context.Context, userID, id uuid.UUID, input UpdateRecurringInput) (*model.RecurringTransaction, error) {
	rt, err := s.recurringRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching recurring transaction %s for update: %w", id, err)
	}
	if rt.UserID != userID {
		return nil, ErrRecurringNotFound
	}

	if input.Type != nil {
		rt.Type = *input.Type
	}
	if input.Amount != nil {
		if input.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, ErrInvalidAmount
		}
		rt.Amount = *input.Amount
	}
	if input.Currency != nil {
		rt.Currency = *input.Currency
	}
	if input.Category != nil {
		rt.Category = *input.Category
	}
	if input.Description != nil {
		rt.Description = *input.Description
	}
	if input.Frequency != nil {
		if !isValidFrequency(*input.Frequency) {
			return nil, ErrInvalidFrequency
		}
		rt.Frequency = *input.Frequency
		rt.NextOccurrence = calculateNextOccurrence(rt.StartDate, *input.Frequency)
	}
	if input.StartDate != nil {
		rt.StartDate = *input.StartDate
		rt.NextOccurrence = calculateNextOccurrence(*input.StartDate, rt.Frequency)
	}
	if input.EndDate != nil {
		rt.EndDate = input.EndDate
	}
	if input.IsActive != nil {
		rt.IsActive = *input.IsActive
	}

	if err := s.recurringRepo.Update(ctx, rt); err != nil {
		return nil, fmt.Errorf("updating recurring transaction %s: %w", id, err)
	}

	return rt, nil
}

// Delete removes a recurring transaction by ID for the given user.
func (s *RecurringService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	rt, err := s.recurringRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching recurring transaction %s for deletion: %w", id, err)
	}
	if rt.UserID != userID {
		return ErrRecurringNotFound
	}
	if err := s.recurringRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting recurring transaction %s: %w", id, err)
	}
	return nil
}

// Pause deactivates a recurring transaction.
func (s *RecurringService) Pause(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error) {
	isActive := false
	return s.Update(ctx, userID, id, UpdateRecurringInput{IsActive: &isActive})
}

// Resume reactivates a paused recurring transaction.
func (s *RecurringService) Resume(ctx context.Context, userID, id uuid.UUID) (*model.RecurringTransaction, error) {
	isActive := true
	return s.Update(ctx, userID, id, UpdateRecurringInput{IsActive: &isActive})
}

// GetUpcoming retrieves upcoming bills for a user, limited to the specified count.
func (s *RecurringService) GetUpcoming(ctx context.Context, userID uuid.UUID, limit int) ([]model.UpcomingBill, error) {
	if limit <= 0 {
		limit = 5
	}
	bills, err := s.recurringRepo.GetUpcoming(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("getting upcoming bills for user %s: %w", userID, err)
	}
	return bills, nil
}

// ProcessDueTransactions generates transactions for all due recurring items.
// This should be called by a cron job. Returns the count of processed items.
func (s *RecurringService) ProcessDueTransactions(ctx context.Context) (int, error) {
	now := time.Now()
	dueItems, err := s.recurringRepo.GetDueTransactions(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("fetching due transactions: %w", err)
	}

	count := 0
	for _, rt := range dueItems {
		tx := &model.Transaction{
			UserID:      rt.UserID,
			Type:        rt.Type,
			Amount:      rt.Amount,
			Currency:    rt.Currency,
			Category:    rt.Category,
			Description: rt.Description + " (recurring)",
			Date:        rt.NextOccurrence,
		}

		if err := s.transactionRepo.Create(ctx, tx); err != nil {
			continue // Log error but continue processing
		}

		nextOccurrence := calculateNextOccurrence(rt.NextOccurrence, rt.Frequency)

		if err := s.recurringRepo.UpdateLastGenerated(ctx, rt.ID, now, nextOccurrence); err != nil {
			continue
		}

		count++
	}

	return count, nil
}

// isValidFrequency checks if the given frequency is a supported value.
func isValidFrequency(f model.RecurringFrequency) bool {
	switch f {
	case model.FrequencyDaily, model.FrequencyWeekly, model.FrequencyBiweekly,
		model.FrequencyMonthly, model.FrequencyYearly:
		return true
	}
	return false
}

// calculateNextOccurrence computes the next occurrence date based on frequency.
func calculateNextOccurrence(from time.Time, frequency model.RecurringFrequency) time.Time {
	switch frequency {
	case model.FrequencyDaily:
		return from.AddDate(0, 0, 1)
	case model.FrequencyWeekly:
		return from.AddDate(0, 0, 7)
	case model.FrequencyBiweekly:
		return from.AddDate(0, 0, 14)
	case model.FrequencyMonthly:
		return from.AddDate(0, 1, 0)
	case model.FrequencyYearly:
		return from.AddDate(1, 0, 0)
	default:
		return from.AddDate(0, 1, 0) // Default to monthly
	}
}
