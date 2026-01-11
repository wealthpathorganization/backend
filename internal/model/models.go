package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type User struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	Email           string     `db:"email" json:"email"`
	PasswordHash    *string    `db:"password_hash" json:"-"`
	Name            string     `db:"name" json:"name"`
	Currency        string     `db:"currency" json:"currency"`
	OAuthProvider   *string    `db:"oauth_provider" json:"oauthProvider,omitempty"`
	OAuthID         *string    `db:"oauth_id" json:"-"`
	AvatarURL       *string    `db:"avatar_url" json:"avatarUrl,omitempty"`
	TOTPSecret      *string    `db:"totp_secret" json:"-"`
	TOTPEnabled     bool       `db:"totp_enabled" json:"totpEnabled"`
	TOTPBackupCodes []string   `db:"totp_backup_codes" json:"-"`
	TOTPVerifiedAt  *time.Time `db:"totp_verified_at" json:"-"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updatedAt"`
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
	ID                uuid.UUID        `db:"id" json:"id"`
	UserID            uuid.UUID        `db:"user_id" json:"userId"`
	Category          string           `db:"category" json:"category"`
	Amount            decimal.Decimal  `db:"amount" json:"amount"`
	Currency          string           `db:"currency" json:"currency"`
	Period            string           `db:"period" json:"period"` // monthly, weekly, yearly
	StartDate         time.Time        `db:"start_date" json:"startDate"`
	EndDate           *time.Time       `db:"end_date" json:"endDate,omitempty"`
	EnableRollover    bool             `db:"enable_rollover" json:"enableRollover"`
	MaxRolloverAmount *decimal.Decimal `db:"max_rollover_amount" json:"maxRolloverAmount,omitempty"`
	RolloverAmount    decimal.Decimal  `db:"rollover_amount" json:"rolloverAmount"`
	CreatedAt         time.Time        `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time        `db:"updated_at" json:"updatedAt"`
}

// BudgetRollover represents a rollover transaction from one period to another.
type BudgetRollover struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	BudgetID        uuid.UUID       `db:"budget_id" json:"budgetId"`
	FromPeriodStart time.Time       `db:"from_period_start" json:"fromPeriodStart"`
	FromPeriodEnd   time.Time       `db:"from_period_end" json:"fromPeriodEnd"`
	ToPeriodStart   time.Time       `db:"to_period_start" json:"toPeriodStart"`
	Amount          decimal.Decimal `db:"amount" json:"amount"`
	CreatedAt       time.Time       `db:"created_at" json:"createdAt"`
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

// Push Notifications

type PushSubscription struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"userId"`
	Endpoint  string    `db:"endpoint" json:"endpoint"`
	P256dh    string    `db:"p256dh" json:"p256dh"`
	Auth      string    `db:"auth" json:"auth"`
	UserAgent *string   `db:"user_agent" json:"userAgent,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type NotificationPreferences struct {
	ID                    uuid.UUID `db:"id" json:"id"`
	UserID                uuid.UUID `db:"user_id" json:"userId"`
	BillRemindersEnabled  bool      `db:"bill_reminders_enabled" json:"billRemindersEnabled"`
	BillReminderDaysBefore int      `db:"bill_reminder_days_before" json:"billReminderDaysBefore"`
	BudgetAlertsEnabled   bool      `db:"budget_alerts_enabled" json:"budgetAlertsEnabled"`
	BudgetAlertThreshold  int       `db:"budget_alert_threshold" json:"budgetAlertThreshold"` // Percentage
	GoalMilestonesEnabled bool      `db:"goal_milestones_enabled" json:"goalMilestonesEnabled"`
	WeeklySummaryEnabled  bool      `db:"weekly_summary_enabled" json:"weeklySummaryEnabled"`
	CreatedAt             time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt             time.Time `db:"updated_at" json:"updatedAt"`
}

type NotificationType string

const (
	NotificationTypeBillReminder  NotificationType = "bill_reminder"
	NotificationTypeBudgetAlert   NotificationType = "budget_alert"
	NotificationTypeGoalMilestone NotificationType = "goal_milestone"
	NotificationTypeWeeklySummary NotificationType = "weekly_summary"
)

type NotificationLog struct {
	ID               uuid.UUID        `db:"id" json:"id"`
	UserID           uuid.UUID        `db:"user_id" json:"userId"`
	NotificationType NotificationType `db:"notification_type" json:"notificationType"`
	ReferenceID      *uuid.UUID       `db:"reference_id" json:"referenceId,omitempty"`
	ReferenceDate    *time.Time       `db:"reference_date" json:"referenceDate,omitempty"`
	Title            string           `db:"title" json:"title"`
	Body             string           `db:"body" json:"body"`
	SentAt           time.Time        `db:"sent_at" json:"sentAt"`
	Success          bool             `db:"success" json:"success"`
	ErrorMessage     *string          `db:"error_message" json:"errorMessage,omitempty"`
}
