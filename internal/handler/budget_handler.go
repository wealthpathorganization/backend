package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/wealthpath/backend/internal/model" // swagger types
	"github.com/wealthpath/backend/internal/service"
)

type BudgetHandler struct {
	service BudgetServiceInterface
}

func NewBudgetHandler(service BudgetServiceInterface) *BudgetHandler {
	return &BudgetHandler{service: service}
}

// Create godoc
// @Summary Create a budget
// @Description Create a new budget for a category
// @Tags budgets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateBudgetInput true "Budget data"
// @Success 201 {object} model.Budget
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /budgets [post]
func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	var input service.CreateBudgetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	budget, err := h.service.Create(r.Context(), userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create budget")
		return
	}

	respondJSON(w, http.StatusCreated, budget)
}

// Get godoc
// @Summary Get a budget
// @Description Get a budget by ID
// @Tags budgets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Budget ID"
// @Success 200 {object} model.Budget
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /budgets/{id} [get]
func (h *BudgetHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	budget, err := h.service.Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "budget not found")
		return
	}

	respondJSON(w, http.StatusOK, budget)
}

// List godoc
// @Summary List budgets
// @Description Get all budgets with spent amounts for the current user
// @Tags budgets
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.BudgetWithSpent
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /budgets [get]
func (h *BudgetHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	budgets, err := h.service.ListWithSpent(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list budgets")
		return
	}

	respondJSON(w, http.StatusOK, budgets)
}

// Update godoc
// @Summary Update a budget
// @Description Update an existing budget
// @Tags budgets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Budget ID"
// @Param input body service.UpdateBudgetInput true "Updated budget data"
// @Success 200 {object} model.Budget
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /budgets/{id} [put]
func (h *BudgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input service.UpdateBudgetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	budget, err := h.service.Update(r.Context(), id, userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update budget")
		return
	}

	respondJSON(w, http.StatusOK, budget)
}

// Delete godoc
// @Summary Delete a budget
// @Description Delete a budget by ID
// @Tags budgets
// @Security BearerAuth
// @Param id path string true "Budget ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /budgets/{id} [delete]
func (h *BudgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete budget")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
