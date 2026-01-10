package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/wealthpath/backend/internal/scraper"
)

func main() {
	// Flags
	parallel := flag.Bool("parallel", true, "Run scrapers in parallel with browser pool")
	output := flag.String("output", "", "Output file for JSON results (default: stdout)")
	timeout := flag.Duration("timeout", 5*time.Minute, "Scrape timeout")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        Vietnamese Bank Interest Rate Scraper CLI             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Create logger
	logLevel := slog.LevelInfo
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	// Create orchestrator
	config := scraper.DefaultOrchestratorConfig()
	orch := scraper.NewOrchestrator(config, logger)

	fmt.Printf("Scraping %d Vietnamese banks...\n", orch.GetBankCount())
	fmt.Printf("Mode: %s\n", map[bool]string{true: "Parallel (with browser pool)", false: "Sequential"}[*parallel])
	fmt.Println()

	startTime := time.Now()

	// Run scraper
	var results []scraper.ScrapeResult
	var err error

	if *parallel {
		results, err = orch.ScrapeAllParallel(ctx)
	} else {
		results, err = orch.ScrapeAll(ctx)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(startTime)

	// Print summary
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("                    RESULTS (%.1fs elapsed)\n", elapsed.Seconds())
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	var totalRates int
	var successCount int

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("❌ %s (%s): %v\n", result.BankName, result.BankCode, result.Error)
		} else {
			successCount++
			totalRates += len(result.Rates)
			fmt.Printf("✅ %s (%s): %d rates\n", result.BankName, result.BankCode, len(result.Rates))
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("SUMMARY: %d/%d banks, %d rates, %.1fs\n", successCount, len(results), totalRates, elapsed.Seconds())
	fmt.Println("═══════════════════════════════════════════════════════════════")

	// Output JSON if requested
	if *output != "" {
		type RateOutput struct {
			BankCode  string  `json:"bank_code"`
			BankName  string  `json:"bank_name"`
			Term      string  `json:"term"`
			Rate      float64 `json:"rate"`
			RateType  string  `json:"rate_type"`
			ScrapedAt string  `json:"scraped_at"`
		}

		var allRates []RateOutput
		for _, result := range results {
			if result.Success {
				for _, r := range result.Rates {
					allRates = append(allRates, RateOutput{
						BankCode:  r.BankCode,
						BankName:  r.BankName,
						Term:      r.TermLabel,
						Rate:      r.Rate.InexactFloat64(),
						RateType:  r.ProductType,
						ScrapedAt: r.ScrapedAt.Format(time.RFC3339),
					})
				}
			}
		}

		data, _ := json.MarshalIndent(allRates, "", "  ")
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✅ Wrote %d rates to %s\n", len(allRates), *output)
	}
}
