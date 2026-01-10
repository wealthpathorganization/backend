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
	vpbankBankCode = "vpbank"
	vpbankBankName = "VPBank"
	vpbankRateURL  = "https://www.vpbank.com.vn/ty-gia#interest_rate"
)

// VPBankScraper scrapes interest rates from VPBank using headless browser
type VPBankScraper struct {
	BrowserBaseScraper
}

// NewVPBankScraper creates a new VPBank scraper
func NewVPBankScraper(client *http.Client) *VPBankScraper {
	return &VPBankScraper{
		BrowserBaseScraper: BrowserBaseScraper{
			BaseScraper: BaseScraper{
				Client:    client,
				BankCode_: vpbankBankCode,
				BankName_: vpbankBankName,
				RateURL:   vpbankRateURL,
			},
			needsBrowser: true,
		},
	}
}

// ScrapeRates scrapes interest rates from VPBank (fallback for non-browser scraping)
func (s *VPBankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	rates := s.parseDepositRates(doc)
	return rates, nil
}

// ScrapeWithBrowser scrapes interest rates using headless browser
// VPBank is a Next.js app that requires hydration before content is visible
func (s *VPBankScraper) ScrapeWithBrowser(ctx context.Context, page *rod.Page) ([]model.InterestRate, error) {
	// Navigate to the interest rate page
	if err := page.Navigate(s.RateURL); err != nil {
		return nil, err
	}

	// Wait for page to fully load
	if err := page.WaitLoad(); err != nil {
		return nil, err
	}

	// Wait for network to be idle (Next.js hydration may make API calls)
	_ = page.WaitRequestIdle(2*time.Second, nil, nil, nil)

	// Wait for rate table to appear after Next.js hydration
	// Try multiple selectors as the page structure may vary
	selectors := []string{
		"table",
		".interest-rate-table",
		"[class*='rate']",
		"[class*='deposit']",
		"[data-testid*='rate']",
	}

	var found bool
	for _, selector := range selectors {
		el, err := page.Timeout(10 * time.Second).Element(selector)
		if err == nil && el != nil {
			if err := el.WaitVisible(); err == nil {
				found = true
				break
			}
		}
	}

	if !found {
		// Give extra time for Next.js hydration to complete
		time.Sleep(3 * time.Second)
	}

	// Parse the HTML from the rendered page
	doc, err := ParseHTMLFromPage(page)
	if err != nil {
		return nil, err
	}

	return s.parseDepositRates(doc), nil
}

// parseDepositRates parses deposit rates from VPBank page
func (s *VPBankScraper) parseDepositRates(doc *goquery.Document) []model.InterestRate {
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
					vpbankBankCode, vpbankBankName, "deposit",
					termMonths, termLabel, rate,
					now, effectiveDate,
				))
			})
		})
	})

	return rates
}
