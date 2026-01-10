package scraper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

// Common scraper errors
var (
	ErrNetworkTimeout  = errors.New("network request timed out")
	ErrParsingFailed   = errors.New("failed to parse rate data")
	ErrInvalidResponse = errors.New("invalid response from bank")
	ErrRateLimited     = errors.New("rate limited by bank server")
	ErrBankUnavailable = errors.New("bank website unavailable")
	ErrNoDataFound     = errors.New("no interest rate data found")
)

// ScrapeError represents an error that occurred during scraping
type ScrapeError struct {
	BankCode  string
	Operation string
	Err       error
	Timestamp time.Time
}

func (e *ScrapeError) Error() string {
	return fmt.Sprintf("[%s] %s: %v at %s",
		e.BankCode, e.Operation, e.Err, e.Timestamp.Format(time.RFC3339))
}

func (e *ScrapeError) Unwrap() error {
	return e.Err
}

// NewScrapeError creates a new ScrapeError
func NewScrapeError(bankCode, operation string, err error) *ScrapeError {
	return &ScrapeError{
		BankCode:  bankCode,
		Operation: operation,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// WithRetry executes a function with exponential backoff retry logic
func WithRetry(ctx context.Context, cfg RetryConfig, logger *slog.Logger, fn func() error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err

			if logger != nil {
				logger.Warn("scrape attempt failed",
					slog.Int("attempt", attempt),
					slog.Int("max_attempts", cfg.MaxAttempts),
					slog.String("error", err.Error()),
				)
			}
		}

		// Don't wait after the last attempt
		if attempt < cfg.MaxAttempts {
			// Add jitter to prevent thundering herd
			jitter := time.Duration(rand.Int63n(int64(delay / 4)))
			waitTime := delay + jitter

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}

			// Increase delay for next attempt
			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return fmt.Errorf("all %d attempts failed: %w", cfg.MaxAttempts, lastErr)
}

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network timeouts and temporary errors should be retried
	if errors.Is(err, ErrNetworkTimeout) {
		return true
	}
	if errors.Is(err, ErrBankUnavailable) {
		return true
	}
	if errors.Is(err, ErrRateLimited) {
		return true
	}

	// Context errors should not be retried
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Parsing errors should not be retried (would get the same result)
	if errors.Is(err, ErrParsingFailed) {
		return false
	}
	if errors.Is(err, ErrNoDataFound) {
		return false
	}

	return false
}
