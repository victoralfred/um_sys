package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

// AnalyticsService handles analytics and metrics operations
type AnalyticsService struct {
	eventRepo  analytics.EventRepository
	metricRepo analytics.MetricRepository
	statsRepo  analytics.StatsRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(
	eventRepo analytics.EventRepository,
	metricRepo analytics.MetricRepository,
	statsRepo analytics.StatsRepository,
) *AnalyticsService {
	return &AnalyticsService{
		eventRepo:  eventRepo,
		metricRepo: metricRepo,
		statsRepo:  statsRepo,
	}
}

// TrackEvent tracks an analytics event
func (s *AnalyticsService) TrackEvent(ctx context.Context, event *analytics.Event) error {
	if event == nil {
		return analytics.ErrEventRequired
	}

	// Set default values if not provided
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	return s.eventRepo.Store(ctx, event)
}

// TrackUserAction tracks a user action with context
func (s *AnalyticsService) TrackUserAction(ctx context.Context, userID uuid.UUID, action string, properties map[string]interface{}) error {
	// Map action to event type
	eventType := s.mapActionToEventType(action)

	event := &analytics.Event{
		ID:         uuid.New(),
		Type:       eventType,
		UserID:     &userID,
		Timestamp:  time.Now(),
		Properties: properties,
		CreatedAt:  time.Now(),
	}

	return s.TrackEvent(ctx, event)
}

// TrackAPICall tracks an API call
func (s *AnalyticsService) TrackAPICall(ctx context.Context, method, path string, statusCode int, responseTime time.Duration, userID *uuid.UUID, sessionID *string) error {
	properties := map[string]interface{}{
		"method":        method,
		"path":          path,
		"status_code":   statusCode,
		"response_time": responseTime.Milliseconds(),
	}

	context := &analytics.EventContext{
		Method:       method,
		Path:         path,
		StatusCode:   &statusCode,
		ResponseTime: func() *int64 { ms := responseTime.Milliseconds(); return &ms }(),
	}

	event := &analytics.Event{
		ID:         uuid.New(),
		Type:       analytics.EventTypeAPICall,
		UserID:     userID,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
		Properties: properties,
		Context:    context,
		CreatedAt:  time.Now(),
	}

	return s.TrackEvent(ctx, event)
}

// RecordMetric records a metric value
func (s *AnalyticsService) RecordMetric(ctx context.Context, metric *analytics.Metric) error {
	if metric == nil {
		return analytics.ErrMetricRequired
	}

	// Set default values if not provided
	if metric.ID == uuid.Nil {
		metric.ID = uuid.New()
	}
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	if metric.CreatedAt.IsZero() {
		metric.CreatedAt = time.Now()
	}

	return s.metricRepo.Store(ctx, metric)
}

// IncrementCounter increments a counter metric
func (s *AnalyticsService) IncrementCounter(ctx context.Context, name string, labels map[string]string, value float64) error {
	metric := &analytics.Metric{
		ID:        uuid.New(),
		Name:      name,
		Type:      analytics.MetricTypeCounter,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	return s.RecordMetric(ctx, metric)
}

// SetGauge sets a gauge metric value
func (s *AnalyticsService) SetGauge(ctx context.Context, name string, labels map[string]string, value float64) error {
	metric := &analytics.Metric{
		ID:        uuid.New(),
		Name:      name,
		Type:      analytics.MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	return s.RecordMetric(ctx, metric)
}

// RecordHistogram records a histogram metric
func (s *AnalyticsService) RecordHistogram(ctx context.Context, name string, labels map[string]string, value float64) error {
	metric := &analytics.Metric{
		ID:        uuid.New(),
		Name:      name,
		Type:      analytics.MetricTypeHistogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	return s.RecordMetric(ctx, metric)
}

// GetEvents retrieves events based on filter
func (s *AnalyticsService) GetEvents(ctx context.Context, filter analytics.EventFilter) ([]*analytics.Event, int64, error) {
	return s.eventRepo.List(ctx, filter)
}

// GetMetrics retrieves metrics based on filter
func (s *AnalyticsService) GetMetrics(ctx context.Context, filter analytics.MetricFilter) ([]*analytics.Metric, int64, error) {
	return s.metricRepo.List(ctx, filter)
}

// GetUsageStats retrieves usage statistics
func (s *AnalyticsService) GetUsageStats(ctx context.Context, filter analytics.StatsFilter) (*analytics.UsageStats, error) {
	return s.statsRepo.GenerateUsageStats(ctx, filter)
}

// GetDashboardData retrieves dashboard analytics data
func (s *AnalyticsService) GetDashboardData(ctx context.Context, period string) (*analytics.DashboardData, error) {
	now := time.Now()
	var startTime time.Time

	// Calculate time range based on period
	switch period {
	case "hourly":
		startTime = now.Add(-24 * time.Hour)
	case "daily":
		startTime = now.AddDate(0, 0, -30)
	case "weekly":
		startTime = now.AddDate(0, 0, -90)
	case "monthly":
		startTime = now.AddDate(0, -12, 0)
	default:
		startTime = now.AddDate(0, 0, -7)
		period = "daily"
	}

	// Generate usage stats
	statsFilter := analytics.StatsFilter{
		Period:    period,
		StartTime: startTime,
		EndTime:   now,
	}

	usageStats, err := s.statsRepo.GenerateUsageStats(ctx, statsFilter)
	if err != nil {
		return nil, err
	}

	// Get recent events
	eventFilter := analytics.EventFilter{
		StartTime: &startTime,
		EndTime:   &now,
		Limit:     10,
	}

	recentEvents, _, err := s.eventRepo.List(ctx, eventFilter)
	if err != nil {
		return nil, err
	}

	// Get latest metrics
	metricNames := []string{
		"api_response_time",
		"error_rate",
		"active_users",
		"memory_usage",
		"cpu_usage",
	}

	latestMetrics, err := s.metricRepo.GetLatest(ctx, metricNames)
	if err != nil {
		return nil, err
	}

	// Convert metrics to map
	metricsMap := make(map[string]*analytics.Metric)
	for _, metric := range latestMetrics {
		metricsMap[metric.Name] = metric
	}

	// Create summary stats
	summary := &analytics.SummaryStats{
		TotalEvents:   usageStats.TotalEvents,
		TotalUsers:    usageStats.UniqueUsers,
		ActiveUsers:   usageStats.UniqueUsers, // Simplified
		SessionsToday: usageStats.TotalSessions,
		ErrorRate:     s.calculateErrorRate(usageStats.EventsByType),
		GrowthRate:    s.calculateGrowthRate(usageStats.UserGrowth),
	}

	if usageStats.AvgSessionTime > 0 {
		summary.AvgSessionTime = int64(usageStats.AvgSessionTime.Seconds())
	}

	// Create chart data
	eventsChart := s.createEventsChartData(usageStats.EventsByType)
	usersChart := s.createUsersChartData(usageStats.UserGrowth)

	return &analytics.DashboardData{
		Summary:      summary,
		EventsChart:  eventsChart,
		UsersChart:   usersChart,
		TopPages:     usageStats.TopPages,
		TopFeatures:  usageStats.TopFeatures,
		RecentEvents: recentEvents,
		Metrics:      metricsMap,
		Alerts:       []*analytics.Alert{}, // Placeholder for alerts
	}, nil
}

// ExportData exports analytics data in specified format
func (s *AnalyticsService) ExportData(ctx context.Context, filter analytics.EventFilter, format string) ([]byte, error) {
	events, _, err := s.eventRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return s.exportAsJSON(events)
	case "csv":
		return s.exportAsCSV(events)
	default:
		return nil, analytics.ErrExportFailed
	}
}

// Helper methods

func (s *AnalyticsService) mapActionToEventType(action string) analytics.EventType {
	switch action {
	case "login":
		return analytics.EventTypeUserLogin
	case "logout":
		return analytics.EventTypeUserLogout
	case "register", "registration":
		return analytics.EventTypeUserRegistration
	case "page_view":
		return analytics.EventTypePageView
	case "button_click":
		return analytics.EventTypeButtonClick
	case "form_submit":
		return analytics.EventTypeFormSubmit
	case "feature_usage":
		return analytics.EventTypeFeatureUsage
	case "error":
		return analytics.EventTypeError
	default:
		return analytics.EventTypeFeatureUsage
	}
}

func (s *AnalyticsService) calculateErrorRate(eventsByType map[analytics.EventType]int64) float64 {
	totalEvents := int64(0)
	errorEvents := int64(0)

	for eventType, count := range eventsByType {
		totalEvents += count
		if eventType == analytics.EventTypeError {
			errorEvents += count
		}
	}

	if totalEvents == 0 {
		return 0.0
	}

	return float64(errorEvents) / float64(totalEvents) * 100
}

func (s *AnalyticsService) calculateGrowthRate(userGrowth []analytics.UserGrowthStats) float64 {
	if len(userGrowth) < 2 {
		return 0.0
	}

	latest := userGrowth[len(userGrowth)-1]
	previous := userGrowth[len(userGrowth)-2]

	if previous.NewUsers == 0 {
		return 0.0
	}

	return float64(latest.NewUsers-previous.NewUsers) / float64(previous.NewUsers) * 100
}

func (s *AnalyticsService) createEventsChartData(eventsByType map[analytics.EventType]int64) *analytics.ChartData {
	labels := make([]string, 0, len(eventsByType))
	data := make([]float64, 0, len(eventsByType))

	for eventType, count := range eventsByType {
		labels = append(labels, string(eventType))
		data = append(data, float64(count))
	}

	return &analytics.ChartData{
		Labels: labels,
		Series: []analytics.Series{
			{
				Name: "Events",
				Data: data,
			},
		},
	}
}

func (s *AnalyticsService) createUsersChartData(userGrowth []analytics.UserGrowthStats) *analytics.ChartData {
	labels := make([]string, len(userGrowth))
	newUsersData := make([]float64, len(userGrowth))
	activeUsersData := make([]float64, len(userGrowth))

	for i, stat := range userGrowth {
		labels[i] = stat.Date.Format("2006-01-02")
		newUsersData[i] = float64(stat.NewUsers)
		activeUsersData[i] = float64(stat.ActiveUsers)
	}

	return &analytics.ChartData{
		Labels: labels,
		Series: []analytics.Series{
			{
				Name: "New Users",
				Data: newUsersData,
			},
			{
				Name: "Active Users",
				Data: activeUsersData,
			},
		},
	}
}

func (s *AnalyticsService) exportAsJSON(events []*analytics.Event) ([]byte, error) {
	// Implementation would use encoding/json
	// Simplified for now
	return []byte("{}"), nil
}

func (s *AnalyticsService) exportAsCSV(events []*analytics.Event) ([]byte, error) {
	// Implementation would create CSV format
	// Simplified for now
	return []byte(""), nil
}
