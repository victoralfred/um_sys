package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

// MockEventRepository is a mock implementation of analytics.EventRepository
type MockEventRepository struct {
	mock.Mock
}

func (m *MockEventRepository) Store(ctx context.Context, event *analytics.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventRepository) Get(ctx context.Context, id uuid.UUID) (*analytics.Event, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*analytics.Event), args.Error(1)
}

func (m *MockEventRepository) List(ctx context.Context, filter analytics.EventFilter) ([]*analytics.Event, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*analytics.Event), args.Get(1).(int64), args.Error(2)
}

func (m *MockEventRepository) GetUserEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*analytics.Event, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*analytics.Event), args.Get(1).(int64), args.Error(2)
}

func (m *MockEventRepository) GetSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]*analytics.Event, int64, error) {
	args := m.Called(ctx, sessionID, limit, offset)
	return args.Get(0).([]*analytics.Event), args.Get(1).(int64), args.Error(2)
}

func (m *MockEventRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockEventRepository) GetEventCounts(ctx context.Context, startTime, endTime time.Time, groupBy string) (map[string]int64, error) {
	args := m.Called(ctx, startTime, endTime, groupBy)
	return args.Get(0).(map[string]int64), args.Error(1)
}

// MockMetricRepository is a mock implementation of analytics.MetricRepository
type MockMetricRepository struct {
	mock.Mock
}

func (m *MockMetricRepository) Store(ctx context.Context, metric *analytics.Metric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockMetricRepository) Get(ctx context.Context, id uuid.UUID) (*analytics.Metric, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*analytics.Metric), args.Error(1)
}

func (m *MockMetricRepository) List(ctx context.Context, filter analytics.MetricFilter) ([]*analytics.Metric, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*analytics.Metric), args.Get(1).(int64), args.Error(2)
}

func (m *MockMetricRepository) GetByName(ctx context.Context, name string, startTime, endTime time.Time) ([]*analytics.Metric, error) {
	args := m.Called(ctx, name, startTime, endTime)
	return args.Get(0).([]*analytics.Metric), args.Error(1)
}

func (m *MockMetricRepository) GetLatest(ctx context.Context, names []string) ([]*analytics.Metric, error) {
	args := m.Called(ctx, names)
	return args.Get(0).([]*analytics.Metric), args.Error(1)
}

func (m *MockMetricRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMetricRepository) Aggregate(ctx context.Context, name string, aggregationType string, startTime, endTime time.Time, groupBy string) (map[string]float64, error) {
	args := m.Called(ctx, name, aggregationType, startTime, endTime, groupBy)
	return args.Get(0).(map[string]float64), args.Error(1)
}

// MockStatsRepository is a mock implementation of analytics.StatsRepository
type MockStatsRepository struct {
	mock.Mock
}

func (m *MockStatsRepository) GenerateUsageStats(ctx context.Context, filter analytics.StatsFilter) (*analytics.UsageStats, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*analytics.UsageStats), args.Error(1)
}

func (m *MockStatsRepository) GetUserGrowth(ctx context.Context, startTime, endTime time.Time, interval string) ([]analytics.UserGrowthStats, error) {
	args := m.Called(ctx, startTime, endTime, interval)
	return args.Get(0).([]analytics.UserGrowthStats), args.Error(1)
}

func (m *MockStatsRepository) GetTopPages(ctx context.Context, startTime, endTime time.Time, limit int) ([]analytics.PageStats, error) {
	args := m.Called(ctx, startTime, endTime, limit)
	return args.Get(0).([]analytics.PageStats), args.Error(1)
}

func (m *MockStatsRepository) GetTopFeatures(ctx context.Context, startTime, endTime time.Time, limit int) ([]analytics.FeatureStats, error) {
	args := m.Called(ctx, startTime, endTime, limit)
	return args.Get(0).([]analytics.FeatureStats), args.Error(1)
}

func (m *MockStatsRepository) GetActiveUsers(ctx context.Context, startTime, endTime time.Time, interval string) (map[string]int64, error) {
	args := m.Called(ctx, startTime, endTime, interval)
	return args.Get(0).(map[string]int64), args.Error(1)
}

// TestAnalyticsService tests
func TestNewAnalyticsService(t *testing.T) {
	eventRepo := &MockEventRepository{}
	metricRepo := &MockMetricRepository{}
	statsRepo := &MockStatsRepository{}

	service := NewAnalyticsService(eventRepo, metricRepo, statsRepo)

	assert.NotNil(t, service)
}

func TestAnalyticsService_TrackEvent(t *testing.T) {
	ctx := context.Background()
	eventRepo := &MockEventRepository{}
	metricRepo := &MockMetricRepository{}
	statsRepo := &MockStatsRepository{}

	service := NewAnalyticsService(eventRepo, metricRepo, statsRepo)

	t.Run("successful event tracking", func(t *testing.T) {
		event := &analytics.Event{
			ID:        uuid.New(),
			Type:      analytics.EventTypeUserLogin,
			UserID:    &uuid.UUID{},
			Timestamp: time.Now(),
		}

		eventRepo.On("Store", ctx, event).Return(nil)

		err := service.TrackEvent(ctx, event)
		assert.NoError(t, err)
		eventRepo.AssertExpectations(t)
	})

	t.Run("nil event error", func(t *testing.T) {
		err := service.TrackEvent(ctx, nil)
		assert.Equal(t, analytics.ErrEventRequired, err)
	})
}

func TestAnalyticsService_TrackUserAction(t *testing.T) {
	ctx := context.Background()
	eventRepo := &MockEventRepository{}
	metricRepo := &MockMetricRepository{}
	statsRepo := &MockStatsRepository{}

	service := NewAnalyticsService(eventRepo, metricRepo, statsRepo)

	t.Run("successful user action tracking", func(t *testing.T) {
		userID := uuid.New()
		action := "button_click"
		properties := map[string]interface{}{
			"button_id": "login-btn",
			"page":      "/login",
		}

		eventRepo.On("Store", ctx, mock.MatchedBy(func(event *analytics.Event) bool {
			return event.Type == analytics.EventTypeButtonClick &&
				event.UserID != nil &&
				*event.UserID == userID
		})).Return(nil)

		err := service.TrackUserAction(ctx, userID, action, properties)
		assert.NoError(t, err)
		eventRepo.AssertExpectations(t)
	})
}

func TestAnalyticsService_RecordMetric(t *testing.T) {
	ctx := context.Background()
	eventRepo := &MockEventRepository{}
	metricRepo := &MockMetricRepository{}
	statsRepo := &MockStatsRepository{}

	service := NewAnalyticsService(eventRepo, metricRepo, statsRepo)

	t.Run("successful metric recording", func(t *testing.T) {
		metric := &analytics.Metric{
			ID:        uuid.New(),
			Name:      "api_response_time",
			Type:      analytics.MetricTypeHistogram,
			Value:     250.5,
			Timestamp: time.Now(),
		}

		metricRepo.On("Store", ctx, metric).Return(nil)

		err := service.RecordMetric(ctx, metric)
		assert.NoError(t, err)
		metricRepo.AssertExpectations(t)
	})

	t.Run("nil metric error", func(t *testing.T) {
		err := service.RecordMetric(ctx, nil)
		assert.Equal(t, analytics.ErrMetricRequired, err)
	})
}

func TestAnalyticsService_IncrementCounter(t *testing.T) {
	ctx := context.Background()
	eventRepo := &MockEventRepository{}
	metricRepo := &MockMetricRepository{}
	statsRepo := &MockStatsRepository{}

	service := NewAnalyticsService(eventRepo, metricRepo, statsRepo)

	t.Run("successful counter increment", func(t *testing.T) {
		name := "user_logins"
		labels := map[string]string{"method": "email"}
		value := 1.0

		metricRepo.On("Store", ctx, mock.MatchedBy(func(metric *analytics.Metric) bool {
			return metric.Name == name &&
				metric.Type == analytics.MetricTypeCounter &&
				metric.Value == value
		})).Return(nil)

		err := service.IncrementCounter(ctx, name, labels, value)
		assert.NoError(t, err)
		metricRepo.AssertExpectations(t)
	})
}

func TestAnalyticsService_GetUsageStats(t *testing.T) {
	ctx := context.Background()
	eventRepo := &MockEventRepository{}
	metricRepo := &MockMetricRepository{}
	statsRepo := &MockStatsRepository{}

	service := NewAnalyticsService(eventRepo, metricRepo, statsRepo)

	t.Run("successful usage stats retrieval", func(t *testing.T) {
		filter := analytics.StatsFilter{
			Period:    "daily",
			StartTime: time.Now().AddDate(0, 0, -7),
			EndTime:   time.Now(),
		}

		expectedStats := &analytics.UsageStats{
			Period:        "daily",
			TotalEvents:   1000,
			UniqueUsers:   250,
			TotalSessions: 300,
		}

		statsRepo.On("GenerateUsageStats", ctx, filter).Return(expectedStats, nil)

		stats, err := service.GetUsageStats(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, expectedStats, stats)
		statsRepo.AssertExpectations(t)
	})
}

func TestAnalyticsService_GetDashboardData(t *testing.T) {
	ctx := context.Background()
	eventRepo := &MockEventRepository{}
	metricRepo := &MockMetricRepository{}
	statsRepo := &MockStatsRepository{}

	service := NewAnalyticsService(eventRepo, metricRepo, statsRepo)

	t.Run("successful dashboard data retrieval", func(t *testing.T) {
		period := "daily"

		// Mock the various repository calls that GetDashboardData would make
		statsRepo.On("GenerateUsageStats", ctx, mock.AnythingOfType("analytics.StatsFilter")).Return(&analytics.UsageStats{}, nil)
		eventRepo.On("List", ctx, mock.AnythingOfType("analytics.EventFilter")).Return([]*analytics.Event{}, int64(0), nil)
		metricRepo.On("GetLatest", ctx, mock.AnythingOfType("[]string")).Return([]*analytics.Metric{}, nil)

		dashboard, err := service.GetDashboardData(ctx, period)
		assert.NoError(t, err)
		assert.NotNil(t, dashboard)
	})
}
