package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RateSubscription represents a user's subscription to rate change notifications
type RateSubscription struct {
	ID               int64           `db:"id" json:"id"`
	UserID           uuid.UUID       `db:"user_id" json:"userId"`
	BankCode         string          `db:"bank_code" json:"bankCode"`
	ProductType      string          `db:"product_type" json:"productType"`
	TermMonths       int             `db:"term_months" json:"termMonths"`
	ThresholdPercent decimal.Decimal `db:"threshold_percent" json:"thresholdPercent,omitempty"`
	NotifyOnIncrease bool            `db:"notify_on_increase" json:"notifyOnIncrease"`
	NotifyOnDecrease bool            `db:"notify_on_decrease" json:"notifyOnDecrease"`
	EmailEnabled     bool            `db:"email_enabled" json:"emailEnabled"`
	PushEnabled      bool            `db:"push_enabled" json:"pushEnabled"`
	LastNotifiedAt   *time.Time      `db:"last_notified_at" json:"lastNotifiedAt,omitempty"`
	LastRate         decimal.Decimal `db:"last_rate" json:"lastRate,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updatedAt"`
}

// RateNotification represents a notification sent for a rate change
type RateNotification struct {
	ID               int64           `db:"id" json:"id"`
	SubscriptionID   int64           `db:"subscription_id" json:"subscriptionId"`
	OldRate          decimal.Decimal `db:"old_rate" json:"oldRate"`
	NewRate          decimal.Decimal `db:"new_rate" json:"newRate"`
	ChangePercent    decimal.Decimal `db:"change_percent" json:"changePercent"`
	NotificationType string          `db:"notification_type" json:"notificationType"` // email, push
	SentAt           time.Time       `db:"sent_at" json:"sentAt"`
	Status           string          `db:"status" json:"status"` // pending, sent, failed
}

// RateChangeAlert represents an alert to be sent to a user
type RateChangeAlert struct {
	Subscription RateSubscription
	BankName     string
	OldRate      decimal.Decimal
	NewRate      decimal.Decimal
	Change       decimal.Decimal // Positive = increase, negative = decrease
	ChangeType   string          // "increase" or "decrease"
}
