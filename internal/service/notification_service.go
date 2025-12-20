// Package service provides business logic for the application.
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
)

// NotificationService handles rate change notifications
type NotificationService struct {
	subRepo     repository.RateSubscriptionRepository
	rateRepo    repository.InterestRateRepository
	userRepo    repository.UserRepository
	emailSender EmailSender
}

// EmailSender defines the interface for sending emails
type EmailSender interface {
	Send(to, subject, body string) error
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	subRepo repository.RateSubscriptionRepository,
	rateRepo repository.InterestRateRepository,
	userRepo repository.UserRepository,
	emailSender EmailSender,
) *NotificationService {
	return &NotificationService{
		subRepo:     subRepo,
		rateRepo:    rateRepo,
		userRepo:    userRepo,
		emailSender: emailSender,
	}
}

// Subscribe creates a new rate subscription for a user
func (s *NotificationService) Subscribe(ctx context.Context, userID uuid.UUID, input *model.RateSubscription) (*model.RateSubscription, error) {
	input.UserID = userID

	// Get current rate to store as baseline
	rates, err := s.rateRepo.List(ctx, input.ProductType, &input.TermMonths, input.BankCode)
	if err != nil {
		return nil, fmt.Errorf("get current rate: %w", err)
	}

	if len(rates) > 0 {
		input.LastRate = rates[0].Rate
	}

	if err := s.subRepo.Create(ctx, input); err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	return input, nil
}

// Unsubscribe removes a rate subscription
func (s *NotificationService) Unsubscribe(ctx context.Context, userID uuid.UUID, subscriptionID int64) error {
	return s.subRepo.Delete(ctx, userID, subscriptionID)
}

// GetUserSubscriptions returns all subscriptions for a user
func (s *NotificationService) GetUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]model.RateSubscription, error) {
	return s.subRepo.ListByUser(ctx, userID)
}

// CheckAndNotify checks for rate changes and sends notifications
// This should be called periodically (e.g., daily via cron)
func (s *NotificationService) CheckAndNotify(ctx context.Context) (int, error) {
	// Get all active subscriptions
	subscriptions, err := s.subRepo.ListAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("list subscriptions: %w", err)
	}

	notificationsSent := 0

	for _, sub := range subscriptions {
		// Get current rate
		rates, err := s.rateRepo.List(ctx, sub.ProductType, &sub.TermMonths, sub.BankCode)
		if err != nil || len(rates) == 0 {
			continue
		}

		currentRate := rates[0].Rate
		if sub.LastRate.IsZero() {
			// No previous rate recorded, just update
			if err := s.subRepo.UpdateLastRate(ctx, sub.ID, currentRate); err != nil {
				log.Printf("Error updating last rate for subscription %d: %v", sub.ID, err)
			}
			continue
		}

		// Check if rate changed
		change := currentRate.Sub(sub.LastRate)
		if change.IsZero() {
			continue
		}

		// Calculate percent change
		changePercent := change.Div(sub.LastRate).Mul(decimal.NewFromInt(100))

		// Check if threshold exceeded (if set)
		if !sub.ThresholdPercent.IsZero() && changePercent.Abs().LessThan(sub.ThresholdPercent) {
			continue
		}

		// Determine if we should notify based on direction
		isIncrease := change.IsPositive()
		if isIncrease && !sub.NotifyOnIncrease {
			continue
		}
		if !isIncrease && !sub.NotifyOnDecrease {
			continue
		}

		// Get bank name
		bankName := sub.BankCode
		for _, bank := range model.VietnameseBanks {
			if bank.Code == sub.BankCode {
				bankName = bank.Name
				break
			}
		}

		// Send notification
		alert := &model.RateChangeAlert{
			Subscription: sub,
			BankName:     bankName,
			OldRate:      sub.LastRate,
			NewRate:      currentRate,
			Change:       change,
			ChangeType:   "increase",
		}
		if !isIncrease {
			alert.ChangeType = "decrease"
		}

		if err := s.sendNotification(ctx, alert); err != nil {
			log.Printf("Error sending notification for subscription %d: %v", sub.ID, err)
			continue
		}

		// Update subscription with new rate
		now := time.Now()
		sub.LastRate = currentRate
		sub.LastNotifiedAt = &now
		if err := s.subRepo.UpdateLastRate(ctx, sub.ID, currentRate); err != nil {
			log.Printf("Error updating subscription %d: %v", sub.ID, err)
		}

		notificationsSent++
	}

	return notificationsSent, nil
}

func (s *NotificationService) sendNotification(ctx context.Context, alert *model.RateChangeAlert) error {
	// Get user email
	user, err := s.userRepo.GetByID(ctx, alert.Subscription.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if alert.Subscription.EmailEnabled && s.emailSender != nil {
		subject := fmt.Sprintf("📈 Lãi suất %s đã thay đổi", alert.BankName)
		if alert.ChangeType == "decrease" {
			subject = fmt.Sprintf("📉 Lãi suất %s đã giảm", alert.BankName)
		}

		body := fmt.Sprintf(`
Xin chào %s,

Lãi suất %s %s kỳ hạn %d tháng đã %s:

• Lãi suất cũ: %s%%
• Lãi suất mới: %s%%
• Thay đổi: %s%%

Truy cập WealthPath để xem chi tiết và so sánh lãi suất các ngân hàng khác.

---
WealthPath - Quản lý tài chính cá nhân
		`,
			user.Name,
			alert.BankName,
			alert.Subscription.ProductType,
			alert.Subscription.TermMonths,
			alert.ChangeType,
			alert.OldRate.StringFixed(2),
			alert.NewRate.StringFixed(2),
			alert.Change.StringFixed(2),
		)

		if err := s.emailSender.Send(user.Email, subject, body); err != nil {
			log.Printf("Failed to send email to %s: %v", user.Email, err)
			// Don't return error, continue with other notifications
		}
	}

	// Log the notification
	notification := &model.RateNotification{
		SubscriptionID:   alert.Subscription.ID,
		OldRate:          alert.OldRate,
		NewRate:          alert.NewRate,
		ChangePercent:    alert.Change.Div(alert.OldRate).Mul(decimal.NewFromInt(100)),
		NotificationType: "email",
		Status:           "sent",
	}

	if err := s.subRepo.LogNotification(ctx, notification); err != nil {
		log.Printf("Failed to log notification: %v", err)
	}

	return nil
}
