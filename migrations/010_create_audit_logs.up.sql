-- Create audit_logs table for comprehensive audit logging
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    description TEXT,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    changes JSONB,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    trace_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX idx_audit_logs_timestamp ON audit_logs (timestamp DESC);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_logs_event_type ON audit_logs (event_type);
CREATE INDEX idx_audit_logs_severity ON audit_logs (severity);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_ip_address ON audit_logs (ip_address) WHERE ip_address IS NOT NULL;
CREATE INDEX idx_audit_logs_request_id ON audit_logs (request_id) WHERE request_id IS NOT NULL;
CREATE INDEX idx_audit_logs_session_id ON audit_logs (session_id) WHERE session_id IS NOT NULL;

-- Create composite indexes for common queries
CREATE INDEX idx_audit_logs_user_timestamp ON audit_logs (user_id, timestamp DESC) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_entity_timestamp ON audit_logs (entity_type, entity_id, timestamp DESC);
CREATE INDEX idx_audit_logs_event_severity ON audit_logs (event_type, severity);

-- Create GIN index for metadata searches
CREATE INDEX idx_audit_logs_metadata ON audit_logs USING GIN (metadata) WHERE metadata IS NOT NULL;

-- Add table comment
COMMENT ON TABLE audit_logs IS 'Comprehensive audit logging for all system events and user actions';
COMMENT ON COLUMN audit_logs.event_type IS 'Type of event (e.g., user.created, user.logged_in, etc.)';
COMMENT ON COLUMN audit_logs.severity IS 'Severity level: info, warning, error, critical';
COMMENT ON COLUMN audit_logs.user_id IS 'ID of the user who is the subject of the event';
COMMENT ON COLUMN audit_logs.actor_id IS 'ID of the user who performed the action (may be different from user_id)';
COMMENT ON COLUMN audit_logs.entity_type IS 'Type of entity affected (user, role, subscription, etc.)';
COMMENT ON COLUMN audit_logs.entity_id IS 'ID of the specific entity affected';
COMMENT ON COLUMN audit_logs.action IS 'Specific action performed (create, update, delete, etc.)';
COMMENT ON COLUMN audit_logs.metadata IS 'Additional structured data related to the event';
COMMENT ON COLUMN audit_logs.changes IS 'Before/after values for update operations';
COMMENT ON COLUMN audit_logs.request_id IS 'Request ID for tracing related events';
COMMENT ON COLUMN audit_logs.session_id IS 'Session ID if the event occurred within a user session';
COMMENT ON COLUMN audit_logs.trace_id IS 'Distributed tracing ID for cross-service events';