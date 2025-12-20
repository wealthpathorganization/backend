package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	_ "github.com/wealthpath/backend/internal/model" // swagger types
	"github.com/wealthpath/backend/internal/service"
)

type DashboardHandler struct {
	service *service.DashboardService
}

func NewDashboardHandler(service *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// GetDashboard godoc
// @Summary Get dashboard data
// @Description Get aggregated financial data for the current month
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.DashboardData
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /dashboard [get]
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	data, err := h.service.GetDashboard(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get dashboard")
		return
	}

	respondJSON(w, http.StatusOK, data)
}

// GetMonthlyDashboard godoc
// @Summary Get monthly dashboard data
// @Description Get aggregated financial data for a specific month
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Param year path int true "Year (e.g., 2024)"
// @Param month path int true "Month (1-12)"
// @Success 200 {object} model.DashboardData
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /dashboard/monthly/{year}/{month} [get]
func (h *DashboardHandler) GetMonthlyDashboard(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year")
		return
	}

	month, err := strconv.Atoi(chi.URLParam(r, "month"))
	if err != nil || month < 1 || month > 12 {
		respondError(w, http.StatusBadRequest, "invalid month")
		return
	}

	data, err := h.service.GetMonthlyDashboard(r.Context(), userID, year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get dashboard")
		return
	}

	respondJSON(w, http.StatusOK, data)
}
