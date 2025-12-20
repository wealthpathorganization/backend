package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
)

// RateSubscriptionRepository defines the interface for rate subscription data access
type RateSubscriptionRepository interface {
	Create(ctx context.Context, sub *model.RateSubscription) error
	GetByID(ctx context.Context, id int64) (*model.RateSubscription, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.RateSubscription, error)
	ListAll(ctx context.Context) ([]model.RateSubscription, error)
	Update(ctx context.Context, sub *model.RateSubscription) error
	UpdateLastRate(ctx context.Context, id int64, rate decimal.Decimal) error
	Delete(ctx context.Context, userID uuid.UUID, id int64) error
	LogNotification(ctx context.Context, notification *model.RateNotification) error
}

type rateSubscriptionRepository struct {
	db *sqlx.DB
}

// NewRateSubscriptionRepository creates a new rate subscription repository
func NewRateSubscriptionRepository(db *sqlx.DB) RateSubscriptionRepository {
	return &rateSubscriptionRepository{db: db}
}

// Create creates a new rate subscription
func (r *rateSubscriptionRepository) Create(ctx context.Context, sub *model.RateSubscription) error {
	query := `
		INSERT INTO rate_subscriptions (
			user_id, bank_code, product_type, term_months, threshold_percent,
			notify_on_increase, notify_on_decrease, email_enabled, push_enabled, last_rate
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, bank_code, product_type, term_months)
		DO UPDATE SET
			threshold_percent = EXCLUDED.threshold_percent,
			notify_on_increase = EXCLUDED.notify_on_increase,
			notify_on_decrease = EXCLUDED.notify_on_decrease,
			email_enabled = EXCLUDED.email_enabled,
			push_enabled = EXCLUDED.push_enabled,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		sub.UserID, sub.BankCode, sub.ProductType, sub.TermMonths, sub.ThresholdPercent,
		sub.NotifyOnIncrease, sub.NotifyOnDecrease, sub.EmailEnabled, sub.PushEnabled, sub.LastRate,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

// GetByID returns a subscription by ID
func (r *rateSubscriptionRepository) GetByID(ctx context.Context, id int64) (*model.RateSubscription, error) {
	var sub model.RateSubscription
	err := r.db.GetContext(ctx, &sub, `
		SELECT * FROM rate_subscriptions WHERE id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return &sub, nil
}

// ListByUser returns all subscriptions for a user
func (r *rateSubscriptionRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.RateSubscription, error) {
	var subs []model.RateSubscription
	err := r.db.SelectContext(ctx, &subs, `
		SELECT * FROM rate_subscriptions WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	return subs, nil
}

// ListAll returns all active subscriptions
func (r *rateSubscriptionRepository) ListAll(ctx context.Context) ([]model.RateSubscription, error) {
	var subs []model.RateSubscription
	err := r.db.SelectContext(ctx, &subs, `
		SELECT * FROM rate_subscriptions 
		WHERE email_enabled = true OR push_enabled = true
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("list all subscriptions: %w", err)
	}
	return subs, nil
}

// Update updates a subscription
func (r *rateSubscriptionRepository) Update(ctx context.Context, sub *model.RateSubscription) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE rate_subscriptions SET
			threshold_percent = $1,
			notify_on_increase = $2,
			notify_on_decrease = $3,
			email_enabled = $4,
			push_enabled = $5,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND user_id = $7
	`, sub.ThresholdPercent, sub.NotifyOnIncrease, sub.NotifyOnDecrease,
		sub.EmailEnabled, sub.PushEnabled, sub.ID, sub.UserID)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

// UpdateLastRate updates the last known rate and notification time
func (r *rateSubscriptionRepository) UpdateLastRate(ctx context.Context, id int64, rate decimal.Decimal) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE rate_subscriptions SET
			last_rate = $1,
			last_notified_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, rate, id)
	if err != nil {
		return fmt.Errorf("update last rate: %w", err)
	}
	return nil
}

// Delete removes a subscription
func (r *rateSubscriptionRepository) Delete(ctx context.Context, userID uuid.UUID, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM rate_subscriptions WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("subscription not found")
	}

	return nil
}

// LogNotification logs a sent notification
func (r *rateSubscriptionRepository) LogNotification(ctx context.Context, notification *model.RateNotification) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO rate_notifications (
			subscription_id, old_rate, new_rate, change_percent, notification_type, status
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, notification.SubscriptionID, notification.OldRate, notification.NewRate,
		notification.ChangePercent, notification.NotificationType, notification.Status)
	if err != nil {
		return fmt.Errorf("log notification: %w", err)
	}
	return nil
}
