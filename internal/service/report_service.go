package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/repository"
)

// AnomalyThreshold is the percentage increase that triggers an anomaly detection (50%).
const AnomalyThreshold = 0.50

// AnomalyHistoryMonths is the number of months to look back for average calculation.
const AnomalyHistoryMonths = 3

// TopCategory represents a top spending category in the monthly report.
type TopCategory struct {
	Category         string `json:"category"`
	Amount           string `json:"amount"`
	Percentage       float64 `json:"percentage"`
	TransactionCount int    `json:"transactionCount"`
}

// Anomaly represents a detected spending anomaly.
type Anomaly struct {
	Type        string `json:"type"`
	Category    string `json:"category"`
	Amount      string `json:"amount"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// MonthComparison represents the comparison with the previous month.
type MonthComparison struct {
	IncomeChange  float64 `json:"incomeChange"`
	ExpenseChange float64 `json:"expenseChange"`
	SavingsChange float64 `json:"savingsChange"`
	Trend         string  `json:"trend"`
}

// MonthlyReport represents a comprehensive monthly financial report.
type MonthlyReport struct {
	Year           int             `json:"year"`
	Month          int             `json:"month"`
	Currency       string          `json:"currency"`
	TotalIncome    string          `json:"totalIncome"`
	TotalExpenses  string          `json:"totalExpenses"`
	NetSavings     string          `json:"netSavings"`
	SavingsRate    float64         `json:"savingsRate"`
	TopCategories  []TopCategory   `json:"topCategories"`
	Anomalies      []Anomaly       `json:"anomalies"`
	ComparedToLast MonthComparison `json:"comparedToLast"`
	GeneratedAt    string          `json:"generatedAt"`
}

// MonthlyAmount represents an amount for a specific month.
type MonthlyAmount struct {
	Month  string `json:"month"`
	Amount string `json:"amount"`
}

// CategoryTrend represents the trend data for a single category.
type CategoryTrend struct {
	Category        string          `json:"category"`
	TotalAmount     string          `json:"totalAmount"`
	AverageAmount   string          `json:"averageAmount"`
	TrendDirection  string          `json:"trendDirection"`
	TrendPercentage float64         `json:"trendPercentage"`
	MonthlyData     []MonthlyAmount `json:"monthlyData"`
}

// CategoryTrendsResponse represents the response for category trends.
type CategoryTrendsResponse struct {
	Currency    string          `json:"currency"`
	PeriodStart string          `json:"periodStart"`
	PeriodEnd   string          `json:"periodEnd"`
	Trends      []CategoryTrend `json:"trends"`
	GeneratedAt string          `json:"generatedAt"`
}

// ReportRepositoryInterface defines the contract for report data access.
type ReportRepositoryInterface interface {
	GetMonthlyTotals(ctx context.Context, userID uuid.UUID, year, month int) (decimal.Decimal, decimal.Decimal, error)
	GetTopExpenseCategories(ctx context.Context, userID uuid.UUID, year, month int, limit int) ([]repository.CategoryTotal, error)
	GetCategoryAverageForPeriod(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, int, error)
	GetIncomeCategoryAverageForPeriod(ctx context.Context, userID uuid.UUID, category string, startDate, endDate time.Time) (decimal.Decimal, int, error)
	GetCategoryAmountForMonth(ctx context.Context, userID uuid.UUID, category string, year, month int) (decimal.Decimal, error)
	GetIncomeCategoryAmountForMonth(ctx context.Context, userID uuid.UUID, category string, year, month int) (decimal.Decimal, error)
	GetCategoryTrendsData(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, categoryLimit int) ([]repository.CategoryMonthlyAmount, error)
	GetDistinctExpenseCategories(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]string, error)
	GetDistinctIncomeCategories(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]string, error)
	GetUserCurrency(ctx context.Context, userID uuid.UUID) (string, error)
}

// ReportService handles business logic for financial reports.
type ReportService struct {
	reportRepo ReportRepositoryInterface
}

// NewReportService creates a new ReportService with the given repository.
func NewReportService(reportRepo ReportRepositoryInterface) *ReportService {
	return &ReportService{reportRepo: reportRepo}
}

// GetMonthlyReport generates a comprehensive monthly financial report.
func (s *ReportService) GetMonthlyReport(ctx context.Context, userID uuid.UUID, year, month int) (*MonthlyReport, error) {
	currency, err := s.reportRepo.GetUserCurrency(ctx, userID)
	if err != nil {
		currency = "USD"
	}

	// Get current month totals
	income, expenses, err := s.reportRepo.GetMonthlyTotals(ctx, userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("getting monthly totals: %w", err)
	}

	netSavings := income.Sub(expenses)
	savingsRate := float64(0)
	if !income.IsZero() {
		savingsRate = netSavings.Div(income).Mul(decimal.NewFromInt(100)).InexactFloat64()
		savingsRate = math.Round(savingsRate*100) / 100
	}

	// Get top expense categories
	topCats, err := s.reportRepo.GetTopExpenseCategories(ctx, userID, year, month, 5)
	if err != nil {
		return nil, fmt.Errorf("getting top categories: %w", err)
	}

	topCategories := make([]TopCategory, len(topCats))
	for i, cat := range topCats {
		percentage := float64(0)
		if !expenses.IsZero() {
			percentage = cat.Amount.Div(expenses).Mul(decimal.NewFromInt(100)).InexactFloat64()
			percentage = math.Round(percentage*100) / 100
		}
		topCategories[i] = TopCategory{
			Category:         cat.Category,
			Amount:           cat.Amount.StringFixed(2),
			Percentage:       percentage,
			TransactionCount: cat.TransactionCount,
		}
	}

	// Detect anomalies
	anomalies, err := s.detectAnomalies(ctx, userID, year, month)
	if err != nil {
		anomalies = []Anomaly{}
	}

	// Get comparison with previous month
	comparison, err := s.getMonthComparison(ctx, userID, year, month)
	if err != nil {
		comparison = MonthComparison{Trend: "stable"}
	}

	return &MonthlyReport{
		Year:           year,
		Month:          month,
		Currency:       currency,
		TotalIncome:    income.StringFixed(2),
		TotalExpenses:  expenses.StringFixed(2),
		NetSavings:     netSavings.StringFixed(2),
		SavingsRate:    savingsRate,
		TopCategories:  topCategories,
		Anomalies:      anomalies,
		ComparedToLast: comparison,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// detectAnomalies identifies unusual spending patterns by comparing current month to 3-month average.
func (s *ReportService) detectAnomalies(ctx context.Context, userID uuid.UUID, year, month int) ([]Anomaly, error) {
	var anomalies []Anomaly

	currentMonthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	historyStart := currentMonthStart.AddDate(0, -AnomalyHistoryMonths, 0)

	// Get all expense categories from the history period
	expenseCategories, err := s.reportRepo.GetDistinctExpenseCategories(ctx, userID, historyStart, currentMonthStart)
	if err != nil {
		return nil, fmt.Errorf("getting expense categories: %w", err)
	}

	// Check each expense category for unusual spending
	for _, category := range expenseCategories {
		avgAmount, monthCount, err := s.reportRepo.GetCategoryAverageForPeriod(ctx, userID, category, historyStart, currentMonthStart)
		if err != nil || monthCount < 2 {
			continue
		}

		currentAmount, err := s.reportRepo.GetCategoryAmountForMonth(ctx, userID, category, year, month)
		if err != nil {
			continue
		}

		// Check for unusual expense (50% or more above average)
		if !avgAmount.IsZero() && currentAmount.GreaterThan(decimal.Zero) {
			percentIncrease := currentAmount.Sub(avgAmount).Div(avgAmount).InexactFloat64()
			if percentIncrease >= AnomalyThreshold {
				percentStr := fmt.Sprintf("%.0f", percentIncrease*100)
				severity := "warning"
				if percentIncrease >= 1.0 {
					severity = "critical"
				}
				anomalies = append(anomalies, Anomaly{
					Type:        "unusual_expense",
					Category:    category,
					Amount:      currentAmount.StringFixed(2),
					Description: fmt.Sprintf("Spending %s%% higher than your %d-month average", percentStr, AnomalyHistoryMonths),
					Severity:    severity,
				})
			}
		}
	}

	// Check for missed income (regular income categories with no income this month)
	incomeCategories, err := s.reportRepo.GetDistinctIncomeCategories(ctx, userID, historyStart, currentMonthStart)
	if err != nil {
		return anomalies, nil
	}

	for _, category := range incomeCategories {
		avgIncome, monthCount, err := s.reportRepo.GetIncomeCategoryAverageForPeriod(ctx, userID, category, historyStart, currentMonthStart)
		if err != nil || monthCount < 2 {
			continue
		}

		currentIncome, err := s.reportRepo.GetIncomeCategoryAmountForMonth(ctx, userID, category, year, month)
		if err != nil {
			continue
		}

		// Check for missed income (typically receives income but none this month)
		if avgIncome.GreaterThan(decimal.NewFromInt(100)) && currentIncome.IsZero() {
			anomalies = append(anomalies, Anomaly{
				Type:        "missed_income",
				Category:    category,
				Amount:      "0.00",
				Description: fmt.Sprintf("No %s income recorded this month (usually $%s/month)", category, avgIncome.StringFixed(0)),
				Severity:    "info",
			})
		}
	}

	return anomalies, nil
}

// getMonthComparison calculates the percentage change compared to the previous month.
func (s *ReportService) getMonthComparison(ctx context.Context, userID uuid.UUID, year, month int) (MonthComparison, error) {
	// Get current month totals
	currentIncome, currentExpenses, err := s.reportRepo.GetMonthlyTotals(ctx, userID, year, month)
	if err != nil {
		return MonthComparison{Trend: "stable"}, err
	}

	// Get previous month totals
	prevMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	prevIncome, prevExpenses, err := s.reportRepo.GetMonthlyTotals(ctx, userID, prevMonth.Year(), int(prevMonth.Month()))
	if err != nil {
		return MonthComparison{Trend: "stable"}, err
	}

	currentSavings := currentIncome.Sub(currentExpenses)
	prevSavings := prevIncome.Sub(prevExpenses)

	incomeChange := calculatePercentageChange(prevIncome, currentIncome)
	expenseChange := calculatePercentageChange(prevExpenses, currentExpenses)
	savingsChange := calculatePercentageChange(prevSavings, currentSavings)

	trend := determineTrend(incomeChange, expenseChange, savingsChange)

	return MonthComparison{
		IncomeChange:  incomeChange,
		ExpenseChange: expenseChange,
		SavingsChange: savingsChange,
		Trend:         trend,
	}, nil
}

// calculatePercentageChange calculates the percentage change between two values.
func calculatePercentageChange(previous, current decimal.Decimal) float64 {
	if previous.IsZero() {
		if current.IsZero() {
			return 0
		}
		return 100
	}
	change := current.Sub(previous).Div(previous.Abs()).Mul(decimal.NewFromInt(100)).InexactFloat64()
	return math.Round(change*100) / 100
}

// determineTrend determines the overall financial trend.
func determineTrend(incomeChange, expenseChange, savingsChange float64) string {
	// Improving: savings increased significantly or expenses decreased while income stable/increased
	if savingsChange > 10 {
		return "improving"
	}
	if expenseChange < -5 && incomeChange >= 0 {
		return "improving"
	}

	// Declining: savings decreased significantly or expenses increased significantly
	if savingsChange < -10 {
		return "declining"
	}
	if expenseChange > 15 && incomeChange <= 0 {
		return "declining"
	}

	return "stable"
}

// GetCategoryTrends retrieves spending trends by category over multiple months.
func (s *ReportService) GetCategoryTrends(ctx context.Context, userID uuid.UUID, months, categoryLimit int) (*CategoryTrendsResponse, error) {
	if months <= 0 {
		months = 6
	}
	if months > 24 {
		months = 24
	}
	if categoryLimit <= 0 {
		categoryLimit = 10
	}
	if categoryLimit > 20 {
		categoryLimit = 20
	}

	currency, err := s.reportRepo.GetUserCurrency(ctx, userID)
	if err != nil {
		currency = "USD"
	}

	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := currentMonthStart.AddDate(0, 1, 0)
	periodStart := currentMonthStart.AddDate(0, -months+1, 0)

	// Get category trends data
	trendsData, err := s.reportRepo.GetCategoryTrendsData(ctx, userID, periodStart, periodEnd, categoryLimit)
	if err != nil {
		return nil, fmt.Errorf("getting category trends data: %w", err)
	}

	// Organize data by category
	categoryData := make(map[string]map[string]decimal.Decimal)
	for _, item := range trendsData {
		if categoryData[item.Category] == nil {
			categoryData[item.Category] = make(map[string]decimal.Decimal)
		}
		categoryData[item.Category][item.Month] = item.Amount
	}

	// Generate all months in the period
	allMonths := generateMonthRange(periodStart, months)

	// Build trends response
	var trends []CategoryTrend
	for category, monthAmounts := range categoryData {
		var monthlyData []MonthlyAmount
		var total decimal.Decimal
		var firstAmount, lastAmount decimal.Decimal
		firstSet, lastSet := false, false

		for _, monthStr := range allMonths {
			amount := monthAmounts[monthStr]
			if amount.IsZero() {
				amount = decimal.Zero
			}
			total = total.Add(amount)

			if !firstSet && amount.GreaterThan(decimal.Zero) {
				firstAmount = amount
				firstSet = true
			}
			if amount.GreaterThan(decimal.Zero) {
				lastAmount = amount
				lastSet = true
			}

			monthlyData = append(monthlyData, MonthlyAmount{
				Month:  monthStr,
				Amount: amount.StringFixed(2),
			})
		}

		average := decimal.Zero
		if len(allMonths) > 0 {
			average = total.Div(decimal.NewFromInt(int64(len(allMonths))))
		}

		trendDirection, trendPercentage := calculateTrend(firstAmount, lastAmount, firstSet && lastSet)

		trends = append(trends, CategoryTrend{
			Category:        category,
			TotalAmount:     total.StringFixed(2),
			AverageAmount:   average.StringFixed(2),
			TrendDirection:  trendDirection,
			TrendPercentage: trendPercentage,
			MonthlyData:     monthlyData,
		})
	}

	return &CategoryTrendsResponse{
		Currency:    currency,
		PeriodStart: periodStart.Format("2006-01-02"),
		PeriodEnd:   periodEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		Trends:      trends,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// generateMonthRange generates a slice of month strings (YYYY-MM) for the given period.
func generateMonthRange(start time.Time, months int) []string {
	result := make([]string, months)
	current := start
	for i := 0; i < months; i++ {
		result[i] = current.Format("2006-01")
		current = current.AddDate(0, 1, 0)
	}
	return result
}

// calculateTrend determines the trend direction and percentage change.
func calculateTrend(first, last decimal.Decimal, hasData bool) (string, float64) {
	if !hasData || first.IsZero() {
		return "stable", 0
	}

	change := last.Sub(first).Div(first).Mul(decimal.NewFromInt(100)).InexactFloat64()
	change = math.Round(change*100) / 100

	if change > 5 {
		return "increasing", change
	}
	if change < -5 {
		return "decreasing", change
	}
	return "stable", change
}
