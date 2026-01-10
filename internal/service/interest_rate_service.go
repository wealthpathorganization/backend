package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/internal/scraper"
)

// InterestRateService handles interest rate operations
type InterestRateService struct {
	repo         repository.InterestRateRepository
	orchestrator *scraper.Orchestrator
}

// NewInterestRateService creates a new interest rate service
func NewInterestRateService(repo repository.InterestRateRepository) *InterestRateService {
	return &InterestRateService{
		repo:         repo,
		orchestrator: scraper.NewOrchestrator(scraper.DefaultOrchestratorConfig(), slog.Default()),
	}
}

// ScrapeAndUpdateRates scrapes rates from all banks and updates the database
func (s *InterestRateService) ScrapeAndUpdateRates(ctx context.Context) (int, error) {
	results, err := s.orchestrator.ScrapeAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("scrape rates: %w", err)
	}

	// Collect all rates from successful scrapes
	var allRates []model.InterestRate
	for _, result := range results {
		if result.Success {
			allRates = append(allRates, result.Rates...)
		}
	}

	if len(allRates) == 0 {
		return 0, fmt.Errorf("no rates scraped successfully")
	}

	if err := s.BulkUpsertRates(ctx, allRates); err != nil {
		return 0, fmt.Errorf("upsert rates: %w", err)
	}

	return len(allRates), nil
}

// GetScraperHealth returns the health status of the scraper
func (s *InterestRateService) GetScraperHealth(nextRunTime time.Time) scraper.HealthStatus {
	return s.orchestrator.GetHealthStatus(nextRunTime)
}

// ListRates returns interest rates with optional filters
func (s *InterestRateService) ListRates(ctx context.Context, productType string, termMonths *int, bankCode string) ([]model.InterestRate, error) {
	return s.repo.List(ctx, productType, termMonths, bankCode)
}

// GetBestRates returns the best rates for a given product type and term
func (s *InterestRateService) GetBestRates(ctx context.Context, productType string, termMonths int, limit int) ([]model.InterestRate, error) {
	return s.repo.GetBestRates(ctx, productType, termMonths, limit)
}

// CompareRates returns rates from all banks for a specific term
func (s *InterestRateService) CompareRates(ctx context.Context, productType string, termMonths int) ([]model.InterestRate, error) {
	return s.repo.List(ctx, productType, &termMonths, "")
}

// GetBanks returns list of supported banks
func (s *InterestRateService) GetBanks() []model.Bank {
	return model.VietnameseBanks
}

// GetRateHistory returns historical rate data for a specific bank/product/term
func (s *InterestRateService) GetRateHistory(ctx context.Context, bankCode, productType string, termMonths, days int) ([]repository.RateHistoryEntry, error) {
	return s.repo.GetHistory(ctx, bankCode, productType, termMonths, days)
}

// UpsertRate creates or updates an interest rate
func (s *InterestRateService) UpsertRate(ctx context.Context, rate *model.InterestRate) error {
	return s.repo.Upsert(ctx, rate)
}

// BulkUpsertRates creates or updates multiple interest rates
func (s *InterestRateService) BulkUpsertRates(ctx context.Context, rates []model.InterestRate) error {
	for _, rate := range rates {
		if err := s.repo.Upsert(ctx, &rate); err != nil {
			return fmt.Errorf("upsert rate for %s: %w", rate.BankCode, err)
		}
	}
	return nil
}

// SeedDefaultRates seeds the database with sample/default rates
// This can be used for initial setup or when scraping fails
func (s *InterestRateService) SeedDefaultRates(ctx context.Context) error {
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	// Sample deposit rates (as of late 2024 - these should be updated regularly)
	// Source: General market rates, actual rates vary
	sampleRates := []model.InterestRate{
		// Vietcombank
		{BankCode: "vcb", BankName: "Vietcombank", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(1.7), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "vcb", BankName: "Vietcombank", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(2.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "vcb", BankName: "Vietcombank", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(3.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "vcb", BankName: "Vietcombank", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(4.7), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},

		// Techcombank
		{BankCode: "tcb", BankName: "Techcombank", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(2.55), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "tcb", BankName: "Techcombank", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(2.65), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "tcb", BankName: "Techcombank", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(3.55), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "tcb", BankName: "Techcombank", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(4.85), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},

		// MB Bank
		{BankCode: "mb", BankName: "MB Bank", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(2.9), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "mb", BankName: "MB Bank", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(3.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "mb", BankName: "MB Bank", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(3.9), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "mb", BankName: "MB Bank", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(5.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},

		// BIDV
		{BankCode: "bidv", BankName: "BIDV", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(1.7), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "bidv", BankName: "BIDV", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(2.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "bidv", BankName: "BIDV", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(3.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "bidv", BankName: "BIDV", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(4.7), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},

		// VPBank
		{BankCode: "vpbank", BankName: "VPBank", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(3.4), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "vpbank", BankName: "VPBank", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(3.5), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "vpbank", BankName: "VPBank", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(4.3), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "vpbank", BankName: "VPBank", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(5.3), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},

		// TPBank
		{BankCode: "tpbank", BankName: "TPBank", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(3.3), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "tpbank", BankName: "TPBank", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(3.4), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "tpbank", BankName: "TPBank", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(4.3), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "tpbank", BankName: "TPBank", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(5.3), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},

		// ACB
		{BankCode: "acb", BankName: "ACB", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(2.7), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "acb", BankName: "ACB", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(2.8), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "acb", BankName: "ACB", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(3.6), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "acb", BankName: "ACB", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(4.6), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},

		// Sacombank
		{BankCode: "sacombank", BankName: "Sacombank", ProductType: "deposit", TermMonths: 1, TermLabel: "1 tháng", Rate: decimal.NewFromFloat(2.8), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "sacombank", BankName: "Sacombank", ProductType: "deposit", TermMonths: 3, TermLabel: "3 tháng", Rate: decimal.NewFromFloat(3.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "sacombank", BankName: "Sacombank", ProductType: "deposit", TermMonths: 6, TermLabel: "6 tháng", Rate: decimal.NewFromFloat(4.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
		{BankCode: "sacombank", BankName: "Sacombank", ProductType: "deposit", TermMonths: 12, TermLabel: "12 tháng", Rate: decimal.NewFromFloat(5.0), Currency: "VND", EffectiveDate: effectiveDate, ScrapedAt: now},
	}

	return s.BulkUpsertRates(ctx, sampleRates)
}
