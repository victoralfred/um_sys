package risk

import (
	"fmt"
	"time"

	"github.com/trading-engine/pkg/types"
)

// ErrorCode represents a categorized error code for risk management operations
type ErrorCode string

const (
	// Data validation errors
	ErrInsufficientData     ErrorCode = "INSUFFICIENT_DATA"
	ErrInvalidConfidence    ErrorCode = "INVALID_CONFIDENCE"
	ErrInvalidPortfolio     ErrorCode = "INVALID_PORTFOLIO"
	ErrCorruptedData        ErrorCode = "CORRUPTED_DATA"
	ErrMissingRequiredField ErrorCode = "MISSING_REQUIRED_FIELD"

	// Calculation errors
	ErrCalculationFailed    ErrorCode = "CALCULATION_FAILED"
	ErrNumericalInstability ErrorCode = "NUMERICAL_INSTABILITY"
	ErrDivisionByZero       ErrorCode = "DIVISION_BY_ZERO"
	ErrOverflow             ErrorCode = "OVERFLOW"
	ErrUnderflow            ErrorCode = "UNDERFLOW"

	// Configuration errors
	ErrInvalidConfig        ErrorCode = "INVALID_CONFIG"
	ErrUnsupportedMethod    ErrorCode = "UNSUPPORTED_METHOD"
	ErrConfigurationMissing ErrorCode = "CONFIGURATION_MISSING"

	// System errors
	ErrTimeout          ErrorCode = "TIMEOUT"
	ErrSystemOverload   ErrorCode = "SYSTEM_OVERLOAD"
	ErrResourceLimited  ErrorCode = "RESOURCE_LIMITED"
	ErrConcurrencyLimit ErrorCode = "CONCURRENCY_LIMIT"

	// External dependency errors
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrCacheUnavailable   ErrorCode = "CACHE_UNAVAILABLE"
	ErrNetworkFailure     ErrorCode = "NETWORK_FAILURE"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	// Business logic errors
	ErrRiskLimitExceeded     ErrorCode = "RISK_LIMIT_EXCEEDED"
	ErrInvalidRiskParameters ErrorCode = "INVALID_RISK_PARAMETERS"
	ErrModelValidationFailed ErrorCode = "MODEL_VALIDATION_FAILED"
)

// ErrorSeverity indicates the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "LOW"      // Non-critical, informational
	SeverityMedium   ErrorSeverity = "MEDIUM"   // Important but recoverable
	SeverityHigh     ErrorSeverity = "HIGH"     // Critical, requires attention
	SeverityCritical ErrorSeverity = "CRITICAL" // System-threatening, immediate action required
)

// ErrorCategory groups related error types
type ErrorCategory string

const (
	CategoryValidation    ErrorCategory = "VALIDATION"
	CategoryCalculation   ErrorCategory = "CALCULATION"
	CategoryConfiguration ErrorCategory = "CONFIGURATION"
	CategorySystem        ErrorCategory = "SYSTEM"
	CategoryDependency    ErrorCategory = "DEPENDENCY"
	CategoryBusiness      ErrorCategory = "BUSINESS"
)

// RiskError is a comprehensive error type for risk management operations
// Implements the error interface with extensive metadata for production debugging
type RiskError struct {
	Code        ErrorCode     `json:"code"`
	Message     string        `json:"message"`
	Severity    ErrorSeverity `json:"severity"`
	Category    ErrorCategory `json:"category"`
	Details     ErrorDetails  `json:"details"`
	Context     ErrorContext  `json:"context"`
	Timestamp   time.Time     `json:"timestamp"`
	RetryConfig *RetryConfig  `json:"retry_config,omitempty"`
	Cause       error         `json:"cause,omitempty"` // Underlying error if any
}

// ErrorDetails contains specific information about the error
type ErrorDetails struct {
	Operation    string                 `json:"operation"`               // Which operation failed
	InputData    map[string]interface{} `json:"input_data,omitempty"`    // Input parameters (sanitized)
	ExpectedData map[string]interface{} `json:"expected_data,omitempty"` // What was expected
	ActualData   map[string]interface{} `json:"actual_data,omitempty"`   // What was actually received
	Constraints  map[string]interface{} `json:"constraints,omitempty"`   // Violated constraints
}

// ErrorContext provides contextual information for debugging and monitoring
type ErrorContext struct {
	RequestID     string            `json:"request_id,omitempty"`
	UserID        string            `json:"user_id,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Component     string            `json:"component"`
	Method        string            `json:"method"`
	StackTrace    []string          `json:"stack_trace,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// RetryConfig specifies retry behavior for recoverable errors
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	BackoffDelay    time.Duration `json:"backoff_delay"`
	MaxBackoff      time.Duration `json:"max_backoff"`
	ExponentialBase float64       `json:"exponential_base"`
	Jitter          bool          `json:"jitter"`
}

// NewRiskError creates a new RiskError with proper initialization
func NewRiskError(code ErrorCode, message string, operation string) *RiskError {
	return &RiskError{
		Code:      code,
		Message:   message,
		Severity:  determineSeverity(code),
		Category:  determineCategory(code),
		Timestamp: time.Now(),
		Details: ErrorDetails{
			Operation: operation,
		},
		Context: ErrorContext{
			Component: "risk-management",
		},
		RetryConfig: determineRetryConfig(code),
	}
}

// Error implements the error interface
func (re *RiskError) Error() string {
	return fmt.Sprintf("[%s] %s: %s (operation: %s)",
		re.Severity, re.Code, re.Message, re.Details.Operation)
}

// WithDetails adds detailed information to the error
func (re *RiskError) WithDetails(key string, value interface{}) *RiskError {
	if re.Details.ActualData == nil {
		re.Details.ActualData = make(map[string]interface{})
	}
	re.Details.ActualData[key] = value
	return re
}

// WithExpected adds expected value information
func (re *RiskError) WithExpected(key string, value interface{}) *RiskError {
	if re.Details.ExpectedData == nil {
		re.Details.ExpectedData = make(map[string]interface{})
	}
	re.Details.ExpectedData[key] = value
	return re
}

// WithConstraint adds constraint violation information
func (re *RiskError) WithConstraint(key string, value interface{}) *RiskError {
	if re.Details.Constraints == nil {
		re.Details.Constraints = make(map[string]interface{})
	}
	re.Details.Constraints[key] = value
	return re
}

// WithContext adds contextual information
func (re *RiskError) WithContext(key string, value string) *RiskError {
	if re.Context.Metadata == nil {
		re.Context.Metadata = make(map[string]string)
	}
	re.Context.Metadata[key] = value
	return re
}

// WithRequestID sets the request ID for tracing
func (re *RiskError) WithRequestID(requestID string) *RiskError {
	re.Context.RequestID = requestID
	return re
}

// WithCorrelationID sets the correlation ID for distributed tracing
func (re *RiskError) WithCorrelationID(correlationID string) *RiskError {
	re.Context.CorrelationID = correlationID
	return re
}

// WithCause wraps an underlying error
func (re *RiskError) WithCause(cause error) *RiskError {
	re.Cause = cause
	return re
}

// IsTemporary indicates if the error condition is temporary and can be retried
func (re *RiskError) IsTemporary() bool {
	switch re.Code {
	case ErrTimeout, ErrSystemOverload, ErrResourceLimited,
		ErrDatabaseConnection, ErrCacheUnavailable,
		ErrNetworkFailure, ErrServiceUnavailable:
		return true
	default:
		return false
	}
}

// IsCritical indicates if the error requires immediate attention
func (re *RiskError) IsCritical() bool {
	return re.Severity == SeverityCritical
}

// ShouldRetry determines if the operation should be retried based on error type and attempt count
func (re *RiskError) ShouldRetry(attemptCount int) bool {
	if !re.IsTemporary() {
		return false
	}

	if re.RetryConfig == nil {
		return false
	}

	return attemptCount < re.RetryConfig.MaxAttempts
}

// GetRetryDelay calculates the delay before next retry attempt
func (re *RiskError) GetRetryDelay(attemptCount int) time.Duration {
	if re.RetryConfig == nil {
		return 0
	}

	// Exponential backoff with jitter
	delay := re.RetryConfig.BackoffDelay
	for i := 0; i < attemptCount; i++ {
		delay = time.Duration(float64(delay) * re.RetryConfig.ExponentialBase)
	}

	if delay > re.RetryConfig.MaxBackoff {
		delay = re.RetryConfig.MaxBackoff
	}

	if re.RetryConfig.Jitter {
		// Add up to 25% jitter to prevent thundering herd
		jitterRange := delay / 4
		delay += time.Duration(time.Now().UnixNano() % int64(jitterRange))
	}

	return delay
}

// Helper functions to determine error properties

func determineSeverity(code ErrorCode) ErrorSeverity {
	switch code {
	case ErrCorruptedData, ErrSystemOverload, ErrConcurrencyLimit:
		return SeverityCritical
	case ErrCalculationFailed, ErrNumericalInstability, ErrTimeout,
		ErrRiskLimitExceeded, ErrModelValidationFailed:
		return SeverityHigh
	case ErrInvalidConfig, ErrUnsupportedMethod, ErrDatabaseConnection,
		ErrServiceUnavailable:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func determineCategory(code ErrorCode) ErrorCategory {
	switch code {
	case ErrInsufficientData, ErrInvalidConfidence, ErrInvalidPortfolio,
		ErrCorruptedData, ErrMissingRequiredField:
		return CategoryValidation
	case ErrCalculationFailed, ErrNumericalInstability, ErrDivisionByZero,
		ErrOverflow, ErrUnderflow:
		return CategoryCalculation
	case ErrInvalidConfig, ErrUnsupportedMethod, ErrConfigurationMissing:
		return CategoryConfiguration
	case ErrTimeout, ErrSystemOverload, ErrResourceLimited, ErrConcurrencyLimit:
		return CategorySystem
	case ErrDatabaseConnection, ErrCacheUnavailable, ErrNetworkFailure,
		ErrServiceUnavailable:
		return CategoryDependency
	case ErrRiskLimitExceeded, ErrInvalidRiskParameters, ErrModelValidationFailed:
		return CategoryBusiness
	default:
		return CategorySystem
	}
}

func determineRetryConfig(code ErrorCode) *RetryConfig {
	switch code {
	case ErrTimeout, ErrSystemOverload, ErrResourceLimited:
		return &RetryConfig{
			MaxAttempts:     3,
			BackoffDelay:    100 * time.Millisecond,
			MaxBackoff:      5 * time.Second,
			ExponentialBase: 2.0,
			Jitter:          true,
		}
	case ErrDatabaseConnection, ErrNetworkFailure:
		return &RetryConfig{
			MaxAttempts:     5,
			BackoffDelay:    250 * time.Millisecond,
			MaxBackoff:      10 * time.Second,
			ExponentialBase: 1.5,
			Jitter:          true,
		}
	case ErrServiceUnavailable, ErrCacheUnavailable:
		return &RetryConfig{
			MaxAttempts:     2,
			BackoffDelay:    500 * time.Millisecond,
			MaxBackoff:      2 * time.Second,
			ExponentialBase: 1.5,
			Jitter:          false,
		}
	default:
		return nil // No retry for non-temporary errors
	}
}

// Convenience constructors for common error scenarios

// NewInsufficientDataError creates an error for insufficient historical data
func NewInsufficientDataError(operation string, required, provided int) *RiskError {
	return NewRiskError(ErrInsufficientData,
		fmt.Sprintf("insufficient historical data for %s", operation), operation).
		WithExpected("min_observations", required).
		WithDetails("provided_observations", provided).
		WithConstraint("minimum_required", required)
}

// NewInvalidConfidenceError creates an error for invalid confidence levels
func NewInvalidConfidenceError(operation string, confidence types.Decimal) *RiskError {
	return NewRiskError(ErrInvalidConfidence,
		fmt.Sprintf("invalid confidence level for %s", operation), operation).
		WithDetails("confidence_level", confidence.String()).
		WithConstraint("valid_range", "0 < confidence < 100")
}

// NewCalculationError creates an error for calculation failures
func NewCalculationError(operation string, cause error) *RiskError {
	return NewRiskError(ErrCalculationFailed,
		fmt.Sprintf("calculation failed for %s", operation), operation).
		WithCause(cause)
}

// NewTimeoutError creates an error for operation timeouts
func NewTimeoutError(operation string, timeout time.Duration) *RiskError {
	return NewRiskError(ErrTimeout,
		fmt.Sprintf("operation %s timed out", operation), operation).
		WithDetails("timeout_duration", timeout.String())
}

// NewSystemOverloadError creates an error for system overload conditions
func NewSystemOverloadError(operation string, currentLoad, maxLoad int) *RiskError {
	return NewRiskError(ErrSystemOverload,
		fmt.Sprintf("system overloaded during %s", operation), operation).
		WithDetails("current_load", currentLoad).
		WithConstraint("max_load", maxLoad)
}
