-- Create user_sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    refresh_token_hash VARCHAR(255) UNIQUE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    device_info JSONB,
    expires_at TIMESTAMP NOT NULL,
    refresh_expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    revoked_reason VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_user_sessions_token_hash ON user_sessions(token_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_user_sessions_refresh_token_hash ON user_sessions(refresh_token_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_user_sessions_created_at ON user_sessions(created_at);