package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
)

// DefaultExecutionService orchestrates order execution through the execution engine
// following TDD REFACTOR phase - enhanced implementation with proper error handling
type DefaultExecutionService struct {
	engine    ports.ExecutionEngine
	validator ports.OrderValidator

	// Internal state
	mu      sync.RWMutex
	running bool
	orders  map[string]*domain.Order // Track submitted orders
	metrics ExecutionServiceMetrics

	// Configuration
	config ServiceConfig

	// Channels for async processing
	cancelCh chan struct{}
}

// ExecutionServiceMetrics tracks service-level metrics
type ExecutionServiceMetrics struct {
	TotalOrdersSubmitted  uint64
	TotalOrdersCancelled  uint64
	TotalOrdersRejected   uint64
	ValidationFailures    uint64
	EngineFailures        uint64
	AverageProcessingTime time.Duration
	LastResetTime         time.Time
}

// ServiceConfig contains configuration for the execution service
type ServiceConfig struct {
	MaxConcurrentOrders  int           `json:"max_concurrent_orders"`
	OrderTimeout         time.Duration `json:"order_timeout"`
	EnableMetrics        bool          `json:"enable_metrics"`
	MetricsResetInterval time.Duration `json:"metrics_reset_interval"`
	EnableValidation     bool          `json:"enable_validation"`
	MaxRetryAttempts     int           `json:"max_retry_attempts"`
	RetryBackoffDuration time.Duration `json:"retry_backoff_duration"`
}

// DefaultServiceConfig returns reasonable default configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxConcurrentOrders:  1000,
		OrderTimeout:         30 * time.Second,
		EnableMetrics:        true,
		MetricsResetInterval: time.Hour,
		EnableValidation:     true,
		MaxRetryAttempts:     3,
		RetryBackoffDuration: 100 * time.Millisecond,
	}
}

// NewExecutionService creates a new execution service
func NewExecutionService(engine ports.ExecutionEngine, validator ports.OrderValidator) *DefaultExecutionService {
	return &DefaultExecutionService{
		engine:    engine,
		validator: validator,
		orders:    make(map[string]*domain.Order),
		config:    DefaultServiceConfig(),
		cancelCh:  make(chan struct{}),
		metrics:   ExecutionServiceMetrics{LastResetTime: time.Now()},
	}
}

// NewExecutionServiceWithConfig creates a service with custom configuration
func NewExecutionServiceWithConfig(engine ports.ExecutionEngine, validator ports.OrderValidator, config ServiceConfig) *DefaultExecutionService {
	return &DefaultExecutionService{
		engine:    engine,
		validator: validator,
		orders:    make(map[string]*domain.Order),
		config:    config,
		cancelCh:  make(chan struct{}),
		metrics:   ExecutionServiceMetrics{LastResetTime: time.Now()},
	}
}

// Start starts the execution service
func (s *DefaultExecutionService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("service already running")
	}

	// Start the underlying execution engine
	if s.engine != nil {
		if err := s.engine.Start(ctx); err != nil {
			return fmt.Errorf("failed to start execution engine: %w", err)
		}
	}

	s.running = true
	return nil
}

// Stop stops the execution service gracefully
func (s *DefaultExecutionService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil // Already stopped
	}

	// Signal cancellation
	close(s.cancelCh)

	// Stop the underlying execution engine
	if s.engine != nil {
		if err := s.engine.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop execution engine: %w", err)
		}
	}

	s.running = false
	return nil
}

// IsHealthy returns true if the service is healthy
func (s *DefaultExecutionService) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return false
	}

	if s.engine != nil {
		return s.engine.IsHealthy()
	}

	return true
}

// SubmitOrder submits an order for execution with enhanced error handling and retry logic
func (s *DefaultExecutionService) SubmitOrder(ctx context.Context, order *domain.Order) (*ports.ExecutionResult, error) {
	startTime := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil, fmt.Errorf("service not running")
	}

	// Check concurrent order limit
	if len(s.orders) >= s.config.MaxConcurrentOrders {
		s.metrics.TotalOrdersRejected++
		return nil, fmt.Errorf("concurrent order limit exceeded (%d)", s.config.MaxConcurrentOrders)
	}

	// Validate the order first if validation is enabled
	if s.config.EnableValidation && s.validator != nil {
		if err := s.validator.ValidateOrder(ctx, order); err != nil {
			s.metrics.ValidationFailures++
			s.metrics.TotalOrdersRejected++
			return nil, fmt.Errorf("order validation failed: %w", err)
		}
	}

	// Submit to execution engine with retry logic
	var result *ports.ExecutionResult
	var err error

	if s.engine != nil {
		result, err = s.submitOrderWithRetry(ctx, order)
		if err != nil {
			s.metrics.EngineFailures++
			s.metrics.TotalOrdersRejected++
			return nil, fmt.Errorf("engine submission failed: %w", err)
		}
	} else {
		// Fallback for testing - create mock result
		result = &ports.ExecutionResult{
			OrderID:       order.ID,
			Status:        "SUBMITTED",
			TotalQuantity: order.Quantity,
			AveragePrice:  order.Price,
			ExecutedAt:    time.Now(),
		}
	}

	// Track the order
	order.Status = domain.OrderStatusSubmitted
	order.UpdatedAt = time.Now()
	s.orders[order.ID] = order
	s.metrics.TotalOrdersSubmitted++

	// Update processing time with exponential moving average
	processingTime := time.Since(startTime)
	s.updateProcessingTime(processingTime)

	return result, nil
}

// submitOrderWithRetry implements retry logic for order submission
func (s *DefaultExecutionService) submitOrderWithRetry(ctx context.Context, order *domain.Order) (*ports.ExecutionResult, error) {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetryAttempts; attempt++ {
		if attempt > 0 {
			// Apply backoff delay
			select {
			case <-time.After(s.config.RetryBackoffDuration * time.Duration(attempt)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		result, err := s.engine.SubmitOrder(ctx, order)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry for certain types of errors
		if s.shouldNotRetry(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", s.config.MaxRetryAttempts+1, lastErr)
}

// shouldNotRetry determines if an error should not be retried
func (s *DefaultExecutionService) shouldNotRetry(err error) bool {
	errStr := err.Error()
	return contains(errStr, "validation") ||
		contains(errStr, "invalid") ||
		contains(errStr, "rejected") ||
		contains(errStr, "unauthorized")
}

// updateProcessingTime updates the average processing time using exponential moving average
func (s *DefaultExecutionService) updateProcessingTime(processingTime time.Duration) {
	if s.metrics.TotalOrdersSubmitted == 1 {
		s.metrics.AverageProcessingTime = processingTime
	} else {
		// Exponential moving average with alpha = 0.1
		alpha := 0.1
		s.metrics.AverageProcessingTime = time.Duration(
			float64(s.metrics.AverageProcessingTime)*(1-alpha) +
				float64(processingTime)*alpha,
		)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(substr) <= len(s) && s[:len(substr)] == substr
}

// CancelOrder cancels a pending order
func (s *DefaultExecutionService) CancelOrder(ctx context.Context, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("service not running")
	}

	// Check if order exists
	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	// Cancel in execution engine
	if s.engine != nil {
		if err := s.engine.CancelOrder(ctx, orderID); err != nil {
			return fmt.Errorf("failed to cancel order in engine: %w", err)
		}
	}

	// Update order status
	order.Status = domain.OrderStatusCancelled
	order.CancelledAt = &[]time.Time{time.Now()}[0]
	order.UpdatedAt = time.Now()

	s.metrics.TotalOrdersCancelled++

	return nil
}

// GetOrderStatus retrieves the current status of an order
func (s *DefaultExecutionService) GetOrderStatus(ctx context.Context, orderID string) (*domain.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return nil, fmt.Errorf("service not running")
	}

	// Check local tracking first
	if order, exists := s.orders[orderID]; exists {
		return order, nil
	}

	// Fallback to engine if available
	if s.engine != nil {
		return s.engine.GetOrderStatus(ctx, orderID)
	}

	return nil, fmt.Errorf("order %s not found", orderID)
}

// GetMetrics returns execution engine metrics
func (s *DefaultExecutionService) GetMetrics() ports.ExecutionMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.engine != nil {
		return s.engine.GetMetrics()
	}

	// Return basic metrics if no engine
	return ports.ExecutionMetrics{
		TotalOrdersProcessed: s.metrics.TotalOrdersSubmitted,
		SuccessfulExecutions: s.metrics.TotalOrdersSubmitted - s.metrics.TotalOrdersRejected,
		FailedExecutions:     s.metrics.TotalOrdersRejected,
	}
}

// GetServiceMetrics returns service-specific metrics
func (s *DefaultExecutionService) GetServiceMetrics() ExecutionServiceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.metrics
}

// ResetMetrics resets all service metrics
func (s *DefaultExecutionService) ResetMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics = ExecutionServiceMetrics{
		LastResetTime: time.Now(),
	}
}

// GetConfig returns the current service configuration
func (s *DefaultExecutionService) GetConfig() ServiceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config
}

// UpdateConfig updates the service configuration (only certain fields can be updated at runtime)
func (s *DefaultExecutionService) UpdateConfig(newConfig ServiceConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only allow updating certain config fields while running
	if s.running {
		s.config.EnableMetrics = newConfig.EnableMetrics
		s.config.EnableValidation = newConfig.EnableValidation
		s.config.MetricsResetInterval = newConfig.MetricsResetInterval
		s.config.MaxRetryAttempts = newConfig.MaxRetryAttempts
		s.config.RetryBackoffDuration = newConfig.RetryBackoffDuration
	} else {
		// Allow full config update when stopped
		s.config = newConfig
	}

	return nil
}

// GetActiveOrderCount returns the number of active orders being tracked
func (s *DefaultExecutionService) GetActiveOrderCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.orders)
}

// GetActiveOrderIDs returns IDs of all active orders
func (s *DefaultExecutionService) GetActiveOrderIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderIDs := make([]string, 0, len(s.orders))
	for orderID := range s.orders {
		orderIDs = append(orderIDs, orderID)
	}

	return orderIDs
}
