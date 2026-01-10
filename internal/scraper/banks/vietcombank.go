package banks

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"regexp"
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

// vcbRateData represents the JSON structure embedded in VCB page
type vcbRateData struct {
	Count       int           `json:"Count"`
	UpdatedDate string        `json:"UpdatedDate"`
	AccountType string        `json:"AccountType"`
	Data        []vcbRateItem `json:"Data"`
}

type vcbRateItem struct {
	TenorType    string   `json:"tenorType"`
	Tenor        string   `json:"tenor"`
	CurrencyCode string   `json:"currencyCode"`
	TenorDisplay string   `json:"tenorDisplay"`
	Rates        *float64 `json:"rates"`
}

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

	// VCB embeds rate data as JSON in a hidden input field
	jsonData := doc.Find("input#currentDataInterestRate").AttrOr("value", "")
	if jsonData == "" {
		return rates
	}

	// Unescape HTML entities in JSON
	jsonData = html.UnescapeString(jsonData)

	var rateData vcbRateData
	if err := json.Unmarshal([]byte(jsonData), &rateData); err != nil {
		return rates
	}

	// Parse term months from tenor string (e.g., "1-months" -> 1)
	termRegex := regexp.MustCompile(`(\d+)-?(months?|days?)`)

	for _, item := range rateData.Data {
		// Only process VND savings rates
		if item.CurrencyCode != "VND" || item.TenorType != "Savings" {
			continue
		}

		if item.Rates == nil || *item.Rates <= 0 {
			continue
		}

		// Parse term months
		matches := termRegex.FindStringSubmatch(strings.ToLower(item.Tenor))
		if len(matches) < 3 {
			continue
		}

		var termMonths int
		termValue := 0
		for _, c := range matches[1] {
			termValue = termValue*10 + int(c-'0')
		}

		unit := matches[2]
		if strings.HasPrefix(unit, "day") {
			// Skip day-based terms or convert (7 days ~ 0.25 months)
			continue
		}
		termMonths = termValue

		if termMonths <= 0 {
			continue
		}

		// Rate is in decimal form (0.021 = 2.1%)
		rate := *item.Rates * 100

		// Skip duplicate rates
		isDuplicate := false
		for _, r := range rates {
			if r.TermMonths == termMonths && r.Rate.InexactFloat64() == rate {
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}

		rates = append(rates, CreateRate(
			vcbBankCode, vcbBankName, "deposit",
			termMonths, item.TenorDisplay, rate,
			now, effectiveDate,
		))
	}

	return rates
}

