package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	UserID     string                 `json:"user_id"`
	SessionID  string                 `json:"session_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
	Metadata   map[string]string      `json:"metadata"`
}

// StreamMetrics tracks stream analytics metrics
type StreamMetrics struct {
	EventsPublished  int64
	EventsConsumed   int64
	ProcessingErrors int64
	LastEventTime    time.Time
	startTime        time.Time
}

// GetThroughput calculates events per second
func (m *StreamMetrics) GetThroughput() float64 {
	duration := time.Since(m.startTime).Seconds()
	if duration == 0 {
		return 0
	}
	return float64(m.EventsPublished) / duration
}

// StreamAnalyticsService handles real-time analytics with Redis Streams
type StreamAnalyticsService struct {
	client         *redis.Client
	consumerGroup  string
	consumerID     string
	metrics        *StreamMetrics
	mu             sync.RWMutex
	circuitBreaker *CircuitBreaker
	shutdownCh     chan struct{}
}

// NewStreamAnalyticsService creates a new stream analytics service
func NewStreamAnalyticsService(redisURL string) (*StreamAnalyticsService, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	service := &StreamAnalyticsService{
		client:        client,
		consumerGroup: "analytics-group",
		consumerID:    uuid.New().String(),
		metrics: &StreamMetrics{
			startTime: time.Now(),
		},
		shutdownCh: make(chan struct{}),
	}

	// Initialize consumer group
	ctx := context.Background()
	_ = client.XGroupCreateMkStream(ctx, "analytics:stream", service.consumerGroup, "0").Err()

	return service, nil
}

// Ping tests the connection
func (s *StreamAnalyticsService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close closes the service
func (s *StreamAnalyticsService) Close() error {
	close(s.shutdownCh)
	return s.client.Close()
}

// PublishEvent publishes an analytics event to the stream
func (s *StreamAnalyticsService) PublishEvent(ctx context.Context, event *AnalyticsEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	streamKey := fmt.Sprintf("analytics:stream:%s", event.Type)
	_, err = s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	atomic.AddInt64(&s.metrics.EventsPublished, 1)
	s.metrics.LastEventTime = time.Now()
	return nil
}

// ConsumeEvents consumes events from the stream
func (s *StreamAnalyticsService) ConsumeEvents(ctx context.Context, handler EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.shutdownCh:
			return nil
		default:
			// Read from multiple streams
			streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    s.consumerGroup,
				Consumer: s.consumerID,
				Streams:  []string{"analytics:stream:>", ">"},
				Count:    10,
				Block:    1 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue
				}
				if s.circuitBreaker != nil && s.circuitBreaker.ShouldTrip() {
					return errors.New("circuit breaker open")
				}
				atomic.AddInt64(&s.metrics.ProcessingErrors, 1)
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					if err := s.processMessage(ctx, handler, message); err != nil {
						atomic.AddInt64(&s.metrics.ProcessingErrors, 1)
						if s.circuitBreaker != nil {
							s.circuitBreaker.RecordFailure()
						}
					}
				}
			}
		}
	}
}

func (s *StreamAnalyticsService) processMessage(ctx context.Context, handler EventHandler, message redis.XMessage) error {
	data, ok := message.Values["data"].(string)
	if !ok {
		return errors.New("invalid message format")
	}

	var event AnalyticsEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if err := handler.Handle(ctx, &event); err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	atomic.AddInt64(&s.metrics.EventsConsumed, 1)

	// Acknowledge message
	streamKey := strings.Split(message.ID, "-")[0]
	_ = s.client.XAck(ctx, streamKey, s.consumerGroup, message.ID)

	return nil
}

// GetMetrics returns stream metrics
func (s *StreamAnalyticsService) GetMetrics() *StreamMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}

// GetConsumerID returns the consumer ID
func (s *StreamAnalyticsService) GetConsumerID() string {
	return s.consumerID
}

// EnableCircuitBreaker enables circuit breaker for fault tolerance
func (s *StreamAnalyticsService) EnableCircuitBreaker(threshold int, timeout time.Duration) {
	s.circuitBreaker = &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
	}
}

// GetStreamInfo returns information about a stream
func (s *StreamAnalyticsService) GetStreamInfo(ctx context.Context, streamType string) (*StreamInfo, error) {
	streamKey := fmt.Sprintf("analytics:stream:%s", streamType)
	info, err := s.client.XInfoStream(ctx, streamKey).Result()
	if err != nil {
		return nil, err
	}

	return &StreamInfo{
		Length:         info.Length,
		ConsumerGroups: info.Groups,
	}, nil
}

// GetConsumerLag returns the consumer lag
func (s *StreamAnalyticsService) GetConsumerLag(ctx context.Context) (int64, error) {
	// Get pending messages for this consumer
	pending, err := s.client.XPending(ctx, "analytics:stream:*", s.consumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

// CreateCheckpoint creates a checkpoint for replay
func (s *StreamAnalyticsService) CreateCheckpoint(ctx context.Context, streamType string) (string, error) {
	streamKey := fmt.Sprintf("analytics:stream:%s", streamType)

	// Get last message ID
	messages, err := s.client.XRevRangeN(ctx, streamKey, "+", "-", 1).Result()
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return "0", nil
	}

	return messages[0].ID, nil
}

// ReplayFromCheckpoint replays events from a checkpoint
func (s *StreamAnalyticsService) ReplayFromCheckpoint(ctx context.Context, checkpoint string, handler EventHandler) error {
	streamKey := "analytics:stream:replay_test"

	// Read messages after checkpoint
	messages, err := s.client.XRange(ctx, streamKey, "("+checkpoint, "+").Result()
	if err != nil {
		return err
	}

	for _, message := range messages {
		if err := s.processMessage(ctx, handler, message); err != nil {
			return err
		}
	}

	return nil
}

// EventHandler interface for processing events
type EventHandler interface {
	Handle(ctx context.Context, event *AnalyticsEvent) error
}

// StreamInfo contains stream information
type StreamInfo struct {
	Length         int64
	ConsumerGroups int64
}

// CircuitBreaker implements a simple circuit breaker
type CircuitBreaker struct {
	failures  int32
	threshold int
	timeout   time.Duration
	lastFail  time.Time
	mu        sync.Mutex
}

func (cb *CircuitBreaker) RecordFailure() {
	atomic.AddInt32(&cb.failures, 1)
	cb.mu.Lock()
	cb.lastFail = time.Now()
	cb.mu.Unlock()
}

func (cb *CircuitBreaker) ShouldTrip() bool {
	failures := atomic.LoadInt32(&cb.failures)
	if int(failures) >= cb.threshold {
		cb.mu.Lock()
		defer cb.mu.Unlock()
		if time.Since(cb.lastFail) < cb.timeout {
			return true
		}
		atomic.StoreInt32(&cb.failures, 0)
	}
	return false
}

// EventAggregator aggregates events in time windows
type EventAggregator struct {
	storage AggregateStorage
	mu      sync.RWMutex
}

func NewEventAggregator(storage AggregateStorage) *EventAggregator {
	return &EventAggregator{
		storage: storage,
	}
}

func (a *EventAggregator) Aggregate(event *AnalyticsEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	window := event.Timestamp.Truncate(time.Minute)
	a.storage.AddToWindow(window, event)
}

func (a *EventAggregator) GetWindow(timestamp time.Time) *AggregateWindow {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.storage.GetWindow(timestamp)
}

// AggregateStorage interface for storing aggregated data
type AggregateStorage interface {
	AddToWindow(window time.Time, event *AnalyticsEvent)
	GetWindow(window time.Time) *AggregateWindow
}

// MemoryAggregateStorage implements in-memory aggregate storage
type MemoryAggregateStorage struct {
	windows map[time.Time]*AggregateWindow
	mu      sync.RWMutex
}

func NewMemoryAggregateStorage() *MemoryAggregateStorage {
	return &MemoryAggregateStorage{
		windows: make(map[time.Time]*AggregateWindow),
	}
}

func (m *MemoryAggregateStorage) AddToWindow(window time.Time, event *AnalyticsEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.windows[window]; !exists {
		m.windows[window] = &AggregateWindow{
			Timestamp: window,
			Metrics:   make(map[string]*Metric),
		}
	}

	w := m.windows[window]
	key := fmt.Sprintf("%s:%s", event.Type, event.UserID)

	if _, exists := w.Metrics[key]; !exists {
		w.Metrics[key] = &Metric{}
	}

	metric := w.Metrics[key]
	metric.Count++

	if val, ok := event.Properties["value"].(float64); ok {
		metric.Sum += val
	}
}

func (m *MemoryAggregateStorage) GetWindow(window time.Time) *AggregateWindow {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.windows[window]
}

// AggregateWindow represents aggregated data for a time window
type AggregateWindow struct {
	Timestamp time.Time
	Metrics   map[string]*Metric
}

// Metric represents aggregated metrics
type Metric struct {
	Count int64
	Sum   float64
}
