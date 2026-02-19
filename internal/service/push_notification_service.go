package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/config"
	"github.com/wealthpath/backend/internal/model"
)

var (
	ErrVAPIDNotConfigured = errors.New("VAPID keys not configured")
	ErrNoSubscriptions    = errors.New("no push subscriptions found")
)

type PushRepositoryInterface interface {
	CreateSubscription(ctx context.Context, sub *model.PushSubscription) error
	GetSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]model.PushSubscription, error)
	GetAllActiveSubscriptions(ctx context.Context) ([]model.PushSubscription, error)
	DeleteSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error
	DeleteSubscriptionByEndpoint(ctx context.Context, endpoint string) error
	GetPreferences(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error)
	UpsertPreferences(ctx context.Context, prefs *model.NotificationPreferences) error
	LogNotification(ctx context.Context, log *model.NotificationLog) error
	HasRecentNotification(ctx context.Context, userID uuid.UUID, notifType model.NotificationType, refID *uuid.UUID, refDate *time.Time) (bool, error)
	// Mobile push token methods
	CreateMobileToken(ctx context.Context, token *model.MobilePushToken) error
	DeleteMobileToken(ctx context.Context, userID uuid.UUID, token string) error
}

type PushNotificationService struct {
	repo   PushRepositoryInterface
	config *config.Config
}

func NewPushNotificationService(repo PushRepositoryInterface, cfg *config.Config) *PushNotificationService {
	return &PushNotificationService{
		repo:   repo,
		config: cfg,
	}
}

// IsConfigured returns true if VAPID keys are configured
func (s *PushNotificationService) IsConfigured() bool {
	return s.config.VAPIDPublicKey != "" && s.config.VAPIDPrivateKey != ""
}

// GetVAPIDPublicKey returns the public VAPID key for clients
func (s *PushNotificationService) GetVAPIDPublicKey() (string, error) {
	if !s.IsConfigured() {
		return "", ErrVAPIDNotConfigured
	}
	return s.config.VAPIDPublicKey, nil
}

// Subscribe creates or updates a push subscription
func (s *PushNotificationService) Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string, userAgent *string) (*model.PushSubscription, error) {
	if !s.IsConfigured() {
		return nil, ErrVAPIDNotConfigured
	}

	sub := &model.PushSubscription{
		ID:        uuid.New(),
		UserID:    userID,
		Endpoint:  endpoint,
		P256dh:    p256dh,
		Auth:      auth,
		UserAgent: userAgent,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// Unsubscribe removes a push subscription
func (s *PushNotificationService) Unsubscribe(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return s.repo.DeleteSubscription(ctx, userID, endpoint)
}

// GetPreferences returns notification preferences for a user
func (s *PushNotificationService) GetPreferences(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		// Return default preferences if not found
		return &model.NotificationPreferences{
			UserID:                userID,
			BillRemindersEnabled:  true,
			BillReminderDaysBefore: 3,
			BudgetAlertsEnabled:   true,
			BudgetAlertThreshold:  90,
			GoalMilestonesEnabled: true,
			WeeklySummaryEnabled:  false,
		}, nil
	}
	return prefs, nil
}

// UpdatePreferences updates notification preferences for a user
func (s *PushNotificationService) UpdatePreferences(ctx context.Context, prefs *model.NotificationPreferences) error {
	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}
	return s.repo.UpsertPreferences(ctx, prefs)
}

// Notification payload for web push
type NotificationPayload struct {
	Title   string                 `json:"title"`
	Body    string                 `json:"body"`
	Icon    string                 `json:"icon,omitempty"`
	Badge   string                 `json:"badge,omitempty"`
	Tag     string                 `json:"tag,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Actions []NotificationAction   `json:"actions,omitempty"`
}

type NotificationAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// SendToUser sends a push notification to all of a user's subscribed devices
func (s *PushNotificationService) SendToUser(ctx context.Context, userID uuid.UUID, payload *NotificationPayload) error {
	if !s.IsConfigured() {
		return ErrVAPIDNotConfigured
	}

	subs, err := s.repo.GetSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		return ErrNoSubscriptions
	}

	return s.sendToSubscriptions(ctx, subs, payload)
}

// sendToSubscriptions sends a notification to multiple subscriptions
func (s *PushNotificationService) sendToSubscriptions(ctx context.Context, subs []model.PushSubscription, payload *NotificationPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		wpSub := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(payloadBytes, wpSub, &webpush.Options{
			Subscriber:      s.config.VAPIDSubject,
			VAPIDPublicKey:  s.config.VAPIDPublicKey,
			VAPIDPrivateKey: s.config.VAPIDPrivateKey,
			TTL:             86400, // 24 hours
		})

		if err != nil {
			// Log the error but continue with other subscriptions
			continue
		}
		defer resp.Body.Close()

		// If subscription is expired or invalid, remove it
		if resp.StatusCode == 404 || resp.StatusCode == 410 {
			_ = s.repo.DeleteSubscriptionByEndpoint(ctx, sub.Endpoint)
		}
	}

	return nil
}

// SendBillReminder sends a bill reminder notification
func (s *PushNotificationService) SendBillReminder(ctx context.Context, userID uuid.UUID, billName string, amount string, dueDate time.Time, recurringID uuid.UUID) error {
	// Check if we've already sent this notification
	refDate := dueDate
	hasRecent, err := s.repo.HasRecentNotification(ctx, userID, model.NotificationTypeBillReminder, &recurringID, &refDate)
	if err != nil {
		return err
	}
	if hasRecent {
		return nil // Already notified
	}

	daysUntil := int(time.Until(dueDate).Hours() / 24)
	var body string
	if daysUntil == 0 {
		body = "Due today"
	} else if daysUntil == 1 {
		body = "Due tomorrow"
	} else {
		body = "Due in " + string(rune(daysUntil+'0')) + " days"
	}

	payload := &NotificationPayload{
		Title: "Upcoming Bill: " + billName,
		Body:  amount + " - " + body,
		Icon:  "/icon-192.png",
		Badge: "/badge-72.png",
		Tag:   "bill-" + recurringID.String(),
		Data: map[string]interface{}{
			"type":        "bill_reminder",
			"recurringId": recurringID.String(),
			"url":         "/recurring",
		},
	}

	err = s.SendToUser(ctx, userID, payload)

	// Log the notification
	log := &model.NotificationLog{
		ID:               uuid.New(),
		UserID:           userID,
		NotificationType: model.NotificationTypeBillReminder,
		ReferenceID:      &recurringID,
		ReferenceDate:    &refDate,
		Title:            payload.Title,
		Body:             payload.Body,
		SentAt:           time.Now(),
		Success:          err == nil || errors.Is(err, ErrNoSubscriptions),
	}
	if err != nil && !errors.Is(err, ErrNoSubscriptions) {
		errMsg := err.Error()
		log.ErrorMessage = &errMsg
	}
	_ = s.repo.LogNotification(ctx, log)

	return err
}

// SendBudgetAlert sends a budget overspending alert
func (s *PushNotificationService) SendBudgetAlert(ctx context.Context, userID uuid.UUID, category string, percentage int, budgetID uuid.UUID) error {
	// Check if we've already sent this notification today
	today := time.Now().Truncate(24 * time.Hour)
	hasRecent, err := s.repo.HasRecentNotification(ctx, userID, model.NotificationTypeBudgetAlert, &budgetID, &today)
	if err != nil {
		return err
	}
	if hasRecent {
		return nil // Already notified today
	}

	var title, body string
	if percentage >= 100 {
		title = "Budget Exceeded: " + category
		body = "You've exceeded your budget for " + category
	} else {
		title = "Budget Alert: " + category
		body = "You've used " + string(rune(percentage/10+'0')) + string(rune(percentage%10+'0')) + "% of your " + category + " budget"
	}

	payload := &NotificationPayload{
		Title: title,
		Body:  body,
		Icon:  "/icon-192.png",
		Badge: "/badge-72.png",
		Tag:   "budget-" + budgetID.String(),
		Data: map[string]interface{}{
			"type":     "budget_alert",
			"budgetId": budgetID.String(),
			"url":      "/budgets",
		},
	}

	err = s.SendToUser(ctx, userID, payload)

	// Log the notification
	log := &model.NotificationLog{
		ID:               uuid.New(),
		UserID:           userID,
		NotificationType: model.NotificationTypeBudgetAlert,
		ReferenceID:      &budgetID,
		ReferenceDate:    &today,
		Title:            payload.Title,
		Body:             payload.Body,
		SentAt:           time.Now(),
		Success:          err == nil || errors.Is(err, ErrNoSubscriptions),
	}
	if err != nil && !errors.Is(err, ErrNoSubscriptions) {
		errMsg := err.Error()
		log.ErrorMessage = &errMsg
	}
	_ = s.repo.LogNotification(ctx, log)

	return err
}

// RegisterMobileToken registers an Expo push token for mobile devices
func (s *PushNotificationService) RegisterMobileToken(ctx context.Context, userID uuid.UUID, token, platform, deviceName string) (*model.MobilePushToken, error) {
	mobileToken := &model.MobilePushToken{
		ID:         uuid.New(),
		UserID:     userID,
		Token:      token,
		Platform:   platform,
		DeviceName: deviceName,
	}

	if err := s.repo.CreateMobileToken(ctx, mobileToken); err != nil {
		return nil, err
	}

	return mobileToken, nil
}

// UnregisterMobileToken removes an Expo push token
func (s *PushNotificationService) UnregisterMobileToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.repo.DeleteMobileToken(ctx, userID, token)
}

// SendGoalMilestone sends a savings goal milestone notification
func (s *PushNotificationService) SendGoalMilestone(ctx context.Context, userID uuid.UUID, goalName string, percentage int, goalID uuid.UUID) error {
	// Check if we've already sent this milestone notification
	today := time.Now().Truncate(24 * time.Hour)
	hasRecent, err := s.repo.HasRecentNotification(ctx, userID, model.NotificationTypeGoalMilestone, &goalID, &today)
	if err != nil {
		return err
	}
	if hasRecent {
		return nil // Already notified
	}

	var title, body string
	if percentage >= 100 {
		title = "Goal Achieved!"
		body = "Congratulations! You've reached your goal: " + goalName
	} else {
		title = "Milestone Reached: " + goalName
		body = "You're " + string(rune(percentage/10+'0')) + string(rune(percentage%10+'0')) + "% of the way to your goal!"
	}

	payload := &NotificationPayload{
		Title: title,
		Body:  body,
		Icon:  "/icon-192.png",
		Badge: "/badge-72.png",
		Tag:   "goal-" + goalID.String(),
		Data: map[string]interface{}{
			"type":   "goal_milestone",
			"goalId": goalID.String(),
			"url":    "/savings",
		},
	}

	err = s.SendToUser(ctx, userID, payload)

	// Log the notification
	log := &model.NotificationLog{
		ID:               uuid.New(),
		UserID:           userID,
		NotificationType: model.NotificationTypeGoalMilestone,
		ReferenceID:      &goalID,
		ReferenceDate:    &today,
		Title:            payload.Title,
		Body:             payload.Body,
		SentAt:           time.Now(),
		Success:          err == nil || errors.Is(err, ErrNoSubscriptions),
	}
	if err != nil && !errors.Is(err, ErrNoSubscriptions) {
		errMsg := err.Error()
		log.ErrorMessage = &errMsg
	}
	_ = s.repo.LogNotification(ctx, log)

	return err
}
