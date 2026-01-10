package pdf

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

// DownloadPDF downloads a PDF from URL to a temp file and returns the file path.
// The caller is responsible for removing the temp file.
func DownloadPDF(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "bank_rates_*.pdf")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("writing PDF: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// DownloadPDFWithClient downloads a PDF using a custom HTTP client
func DownloadPDFWithClient(client *http.Client, url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/pdf,*/*")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "bank_rates_*.pdf")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("writing PDF: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// ExtractText extracts all text from a PDF file
func ExtractText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening PDF: %w", err)
	}
	defer f.Close()

	var text strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text.WriteString(content)
		text.WriteString("\n")
	}

	return text.String(), nil
}

// RateInfo holds parsed rate information
type RateInfo struct {
	TermMonths int
	Rate       float64
	TermLabel  string
}

// ParseRateTable parses interest rate table from PDF text
// Returns a slice of RateInfo with term months, rate, and label
func ParseRateTable(text string) []RateInfo {
	var rates []RateInfo
	seenRates := make(map[string]bool) // Deduplication

	// Normalize text: collapse multiple spaces/tabs to single space
	normalizedText := normalizeVietnameseText(text)

	// Common Vietnamese term patterns - handle "01 tháng" format
	termPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\d{1,2})\s*(?:tháng|thang)`),
		regexp.MustCompile(`(?i)(\d{1,2})\s*month`),
		regexp.MustCompile(`(?i)(\d{1,2})T\b`), // 1T, 3T, 6T format
	}

	// Rate pattern: digits with decimal (comma or dot), optionally followed by %
	ratePattern := regexp.MustCompile(`(\d{1,2})[.,](\d{1,2})\s*%?`)

	lines := strings.Split(normalizedText, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to find term in the line
		var termMonths int

		for _, pattern := range termPatterns {
			match := pattern.FindStringSubmatch(line)
			if match != nil {
				months, err := strconv.Atoi(match[1])
				if err == nil && months > 0 && months <= 60 {
					termMonths = months
					break
				}
			}
		}

		if termMonths == 0 {
			continue
		}

		// Find rate in the same line first
		rateMatches := ratePattern.FindAllStringSubmatch(line, -1)
		foundRate := false

		// If no rate in same line, look in next few lines (table format)
		if len(rateMatches) == 0 {
			// Look at next 3 lines for rate
			for j := i + 1; j < len(lines) && j <= i+3; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					continue
				}
				rateMatches = ratePattern.FindAllStringSubmatch(nextLine, -1)
				if len(rateMatches) > 0 {
					break
				}
			}
		}

		for _, match := range rateMatches {
			if foundRate {
				break // Only take first valid rate per term
			}

			rateStr := match[1] + "." + match[2]
			rate, err := strconv.ParseFloat(rateStr, 64)
			if err != nil {
				continue
			}

			// Validate rate is reasonable (0.1% to 20%)
			if rate < 0.1 || rate > 20 {
				continue
			}

			// Deduplication key
			key := fmt.Sprintf("%d-%.2f", termMonths, rate)
			if seenRates[key] {
				continue
			}
			seenRates[key] = true

			rates = append(rates, RateInfo{
				TermMonths: termMonths,
				Rate:       rate,
				TermLabel:  normalizeTermLabel(termMonths),
			})
			foundRate = true
		}
	}

	return rates
}

// normalizeVietnameseText normalizes Vietnamese text from PDFs
// Handles cases where diacritics are separated or have extra spaces
func normalizeVietnameseText(text string) string {
	// Replace various space characters with regular space
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\u00A0", " ") // non-breaking space

	// Collapse multiple spaces to single space
	spacePattern := regexp.MustCompile(`[ ]+`)
	text = spacePattern.ReplaceAllString(text, " ")

	// Handle common Vietnamese word patterns where letters may be separated
	// Various representations of "tháng" that may appear in PDFs
	thangReplacements := map[string]string{
		"t h á n g":   "tháng",
		"t h a n g":   "tháng",
		"th á ng":     "tháng",
		"thá ng":      "tháng",
		"t háng":      "tháng",
		"tha\u0301ng": "tháng", // a + combining acute
		"th\u00E1ng":  "tháng", // precomposed á
	}
	for old, new := range thangReplacements {
		text = strings.ReplaceAll(text, old, new)
	}

	return text
}

// ParseRateTableAdvanced parses rate tables with more flexibility
// It can handle tables where term and rate are on separate lines
func ParseRateTableAdvanced(text string) []RateInfo {
	// First try the simple line-by-line approach
	rates := ParseRateTable(text)
	if len(rates) > 0 {
		return rates
	}

	// If that didn't work, try to extract all terms and rates separately
	// and match them based on position
	var allTerms []int
	var allRates []float64

	normalizedText := normalizeVietnameseText(text)

	termPattern := regexp.MustCompile(`(?i)(\d{1,2})\s*(?:tháng|thang|month|T\b)`)
	ratePattern := regexp.MustCompile(`(\d{1,2})[.,](\d{1,2})\s*%`)

	// Find all terms
	termMatches := termPattern.FindAllStringSubmatch(normalizedText, -1)
	for _, match := range termMatches {
		months, err := strconv.Atoi(match[1])
		if err == nil && months > 0 && months <= 60 {
			allTerms = append(allTerms, months)
		}
	}

	// Find all rates
	rateMatches := ratePattern.FindAllStringSubmatch(normalizedText, -1)
	for _, match := range rateMatches {
		rateStr := match[1] + "." + match[2]
		rate, err := strconv.ParseFloat(rateStr, 64)
		if err == nil && rate >= 0.1 && rate <= 20 {
			allRates = append(allRates, rate)
		}
	}

	// Try to pair them if counts match
	if len(allTerms) == len(allRates) && len(allTerms) > 0 {
		seenRates := make(map[string]bool)
		for i := 0; i < len(allTerms); i++ {
			key := fmt.Sprintf("%d-%.2f", allTerms[i], allRates[i])
			if seenRates[key] {
				continue
			}
			seenRates[key] = true

			rates = append(rates, RateInfo{
				TermMonths: allTerms[i],
				Rate:       allRates[i],
				TermLabel:  normalizeTermLabel(allTerms[i]),
			})
		}
	}

	return rates
}

// normalizeTermLabel returns a standardized Vietnamese term label
func normalizeTermLabel(months int) string {
	return fmt.Sprintf("%d tháng", months)
}

// GetTermLabel returns a Vietnamese term label for a given number of months
func GetTermLabel(months int) string {
	return normalizeTermLabel(months)
}
