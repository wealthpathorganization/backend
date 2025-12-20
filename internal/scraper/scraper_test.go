package scraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScraper(t *testing.T) {
	s := NewScraper()

	assert.NotNil(t, s)
	assert.NotNil(t, s.client)
	assert.GreaterOrEqual(t, len(s.scrapers), 7) // At least 7 bank scrapers
}

func TestScraper_ScrapeAll(t *testing.T) {
	s := NewScraper()
	ctx := context.Background()

	rates, err := s.ScrapeAll(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)

	// Should have rates from multiple banks
	bankCodes := make(map[string]bool)
	for _, rate := range rates {
		bankCodes[rate.BankCode] = true
	}
	assert.GreaterOrEqual(t, len(bankCodes), 5) // At least 5 different banks
}

func TestScraper_ScrapeBank(t *testing.T) {
	s := NewScraper()
	ctx := context.Background()

	t.Run("valid bank code", func(t *testing.T) {
		rates, err := s.ScrapeBank(ctx, "vcb")

		assert.NoError(t, err)
		assert.NotEmpty(t, rates)
		for _, rate := range rates {
			assert.Equal(t, "vcb", rate.BankCode)
			assert.Equal(t, "Vietcombank", rate.BankName)
		}
	})

	t.Run("invalid bank code", func(t *testing.T) {
		_, err := s.ScrapeBank(ctx, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no scraper found")
	})
}

func TestVietcombankScraper(t *testing.T) {
	s := NewVietcombankScraper(nil)
	ctx := context.Background()

	assert.Equal(t, "vcb", s.BankCode())
	assert.Equal(t, "Vietcombank", s.BankName())

	// Will return fallback rates since we don't have a real HTTP client
	rates, err := s.ScrapeRates(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)

	// Verify rate structure
	for _, rate := range rates {
		assert.Equal(t, "vcb", rate.BankCode)
		assert.Equal(t, "VND", rate.Currency)
		assert.NotEmpty(t, rate.ProductType)
		assert.NotZero(t, rate.Rate)
	}

	// Should have deposit rates
	hasDeposit := false
	for _, rate := range rates {
		if rate.ProductType == "deposit" {
			hasDeposit = true
			break
		}
	}
	assert.True(t, hasDeposit, "Should have deposit rates")
}

func TestTechcombankScraper(t *testing.T) {
	s := NewTechcombankScraper(nil)
	ctx := context.Background()

	assert.Equal(t, "tcb", s.BankCode())
	assert.Equal(t, "Techcombank", s.BankName())

	rates, err := s.ScrapeRates(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)

	for _, rate := range rates {
		assert.Equal(t, "tcb", rate.BankCode)
	}
}

func TestMBBankScraper(t *testing.T) {
	s := NewMBBankScraper(nil)
	ctx := context.Background()

	assert.Equal(t, "mb", s.BankCode())
	assert.Equal(t, "MB Bank", s.BankName())

	rates, err := s.ScrapeRates(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)

	for _, rate := range rates {
		assert.Equal(t, "mb", rate.BankCode)
	}
}

func TestBIDVScraper(t *testing.T) {
	s := NewBIDVScraper(nil)
	ctx := context.Background()

	assert.Equal(t, "bidv", s.BankCode())
	assert.Equal(t, "BIDV", s.BankName())

	rates, err := s.ScrapeRates(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)
}

func TestVPBankScraper(t *testing.T) {
	s := NewVPBankScraper(nil)
	ctx := context.Background()

	assert.Equal(t, "vpbank", s.BankCode())
	assert.Equal(t, "VPBank", s.BankName())

	rates, err := s.ScrapeRates(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)
}

func TestTPBankScraper(t *testing.T) {
	s := NewTPBankScraper(nil)
	ctx := context.Background()

	assert.Equal(t, "tpbank", s.BankCode())
	assert.Equal(t, "TPBank", s.BankName())

	rates, err := s.ScrapeRates(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)
}

func TestACBScraper(t *testing.T) {
	s := NewACBScraper(nil)
	ctx := context.Background()

	assert.Equal(t, "acb", s.BankCode())
	assert.Equal(t, "ACB", s.BankName())

	rates, err := s.ScrapeRates(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, rates)
}

func TestParseRateFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
		hasError bool
	}{
		{"simple percentage", "5.5%", 5.5, false},
		{"without percent", "4.7", 4.7, false},
		{"with spaces", "  3.5%  ", 3.5, false},
		{"comma decimal", "4,5%", 4.5, false},
		{"empty string", "", 0, true},
		{"no number", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRateFromString(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, result.InexactFloat64(), 0.01)
			}
		})
	}
}

func TestScraperRatesHaveLoanAndMortgage(t *testing.T) {
	s := NewScraper()
	ctx := context.Background()

	rates, err := s.ScrapeAll(ctx)
	assert.NoError(t, err)

	productTypes := make(map[string]int)
	for _, rate := range rates {
		productTypes[rate.ProductType]++
	}

	assert.Greater(t, productTypes["deposit"], 0, "Should have deposit rates")
	assert.Greater(t, productTypes["loan"], 0, "Should have loan rates")
	assert.Greater(t, productTypes["mortgage"], 0, "Should have mortgage rates")
}
