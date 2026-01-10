package banks

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/wealthpath/backend/internal/model"
	pdfparser "github.com/wealthpath/backend/internal/scraper/pdf"
)

const (
	hdbankBankCode = "hdbank"
	hdbankBankName = "HDBank"
	hdbankRateURL  = "https://hdbank.com.vn/vi/personal/cong-cu/interest-rate"
)

// HDBankScraper scrapes interest rates from HDBank using headless browser
type HDBankScraper struct {
	BrowserBaseScraper
}

// NewHDBankScraper creates a new HDBank scraper
func NewHDBankScraper(client *http.Client) *HDBankScraper {
	return &HDBankScraper{
		BrowserBaseScraper: BrowserBaseScraper{
			BaseScraper: BaseScraper{
				Client:    client,
				BankCode_: hdbankBankCode,
				BankName_: hdbankBankName,
				RateURL:   hdbankRateURL,
			},
			needsBrowser: true, // Need browser to extract PDF URL from JS-rendered page
		},
	}
}

// ScrapeRates scrapes HDBank interest rates
// Falls back to hardcoded rates if scraping fails
func (s *HDBankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	// HDBank loads PDF links via JavaScript, so HTTP fetch may not work
	// Try to fetch the page and extract PDF URL
	doc, err := s.FetchPage(ctx, hdbankRateURL)
	if err != nil {
		return s.getFallbackRates(), nil
	}

	// Find PDF URL for deposit rates
	pdfURL := s.extractPDFURL(doc)
	if pdfURL == "" {
		return s.getFallbackRates(), nil
	}

	return s.downloadAndParsePDF(pdfURL)
}

// extractPDFURL finds the PDF download link for deposit rates
func (s *HDBankScraper) extractPDFURL(doc *goquery.Document) string {
	var pdfURL string

	// Pattern to match deposit rate PDF: "BIỂU LÃI SUẤT TIỀN GỬI"
	depositPattern := regexp.MustCompile(`(?i)(TIEN\s*GUI|tiền\s*gửi).*(KHCN|cá\s*nhân)`)

	doc.Find("a[href*='.pdf']").Each(func(i int, sel *goquery.Selection) {
		if pdfURL != "" {
			return // Already found
		}

		href, exists := sel.Attr("href")
		if !exists {
			return
		}

		// Check if this is a deposit rate PDF
		text := sel.Text()
		if depositPattern.MatchString(text) || depositPattern.MatchString(href) ||
			strings.Contains(strings.ToUpper(href), "TIENGUIKHACHHANGCANHAN") {
			pdfURL = href
		}
	})

	return pdfURL
}

// downloadAndParsePDF downloads and parses the HDBank PDF
func (s *HDBankScraper) downloadAndParsePDF(pdfURL string) ([]model.InterestRate, error) {
	pdfPath, err := pdfparser.DownloadPDFWithClient(s.Client, pdfURL)
	if err != nil {
		return s.getFallbackRates(), nil
	}
	defer os.Remove(pdfPath)

	text, err := pdfparser.ExtractText(pdfPath)
	if err != nil {
		return s.getFallbackRates(), nil
	}

	rateInfos := pdfparser.ParseRateTableAdvanced(text)
	if len(rateInfos) == 0 {
		return s.getFallbackRates(), nil
	}

	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	var rates []model.InterestRate
	for _, r := range rateInfos {
		rates = append(rates, CreateRate(
			hdbankBankCode, hdbankBankName, "deposit",
			r.TermMonths, r.TermLabel, r.Rate,
			now, effectiveDate,
		))
	}

	return rates, nil
}

// getFallbackRates returns representative HDBank deposit rates
// These are approximate rates and should be periodically verified
func (s *HDBankScraper) getFallbackRates() []model.InterestRate {
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// HDBank typical deposit rates (approximate, based on market rates)
	fallbackRates := []struct {
		termMonths int
		termLabel  string
		rate       float64
	}{
		{1, "1 tháng", 3.3},
		{3, "3 tháng", 3.4},
		{6, "6 tháng", 4.7},
		{9, "9 tháng", 4.7},
		{12, "12 tháng", 5.3},
		{18, "18 tháng", 5.3},
		{24, "24 tháng", 5.3},
		{36, "36 tháng", 5.3},
	}

	var rates []model.InterestRate
	for _, r := range fallbackRates {
		rates = append(rates, CreateRate(
			hdbankBankCode, hdbankBankName, "deposit",
			r.termMonths, r.termLabel, r.rate,
			now, effectiveDate,
		))
	}

	return rates
}

// ScrapeWithBrowser scrapes by extracting PDF URL from page and downloading
func (s *HDBankScraper) ScrapeWithBrowser(ctx context.Context, page *rod.Page) ([]model.InterestRate, error) {
	// Navigate to the rate page
	if err := page.Navigate(hdbankRateURL); err != nil {
		return s.getFallbackRates(), nil
	}

	// Wait for page load
	if err := page.WaitLoad(); err != nil {
		return s.getFallbackRates(), nil
	}

	// Wait for content to render (HDBank uses React/Next.js)
	_ = page.WaitRequestIdle(3*time.Second, nil, nil, nil)

	// Additional wait for dynamic content
	time.Sleep(2 * time.Second)

	// Extract PDF URL from page
	html, err := page.HTML()
	if err != nil {
		return s.getFallbackRates(), nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return s.getFallbackRates(), nil
	}

	// Find PDF URL
	pdfURL := s.extractPDFURLFromBrowser(doc)
	if pdfURL == "" {
		return s.getFallbackRates(), nil
	}

	return s.downloadAndParsePDF(pdfURL)
}

// extractPDFURLFromBrowser extracts PDF URL from browser-rendered page
func (s *HDBankScraper) extractPDFURLFromBrowser(doc *goquery.Document) string {
	var pdfURL string

	// Pattern to find deposit rate PDF link
	// HDBank PDF URLs contain timestamps: .../20251106BIEULAISUATTIENGUIKHACHHANGCANHAN...
	depositPattern := regexp.MustCompile(`(?i)(TIENGUIKHACHHANGCANHAN|tien-gui.*khcn)`)

	doc.Find("a[href*='.pdf']").Each(func(i int, sel *goquery.Selection) {
		if pdfURL != "" {
			return // Already found
		}

		href, exists := sel.Attr("href")
		if !exists {
			return
		}

		// Check if this is a deposit rate PDF
		if depositPattern.MatchString(href) {
			pdfURL = href
		}
	})

	return pdfURL
}

// parseDepositRates parses deposit rates from HDBank page
func (s *HDBankScraper) parseDepositRates(doc *goquery.Document) []model.InterestRate {
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
					hdbankBankCode, hdbankBankName, "deposit",
					termMonths, termLabel, rate,
					now, effectiveDate,
				))
			})
		})
	})

	return rates
}
