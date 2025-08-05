-- Create feature_flags table
CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    rollout_percentage INT CHECK (rollout_percentage >= 0 AND rollout_percentage <= 100) DEFAULT 0,
    targeting_rules JSONB,
    variants JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create user_feature_flags table for user-specific overrides
CREATE TABLE IF NOT EXISTS user_feature_flags (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature_flag_id UUID NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    is_enabled BOOLEAN NOT NULL,
    variant VARCHAR(100),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, feature_flag_id)
);

-- Create indexes
CREATE INDEX idx_feature_flags_key ON feature_flags(key);
CREATE INDEX idx_feature_flags_is_enabled ON feature_flags(is_enabled);
CREATE INDEX idx_user_feature_flags_user_id ON user_feature_flags(user_id);
CREATE INDEX idx_user_feature_flags_feature_flag_id ON user_feature_flags(feature_flag_id);