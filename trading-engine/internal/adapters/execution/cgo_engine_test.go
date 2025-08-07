package execution

import (
	"context"
	"testing"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// TDD: Step 1 - RED - Write failing tests for CGO execution engine

func TestCGOEngineBasicFlow(t *testing.T) {
	ctx := context.Background()
	engine := NewCGOExecutionEngine()
	defer engine.Stop(ctx)

	// Test initialization
	err := engine.Initialize("{}")
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	// Test start
	err = engine.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	// Test health check
	if !engine.IsHealthy() {
		t.Error("Engine should be healthy after start")
	}

	// Give market simulator time to initialize
	time.Sleep(200 * time.Millisecond)

	// Create test order
	asset := &domain.Asset{
		Symbol:    "AAPL",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "TEST_CGO_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		Price:         types.Zero(), // Market order
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "TEST_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	// Execute order
	result, err := engine.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to execute order: %v", err)
	}

	// Verify execution result
	if result == nil {
		t.Fatal("Expected execution result")
	}

	if result.OrderID != order.ID {
		t.Errorf("Expected order ID %s, got %s", order.ID, result.OrderID)
	}

	if result.Status != "FILLED" && result.Status != "PARTIALLY_FILLED" {
		t.Errorf("Expected order to be filled, got status %s", result.Status)
	}

	if result.TotalQuantity.IsZero() {
		t.Error("Expected non-zero executed quantity")
	}

	if result.AveragePrice.IsZero() {
		t.Error("Expected non-zero average price")
	}

	if len(result.Fills) != 1 {
		t.Errorf("Expected 1 fill, got %d", len(result.Fills))
	}

	// Verify fill details
	fill := result.Fills[0]
	if fill.OrderID != order.ID {
		t.Errorf("Expected fill order ID %s, got %s", order.ID, fill.OrderID)
	}

	if fill.Price.IsZero() {
		t.Error("Expected non-zero fill price")
	}

	if fill.Quantity.IsZero() {
		t.Error("Expected non-zero fill quantity")
	}

	// Test market data retrieval
	marketData, err := engine.GetOrderBook("AAPL")
	if err != nil {
		t.Fatalf("Failed to get market data: %v", err)
	}

	if marketData == nil {
		t.Fatal("Expected market data")
	}

	if marketData.BidPrice.IsZero() {
		t.Error("Expected non-zero bid price")
	}

	if marketData.AskPrice.IsZero() {
		t.Error("Expected non-zero ask price")
	}

	// Ask price should be higher than bid price
	if marketData.AskPrice.Cmp(marketData.BidPrice) <= 0 {
		t.Errorf("Ask price (%s) should be higher than bid price (%s)",
			marketData.AskPrice.String(), marketData.BidPrice.String())
	}

	// Test metrics
	metrics := engine.GetMetrics()
	if metrics.TotalOrdersProcessed == 0 {
		t.Error("Expected non-zero total orders processed")
	}

	if metrics.SuccessfulExecutions == 0 {
		t.Error("Expected non-zero successful executions")
	}

	// Test cancellation (should fail for non-existent order)
	err = engine.CancelOrder(ctx, "NON_EXISTENT")
	if err == nil {
		t.Error("Expected error when cancelling non-existent order")
	}

	// Test stop
	err = engine.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop engine: %v", err)
	}

	// Engine should not be healthy after stop
	if engine.IsHealthy() {
		t.Error("Engine should not be healthy after stop")
	}
}

func TestCGOEngineErrorHandling(t *testing.T) {
	ctx := context.Background()
	engine := NewCGOExecutionEngine()

	// Test operations on uninitialized engine
	err := engine.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting uninitialized engine")
	}

	if engine.IsHealthy() {
		t.Error("Uninitialized engine should not be healthy")
	}

	// Initialize but don't start
	err = engine.Initialize("{}")
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	// Test operations on initialized but not running engine
	if engine.IsHealthy() {
		t.Error("Non-running engine should not be healthy")
	}

	asset := &domain.Asset{
		Symbol:    "AAPL",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "TEST_ERROR_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "TEST_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	_, err = engine.SubmitOrder(ctx, order)
	if err == nil {
		t.Error("Expected error when executing order on non-running engine")
	}

	_, err = engine.GetOrderBook("AAPL")
	if err == nil {
		t.Error("Expected error when getting order book on non-running engine")
	}

	err = engine.CancelOrder(ctx, "TEST_ORDER")
	if err == nil {
		t.Error("Expected error when cancelling order on non-running engine")
	}

	// Clean up
	defer engine.Stop(ctx)
}