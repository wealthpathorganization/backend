package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/wealthpath/backend/internal/model" // swagger types
	"github.com/wealthpath/backend/internal/service"
)

type SavingsGoalHandler struct {
	service SavingsGoalServiceInterface
}

func NewSavingsGoalHandler(service SavingsGoalServiceInterface) *SavingsGoalHandler {
	return &SavingsGoalHandler{service: service}
}

// Create godoc
// @Summary Create a savings goal
// @Description Create a new savings goal
// @Tags savings-goals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateSavingsGoalInput true "Savings goal data"
// @Success 201 {object} model.SavingsGoal
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /savings-goals [post]
func (h *SavingsGoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	var input service.CreateSavingsGoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	goal, err := h.service.Create(r.Context(), userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create savings goal")
		return
	}

	respondJSON(w, http.StatusCreated, goal)
}

// Get godoc
// @Summary Get a savings goal
// @Description Get a savings goal by ID
// @Tags savings-goals
// @Produce json
// @Security BearerAuth
// @Param id path string true "Savings Goal ID"
// @Success 200 {object} model.SavingsGoal
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /savings-goals/{id} [get]
func (h *SavingsGoalHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	goal, err := h.service.Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "savings goal not found")
		return
	}

	respondJSON(w, http.StatusOK, goal)
}

// List godoc
// @Summary List savings goals
// @Description Get all savings goals for the current user with projection data
// @Tags savings-goals
// @Produce json
// @Security BearerAuth
// @Success 200 {array} service.SavingsGoalWithProjection
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /savings-goals [get]
func (h *SavingsGoalHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	goals, err := h.service.ListWithProjections(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list savings goals")
		return
	}

	respondJSON(w, http.StatusOK, goals)
}

// Update godoc
// @Summary Update a savings goal
// @Description Update an existing savings goal
// @Tags savings-goals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Savings Goal ID"
// @Param input body service.UpdateSavingsGoalInput true "Updated savings goal data"
// @Success 200 {object} model.SavingsGoal
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /savings-goals/{id} [put]
func (h *SavingsGoalHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input service.UpdateSavingsGoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	goal, err := h.service.Update(r.Context(), id, userID, input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update savings goal")
		return
	}

	respondJSON(w, http.StatusOK, goal)
}

// Delete godoc
// @Summary Delete a savings goal
// @Description Delete a savings goal by ID
// @Tags savings-goals
// @Security BearerAuth
// @Param id path string true "Savings Goal ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /savings-goals/{id} [delete]
func (h *SavingsGoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete savings goal")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Contribute godoc
// @Summary Contribute to a savings goal
// @Description Add money to a savings goal
// @Tags savings-goals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Savings Goal ID"
// @Param input body service.ContributeInput true "Contribution amount"
// @Success 200 {object} model.SavingsGoal
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /savings-goals/{id}/contribute [post]
func (h *SavingsGoalHandler) Contribute(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input service.ContributeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	goal, err := h.service.Contribute(r.Context(), id, userID, input.Amount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to contribute")
		return
	}

	respondJSON(w, http.StatusOK, goal)
}
