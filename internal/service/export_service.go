package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"

	"github.com/wealthpath/backend/internal/repository"
)

// ExportService handles data export functionality for transactions and reports.
type ExportService struct {
	transactionRepo repository.TransactionRepositoryInterface
}

// NewExportService creates a new ExportService with the given repository.
func NewExportService(transactionRepo repository.TransactionRepositoryInterface) *ExportService {
	return &ExportService{transactionRepo: transactionRepo}
}

// ExportTransactionsCSV exports transactions to CSV format.
// Applies the given filters and returns a CSV byte buffer.
func (s *ExportService) ExportTransactionsCSV(ctx context.Context, userID uuid.UUID, filters repository.TransactionFilters) ([]byte, error) {
	// Set reasonable limit for export
	filters.Limit = 10000
	filters.Offset = 0

	transactions, err := s.transactionRepo.List(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("fetching transactions for export: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"Date", "Type", "Category", "Amount", "Currency", "Description"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("writing CSV header: %w", err)
	}

	// Write data rows
	for _, tx := range transactions {
		row := []string{
			tx.Date.Format("2006-01-02"),
			string(tx.Type),
			tx.Category,
			tx.Amount.String(),
			tx.Currency,
			tx.Description,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("writing CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flushing CSV writer: %w", err)
	}

	return buf.Bytes(), nil
}

// MonthlyReportData contains the data needed for PDF report generation.
type MonthlyReportData struct {
	Year          int
	Month         int
	Currency      string
	TotalIncome   string
	TotalExpenses string
	NetSavings    string
	SavingsRate   float64
	TopCategories []TopCategoryData
}

// TopCategoryData represents a category's spending data.
type TopCategoryData struct {
	Category   string
	Amount     string
	Percentage float64
}

// ExportMonthlyReportPDF generates a PDF report for the given month.
func (s *ExportService) ExportMonthlyReportPDF(reportData MonthlyReportData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 24)
	pdf.SetTextColor(33, 37, 41)
	pdf.CellFormat(0, 12, "WealthPath", "", 1, "C", false, 0, "")

	// Subtitle with month/year
	pdf.SetFont("Arial", "", 14)
	pdf.SetTextColor(108, 117, 125)
	monthName := time.Month(reportData.Month).String()
	pdf.CellFormat(0, 8, fmt.Sprintf("Monthly Report - %s %d", monthName, reportData.Year), "", 1, "C", false, 0, "")

	pdf.Ln(10)

	// Summary Section
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(33, 37, 41)
	pdf.CellFormat(0, 8, "Summary", "", 1, "L", false, 0, "")

	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(5)

	// Summary Table
	pdf.SetFont("Arial", "", 11)
	colWidth := float64(85)

	// Income
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(colWidth, 7, "Total Income", "", 0, "L", false, 0, "")
	pdf.SetTextColor(40, 167, 69)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(colWidth, 7, reportData.TotalIncome, "", 1, "R", false, 0, "")

	// Expenses
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(colWidth, 7, "Total Expenses", "", 0, "L", false, 0, "")
	pdf.SetTextColor(220, 53, 69)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(colWidth, 7, reportData.TotalExpenses, "", 1, "R", false, 0, "")

	// Net Savings
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(colWidth, 7, "Net Savings", "", 0, "L", false, 0, "")
	pdf.SetTextColor(33, 37, 41)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(colWidth, 7, reportData.NetSavings, "", 1, "R", false, 0, "")

	// Savings Rate
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(colWidth, 7, "Savings Rate", "", 0, "L", false, 0, "")
	pdf.SetTextColor(33, 37, 41)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(colWidth, 7, fmt.Sprintf("%.1f%%", reportData.SavingsRate), "", 1, "R", false, 0, "")

	pdf.Ln(15)

	// Top Categories Section
	if len(reportData.TopCategories) > 0 {
		pdf.SetFont("Arial", "B", 14)
		pdf.SetTextColor(33, 37, 41)
		pdf.CellFormat(0, 8, "Top Spending Categories", "", 1, "L", false, 0, "")

		pdf.SetDrawColor(200, 200, 200)
		pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
		pdf.Ln(5)

		// Table header
		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(248, 249, 250)
		pdf.SetTextColor(33, 37, 41)
		pdf.CellFormat(80, 8, "Category", "1", 0, "L", true, 0, "")
		pdf.CellFormat(45, 8, "Amount", "1", 0, "R", true, 0, "")
		pdf.CellFormat(45, 8, "% of Total", "1", 1, "R", true, 0, "")

		// Table rows
		pdf.SetFont("Arial", "", 10)
		for _, cat := range reportData.TopCategories {
			pdf.SetTextColor(33, 37, 41)
			pdf.CellFormat(80, 7, cat.Category, "1", 0, "L", false, 0, "")
			pdf.CellFormat(45, 7, cat.Amount, "1", 0, "R", false, 0, "")
			pdf.CellFormat(45, 7, fmt.Sprintf("%.1f%%", cat.Percentage), "1", 1, "R", false, 0, "")
		}
	}

	// Footer
	pdf.SetY(-25)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated by WealthPath on %s", time.Now().Format("January 2, 2006")), "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}

	return buf.Bytes(), nil
}

