// Package service implements the business logic layer for the WealthPath application.
// It contains use cases that orchestrate domain operations and enforce business rules.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/pkg/currency"
	"github.com/wealthpath/backend/pkg/datetime"
)

// TransactionRepositoryInterface defines the contract for transaction data access.
// Implementations must be safe for concurrent use.
type TransactionRepositoryInterface interface {
	Create(ctx context.Context, tx *model.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
	List(ctx context.Context, userID uuid.UUID, filters repository.TransactionFilters) ([]model.Transaction, error)
	Update(ctx context.Context, tx *model.Transaction) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// TransactionService handles business logic for financial transactions.
// It enforces validation rules and coordinates repository operations.
type TransactionService struct {
	repo TransactionRepositoryInterface
}

// NewTransactionService creates a new TransactionService with the given repository.
// The repository must not be nil.
func NewTransactionService(repo TransactionRepositoryInterface) *TransactionService {
	return &TransactionService{repo: repo}
}

type CreateTransactionInput struct {
	Type        model.TransactionType `json:"type"`
	Amount      decimal.Decimal       `json:"amount"`
	Currency    string                `json:"currency"`
	Category    string                `json:"category"`
	Description string                `json:"description"`
	Date        datetime.Date         `json:"date"`
}

type UpdateTransactionInput struct {
	Type        model.TransactionType `json:"type"`
	Amount      decimal.Decimal       `json:"amount"`
	Currency    string                `json:"currency"`
	Category    string                `json:"category"`
	Description string                `json:"description"`
	Date        datetime.Date         `json:"date"`
}

type ListTransactionsInput struct {
	Type      *string    `json:"type"`
	Category  *string    `json:"category"`
	StartDate *time.Time `json:"startDate"`
	EndDate   *time.Time `json:"endDate"`
	Page      int        `json:"page"`
	PageSize  int        `json:"pageSize"`
}

// Create validates and persists a new transaction for the given user.
// It sets default currency to USD if not specified and validates the currency code.
func (s *TransactionService) Create(ctx context.Context, userID uuid.UUID, input CreateTransactionInput) (*model.Transaction, error) {
	curr := input.Currency
	if curr == "" {
		curr = string(currency.DefaultCurrency)
	}
	if !currency.IsValid(curr) {
		return nil, fmt.Errorf("invalid currency code: %s", curr)
	}

	tx := &model.Transaction{
		UserID:      userID,
		Type:        input.Type,
		Amount:      input.Amount,
		Currency:    curr,
		Category:    input.Category,
		Description: input.Description,
		Date:        input.Date.Time,
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("creating transaction: %w", err)
	}

	return tx, nil
}

// Get retrieves a transaction by its ID.
// Returns ErrTransactionNotFound if the transaction does not exist.
func (s *TransactionService) Get(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting transaction %s: %w", id, err)
	}
	return tx, nil
}

// List retrieves transactions for a user with optional filters and pagination.
// PageSize is capped at 100 and defaults to 20.
func (s *TransactionService) List(ctx context.Context, userID uuid.UUID, input ListTransactionsInput) ([]model.Transaction, error) {
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}

	filters := repository.TransactionFilters{
		Type:      input.Type,
		Category:  input.Category,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
		Limit:     input.PageSize,
		Offset:    input.Page * input.PageSize,
	}

	txs, err := s.repo.List(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("listing transactions for user %s: %w", userID, err)
	}
	return txs, nil
}

// Update modifies an existing transaction.
// Returns ErrTransactionNotFound if the transaction does not exist or belongs to another user.
func (s *TransactionService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input UpdateTransactionInput) (*model.Transaction, error) {
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching transaction %s for update: %w", id, err)
	}

	if tx.UserID != userID {
		return nil, repository.ErrTransactionNotFound
	}

	curr := input.Currency
	if curr != "" && !currency.IsValid(curr) {
		return nil, fmt.Errorf("invalid currency code: %s", curr)
	}

	tx.Type = input.Type
	tx.Amount = input.Amount
	if curr != "" {
		tx.Currency = curr
	}
	tx.Category = input.Category
	tx.Description = input.Description
	tx.Date = input.Date.Time

	if err := s.repo.Update(ctx, tx); err != nil {
		return nil, fmt.Errorf("updating transaction %s: %w", id, err)
	}

	return tx, nil
}

// Delete removes a transaction by ID for the given user.
// Returns ErrTransactionNotFound if the transaction does not exist or belongs to another user.
func (s *TransactionService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("deleting transaction %s: %w", id, err)
	}
	return nil
}
