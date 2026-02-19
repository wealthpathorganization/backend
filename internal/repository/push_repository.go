package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/wealthpath/backend/internal/model"
)

type PushRepository struct {
	db *sqlx.DB
}

func NewPushRepository(db *sqlx.DB) *PushRepository {
	return &PushRepository{db: db}
}

// Subscriptions

func (r *PushRepository) CreateSubscription(ctx context.Context, sub *model.PushSubscription) error {
	query := `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, user_agent, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (user_id, endpoint) DO UPDATE SET
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth,
			user_agent = EXCLUDED.user_agent,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		sub.ID, sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, sub.UserAgent,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

func (r *PushRepository) GetSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]model.PushSubscription, error) {
	var subs []model.PushSubscription
	query := `SELECT * FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &subs, query, userID)
	return subs, err
}

func (r *PushRepository) GetAllActiveSubscriptions(ctx context.Context) ([]model.PushSubscription, error) {
	var subs []model.PushSubscription
	query := `SELECT * FROM push_subscriptions ORDER BY user_id, created_at DESC`
	err := r.db.SelectContext(ctx, &subs, query)
	return subs, err
}

func (r *PushRepository) DeleteSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	query := `DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`
	_, err := r.db.ExecContext(ctx, query, userID, endpoint)
	return err
}

func (r *PushRepository) DeleteSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	query := `DELETE FROM push_subscriptions WHERE endpoint = $1`
	_, err := r.db.ExecContext(ctx, query, endpoint)
	return err
}

// Notification Preferences

func (r *PushRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error) {
	var prefs model.NotificationPreferences
	query := `SELECT * FROM notification_preferences WHERE user_id = $1`
	err := r.db.GetContext(ctx, &prefs, query, userID)
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

func (r *PushRepository) UpsertPreferences(ctx context.Context, prefs *model.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			id, user_id, bill_reminders_enabled, bill_reminder_days_before,
			budget_alerts_enabled, budget_alert_threshold,
			goal_milestones_enabled, weekly_summary_enabled,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			bill_reminders_enabled = EXCLUDED.bill_reminders_enabled,
			bill_reminder_days_before = EXCLUDED.bill_reminder_days_before,
			budget_alerts_enabled = EXCLUDED.budget_alerts_enabled,
			budget_alert_threshold = EXCLUDED.budget_alert_threshold,
			goal_milestones_enabled = EXCLUDED.goal_milestones_enabled,
			weekly_summary_enabled = EXCLUDED.weekly_summary_enabled,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		prefs.ID, prefs.UserID,
		prefs.BillRemindersEnabled, prefs.BillReminderDaysBefore,
		prefs.BudgetAlertsEnabled, prefs.BudgetAlertThreshold,
		prefs.GoalMilestonesEnabled, prefs.WeeklySummaryEnabled,
	).Scan(&prefs.ID, &prefs.CreatedAt, &prefs.UpdatedAt)
}

// Notification Log

func (r *PushRepository) LogNotification(ctx context.Context, log *model.NotificationLog) error {
	query := `
		INSERT INTO notification_log (id, user_id, notification_type, reference_id, reference_date, title, body, sent_at, success, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING sent_at`

	return r.db.QueryRowxContext(ctx, query,
		log.ID, log.UserID, log.NotificationType, log.ReferenceID, log.ReferenceDate,
		log.Title, log.Body, log.SentAt, log.Success, log.ErrorMessage,
	).Scan(&log.SentAt)
}

func (r *PushRepository) HasRecentNotification(ctx context.Context, userID uuid.UUID, notifType model.NotificationType, refID *uuid.UUID, refDate *time.Time) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM notification_log
		WHERE user_id = $1
		AND notification_type = $2
		AND ($3::uuid IS NULL OR reference_id = $3)
		AND ($4::date IS NULL OR reference_date = $4)
		AND success = TRUE
		AND sent_at > NOW() - INTERVAL '24 hours'`

	err := r.db.GetContext(ctx, &count, query, userID, notifType, refID, refDate)
	return count > 0, err
}

// GetUsersWithBillsDue returns users who have bills due within the specified days
func (r *PushRepository) GetUsersWithBillsDue(ctx context.Context, daysAhead int) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID
	query := `
		SELECT DISTINCT rt.user_id
		FROM recurring_transactions rt
		JOIN notification_preferences np ON rt.user_id = np.user_id
		WHERE rt.is_active = TRUE
		AND rt.type = 'expense'
		AND np.bill_reminders_enabled = TRUE
		AND rt.next_occurrence BETWEEN CURRENT_DATE AND CURRENT_DATE + $1 * INTERVAL '1 day'`

	err := r.db.SelectContext(ctx, &userIDs, query, daysAhead)
	return userIDs, err
}

// GetUsersNearBudgetLimit returns users whose budget usage exceeds the threshold
func (r *PushRepository) GetUsersNearBudgetLimit(ctx context.Context) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID
	query := `
		SELECT DISTINCT b.user_id
		FROM budgets b
		JOIN notification_preferences np ON b.user_id = np.user_id
		WHERE np.budget_alerts_enabled = TRUE
		AND b.start_date <= CURRENT_DATE
		AND (b.end_date IS NULL OR b.end_date >= CURRENT_DATE)`

	err := r.db.SelectContext(ctx, &userIDs, query)
	return userIDs, err
}

// Mobile Push Tokens

// CreateMobileToken creates or updates a mobile push token
func (r *PushRepository) CreateMobileToken(ctx context.Context, token *model.MobilePushToken) error {
	query := `
		INSERT INTO mobile_push_tokens (id, user_id, token, platform, device_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (user_id, token) DO UPDATE SET
			platform = EXCLUDED.platform,
			device_name = EXCLUDED.device_name,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		token.ID, token.UserID, token.Token, token.Platform, token.DeviceName,
	).Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt)
}

// DeleteMobileToken removes a mobile push token
func (r *PushRepository) DeleteMobileToken(ctx context.Context, userID uuid.UUID, token string) error {
	query := `DELETE FROM mobile_push_tokens WHERE user_id = $1 AND token = $2`
	_, err := r.db.ExecContext(ctx, query, userID, token)
	return err
}
