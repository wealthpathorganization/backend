package scraper

import (
	"context"
	"sync"
	"time"
)

// ScrapeMetrics holds metrics for a single scrape operation
type ScrapeMetrics struct {
	BankCode     string
	StartedAt    time.Time
	CompletedAt  time.Time
	RatesScraped int
	Success      bool
	ErrorMessage string
	Duration     time.Duration
}

// MetricsCollector collects and aggregates scrape metrics
type MetricsCollector struct {
	mu            sync.RWMutex
	currentRun    map[string]*ScrapeMetrics
	lastRun       map[string]*ScrapeMetrics
	totalRuns     int
	successfulRuns int
	failedRuns    int
	lastRunTime   time.Time
}

// NewMetricsCollector creates a new MetricsCollector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		currentRun: make(map[string]*ScrapeMetrics),
		lastRun:    make(map[string]*ScrapeMetrics),
	}
}

// StartScrape records the start of a scrape operation for a bank
func (mc *MetricsCollector) StartScrape(bankCode string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.currentRun[bankCode] = &ScrapeMetrics{
		BankCode:  bankCode,
		StartedAt: time.Now(),
	}
}

// RecordSuccess records a successful scrape operation
func (mc *MetricsCollector) RecordSuccess(bankCode string, ratesScraped int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if metrics, ok := mc.currentRun[bankCode]; ok {
		metrics.CompletedAt = time.Now()
		metrics.Duration = metrics.CompletedAt.Sub(metrics.StartedAt)
		metrics.RatesScraped = ratesScraped
		metrics.Success = true
	}
}

// RecordFailure records a failed scrape operation
func (mc *MetricsCollector) RecordFailure(bankCode string, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if metrics, ok := mc.currentRun[bankCode]; ok {
		metrics.CompletedAt = time.Now()
		metrics.Duration = metrics.CompletedAt.Sub(metrics.StartedAt)
		metrics.Success = false
		if err != nil {
			metrics.ErrorMessage = err.Error()
		}
	}
}

// FinishRun marks the current run as complete and moves metrics to lastRun
func (mc *MetricsCollector) FinishRun() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Count successes and failures
	for _, metrics := range mc.currentRun {
		if metrics.Success {
			mc.successfulRuns++
		} else {
			mc.failedRuns++
		}
	}

	mc.totalRuns++
	mc.lastRunTime = time.Now()
	mc.lastRun = mc.currentRun
	mc.currentRun = make(map[string]*ScrapeMetrics)
}

// GetLastRunMetrics returns metrics from the last completed run
func (mc *MetricsCollector) GetLastRunMetrics() map[string]*ScrapeMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*ScrapeMetrics, len(mc.lastRun))
	for k, v := range mc.lastRun {
		metricsCopy := *v
		result[k] = &metricsCopy
	}
	return result
}

// GetCurrentRunMetrics returns metrics from the current run (in progress)
func (mc *MetricsCollector) GetCurrentRunMetrics() map[string]*ScrapeMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*ScrapeMetrics, len(mc.currentRun))
	for k, v := range mc.currentRun {
		metricsCopy := *v
		result[k] = &metricsCopy
	}
	return result
}

// GetSummary returns a summary of all scrape operations
func (mc *MetricsCollector) GetSummary() MetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var lastRunSuccesses, lastRunFailures int
	var totalDuration time.Duration
	var totalRatesScraped int

	for _, metrics := range mc.lastRun {
		if metrics.Success {
			lastRunSuccesses++
			totalRatesScraped += metrics.RatesScraped
		} else {
			lastRunFailures++
		}
		totalDuration += metrics.Duration
	}

	return MetricsSummary{
		TotalRuns:          mc.totalRuns,
		TotalSuccessful:    mc.successfulRuns,
		TotalFailed:        mc.failedRuns,
		LastRunTime:        mc.lastRunTime,
		LastRunSuccesses:   lastRunSuccesses,
		LastRunFailures:    lastRunFailures,
		LastRunDuration:    totalDuration,
		LastRunRatesScraped: totalRatesScraped,
	}
}

// MetricsSummary provides an overview of scraping performance
type MetricsSummary struct {
	TotalRuns           int
	TotalSuccessful     int
	TotalFailed         int
	LastRunTime         time.Time
	LastRunSuccesses    int
	LastRunFailures     int
	LastRunDuration     time.Duration
	LastRunRatesScraped int
}

// MetricsRepository defines the interface for persisting scrape metrics
type MetricsRepository interface {
	SaveMetrics(ctx context.Context, metrics *ScrapeMetrics) error
	GetMetricsByBank(ctx context.Context, bankCode string, limit int) ([]*ScrapeMetrics, error)
	GetRecentMetrics(ctx context.Context, limit int) ([]*ScrapeMetrics, error)
	GetSuccessRate(ctx context.Context, bankCode string, days int) (float64, error)
}

// HealthStatus represents the health of the scraper system
type HealthStatus struct {
	Healthy         bool              `json:"healthy"`
	LastRunTime     time.Time         `json:"last_run_time"`
	NextRunTime     time.Time         `json:"next_run_time"`
	TotalBanks      int               `json:"total_banks"`
	HealthyBanks    int               `json:"healthy_banks"`
	UnhealthyBanks  []string          `json:"unhealthy_banks,omitempty"`
	BankStatuses    map[string]string `json:"bank_statuses"`
	Message         string            `json:"message,omitempty"`
}

// GetHealthStatus returns the current health status of the scraper
func (mc *MetricsCollector) GetHealthStatus(nextRunTime time.Time, totalBanks int) HealthStatus {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	status := HealthStatus{
		LastRunTime:  mc.lastRunTime,
		NextRunTime:  nextRunTime,
		TotalBanks:   totalBanks,
		BankStatuses: make(map[string]string),
	}

	var healthyCount int
	var unhealthyBanks []string

	for bankCode, metrics := range mc.lastRun {
		if metrics.Success {
			healthyCount++
			status.BankStatuses[bankCode] = "healthy"
		} else {
			unhealthyBanks = append(unhealthyBanks, bankCode)
			status.BankStatuses[bankCode] = "unhealthy: " + metrics.ErrorMessage
		}
	}

	status.HealthyBanks = healthyCount
	status.UnhealthyBanks = unhealthyBanks

	// Consider healthy if at least 70% of banks are successful
	if totalBanks > 0 {
		successRate := float64(healthyCount) / float64(totalBanks)
		status.Healthy = successRate >= 0.7
	}

	if status.Healthy {
		status.Message = "Scraper is operating normally"
	} else if len(mc.lastRun) == 0 {
		status.Message = "No scrape runs recorded yet"
		status.Healthy = true // Consider healthy if no runs yet
	} else {
		status.Message = "Some banks are experiencing issues"
	}

	return status
}
