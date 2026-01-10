package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type User struct {
	ID            uuid.UUID `db:"id" json:"id"`
	Email         string    `db:"email" json:"email"`
	PasswordHash  *string   `db:"password_hash" json:"-"`
	Name          string    `db:"name" json:"name"`
	Currency      string    `db:"currency" json:"currency"`
	OAuthProvider *string   `db:"oauth_provider" json:"oauthProvider,omitempty"`
	OAuthID       *string   `db:"oauth_id" json:"-"`
	AvatarURL     *string   `db:"avatar_url" json:"avatarUrl,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `db:"updated_at" json:"updatedAt"`
}

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeExpense TransactionType = "expense"
)

type Transaction struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	UserID      uuid.UUID       `db:"user_id" json:"userId"`
	Type        TransactionType `db:"type" json:"type"`
	Amount      decimal.Decimal `db:"amount" json:"amount"`
	Currency    string          `db:"currency" json:"currency"`
	Category    string          `db:"category" json:"category"`
	Description string          `db:"description" json:"description"`
	Date        time.Time       `db:"date" json:"date"`
	CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updatedAt"`
}

type Budget struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	UserID    uuid.UUID       `db:"user_id" json:"userId"`
	Category  string          `db:"category" json:"category"`
	Amount    decimal.Decimal `db:"amount" json:"amount"`
	Currency  string          `db:"currency" json:"currency"`
	Period    string          `db:"period" json:"period"` // monthly, weekly, yearly
	StartDate time.Time       `db:"start_date" json:"startDate"`
	EndDate   *time.Time      `db:"end_date" json:"endDate,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time       `db:"updated_at" json:"updatedAt"`
}

type BudgetWithSpent struct {
	Budget
	Spent      decimal.Decimal `db:"spent" json:"spent"`
	Remaining  decimal.Decimal `json:"remaining"`
	Percentage float64         `json:"percentage"`
}

type SavingsGoal struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	UserID        uuid.UUID       `db:"user_id" json:"userId"`
	Name          string          `db:"name" json:"name"`
	TargetAmount  decimal.Decimal `db:"target_amount" json:"targetAmount"`
	CurrentAmount decimal.Decimal `db:"current_amount" json:"currentAmount"`
	Currency      string          `db:"currency" json:"currency"`
	TargetDate    *time.Time      `db:"target_date" json:"targetDate,omitempty"`
	Color         string          `db:"color" json:"color"`
	Icon          string          `db:"icon" json:"icon"`
	CreatedAt     time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updatedAt"`
}

type DebtType string

const (
	DebtTypeMortgage     DebtType = "mortgage"
	DebtTypeAutoLoan     DebtType = "auto_loan"
	DebtTypeStudentLoan  DebtType = "student_loan"
	DebtTypeCreditCard   DebtType = "credit_card"
	DebtTypePersonalLoan DebtType = "personal_loan"
	DebtTypeOther        DebtType = "other"
)

type Debt struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	UserID         uuid.UUID       `db:"user_id" json:"userId"`
	Name           string          `db:"name" json:"name"`
	Type           DebtType        `db:"type" json:"type"`
	OriginalAmount decimal.Decimal `db:"original_amount" json:"originalAmount"`
	CurrentBalance decimal.Decimal `db:"current_balance" json:"currentBalance"`
	InterestRate   decimal.Decimal `db:"interest_rate" json:"interestRate"` // APR as percentage
	MinimumPayment decimal.Decimal `db:"minimum_payment" json:"minimumPayment"`
	Currency       string          `db:"currency" json:"currency"`
	DueDay         int             `db:"due_day" json:"dueDay"` // Day of month
	StartDate      time.Time       `db:"start_date" json:"startDate"`
	ExpectedPayoff *time.Time      `db:"expected_payoff" json:"expectedPayoff,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updatedAt"`
}

type DebtPayment struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	DebtID    uuid.UUID       `db:"debt_id" json:"debtId"`
	Amount    decimal.Decimal `db:"amount" json:"amount"`
	Principal decimal.Decimal `db:"principal" json:"principal"`
	Interest  decimal.Decimal `db:"interest" json:"interest"`
	Date      time.Time       `db:"date" json:"date"`
	CreatedAt time.Time       `db:"created_at" json:"createdAt"`
}

type PayoffPlan struct {
	DebtID           uuid.UUID         `json:"debtId"`
	CurrentBalance   decimal.Decimal   `json:"currentBalance"`
	MonthlyPayment   decimal.Decimal   `json:"monthlyPayment"`
	TotalInterest    decimal.Decimal   `json:"totalInterest"`
	TotalPayment     decimal.Decimal   `json:"totalPayment"`
	PayoffDate       time.Time         `json:"payoffDate"`
	MonthsToPayoff   int               `json:"monthsToPayoff"`
	AmortizationPlan []AmortizationRow `json:"amortizationPlan"`
}

type AmortizationRow struct {
	Month            int             `json:"month"`
	Payment          decimal.Decimal `json:"payment"`
	Principal        decimal.Decimal `json:"principal"`
	Interest         decimal.Decimal `json:"interest"`
	RemainingBalance decimal.Decimal `json:"remainingBalance"`
}

// Dashboard aggregates
type DashboardData struct {
	TotalIncome        decimal.Decimal            `json:"totalIncome"`
	TotalExpenses      decimal.Decimal            `json:"totalExpenses"`
	NetCashFlow        decimal.Decimal            `json:"netCashFlow"`
	TotalSavings       decimal.Decimal            `json:"totalSavings"`
	TotalDebt          decimal.Decimal            `json:"totalDebt"`
	BudgetSummary      []BudgetWithSpent          `json:"budgetSummary"`
	SavingsGoals       []SavingsGoal              `json:"savingsGoals"`
	RecentTransactions []Transaction              `json:"recentTransactions"`
	ExpensesByCategory map[string]decimal.Decimal `json:"expensesByCategory"`
	IncomeVsExpenses   []MonthlyComparison        `json:"incomeVsExpenses"`
}

type MonthlyComparison struct {
	Month    string          `json:"month"`
	Income   decimal.Decimal `json:"income"`
	Expenses decimal.Decimal `json:"expenses"`
}

// Recurring Transactions
type RecurringFrequency string

const (
	FrequencyDaily    RecurringFrequency = "daily"
	FrequencyWeekly   RecurringFrequency = "weekly"
	FrequencyBiweekly RecurringFrequency = "biweekly"
	FrequencyMonthly  RecurringFrequency = "monthly"
	FrequencyYearly   RecurringFrequency = "yearly"
)

type RecurringTransaction struct {
	ID             uuid.UUID          `db:"id" json:"id"`
	UserID         uuid.UUID          `db:"user_id" json:"userId"`
	Type           TransactionType    `db:"type" json:"type"`
	Amount         decimal.Decimal    `db:"amount" json:"amount"`
	Currency       string             `db:"currency" json:"currency"`
	Category       string             `db:"category" json:"category"`
	Description    string             `db:"description" json:"description"`
	Frequency      RecurringFrequency `db:"frequency" json:"frequency"`
	StartDate      time.Time          `db:"start_date" json:"startDate"`
	EndDate        *time.Time         `db:"end_date" json:"endDate,omitempty"`
	NextOccurrence time.Time          `db:"next_occurrence" json:"nextOccurrence"`
	LastGenerated  *time.Time         `db:"last_generated" json:"lastGenerated,omitempty"`
	IsActive       bool               `db:"is_active" json:"isActive"`
	CreatedAt      time.Time          `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time          `db:"updated_at" json:"updatedAt"`
}

// UpcomingBill is a simplified view for dashboard widget
type UpcomingBill struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	Description string          `db:"description" json:"description"`
	Amount      decimal.Decimal `db:"amount" json:"amount"`
	Currency    string          `db:"currency" json:"currency"`
	Category    string          `db:"category" json:"category"`
	DueDate     time.Time       `db:"due_date" json:"dueDate"`
	Type        TransactionType `db:"type" json:"type"`
}

// Categories
var ExpenseCategories = []string{
	"Housing",
	"Transportation",
	"Food & Dining",
	"Utilities",
	"Healthcare",
	"Insurance",
	"Entertainment",
	"Shopping",
	"Personal Care",
	"Education",
	"Travel",
	"Gifts & Donations",
	"Investments",
	"Debt Payments",
	"Other",
}

var IncomeCategories = []string{
	"Salary",
	"Freelance",
	"Business",
	"Investments",
	"Rental",
	"Gifts",
	"Refunds",
	"Other",
}
