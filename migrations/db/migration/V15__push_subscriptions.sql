-- Push subscriptions table for web push notifications
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,  -- Browser public key
    auth TEXT NOT NULL,    -- Auth secret
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, endpoint)
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);
CREATE INDEX idx_push_subscriptions_endpoint ON push_subscriptions(endpoint);

-- Notification preferences table
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    bill_reminders_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    bill_reminder_days_before INTEGER NOT NULL DEFAULT 3,
    budget_alerts_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    budget_alert_threshold INTEGER NOT NULL DEFAULT 90,  -- Percentage
    goal_milestones_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    weekly_summary_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notification_preferences_user_id ON notification_preferences(user_id);

-- Notification log to track sent notifications and avoid duplicates
CREATE TABLE IF NOT EXISTS notification_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type TEXT NOT NULL,  -- 'bill_reminder', 'budget_alert', 'goal_milestone', 'weekly_summary'
    reference_id UUID,                -- Optional: ID of the related entity (budget_id, goal_id, etc.)
    reference_date DATE,              -- For time-based deduplication (e.g., bill date)
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT
);

CREATE INDEX idx_notification_log_user_id ON notification_log(user_id);
CREATE INDEX idx_notification_log_type_ref ON notification_log(notification_type, reference_id, reference_date);
CREATE INDEX idx_notification_log_sent_at ON notification_log(sent_at);
