package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig defines logger configuration
type LogConfig struct {
	Level                string
	Format               string
	Output               string
	FilePath             string
	Buffer               *bytes.Buffer
	SampleRate           float64
	MaskPII              bool
	PIIFields            []string
	Async                bool
	BufferSize           int
	SlowRequestThreshold time.Duration
	MaxSize              int // MB
	MaxBackups           int
	MaxAge               int // days
	RotateInterval       time.Duration
	EnableTracing        bool
	TracingEndpoint      string
	EnableMetrics        bool
	MetricsEndpoint      string
	ZeroAllocation       bool
	BatchSize            int
	BatchDelay           time.Duration
}

// Fields represents log fields
type Fields map[string]interface{}

// StructuredLogger implements structured logging
type StructuredLogger struct {
	config     *LogConfig
	writer     io.Writer
	level      LogLevel
	piiFields  map[string]bool
	asyncChan  chan *logMessage
	stats      *LogStats
	metrics    *LogMetrics
	mu         sync.RWMutex
	rotator    *lumberjack.Logger
	lastRotate time.Time
	batch      []*logMessage
	batchMu    sync.Mutex
	batchTimer *time.Timer
}

// LogLevel represents log level
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type logMessage struct {
	Timestamp time.Time
	Level     string
	Message   string
	Fields    Fields
	Context   map[string]interface{}
	Trace     *TraceInfo
	Error     error
}

// LogStats tracks logging statistics
type LogStats struct {
	TotalLogs int64
	Rotations int64
	Dropped   int64
	Sampled   int64
}

// LogMetrics tracks log level metrics
type LogMetrics struct {
	DebugCount int64
	InfoCount  int64
	WarnCount  int64
	ErrorCount int64
	TotalCount int64
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *LogConfig) (*StructuredLogger, error) {
	logger := &StructuredLogger{
		config:    config,
		stats:     &LogStats{},
		metrics:   &LogMetrics{},
		piiFields: make(map[string]bool),
	}

	// Set default sample rate if not specified
	if config.SampleRate == 0 {
		config.SampleRate = 1.0
	}

	// Parse log level
	logger.level = parseLevel(config.Level)

	// Setup PII fields
	for _, field := range config.PIIFields {
		logger.piiFields[field] = true
	}

	// Setup writer
	if err := logger.setupWriter(); err != nil {
		return nil, err
	}

	// Setup async logging
	if config.Async {
		bufferSize := config.BufferSize
		if bufferSize == 0 {
			bufferSize = 1000
		}
		logger.asyncChan = make(chan *logMessage, bufferSize)
		go logger.processAsync()
	}

	// Setup batching
	if config.BatchSize > 0 {
		logger.batch = make([]*logMessage, 0, config.BatchSize)
		logger.startBatchTimer()
	}

	return logger, nil
}

// NewStructuredLoggerFromFile creates a logger from config file
func NewStructuredLoggerFromFile(configFile string) (*StructuredLogger, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config LogConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return NewStructuredLogger(&config)
}

func (l *StructuredLogger) setupWriter() error {
	switch l.config.Output {
	case "stdout":
		l.writer = os.Stdout
	case "stderr":
		l.writer = os.Stderr
	case "buffer":
		if l.config.Buffer == nil {
			l.config.Buffer = &bytes.Buffer{}
		}
		l.writer = l.config.Buffer
	case "file":
		if l.config.FilePath == "" {
			return fmt.Errorf("file path required for file output")
		}
		// Create directory if needed
		dir := filepath.Dir(l.config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// Setup log rotation
		l.rotator = &lumberjack.Logger{
			Filename:   l.config.FilePath,
			MaxSize:    l.config.MaxSize,
			MaxBackups: l.config.MaxBackups,
			MaxAge:     l.config.MaxAge,
		}
		l.writer = l.rotator
		l.lastRotate = time.Now()
	default:
		l.writer = os.Stdout
	}
	return nil
}

// Debug logs debug message
func (l *StructuredLogger) Debug(ctx context.Context, message string, fields Fields) {
	if l.level > DebugLevel {
		return
	}
	atomic.AddInt64(&l.metrics.DebugCount, 1)
	l.log(ctx, "debug", message, fields, nil)
}

// Info logs info message
func (l *StructuredLogger) Info(ctx context.Context, message string, fields Fields) {
	if l.level > InfoLevel {
		return
	}
	atomic.AddInt64(&l.metrics.InfoCount, 1)
	l.log(ctx, "info", message, fields, nil)
}

// Warn logs warning message
func (l *StructuredLogger) Warn(ctx context.Context, message string, fields Fields) {
	if l.level > WarnLevel {
		return
	}
	atomic.AddInt64(&l.metrics.WarnCount, 1)
	l.log(ctx, "warn", message, fields, nil)
}

// Error logs error message
func (l *StructuredLogger) Error(ctx context.Context, message string, err error) {
	atomic.AddInt64(&l.metrics.ErrorCount, 1)
	l.log(ctx, "error", message, nil, err)
}

func (l *StructuredLogger) log(ctx context.Context, level string, message string, fields Fields, err error) {
	// Sampling (except errors)
	if level != "error" && l.config.SampleRate < 1.0 {
		if rand.Float64() > l.config.SampleRate {
			atomic.AddInt64(&l.stats.Sampled, 1)
			return
		}
	}

	// Create log message
	msg := &logMessage{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    l.maskPII(fields),
		Context:   l.extractContext(ctx),
		Trace:     l.extractTrace(ctx),
		Error:     err,
	}

	atomic.AddInt64(&l.stats.TotalLogs, 1)
	atomic.AddInt64(&l.metrics.TotalCount, 1)

	// Check rotation
	l.checkRotation()

	// Handle async or sync
	if l.config.Async {
		select {
		case l.asyncChan <- msg:
		default:
			atomic.AddInt64(&l.stats.Dropped, 1)
		}
	} else if l.config.BatchSize > 0 {
		l.addToBatch(msg)
	} else {
		l.writeMessage(msg)
	}
}

func (l *StructuredLogger) maskPII(fields Fields) Fields {
	if !l.config.MaskPII || fields == nil {
		return fields
	}

	masked := make(Fields)
	for k, v := range fields {
		if l.piiFields[k] {
			masked[k] = "***REDACTED***"
		} else {
			masked[k] = v
		}
	}
	return masked
}

func (l *StructuredLogger) extractContext(ctx context.Context) map[string]interface{} {
	context := make(map[string]interface{})

	// Extract common context values
	if reqID := ctx.Value(requestIDKey); reqID != nil {
		context["request_id"] = reqID
	}
	if userID := ctx.Value(userIDKey); userID != nil {
		context["user_id"] = userID
	}
	if traceID := ctx.Value(traceIDKey); traceID != nil {
		context["trace_id"] = traceID
	}

	return context
}

func (l *StructuredLogger) extractTrace(ctx context.Context) *TraceInfo {
	if !l.config.EnableTracing {
		return nil
	}

	if traceCtx, ok := ctx.Value(traceContextKey).(*TraceContext); ok {
		return &TraceInfo{
			TraceID: traceCtx.TraceID,
			SpanID:  traceCtx.SpanID,
		}
	}

	return nil
}

func (l *StructuredLogger) writeMessage(msg *logMessage) {
	entry := map[string]interface{}{
		"timestamp": msg.Timestamp.Format(time.RFC3339),
		"level":     msg.Level,
		"message":   msg.Message,
	}

	if msg.Fields != nil {
		entry["fields"] = msg.Fields
	}
	if len(msg.Context) > 0 {
		entry["context"] = msg.Context
	}
	if msg.Trace != nil {
		entry["trace"] = msg.Trace
	}
	if msg.Error != nil {
		entry["error"] = msg.Error.Error()
	}

	data, _ := json.Marshal(entry)
	_, _ = l.writer.Write(append(data, '\n'))
}

func (l *StructuredLogger) processAsync() {
	for msg := range l.asyncChan {
		l.writeMessage(msg)
	}
}

func (l *StructuredLogger) addToBatch(msg *logMessage) {
	l.batchMu.Lock()
	l.batch = append(l.batch, msg)

	if len(l.batch) >= l.config.BatchSize {
		l.flushBatch()
	}
	l.batchMu.Unlock()
}

func (l *StructuredLogger) flushBatch() {
	if len(l.batch) == 0 {
		return
	}

	for _, msg := range l.batch {
		l.writeMessage(msg)
	}
	l.batch = l.batch[:0]
}

func (l *StructuredLogger) startBatchTimer() {
	if l.config.BatchDelay == 0 {
		l.config.BatchDelay = 100 * time.Millisecond
	}

	l.batchTimer = time.AfterFunc(l.config.BatchDelay, func() {
		l.batchMu.Lock()
		l.flushBatch()
		l.batchMu.Unlock()
		l.startBatchTimer()
	})
}

func (l *StructuredLogger) checkRotation() {
	if l.config.RotateInterval > 0 && time.Since(l.lastRotate) > l.config.RotateInterval {
		if l.rotator != nil {
			_ = l.rotator.Rotate()
			atomic.AddInt64(&l.stats.Rotations, 1)
			l.lastRotate = time.Now()
		}
	}
}

// Flush flushes any buffered logs
func (l *StructuredLogger) Flush() {
	if l.config.Async && l.asyncChan != nil {
		close(l.asyncChan)
		time.Sleep(10 * time.Millisecond) // Wait for async processing
	}

	if l.config.BatchSize > 0 {
		l.batchMu.Lock()
		l.flushBatch()
		l.batchMu.Unlock()
	}
}

// GetStats returns logging statistics
func (l *StructuredLogger) GetStats() *LogStats {
	return &LogStats{
		TotalLogs: atomic.LoadInt64(&l.stats.TotalLogs),
		Rotations: atomic.LoadInt64(&l.stats.Rotations),
		Dropped:   atomic.LoadInt64(&l.stats.Dropped),
		Sampled:   atomic.LoadInt64(&l.stats.Sampled),
	}
}

// GetMetrics returns log metrics
func (l *StructuredLogger) GetMetrics() *LogMetrics {
	return &LogMetrics{
		DebugCount: atomic.LoadInt64(&l.metrics.DebugCount),
		InfoCount:  atomic.LoadInt64(&l.metrics.InfoCount),
		WarnCount:  atomic.LoadInt64(&l.metrics.WarnCount),
		ErrorCount: atomic.LoadInt64(&l.metrics.ErrorCount),
		TotalCount: atomic.LoadInt64(&l.metrics.TotalCount),
	}
}

// SetLevel changes log level dynamically
func (l *StructuredLogger) SetLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = parseLevel(level)
	l.config.Level = level
}

// GetConfig returns current configuration
func (l *StructuredLogger) GetConfig() *LogConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// ReloadConfig reloads configuration
func (l *StructuredLogger) ReloadConfig() error {
	// This would typically re-read from file
	// For now, just return nil
	return nil
}

// HTTPLoggingMiddleware creates HTTP logging middleware
func HTTPLoggingMiddleware(logger *StructuredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		latency := time.Since(start)
		status := c.Writer.Status()

		fields := Fields{
			"method":     c.Request.Method,
			"path":       path,
			"status":     status,
			"latency":    latency.Milliseconds(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		if raw != "" {
			fields["query"] = raw
		}

		if reqID := c.Request.Header.Get("X-Request-ID"); reqID != "" {
			fields["request_id"] = reqID
		}

		// Determine log level
		ctx := c.Request.Context()
		message := "HTTP Request"

		if status >= 500 {
			logger.Error(ctx, message, fmt.Errorf("status %d", status))
		} else if logger.config.SlowRequestThreshold > 0 && latency > logger.config.SlowRequestThreshold {
			message = "Slow HTTP Request"
			logger.Warn(ctx, message, fields)
		} else {
			logger.Info(ctx, message, fields)
		}
	}
}

// TraceContext holds trace information
type TraceContext struct {
	TraceID string
	SpanID  string
}

// TraceInfo for logging
type TraceInfo struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

// Define context keys
type contextKey string

const (
	traceContextKey contextKey = "trace_context"
	requestIDKey    contextKey = "request_id"
	userIDKey       contextKey = "user_id"
	traceIDKey      contextKey = "trace_id"
)

// WithTraceContext adds trace context
func WithTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	return context.WithValue(ctx, traceContextKey, &TraceContext{
		TraceID: traceID,
		SpanID:  spanID,
	})
}

func parseLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}
