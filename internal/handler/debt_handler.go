package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	_ "github.com/wealthpath/backend/internal/model" // swagger types
	"github.com/wealthpath/backend/internal/service"
)

type DebtHandler struct {
	service DebtServiceInterface
}

func NewDebtHandler(service DebtServiceInterface) *DebtHandler {
	return &DebtHandler{service: service}
}

// Create godoc
// @Summary Create a debt
// @Description Create a new debt entry
// @Tags debts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateDebtInput true "Debt data"
// @Success 201 {object} model.Debt
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /debts [post]
func (h *DebtHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	var input service.CreateDebtInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	debt, err := h.service.Create(r.Context(), userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create debt")
		return
	}

	respondJSON(w, http.StatusCreated, debt)
}

// Get godoc
// @Summary Get a debt
// @Description Get a debt by ID
// @Tags debts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Debt ID"
// @Success 200 {object} model.Debt
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /debts/{id} [get]
func (h *DebtHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	debt, err := h.service.Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "debt not found")
		return
	}

	respondJSON(w, http.StatusOK, debt)
}

// List godoc
// @Summary List debts
// @Description Get all debts for the current user
// @Tags debts
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.Debt
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /debts [get]
func (h *DebtHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	debts, err := h.service.List(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list debts")
		return
	}

	respondJSON(w, http.StatusOK, debts)
}

// Update godoc
// @Summary Update a debt
// @Description Update an existing debt
// @Tags debts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Debt ID"
// @Param input body service.UpdateDebtInput true "Updated debt data"
// @Success 200 {object} model.Debt
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /debts/{id} [put]
func (h *DebtHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input service.UpdateDebtInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	debt, err := h.service.Update(r.Context(), id, userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update debt")
		return
	}

	respondJSON(w, http.StatusOK, debt)
}

// Delete godoc
// @Summary Delete a debt
// @Description Delete a debt by ID
// @Tags debts
// @Security BearerAuth
// @Param id path string true "Debt ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /debts/{id} [delete]
func (h *DebtHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete debt")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MakePayment godoc
// @Summary Make a debt payment
// @Description Record a payment against a debt
// @Tags debts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Debt ID"
// @Param input body service.MakePaymentInput true "Payment data"
// @Success 200 {object} model.Debt
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /debts/{id}/payment [post]
func (h *DebtHandler) MakePayment(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input service.MakePaymentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	debt, err := h.service.MakePayment(r.Context(), id, userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to make payment")
		return
	}

	respondJSON(w, http.StatusOK, debt)
}

// GetPayoffPlan godoc
// @Summary Get debt payoff plan
// @Description Calculate a payoff plan for a debt
// @Tags debts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Debt ID"
// @Param monthlyPayment query number false "Monthly payment amount"
// @Success 200 {object} model.PayoffPlan
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /debts/{id}/payoff-plan [get]
func (h *DebtHandler) GetPayoffPlan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	monthlyPayment := decimal.Zero
	if mp := r.URL.Query().Get("monthlyPayment"); mp != "" {
		if p, err := decimal.NewFromString(mp); err == nil {
			monthlyPayment = p
		}
	}

	plan, err := h.service.GetPayoffPlan(r.Context(), id, monthlyPayment)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get payoff plan")
		return
	}

	respondJSON(w, http.StatusOK, plan)
}

// InterestCalculator godoc
// @Summary Calculate loan interest
// @Description Calculate interest and payment schedule for a loan
// @Tags debts
// @Produce json
// @Param principal query number true "Loan principal amount"
// @Param interestRate query number true "Annual interest rate (percentage)"
// @Param termMonths query int true "Loan term in months"
// @Param paymentType query string false "Payment type (fixed or interest_only)" default(fixed)
// @Success 200 {object} service.InterestCalculatorResult
// @Failure 400 {object} ErrorResponse
// @Router /debts/calculator [get]
func (h *DebtHandler) InterestCalculator(w http.ResponseWriter, r *http.Request) {
	var input service.InterestCalculatorInput

	principal := r.URL.Query().Get("principal")
	if p, err := decimal.NewFromString(principal); err == nil {
		input.Principal = p
	}

	interestRate := r.URL.Query().Get("interestRate")
	if ir, err := decimal.NewFromString(interestRate); err == nil {
		input.InterestRate = ir
	}

	termMonths := r.URL.Query().Get("termMonths")
	if tm, err := decimal.NewFromString(termMonths); err == nil {
		input.TermMonths = int(tm.IntPart())
	}

	input.PaymentType = r.URL.Query().Get("paymentType")
	if input.PaymentType == "" {
		input.PaymentType = "fixed"
	}

	result, err := h.service.CalculateInterest(input)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid calculator input")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
