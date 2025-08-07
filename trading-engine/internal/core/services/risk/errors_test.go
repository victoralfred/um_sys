package risk

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

// TestRiskError_CreationAndBasicProperties tests basic error creation and properties
func TestRiskError_CreationAndBasicProperties(t *testing.T) {
	tests := []struct {
		name          string
		code          ErrorCode
		message       string
		operation     string
		expectedSev   ErrorSeverity
		expectedCat   ErrorCategory
		expectRetry   bool
		expectTemp    bool
	}{
		{
			name:        "Critical data corruption error",
			code:        ErrCorruptedData,
			message:     "Data corruption detected",
			operation:   "VaR_calculation",
			expectedSev: SeverityCritical,
			expectedCat: CategoryValidation,
			expectRetry: false,
			expectTemp:  false,
		},
		{
			name:        "High severity calculation error",
			code:        ErrCalculationFailed,
			message:     "Calculation failed",
			operation:   "CVaR_calculation",
			expectedSev: SeverityHigh,
			expectedCat: CategoryCalculation,
			expectRetry: false,
			expectTemp:  false,
		},
		{
			name:        "Temporary timeout error",
			code:        ErrTimeout,
			message:     "Operation timed out",
			operation:   "risk_analysis",
			expectedSev: SeverityHigh,
			expectedCat: CategorySystem,
			expectRetry: true,
			expectTemp:  true,
		},
		{
			name:        "Low severity validation error",
			code:        ErrInvalidConfidence,
			message:     "Invalid confidence level",
			operation:   "parameter_validation",
			expectedSev: SeverityLow,
			expectedCat: CategoryValidation,
			expectRetry: false,
			expectTemp:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewRiskError(tt.code, tt.message, tt.operation)

			// Test basic properties
			if err.Code != tt.code {
				t.Errorf("Expected code %v, got %v", tt.code, err.Code)
			}

			if err.Message != tt.message {
				t.Errorf("Expected message %v, got %v", tt.message, err.Message)
			}

			if err.Details.Operation != tt.operation {
				t.Errorf("Expected operation %v, got %v", tt.operation, err.Details.Operation)
			}

			if err.Severity != tt.expectedSev {
				t.Errorf("Expected severity %v, got %v", tt.expectedSev, err.Severity)
			}

			if err.Category != tt.expectedCat {
				t.Errorf("Expected category %v, got %v", tt.expectedCat, err.Category)
			}

			// Test temporal properties
			if err.IsTemporary() != tt.expectTemp {
				t.Errorf("Expected temporary %v, got %v", tt.expectTemp, err.IsTemporary())
			}

			// Test error interface
			errorString := err.Error()
			if errorString == "" {
				t.Error("Error() should return non-empty string")
			}

			// Test timestamp
			if err.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}

			// Test component context
			if err.Context.Component != "risk-management" {
				t.Errorf("Expected component 'risk-management', got %v", err.Context.Component)
			}
		})
	}
}

// TestRiskError_FluentInterface tests the fluent interface for error building
func TestRiskError_FluentInterface(t *testing.T) {
	requestID := "req-12345"
	correlationID := "corr-67890"
	requiredObs := 250
	providedObs := 100

	err := NewRiskError(ErrInsufficientData, "Not enough data", "VaR_calculation").
		WithRequestID(requestID).
		WithCorrelationID(correlationID).
		WithDetails("provided_observations", providedObs).
		WithExpected("required_observations", requiredObs).
		WithConstraint("minimum_data", "250 observations").
		WithContext("portfolio_id", "PORT-001").
		WithCause(errors.New("database query failed"))

	// Test request context
	if err.Context.RequestID != requestID {
		t.Errorf("Expected request ID %v, got %v", requestID, err.Context.RequestID)
	}

	if err.Context.CorrelationID != correlationID {
		t.Errorf("Expected correlation ID %v, got %v", correlationID, err.Context.CorrelationID)
	}

	// Test details
	if err.Details.ActualData["provided_observations"] != providedObs {
		t.Errorf("Expected provided observations %v in actual data", providedObs)
	}

	if err.Details.ExpectedData["required_observations"] != requiredObs {
		t.Errorf("Expected required observations %v in expected data", requiredObs)
	}

	// Test constraints
	if err.Details.Constraints["minimum_data"] != "250 observations" {
		t.Error("Expected minimum data constraint")
	}

	// Test context metadata
	if err.Context.Metadata["portfolio_id"] != "PORT-001" {
		t.Error("Expected portfolio ID in metadata")
	}

	// Test cause
	if err.Cause == nil {
		t.Error("Expected cause to be set")
	}
}

// TestRiskError_RetryLogic tests retry logic and backoff calculations
func TestRiskError_RetryLogic(t *testing.T) {
	tests := []struct {
		name            string
		code            ErrorCode
		expectRetryable bool
		maxAttempts     int
		baseDelay       time.Duration
	}{
		{
			name:            "Timeout error should be retryable",
			code:            ErrTimeout,
			expectRetryable: true,
			maxAttempts:     3,
			baseDelay:       100 * time.Millisecond,
		},
		{
			name:            "Database connection error should be retryable",
			code:            ErrDatabaseConnection,
			expectRetryable: true,
			maxAttempts:     5,
			baseDelay:       250 * time.Millisecond,
		},
		{
			name:            "Validation error should not be retryable",
			code:            ErrInvalidConfidence,
			expectRetryable: false,
			maxAttempts:     0,
			baseDelay:       0,
		},
		{
			name:            "Calculation error should not be retryable",
			code:            ErrCalculationFailed,
			expectRetryable: false,
			maxAttempts:     0,
			baseDelay:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewRiskError(tt.code, "Test error", "test_operation")

			// Test retryability
			if err.IsTemporary() != tt.expectRetryable {
				t.Errorf("Expected retryable %v, got %v", tt.expectRetryable, err.IsTemporary())
			}

			if tt.expectRetryable {
				// Test retry attempts
				for attempt := 0; attempt < tt.maxAttempts; attempt++ {
					if !err.ShouldRetry(attempt) {
						t.Errorf("Should retry at attempt %d", attempt)
					}

					delay := err.GetRetryDelay(attempt)
					if delay == 0 {
						t.Errorf("Expected non-zero delay at attempt %d", attempt)
					}

					// Verify exponential backoff
					if attempt > 0 {
						prevDelay := err.GetRetryDelay(attempt - 1)
						if delay <= prevDelay {
							t.Errorf("Expected increasing delay, got %v after %v", delay, prevDelay)
						}
					}
				}

				// Test that it stops retrying after max attempts
				if err.ShouldRetry(tt.maxAttempts) {
					t.Errorf("Should not retry after %d attempts", tt.maxAttempts)
				}

				// Test retry config
				if err.RetryConfig == nil {
					t.Error("Expected retry config to be set")
				} else {
					if err.RetryConfig.MaxAttempts != tt.maxAttempts {
						t.Errorf("Expected max attempts %d, got %d", tt.maxAttempts, err.RetryConfig.MaxAttempts)
					}
				}
			} else {
				// Non-retryable errors should not have retry config
				if err.ShouldRetry(0) {
					t.Error("Non-retryable error should not allow retry")
				}
			}
		})
	}
}

// TestRiskError_ConvenienceConstructors tests convenience constructor functions
func TestRiskError_ConvenienceConstructors(t *testing.T) {
	t.Run("NewInsufficientDataError", func(t *testing.T) {
		required := 250
		provided := 100
		operation := "VaR_calculation"

		err := NewInsufficientDataError(operation, required, provided)

		if err.Code != ErrInsufficientData {
			t.Errorf("Expected code %v, got %v", ErrInsufficientData, err.Code)
		}

		if err.Details.Operation != operation {
			t.Errorf("Expected operation %v, got %v", operation, err.Details.Operation)
		}

		if err.Details.ExpectedData["min_observations"] != required {
			t.Errorf("Expected min_observations %v", required)
		}

		if err.Details.ActualData["provided_observations"] != provided {
			t.Errorf("Expected provided_observations %v", provided)
		}
	})

	t.Run("NewInvalidConfidenceError", func(t *testing.T) {
		confidence := types.NewDecimalFromFloat(105.0)
		operation := "CVaR_calculation"

		err := NewInvalidConfidenceError(operation, confidence)

		if err.Code != ErrInvalidConfidence {
			t.Errorf("Expected code %v, got %v", ErrInvalidConfidence, err.Code)
		}

		if err.Details.ActualData["confidence_level"] != confidence.String() {
			t.Errorf("Expected confidence level %v", confidence.String())
		}

		if err.Details.Constraints["valid_range"] != "0 < confidence < 100" {
			t.Error("Expected valid range constraint")
		}
	})

	t.Run("NewCalculationError", func(t *testing.T) {
		operation := "risk_calculation"
		cause := errors.New("division by zero")

		err := NewCalculationError(operation, cause)

		if err.Code != ErrCalculationFailed {
			t.Errorf("Expected code %v, got %v", ErrCalculationFailed, err.Code)
		}

		if err.Cause != cause {
			t.Errorf("Expected cause %v, got %v", cause, err.Cause)
		}
	})

	t.Run("NewTimeoutError", func(t *testing.T) {
		operation := "portfolio_analysis"
		timeout := 5 * time.Second

		err := NewTimeoutError(operation, timeout)

		if err.Code != ErrTimeout {
			t.Errorf("Expected code %v, got %v", ErrTimeout, err.Code)
		}

		if err.Details.ActualData["timeout_duration"] != timeout.String() {
			t.Errorf("Expected timeout duration %v", timeout.String())
		}
	})

	t.Run("NewSystemOverloadError", func(t *testing.T) {
		operation := "concurrent_calculations"
		currentLoad := 150
		maxLoad := 100

		err := NewSystemOverloadError(operation, currentLoad, maxLoad)

		if err.Code != ErrSystemOverload {
			t.Errorf("Expected code %v, got %v", ErrSystemOverload, err.Code)
		}

		if err.Details.ActualData["current_load"] != currentLoad {
			t.Errorf("Expected current load %v", currentLoad)
		}

		if err.Details.Constraints["max_load"] != maxLoad {
			t.Errorf("Expected max load constraint %v", maxLoad)
		}
	})
}

// TestRiskError_CriticalErrorClassification tests critical error identification
func TestRiskError_CriticalErrorClassification(t *testing.T) {
	criticalCodes := []ErrorCode{
		ErrCorruptedData,
		ErrSystemOverload,
		ErrConcurrencyLimit,
	}

	nonCriticalCodes := []ErrorCode{
		ErrInsufficientData,
		ErrInvalidConfidence,
		ErrTimeout,
		ErrDatabaseConnection,
	}

	for _, code := range criticalCodes {
		t.Run(string(code)+"_should_be_critical", func(t *testing.T) {
			err := NewRiskError(code, "Test error", "test_operation")
			if !err.IsCritical() {
				t.Errorf("Error code %v should be critical", code)
			}
		})
	}

	for _, code := range nonCriticalCodes {
		t.Run(string(code)+"_should_not_be_critical", func(t *testing.T) {
			err := NewRiskError(code, "Test error", "test_operation")
			if err.IsCritical() {
				t.Errorf("Error code %v should not be critical", code)
			}
		})
	}
}

// TestRiskError_ErrorStringFormatting tests error string representation
func TestRiskError_ErrorStringFormatting(t *testing.T) {
	err := NewRiskError(ErrCalculationFailed, "Matrix inversion failed", "portfolio_optimization")

	errorStr := err.Error()

	expectedComponents := []string{
		string(SeverityHigh),
		string(ErrCalculationFailed),
		"Matrix inversion failed",
		"portfolio_optimization",
	}

	for _, component := range expectedComponents {
		if !containsString(errorStr, component) {
			t.Errorf("Error string should contain '%s': %s", component, errorStr)
		}
	}
}

// TestRiskError_BackoffDelayProgression tests exponential backoff progression
func TestRiskError_BackoffDelayProgression(t *testing.T) {
	err := NewRiskError(ErrTimeout, "Operation timed out", "risk_calculation")

	baseDelay := 100 * time.Millisecond
	exponentialBase := 2.0

	// Test that delays increase exponentially
	for attempt := 0; attempt < 4; attempt++ {
		delay := err.GetRetryDelay(attempt)

		// Calculate expected delay (without jitter)
		expectedBase := baseDelay
		for i := 0; i < attempt; i++ {
			expectedBase = time.Duration(float64(expectedBase) * exponentialBase)
		}

		// Allow for some jitter variance
		minExpected := expectedBase
		maxExpected := expectedBase + (expectedBase / 4) // 25% jitter

		if delay < minExpected {
			t.Errorf("Delay at attempt %d too small: got %v, expected >= %v", attempt, delay, minExpected)
		}

		if delay > maxExpected {
			t.Errorf("Delay at attempt %d too large: got %v, expected <= %v", attempt, delay, maxExpected)
		}
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsString(s[1:], substr) || 
		(len(s) > len(substr) && s[:len(substr)] == substr))
}

// TestRiskError_ThreadSafety tests that error creation is thread-safe
func TestRiskError_ThreadSafety(t *testing.T) {
	// This is a basic test - in a real scenario you'd use goroutines to test concurrency
	errors := make([]*RiskError, 100)
	
	for i := 0; i < 100; i++ {
		errors[i] = NewRiskError(ErrTimeout, "Concurrent test", "thread_safety_test").
			WithRequestID(fmt.Sprintf("req-%d", i)).
			WithDetails("iteration", i)
	}
	
	// Verify all errors were created properly
	for i, err := range errors {
		if err.Context.RequestID != fmt.Sprintf("req-%d", i) {
			t.Errorf("Request ID mismatch at index %d", i)
		}
		
		if err.Details.ActualData["iteration"] != i {
			t.Errorf("Iteration mismatch at index %d", i)
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkRiskError_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRiskError(ErrCalculationFailed, "Benchmark test error", "benchmark_operation")
	}
}

func BenchmarkRiskError_FluentConstruction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRiskError(ErrTimeout, "Benchmark test", "benchmark_operation").
			WithRequestID("req-123").
			WithCorrelationID("corr-456").
			WithDetails("iteration", i).
			WithExpected("max_time", "1s").
			WithConstraint("timeout", "5s")
	}
}

func BenchmarkRiskError_RetryCalculation(b *testing.B) {
	err := NewRiskError(ErrTimeout, "Benchmark test", "benchmark_operation")
	
	for i := 0; i < b.N; i++ {
		_ = err.ShouldRetry(i % 5)
		_ = err.GetRetryDelay(i % 5)
	}
}