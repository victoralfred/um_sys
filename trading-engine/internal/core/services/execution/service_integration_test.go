package execution

import (
	"context"
	"testing"
	"time"

	"github.com/trading-engine/internal/adapters/execution"
	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// TDD Integration Tests - Testing with real CGO execution engine

func TestExecutionServiceWithCGOEngine(t *testing.T) {
	ctx := context.Background()

	// Create real CGO execution engine
	cgoEngine := execution.NewCGOExecutionEngine()

	// Ensure clean state by trying to stop any existing engine first
	cgoEngine.Stop(ctx) // Ignore error - might not be running

	err := cgoEngine.Initialize("{}")
	if err != nil {
		t.Fatalf("Failed to initialize CGO engine: %v", err)
	}

	err = cgoEngine.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start CGO engine: %v", err)
	}
	defer func() {
		if stopErr := cgoEngine.Stop(ctx); stopErr != nil {
			t.Logf("Warning: Failed to stop CGO engine: %v", stopErr)
		}
	}()

	// Create execution service with real engine
	service := NewExecutionService(cgoEngine, nil)
	err = service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution service: %v", err)
	}
	defer service.Stop(ctx)

	// Give market simulator time to initialize
	time.Sleep(200 * time.Millisecond)

	// Test real order execution through service
	asset := &domain.Asset{
		Symbol:    "AAPL",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "INTEGRATION_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(50.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "INTEGRATION_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	// Submit order through service (which will use CGO engine)
	result, err := service.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order through service: %v", err)
	}

	// Verify integration worked
	if result == nil {
		t.Fatal("Expected execution result from CGO engine")
	}

	if result.Status != "FILLED" && result.Status != "PARTIALLY_FILLED" {
		t.Errorf("Expected order to be filled by CGO engine, got status %s", result.Status)
	}

	if result.TotalQuantity.IsZero() {
		t.Error("Expected non-zero executed quantity from CGO engine")
	}

	if result.AveragePrice.IsZero() {
		t.Error("Expected non-zero average price from CGO engine")
	}

	// Test order tracking through service
	trackedOrder, err := service.GetOrderStatus(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get order status: %v", err)
	}

	if trackedOrder == nil {
		t.Fatal("Expected tracked order")
	}

	if trackedOrder.ID != order.ID {
		t.Errorf("Expected tracked order ID %s, got %s", order.ID, trackedOrder.ID)
	}

	// Test metrics from CGO engine through service
	metrics := service.GetMetrics()
	if metrics.TotalOrdersProcessed == 0 {
		t.Error("Expected non-zero orders processed from CGO engine")
	}

	if metrics.SuccessfulExecutions == 0 {
		t.Error("Expected non-zero successful executions from CGO engine")
	}
}

func TestExecutionServiceValidationIntegration(t *testing.T) {
	ctx := context.Background()

	// Create CGO engine
	cgoEngine := execution.NewCGOExecutionEngine()

	// Ensure clean state by trying to stop any existing engine first
	cgoEngine.Stop(ctx) // Ignore error - might not be running

	err := cgoEngine.Initialize("{}")
	if err != nil {
		t.Fatalf("Failed to initialize CGO engine: %v", err)
	}

	err = cgoEngine.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start CGO engine: %v", err)
	}
	defer func() {
		if stopErr := cgoEngine.Stop(ctx); stopErr != nil {
			t.Logf("Warning: Failed to stop CGO engine: %v", stopErr)
		}
	}()

	// Create validation that rejects invalid orders
	validator := &MockOrderValidator{
		shouldReject: true,
		rejectReason: "Order size exceeds daily limit",
	}

	// Create service with both real engine and validator
	service := NewExecutionService(cgoEngine, validator)
	err = service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution service: %v", err)
	}
	defer service.Stop(ctx)

	// Create order that will be rejected by validator
	asset := &domain.Asset{
		Symbol:    "GOOGL",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "VALIDATION_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(10000.0), // Large order
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "VALIDATION_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	// Order should be rejected by validator before reaching CGO engine
	_, err = service.SubmitOrder(ctx, order)
	if err == nil {
		t.Error("Expected validation error for large order")
	}

	// Verify error message contains validation reason
	expectedError := "order validation failed"
	if len(err.Error()) < len(expectedError) || err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected validation error, got: %v", err)
	}

	// Verify order was not submitted to CGO engine
	serviceMetrics := service.GetServiceMetrics()
	if serviceMetrics.ValidationFailures == 0 {
		t.Error("Expected validation failure to be recorded")
	}

	if serviceMetrics.TotalOrdersRejected == 0 {
		t.Error("Expected rejected order to be recorded")
	}
}
