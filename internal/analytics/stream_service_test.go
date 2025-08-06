package analytics

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamAnalyticsService(t *testing.T) {
	ctx := context.Background()

	t.Run("Initialize stream analytics service", func(t *testing.T) {
		service, err := NewStreamAnalyticsService(getTestRedisURL())
		assert.NoError(t, err)
		assert.NotNil(t, service)

		// Test connection
		err = service.Ping(ctx)
		assert.NoError(t, err)

		// Cleanup
		_ = service.Close()
	})

	t.Run("Publish analytics event to stream", func(t *testing.T) {
		service, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service.Close() }()

		event := &AnalyticsEvent{
			Type:      "user_action",
			UserID:    uuid.New().String(),
			SessionID: uuid.New().String(),
			Properties: map[string]interface{}{
				"action": "click",
				"target": "button",
				"value":  1.0,
			},
			Metadata: map[string]string{
				"page": "/dashboard",
				"ip":   "192.168.1.1",
			},
		}

		err = service.PublishEvent(ctx, event)
		assert.NoError(t, err)
		assert.NotEmpty(t, event.ID)
		assert.False(t, event.Timestamp.IsZero())

		// Verify metrics
		metrics := service.GetMetrics()
		assert.Equal(t, int64(1), metrics.EventsPublished)
	})

	t.Run("Consume events from stream", func(t *testing.T) {
		service, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service.Close() }()

		// Publish test event
		event := &AnalyticsEvent{
			Type:   "test_event",
			UserID: "test_user",
			Properties: map[string]interface{}{
				"test": true,
			},
		}
		err = service.PublishEvent(ctx, event)
		require.NoError(t, err)

		// Consume event
		consumed := make(chan *AnalyticsEvent, 1)
		handler := &TestEventHandler{
			OnHandle: func(ctx context.Context, e *AnalyticsEvent) error {
				consumed <- e
				return nil
			},
		}

		// Start consumer in background
		consumerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		go func() { _ = service.ConsumeEvents(consumerCtx, handler) }()

		// Wait for event
		select {
		case e := <-consumed:
			assert.Equal(t, "test_event", e.Type)
			assert.Equal(t, "test_user", e.UserID)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for event")
		}

		// Verify metrics
		metrics := service.GetMetrics()
		assert.GreaterOrEqual(t, metrics.EventsConsumed, int64(1))
	})

	t.Run("Handle high volume events", func(t *testing.T) {
		service, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service.Close() }()

		// Publish 1000 events
		numEvents := 1000
		for i := 0; i < numEvents; i++ {
			event := &AnalyticsEvent{
				Type:   "bulk_test",
				UserID: uuid.New().String(),
				Properties: map[string]interface{}{
					"index": i,
					"value": float64(i),
				},
			}
			err = service.PublishEvent(ctx, event)
			require.NoError(t, err)
		}

		// Verify all published
		metrics := service.GetMetrics()
		assert.Equal(t, int64(numEvents), metrics.EventsPublished)

		// Check throughput
		assert.Greater(t, metrics.GetThroughput(), float64(100)) // At least 100 events/sec
	})

	t.Run("Event aggregation in time windows", func(t *testing.T) {
		aggregator := NewEventAggregator(NewMemoryAggregateStorage())

		// Create events in different time windows
		now := time.Now()
		events := []*AnalyticsEvent{
			{
				Type:      "api_request",
				UserID:    "user1",
				Timestamp: now,
				Properties: map[string]interface{}{
					"value":    100.0,
					"endpoint": "/api/users",
				},
			},
			{
				Type:      "api_request",
				UserID:    "user2",
				Timestamp: now.Add(30 * time.Second),
				Properties: map[string]interface{}{
					"value":    200.0,
					"endpoint": "/api/users",
				},
			},
			{
				Type:      "api_request",
				UserID:    "user3",
				Timestamp: now.Add(2 * time.Minute),
				Properties: map[string]interface{}{
					"value":    150.0,
					"endpoint": "/api/posts",
				},
			},
		}

		// Aggregate events
		for _, event := range events {
			aggregator.Aggregate(event)
		}

		// Get 1-minute window metrics
		window := aggregator.GetWindow(now.Truncate(time.Minute))
		assert.NotNil(t, window)
		assert.Equal(t, 2, len(window.Metrics)) // 2 events in first minute

		// Check aggregated values
		for _, metric := range window.Metrics {
			assert.Greater(t, metric.Count, int64(0))
			assert.Greater(t, metric.Sum, 0.0)
		}
	})

	t.Run("Stream with consumer groups", func(t *testing.T) {
		service1, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service1.Close() }()

		service2, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service2.Close() }()

		// Both services should have different consumer IDs
		assert.NotEqual(t, service1.GetConsumerID(), service2.GetConsumerID())

		// Publish event
		event := &AnalyticsEvent{
			Type:   "group_test",
			UserID: "user1",
		}
		err = service1.PublishEvent(ctx, event)
		require.NoError(t, err)

		// Only one consumer should process the event
		processed := int32(0)
		handler := &TestEventHandler{
			OnHandle: func(ctx context.Context, e *AnalyticsEvent) error {
				atomic.AddInt32(&processed, 1)
				return nil
			},
		}

		ctx1, cancel1 := context.WithTimeout(ctx, 1*time.Second)
		defer cancel1()
		ctx2, cancel2 := context.WithTimeout(ctx, 1*time.Second)
		defer cancel2()

		go func() { _ = service1.ConsumeEvents(ctx1, handler) }()
		go func() { _ = service2.ConsumeEvents(ctx2, handler) }()

		time.Sleep(2 * time.Second)
		assert.Equal(t, int32(1), atomic.LoadInt32(&processed))
	})

	t.Run("Handle stream failures and recovery", func(t *testing.T) {
		service, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service.Close() }()

		// Enable circuit breaker
		service.EnableCircuitBreaker(3, 5*time.Second)

		// Simulate failures
		failureHandler := &TestEventHandler{
			OnHandle: func(ctx context.Context, e *AnalyticsEvent) error {
				return errors.New("processing failed")
			},
		}

		// Publish event
		event := &AnalyticsEvent{Type: "failure_test"}
		_ = service.PublishEvent(ctx, event)

		// Try to consume - should fail
		ctx1, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err = service.ConsumeEvents(ctx1, failureHandler)
		assert.Error(t, err)

		// After multiple failures, circuit should open
		metrics := service.GetMetrics()
		assert.Greater(t, metrics.ProcessingErrors, int64(0))
	})

	t.Run("Stream metrics and monitoring", func(t *testing.T) {
		service, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service.Close() }()

		// Publish several events
		for i := 0; i < 10; i++ {
			event := &AnalyticsEvent{
				Type:   "metrics_test",
				UserID: uuid.New().String(),
			}
			_ = service.PublishEvent(ctx, event)
		}

		// Get metrics
		metrics := service.GetMetrics()
		assert.Equal(t, int64(10), metrics.EventsPublished)
		assert.NotZero(t, metrics.LastEventTime)

		// Get stream info
		info, err := service.GetStreamInfo(ctx, "metrics_test")
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.GreaterOrEqual(t, info.Length, int64(10))

		// Check consumer lag
		lag, err := service.GetConsumerLag(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, lag, int64(0))
	})

	t.Run("Event replay from checkpoint", func(t *testing.T) {
		service, err := NewStreamAnalyticsService(getTestRedisURL())
		require.NoError(t, err)
		defer func() { _ = service.Close() }()

		// Publish events with checkpoints
		events := []string{"event1", "event2", "event3"}
		for _, name := range events {
			event := &AnalyticsEvent{
				Type:   "replay_test",
				UserID: name,
			}
			err = service.PublishEvent(ctx, event)
			require.NoError(t, err)
		}

		// Create checkpoint after second event
		checkpoint, err := service.CreateCheckpoint(ctx, "replay_test")
		assert.NoError(t, err)
		assert.NotEmpty(t, checkpoint)

		// Publish more events
		event := &AnalyticsEvent{
			Type:   "replay_test",
			UserID: "event4",
		}
		_ = service.PublishEvent(ctx, event)

		// Replay from checkpoint
		replayed := []string{}
		replayHandler := &TestEventHandler{
			OnHandle: func(ctx context.Context, e *AnalyticsEvent) error {
				replayed = append(replayed, e.UserID)
				return nil
			},
		}

		err = service.ReplayFromCheckpoint(ctx, checkpoint, replayHandler)
		assert.NoError(t, err)
		assert.Contains(t, replayed, "event3")
		assert.Contains(t, replayed, "event4")
	})
}

// TestEventHandler is a test implementation of EventHandler
type TestEventHandler struct {
	OnHandle func(context.Context, *AnalyticsEvent) error
}

func (h *TestEventHandler) Handle(ctx context.Context, event *AnalyticsEvent) error {
	if h.OnHandle != nil {
		return h.OnHandle(ctx, event)
	}
	return nil
}

func getTestRedisURL() string {
	// Use test Redis instance
	return "redis://localhost:6379/1"
}
