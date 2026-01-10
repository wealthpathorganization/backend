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
	tpbankBankCode = "tpbank"
	tpbankBankName = "TPBank"
	tpbankRateURL  = "https://tpb.vn/lai-suat"
)

// TPBankScraper scrapes interest rates from TPBank
type TPBankScraper struct {
	BaseScraper
}

// NewTPBankScraper creates a new TPBank scraper
func NewTPBankScraper(client *http.Client) *TPBankScraper {
	return &TPBankScraper{
		BaseScraper: BaseScraper{
			Client:    client,
			BankCode_: tpbankBankCode,
			BankName_: tpbankBankName,
			RateURL:   tpbankRateURL,
		},
	}
}

// ScrapeRates scrapes interest rates from TPBank
func (s *TPBankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	rates := s.parseDepositRates(doc)
	return rates, nil
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

