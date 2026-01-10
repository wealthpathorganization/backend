// Package scheduler provides cron-based job scheduling for the interest rate scraper.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wealthpath/backend/internal/service"
)

// Config holds the scheduler configuration
type Config struct {
	// Schedule is a cron expression for when to run the scraper (e.g., "0 * * * *" for hourly)
	Schedule string
	// Timeout is the maximum duration for a complete scrape cycle
	Timeout time.Duration
	// Enabled determines if the scheduler should run
	Enabled bool
}

// DefaultConfig returns the default scheduler configuration
func DefaultConfig() Config {
	return Config{
		Schedule: "0 * * * *", // Every hour at minute 0
		Timeout:  5 * time.Minute,
		Enabled:  true,
	}
}

// Scheduler manages scheduled scraping jobs
type Scheduler struct {
	cron        *cron.Cron
	rateService *service.InterestRateService
	config      Config
	logger      *slog.Logger
	entryID     cron.EntryID
}

// New creates a new Scheduler instance
func New(cfg Config, rateService *service.InterestRateService, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		rateService: rateService,
		config:      cfg,
		logger:      logger,
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	if !s.config.Enabled {
		s.logger.Info("Scheduler is disabled, skipping start")
		return nil
	}

	// Convert standard cron (5 fields) to cron with seconds (6 fields)
	// Add "0" at the beginning for seconds
	schedule := "0 " + s.config.Schedule

	entryID, err := s.cron.AddFunc(schedule, func() {
		s.runScrapeJob()
	})
	if err != nil {
		return err
	}

	s.entryID = entryID
	s.cron.Start()

	s.logger.Info("Scheduler started",
		slog.String("schedule", s.config.Schedule),
		slog.Duration("timeout", s.config.Timeout),
	)

	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() context.Context {
	s.logger.Info("Stopping scheduler...")
	return s.cron.Stop()
}

// RunNow triggers an immediate scrape job (useful for manual triggers)
func (s *Scheduler) RunNow() {
	go s.runScrapeJob()
}

// runScrapeJob executes the scraping job
func (s *Scheduler) runScrapeJob() {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	startTime := time.Now()
	s.logger.Info("Starting scheduled scrape job",
		slog.Time("start_time", startTime),
	)

	count, err := s.rateService.ScrapeAndUpdateRates(ctx)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Scrape job failed",
			slog.String("error", err.Error()),
			slog.Duration("duration", duration),
		)
		return
	}

	s.logger.Info("Scrape job completed successfully",
		slog.Int("rates_scraped", count),
		slog.Duration("duration", duration),
	)
}

// GetNextRunTime returns the next scheduled run time
func (s *Scheduler) GetNextRunTime() time.Time {
	if s.entryID == 0 {
		return time.Time{}
	}
	entry := s.cron.Entry(s.entryID)
	return entry.Next
}

// GetLastRunTime returns the last run time
func (s *Scheduler) GetLastRunTime() time.Time {
	if s.entryID == 0 {
		return time.Time{}
	}
	entry := s.cron.Entry(s.entryID)
	return entry.Prev
}

// IsRunning returns true if the scheduler is running
func (s *Scheduler) IsRunning() bool {
	return s.cron != nil && len(s.cron.Entries()) > 0
}
