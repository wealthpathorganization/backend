package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/wealthpath/backend/internal/model" // swagger types
	"github.com/wealthpath/backend/internal/service"
)

type RecurringHandler struct {
	recurringService RecurringServiceInterface
}

func NewRecurringHandler(recurringService RecurringServiceInterface) *RecurringHandler {
	return &RecurringHandler{recurringService: recurringService}
}

// Create godoc
// @Summary Create a recurring transaction
// @Description Create a new recurring transaction (income or expense)
// @Tags recurring
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateRecurringInput true "Recurring transaction data"
// @Success 201 {object} model.RecurringTransaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /recurring [post]
func (h *RecurringHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input service.CreateRecurringInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rt, err := h.recurringService.Create(r.Context(), userID, input)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, rt)
}

// List godoc
// @Summary List recurring transactions
// @Description Get all recurring transactions for the current user
// @Tags recurring
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.RecurringTransaction
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /recurring [get]
func (h *RecurringHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.recurringService.GetByUserID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get recurring transactions")
		return
	}

	respondJSON(w, http.StatusOK, items)
}

// Get godoc
// @Summary Get a recurring transaction
// @Description Get a recurring transaction by ID
// @Tags recurring
// @Produce json
// @Security BearerAuth
// @Param id path string true "Recurring Transaction ID"
// @Success 200 {object} model.RecurringTransaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /recurring/{id} [get]
func (h *RecurringHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rt, err := h.recurringService.GetByID(r.Context(), userID, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "recurring transaction not found")
		return
	}

	respondJSON(w, http.StatusOK, rt)
}

// Update godoc
// @Summary Update a recurring transaction
// @Description Update an existing recurring transaction
// @Tags recurring
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Recurring Transaction ID"
// @Param input body service.UpdateRecurringInput true "Updated recurring transaction data"
// @Success 200 {object} model.RecurringTransaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /recurring/{id} [put]
func (h *RecurringHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input service.UpdateRecurringInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rt, err := h.recurringService.Update(r.Context(), userID, id, input)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, rt)
}

// Delete godoc
// @Summary Delete a recurring transaction
// @Description Delete a recurring transaction by ID
// @Tags recurring
// @Security BearerAuth
// @Param id path string true "Recurring Transaction ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /recurring/{id} [delete]
func (h *RecurringHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.recurringService.Delete(r.Context(), userID, id); err != nil {
		respondError(w, http.StatusNotFound, "recurring transaction not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Pause godoc
// @Summary Pause a recurring transaction
// @Description Pause a recurring transaction to stop automatic generation
// @Tags recurring
// @Produce json
// @Security BearerAuth
// @Param id path string true "Recurring Transaction ID"
// @Success 200 {object} model.RecurringTransaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /recurring/{id}/pause [post]
func (h *RecurringHandler) Pause(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rt, err := h.recurringService.Pause(r.Context(), userID, id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, rt)
}

// Resume godoc
// @Summary Resume a recurring transaction
// @Description Resume a paused recurring transaction
// @Tags recurring
// @Produce json
// @Security BearerAuth
// @Param id path string true "Recurring Transaction ID"
// @Success 200 {object} model.RecurringTransaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /recurring/{id}/resume [post]
func (h *RecurringHandler) Resume(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rt, err := h.recurringService.Resume(r.Context(), userID, id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, rt)
}

// Upcoming godoc
// @Summary Get upcoming recurring transactions
// @Description Get the next 10 upcoming recurring transactions
// @Tags recurring
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.RecurringTransaction
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /recurring/upcoming [get]
func (h *RecurringHandler) Upcoming(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.recurringService.GetUpcoming(r.Context(), userID, 10)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get upcoming items")
		return
	}

	respondJSON(w, http.StatusOK, items)
}
