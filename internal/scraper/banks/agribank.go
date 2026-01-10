package banks

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/wealthpath/backend/internal/model"
)

const (
	agribankBankCode = "agribank"
	agribankBankName = "Agribank"
	agribankRateURL  = "https://www.agribank.com.vn/vn/lai-suat-tien-gui"
)

// AgribankScraper scrapes interest rates from Agribank
// Uses headless browser due to IBM WebSphere portal requiring JS rendering
type AgribankScraper struct {
	BrowserBaseScraper
}

// NewAgribankScraper creates a new Agribank scraper
func NewAgribankScraper(client *http.Client) *AgribankScraper {
	return &AgribankScraper{
		BrowserBaseScraper: BrowserBaseScraper{
			BaseScraper: BaseScraper{
				Client:    client,
				BankCode_: agribankBankCode,
				BankName_: agribankBankName,
				RateURL:   agribankRateURL,
			},
			needsBrowser: true,
		},
	}
}

// ScrapeRates scrapes interest rates from Agribank using HTTP (fallback)
func (s *AgribankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}

	rates := s.parseDepositRates(doc)
	if len(rates) == 0 {
		return nil, fmt.Errorf("no rates found - page may require browser rendering")
	}
	return rates, nil
}

// ScrapeWithBrowser scrapes interest rates using headless browser
func (s *AgribankScraper) ScrapeWithBrowser(ctx context.Context, page *rod.Page) ([]model.InterestRate, error) {
	// Navigate to the rate page
	if err := page.Navigate(s.RateURL); err != nil {
		return nil, fmt.Errorf("navigating to %s: %w", s.RateURL, err)
	}

	// Wait for page to load completely
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("waiting for page load: %w", err)
	}

	// Wait for network to be idle (IBM WebSphere loads content dynamically)
	if err := page.WaitRequestIdle(2*time.Second, nil, nil, nil); err != nil {
		// Not critical, continue as the table might already be loaded
	}

	// Wait for the rate table to appear
	// Try multiple selectors as the page structure may vary
	selectors := []string{
		"table",
		".lai-suat-table",
		"[class*='interest']",
		"[class*='rate']",
	}

	var tableFound bool
	for _, selector := range selectors {
		el, err := page.Timeout(5 * time.Second).Element(selector)
		if err == nil && el != nil {
			if err := el.WaitVisible(); err == nil {
				tableFound = true
				break
			}
		}
	}

	if !tableFound {
		// Give additional time for JS rendering
		time.Sleep(2 * time.Second)
	}

	// Parse HTML with goquery
	doc, err := ParseHTMLFromPage(page)
	if err != nil {
		return nil, fmt.Errorf("parsing page HTML: %w", err)
	}

	rates := s.parseDepositRates(doc)
	if len(rates) == 0 {
		return nil, fmt.Errorf("no rates found on page")
	}

	return rates, nil
}

// parseDepositRates parses deposit rates from Agribank page
func (s *AgribankScraper) parseDepositRates(doc *goquery.Document) []model.InterestRate {
	var rates []model.InterestRate
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		headerText := table.Find("thead, th").Text()
		if !strings.Contains(strings.ToLower(headerText), "lãi suất") &&
			!strings.Contains(strings.ToLower(headerText), "kỳ hạn") {
			return
		}

		table.Find("tbody tr, tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				return
			}

			termText := strings.TrimSpace(cells.First().Text())
			termMonths, termLabel, err := ParseTermMonths(termText)
			if err != nil || termMonths <= 0 {
				return
			}

			cells.Each(func(k int, cell *goquery.Selection) {
				if k == 0 {
					return
				}

				rateText := strings.TrimSpace(cell.Text())
				rate, err := ParseRateFromString(rateText)
				if err != nil || rate <= 0 {
					return
				}

				isDuplicate := false
				for _, r := range rates {
					if r.TermMonths == termMonths && r.Rate.InexactFloat64() == rate {
						isDuplicate = true
						break
					}
				}
				if isDuplicate {
					return
				}

				rates = append(rates, CreateRate(
					agribankBankCode, agribankBankName, "deposit",
					termMonths, termLabel, rate,
					now, effectiveDate,
				))
			})
		})
	})

	return rates
}
