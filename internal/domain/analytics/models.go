package analytics

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents different types of analytics events
type EventType string

const (
	EventTypeUserLogin        EventType = "user_login"
	EventTypeUserLogout       EventType = "user_logout"
	EventTypeUserRegistration EventType = "user_registration"
	EventTypeAPICall          EventType = "api_call"
	EventTypeFeatureUsage     EventType = "feature_usage"
	EventTypePageView         EventType = "page_view"
	EventTypeButtonClick      EventType = "button_click"
	EventTypeFormSubmit       EventType = "form_submit"
	EventTypeError            EventType = "error"
)

// Event represents an analytics event
type Event struct {
	ID         uuid.UUID              `json:"id"`
	Type       EventType              `json:"type"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	SessionID  *string                `json:"session_id,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Context    *EventContext          `json:"context,omitempty"`
	Version    int                    `json:"version,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// EventContext provides additional context for events
type EventContext struct {
	IPAddress    string            `json:"ip_address,omitempty"`
	UserAgent    string            `json:"user_agent,omitempty"`
	Referrer     string            `json:"referrer,omitempty"`
	Path         string            `json:"path,omitempty"`
	Method       string            `json:"method,omitempty"`
	StatusCode   *int              `json:"status_code,omitempty"`
	ResponseTime *int64            `json:"response_time_ms,omitempty"`
	CustomData   map[string]string `json:"custom_data,omitempty"`
}

// MetricType represents different types of metrics
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a metric data point
type Metric struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	TTL       *time.Duration         `json:"ttl,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// UsageStats represents usage statistics
type UsageStats struct {
	Period         string                 `json:"period"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	TotalEvents    int64                  `json:"total_events"`
	UniqueUsers    int64                  `json:"unique_users"`
	TotalSessions  int64                  `json:"total_sessions"`
	AvgSessionTime time.Duration          `json:"avg_session_time"`
	EventsByType   map[EventType]int64    `json:"events_by_type"`
	TopPages       []PageStats            `json:"top_pages"`
	TopFeatures    []FeatureStats         `json:"top_features"`
	UserGrowth     []UserGrowthStats      `json:"user_growth"`
	CustomMetrics  map[string]interface{} `json:"custom_metrics,omitempty"`
}

// PageStats represents page visit statistics
type PageStats struct {
	Path        string        `json:"path"`
	Views       int64         `json:"views"`
	UniqueViews int64         `json:"unique_views"`
	AvgTime     time.Duration `json:"avg_time"`
}

// FeatureStats represents feature usage statistics
type FeatureStats struct {
	Feature string `json:"feature"`
	Usage   int64  `json:"usage"`
	Users   int64  `json:"users"`
}

// UserGrowthStats represents user growth over time
type UserGrowthStats struct {
	Date        time.Time `json:"date"`
	NewUsers    int64     `json:"new_users"`
	ActiveUsers int64     `json:"active_users"`
	ChurnRate   float64   `json:"churn_rate"`
}

// EventFilter represents filters for querying events
type EventFilter struct {
	Types     []EventType `json:"types,omitempty"`
	UserID    *uuid.UUID  `json:"user_id,omitempty"`
	SessionID *string     `json:"session_id,omitempty"`
	StartTime *time.Time  `json:"start_time,omitempty"`
	EndTime   *time.Time  `json:"end_time,omitempty"`
	Path      *string     `json:"path,omitempty"`
	IPAddress *string     `json:"ip_address,omitempty"`
	Limit     int         `json:"limit,omitempty"`
	Offset    int         `json:"offset,omitempty"`
}

// MetricFilter represents filters for querying metrics
type MetricFilter struct {
	Names     []string          `json:"names,omitempty"`
	Types     []MetricType      `json:"types,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	StartTime *time.Time        `json:"start_time,omitempty"`
	EndTime   *time.Time        `json:"end_time,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
}

// StatsFilter represents filters for usage statistics
type StatsFilter struct {
	Period    string     `json:"period"` // hourly, daily, weekly, monthly
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Feature   *string    `json:"feature,omitempty"`
	GroupBy   []string   `json:"group_by,omitempty"`
}
