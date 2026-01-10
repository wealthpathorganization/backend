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
	vcbBankCode = "vcb"
	vcbBankName = "Vietcombank"
	vcbRateURL  = "https://www.vietcombank.com.vn/vi-VN/KHCN/Cong-cu-Tien-ich/KHCN---Lai-suat"
)

// VietcombankScraper scrapes interest rates from Vietcombank
type VietcombankScraper struct {
	BaseScraper
}

// NewVietcombankScraper creates a new Vietcombank scraper
func NewVietcombankScraper(client *http.Client) *VietcombankScraper {
	return &VietcombankScraper{
		BaseScraper: BaseScraper{
			Client:    client,
			BankCode_: vcbBankCode,
			BankName_: vcbBankName,
			RateURL:   vcbRateURL,
		},
	}
}

// ScrapeRates scrapes interest rates from Vietcombank
func (s *VietcombankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	rates := s.parseDepositRates(doc)
	return rates, nil
}

// parseDepositRates parses deposit rates from the VCB page
func (s *VietcombankScraper) parseDepositRates(doc *goquery.Document) []model.InterestRate {
	var rates []model.InterestRate
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// VCB displays rates in tables with class "table-interest-rate" or similar
	// Look for tables containing interest rate data
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		// Check if this table contains interest rate headers
		headerText := table.Find("thead, th").Text()
		if !strings.Contains(strings.ToLower(headerText), "lãi suất") &&
			!strings.Contains(strings.ToLower(headerText), "kỳ hạn") {
			return
		}

		// Parse rows
		table.Find("tbody tr, tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				return
			}

			// First cell is usually the term
			termText := strings.TrimSpace(cells.First().Text())
			termMonths, termLabel, err := ParseTermMonths(termText)
			if err != nil || termMonths <= 0 {
				return
			}

			// Look for rate values in subsequent cells
			cells.Each(func(k int, cell *goquery.Selection) {
				if k == 0 {
					return // Skip term cell
				}

				rateText := strings.TrimSpace(cell.Text())
				rate, err := ParseRateFromString(rateText)
				if err != nil || rate <= 0 {
					return
				}

				// Skip duplicate rates
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
					vcbBankCode, vcbBankName, "deposit",
					termMonths, termLabel, rate,
					now, effectiveDate,
				))
			})
		})
	})

	return rates
}

