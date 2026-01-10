package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
)

// CalendarBill represents a bill or income item on the calendar.
type CalendarBill struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Amount    string `json:"amount"`
	Category  string `json:"category"`
	DueDate   string `json:"dueDate"`
	Frequency string `json:"frequency"`
	IsActive  bool   `json:"isActive"`
	Type      string `json:"type"`
}

// CalendarSummary represents the summary of bills for a month.
type CalendarSummary struct {
	TotalIncome   string `json:"totalIncome"`
	TotalExpenses string `json:"totalExpenses"`
	NetCashFlow   string `json:"netCashFlow"`
	BillCount     int    `json:"billCount"`
	IncomeCount   int    `json:"incomeCount"`
	ExpenseCount  int    `json:"expenseCount"`
}

// CalendarResponse represents the response for the recurring calendar endpoint.
type CalendarResponse struct {
	Year        int             `json:"year"`
	Month       int             `json:"month"`
	Currency    string          `json:"currency"`
	Bills       []CalendarBill  `json:"bills"`
	Summary     CalendarSummary `json:"summary"`
	GeneratedAt string          `json:"generatedAt"`
}

// CalendarHandler handles HTTP requests for the recurring calendar.
type CalendarHandler struct {
	recurringService RecurringServiceInterface
	getUserCurrency  func(ctx context.Context, userID uuid.UUID) string
}

// NewCalendarHandler creates a new CalendarHandler with the given service.
func NewCalendarHandler(recurringService RecurringServiceInterface, getUserCurrency func(ctx context.Context, userID uuid.UUID) string) *CalendarHandler {
	return &CalendarHandler{
		recurringService: recurringService,
		getUserCurrency:  getUserCurrency,
	}
}

// GetCalendar godoc
// @Summary Get recurring bills calendar
// @Description Returns recurring bills and income for a specific month in calendar format
// @Tags recurring
// @Produce json
// @Security BearerAuth
// @Param year query int true "Year (e.g., 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} CalendarResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /recurring/calendar [get]
func (h *CalendarHandler) GetCalendar(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse and validate year parameter
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "year parameter is required",
			Field: "year",
		})
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "invalid year parameter: must be a number",
			Field: "year",
		})
		return
	}

	if year < 1900 || year > 2100 {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "invalid year parameter: must be between 1900 and 2100",
			Field: "year",
		})
		return
	}

	// Parse and validate month parameter
	monthStr := r.URL.Query().Get("month")
	if monthStr == "" {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "month parameter is required",
			Field: "month",
		})
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "invalid month parameter: must be a number",
			Field: "month",
		})
		return
	}

	if month < 1 || month > 12 {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "invalid month parameter: must be between 1 and 12",
			Field: "month",
		})
		return
	}

	// Get user's currency
	currency := "USD"
	if h.getUserCurrency != nil {
		currency = h.getUserCurrency(r.Context(), userID)
	}

	// Get all recurring transactions for the user
	recurring, err := h.recurringService.GetByUserID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get recurring transactions")
		return
	}

	// Expand recurring transactions to calendar bills for the specified month
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	var bills []CalendarBill
	var totalIncome, totalExpenses decimal.Decimal
	var incomeCount, expenseCount int

	for _, rt := range recurring {
		if !rt.IsActive {
			continue
		}

		// Check if the recurring transaction is active during this month
		if rt.EndDate != nil && rt.EndDate.Before(monthStart) {
			continue
		}
		if rt.StartDate.After(monthEnd) {
			continue
		}

		// Expand to occurrences within the month
		occurrences := expandRecurringToMonth(rt, year, month)
		for _, occurrence := range occurrences {
			bills = append(bills, CalendarBill{
				ID:        rt.ID.String(),
				Name:      rt.Description,
				Amount:    rt.Amount.StringFixed(2),
				Category:  rt.Category,
				DueDate:   occurrence.Format("2006-01-02"),
				Frequency: string(rt.Frequency),
				IsActive:  rt.IsActive,
				Type:      string(rt.Type),
			})

			if rt.Type == model.TransactionTypeIncome {
				totalIncome = totalIncome.Add(rt.Amount)
				incomeCount++
			} else {
				totalExpenses = totalExpenses.Add(rt.Amount)
				expenseCount++
			}
		}
	}

	netCashFlow := totalIncome.Sub(totalExpenses)

	response := CalendarResponse{
		Year:     year,
		Month:    month,
		Currency: currency,
		Bills:    bills,
		Summary: CalendarSummary{
			TotalIncome:   totalIncome.StringFixed(2),
			TotalExpenses: totalExpenses.StringFixed(2),
			NetCashFlow:   netCashFlow.StringFixed(2),
			BillCount:     len(bills),
			IncomeCount:   incomeCount,
			ExpenseCount:  expenseCount,
		},
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, response)
}

// expandRecurringToMonth expands a recurring transaction to all occurrences within a specific month.
func expandRecurringToMonth(rt model.RecurringTransaction, year, month int) []time.Time {
	var occurrences []time.Time

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Find the first occurrence on or after month start
	current := rt.StartDate
	if current.Before(monthStart) {
		// Advance to the first occurrence within or after month start
		current = advanceToMonth(current, rt.Frequency, monthStart)
	}

	// Collect all occurrences within the month
	for current.Before(monthEnd) {
		if !current.Before(monthStart) && current.Before(monthEnd) {
			// Check end date
			if rt.EndDate != nil && current.After(*rt.EndDate) {
				break
			}
			occurrences = append(occurrences, current)
		}
		current = advanceByFrequency(current, rt.Frequency)

		// Safety check to prevent infinite loops
		if len(occurrences) > 100 {
			break
		}
	}

	return occurrences
}

// advanceToMonth advances a date by frequency until it reaches or passes the target month.
func advanceToMonth(start time.Time, frequency model.RecurringFrequency, target time.Time) time.Time {
	current := start
	for current.Before(target) {
		next := advanceByFrequency(current, frequency)
		if next.After(target) || next.Equal(target) {
			// Check if current is closer to target than next
			if !current.Before(target) {
				return current
			}
		}
		current = next
	}
	return current
}

// advanceByFrequency advances a date by the specified frequency.
func advanceByFrequency(from time.Time, frequency model.RecurringFrequency) time.Time {
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
		return from.AddDate(0, 1, 0)
	}
}
