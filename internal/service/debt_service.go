package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
)

// DebtRepositoryInterface defines the contract for debt data access.
// Implementations must be safe for concurrent use.
type DebtRepositoryInterface interface {
	Create(ctx context.Context, debt *model.Debt) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Debt, error)
	List(ctx context.Context, userID uuid.UUID) ([]model.Debt, error)
	Update(ctx context.Context, debt *model.Debt) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	RecordPayment(ctx context.Context, payment *model.DebtPayment) error
}

// DebtService handles business logic for debt management and payoff calculations.
type DebtService struct {
	repo DebtRepositoryInterface
}

// NewDebtService creates a new DebtService with the given repository.
func NewDebtService(repo DebtRepositoryInterface) *DebtService {
	return &DebtService{repo: repo}
}

type CreateDebtInput struct {
	Name           string          `json:"name"`
	Type           model.DebtType  `json:"type"`
	OriginalAmount decimal.Decimal `json:"originalAmount"`
	CurrentBalance decimal.Decimal `json:"currentBalance"`
	InterestRate   decimal.Decimal `json:"interestRate"` // APR as percentage (e.g., 5.5 for 5.5%)
	MinimumPayment decimal.Decimal `json:"minimumPayment"`
	Currency       string          `json:"currency"`
	DueDay         int             `json:"dueDay"`
	StartDate      time.Time       `json:"startDate"`
}

type UpdateDebtInput struct {
	Name           string          `json:"name"`
	Type           model.DebtType  `json:"type"`
	OriginalAmount decimal.Decimal `json:"originalAmount"`
	CurrentBalance decimal.Decimal `json:"currentBalance"`
	InterestRate   decimal.Decimal `json:"interestRate"`
	MinimumPayment decimal.Decimal `json:"minimumPayment"`
	Currency       string          `json:"currency"`
	DueDay         int             `json:"dueDay"`
	StartDate      time.Time       `json:"startDate"`
}

type MakePaymentInput struct {
	Amount decimal.Decimal `json:"amount"`
	Date   time.Time       `json:"date"`
}

type InterestCalculatorInput struct {
	Principal    decimal.Decimal `json:"principal"`
	InterestRate decimal.Decimal `json:"interestRate"` // APR as percentage
	TermMonths   int             `json:"termMonths"`
	PaymentType  string          `json:"paymentType"` // "fixed" or "minimum"
}

type InterestCalculatorResult struct {
	MonthlyPayment decimal.Decimal `json:"monthlyPayment"`
	TotalPayment   decimal.Decimal `json:"totalPayment"`
	TotalInterest  decimal.Decimal `json:"totalInterest"`
	PayoffDate     time.Time       `json:"payoffDate"`
}

// Create creates a new debt record for the given user.
// Defaults currency to USD and sets current balance to original amount if not specified.
func (s *DebtService) Create(ctx context.Context, userID uuid.UUID, input CreateDebtInput) (*model.Debt, error) {
	debt := &model.Debt{
		UserID:         userID,
		Name:           input.Name,
		Type:           input.Type,
		OriginalAmount: input.OriginalAmount,
		CurrentBalance: input.CurrentBalance,
		InterestRate:   input.InterestRate,
		MinimumPayment: input.MinimumPayment,
		Currency:       input.Currency,
		DueDay:         input.DueDay,
		StartDate:      input.StartDate,
	}

	if debt.Currency == "" {
		debt.Currency = "USD"
	}
	if debt.CurrentBalance.IsZero() {
		debt.CurrentBalance = debt.OriginalAmount
	}

	if err := s.repo.Create(ctx, debt); err != nil {
		return nil, fmt.Errorf("creating debt: %w", err)
	}

	return debt, nil
}

// Get retrieves a debt by its ID.
func (s *DebtService) Get(ctx context.Context, id uuid.UUID) (*model.Debt, error) {
	debt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting debt %s: %w", id, err)
	}
	return debt, nil
}

// List retrieves all debts for a user.
func (s *DebtService) List(ctx context.Context, userID uuid.UUID) ([]model.Debt, error) {
	debts, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing debts for user %s: %w", userID, err)
	}
	return debts, nil
}

// Update modifies an existing debt.
// Returns ErrDebtNotFound if the debt does not exist or belongs to another user.
func (s *DebtService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input UpdateDebtInput) (*model.Debt, error) {
	debt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching debt %s for update: %w", id, err)
	}

	if debt.UserID != userID {
		return nil, repository.ErrDebtNotFound
	}

	debt.Name = input.Name
	debt.Type = input.Type
	debt.OriginalAmount = input.OriginalAmount
	debt.CurrentBalance = input.CurrentBalance
	debt.InterestRate = input.InterestRate
	debt.MinimumPayment = input.MinimumPayment
	debt.Currency = input.Currency
	debt.DueDay = input.DueDay
	debt.StartDate = input.StartDate

	if err := s.repo.Update(ctx, debt); err != nil {
		return nil, fmt.Errorf("updating debt %s: %w", id, err)
	}

	return debt, nil
}

// Delete removes a debt by ID for the given user.
func (s *DebtService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("deleting debt %s: %w", id, err)
	}
	return nil
}

// MakePayment records a payment against a debt, splitting it into principal and interest.
// The interest portion is calculated using the monthly rate.
func (s *DebtService) MakePayment(ctx context.Context, debtID uuid.UUID, userID uuid.UUID, input MakePaymentInput) (*model.Debt, error) {
	debt, err := s.repo.GetByID(ctx, debtID)
	if err != nil {
		return nil, fmt.Errorf("fetching debt %s for payment: %w", debtID, err)
	}

	if debt.UserID != userID {
		return nil, repository.ErrDebtNotFound
	}

	// Calculate interest portion (monthly rate)
	monthlyRate := debt.InterestRate.Div(decimal.NewFromInt(100)).Div(decimal.NewFromInt(12))
	interestPortion := debt.CurrentBalance.Mul(monthlyRate)
	principalPortion := input.Amount.Sub(interestPortion)

	if principalPortion.IsNegative() {
		principalPortion = decimal.Zero
		interestPortion = input.Amount
	}

	payment := &model.DebtPayment{
		DebtID:    debtID,
		Amount:    input.Amount,
		Principal: principalPortion,
		Interest:  interestPortion,
		Date:      input.Date,
	}

	if err := s.repo.RecordPayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("recording payment for debt %s: %w", debtID, err)
	}

	return s.repo.GetByID(ctx, debtID)
}

// GetPayoffPlan calculates a debt payoff plan based on the monthly payment amount.
// If monthlyPayment is zero, uses the minimum payment from the debt.
func (s *DebtService) GetPayoffPlan(ctx context.Context, id uuid.UUID, monthlyPayment decimal.Decimal) (*model.PayoffPlan, error) {
	debt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching debt %s for payoff plan: %w", id, err)
	}

	if monthlyPayment.IsZero() {
		monthlyPayment = debt.MinimumPayment
	}

	plan := calculatePayoffPlan(debt, monthlyPayment)
	return plan, nil
}

// CalculateInterest computes loan amortization details using the standard formula.
// Returns monthly payment, total payment, total interest, and payoff date.
func (s *DebtService) CalculateInterest(input InterestCalculatorInput) (*InterestCalculatorResult, error) {
	monthlyRate := input.InterestRate.Div(decimal.NewFromInt(100)).Div(decimal.NewFromInt(12))

	// Calculate fixed monthly payment using amortization formula
	// M = P * [r(1+r)^n] / [(1+r)^n - 1]
	r := monthlyRate.InexactFloat64()
	p := input.Principal.InexactFloat64()
	n := float64(input.TermMonths)

	var monthlyPayment float64
	if r == 0 {
		monthlyPayment = p / n
	} else {
		monthlyPayment = p * (r * math.Pow(1+r, n)) / (math.Pow(1+r, n) - 1)
	}

	totalPayment := monthlyPayment * n
	totalInterest := totalPayment - p

	return &InterestCalculatorResult{
		MonthlyPayment: decimal.NewFromFloat(monthlyPayment).Round(2),
		TotalPayment:   decimal.NewFromFloat(totalPayment).Round(2),
		TotalInterest:  decimal.NewFromFloat(totalInterest).Round(2),
		PayoffDate:     time.Now().AddDate(0, input.TermMonths, 0),
	}, nil
}

func calculatePayoffPlan(debt *model.Debt, monthlyPayment decimal.Decimal) *model.PayoffPlan {
	balance := debt.CurrentBalance
	monthlyRate := debt.InterestRate.Div(decimal.NewFromInt(100)).Div(decimal.NewFromInt(12))

	totalInterest := decimal.Zero
	totalPayment := decimal.Zero
	months := 0
	maxMonths := 360 // 30 years cap

	amortization := make([]model.AmortizationRow, 0)

	for balance.IsPositive() && months < maxMonths {
		months++

		interest := balance.Mul(monthlyRate).Round(2)
		payment := monthlyPayment

		if payment.GreaterThan(balance.Add(interest)) {
			payment = balance.Add(interest)
		}

		principal := payment.Sub(interest)
		balance = balance.Sub(principal)

		totalInterest = totalInterest.Add(interest)
		totalPayment = totalPayment.Add(payment)

		amortization = append(amortization, model.AmortizationRow{
			Month:            months,
			Payment:          payment,
			Principal:        principal,
			Interest:         interest,
			RemainingBalance: balance,
		})

		if balance.LessThanOrEqual(decimal.Zero) {
			break
		}
	}

	return &model.PayoffPlan{
		DebtID:           debt.ID,
		CurrentBalance:   debt.CurrentBalance,
		MonthlyPayment:   monthlyPayment,
		TotalInterest:    totalInterest,
		TotalPayment:     totalPayment,
		PayoffDate:       time.Now().AddDate(0, months, 0),
		MonthsToPayoff:   months,
		AmortizationPlan: amortization,
	}
}
