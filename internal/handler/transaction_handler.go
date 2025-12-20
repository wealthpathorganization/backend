// Package handler implements HTTP handlers for the WealthPath REST API.
// Each handler validates input, delegates to services, and formats responses.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/apperror"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/internal/service"
)

// TransactionHandlerServiceInterface defines the service contract for transaction operations.
// This interface enables dependency injection and testability.
type TransactionHandlerServiceInterface interface {
	Create(ctx context.Context, userID uuid.UUID, input service.CreateTransactionInput) (*model.Transaction, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
	List(ctx context.Context, userID uuid.UUID, input service.ListTransactionsInput) ([]model.Transaction, error)
	Update(ctx context.Context, id, userID uuid.UUID, input service.UpdateTransactionInput) (*model.Transaction, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// TransactionHandler handles HTTP requests for transaction operations.
type TransactionHandler struct {
	service TransactionHandlerServiceInterface
}

// NewTransactionHandler creates a new TransactionHandler with the given service.
func NewTransactionHandler(service TransactionHandlerServiceInterface) *TransactionHandler {
	return &TransactionHandler{service: service}
}

// Create godoc
// @Summary Create a transaction
// @Description Create a new income or expense transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.CreateTransactionInput true "Transaction data"
// @Success 201 {object} model.Transaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions [post]
func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	var input service.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondAppError(w, apperror.BadRequest("invalid request body: "+err.Error()))
		return
	}

	// Validate required fields
	if input.Type == "" {
		respondAppError(w, apperror.ValidationError("type", "type is required (income or expense)"))
		return
	}
	if input.Amount.IsZero() {
		respondAppError(w, apperror.ValidationError("amount", "amount is required and must be greater than 0"))
		return
	}
	if input.Category == "" {
		respondAppError(w, apperror.ValidationError("category", "category is required"))
		return
	}

	tx, err := h.service.Create(r.Context(), userID, input)
	if err != nil {
		respondAppError(w, apperror.Internal(err))
		return
	}

	respondJSON(w, http.StatusCreated, tx)
}

// Get godoc
// @Summary Get a transaction
// @Description Get a transaction by ID
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID"
// @Success 200 {object} model.Transaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/{id} [get]
func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid transaction ID"))
		return
	}

	tx, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			respondAppError(w, apperror.NotFound("transaction"))
			return
		}
		respondAppError(w, apperror.Internal(err))
		return
	}

	respondJSON(w, http.StatusOK, tx)
}

// List godoc
// @Summary List transactions
// @Description Get a list of transactions with optional filters
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(0)
// @Param pageSize query int false "Items per page" default(20)
// @Param type query string false "Filter by type (income or expense)"
// @Param category query string false "Filter by category"
// @Param startDate query string false "Filter by start date (YYYY-MM-DD)"
// @Param endDate query string false "Filter by end date (YYYY-MM-DD)"
// @Success 200 {array} model.Transaction
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions [get]
func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	input := service.ListTransactionsInput{
		Page:     0,
		PageSize: 20,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			input.Page = p
		}
	}
	if pageSize := r.URL.Query().Get("pageSize"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			input.PageSize = ps
		}
	}
	if txType := r.URL.Query().Get("type"); txType != "" {
		input.Type = &txType
	}
	if category := r.URL.Query().Get("category"); category != "" {
		input.Category = &category
	}
	if startDate := r.URL.Query().Get("startDate"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			input.StartDate = &t
		}
	}
	if endDate := r.URL.Query().Get("endDate"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			input.EndDate = &t
		}
	}

	transactions, err := h.service.List(r.Context(), userID, input)
	if err != nil {
		respondAppError(w, apperror.Internal(err))
		return
	}

	respondJSON(w, http.StatusOK, transactions)
}

// Update godoc
// @Summary Update a transaction
// @Description Update an existing transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID"
// @Param input body service.UpdateTransactionInput true "Updated transaction data"
// @Success 200 {object} model.Transaction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/{id} [put]
func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid transaction ID"))
		return
	}

	var input service.UpdateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondAppError(w, apperror.BadRequest("invalid request body: "+err.Error()))
		return
	}

	tx, err := h.service.Update(r.Context(), id, userID, input)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			respondAppError(w, apperror.NotFound("transaction"))
			return
		}
		respondAppError(w, apperror.Internal(err))
		return
	}

	respondJSON(w, http.StatusOK, tx)
}

// Delete godoc
// @Summary Delete a transaction
// @Description Delete a transaction by ID
// @Tags transactions
// @Security BearerAuth
// @Param id path string true "Transaction ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/{id} [delete]
func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid transaction ID"))
		return
	}

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			respondAppError(w, apperror.NotFound("transaction"))
			return
		}
		respondAppError(w, apperror.Internal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
