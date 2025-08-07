package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"
)

// LogLevel represents different logging levels for risk management operations
type LogLevel int

const (
	LogLevelDebug LogLevel = iota - 1
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelCritical
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// LoggerConfig contains configuration for the structured logger
type LoggerConfig struct {
	Level           LogLevel `json:"level"`
	Format          string   `json:"format"` // "json" or "text"
	EnableColors    bool     `json:"enable_colors"`
	TimestampFormat string   `json:"timestamp_format"`
	AddSource       bool     `json:"add_source"`
	Component       string   `json:"component"`
	ServiceName     string   `json:"service_name"`
	ServiceVersion  string   `json:"service_version"`
	Environment     string   `json:"environment"`
}

// DefaultLoggerConfig returns a production-ready logger configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:           LogLevelInfo,
		Format:          "json",
		EnableColors:    false,
		TimestampFormat: time.RFC3339Nano,
		AddSource:       true,
		Component:       "risk-management",
		ServiceName:     "trading-engine",
		ServiceVersion:  "1.0.0",
		Environment:     "production",
	}
}

// ContextKey represents keys used for context values
type ContextKey string

const (
	ContextKeyRequestID     ContextKey = "request_id"
	ContextKeyCorrelationID ContextKey = "correlation_id"
	ContextKeyUserID        ContextKey = "user_id"
	ContextKeySessionID     ContextKey = "session_id"
	ContextKeyPortfolioID   ContextKey = "portfolio_id"
	ContextKeyOperation     ContextKey = "operation"
)

// RiskLogger provides structured logging specifically designed for risk management operations
type RiskLogger struct {
	logger *slog.Logger
	config LoggerConfig
}

// NewRiskLogger creates a new structured logger for risk management operations
func NewRiskLogger(config LoggerConfig) *RiskLogger {
	// Create handler based on configuration
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     slog.Level(config.Level),
		AddSource: config.AddSource,
	}

	if config.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// Wrap with context handler to add service metadata
	contextHandler := NewContextHandler(handler, map[string]interface{}{
		"service":   config.ServiceName,
		"version":   config.ServiceVersion,
		"component": config.Component,
		"env":       config.Environment,
		"timestamp": time.Now().Format(config.TimestampFormat),
	})

	logger := slog.New(contextHandler)

	return &RiskLogger{
		logger: logger,
		config: config,
	}
}

// WithContext adds context values to the logger
func (rl *RiskLogger) WithContext(ctx context.Context) *RiskLogger {
	attrs := []slog.Attr{}

	// Extract known context values
	if requestID := ctx.Value(ContextKeyRequestID); requestID != nil {
		attrs = append(attrs, slog.String("request_id", requestID.(string)))
	}

	if correlationID := ctx.Value(ContextKeyCorrelationID); correlationID != nil {
		attrs = append(attrs, slog.String("correlation_id", correlationID.(string)))
	}

	if userID := ctx.Value(ContextKeyUserID); userID != nil {
		attrs = append(attrs, slog.String("user_id", userID.(string)))
	}

	if sessionID := ctx.Value(ContextKeySessionID); sessionID != nil {
		attrs = append(attrs, slog.String("session_id", sessionID.(string)))
	}

	if portfolioID := ctx.Value(ContextKeyPortfolioID); portfolioID != nil {
		attrs = append(attrs, slog.String("portfolio_id", portfolioID.(string)))
	}

	if operation := ctx.Value(ContextKeyOperation); operation != nil {
		attrs = append(attrs, slog.String("operation", operation.(string)))
	}

	// Create new logger with context
	contextLogger := rl.logger.With(convertAttrsToAny(attrs)...)

	return &RiskLogger{
		logger: contextLogger,
		config: rl.config,
	}
}

// Debug logs debug-level messages for development and troubleshooting
func (rl *RiskLogger) Debug(msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(context.Background(), slog.LevelDebug, msg, attrs...)
}

// DebugContext logs debug-level messages with context
func (rl *RiskLogger) DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// Info logs informational messages for normal operations
func (rl *RiskLogger) Info(msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}

// InfoContext logs informational messages with context
func (rl *RiskLogger) InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn logs warning messages for potentially problematic situations
func (rl *RiskLogger) Warn(msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
}

// WarnContext logs warning messages with context
func (rl *RiskLogger) WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// Error logs error messages for failures and exceptions
func (rl *RiskLogger) Error(msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

// ErrorContext logs error messages with context
func (rl *RiskLogger) ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

// Critical logs critical messages for system-threatening issues
func (rl *RiskLogger) Critical(msg string, attrs ...slog.Attr) {
	// Map to ERROR level since slog doesn't have CRITICAL
	rl.logger.LogAttrs(context.Background(), slog.LevelError, msg,
		append(attrs, slog.String("severity", "CRITICAL"))...)
}

// CriticalContext logs critical messages with context
func (rl *RiskLogger) CriticalContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	rl.logger.LogAttrs(ctx, slog.LevelError, msg,
		append(attrs, slog.String("severity", "CRITICAL"))...)
}

// LogError logs a RiskError with full context and metadata
func (rl *RiskLogger) LogError(ctx context.Context, err *RiskError) {
	attrs := []slog.Attr{
		slog.String("error_code", string(err.Code)),
		slog.String("error_severity", string(err.Severity)),
		slog.String("error_category", string(err.Category)),
		slog.String("operation", err.Details.Operation),
		slog.Time("error_timestamp", err.Timestamp),
	}

	// Add error details
	if len(err.Details.ActualData) > 0 {
		if jsonData, jsonErr := json.Marshal(err.Details.ActualData); jsonErr == nil {
			attrs = append(attrs, slog.String("actual_data", string(jsonData)))
		}
	}

	if len(err.Details.ExpectedData) > 0 {
		if jsonData, jsonErr := json.Marshal(err.Details.ExpectedData); jsonErr == nil {
			attrs = append(attrs, slog.String("expected_data", string(jsonData)))
		}
	}

	if len(err.Details.Constraints) > 0 {
		if jsonData, jsonErr := json.Marshal(err.Details.Constraints); jsonErr == nil {
			attrs = append(attrs, slog.String("constraints", string(jsonData)))
		}
	}

	// Add error context
	if err.Context.RequestID != "" {
		attrs = append(attrs, slog.String("request_id", err.Context.RequestID))
	}

	if err.Context.CorrelationID != "" {
		attrs = append(attrs, slog.String("correlation_id", err.Context.CorrelationID))
	}

	// Add retry information (always include retryable status)
	attrs = append(attrs, slog.Bool("retryable", err.IsTemporary()))

	// Add retry configuration if available
	if err.RetryConfig != nil {
		attrs = append(attrs,
			slog.Int("max_retry_attempts", err.RetryConfig.MaxAttempts),
			slog.Duration("base_backoff", err.RetryConfig.BackoffDelay),
		)
	}

	// Add underlying cause if present
	if err.Cause != nil {
		attrs = append(attrs, slog.String("underlying_cause", err.Cause.Error()))
	}

	// Log at appropriate level based on severity
	switch err.Severity {
	case SeverityCritical:
		rl.CriticalContext(ctx, err.Message, attrs...)
	case SeverityHigh:
		rl.ErrorContext(ctx, err.Message, attrs...)
	case SeverityMedium:
		rl.WarnContext(ctx, err.Message, attrs...)
	default:
		rl.InfoContext(ctx, err.Message, attrs...)
	}
}

// LogCalculationStart logs the beginning of a risk calculation
func (rl *RiskLogger) LogCalculationStart(ctx context.Context, calculationType, method string, dataPoints int) {
	rl.InfoContext(ctx, "Risk calculation started",
		slog.String("calculation_type", calculationType),
		slog.String("method", method),
		slog.Int("data_points", dataPoints),
		slog.Time("start_time", time.Now()),
	)
}

// LogCalculationComplete logs the completion of a risk calculation with performance metrics
func (rl *RiskLogger) LogCalculationComplete(ctx context.Context, calculationType, method string,
	duration time.Duration, success bool, result interface{}) {
	attrs := []slog.Attr{
		slog.String("calculation_type", calculationType),
		slog.String("method", method),
		slog.Duration("duration", duration),
		slog.Bool("success", success),
		slog.Time("end_time", time.Now()),
	}

	// Add performance classification
	slaThreshold := time.Millisecond
	if duration <= slaThreshold {
		attrs = append(attrs, slog.String("performance", "COMPLIANT"))
	} else {
		attrs = append(attrs, slog.String("performance", "SLA_VIOLATION"))
		attrs = append(attrs, slog.Float64("sla_multiplier", float64(duration.Nanoseconds())/float64(slaThreshold.Nanoseconds())))
	}

	// Add result summary if provided
	if result != nil {
		if jsonResult, err := json.Marshal(result); err == nil {
			attrs = append(attrs, slog.String("result_summary", string(jsonResult)))
		}
	}

	if success {
		rl.InfoContext(ctx, "Risk calculation completed successfully", attrs...)
	} else {
		rl.WarnContext(ctx, "Risk calculation completed with issues", attrs...)
	}
}

// LogSystemMetrics logs system performance and resource usage metrics
func (rl *RiskLogger) LogSystemMetrics(ctx context.Context, metrics SystemMetrics) {
	rl.InfoContext(ctx, "System metrics report",
		slog.Int("goroutines", metrics.GoroutineCount),
		slog.Uint64("memory_alloc", metrics.MemoryAllocBytes),
		slog.Uint64("memory_sys", metrics.MemorySysBytes),
		slog.Uint64("gc_cycles", metrics.GCCycles),
		slog.Duration("gc_pause", metrics.LastGCPause),
		slog.Int("concurrent_calculations", metrics.ConcurrentCalculations),
		slog.Float64("cpu_usage_percent", metrics.CPUUsagePercent),
	)
}

// SystemMetrics contains system performance metrics
type SystemMetrics struct {
	GoroutineCount         int           `json:"goroutine_count"`
	MemoryAllocBytes       uint64        `json:"memory_alloc_bytes"`
	MemorySysBytes         uint64        `json:"memory_sys_bytes"`
	GCCycles               uint64        `json:"gc_cycles"`
	LastGCPause            time.Duration `json:"last_gc_pause"`
	ConcurrentCalculations int           `json:"concurrent_calculations"`
	CPUUsagePercent        float64       `json:"cpu_usage_percent"`
}

// GetCurrentSystemMetrics collects current system performance metrics
func GetCurrentSystemMetrics() SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemMetrics{
		GoroutineCount:   runtime.NumGoroutine(),
		MemoryAllocBytes: memStats.Alloc,
		MemorySysBytes:   memStats.Sys,
		GCCycles:         uint64(memStats.NumGC),
		LastGCPause:      time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256]),
		// Note: ConcurrentCalculations and CPUUsagePercent would be populated
		// by application-specific monitoring
	}
}

// ContextHandler wraps slog.Handler to add persistent context attributes
type ContextHandler struct {
	handler    slog.Handler
	attributes map[string]interface{}
}

// NewContextHandler creates a new context handler with persistent attributes
func NewContextHandler(handler slog.Handler, attributes map[string]interface{}) *ContextHandler {
	return &ContextHandler{
		handler:    handler,
		attributes: attributes,
	}
}

// Enabled reports whether the handler handles records at the given level
func (ch *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return ch.handler.Enabled(ctx, level)
}

// Handle handles the Record, adding persistent context attributes
func (ch *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	// Add persistent attributes to the record
	for key, value := range ch.attributes {
		switch v := value.(type) {
		case string:
			record.AddAttrs(slog.String(key, v))
		case int:
			record.AddAttrs(slog.Int(key, v))
		case int64:
			record.AddAttrs(slog.Int64(key, v))
		case float64:
			record.AddAttrs(slog.Float64(key, v))
		case bool:
			record.AddAttrs(slog.Bool(key, v))
		case time.Time:
			record.AddAttrs(slog.Time(key, v))
		case time.Duration:
			record.AddAttrs(slog.Duration(key, v))
		default:
			// Convert to string for unknown types
			record.AddAttrs(slog.String(key, fmt.Sprintf("%v", v)))
		}
	}

	return ch.handler.Handle(ctx, record)
}

// WithAttrs returns a new Handler whose attributes consist of both the receiver's
// attributes and the arguments
func (ch *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		handler:    ch.handler.WithAttrs(attrs),
		attributes: ch.attributes,
	}
}

// WithGroup returns a new Handler with the given group appended to the receiver's
// existing groups
func (ch *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		handler:    ch.handler.WithGroup(name),
		attributes: ch.attributes,
	}
}

// Utility functions for context management

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, ContextKeyCorrelationID, correlationID)
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// WithPortfolioID adds a portfolio ID to the context
func WithPortfolioID(ctx context.Context, portfolioID string) context.Context {
	return context.WithValue(ctx, ContextKeyPortfolioID, portfolioID)
}

// WithOperation adds an operation name to the context
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, ContextKeyOperation, operation)
}

// GetRequestID extracts the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID := ctx.Value(ContextKeyRequestID); requestID != nil {
		return requestID.(string)
	}
	return ""
}

// GetCorrelationID extracts the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if correlationID := ctx.Value(ContextKeyCorrelationID); correlationID != nil {
		return correlationID.(string)
	}
	return ""
}

// convertAttrsToAny converts slog.Attr slice to []any for With method
func convertAttrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, attr := range attrs {
		result[i] = attr
	}
	return result
}
