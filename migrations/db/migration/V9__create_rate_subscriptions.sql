-- User subscriptions for interest rate notifications
CREATE TABLE IF NOT EXISTS rate_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bank_code VARCHAR(20) NOT NULL,
    product_type VARCHAR(50) NOT NULL DEFAULT 'deposit',
    term_months INT NOT NULL,
    threshold_percent DECIMAL(5,2), -- Notify if rate changes by this percentage
    notify_on_increase BOOLEAN DEFAULT true,
    notify_on_decrease BOOLEAN DEFAULT true,
    email_enabled BOOLEAN DEFAULT true,
    push_enabled BOOLEAN DEFAULT false,
    last_notified_at TIMESTAMP,
    last_rate DECIMAL(5,2),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_rate_subs_user_id ON rate_subscriptions(user_id);
CREATE INDEX idx_rate_subs_bank_product ON rate_subscriptions(bank_code, product_type, term_months);

-- Unique constraint - one subscription per user/bank/product/term
CREATE UNIQUE INDEX idx_rate_subs_unique ON rate_subscriptions(user_id, bank_code, product_type, term_months);

-- Rate change notifications log
CREATE TABLE IF NOT EXISTS rate_notifications (
    id SERIAL PRIMARY KEY,
    subscription_id INT NOT NULL REFERENCES rate_subscriptions(id) ON DELETE CASCADE,
    old_rate DECIMAL(5,2) NOT NULL,
    new_rate DECIMAL(5,2) NOT NULL,
    change_percent DECIMAL(5,2) NOT NULL,
    notification_type VARCHAR(20) NOT NULL, -- email, push
    sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' -- pending, sent, failed
);

CREATE INDEX idx_rate_notif_sub_id ON rate_notifications(subscription_id);
CREATE INDEX idx_rate_notif_status ON rate_notifications(status);

-- Comments
COMMENT ON TABLE rate_subscriptions IS 'User subscriptions for interest rate change notifications';
COMMENT ON TABLE rate_notifications IS 'Log of sent rate change notifications';

