package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventRepository defines the interface for analytics event persistence
type EventRepository interface {
	// Store stores an analytics event
	Store(ctx context.Context, event *Event) error

	// Get retrieves an event by ID
	Get(ctx context.Context, id uuid.UUID) (*Event, error)

	// List retrieves events based on filter criteria
	List(ctx context.Context, filter EventFilter) ([]*Event, int64, error)

	// GetUserEvents retrieves events for a specific user
	GetUserEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Event, int64, error)

	// GetSessionEvents retrieves events for a specific session
	GetSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]*Event, int64, error)

	// DeleteOlderThan deletes events older than specified time
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)

	// GetEventCounts gets event counts by type within time range
	GetEventCounts(ctx context.Context, startTime, endTime time.Time, groupBy string) (map[string]int64, error)
}

// MetricRepository defines the interface for metrics persistence
type MetricRepository interface {
	// Store stores a metric
	Store(ctx context.Context, metric *Metric) error

	// Get retrieves a metric by ID
	Get(ctx context.Context, id uuid.UUID) (*Metric, error)

	// List retrieves metrics based on filter criteria
	List(ctx context.Context, filter MetricFilter) ([]*Metric, int64, error)

	// GetByName retrieves metrics by name within time range
	GetByName(ctx context.Context, name string, startTime, endTime time.Time) ([]*Metric, error)

	// GetLatest retrieves the latest metric for each name
	GetLatest(ctx context.Context, names []string) ([]*Metric, error)

	// DeleteExpired deletes metrics that have exceeded their TTL
	DeleteExpired(ctx context.Context) (int64, error)

	// Aggregate performs aggregation operations on metrics
	Aggregate(ctx context.Context, name string, aggregationType string, startTime, endTime time.Time, groupBy string) (map[string]float64, error)
}

// StatsRepository defines the interface for usage statistics
type StatsRepository interface {
	// GenerateUsageStats generates usage statistics for a time period
	GenerateUsageStats(ctx context.Context, filter StatsFilter) (*UsageStats, error)

	// GetUserGrowth gets user growth statistics
	GetUserGrowth(ctx context.Context, startTime, endTime time.Time, interval string) ([]UserGrowthStats, error)

	// GetTopPages gets top visited pages
	GetTopPages(ctx context.Context, startTime, endTime time.Time, limit int) ([]PageStats, error)

	// GetTopFeatures gets top used features
	GetTopFeatures(ctx context.Context, startTime, endTime time.Time, limit int) ([]FeatureStats, error)

	// GetActiveUsers gets active user counts
	GetActiveUsers(ctx context.Context, startTime, endTime time.Time, interval string) (map[string]int64, error)
}

// AnalyticsService defines the interface for analytics operations
type AnalyticsService interface {
	// TrackEvent tracks an analytics event
	TrackEvent(ctx context.Context, event *Event) error

	// TrackUserAction tracks a user action with context
	TrackUserAction(ctx context.Context, userID uuid.UUID, action string, properties map[string]interface{}) error

	// TrackAPICall tracks an API call
	TrackAPICall(ctx context.Context, method, path string, statusCode int, responseTime time.Duration, userID *uuid.UUID, sessionID *string) error

	// RecordMetric records a metric value
	RecordMetric(ctx context.Context, metric *Metric) error

	// IncrementCounter increments a counter metric
	IncrementCounter(ctx context.Context, name string, labels map[string]string, value float64) error

	// SetGauge sets a gauge metric value
	SetGauge(ctx context.Context, name string, labels map[string]string, value float64) error

	// RecordHistogram records a histogram metric
	RecordHistogram(ctx context.Context, name string, labels map[string]string, value float64) error

	// GetEvents retrieves events based on filter
	GetEvents(ctx context.Context, filter EventFilter) ([]*Event, int64, error)

	// GetMetrics retrieves metrics based on filter
	GetMetrics(ctx context.Context, filter MetricFilter) ([]*Metric, int64, error)

	// GetUsageStats retrieves usage statistics
	GetUsageStats(ctx context.Context, filter StatsFilter) (*UsageStats, error)

	// GetDashboardData retrieves dashboard analytics data
	GetDashboardData(ctx context.Context, period string) (*DashboardData, error)

	// ExportData exports analytics data in specified format
	ExportData(ctx context.Context, filter EventFilter, format string) ([]byte, error)
}

// DashboardData represents data for analytics dashboard
type DashboardData struct {
	Summary      *SummaryStats      `json:"summary"`
	EventsChart  *ChartData         `json:"events_chart"`
	UsersChart   *ChartData         `json:"users_chart"`
	TopPages     []PageStats        `json:"top_pages"`
	TopFeatures  []FeatureStats     `json:"top_features"`
	RecentEvents []*Event           `json:"recent_events"`
	Metrics      map[string]*Metric `json:"metrics"`
	Alerts       []*Alert           `json:"alerts"`
}

// SummaryStats represents summary statistics
type SummaryStats struct {
	TotalEvents    int64   `json:"total_events"`
	TotalUsers     int64   `json:"total_users"`
	ActiveUsers    int64   `json:"active_users"`
	NewUsers       int64   `json:"new_users"`
	SessionsToday  int64   `json:"sessions_today"`
	AvgSessionTime int64   `json:"avg_session_time_seconds"`
	ErrorRate      float64 `json:"error_rate"`
	GrowthRate     float64 `json:"growth_rate"`
}

// ChartData represents data for charts
type ChartData struct {
	Labels []string `json:"labels"`
	Series []Series `json:"series"`
}

// Series represents a data series in a chart
type Series struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"`
}

// Alert represents an analytics alert
type Alert struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Triggered bool      `json:"triggered"`
	CreatedAt time.Time `json:"created_at"`
}
