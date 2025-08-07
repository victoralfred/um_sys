package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeUserCreated         EventType = "user.created"
	EventTypeUserUpdated         EventType = "user.updated"
	EventTypeUserDeleted         EventType = "user.deleted"
	EventTypeUserLoggedIn        EventType = "user.logged_in"
	EventTypeUserLoggedOut       EventType = "user.logged_out"
	EventTypeUserPasswordChanged EventType = "user.password_changed"
	EventTypeUserMFAEnabled      EventType = "user.mfa_enabled"
	EventTypeUserMFADisabled     EventType = "user.mfa_disabled"

	EventTypeRoleCreated  EventType = "role.created"
	EventTypeRoleUpdated  EventType = "role.updated"
	EventTypeRoleDeleted  EventType = "role.deleted"
	EventTypeRoleAssigned EventType = "role.assigned"
	EventTypeRoleRevoked  EventType = "role.revoked"

	EventTypePermissionGranted EventType = "permission.granted"
	EventTypePermissionRevoked EventType = "permission.revoked"

	EventTypeSubscriptionCreated  EventType = "subscription.created"
	EventTypeSubscriptionUpdated  EventType = "subscription.updated"
	EventTypeSubscriptionCanceled EventType = "subscription.canceled"
	EventTypeSubscriptionRenewed  EventType = "subscription.renewed"

	EventTypePaymentProcessed EventType = "payment.processed"
	EventTypePaymentFailed    EventType = "payment.failed"
	EventTypePaymentRefunded  EventType = "payment.refunded"

	EventTypeSecurityAlert EventType = "security.alert"
	EventTypeAccessDenied  EventType = "access.denied"
	EventTypeRateLimited   EventType = "rate.limited"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

type LogEntry struct {
	ID          uuid.UUID              `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   EventType              `json:"event_type"`
	Severity    Severity               `json:"severity"`
	UserID      *uuid.UUID             `json:"user_id,omitempty"`
	ActorID     *uuid.UUID             `json:"actor_id,omitempty"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Changes     *Changes               `json:"changes,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type Changes struct {
	Before json.RawMessage `json:"before,omitempty"`
	After  json.RawMessage `json:"after,omitempty"`
	Fields []string        `json:"fields,omitempty"`
}

type LogFilter struct {
	UserID     *uuid.UUID
	ActorID    *uuid.UUID
	EventTypes []EventType
	Severities []Severity
	EntityType string
	EntityID   string
	IPAddress  string
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
}

type LogSummary struct {
	TotalEvents      int64               `json:"total_events"`
	EventsByType     map[EventType]int64 `json:"events_by_type"`
	EventsBySeverity map[Severity]int64  `json:"events_by_severity"`
	UniqueUsers      int64               `json:"unique_users"`
	UniqueIPs        int64               `json:"unique_ips"`
	TimeRange        TimeRange           `json:"time_range"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type AuditConfig struct {
	Enabled           bool        `json:"enabled"`
	RetentionDays     int         `json:"retention_days"`
	MaxEntriesPerUser int         `json:"max_entries_per_user"`
	SensitiveFields   []string    `json:"sensitive_fields"`
	ExcludedEvents    []EventType `json:"excluded_events"`
}

type CreateLogRequest struct {
	EventType   EventType              `json:"event_type"`
	Severity    Severity               `json:"severity"`
	UserID      *uuid.UUID             `json:"user_id,omitempty"`
	ActorID     *uuid.UUID             `json:"actor_id,omitempty"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Changes     *Changes               `json:"changes,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
}

type ExportRequest struct {
	Filter   LogFilter `json:"filter"`
	Format   string    `json:"format"`
	Fields   []string  `json:"fields,omitempty"`
	Compress bool      `json:"compress"`
}

type ExportResponse struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	URL       string    `json:"url,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type AlertRule struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	EventTypes  []EventType            `json:"event_types"`
	Severities  []Severity             `json:"severities"`
	Threshold   int                    `json:"threshold"`
	TimeWindow  time.Duration          `json:"time_window"`
	Actions     []AlertAction          `json:"actions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type AlertAction struct {
	Type     string                 `json:"type"`
	Target   string                 `json:"target"`
	Template string                 `json:"template,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

type ComplianceReport struct {
	ID          uuid.UUID              `json:"id"`
	ReportType  string                 `json:"report_type"`
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	Status      string                 `json:"status"`
	Summary     map[string]interface{} `json:"summary"`
	Details     json.RawMessage        `json:"details"`
	GeneratedAt time.Time              `json:"generated_at"`
	GeneratedBy uuid.UUID              `json:"generated_by"`
}
