// Package banks provides bank-specific interest rate scrapers.
package banks

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
)

// Common user agents for rotation
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
}

// BaseScraper provides common functionality for all bank scrapers
type BaseScraper struct {
	Client    *http.Client
	BankCode_ string
	BankName_ string
	RateURL   string
}

// BankCode returns the bank code
func (b *BaseScraper) BankCode() string { return b.BankCode_ }

// BankName returns the bank name
func (b *BaseScraper) BankName() string { return b.BankName_ }

// GetRandomUserAgent returns a random user agent
func GetRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// FetchPage fetches a web page and returns a goquery document
func (b *BaseScraper) FetchPage(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", GetRandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	return doc, nil
}

// FetchJSON fetches JSON from a URL and returns the response body
func (b *BaseScraper) FetchJSON(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", GetRandomUserAgent())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching JSON: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return body, nil
}

// CreateRate is a helper to create a standard rate entry
func CreateRate(bankCode, bankName, productType string, termMonths int, termLabel string, rate float64, now, effectiveDate time.Time) model.InterestRate {
	return model.InterestRate{
		BankCode:      bankCode,
		BankName:      bankName,
		ProductType:   productType,
		TermMonths:    termMonths,
		TermLabel:     termLabel,
		Rate:          decimal.NewFromFloat(rate),
		Currency:      "VND",
		EffectiveDate: effectiveDate,
		ScrapedAt:     now,
	}
}

// ParseRateFromString parses an interest rate from a string
func ParseRateFromString(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.ReplaceAll(s, " ", "")

	// Handle Vietnamese decimal separator
	re := regexp.MustCompile(`[\d.]+`)
	matches := re.FindString(s)
	if matches == "" {
		return 0, fmt.Errorf("no number found in: %s", s)
	}

	rate, err := strconv.ParseFloat(matches, 64)
	if err != nil {
		return 0, err
	}

	// Validate rate is in reasonable range (0-30%)
	if rate < 0 || rate > 30 {
		return 0, fmt.Errorf("rate out of range: %f", rate)
	}

	return rate, nil
}

// ParseTermMonths parses a term string and returns months
func ParseTermMonths(s string) (int, string, error) {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	// Common patterns
	patterns := []struct {
		regex    *regexp.Regexp
		multiply int
	}{
		{regexp.MustCompile(`(\d+)\s*(tháng|thang|month|months)`), 1},
		{regexp.MustCompile(`(\d+)\s*(năm|nam|year|years)`), 12},
		{regexp.MustCompile(`(\d+)\s*(tuần|tuan|week|weeks)`), 0}, // Less than a month
		{regexp.MustCompile(`(\d+)\s*(ngày|ngay|day|days)`), 0},   // Less than a month
		{regexp.MustCompile(`không\s*kỳ\s*hạn|kkh`), 0},           // No term (flexible)
	}

	for _, p := range patterns {
		matches := p.regex.FindStringSubmatch(s)
		if len(matches) >= 2 {
			if p.multiply == 0 {
				// Special cases
				if strings.Contains(s, "tuần") || strings.Contains(s, "week") {
					num, _ := strconv.Atoi(matches[1])
					return 0, fmt.Sprintf("%d tuần", num), nil
				}
				if strings.Contains(s, "ngày") || strings.Contains(s, "day") {
					num, _ := strconv.Atoi(matches[1])
					return 0, fmt.Sprintf("%d ngày", num), nil
				}
				return 0, "Không kỳ hạn", nil
			}
			num, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0, "", err
			}
			months := num * p.multiply
			var label string
			if p.multiply == 12 {
				label = fmt.Sprintf("%d năm", num)
			} else {
				label = fmt.Sprintf("%d tháng", months)
			}
			return months, label, nil
		}
	}

	// Try to extract just a number (assume months)
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 2 {
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, "", err
		}
		return num, fmt.Sprintf("%d tháng", num), nil
	}

	return 0, "", fmt.Errorf("could not parse term: %s", s)
}

// StandardTermLabels maps term months to Vietnamese labels
var StandardTermLabels = map[int]string{
	1:   "1 tháng",
	3:   "3 tháng",
	6:   "6 tháng",
	9:   "9 tháng",
	12:  "12 tháng",
	13:  "13 tháng",
	18:  "18 tháng",
	24:  "24 tháng",
	36:  "36 tháng",
	48:  "48 tháng",
	60:  "60 tháng",
	120: "10 năm",
	180: "15 năm",
	240: "20 năm",
	300: "25 năm",
	360: "30 năm",
}

// GetTermLabel returns the standard Vietnamese label for a term
func GetTermLabel(months int) string {
	if label, ok := StandardTermLabels[months]; ok {
		return label
	}
	if months >= 12 && months%12 == 0 {
		return fmt.Sprintf("%d năm", months/12)
	}
	return fmt.Sprintf("%d tháng", months)
}
