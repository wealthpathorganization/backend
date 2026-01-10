package banks

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/wealthpath/backend/internal/model"
)

const (
	bidvBankCode = "bidv"
	bidvBankName = "BIDV"
	bidvRateURL  = "https://bidv.com.vn/ServicesBIDV/InterestDetailServlet"
)

// bidvAPIResponse represents the JSON response from BIDV API
type bidvAPIResponse struct {
	HaNoi struct {
		Data []bidvRateItem `json:"data"`
	} `json:"hanoi"`
	Status int `json:"status"`
}

type bidvRateItem struct {
	TitleVI string `json:"title_vi"`
	TitleEN string `json:"title_en"`
	VND     string `json:"VND"`
	USD     string `json:"USD"`
}

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

// ScrapeRates scrapes interest rates from BIDV via JSON API
func (s *BIDVScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	body, err := s.FetchJSON(ctx, s.RateURL)
	if err != nil {
		return nil, err
	}

	var apiResp bidvAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	rates := s.parseRates(apiResp)
	return rates, nil
}

// parseRates parses the BIDV API response
func (s *BIDVScraper) parseRates(apiResp bidvAPIResponse) []model.InterestRate {
	var rates []model.InterestRate
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Parse term months from Vietnamese title (e.g., "1 Tháng" -> 1)
	termRegex := regexp.MustCompile(`(\d+)\s*(Tháng|tháng)`)

	for _, item := range apiResp.HaNoi.Data {
		// Skip if no VND rate
		if item.VND == "" {
			continue
		}

		// Parse rate value
		rate, err := ParseRateFromString(item.VND)
		if err != nil || rate <= 0 {
			continue
		}

		// Parse term months
		var termMonths int
		var termLabel string

		matches := termRegex.FindStringSubmatch(item.TitleVI)
		if len(matches) >= 2 {
			termMonths, _ = strconv.Atoi(matches[1])
			termLabel = item.TitleVI
		} else if item.TitleVI == "Không kỳ hạn" {
			// Skip "Không kỳ hạn" (no term) - very low rate, not useful
			continue
		}

		if termMonths <= 0 {
			continue
		}

		rates = append(rates, CreateRate(
			bidvBankCode, bidvBankName, "deposit",
			termMonths, termLabel, rate,
			now, effectiveDate,
		))
	}

	return rates
}
