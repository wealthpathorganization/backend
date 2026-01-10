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
	tcbBankCode = "tcb"
	tcbBankName = "Techcombank"
	tcbRateURL  = "https://www.techcombank.com.vn/cong-cu-tien-ich/ty-gia-lai-suat"
)

// TechcombankScraper scrapes interest rates from Techcombank
type TechcombankScraper struct {
	BaseScraper
}

// NewTechcombankScraper creates a new Techcombank scraper
func NewTechcombankScraper(client *http.Client) *TechcombankScraper {
	return &TechcombankScraper{
		BaseScraper: BaseScraper{
			Client:    client,
			BankCode_: tcbBankCode,
			BankName_: tcbBankName,
			RateURL:   tcbRateURL,
		},
	}
}

// ScrapeRates scrapes interest rates from Techcombank
func (s *TechcombankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	doc, err := s.FetchPage(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	rates := s.parseDepositRates(doc)
	return rates, nil
}

// parseDepositRates parses deposit rates from the TCB page
func (s *TechcombankScraper) parseDepositRates(doc *goquery.Document) []model.InterestRate {
	var rates []model.InterestRate
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// TCB uses tables for interest rates
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
					tcbBankCode, tcbBankName, "deposit",
					termMonths, termLabel, rate,
					now, effectiveDate,
				))
			})
		})
	})

	return rates
}

