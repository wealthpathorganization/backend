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
	tpbankBankCode = "tpbank"
	tpbankBankName = "TPBank"
	tpbankRateURL  = "https://tpb.vn/cong-cu-tinh-toan/lai-suat"
)

// TPBankScraper scrapes interest rates from TPBank
type TPBankScraper struct {
	BrowserBaseScraper
}

// NewTPBankScraper creates a new TPBank scraper
func NewTPBankScraper(client *http.Client) *TPBankScraper {
	return &TPBankScraper{
		BrowserBaseScraper: BrowserBaseScraper{
			BaseScraper: BaseScraper{
				Client:    client,
				BankCode_: tpbankBankCode,
				BankName_: tpbankBankName,
				RateURL:   tpbankRateURL,
			},
			needsBrowser: true,
		},
	}
}

// ScrapeRates scrapes interest rates from TPBank using HTTP (fallback)
func (s *TPBankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	rates := s.parseDepositRates(doc)
	return rates, nil
}

// ScrapeWithBrowser scrapes interest rates from TPBank using headless browser
func (s *TPBankScraper) ScrapeWithBrowser(ctx context.Context, page *rod.Page) ([]model.InterestRate, error) {
	// Navigate to the rate page
	if err := page.Navigate(s.RateURL); err != nil {
		return nil, err
	}

	// Wait for page to fully load (IBM WebSphere portal needs JS rendering)
	if err := page.WaitLoad(); err != nil {
		return nil, err
	}

	// Wait for network to be idle to ensure AJAX content is loaded
	_ = page.WaitRequestIdle(2*time.Second, nil, nil, nil)

	// Wait for rate table to appear - try common table selectors
	tableSelectors := []string{
		"table.interest-rate",
		"table.rate-table",
		".interest-rate table",
		".lai-suat table",
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
		// Give extra time for dynamic content
		time.Sleep(2 * time.Second)
	}

	// Parse HTML with goquery
	doc, err := ParseHTMLFromPage(page)
	if err != nil {
		return nil, err
	}

	return s.parseDepositRates(doc), nil
}

// parseDepositRates parses deposit rates from TPBank page
func (s *TPBankScraper) parseDepositRates(doc *goquery.Document) []model.InterestRate {
	var rates []model.InterestRate
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	doc.Find("table").Each(func(i int, table *goquery.Selection) {
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
					tpbankBankCode, tpbankBankName, "deposit",
					termMonths, termLabel, rate,
					now, effectiveDate,
				))
			})
		})
	})

	return rates
}
