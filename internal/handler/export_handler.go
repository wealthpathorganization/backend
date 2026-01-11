package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/internal/service"
)

// ExportHandler handles data export endpoints.
type ExportHandler struct {
	exportService *service.ExportService
	reportService ReportServiceInterface
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(exportService *service.ExportService, reportService ReportServiceInterface) *ExportHandler {
	return &ExportHandler{
		exportService: exportService,
		reportService: reportService,
	}
}

// ExportTransactionsCSV godoc
// @Summary Export transactions to CSV
// @Description Export transactions to CSV format with optional filters
// @Tags export
// @Produce text/csv
// @Security BearerAuth
// @Param type query string false "Transaction type (income or expense)"
// @Param category query string false "Single category filter"
// @Param categories query string false "Comma-separated categories"
// @Param search query string false "Search in description"
// @Param startDate query string false "Start date (YYYY-MM-DD)"
// @Param endDate query string false "End date (YYYY-MM-DD)"
// @Success 200 {file} file "CSV file"
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/export/csv [get]
func (h *ExportHandler) ExportTransactionsCSV(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	// Parse filters
	filters := repository.TransactionFilters{}

	if t := r.URL.Query().Get("type"); t != "" {
		if tp := parseTransactionType(t); tp != nil {
			filters.Type = tp
		}
	}
	if c := r.URL.Query().Get("category"); c != "" {
		filters.Category = &c
	}
	if cats := r.URL.Query().Get("categories"); cats != "" {
		filters.Categories = splitAndTrim(cats, ",")
	}
	if s := r.URL.Query().Get("search"); s != "" {
		filters.Search = &s
	}
	if sd := r.URL.Query().Get("startDate"); sd != "" {
		if t, err := time.Parse("2006-01-02", sd); err == nil {
			filters.StartDate = &t
		}
	}
	if ed := r.URL.Query().Get("endDate"); ed != "" {
		if t, err := time.Parse("2006-01-02", ed); err == nil {
			filters.EndDate = &t
		}
	}

	csvData, err := h.exportService.ExportTransactionsCSV(r.Context(), userID, filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to export transactions")
		return
	}

	// Set headers for CSV download
	filename := fmt.Sprintf("transactions_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(csvData)))
	w.Write(csvData)
}

// ExportMonthlyReportPDF godoc
// @Summary Export monthly report to PDF
// @Description Export a monthly financial report to PDF format
// @Tags export
// @Produce application/pdf
// @Security BearerAuth
// @Param year path int true "Year"
// @Param month path int true "Month (1-12)"
// @Success 200 {file} file "PDF file"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /reports/monthly/{year}/{month}/export/pdf [get]
func (h *ExportHandler) ExportMonthlyReportPDF(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil || year < 2000 || year > 2100 {
		respondError(w, http.StatusBadRequest, "invalid year")
		return
	}

	month, err := strconv.Atoi(chi.URLParam(r, "month"))
	if err != nil || month < 1 || month > 12 {
		respondError(w, http.StatusBadRequest, "invalid month")
		return
	}

	// Get the monthly report data
	report, err := h.reportService.GetMonthlyReport(r.Context(), userID, year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get monthly report")
		return
	}

	// Convert to PDF format (report already has formatted amounts)
	topCategories := make([]service.TopCategoryData, 0, len(report.TopCategories))
	for _, cat := range report.TopCategories {
		topCategories = append(topCategories, service.TopCategoryData{
			Category:   cat.Category,
			Amount:     cat.Amount,
			Percentage: cat.Percentage,
		})
	}

	reportData := service.MonthlyReportData{
		Year:          year,
		Month:         month,
		Currency:      report.Currency,
		TotalIncome:   report.TotalIncome,
		TotalExpenses: report.TotalExpenses,
		NetSavings:    report.NetSavings,
		SavingsRate:   report.SavingsRate,
		TopCategories: topCategories,
	}

	pdfData, err := h.exportService.ExportMonthlyReportPDF(reportData)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	// Set headers for PDF download
	filename := fmt.Sprintf("wealthpath_report_%d_%02d.pdf", year, month)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))
	w.Write(pdfData)
}
