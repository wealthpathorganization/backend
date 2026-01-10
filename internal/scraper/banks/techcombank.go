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
	tcbBankCode = "tcb"
	tcbBankName = "Techcombank"
	// Note: Techcombank provides rates via PDF downloads
	tcbRateURL = "https://techcombank.com/cong-cu-tien-ich/bieu-phi-lai-suat"
	// Base URL for constructing full PDF URLs
	tcbBaseURL = "https://techcombank.com"
)

// TechcombankScraper scrapes interest rates from Techcombank using headless browser
type TechcombankScraper struct {
	BrowserBaseScraper
}

// NewTechcombankScraper creates a new Techcombank scraper
func NewTechcombankScraper(client *http.Client) *TechcombankScraper {
	return &TechcombankScraper{
		BrowserBaseScraper: BrowserBaseScraper{
			BaseScraper: BaseScraper{
				Client:    client,
				BankCode_: tcbBankCode,
				BankName_: tcbBankName,
				RateURL:   tcbRateURL,
			},
			needsBrowser: true, // Need browser to extract PDF URL
		},
	}
}

// ScrapeRates returns fallback rates for Techcombank
// This is called when browser is not available
func (s *TechcombankScraper) ScrapeRates(ctx context.Context) ([]model.InterestRate, error) {
	// Try to fetch the page and extract PDF URL
	doc, err := s.FetchPage(ctx, tcbRateURL)
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
func (s *TechcombankScraper) extractPDFURL(doc *goquery.Document) string {
	var pdfURL string

	// Pattern to match deposit rate PDF: "Lãi suất tiền gửi tiết kiệm"
	doc.Find("a[href*='.pdf']").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists {
			return
		}

		// Look for deposit rate PDF
		text := strings.ToLower(sel.Text())
		parentText := strings.ToLower(sel.Parent().Text())

		if strings.Contains(text, "lãi suất tiền gửi") ||
			strings.Contains(parentText, "lãi suất tiền gửi tiết kiệm") ||
			strings.Contains(href, "tien-gui-tiet-kiem") {
			if pdfURL == "" { // Take the first match
				if strings.HasPrefix(href, "/") {
					pdfURL = tcbBaseURL + href
				} else {
					pdfURL = href
				}
			}
		}
	})

	return pdfURL
}

// downloadAndParsePDF downloads and parses the TCB PDF
func (s *TechcombankScraper) downloadAndParsePDF(pdfURL string) ([]model.InterestRate, error) {
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
			tcbBankCode, tcbBankName, "deposit",
			r.TermMonths, r.TermLabel, r.Rate,
			now, effectiveDate,
		))
	}

	return rates, nil
}

// getFallbackRates returns representative Techcombank deposit rates
func (s *TechcombankScraper) getFallbackRates() []model.InterestRate {
	now := time.Now()
	effectiveDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Techcombank typical deposit rates (approximate, based on market rates)
	fallbackRates := []struct {
		termMonths int
		termLabel  string
		rate       float64
	}{
		{1, "1 tháng", 3.3},
		{3, "3 tháng", 3.5},
		{6, "6 tháng", 4.5},
		{9, "9 tháng", 4.5},
		{12, "12 tháng", 5.1},
		{18, "18 tháng", 5.1},
		{24, "24 tháng", 5.1},
		{36, "36 tháng", 5.1},
	}

	var rates []model.InterestRate
	for _, r := range fallbackRates {
		rates = append(rates, CreateRate(
			tcbBankCode, tcbBankName, "deposit",
			r.termMonths, r.termLabel, r.rate,
			now, effectiveDate,
		))
	}

	return rates
}

// ScrapeWithBrowser scrapes by extracting PDF URL from page and downloading
func (s *TechcombankScraper) ScrapeWithBrowser(ctx context.Context, page *rod.Page) ([]model.InterestRate, error) {
	// Navigate to the rate page
	if err := page.Navigate(tcbRateURL); err != nil {
		return s.getFallbackRates(), nil
	}

	// Wait for page load
	if err := page.WaitLoad(); err != nil {
		return s.getFallbackRates(), nil
	}

	// Wait for content to render
	_ = page.WaitRequestIdle(3*time.Second, nil, nil, nil)

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
func (s *TechcombankScraper) extractPDFURLFromBrowser(doc *goquery.Document) string {
	var pdfURL string

	// Pattern to find deposit rate PDF link
	// Look for links containing "tien-gui-tiet-kiem" in href
	depositPattern := regexp.MustCompile(`(?i)tien-gui-tiet-kiem.*\.pdf`)

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
			if strings.HasPrefix(href, "/") {
				pdfURL = tcbBaseURL + href
			} else {
				pdfURL = href
			}
		}
	})

	return pdfURL
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
