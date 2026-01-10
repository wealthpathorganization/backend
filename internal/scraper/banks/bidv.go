package banks

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/wealthpath/backend/internal/model"
)

const (
	bidvBankCode = "bidv"
	bidvBankName = "BIDV"
	bidvRateURL  = "https://www.bidv.com.vn/vn/ca-nhan/san-pham-dich-vu/tien-gui/lai-suat"
)

// BIDVScraper scrapes interest rates from BIDV
type BIDVScraper struct {
	BaseScraper
}

// NewBIDVScraper creates a new BIDV scraper
func NewBIDVScraper(client *http.Client) *BIDVScraper {
	return &BIDVScraper{
		BaseScraper: BaseScraper{
			Client:    client,
			BankCode_: bidvBankCode,
			BankName_: bidvBankName,
			RateURL:   bidvRateURL,
		},
	}
}

// ScrapeRates scrapes interest rates from BIDV
func (s *BIDVScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	rates := s.parseRates(doc)
	return rates, nil
}

// parseRates parses the BIDV interest rate page
func (s *BIDVScraper) parseRates(doc *goquery.Document) []model.InterestRate {
	var rates []model.InterestRate
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Look for interest rate tables
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		table.Find("tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				return
			}

			termText := strings.TrimSpace(cells.First().Text())
			rateText := strings.TrimSpace(cells.Last().Text())

			termMonths, termLabel, err := ParseTermMonths(termText)
			if err != nil || termMonths <= 0 {
				return
			}

			rate, err := ParseRateFromString(rateText)
			if err != nil || rate <= 0 {
				return
			}

			rates = append(rates, CreateRate(
				bidvBankCode, bidvBankName, "deposit",
				termMonths, termLabel, rate,
				now, effectiveDate,
			))
		})
	})

	return rates
}

