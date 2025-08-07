package execution

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// TDD RED PHASE: Write failing tests first

func TestExecutionServiceInitialization(t *testing.T) {
	// This test will FAIL - service doesn't exist yet
	service := NewExecutionService(nil, nil)
	if service == nil {
		t.Fatal("Expected execution service to be created")
	}
}

func TestExecutionServiceStart(t *testing.T) {
	// This test will FAIL - service doesn't exist yet
	ctx := context.Background()
	service := NewExecutionService(nil, nil)
	
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution service: %v", err)
	}
	
	if !service.IsHealthy() {
		t.Error("Service should be healthy after start")
	}
	
	// Clean up
	defer service.Stop(ctx)
}

func TestExecutionServiceOrderSubmission(t *testing.T) {
	// This test will FAIL - service doesn't exist yet
	ctx := context.Background()
	
	// Mock execution engine will be needed
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}
	
	service := NewExecutionService(mockEngine, mockValidator)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	// Create test order
	asset := &domain.Asset{
		Symbol:    "AAPL",
		AssetType: domain.AssetTypeStock,
	}
	
	order := &domain.Order{
		ID:            "TEST_SERVICE_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SERVICE_TEST",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	// Submit order through service
	result, err := service.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected execution result")
	}
	
	if result.Status != "SUBMITTED" && result.Status != "FILLED" {
		t.Errorf("Expected order to be submitted or filled, got %s", result.Status)
	}
}

func TestExecutionServiceOrderValidation(t *testing.T) {
	// This test will FAIL - service doesn't exist yet
	ctx := context.Background()
	
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{
		shouldReject: true,
		rejectReason: "Invalid order size",
	}
	
	service := NewExecutionService(mockEngine, mockValidator)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	// Create invalid test order
	asset := &domain.Asset{
		Symbol:    "INVALID",
		AssetType: domain.AssetTypeStock,
	}
	
	order := &domain.Order{
		ID:            "INVALID_ORDER",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(-100.0), // Invalid negative quantity
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "INVALID_TEST",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	// Should fail validation
	_, err = service.SubmitOrder(ctx, order)
	if err == nil {
		t.Error("Expected error for invalid order")
	}
}

func TestExecutionServiceOrderTracking(t *testing.T) {
	// This test will FAIL - service doesn't exist yet
	ctx := context.Background()
	
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}
	
	service := NewExecutionService(mockEngine, mockValidator)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	// Submit order
	asset := &domain.Asset{
		Symbol:    "GOOGL",
		AssetType: domain.AssetTypeStock,
	}
	
	order := &domain.Order{
		ID:            "TRACK_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(50.0),
		Price:         types.NewDecimalFromFloat(2500.0),
		TimeInForce:   domain.TimeInForceGTC,
		ClientOrderID: "TRACK_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	_, err = service.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}
	
	// Track order status
	trackedOrder, err := service.GetOrderStatus(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get order status: %v", err)
	}
	
	if trackedOrder == nil {
		t.Fatal("Expected tracked order")
	}
	
	if trackedOrder.ID != order.ID {
		t.Errorf("Expected order ID %s, got %s", order.ID, trackedOrder.ID)
	}
}

func TestExecutionServiceOrderCancellation(t *testing.T) {
	// This test will FAIL - service doesn't exist yet
	ctx := context.Background()
	
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}
	
	service := NewExecutionService(mockEngine, mockValidator)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	// Submit order first
	asset := &domain.Asset{
		Symbol:    "MSFT",
		AssetType: domain.AssetTypeStock,
	}
	
	order := &domain.Order{
		ID:            "CANCEL_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideSell,
		Quantity:      types.NewDecimalFromFloat(75.0),
		Price:         types.NewDecimalFromFloat(300.0),
		TimeInForce:   domain.TimeInForceGTC,
		ClientOrderID: "CANCEL_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	_, err = service.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}
	
	// Cancel the order
	err = service.CancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to cancel order: %v", err)
	}
	
	// Verify order is cancelled
	cancelledOrder, err := service.GetOrderStatus(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get cancelled order status: %v", err)
	}
	
	if cancelledOrder.Status != domain.OrderStatusCancelled {
		t.Errorf("Expected order to be cancelled, got status %s", cancelledOrder.Status.String())
	}
}

func TestExecutionServiceMetrics(t *testing.T) {
	ctx := context.Background()
	
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}
	
	service := NewExecutionService(mockEngine, mockValidator)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	// Get initial metrics
	metrics := service.GetMetrics()
	if metrics.TotalOrdersProcessed != 0 {
		t.Errorf("Expected 0 initial orders, got %d", metrics.TotalOrdersProcessed)
	}
	
	// Submit an order to generate metrics
	asset := &domain.Asset{
		Symbol:    "TSLA",
		AssetType: domain.AssetTypeStock,
	}
	
	order := &domain.Order{
		ID:            "METRICS_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(25.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "METRICS_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	_, err = service.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}
	
	// Check updated metrics
	updatedMetrics := service.GetMetrics()
	if updatedMetrics.TotalOrdersProcessed == 0 {
		t.Error("Expected non-zero orders processed after submission")
	}
}

func TestExecutionServiceEnhancedFeatures(t *testing.T) {
	ctx := context.Background()
	
	// Test enhanced service with custom config
	config := ServiceConfig{
		MaxConcurrentOrders:   2,
		OrderTimeout:          5 * time.Second,
		EnableMetrics:         true,
		MetricsResetInterval:  time.Minute,
		EnableValidation:      true,
		MaxRetryAttempts:      1,
		RetryBackoffDuration:  50 * time.Millisecond,
	}
	
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}
	
	service := NewExecutionServiceWithConfig(mockEngine, mockValidator, config)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	// Test concurrent order limit
	asset := &domain.Asset{
		Symbol:    "TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	// Submit maximum allowed orders
	for i := 0; i < config.MaxConcurrentOrders; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("CONCURRENT_TEST_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeMarket,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(10.0),
			TimeInForce:   domain.TimeInForceIOC,
			ClientOrderID: fmt.Sprintf("CONCURRENT_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}
		
		_, err = service.SubmitOrder(ctx, order)
		if err != nil {
			t.Fatalf("Failed to submit order %d: %v", i, err)
		}
	}
	
	// Next order should be rejected due to limit
	limitOrder := &domain.Order{
		ID:            "LIMIT_EXCEEDED",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(10.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "LIMIT_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	_, err = service.SubmitOrder(ctx, limitOrder)
	if err == nil {
		t.Error("Expected error for exceeding concurrent order limit")
	}
	
	// Test active order tracking
	activeCount := service.GetActiveOrderCount()
	if activeCount != config.MaxConcurrentOrders {
		t.Errorf("Expected %d active orders, got %d", config.MaxConcurrentOrders, activeCount)
	}
	
	// Test service metrics
	serviceMetrics := service.GetServiceMetrics()
	if serviceMetrics.TotalOrdersSubmitted != uint64(config.MaxConcurrentOrders) {
		t.Errorf("Expected %d submitted orders, got %d", config.MaxConcurrentOrders, serviceMetrics.TotalOrdersSubmitted)
	}
	
	if serviceMetrics.TotalOrdersRejected != 1 {
		t.Errorf("Expected 1 rejected order, got %d", serviceMetrics.TotalOrdersRejected)
	}
	
	// Test metrics reset
	service.ResetMetrics()
	resetMetrics := service.GetServiceMetrics()
	if resetMetrics.TotalOrdersSubmitted != 0 {
		t.Errorf("Expected 0 orders after reset, got %d", resetMetrics.TotalOrdersSubmitted)
	}
	
	// Test config retrieval
	retrievedConfig := service.GetConfig()
	if retrievedConfig.MaxConcurrentOrders != config.MaxConcurrentOrders {
		t.Errorf("Config mismatch: expected %d, got %d", config.MaxConcurrentOrders, retrievedConfig.MaxConcurrentOrders)
	}
}

// Mock implementations for testing (will fail until implemented)

type MockExecutionEngine struct {
	healthy     bool
	submissions []string
}

func (m *MockExecutionEngine) Start(ctx context.Context) error {
	m.healthy = true
	return nil
}

func (m *MockExecutionEngine) Stop(ctx context.Context) error {
	m.healthy = false
	return nil
}

func (m *MockExecutionEngine) IsHealthy() bool {
	return m.healthy
}

func (m *MockExecutionEngine) SubmitOrder(ctx context.Context, order *domain.Order) (*ports.ExecutionResult, error) {
	m.submissions = append(m.submissions, order.ID)
	return &ports.ExecutionResult{
		OrderID:       order.ID,
		Status:        "SUBMITTED",
		TotalQuantity: order.Quantity,
		AveragePrice:  order.Price,
		ExecutedAt:    time.Now(),
	}, nil
}

func (m *MockExecutionEngine) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

func (m *MockExecutionEngine) ModifyOrder(ctx context.Context, orderID string, modification ports.OrderModification) error {
	return nil
}

func (m *MockExecutionEngine) GetOrderStatus(ctx context.Context, orderID string) (*domain.Order, error) {
	return &domain.Order{
		ID:     orderID,
		Status: domain.OrderStatusSubmitted,
	}, nil
}

func (m *MockExecutionEngine) GetExecutionHistory(ctx context.Context, orderID string) ([]ports.Fill, error) {
	return []ports.Fill{}, nil
}

func (m *MockExecutionEngine) GetMetrics() ports.ExecutionMetrics {
	return ports.ExecutionMetrics{
		TotalOrdersProcessed: uint64(len(m.submissions)),
		SuccessfulExecutions: uint64(len(m.submissions)),
	}
}

type MockOrderValidator struct {
	shouldReject bool
	rejectReason string
}

func (m *MockOrderValidator) ValidateOrder(ctx context.Context, order *domain.Order) error {
	if m.shouldReject {
		return fmt.Errorf("%s", m.rejectReason)
	}
	return nil
}

func (m *MockOrderValidator) ValidateRiskLimits(ctx context.Context, order *domain.Order, portfolio *domain.Portfolio) error {
	return nil
}

func (m *MockOrderValidator) ValidateMarketConditions(ctx context.Context, order *domain.Order, marketData ports.MarketData) error {
	return nil
}