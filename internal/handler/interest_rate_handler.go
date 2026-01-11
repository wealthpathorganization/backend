package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/wealthpath/backend/internal/service"
)

// InterestRateHandler handles interest rate API requests
type InterestRateHandler struct {
	service *service.InterestRateService
}

// NewInterestRateHandler creates a new interest rate handler
func NewInterestRateHandler(svc *service.InterestRateService) *InterestRateHandler {
	return &InterestRateHandler{service: svc}
}

// ListRates godoc
// @Summary List interest rates
// @Description Get interest rates with optional filters
// @Tags interest-rates
// @Accept json
// @Produce json
// @Param type query string false "Product type (deposit, loan, mortgage)"
// @Param term query int false "Term in months"
// @Param bank query string false "Bank code (vcb, tcb, mb, etc.)"
// @Success 200 {array} model.InterestRate
// @Router /interest-rates [get]
func (h *InterestRateHandler) ListRates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	productType := r.URL.Query().Get("type")
	if productType == "" {
		productType = "deposit" // Default to deposit rates
	}

	var termMonths *int
	if termStr := r.URL.Query().Get("term"); termStr != "" {
		term, err := strconv.Atoi(termStr)
		if err == nil {
			termMonths = &term
		}
	}

	bankCode := r.URL.Query().Get("bank")

	rates, err := h.service.ListRates(ctx, productType, termMonths, bankCode)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch interest rates")
		return
	}

	respondJSON(w, http.StatusOK, rates)
}

// GetBestRates godoc
// @Summary Get best interest rates
// @Description Get top interest rates for a specific term
// @Tags interest-rates
// @Accept json
// @Produce json
// @Param type query string false "Product type (deposit, loan, mortgage)" default(deposit)
// @Param term query int true "Term in months"
// @Param limit query int false "Number of results" default(5)
// @Success 200 {array} model.InterestRate
// @Router /interest-rates/best [get]
func (h *InterestRateHandler) GetBestRates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	productType := r.URL.Query().Get("type")
	if productType == "" {
		productType = "deposit"
	}

	termStr := r.URL.Query().Get("term")
	if termStr == "" {
		respondError(w, http.StatusBadRequest, "term parameter is required")
		return
	}

	termMonths, err := strconv.Atoi(termStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid term parameter")
		return
	}

	limit := 5
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	rates, err := h.service.GetBestRates(ctx, productType, termMonths, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch best rates")
		return
	}

	respondJSON(w, http.StatusOK, rates)
}

// CompareRates godoc
// @Summary Compare rates across banks
// @Description Get rates from all banks for a specific term
// @Tags interest-rates
// @Accept json
// @Produce json
// @Param type query string false "Product type (deposit, loan, mortgage)" default(deposit)
// @Param term query int true "Term in months"
// @Success 200 {array} model.InterestRate
// @Router /interest-rates/compare [get]
func (h *InterestRateHandler) CompareRates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	productType := r.URL.Query().Get("type")
	if productType == "" {
		productType = "deposit"
	}

	termStr := r.URL.Query().Get("term")
	if termStr == "" {
		respondError(w, http.StatusBadRequest, "term parameter is required")
		return
	}

	termMonths, err := strconv.Atoi(termStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid term parameter")
		return
	}

	rates, err := h.service.CompareRates(ctx, productType, termMonths)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to compare rates")
		return
	}

	respondJSON(w, http.StatusOK, rates)
}

// GetBanks godoc
// @Summary Get list of supported banks
// @Description Get list of Vietnamese banks with interest rate data
// @Tags interest-rates
// @Accept json
// @Produce json
// @Success 200 {array} model.Bank
// @Router /interest-rates/banks [get]
func (h *InterestRateHandler) GetBanks(w http.ResponseWriter, r *http.Request) {
	banks := h.service.GetBanks()
	respondJSON(w, http.StatusOK, banks)
}

// GetHistory godoc
// @Summary Get historical interest rates
// @Description Get historical rate data for charting
// @Tags interest-rates
// @Accept json
// @Produce json
// @Param bank query string true "Bank code (vcb, tcb, etc.)"
// @Param type query string false "Product type (deposit, loan, mortgage)" default(deposit)
// @Param term query int true "Term in months"
// @Param days query int false "Number of days of history" default(90)
// @Success 200 {array} repository.RateHistoryEntry
// @Router /interest-rates/history [get]
func (h *InterestRateHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bankCode := r.URL.Query().Get("bank")
	if bankCode == "" {
		respondError(w, http.StatusBadRequest, "bank parameter is required")
		return
	}

	productType := r.URL.Query().Get("type")
	if productType == "" {
		productType = "deposit"
	}

	termStr := r.URL.Query().Get("term")
	if termStr == "" {
		respondError(w, http.StatusBadRequest, "term parameter is required")
		return
	}

	termMonths, err := strconv.Atoi(termStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid term parameter")
		return
	}

	days := 90
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	history, err := h.service.GetRateHistory(ctx, bankCode, productType, termMonths, days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch rate history")
		return
	}

	respondJSON(w, http.StatusOK, history)
}

// SeedRates godoc
// @Summary Seed default interest rates
// @Description Populate database with sample interest rates (admin only)
// @Tags interest-rates
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /interest-rates/seed [post]
func (h *InterestRateHandler) SeedRates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.service.SeedDefaultRates(ctx); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to seed rates: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Interest rates seeded successfully"})
}

// ScrapeRates godoc
// @Summary Scrape live interest rates from banks
// @Description Scrape and update interest rates from Vietnamese banks
// @Tags interest-rates
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /interest-rates/scrape [post]
func (h *InterestRateHandler) ScrapeRates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	count, err := h.service.ScrapeAndUpdateRates(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to scrape rates: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Interest rates scraped successfully",
		"count":   count,
	})
}

// GetScraperHealth godoc
// @Summary Get scraper health status
// @Description Get the health status of the interest rate scraper
// @Tags interest-rates
// @Accept json
// @Produce json
// @Success 200 {object} scraper.HealthStatus
// @Router /interest-rates/scraper-health [get]
func (h *InterestRateHandler) GetScraperHealth(w http.ResponseWriter, r *http.Request) {
	// For now, use zero time for next run - can be enhanced to pass scheduler info
	health := h.service.GetScraperHealth(time.Time{})
	respondJSON(w, http.StatusOK, health)
}
