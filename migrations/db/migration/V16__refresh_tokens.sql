-- Create refresh_tokens table for "Remember Me" feature
-- This table stores refresh tokens that allow users to maintain long-lived sessions

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    device_info JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(100),

    CONSTRAINT uq_refresh_tokens_token_hash UNIQUE (token_hash)
);

-- Index for efficient user lookups (list sessions, revoke all)
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Index for cleanup job to find expired tokens
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Index for finding active (non-revoked) tokens
CREATE INDEX idx_refresh_tokens_user_active ON refresh_tokens(user_id) WHERE revoked_at IS NULL;

COMMENT ON TABLE refresh_tokens IS 'Stores refresh tokens for remember me functionality and session management';
COMMENT ON COLUMN refresh_tokens.token_hash IS 'SHA-256 hash of the refresh token (never store raw token)';
COMMENT ON COLUMN refresh_tokens.device_info IS 'JSON containing browser, OS, device type, and IP for session display';
COMMENT ON COLUMN refresh_tokens.revoked_reason IS 'Why token was revoked: logout, password_change, admin, security, etc.';
