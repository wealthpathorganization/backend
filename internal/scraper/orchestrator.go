package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/scraper/banks"
	"github.com/wealthpath/backend/internal/scraper/browser"
)

// OrchestratorConfig holds configuration for the scraper orchestrator
type OrchestratorConfig struct {
	// MinDelay is the minimum delay between scraping different banks
	MinDelay time.Duration
	// MaxDelay is the maximum delay between scraping different banks
	MaxDelay time.Duration
	// RequestTimeout is the timeout for individual scrape requests
	RequestTimeout time.Duration
	// RetryConfig holds retry configuration for failed scrapes
	RetryConfig RetryConfig
	// MaxBrowserPages is the maximum number of concurrent browser pages (default: 3)
	MaxBrowserPages int
	// EnableParallel enables parallel scraping (default: false for backward compatibility)
	EnableParallel bool
}

// DefaultOrchestratorConfig returns the default orchestrator configuration
func DefaultOrchestratorConfig() OrchestratorConfig {
	return OrchestratorConfig{
		MinDelay:        2 * time.Second,
		MaxDelay:        5 * time.Second,
		RequestTimeout:  60 * time.Second, // Increased for JS-heavy pages
		RetryConfig:     DefaultRetryConfig(),
		MaxBrowserPages: 3,
		EnableParallel:  false,
	}
}

// ScrapeResult holds the result of scraping a single bank
type ScrapeResult struct {
	BankCode     string
	BankName     string
	Rates        []model.InterestRate
	Success      bool
	Error        error
	Duration     time.Duration
	RatesScraped int
}

// bankScraperInterface is implemented by bank scrapers in the banks package
type bankScraperInterface interface {
	BankCode() string
	BankName() string
	ScrapeRates(ctx context.Context) ([]model.InterestRate, error)
}

// Orchestrator coordinates scraping from multiple banks
type Orchestrator struct {
	config   OrchestratorConfig
	scrapers []bankScraperInterface
	metrics  *MetricsCollector
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewOrchestrator creates a new scraper orchestrator
func NewOrchestrator(cfg OrchestratorConfig, logger *slog.Logger) *Orchestrator {
	if logger == nil {
		logger = slog.Default()
	}

	client := &http.Client{
		Timeout: cfg.RequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 2,
		},
	}

	// Initialize all bank scrapers
	scrapers := []bankScraperInterface{
		banks.NewVietcombankScraper(client),
		banks.NewTechcombankScraper(client),
		banks.NewMBBankScraper(client),
		banks.NewBIDVScraper(client),
		banks.NewVPBankScraper(client),
		banks.NewTPBankScraper(client),
		banks.NewACBScraper(client),
		banks.NewAgribankScraper(client),
		banks.NewSacombankScraper(client),
		banks.NewHDBankScraper(client),
	}

	return &Orchestrator{
		config:   cfg,
		scrapers: scrapers,
		metrics:  NewMetricsCollector(),
		logger:   logger,
	}
}

// ScrapeAll scrapes rates from all banks with rate limiting
func (o *Orchestrator) ScrapeAll(ctx context.Context) ([]ScrapeResult, error) {
	o.logger.Info("Starting scrape of all banks",
		slog.Int("bank_count", len(o.scrapers)),
	)

	results := make([]ScrapeResult, 0, len(o.scrapers))

	for i, scraper := range o.scrapers {
		// Check context before scraping
		select {
		case <-ctx.Done():
			o.logger.Warn("Scrape cancelled",
				slog.Int("completed", i),
				slog.Int("total", len(o.scrapers)),
			)
			o.metrics.FinishRun()
			return results, ctx.Err()
		default:
		}

		result := o.scrapeBank(ctx, scraper)
		results = append(results, result)

		// Add delay between banks (except for the last one)
		if i < len(o.scrapers)-1 {
			delay := o.randomDelay()
			o.logger.Debug("Waiting before next bank",
				slog.String("next_bank", o.scrapers[i+1].BankCode()),
				slog.Duration("delay", delay),
			)

			select {
			case <-ctx.Done():
				o.metrics.FinishRun()
				return results, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	o.metrics.FinishRun()

	// Log summary
	var successCount, failCount, totalRates int
	for _, r := range results {
		if r.Success {
			successCount++
			totalRates += r.RatesScraped
		} else {
			failCount++
		}
	}

	o.logger.Info("Scrape completed",
		slog.Int("successful", successCount),
		slog.Int("failed", failCount),
		slog.Int("total_rates", totalRates),
	)

	return results, nil
}

// scrapeBank scrapes a single bank with retry logic
func (o *Orchestrator) scrapeBank(ctx context.Context, scraper bankScraperInterface) ScrapeResult {
	bankCode := scraper.BankCode()
	bankName := scraper.BankName()

	o.logger.Info("Scraping bank",
		slog.String("bank_code", bankCode),
		slog.String("bank_name", bankName),
	)

	o.metrics.StartScrape(bankCode)
	startTime := time.Now()

	var rates []model.InterestRate
	var scrapeErr error

	// Use retry logic
	err := WithRetry(ctx, o.config.RetryConfig, o.logger, func() error {
		var err error
		rates, err = scraper.ScrapeRates(ctx)
		if err != nil {
			return err
		}
		if len(rates) == 0 {
			return ErrNoDataFound
		}
		return nil
	})

	duration := time.Since(startTime)

	if err != nil {
		scrapeErr = err
		o.metrics.RecordFailure(bankCode, err)
		o.logger.Error("Failed to scrape bank",
			slog.String("bank_code", bankCode),
			slog.String("error", err.Error()),
			slog.Duration("duration", duration),
		)

		return ScrapeResult{
			BankCode: bankCode,
			BankName: bankName,
			Success:  false,
			Error:    scrapeErr,
			Duration: duration,
		}
	}

	o.metrics.RecordSuccess(bankCode, len(rates))
	o.logger.Info("Successfully scraped bank",
		slog.String("bank_code", bankCode),
		slog.Int("rates_count", len(rates)),
		slog.Duration("duration", duration),
	)

	return ScrapeResult{
		BankCode:     bankCode,
		BankName:     bankName,
		Rates:        rates,
		Success:      true,
		Duration:     duration,
		RatesScraped: len(rates),
	}
}

// scrapeBankWithBrowser scrapes a bank using a headless browser
func (o *Orchestrator) scrapeBankWithBrowser(ctx context.Context, scraper bankScraperInterface, page *rod.Page) ScrapeResult {
	bankCode := scraper.BankCode()
	bankName := scraper.BankName()

	o.logger.Info("Scraping bank with browser",
		slog.String("bank_code", bankCode),
		slog.String("bank_name", bankName),
	)

	o.metrics.StartScrape(bankCode)
	startTime := time.Now()

	var rates []model.InterestRate
	var scrapeErr error

	// Check if scraper implements BrowserScraper interface
	browserScraper, ok := scraper.(banks.BrowserScraper)
	if !ok {
		// Fallback to regular scraping
		return o.scrapeBank(ctx, scraper)
	}

	// Use retry logic with browser
	err := WithRetry(ctx, o.config.RetryConfig, o.logger, func() error {
		var err error
		rates, err = browserScraper.ScrapeWithBrowser(ctx, page)
		if err != nil {
			return err
		}
		if len(rates) == 0 {
			return ErrNoDataFound
		}
		return nil
	})

	duration := time.Since(startTime)

	if err != nil {
		scrapeErr = err
		o.metrics.RecordFailure(bankCode, err)
		o.logger.Error("Failed to scrape bank with browser",
			slog.String("bank_code", bankCode),
			slog.String("error", err.Error()),
			slog.Duration("duration", duration),
		)

		return ScrapeResult{
			BankCode: bankCode,
			BankName: bankName,
			Success:  false,
			Error:    scrapeErr,
			Duration: duration,
		}
	}

	o.metrics.RecordSuccess(bankCode, len(rates))
	o.logger.Info("Successfully scraped bank with browser",
		slog.String("bank_code", bankCode),
		slog.Int("rates_count", len(rates)),
		slog.Duration("duration", duration),
	)

	return ScrapeResult{
		BankCode:     bankCode,
		BankName:     bankName,
		Rates:        rates,
		Success:      true,
		Duration:     duration,
		RatesScraped: len(rates),
	}
}

// indexedScraper holds a scraper with its original index
type indexedScraper struct {
	index   int
	scraper bankScraperInterface
}

// indexedResult holds a result with its original index
type indexedResult struct {
	index  int
	result ScrapeResult
}

// ScrapeAllParallel scrapes all banks in parallel using goroutines and browser pool
func (o *Orchestrator) ScrapeAllParallel(ctx context.Context) ([]ScrapeResult, error) {
	o.logger.Info("Starting parallel scrape of all banks",
		slog.Int("bank_count", len(o.scrapers)),
		slog.Int("max_browser_pages", o.config.MaxBrowserPages),
	)

	results := make([]ScrapeResult, len(o.scrapers))
	var wg sync.WaitGroup
	resultsCh := make(chan indexedResult, len(o.scrapers))

	// Separate scrapers by type
	var httpScrapers, browserScrapers []indexedScraper
	for i, s := range o.scrapers {
		if bs, ok := s.(banks.BrowserScraper); ok && bs.NeedsBrowser() {
			browserScrapers = append(browserScrapers, indexedScraper{i, s})
		} else {
			httpScrapers = append(httpScrapers, indexedScraper{i, s})
		}
	}

	o.logger.Info("Scraper classification",
		slog.Int("http_scrapers", len(httpScrapers)),
		slog.Int("browser_scrapers", len(browserScrapers)),
	)

	// Process HTTP scrapers in parallel (simple goroutines)
	for _, is := range httpScrapers {
		wg.Add(1)
		go func(idx int, scraper bankScraperInterface) {
			defer wg.Done()
			result := o.scrapeBank(ctx, scraper)
			resultsCh <- indexedResult{idx, result}
		}(is.index, is.scraper)
	}

	// Process browser scrapers using pool (if any)
	if len(browserScrapers) > 0 {
		poolCfg := browser.DefaultPoolConfig()
		poolCfg.MaxPages = o.config.MaxBrowserPages
		poolCfg.PageTimeout = o.config.RequestTimeout

		browserPool, err := browser.NewPool(poolCfg, o.logger)
		if err != nil {
			o.logger.Error("Failed to create browser pool", slog.String("error", err.Error()))
			// Fall back to HTTP scraping for browser scrapers
			for _, is := range browserScrapers {
				wg.Add(1)
				go func(idx int, scraper bankScraperInterface) {
					defer wg.Done()
					result := o.scrapeBank(ctx, scraper)
					resultsCh <- indexedResult{idx, result}
				}(is.index, is.scraper)
			}
		} else {
			defer browserPool.Close()

			// Use semaphore to limit concurrent browser scrapes
			browserSem := make(chan struct{}, o.config.MaxBrowserPages)
			for _, is := range browserScrapers {
				wg.Add(1)
				go func(idx int, scraper bankScraperInterface) {
					defer wg.Done()

					// Acquire semaphore
					select {
					case browserSem <- struct{}{}:
						defer func() { <-browserSem }()
					case <-ctx.Done():
						resultsCh <- indexedResult{idx, ScrapeResult{
							BankCode: scraper.BankCode(),
							BankName: scraper.BankName(),
							Success:  false,
							Error:    ctx.Err(),
						}}
						return
					}

					// Acquire page from pool
					page, err := browserPool.Acquire(ctx)
					if err != nil {
						resultsCh <- indexedResult{idx, ScrapeResult{
							BankCode: scraper.BankCode(),
							BankName: scraper.BankName(),
							Success:  false,
							Error:    err,
						}}
						return
					}
					defer browserPool.Release(page)

					result := o.scrapeBankWithBrowser(ctx, scraper, page)
					resultsCh <- indexedResult{idx, result}
				}(is.index, is.scraper)
			}
		}
	}

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for ir := range resultsCh {
		results[ir.index] = ir.result
	}

	o.metrics.FinishRun()

	// Log summary
	var successCount, failCount, totalRates int
	for _, r := range results {
		if r.Success {
			successCount++
			totalRates += r.RatesScraped
		} else {
			failCount++
		}
	}

	o.logger.Info("Parallel scrape completed",
		slog.Int("successful", successCount),
		slog.Int("failed", failCount),
		slog.Int("total_rates", totalRates),
	)

	return results, nil
}

// ScrapeBank scrapes rates from a specific bank
func (o *Orchestrator) ScrapeBank(ctx context.Context, bankCode string) (ScrapeResult, error) {
	for _, scraper := range o.scrapers {
		if scraper.BankCode() == bankCode {
			return o.scrapeBank(ctx, scraper), nil
		}
	}
	return ScrapeResult{}, fmt.Errorf("no scraper found for bank: %s", bankCode)
}

// GetAllRates returns all scraped rates from the most recent successful scrape
func (o *Orchestrator) GetAllRates(ctx context.Context) ([]model.InterestRate, error) {
	results, err := o.ScrapeAll(ctx)
	if err != nil {
		return nil, err
	}

	var allRates []model.InterestRate
	for _, result := range results {
		if result.Success {
			allRates = append(allRates, result.Rates...)
		}
	}

	return allRates, nil
}

// GetMetrics returns the metrics collector
func (o *Orchestrator) GetMetrics() *MetricsCollector {
	return o.metrics
}

// GetHealthStatus returns the current health status
func (o *Orchestrator) GetHealthStatus(nextRunTime time.Time) HealthStatus {
	return o.metrics.GetHealthStatus(nextRunTime, len(o.scrapers))
}

// GetBankCount returns the number of configured bank scrapers
func (o *Orchestrator) GetBankCount() int {
	return len(o.scrapers)
}

// randomDelay returns a random delay between MinDelay and MaxDelay
func (o *Orchestrator) randomDelay() time.Duration {
	if o.config.MaxDelay <= o.config.MinDelay {
		return o.config.MinDelay
	}
	diff := o.config.MaxDelay - o.config.MinDelay
	jitter := time.Duration(rand.Int63n(int64(diff)))
	return o.config.MinDelay + jitter
}
