package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredLogger(t *testing.T) {
	ctx := context.Background()

	t.Run("Initialize structured logger", func(t *testing.T) {
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			SampleRate: 1.0,
		}

		logger, err := NewStructuredLogger(config)
		assert.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("Log with different levels", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LogConfig{
			Level:      "debug",
			Format:     "json",
			Output:     "buffer",
			Buffer:     &buf,
			SampleRate: 1.0, // Log everything
		}

		logger, err := NewStructuredLogger(config)
		require.NoError(t, err)

		// Test different log levels
		logger.Debug(ctx, "debug message", Fields{"key": "value"})
		logger.Info(ctx, "info message", Fields{"user_id": "123"})
		logger.Warn(ctx, "warning message", Fields{"threshold": 0.8})
		logger.Error(ctx, "error message", errors.New("test error"))

		// Verify logs are structured
		logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
		assert.Len(t, logs, 4)

		for _, log := range logs {
			var entry LogEntry
			err := json.Unmarshal([]byte(log), &entry)
			assert.NoError(t, err)
			assert.NotEmpty(t, entry.Timestamp)
			assert.NotEmpty(t, entry.Level)
			assert.NotEmpty(t, entry.Message)
		}
	})

	t.Run("Context enrichment", func(t *testing.T) {
		var buf bytes.Buffer
		logger := createTestLogger(&buf)

		// Add context values
		ctx = context.WithValue(ctx, requestIDKey, "req-123")
		ctx = context.WithValue(ctx, userIDKey, "user-456")
		ctx = context.WithValue(ctx, traceIDKey, "trace-789")

		logger.Info(ctx, "operation completed", nil)

		// Verify context is included
		var entry LogEntry
		err := json.Unmarshal(buf.Bytes(), &entry)
		assert.NoError(t, err)
		assert.Equal(t, "req-123", entry.Context["request_id"])
		assert.Equal(t, "user-456", entry.Context["user_id"])
		assert.Equal(t, "trace-789", entry.Context["trace_id"])
	})

	t.Run("PII masking", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			Buffer:     &buf,
			MaskPII:    true,
			PIIFields:  []string{"email", "ssn", "credit_card", "password"},
			SampleRate: 1.0, // Log everything
		}

		logger, _ := NewStructuredLogger(config)

		fields := Fields{
			"user_id":     "123",
			"email":       "user@example.com",
			"ssn":         "123-45-6789",
			"credit_card": "4111111111111111",
			"password":    "secret123",
			"public_data": "visible",
		}

		logger.Info(ctx, "user data", fields)

		// Verify PII is masked
		var entry LogEntry
		_ = json.Unmarshal(buf.Bytes(), &entry)
		assert.Equal(t, "***REDACTED***", entry.Fields["email"])
		assert.Equal(t, "***REDACTED***", entry.Fields["ssn"])
		assert.Equal(t, "***REDACTED***", entry.Fields["credit_card"])
		assert.Equal(t, "***REDACTED***", entry.Fields["password"])
		assert.Equal(t, "visible", entry.Fields["public_data"])
		assert.Equal(t, "123", entry.Fields["user_id"])
	})

	t.Run("Sampling", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			Buffer:     &buf,
			SampleRate: 0.5, // 50% sampling
		}

		logger, _ := NewStructuredLogger(config)

		// Log many messages
		for i := 0; i < 100; i++ {
			logger.Info(ctx, "sampled message", Fields{"index": i})
		}

		// Verify approximately 50% are logged
		logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
		// Allow 40-60% range for randomness
		assert.GreaterOrEqual(t, len(logs), 40)
		assert.LessOrEqual(t, len(logs), 60)
	})

	t.Run("Error logs always sampled", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			Buffer:     &buf,
			SampleRate: 0.0, // 0% sampling
		}

		logger, _ := NewStructuredLogger(config)

		// Log info and error messages
		for i := 0; i < 10; i++ {
			logger.Info(ctx, "info message", nil)
			logger.Error(ctx, "error message", errors.New("test"))
		}

		// Only error messages should be logged
		logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
		for _, log := range logs {
			if log == "" {
				continue
			}
			var entry LogEntry
			_ = json.Unmarshal([]byte(log), &entry)
			assert.Equal(t, "error", entry.Level)
		}
	})

	t.Run("Async logging with buffer", func(t *testing.T) {
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			Async:      true,
			BufferSize: 100,
			SampleRate: 1.0, // Log everything
		}

		logger, _ := NewStructuredLogger(config)

		// Log many messages quickly
		start := time.Now()
		for i := 0; i < 1000; i++ {
			logger.Info(ctx, "async message", Fields{"index": i})
		}
		duration := time.Since(start)

		// Async logging should be fast
		assert.Less(t, duration, 100*time.Millisecond)

		// Flush and verify messages
		logger.Flush()
		stats := logger.GetStats()
		assert.Equal(t, int64(1000), stats.TotalLogs)
	})
}

func TestLoggingMiddleware(t *testing.T) {
	t.Run("HTTP request logging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := createTestLogger(&buf)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(HTTPLoggingMiddleware(logger))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test?param=value", nil)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("X-Request-ID", "req-123")

		router.ServeHTTP(w, req)

		// Verify request is logged
		var entry LogEntry
		logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
		_ = json.Unmarshal([]byte(logs[len(logs)-1]), &entry)

		assert.Equal(t, "HTTP Request", entry.Message)
		assert.Equal(t, "GET", entry.Fields["method"])
		assert.Equal(t, "/test", entry.Fields["path"])
		assert.Equal(t, float64(200), entry.Fields["status"])
		assert.NotNil(t, entry.Fields["latency"])
		assert.Equal(t, "test-agent", entry.Fields["user_agent"])
		assert.Equal(t, "req-123", entry.Fields["request_id"])
	})

	t.Run("Error response logging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := createTestLogger(&buf)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(HTTPLoggingMiddleware(logger))

		router.GET("/error", func(c *gin.Context) {
			c.JSON(500, gin.H{"error": "internal error"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/error", nil)
		router.ServeHTTP(w, req)

		// Verify error is logged at error level
		var entry LogEntry
		logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
		_ = json.Unmarshal([]byte(logs[len(logs)-1]), &entry)

		assert.Equal(t, "error", entry.Level)
		assert.Equal(t, float64(500), entry.Fields["status"])
	})

	t.Run("Slow request logging", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LogConfig{
			Level:                "info",
			Format:               "json",
			Output:               "buffer",
			Buffer:               &buf,
			SlowRequestThreshold: 100 * time.Millisecond,
			SampleRate:           1.0, // Log everything
		}
		logger, _ := NewStructuredLogger(config)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(HTTPLoggingMiddleware(logger))

		router.GET("/slow", func(c *gin.Context) {
			time.Sleep(150 * time.Millisecond)
			c.JSON(200, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/slow", nil)
		router.ServeHTTP(w, req)

		// Should log as warning due to slow response
		var entry LogEntry
		logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
		_ = json.Unmarshal([]byte(logs[len(logs)-1]), &entry)

		assert.Equal(t, "warn", entry.Level)
		assert.Contains(t, entry.Message, "Slow")
	})
}

func TestLogRotation(t *testing.T) {
	ctx := context.Background()
	t.Run("Size-based rotation", func(t *testing.T) {
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "file",
			FilePath:   "/tmp/test.log",
			MaxSize:    1, // 1 MB
			MaxBackups: 3,
			MaxAge:     7,
			SampleRate: 1.0, // Log everything
		}

		logger, err := NewStructuredLogger(config)
		assert.NoError(t, err)

		// Write enough data to trigger rotation
		largeData := strings.Repeat("x", 1024) // 1KB
		for i := 0; i < 1100; i++ {            // > 1MB
			logger.Info(ctx, "large message", Fields{"data": largeData})
		}

		// Verify rotation occurred
		stats := logger.GetStats()
		assert.Greater(t, stats.Rotations, int64(0))
	})

	t.Run("Time-based rotation", func(t *testing.T) {
		config := &LogConfig{
			Level:          "info",
			Format:         "json",
			Output:         "file",
			FilePath:       "/tmp/test-time.log",
			RotateInterval: 1 * time.Second,
			SampleRate:     1.0, // Log everything
		}

		logger, _ := NewStructuredLogger(config)

		// Log before and after rotation
		logger.Info(ctx, "before rotation", nil)
		time.Sleep(1100 * time.Millisecond)
		logger.Info(ctx, "after rotation", nil)

		stats := logger.GetStats()
		assert.Greater(t, stats.Rotations, int64(0))
	})
}

func TestOpenTelemetryIntegration(t *testing.T) {
	ctx := context.Background()
	t.Run("Trace context propagation", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LogConfig{
			Level:           "info",
			Format:          "json",
			Output:          "buffer",
			Buffer:          &buf,
			EnableTracing:   true,
			TracingEndpoint: "localhost:4317",
			SampleRate:      1.0, // Log everything
		}

		logger, _ := NewStructuredLogger(config)

		// Create span context
		traceID := uuid.New().String()
		spanID := uuid.New().String()[:16]
		ctx := WithTraceContext(ctx, traceID, spanID)

		logger.Info(ctx, "traced operation", nil)

		// Verify trace context is included
		var entry LogEntry
		_ = json.Unmarshal(buf.Bytes(), &entry)
		assert.Equal(t, traceID, entry.Trace.TraceID)
		assert.Equal(t, spanID, entry.Trace.SpanID)
	})

	t.Run("Metrics export", func(t *testing.T) {
		config := &LogConfig{
			Level:           "info",
			Format:          "json",
			EnableMetrics:   true,
			MetricsEndpoint: "localhost:8125",
			SampleRate:      1.0, // Log everything
		}

		logger, _ := NewStructuredLogger(config)

		// Log various levels
		for i := 0; i < 10; i++ {
			logger.Info(ctx, "info", nil)
			logger.Warn(ctx, "warn", nil)
			logger.Error(ctx, "error", errors.New("test"))
		}

		// Get metrics
		metrics := logger.GetMetrics()
		assert.Equal(t, int64(10), metrics.InfoCount)
		assert.Equal(t, int64(10), metrics.WarnCount)
		assert.Equal(t, int64(10), metrics.ErrorCount)
		assert.Equal(t, int64(30), metrics.TotalCount)
	})
}

func TestPerformanceOptimizations(t *testing.T) {
	ctx := context.Background()
	t.Run("Zero allocation logging", func(t *testing.T) {
		config := &LogConfig{
			Level:          "info",
			Format:         "json",
			Output:         "buffer",
			ZeroAllocation: true,
		}

		logger, _ := NewStructuredLogger(config)

		// Benchmark allocations
		allocs := testing.AllocsPerRun(100, func() {
			logger.Info(ctx, "test message", Fields{"key": "value"})
		})

		// Should have minimal allocations
		assert.Less(t, allocs, float64(5))
	})

	t.Run("Batch writing", func(t *testing.T) {
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			BatchSize:  100,
			BatchDelay: 10 * time.Millisecond,
		}

		logger, _ := NewStructuredLogger(config)

		// Log messages rapidly
		start := time.Now()
		for i := 0; i < 1000; i++ {
			logger.Info(ctx, "batch message", Fields{"index": i})
		}
		duration := time.Since(start)

		// Batching should make it fast
		assert.Less(t, duration, 50*time.Millisecond)
	})
}

func TestLoggerConfiguration(t *testing.T) {
	ctx := context.Background()
	t.Run("Dynamic level change", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "buffer",
			Buffer:     &buf,
			SampleRate: 1.0, // Log everything
		}

		logger, _ := NewStructuredLogger(config)

		// Debug should not be logged
		logger.Debug(ctx, "debug 1", nil)
		logger.Info(ctx, "info 1", nil)

		// Change level to debug
		logger.SetLevel("debug")

		// Now debug should be logged
		logger.Debug(ctx, "debug 2", nil)
		logger.Info(ctx, "info 2", nil)

		logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
		assert.Len(t, logs, 3) // info 1, debug 2, info 2
	})

	t.Run("Configuration hot reload", func(t *testing.T) {
		configFile := "/tmp/log-config.json"
		initialConfig := LogConfig{
			Level:  "info",
			Format: "json",
		}

		// Write initial config
		data, _ := json.Marshal(initialConfig)
		err := os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		logger, _ := NewStructuredLoggerFromFile(configFile)

		// Update config file
		updatedConfig := LogConfig{
			Level:   "debug",
			Format:  "json",
			MaskPII: true,
		}
		data, _ = json.Marshal(updatedConfig)
		_ = os.WriteFile(configFile, data, 0644)

		// Trigger reload
		_ = logger.ReloadConfig()

		// Verify new config is applied
		config := logger.GetConfig()
		assert.Equal(t, "debug", config.Level)
		assert.True(t, config.MaskPII)
	})
}

// Helper functions
func createTestLogger(buf *bytes.Buffer) *StructuredLogger {
	config := &LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "buffer",
		Buffer:     buf,
		SampleRate: 1.0, // Log everything
	}
	logger, _ := NewStructuredLogger(config)
	return logger
}

// Test types
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	Context   map[string]interface{} `json:"context"`
	Trace     TraceInfo              `json:"trace,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// TraceInfo is defined in structured_logger.go
