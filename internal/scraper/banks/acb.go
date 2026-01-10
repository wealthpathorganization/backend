package banks

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/wealthpath/backend/internal/model"
)

const (
	acbBankCode = "acb"
	acbBankName = "ACB"
	acbRateURL  = "https://acb.com.vn/lai-suat-tien-gui"
)

// ACBScraper scrapes interest rates from ACB
type ACBScraper struct {
	BrowserBaseScraper
}

// NewACBScraper creates a new ACB scraper
func NewACBScraper(client *http.Client) *ACBScraper {
	return &ACBScraper{
		BrowserBaseScraper: BrowserBaseScraper{
			BaseScraper: BaseScraper{
				Client:    client,
				BankCode_: acbBankCode,
				BankName_: acbBankName,
				RateURL:   acbRateURL,
			},
			needsBrowser: true,
		},
	}
}

// ScrapeRates scrapes interest rates from ACB using HTTP (fallback)
func (s *ACBScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	rates := s.parseDepositRates(doc)
	return rates, nil
}

// ScrapeWithBrowser scrapes interest rates from ACB using headless browser
func (s *ACBScraper) ScrapeWithBrowser(ctx context.Context, page *rod.Page) ([]model.InterestRate, error) {
	// Navigate to the rate page
	if err := page.Navigate(s.RateURL); err != nil {
		return nil, err
	}

	// Wait for page to fully load
	if err := page.WaitLoad(); err != nil {
		return nil, err
	}

	// Wait for network to be idle to ensure AJAX/XHR content is loaded
	// ACB uses AJAX to load rate tables
	_ = page.WaitRequestIdle(3*time.Second, nil, nil, nil)

	// Wait for rate table to appear - try ACB-specific and common selectors
	tableSelectors := []string{
		".interest-rate-table",
		".rate-table",
		".lai-suat-table",
		"table.table-bordered",
		".wpthemeContentContainer table",
		"table",
	}

	var tableFound bool
	for _, selector := range tableSelectors {
		el, err := page.Timeout(5 * time.Second).Element(selector)
		if err == nil && el != nil {
			if err := el.WaitVisible(); err == nil {
				tableFound = true
				break
			}
		}
	}

	if !tableFound {
		// Give extra time for AJAX content to load
		time.Sleep(3 * time.Second)
	}

	// Parse HTML with goquery
	doc, err := ParseHTMLFromPage(page)
	if err != nil {
		return nil, err
	}

	return s.parseDepositRates(doc), nil
}

// parseDepositRates parses deposit rates from ACB page
func (s *ACBScraper) parseDepositRates(doc *goquery.Document) []model.InterestRate {
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
					acbBankCode, acbBankName, "deposit",
					termMonths, termLabel, rate,
					now, effectiveDate,
				))
			})
		})
	})

	return rates
}
