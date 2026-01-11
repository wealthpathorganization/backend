package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/service"
)

type PushServiceInterface interface {
	IsConfigured() bool
	GetVAPIDPublicKey() (string, error)
	Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string, userAgent *string) (*model.PushSubscription, error)
	Unsubscribe(ctx context.Context, userID uuid.UUID, endpoint string) error
	GetPreferences(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error)
	UpdatePreferences(ctx context.Context, prefs *model.NotificationPreferences) error
}

type PushHandler struct {
	service PushServiceInterface
}

func NewPushHandler(service PushServiceInterface) *PushHandler {
	return &PushHandler{service: service}
}

// GetVAPIDPublicKey returns the VAPID public key for push subscription
// @Summary Get VAPID public key
// @Tags notifications
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/notifications/vapid-public-key [get]
func (h *PushHandler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	key, err := h.service.GetVAPIDPublicKey()
	if err != nil {
		if errors.Is(err, service.ErrVAPIDNotConfigured) {
			respondError(w, http.StatusServiceUnavailable, "Push notifications not configured")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get VAPID key")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"publicKey": key})
}

type subscribeRequest struct {
	Endpoint  string  `json:"endpoint"`
	P256dh    string  `json:"p256dh"`
	Auth      string  `json:"auth"`
	UserAgent *string `json:"userAgent,omitempty"`
}

// Subscribe creates a new push subscription
// @Summary Subscribe to push notifications
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body subscribeRequest true "Subscription data"
// @Success 201 {object} model.PushSubscription
// @Router /api/notifications/subscribe [post]
func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Endpoint == "" || req.P256dh == "" || req.Auth == "" {
		respondError(w, http.StatusBadRequest, "endpoint, p256dh, and auth are required")
		return
	}

	sub, err := h.service.Subscribe(r.Context(), userID, req.Endpoint, req.P256dh, req.Auth, req.UserAgent)
	if err != nil {
		if errors.Is(err, service.ErrVAPIDNotConfigured) {
			respondError(w, http.StatusServiceUnavailable, "Push notifications not configured")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to subscribe")
		return
	}

	respondJSON(w, http.StatusCreated, sub)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

// Unsubscribe removes a push subscription
// @Summary Unsubscribe from push notifications
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body unsubscribeRequest true "Subscription endpoint"
// @Success 204
// @Router /api/notifications/unsubscribe [delete]
func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req unsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Endpoint == "" {
		respondError(w, http.StatusBadRequest, "endpoint is required")
		return
	}

	if err := h.service.Unsubscribe(r.Context(), userID, req.Endpoint); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to unsubscribe")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPreferences returns notification preferences for the current user
// @Summary Get notification preferences
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.NotificationPreferences
// @Router /api/notifications/preferences [get]
func (h *PushHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	prefs, err := h.service.GetPreferences(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get preferences")
		return
	}

	respondJSON(w, http.StatusOK, prefs)
}

type updatePreferencesRequest struct {
	BillRemindersEnabled   *bool `json:"billRemindersEnabled,omitempty"`
	BillReminderDaysBefore *int  `json:"billReminderDaysBefore,omitempty"`
	BudgetAlertsEnabled    *bool `json:"budgetAlertsEnabled,omitempty"`
	BudgetAlertThreshold   *int  `json:"budgetAlertThreshold,omitempty"`
	GoalMilestonesEnabled  *bool `json:"goalMilestonesEnabled,omitempty"`
	WeeklySummaryEnabled   *bool `json:"weeklySummaryEnabled,omitempty"`
}

// UpdatePreferences updates notification preferences for the current user
// @Summary Update notification preferences
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body updatePreferencesRequest true "Preferences to update"
// @Success 200 {object} model.NotificationPreferences
// @Router /api/notifications/preferences [put]
func (h *PushHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get current preferences
	prefs, err := h.service.GetPreferences(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get preferences")
		return
	}

	// Decode request
	var req updatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update only provided fields
	if req.BillRemindersEnabled != nil {
		prefs.BillRemindersEnabled = *req.BillRemindersEnabled
	}
	if req.BillReminderDaysBefore != nil {
		prefs.BillReminderDaysBefore = *req.BillReminderDaysBefore
	}
	if req.BudgetAlertsEnabled != nil {
		prefs.BudgetAlertsEnabled = *req.BudgetAlertsEnabled
	}
	if req.BudgetAlertThreshold != nil {
		prefs.BudgetAlertThreshold = *req.BudgetAlertThreshold
	}
	if req.GoalMilestonesEnabled != nil {
		prefs.GoalMilestonesEnabled = *req.GoalMilestonesEnabled
	}
	if req.WeeklySummaryEnabled != nil {
		prefs.WeeklySummaryEnabled = *req.WeeklySummaryEnabled
	}

	prefs.UserID = userID

	if err := h.service.UpdatePreferences(r.Context(), prefs); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update preferences")
		return
	}

	respondJSON(w, http.StatusOK, prefs)
}
