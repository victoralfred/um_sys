package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// OrderManagerConfig contains configuration for the order manager
type OrderManagerConfig struct {
	MaxOrders            int           `json:"max_orders"`
	EnableMetrics        bool          `json:"enable_metrics"`
	MetricsResetInterval time.Duration `json:"metrics_reset_interval"`
	EnableEventCallbacks bool          `json:"enable_event_callbacks"`
	OrderTimeoutDuration time.Duration `json:"order_timeout_duration"`
	EnableAutoCleanup    bool          `json:"enable_auto_cleanup"`
	CleanupInterval      time.Duration `json:"cleanup_interval"`
	MaxHistoryPerOrder   int           `json:"max_history_per_order"`
}

// DefaultOrderManagerConfig returns reasonable default configuration
func DefaultOrderManagerConfig() OrderManagerConfig {
	return OrderManagerConfig{
		MaxOrders:            10000,
		EnableMetrics:        true,
		MetricsResetInterval: time.Hour,
		EnableEventCallbacks: true,
		OrderTimeoutDuration: 24 * time.Hour,
		EnableAutoCleanup:    true,
		CleanupInterval:      time.Hour,
		MaxHistoryPerOrder:   100,
	}
}

// OrderManagerMetrics tracks order management metrics
type OrderManagerMetrics struct {
	TotalOrdersProcessed  uint64        `json:"total_orders_processed"`
	ActiveOrders          uint64        `json:"active_orders"`
	FilledOrders          uint64        `json:"filled_orders"`
	CancelledOrders       uint64        `json:"cancelled_orders"`
	RejectedOrders        uint64        `json:"rejected_orders"`
	ExpiredOrders         uint64        `json:"expired_orders"`
	TotalFillsProcessed   uint64        `json:"total_fills_processed"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	LastResetTime         time.Time     `json:"last_reset_time"`
}

// FillEventHandler handles fill events
type FillEventHandler interface {
	OnFillProcessed(ctx context.Context, orderID string, fill *ports.Fill, order *domain.Order) error
}

// StatusEventHandler handles status change events
type StatusEventHandler interface {
	OnStatusChanged(ctx context.Context, orderID string, oldStatus, newStatus domain.OrderStatus, order *domain.Order) error
}

// DefaultOrderManager implements OrderManager interface with state machine
// TDD REFACTOR phase - enhanced production-ready implementation
type DefaultOrderManager struct {
	mu             sync.RWMutex
	orders         map[string]*domain.Order               // Track all orders by ID
	fills          map[string][]ports.Fill                // Track fills by order ID
	ordersByStatus map[domain.OrderStatus][]*domain.Order // Index by status for fast queries

	// Configuration and metrics
	config  OrderManagerConfig
	metrics OrderManagerMetrics

	// Event callbacks
	fillHandlers   []FillEventHandler
	statusHandlers []StatusEventHandler

	// Lifecycle management
	ctx     context.Context
	cancel  context.CancelFunc
	stopped chan struct{}
}

// NewOrderManager creates a new order manager with default configuration
func NewOrderManager() *DefaultOrderManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &DefaultOrderManager{
		orders:         make(map[string]*domain.Order),
		fills:          make(map[string][]ports.Fill),
		ordersByStatus: make(map[domain.OrderStatus][]*domain.Order),
		config:         DefaultOrderManagerConfig(),
		metrics:        OrderManagerMetrics{LastResetTime: time.Now()},
		fillHandlers:   make([]FillEventHandler, 0),
		statusHandlers: make([]StatusEventHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
		stopped:        make(chan struct{}),
	}

	// Initialize status index
	manager.initializeStatusIndex()

	// Start background goroutines if enabled
	if manager.config.EnableAutoCleanup {
		go manager.cleanupRoutine()
	}

	return manager
}

// NewOrderManagerWithConfig creates an order manager with custom configuration
func NewOrderManagerWithConfig(config OrderManagerConfig) *DefaultOrderManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &DefaultOrderManager{
		orders:         make(map[string]*domain.Order),
		fills:          make(map[string][]ports.Fill),
		ordersByStatus: make(map[domain.OrderStatus][]*domain.Order),
		config:         config,
		metrics:        OrderManagerMetrics{LastResetTime: time.Now()},
		fillHandlers:   make([]FillEventHandler, 0),
		statusHandlers: make([]StatusEventHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
		stopped:        make(chan struct{}),
	}

	// Initialize status index
	manager.initializeStatusIndex()

	// Start background goroutines if enabled
	if manager.config.EnableAutoCleanup {
		go manager.cleanupRoutine()
	}

	return manager
}

// initializeStatusIndex sets up the status-based index
func (m *DefaultOrderManager) initializeStatusIndex() {
	statuses := []domain.OrderStatus{
		domain.OrderStatusPending,
		domain.OrderStatusSubmitted,
		domain.OrderStatusPartiallyFilled,
		domain.OrderStatusFilled,
		domain.OrderStatusCancelled,
		domain.OrderStatusRejected,
		domain.OrderStatusExpired,
	}

	for _, status := range statuses {
		m.ordersByStatus[status] = make([]*domain.Order, 0)
	}
}

// SubmitOrder submits a new order to the management system
func (m *DefaultOrderManager) SubmitOrder(ctx context.Context, order *domain.Order) error {
	startTime := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check order limit
	if len(m.orders) >= m.config.MaxOrders {
		return fmt.Errorf("order limit exceeded (%d)", m.config.MaxOrders)
	}

	// Check if order already exists
	if _, exists := m.orders[order.ID]; exists {
		return fmt.Errorf("order %s already exists", order.ID)
	}

	// Store the order
	orderCopy := *order
	m.orders[order.ID] = &orderCopy

	// Add to status index
	m.addToStatusIndex(&orderCopy)

	// Update metrics
	if m.config.EnableMetrics {
		m.metrics.TotalOrdersProcessed++
		m.updateActiveOrdersCount()
		m.updateProcessingTime(time.Since(startTime))
	}

	return nil
}

// ProcessFill processes a fill notification with enhanced tracking
func (m *DefaultOrderManager) ProcessFill(ctx context.Context, fill *ports.Fill) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the order
	order, exists := m.orders[fill.OrderID]
	if !exists {
		return fmt.Errorf("order %s not found", fill.OrderID)
	}

	oldStatus := order.Status

	// Add fill to history (with size limit)
	fills := m.fills[fill.OrderID]
	fills = append(fills, *fill)

	// Limit fill history size
	if len(fills) > m.config.MaxHistoryPerOrder {
		fills = fills[len(fills)-m.config.MaxHistoryPerOrder:]
	}
	m.fills[fill.OrderID] = fills

	// Calculate total filled quantity
	totalFilled := types.NewDecimalFromFloat(0.0)
	for _, f := range fills {
		totalFilled = totalFilled.Add(f.Quantity)
	}

	// Remove from old status index
	m.removeFromStatusIndex(order)

	// Update order status based on fill quantity
	if totalFilled.Cmp(order.Quantity) >= 0 {
		// Fully filled
		order.Status = domain.OrderStatusFilled
	} else {
		// Partially filled
		order.Status = domain.OrderStatusPartiallyFilled
	}

	order.UpdatedAt = time.Now()

	// Add to new status index
	m.addToStatusIndex(order)

	// Update metrics
	if m.config.EnableMetrics {
		m.metrics.TotalFillsProcessed++
		m.updateStatusMetrics(order.Status)
		m.updateActiveOrdersCount()
	}

	// Fire fill events
	if m.config.EnableEventCallbacks {
		go m.notifyFillHandlers(ctx, fill.OrderID, fill, order)

		// Also fire status change event if status changed
		if oldStatus != order.Status {
			go m.notifyStatusHandlers(ctx, fill.OrderID, oldStatus, order.Status, order)
		}
	}

	return nil
}

// ProcessReject processes an order rejection with enhanced tracking
func (m *DefaultOrderManager) ProcessReject(ctx context.Context, orderID string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the order
	order, exists := m.orders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	oldStatus := order.Status

	// Remove from old status index
	m.removeFromStatusIndex(order)

	// Update status to rejected
	order.Status = domain.OrderStatusRejected
	order.UpdatedAt = time.Now()

	// Add to new status index
	m.addToStatusIndex(order)

	// Update metrics
	if m.config.EnableMetrics {
		m.updateStatusMetrics(order.Status)
		m.updateActiveOrdersCount()
	}

	// Fire status change events
	if m.config.EnableEventCallbacks {
		go m.notifyStatusHandlers(ctx, orderID, oldStatus, order.Status, order)
	}

	return nil
}

// UpdateOrderStatus updates the status of an order with enhanced tracking
func (m *DefaultOrderManager) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the order
	order, exists := m.orders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	oldStatus := order.Status

	// Validate the transition
	if !m.isValidTransition(oldStatus, status) {
		return fmt.Errorf("invalid transition from %s to %s", oldStatus.String(), status.String())
	}

	// Remove from old status index
	m.removeFromStatusIndex(order)

	// Update status
	order.Status = status
	order.UpdatedAt = time.Now()

	// Add to new status index
	m.addToStatusIndex(order)

	// Update metrics
	if m.config.EnableMetrics {
		m.updateStatusMetrics(status)
		m.updateActiveOrdersCount()
	}

	// Fire status change events
	if m.config.EnableEventCallbacks {
		go m.notifyStatusHandlers(ctx, orderID, oldStatus, status, order)
	}

	return nil
}

// GetOrder retrieves an order by ID
func (m *DefaultOrderManager) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	order, exists := m.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	// Return a copy to prevent external mutation
	orderCopy := *order
	return &orderCopy, nil
}

// GetOrdersByStatus retrieves orders by status
func (m *DefaultOrderManager) GetOrdersByStatus(ctx context.Context, status domain.OrderStatus) ([]*domain.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matchingOrders []*domain.Order

	for _, order := range m.orders {
		if order.Status == status {
			// Add copy to prevent external mutation
			orderCopy := *order
			matchingOrders = append(matchingOrders, &orderCopy)
		}
	}

	return matchingOrders, nil
}

// ValidateOrderTransition validates a status transition
func (m *DefaultOrderManager) ValidateOrderTransition(ctx context.Context, orderID string, newStatus domain.OrderStatus) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find the order
	order, exists := m.orders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	if !m.isValidTransition(order.Status, newStatus) {
		return fmt.Errorf("invalid transition from %s to %s", order.Status.String(), newStatus.String())
	}

	return nil
}

// isValidTransition implements the order state machine logic
func (m *DefaultOrderManager) isValidTransition(current, next domain.OrderStatus) bool {
	// Define valid state transitions based on order lifecycle
	validTransitions := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPending: {
			domain.OrderStatusSubmitted,
			domain.OrderStatusCancelled,
			domain.OrderStatusRejected,
		},
		domain.OrderStatusSubmitted: {
			domain.OrderStatusPartiallyFilled,
			domain.OrderStatusFilled,
			domain.OrderStatusCancelled,
			domain.OrderStatusRejected,
			domain.OrderStatusExpired,
		},
		domain.OrderStatusPartiallyFilled: {
			domain.OrderStatusFilled,
			domain.OrderStatusCancelled,
			domain.OrderStatusExpired,
		},
		// Terminal states cannot transition
		domain.OrderStatusFilled:    {},
		domain.OrderStatusCancelled: {},
		domain.OrderStatusRejected:  {},
		domain.OrderStatusExpired:   {},
	}

	allowedTransitions, exists := validTransitions[current]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == next {
			return true
		}
	}

	return false
}

// Helper methods for enhanced functionality

// addToStatusIndex adds an order to the status-based index
func (m *DefaultOrderManager) addToStatusIndex(order *domain.Order) {
	m.ordersByStatus[order.Status] = append(m.ordersByStatus[order.Status], order)
}

// removeFromStatusIndex removes an order from the status-based index
func (m *DefaultOrderManager) removeFromStatusIndex(order *domain.Order) {
	orders := m.ordersByStatus[order.Status]
	for i, o := range orders {
		if o.ID == order.ID {
			// Remove by replacing with last element and truncating
			m.ordersByStatus[order.Status][i] = orders[len(orders)-1]
			m.ordersByStatus[order.Status] = orders[:len(orders)-1]
			break
		}
	}
}

// updateProcessingTime updates average processing time using exponential moving average
func (m *DefaultOrderManager) updateProcessingTime(duration time.Duration) {
	if m.metrics.TotalOrdersProcessed == 1 {
		m.metrics.AverageProcessingTime = duration
	} else {
		// Exponential moving average with alpha = 0.1
		alpha := 0.1
		m.metrics.AverageProcessingTime = time.Duration(
			float64(m.metrics.AverageProcessingTime)*(1-alpha) +
				float64(duration)*alpha,
		)
	}
}

// updateActiveOrdersCount updates the active orders metric
func (m *DefaultOrderManager) updateActiveOrdersCount() {
	activeCount := uint64(0)
	activeStatuses := []domain.OrderStatus{
		domain.OrderStatusPending,
		domain.OrderStatusSubmitted,
		domain.OrderStatusPartiallyFilled,
	}

	for _, status := range activeStatuses {
		activeCount += uint64(len(m.ordersByStatus[status]))
	}

	m.metrics.ActiveOrders = activeCount
}

// updateStatusMetrics updates status-specific metrics
func (m *DefaultOrderManager) updateStatusMetrics(status domain.OrderStatus) {
	switch status {
	case domain.OrderStatusFilled:
		m.metrics.FilledOrders++
	case domain.OrderStatusCancelled:
		m.metrics.CancelledOrders++
	case domain.OrderStatusRejected:
		m.metrics.RejectedOrders++
	case domain.OrderStatusExpired:
		m.metrics.ExpiredOrders++
	}
}

// notifyStatusHandlers notifies all registered status handlers
func (m *DefaultOrderManager) notifyStatusHandlers(ctx context.Context, orderID string, oldStatus, newStatus domain.OrderStatus, order *domain.Order) {
	for _, handler := range m.statusHandlers {
		if err := handler.OnStatusChanged(ctx, orderID, oldStatus, newStatus, order); err != nil {
			// Log error but don't fail the operation
			// In a real system, you'd use a proper logger here
			fmt.Printf("Status handler error for order %s: %v\n", orderID, err)
		}
	}
}

// notifyFillHandlers notifies all registered fill handlers
func (m *DefaultOrderManager) notifyFillHandlers(ctx context.Context, orderID string, fill *ports.Fill, order *domain.Order) {
	for _, handler := range m.fillHandlers {
		if err := handler.OnFillProcessed(ctx, orderID, fill, order); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Fill handler error for order %s: %v\n", orderID, err)
		}
	}
}

// cleanupRoutine runs periodic cleanup of old orders
func (m *DefaultOrderManager) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			close(m.stopped)
			return
		case <-ticker.C:
			m.performCleanup()
		}
	}
}

// performCleanup removes old terminal orders
func (m *DefaultOrderManager) performCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoffTime := time.Now().Add(-m.config.OrderTimeoutDuration)
	terminalStatuses := []domain.OrderStatus{
		domain.OrderStatusFilled,
		domain.OrderStatusCancelled,
		domain.OrderStatusRejected,
		domain.OrderStatusExpired,
	}

	for _, status := range terminalStatuses {
		orders := m.ordersByStatus[status]
		newOrders := make([]*domain.Order, 0, len(orders))

		for _, order := range orders {
			if order.UpdatedAt.After(cutoffTime) {
				newOrders = append(newOrders, order)
			} else {
				// Remove from main map and fills
				delete(m.orders, order.ID)
				delete(m.fills, order.ID)
			}
		}

		m.ordersByStatus[status] = newOrders
	}
}

// Event handler registration methods

// AddFillHandler adds a fill event handler
func (m *DefaultOrderManager) AddFillHandler(handler FillEventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fillHandlers = append(m.fillHandlers, handler)
}

// AddStatusHandler adds a status change event handler
func (m *DefaultOrderManager) AddStatusHandler(handler StatusEventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusHandlers = append(m.statusHandlers, handler)
}

// Metrics and configuration methods

// GetMetrics returns current order manager metrics
func (m *DefaultOrderManager) GetMetrics() OrderManagerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

// ResetMetrics resets all metrics
func (m *DefaultOrderManager) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = OrderManagerMetrics{LastResetTime: time.Now()}
}

// GetConfig returns current configuration
func (m *DefaultOrderManager) GetConfig() OrderManagerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Stop gracefully stops the order manager
func (m *DefaultOrderManager) Stop() error {
	m.cancel()

	// Wait for cleanup routine to finish
	if m.config.EnableAutoCleanup {
		<-m.stopped
	}

	return nil
}
