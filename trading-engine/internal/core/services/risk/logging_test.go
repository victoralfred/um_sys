package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// TestRiskLogger_Creation tests logger creation and configuration
func TestRiskLogger_Creation(t *testing.T) {
	tests := []struct {
		name     string
		config   LoggerConfig
		expected LogLevel
	}{
		{
			name:     "Default configuration",
			config:   DefaultLoggerConfig(),
			expected: LogLevelInfo,
		},
		{
			name: "Debug configuration",
			config: LoggerConfig{
				Level:     LogLevelDebug,
				Format:    "json",
				Component: "test-component",
				AddSource: true,
			},
			expected: LogLevelDebug,
		},
		{
			name: "Production configuration",
			config: LoggerConfig{
				Level:        LogLevelWarn,
				Format:       "json",
				EnableColors: false,
				Component:    "risk-management",
				ServiceName:  "trading-engine",
				Environment:  "production",
			},
			expected: LogLevelWarn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewRiskLogger(tt.config)

			if logger == nil {
				t.Fatal("Expected logger to be created")
			}

			if logger.config.Level != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, logger.config.Level)
			}

			if logger.config.Component != tt.config.Component {
				t.Errorf("Expected component %v, got %v", tt.config.Component, logger.config.Component)
			}
		})
	}
}

// TestRiskLogger_ContextExtraction tests context value extraction
func TestRiskLogger_ContextExtraction(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a custom handler that writes to our buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	config := LoggerConfig{
		Level:     LogLevelDebug,
		Format:    "json",
		Component: "test-component",
	}

	logger := &RiskLogger{
		logger: slog.New(handler),
		config: config,
	}

	// Create context with values
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-12345")
	ctx = WithCorrelationID(ctx, "corr-67890")
	ctx = WithUserID(ctx, "user-999")
	ctx = WithPortfolioID(ctx, "port-abc")
	ctx = WithOperation(ctx, "VaR_calculation")

	// Create logger with context
	contextLogger := logger.WithContext(ctx)

	// Log a message
	contextLogger.Info("Test message with context")

	// Parse the JSON output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Verify context values are present
	expectedValues := map[string]string{
		"request_id":     "req-12345",
		"correlation_id": "corr-67890",
		"user_id":        "user-999",
		"portfolio_id":   "port-abc",
		"operation":      "VaR_calculation",
	}

	for key, expected := range expectedValues {
		if actual, exists := logEntry[key]; !exists {
			t.Errorf("Expected %s to be present in log output", key)
		} else if actual != expected {
			t.Errorf("Expected %s=%s, got %s=%v", key, expected, key, actual)
		}
	}
}

// TestRiskLogger_LogLevels tests different log levels
func TestRiskLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	logger := &RiskLogger{
		logger: slog.New(handler),
		config: LoggerConfig{Level: LogLevelDebug, Format: "json"},
	}

	ctx := context.Background()

	tests := []struct {
		name     string
		logFunc  func()
		level    string
		severity string
	}{
		{
			name: "Debug level",
			logFunc: func() {
				logger.DebugContext(ctx, "Debug message", slog.String("test", "debug"))
			},
			level: "DEBUG",
		},
		{
			name: "Info level",
			logFunc: func() {
				logger.InfoContext(ctx, "Info message", slog.String("test", "info"))
			},
			level: "INFO",
		},
		{
			name: "Warn level",
			logFunc: func() {
				logger.WarnContext(ctx, "Warning message", slog.String("test", "warn"))
			},
			level: "WARN",
		},
		{
			name: "Error level",
			logFunc: func() {
				logger.ErrorContext(ctx, "Error message", slog.String("test", "error"))
			},
			level: "ERROR",
		},
		{
			name: "Critical level",
			logFunc: func() {
				logger.CriticalContext(ctx, "Critical message", slog.String("test", "critical"))
			},
			level:    "ERROR", // Maps to ERROR level
			severity: "CRITICAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			if err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}

			// Check log level
			if level, exists := logEntry["level"]; !exists {
				t.Error("Expected level field in log output")
			} else if level != tt.level {
				t.Errorf("Expected level %s, got %v", tt.level, level)
			}

			// Check severity for critical logs
			if tt.severity != "" {
				if severity, exists := logEntry["severity"]; !exists {
					t.Error("Expected severity field for critical log")
				} else if severity != tt.severity {
					t.Errorf("Expected severity %s, got %v", tt.severity, severity)
				}
			}
		})
	}
}

// TestRiskLogger_ErrorIntegration tests integration with RiskError
func TestRiskLogger_ErrorIntegration(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	logger := &RiskLogger{
		logger: slog.New(handler),
		config: LoggerConfig{Level: LogLevelDebug, Format: "json"},
	}

	ctx := WithRequestID(context.Background(), "req-error-test")

	// Create a risk error
	riskErr := NewInsufficientDataError("VaR_calculation", 250, 100).
		WithRequestID("req-error-test").
		WithCorrelationID("corr-error-test").
		WithContext("portfolio_id", "PORT-123")

	// Log the error
	logger.LogError(ctx, riskErr)

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Verify error fields are logged
	expectedFields := map[string]interface{}{
		"error_code":     "INSUFFICIENT_DATA",
		"error_severity": "LOW",
		"error_category": "VALIDATION",
		"operation":      "VaR_calculation",
		"request_id":     "req-error-test",
		"correlation_id": "corr-error-test",
		"retryable":      false,
	}

	for field, expected := range expectedFields {
		if actual, exists := logEntry[field]; !exists {
			t.Errorf("Expected field %s to be present", field)
		} else if actual != expected {
			t.Errorf("Expected %s=%v, got %v", field, expected, actual)
		}
	}

	// Verify that detailed data is present
	if _, exists := logEntry["expected_data"]; !exists {
		t.Error("Expected 'expected_data' field to be present")
	}

	if _, exists := logEntry["actual_data"]; !exists {
		t.Error("Expected 'actual_data' field to be present")
	}
}

// TestRiskLogger_CalculationLogging tests calculation-specific logging
func TestRiskLogger_CalculationLogging(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	logger := &RiskLogger{
		logger: slog.New(handler),
		config: LoggerConfig{Level: LogLevelInfo, Format: "json"},
	}

	ctx := WithOperation(context.Background(), "CVaR_calculation")

	// Test calculation start logging
	buf.Reset()
	logger.LogCalculationStart(ctx, "CVaR", "historical", 1000)

	var startEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &startEntry)
	if err != nil {
		t.Fatalf("Failed to parse calculation start log: %v", err)
	}

	if startEntry["calculation_type"] != "CVaR" {
		t.Errorf("Expected calculation_type 'CVaR', got %v", startEntry["calculation_type"])
	}

	if startEntry["method"] != "historical" {
		t.Errorf("Expected method 'historical', got %v", startEntry["method"])
	}

	if startEntry["data_points"] != float64(1000) { // JSON numbers are float64
		t.Errorf("Expected data_points 1000, got %v", startEntry["data_points"])
	}

	// Test calculation complete logging - SLA compliant
	buf.Reset()
	duration := 500 * time.Microsecond // Under 1ms SLA
	result := map[string]interface{}{"cvar": 1234.56}

	logger.LogCalculationComplete(ctx, "CVaR", "historical", duration, true, result)

	var completeEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &completeEntry)
	if err != nil {
		t.Fatalf("Failed to parse calculation complete log: %v", err)
	}

	if completeEntry["success"] != true {
		t.Errorf("Expected success true, got %v", completeEntry["success"])
	}

	if completeEntry["performance"] != "COMPLIANT" {
		t.Errorf("Expected performance 'COMPLIANT', got %v", completeEntry["performance"])
	}

	// Test calculation complete logging - SLA violation
	buf.Reset()
	slowDuration := 5 * time.Millisecond // Over 1ms SLA

	logger.LogCalculationComplete(ctx, "CVaR", "historical", slowDuration, true, result)

	var slowEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &slowEntry)
	if err != nil {
		t.Fatalf("Failed to parse slow calculation log: %v", err)
	}

	if slowEntry["performance"] != "SLA_VIOLATION" {
		t.Errorf("Expected performance 'SLA_VIOLATION', got %v", slowEntry["performance"])
	}

	if _, exists := slowEntry["sla_multiplier"]; !exists {
		t.Error("Expected sla_multiplier field for SLA violation")
	}
}

// TestRiskLogger_SystemMetrics tests system metrics logging
func TestRiskLogger_SystemMetrics(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	logger := &RiskLogger{
		logger: slog.New(handler),
		config: LoggerConfig{Level: LogLevelInfo, Format: "json"},
	}

	ctx := context.Background()

	// Create sample metrics
	metrics := SystemMetrics{
		GoroutineCount:         50,
		MemoryAllocBytes:       1024 * 1024, // 1MB
		MemorySysBytes:         2048 * 1024, // 2MB
		GCCycles:               10,
		LastGCPause:            time.Millisecond,
		ConcurrentCalculations: 5,
		CPUUsagePercent:        25.5,
	}

	logger.LogSystemMetrics(ctx, metrics)

	var metricsEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &metricsEntry)
	if err != nil {
		t.Fatalf("Failed to parse metrics log: %v", err)
	}

	// Verify metrics fields
	expectedFields := map[string]interface{}{
		"goroutines":              float64(50),
		"memory_alloc":            float64(1024 * 1024),
		"concurrent_calculations": float64(5),
		"cpu_usage_percent":       25.5,
	}

	for field, expected := range expectedFields {
		if actual, exists := metricsEntry[field]; !exists {
			t.Errorf("Expected field %s to be present", field)
		} else if actual != expected {
			t.Errorf("Expected %s=%v, got %v", field, expected, actual)
		}
	}
}

// TestContextUtilities tests context utility functions
func TestContextUtilities(t *testing.T) {
	ctx := context.Background()

	// Test setting and getting context values
	tests := []struct {
		name    string
		setFunc func(context.Context, string) context.Context
		getFunc func(context.Context) string
		value   string
	}{
		{
			name:    "RequestID",
			setFunc: WithRequestID,
			getFunc: GetRequestID,
			value:   "req-12345",
		},
		{
			name:    "CorrelationID",
			setFunc: WithCorrelationID,
			getFunc: GetCorrelationID,
			value:   "corr-67890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test setting and getting
			ctxWithValue := tt.setFunc(ctx, tt.value)
			retrieved := tt.getFunc(ctxWithValue)

			if retrieved != tt.value {
				t.Errorf("Expected %s, got %s", tt.value, retrieved)
			}

			// Test getting from empty context
			empty := tt.getFunc(ctx)
			if empty != "" {
				t.Errorf("Expected empty string from empty context, got %s", empty)
			}
		})
	}
}

// TestLogLevel_String tests LogLevel string representation
func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelCritical, "CRITICAL"},
		{LogLevel(999), "UNKNOWN"}, // Unknown level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			actual := tt.level.String()
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TestGetCurrentSystemMetrics tests system metrics collection
func TestGetCurrentSystemMetrics(t *testing.T) {
	metrics := GetCurrentSystemMetrics()

	// Verify that metrics are reasonable
	if metrics.GoroutineCount <= 0 {
		t.Error("Expected positive goroutine count")
	}

	if metrics.MemoryAllocBytes == 0 {
		t.Error("Expected non-zero memory allocation")
	}

	if metrics.MemorySysBytes == 0 {
		t.Error("Expected non-zero system memory")
	}

	// Verify that metrics are consistent
	if metrics.MemorySysBytes < metrics.MemoryAllocBytes {
		t.Error("System memory should be >= allocated memory")
	}
}

// Benchmark tests for logging performance
func BenchmarkRiskLogger_SimpleLog(b *testing.B) {
	logger := NewRiskLogger(DefaultLoggerConfig())

	for i := 0; i < b.N; i++ {
		logger.Info("Simple log message", slog.Int("iteration", i))
	}
}

func BenchmarkRiskLogger_ContextLog(b *testing.B) {
	logger := NewRiskLogger(DefaultLoggerConfig())
	ctx := WithRequestID(WithCorrelationID(context.Background(), "corr-bench"), "req-bench")
	contextLogger := logger.WithContext(ctx)

	for i := 0; i < b.N; i++ {
		contextLogger.InfoContext(ctx, "Context log message",
			slog.Int("iteration", i),
			slog.String("operation", "benchmark"),
		)
	}
}

func BenchmarkRiskLogger_ErrorLog(b *testing.B) {
	logger := NewRiskLogger(DefaultLoggerConfig())
	ctx := WithRequestID(context.Background(), "req-error-bench")

	// Pre-create error to avoid construction cost in benchmark
	err := NewCalculationError("benchmark_operation",
		NewRiskError(ErrDivisionByZero, "Division by zero", "calculation"))

	for i := 0; i < b.N; i++ {
		logger.LogError(ctx, err)
	}
}

// TestDefaultLoggerConfig tests the default configuration
func TestDefaultLoggerConfig(t *testing.T) {
	config := DefaultLoggerConfig()

	// Verify default values directly
	if config.Level != LogLevelInfo {
		t.Errorf("Expected Level %v, got %v", LogLevelInfo, config.Level)
	}

	if config.Format != "json" {
		t.Errorf("Expected Format 'json', got %v", config.Format)
	}

	if config.EnableColors != false {
		t.Errorf("Expected EnableColors false, got %v", config.EnableColors)
	}

	if config.Component != "risk-management" {
		t.Errorf("Expected Component 'risk-management', got %v", config.Component)
	}

	// Verify timestamp format is valid
	_, err := time.Parse(config.TimestampFormat, time.Now().Format(config.TimestampFormat))
	if err != nil {
		t.Errorf("Invalid timestamp format: %v", err)
	}
}

// Integration test combining error handling and logging
func TestIntegration_ErrorHandlingWithLogging(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	logger := &RiskLogger{
		logger: slog.New(handler),
		config: LoggerConfig{Level: LogLevelDebug, Format: "json"},
	}

	// Simulate a complete risk calculation workflow with errors
	ctx := WithRequestID(
		WithCorrelationID(
			WithPortfolioID(
				WithOperation(context.Background(), "complete_risk_analysis"),
				"PORTFOLIO-123"),
			"corr-integration-test"),
		"req-integration-test")

	contextLogger := logger.WithContext(ctx)

	// Log calculation start
	buf.Reset()
	contextLogger.LogCalculationStart(ctx, "VaR", "historical", 100)

	// Verify start log contains all context
	if !strings.Contains(buf.String(), "req-integration-test") {
		t.Error("Expected request ID in calculation start log")
	}

	// Simulate an error during calculation
	insufficientDataError := NewInsufficientDataError("VaR_calculation", 250, 100).
		WithRequestID("req-integration-test").
		WithCorrelationID("corr-integration-test")

	buf.Reset()
	contextLogger.LogError(ctx, insufficientDataError)

	// Parse and verify error log
	var errorEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &errorEntry); err != nil {
		t.Fatalf("Failed to parse error log: %v", err)
	}

	// Verify integration of context and error information
	if errorEntry["request_id"] != "req-integration-test" {
		t.Error("Expected request ID in error log")
	}

	if errorEntry["error_code"] != "INSUFFICIENT_DATA" {
		t.Error("Expected error code in error log")
	}

	// Log calculation failure
	buf.Reset()
	contextLogger.LogCalculationComplete(ctx, "VaR", "historical",
		2*time.Millisecond, false, nil)

	if !strings.Contains(buf.String(), "SLA_VIOLATION") {
		t.Error("Expected SLA violation in failed calculation log")
	}
}
