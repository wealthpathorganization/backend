package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/wealthpath/backend/internal/scraper"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create orchestrator with short timeout for testing
	cfg := scraper.OrchestratorConfig{
		MinDelay:       100 * time.Millisecond,
		MaxDelay:       200 * time.Millisecond,
		RequestTimeout: 10 * time.Second,
		RetryConfig: scraper.RetryConfig{
			MaxAttempts:  1,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
		},
	}

	orch := scraper.NewOrchestrator(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println("\n=== Starting Scrape Test ===\n")

	results, err := orch.ScrapeAll(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\n=== Scrape Results ===\n")

	var totalRates int
	for _, r := range results {
		status := "✓"
		if !r.Success {
			status = "✗"
		}
		fmt.Printf("%s %s (%s): %d rates in %v\n",
			status, r.BankName, r.BankCode, r.RatesScraped, r.Duration.Round(time.Millisecond))
		totalRates += r.RatesScraped

		// Show sample rates for first successful bank
		if r.Success && len(r.Rates) > 0 {
			fmt.Println("   Sample rates:")
			for i, rate := range r.Rates {
				if i >= 3 {
					fmt.Printf("   ... and %d more\n", len(r.Rates)-3)
					break
				}
				fmt.Printf("   - %s %s: %.2f%%\n", rate.ProductType, rate.TermLabel, rate.Rate.InexactFloat64())
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total banks: %d\n", len(results))
	fmt.Printf("Total rates: %d\n", totalRates)

	// Show health status
	health := orch.GetHealthStatus(time.Now().Add(time.Hour))
	healthJSON, _ := json.MarshalIndent(health, "", "  ")
	fmt.Printf("\n=== Health Status ===\n%s\n", healthJSON)
}
