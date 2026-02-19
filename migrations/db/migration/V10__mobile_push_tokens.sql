-- V10__mobile_push_tokens.sql
-- Add mobile push token table for Expo notifications

CREATE TABLE IF NOT EXISTS mobile_push_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    platform VARCHAR(20) NOT NULL, -- 'ios' or 'android'
    device_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, token)
);

-- Index for fast lookup by user_id
CREATE INDEX IF NOT EXISTS idx_mobile_push_tokens_user_id ON mobile_push_tokens(user_id);

-- Index for fast lookup by token (for unregistration)
CREATE INDEX IF NOT EXISTS idx_mobile_push_tokens_token ON mobile_push_tokens(token);
