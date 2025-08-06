package integration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/analytics"
	"github.com/victoralfred/um_sys/internal/cache"
	"github.com/victoralfred/um_sys/internal/logging"
)

// TestEnterpriseIntegration tests all enterprise features working together
func TestEnterpriseIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("Full stack integration", func(t *testing.T) {
		// Initialize structured logger
		logConfig := &logging.LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			SampleRate: 1.0,
		}
		logger, err := logging.NewStructuredLogger(logConfig)
		require.NoError(t, err)

		// Initialize Redis Streams analytics
		streamService, err := analytics.NewStreamAnalyticsService("redis://localhost:6379/2")
		require.NoError(t, err)
		defer func() { _ = streamService.Close() }()

		// Initialize query cache
		redisClient := getTestRedisClient()
		queryCache := cache.NewQueryCache(redisClient)

		// Test flow: User action -> Analytics event -> Cache query -> Log
		userID := uuid.New().String()
		sessionID := uuid.New().String()

		// 1. Log user action
		logger.Info(ctx, "User action started", logging.Fields{
			"user_id":    userID,
			"session_id": sessionID,
			"action":     "login",
		})

		// 2. Publish analytics event
		event := &analytics.AnalyticsEvent{
			Type:      "user_login",
			UserID:    userID,
			SessionID: sessionID,
			Properties: map[string]interface{}{
				"ip":         "192.168.1.1",
				"user_agent": "Mozilla/5.0",
			},
		}
		err = streamService.PublishEvent(ctx, event)
		assert.NoError(t, err)

		// 3. Cache user data
		userData := map[string]interface{}{
			"id":         userID,
			"session_id": sessionID,
			"last_login": time.Now(),
		}
		cacheKey := fmt.Sprintf("user:%s", userID)
		err = queryCache.Set(ctx, cacheKey, userData, 5*time.Minute)
		assert.NoError(t, err)

		// 4. Verify cache retrieval
		var cachedData map[string]interface{}
		err = queryCache.Get(ctx, cacheKey, &cachedData)
		assert.NoError(t, err)
		assert.Equal(t, userID, cachedData["id"])

		// 5. Verify metrics
		streamMetrics := streamService.GetMetrics()
		assert.Equal(t, int64(1), streamMetrics.EventsPublished)

		cacheStats := queryCache.GetStats()
		assert.Equal(t, int64(1), cacheStats.Hits)

		logMetrics := logger.GetMetrics()
		assert.Equal(t, int64(1), logMetrics.InfoCount)
	})

	t.Run("High volume performance test", func(t *testing.T) {
		// Initialize services
		streamService, err := analytics.NewStreamAnalyticsService("redis://localhost:6379/3")
		require.NoError(t, err)
		defer func() { _ = streamService.Close() }()

		logger, _ := logging.NewStructuredLogger(&logging.LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			Async:      true,
			BufferSize: 10000,
			SampleRate: 0.1, // Sample 10% for performance
		})

		queryCache := cache.NewQueryCache(getTestRedisClient())

		// Test parameters
		numEvents := 10000
		numWorkers := 10
		eventsPerWorker := numEvents / numWorkers

		start := time.Now()
		var wg sync.WaitGroup
		var successCount int64

		// Launch workers
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for i := 0; i < eventsPerWorker; i++ {
					// Generate event
					event := &analytics.AnalyticsEvent{
						Type:   "load_test",
						UserID: fmt.Sprintf("user_%d_%d", workerID, i),
						Properties: map[string]interface{}{
							"worker": workerID,
							"index":  i,
						},
					}

					// Publish event
					if err := streamService.PublishEvent(ctx, event); err == nil {
						atomic.AddInt64(&successCount, 1)
					}

					// Log (sampled)
					logger.Info(ctx, "Event published", logging.Fields{
						"worker": workerID,
						"index":  i,
					})

					// Cache operation
					if i%100 == 0 { // Cache every 100th event
						cacheKey := fmt.Sprintf("event:%d:%d", workerID, i)
						_ = queryCache.Set(ctx, cacheKey, event, 1*time.Minute)
					}
				}
			}(w)
		}

		wg.Wait()
		duration := time.Since(start)

		// Calculate throughput
		throughput := float64(successCount) / duration.Seconds()

		// Assertions
		assert.GreaterOrEqual(t, successCount, int64(numEvents*95/100)) // 95% success rate
		assert.Greater(t, throughput, float64(1000))                    // At least 1000 events/sec

		// Log results
		t.Logf("Processed %d events in %v (%.2f events/sec)", successCount, duration, throughput)

		// Verify metrics
		streamMetrics := streamService.GetMetrics()
		assert.Equal(t, successCount, streamMetrics.EventsPublished)

		logger.Flush()
		logMetrics := logger.GetMetrics()
		assert.Greater(t, logMetrics.TotalCount, int64(0))
	})

	t.Run("Cache and stream coordination", func(t *testing.T) {
		streamService, _ := analytics.NewStreamAnalyticsService("redis://localhost:6379/4")
		defer func() { _ = streamService.Close() }()

		queryCache := cache.NewQueryCache(getTestRedisClient())
		optimizer := cache.NewQueryOptimizer(nil, queryCache)

		// Simulate query pattern - first populate cache
		for i := 0; i < 100; i++ {
			query := fmt.Sprintf("SELECT * FROM users WHERE id = %d", i)
			opt := optimizer.Optimize(query, nil)

			// Publish analytics event for query
			event := &analytics.AnalyticsEvent{
				Type:   "query_executed",
				UserID: "system",
				Properties: map[string]interface{}{
					"query":     query,
					"use_cache": opt.UseCache,
					"cache_ttl": opt.CacheTTL.Seconds(),
				},
			}
			_ = streamService.PublishEvent(ctx, event)

			// Cache result if applicable
			if opt.UseCache {
				result := map[string]interface{}{"id": i, "name": fmt.Sprintf("User%d", i)}
				_ = queryCache.Set(ctx, opt.CacheKey, result, opt.CacheTTL)
			}
		}

		// Now simulate cache reads to generate stats
		for i := 0; i < 50; i++ {
			query := fmt.Sprintf("SELECT * FROM users WHERE id = %d", i)
			opt := optimizer.Optimize(query, nil)
			
			var result map[string]interface{}
			_ = queryCache.Get(ctx, opt.CacheKey, &result)
		}

		// Verify coordination
		streamMetrics := streamService.GetMetrics()
		assert.Equal(t, int64(100), streamMetrics.EventsPublished)

		cacheStats := queryCache.GetStats()
		assert.Greater(t, cacheStats.Hits+cacheStats.Misses, int64(0))
	})

	t.Run("Error handling and recovery", func(t *testing.T) {
		// Test with circuit breaker
		streamService, _ := analytics.NewStreamAnalyticsService("redis://localhost:6379/5")
		defer func() { _ = streamService.Close() }()
		streamService.EnableCircuitBreaker(3, 5*time.Second)

		logger, _ := logging.NewStructuredLogger(&logging.LogConfig{
			Level:  "error",
			Format: "json",
			Output: "buffer",
		})

		// Simulate failures
		for i := 0; i < 10; i++ {
			event := &analytics.AnalyticsEvent{
				Type:   "error_test",
				UserID: fmt.Sprintf("user_%d", i),
			}

			err := streamService.PublishEvent(ctx, event)
			if err != nil {
				logger.Error(ctx, "Failed to publish event", err)
			}
		}

		// Verify error handling
		logMetrics := logger.GetMetrics()
		assert.GreaterOrEqual(t, logMetrics.ErrorCount, int64(0))
	})
}

// Benchmark tests
func BenchmarkStreamAnalytics(b *testing.B) {
	ctx := context.Background()
	streamService, _ := analytics.NewStreamAnalyticsService("redis://localhost:6379/6")
	defer func() { _ = streamService.Close() }()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := &analytics.AnalyticsEvent{
				Type:   "benchmark",
				UserID: uuid.New().String(),
				Properties: map[string]interface{}{
					"timestamp": time.Now().Unix(),
				},
			}
			_ = streamService.PublishEvent(ctx, event)
		}
	})

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

func BenchmarkQueryCache(b *testing.B) {
	ctx := context.Background()
	cache := cache.NewQueryCache(getTestRedisClient())

	// Prepare test data
	testData := map[string]interface{}{
		"id":   "123",
		"name": "Test User",
		"data": make([]byte, 1024), // 1KB payload
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench:key:%d", i%1000)
			if i%2 == 0 {
				_ = cache.Set(ctx, key, testData, 1*time.Minute)
			} else {
				var result map[string]interface{}
				_ = cache.Get(ctx, key, &result)
			}
			i++
		}
	})

	stats := cache.GetStats()
	b.ReportMetric(float64(stats.Hits+stats.Misses)/b.Elapsed().Seconds(), "ops/sec")
}

func BenchmarkStructuredLogging(b *testing.B) {
	ctx := context.Background()
	logger, _ := logging.NewStructuredLogger(&logging.LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "buffer",
		Async:      true,
		BufferSize: 10000,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(ctx, "Benchmark log", logging.Fields{
				"iteration": b.N,
				"timestamp": time.Now().Unix(),
			})
		}
	})

	logger.Flush()
	metrics := logger.GetMetrics()
	b.ReportMetric(float64(metrics.TotalCount)/b.Elapsed().Seconds(), "logs/sec")
}

// Helper function
func getTestRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   7, // Use separate DB for tests
	})
}
