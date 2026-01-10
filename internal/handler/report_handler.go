package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/service"
)

// ReportServiceInterface defines the contract for report business logic.
type ReportServiceInterface interface {
	GetMonthlyReport(ctx context.Context, userID uuid.UUID, year, month int) (*service.MonthlyReport, error)
	GetCategoryTrends(ctx context.Context, userID uuid.UUID, months, limit int) (*service.CategoryTrendsResponse, error)
}

// ReportHandler handles HTTP requests for financial reports.
type ReportHandler struct {
	reportService ReportServiceInterface
}

// NewReportHandler creates a new ReportHandler with the given service.
func NewReportHandler(reportService ReportServiceInterface) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// GetMonthlyReport godoc
// @Summary Get monthly financial report
// @Description Returns a comprehensive monthly financial report with analytics and comparisons
// @Tags reports
// @Produce json
// @Security BearerAuth
// @Param year query int true "Year for the report (e.g., 2026)"
// @Param month query int true "Month for the report (1-12)"
// @Success 200 {object} service.MonthlyReport
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /reports/monthly [get]
func (h *ReportHandler) GetMonthlyReport(w http.ResponseWriter, r *http.Request) {
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

	report, err := h.reportService.GetMonthlyReport(r.Context(), userID, year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate report")
		return
	}

	respondJSON(w, http.StatusOK, report)
}

// GetCategoryTrends godoc
// @Summary Get category spending trends
// @Description Returns spending trends by category over multiple months for trend analysis and charts
// @Tags reports
// @Produce json
// @Security BearerAuth
// @Param months query int false "Number of months to include (default: 6, max: 24)"
// @Param limit query int false "Number of categories to return (default: 10, max: 20)"
// @Success 200 {object} service.CategoryTrendsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /reports/category-trends [get]
func (h *ReportHandler) GetCategoryTrends(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse months parameter (optional, default 6)
	months := 6
	if monthsStr := r.URL.Query().Get("months"); monthsStr != "" {
		m, err := strconv.Atoi(monthsStr)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid months parameter: must be a number",
				Field: "months",
			})
			return
		}
		if m < 1 || m > 24 {
			respondJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid months parameter: must be between 1 and 24",
				Field: "months",
			})
			return
		}
		months = m
	}

	// Parse limit parameter (optional, default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid limit parameter: must be a number",
				Field: "limit",
			})
			return
		}
		if l < 1 || l > 20 {
			respondJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid limit parameter: must be between 1 and 20",
				Field: "limit",
			})
			return
		}
		limit = l
	}

	trends, err := h.reportService.GetCategoryTrends(r.Context(), userID, months, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate category trends")
		return
	}

	respondJSON(w, http.StatusOK, trends)
}
