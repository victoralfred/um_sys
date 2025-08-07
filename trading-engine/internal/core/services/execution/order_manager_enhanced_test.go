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

// TDD REFACTOR phase tests for enhanced OrderManager functionality

func TestOrderManagerWithCustomConfig(t *testing.T) {
	config := OrderManagerConfig{
		MaxOrders:            5,
		EnableMetrics:        true,
		MetricsResetInterval: time.Minute,
		EnableEventCallbacks: false, // Disable for this test
		OrderTimeoutDuration: time.Hour,
		EnableAutoCleanup:    false, // Disable for this test
		CleanupInterval:      time.Hour,
		MaxHistoryPerOrder:   10,
	}

	manager := NewOrderManagerWithConfig(config)
	defer manager.Stop()

	// Test max orders limit
	asset := &domain.Asset{
		Symbol:    "TEST",
		AssetType: domain.AssetTypeStock,
	}

	// Submit maximum allowed orders
	for i := 0; i < config.MaxOrders; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("MAX_TEST_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeMarket,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(10.0),
			TimeInForce:   domain.TimeInForceIOC,
			ClientOrderID: fmt.Sprintf("MAX_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}

		err := manager.SubmitOrder(context.Background(), order)
		if err != nil {
			t.Fatalf("Failed to submit order %d: %v", i, err)
		}
	}

	// Next order should be rejected
	extraOrder := &domain.Order{
		ID:            "EXTRA_ORDER",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(10.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "EXTRA_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	err := manager.SubmitOrder(context.Background(), extraOrder)
	if err == nil {
		t.Error("Expected error for exceeding max orders limit")
	}

	// Test metrics
	metrics := manager.GetMetrics()
	if metrics.TotalOrdersProcessed != uint64(config.MaxOrders) {
		t.Errorf("Expected %d processed orders, got %d", config.MaxOrders, metrics.TotalOrdersProcessed)
	}

	if metrics.ActiveOrders != uint64(config.MaxOrders) {
		t.Errorf("Expected %d active orders, got %d", config.MaxOrders, metrics.ActiveOrders)
	}
}

func TestOrderManagerEventHandlers(t *testing.T) {
	manager := NewOrderManager()
	defer manager.Stop()

	// Create test handlers
	fillHandler := &TestFillHandler{calls: make([]string, 0)}
	statusHandler := &TestStatusHandler{calls: make([]string, 0)}

	manager.AddFillHandler(fillHandler)
	manager.AddStatusHandler(statusHandler)

	ctx := context.Background()
	asset := &domain.Asset{
		Symbol:    "EVENT_TEST",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "EVENT_ORDER_001",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		Price:         types.NewDecimalFromFloat(50.0),
		TimeInForce:   domain.TimeInForceGTC,
		ClientOrderID: "EVENT_CLIENT_001",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	// Submit order
	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Update status to trigger status handler
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusSubmitted)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Wait for async handler calls
	time.Sleep(50 * time.Millisecond)

	// Process fill to trigger fill handler
	fill := &ports.Fill{
		ID:        "EVENT_FILL_001",
		OrderID:   order.ID,
		Price:     types.NewDecimalFromFloat(50.25),
		Quantity:  types.NewDecimalFromFloat(50.0),
		Timestamp: time.Now(),
		Venue:     "EVENT_VENUE",
	}

	err = manager.ProcessFill(ctx, fill)
	if err != nil {
		t.Fatalf("Failed to process fill: %v", err)
	}

	// Wait for async handler calls
	time.Sleep(50 * time.Millisecond)

	// Verify handler calls
	if len(statusHandler.calls) == 0 {
		t.Error("Expected status handler to be called")
	}

	if len(fillHandler.calls) == 0 {
		t.Error("Expected fill handler to be called")
	}
}

func TestOrderManagerMetricsTracking(t *testing.T) {
	manager := NewOrderManager()
	defer manager.Stop()

	ctx := context.Background()
	asset := &domain.Asset{
		Symbol:    "METRICS_TEST",
		AssetType: domain.AssetTypeStock,
	}

	// Submit multiple orders and track metrics
	for i := 0; i < 5; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("METRICS_ORDER_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeMarket,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(10.0),
			TimeInForce:   domain.TimeInForceIOC,
			ClientOrderID: fmt.Sprintf("METRICS_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}

		err := manager.SubmitOrder(ctx, order)
		if err != nil {
			t.Fatalf("Failed to submit order %d: %v", i, err)
		}
	}

	// Fill 2 orders
	for i := 0; i < 2; i++ {
		orderID := fmt.Sprintf("METRICS_ORDER_%d", i)
		err := manager.UpdateOrderStatus(ctx, orderID, domain.OrderStatusSubmitted)
		if err != nil {
			t.Fatalf("Failed to update order status: %v", err)
		}

		err = manager.UpdateOrderStatus(ctx, orderID, domain.OrderStatusFilled)
		if err != nil {
			t.Fatalf("Failed to fill order: %v", err)
		}
	}

	// Cancel 1 order
	err := manager.UpdateOrderStatus(ctx, "METRICS_ORDER_2", domain.OrderStatusCancelled)
	if err != nil {
		t.Fatalf("Failed to cancel order: %v", err)
	}

	// Reject 1 order
	err = manager.ProcessReject(ctx, "METRICS_ORDER_3", "Test rejection")
	if err != nil {
		t.Fatalf("Failed to reject order: %v", err)
	}

	// Check metrics
	metrics := manager.GetMetrics()

	if metrics.TotalOrdersProcessed != 5 {
		t.Errorf("Expected 5 processed orders, got %d", metrics.TotalOrdersProcessed)
	}

	if metrics.FilledOrders != 2 {
		t.Errorf("Expected 2 filled orders, got %d", metrics.FilledOrders)
	}

	if metrics.CancelledOrders != 1 {
		t.Errorf("Expected 1 cancelled order, got %d", metrics.CancelledOrders)
	}

	if metrics.RejectedOrders != 1 {
		t.Errorf("Expected 1 rejected order, got %d", metrics.RejectedOrders)
	}

	if metrics.ActiveOrders != 1 { // Only METRICS_ORDER_4 should be active
		t.Errorf("Expected 1 active order, got %d", metrics.ActiveOrders)
	}

	// Test metrics reset
	manager.ResetMetrics()
	resetMetrics := manager.GetMetrics()

	if resetMetrics.TotalOrdersProcessed != 0 {
		t.Errorf("Expected 0 processed orders after reset, got %d", resetMetrics.TotalOrdersProcessed)
	}
}

func TestOrderManagerDuplicateOrder(t *testing.T) {
	manager := NewOrderManager()
	defer manager.Stop()

	ctx := context.Background()
	asset := &domain.Asset{
		Symbol:    "DUP_TEST",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "DUPLICATE_ORDER",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(10.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "DUP_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	// Submit order first time - should succeed
	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order first time: %v", err)
	}

	// Submit same order again - should fail
	err = manager.SubmitOrder(ctx, order)
	if err == nil {
		t.Error("Expected error for duplicate order")
	}
}

func TestOrderManagerFillHistoryLimit(t *testing.T) {
	config := OrderManagerConfig{
		MaxOrders:            100,
		EnableMetrics:        true,
		MetricsResetInterval: time.Hour,
		EnableEventCallbacks: false,
		OrderTimeoutDuration: time.Hour,
		EnableAutoCleanup:    false,
		CleanupInterval:      time.Hour,
		MaxHistoryPerOrder:   3, // Limit to 3 fills
	}

	manager := NewOrderManagerWithConfig(config)
	defer manager.Stop()

	ctx := context.Background()
	asset := &domain.Asset{
		Symbol:    "FILL_LIMIT_TEST",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "FILL_LIMIT_ORDER",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		Price:         types.NewDecimalFromFloat(50.0),
		TimeInForce:   domain.TimeInForceGTC,
		ClientOrderID: "FILL_LIMIT_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusSubmitted)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Process 5 fills (more than the limit of 3)
	for i := 0; i < 5; i++ {
		fill := &ports.Fill{
			ID:        fmt.Sprintf("FILL_LIMIT_%d", i),
			OrderID:   order.ID,
			Price:     types.NewDecimalFromFloat(50.0 + float64(i)*0.1),
			Quantity:  types.NewDecimalFromFloat(10.0),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Venue:     "FILL_VENUE",
		}

		err = manager.ProcessFill(ctx, fill)
		if err != nil {
			t.Fatalf("Failed to process fill %d: %v", i, err)
		}
	}

	// Check that only the last 3 fills are retained
	manager.mu.RLock()
	fills := manager.fills[order.ID]
	manager.mu.RUnlock()

	if len(fills) != 3 {
		t.Errorf("Expected 3 fills in history, got %d", len(fills))
	}

	// Verify the fills are the most recent ones (2, 3, 4)
	expectedIDs := []string{"FILL_LIMIT_2", "FILL_LIMIT_3", "FILL_LIMIT_4"}
	for i, fill := range fills {
		if fill.ID != expectedIDs[i] {
			t.Errorf("Expected fill ID %s at index %d, got %s", expectedIDs[i], i, fill.ID)
		}
	}
}

// Test helper structs

type TestFillHandler struct {
	calls []string
}

func (h *TestFillHandler) OnFillProcessed(ctx context.Context, orderID string, fill *ports.Fill, order *domain.Order) error {
	h.calls = append(h.calls, fmt.Sprintf("fill_%s_%s", orderID, fill.ID))
	return nil
}

type TestStatusHandler struct {
	calls []string
}

func (h *TestStatusHandler) OnStatusChanged(ctx context.Context, orderID string, oldStatus, newStatus domain.OrderStatus, order *domain.Order) error {
	h.calls = append(h.calls, fmt.Sprintf("status_%s_%s_%s", orderID, oldStatus.String(), newStatus.String()))
	return nil
}
