-- Create analytics_events table
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    properties JSONB,
    context JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create analytics_metrics table
CREATE TABLE IF NOT EXISTS analytics_metrics (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('counter', 'gauge', 'histogram', 'summary')),
    value DOUBLE PRECISION NOT NULL,
    labels JSONB,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    ttl BIGINT, -- TTL in seconds
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for analytics_events
CREATE INDEX IF NOT EXISTS idx_analytics_events_type ON analytics_events(type);
CREATE INDEX IF NOT EXISTS idx_analytics_events_user_id ON analytics_events(user_id);
CREATE INDEX IF NOT EXISTS idx_analytics_events_session_id ON analytics_events(session_id);
CREATE INDEX IF NOT EXISTS idx_analytics_events_timestamp ON analytics_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_user_timestamp ON analytics_events(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_type_timestamp ON analytics_events(type, timestamp DESC);

-- Create GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_analytics_events_properties ON analytics_events USING GIN(properties);
CREATE INDEX IF NOT EXISTS idx_analytics_events_context ON analytics_events USING GIN(context);

-- Create composite index for common queries
CREATE INDEX IF NOT EXISTS idx_analytics_events_composite ON analytics_events(type, user_id, timestamp DESC);

-- Create indexes for analytics_metrics
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_name ON analytics_metrics(name);
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_type ON analytics_metrics(type);
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_timestamp ON analytics_metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_name_timestamp ON analytics_metrics(name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_type_timestamp ON analytics_metrics(type, timestamp DESC);

-- Create GIN index for labels JSONB column
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_labels ON analytics_metrics USING GIN(labels);

-- Create composite index for metrics queries
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_composite ON analytics_metrics(name, type, timestamp DESC);

-- Create index for TTL-based cleanup
CREATE INDEX IF NOT EXISTS idx_analytics_metrics_ttl_cleanup ON analytics_metrics(created_at, ttl) WHERE ttl IS NOT NULL;