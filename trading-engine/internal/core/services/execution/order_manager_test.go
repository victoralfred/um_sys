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

// TDD RED PHASE: Write failing tests for OrderManager first

func TestOrderManagerInitialization(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	manager := NewOrderManager()
	if manager == nil {
		t.Fatal("Expected order manager to be created")
	}
}

func TestOrderManagerSubmitOrder(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	ctx := context.Background()
	manager := NewOrderManager()

	asset := &domain.Asset{
		Symbol:    "AAPL",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "OM_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "OM_CLIENT_001",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Order should now be tracked
	retrievedOrder, err := manager.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve order: %v", err)
	}

	if retrievedOrder.ID != order.ID {
		t.Errorf("Expected order ID %s, got %s", order.ID, retrievedOrder.ID)
	}
}

func TestOrderManagerStateTransitions(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	ctx := context.Background()
	manager := NewOrderManager()

	asset := &domain.Asset{
		Symbol:    "GOOGL",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "STATE_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(50.0),
		Price:         types.NewDecimalFromFloat(2500.0),
		TimeInForce:   domain.TimeInForceGTC,
		ClientOrderID: "STATE_CLIENT_001",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	// Submit order
	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Test valid transition: PENDING -> SUBMITTED
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusSubmitted)
	if err != nil {
		t.Fatalf("Failed to update order status to SUBMITTED: %v", err)
	}

	// Test valid transition: SUBMITTED -> PARTIALLY_FILLED
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusPartiallyFilled)
	if err != nil {
		t.Fatalf("Failed to update order status to PARTIALLY_FILLED: %v", err)
	}

	// Test valid transition: PARTIALLY_FILLED -> FILLED
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusFilled)
	if err != nil {
		t.Fatalf("Failed to update order status to FILLED: %v", err)
	}

	// Verify final status
	finalOrder, err := manager.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get final order: %v", err)
	}

	if finalOrder.Status != domain.OrderStatusFilled {
		t.Errorf("Expected final status FILLED, got %s", finalOrder.Status.String())
	}
}

func TestOrderManagerInvalidStateTransitions(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	ctx := context.Background()
	manager := NewOrderManager()

	asset := &domain.Asset{
		Symbol:    "MSFT",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "INVALID_TRANSITION_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideSell,
		Quantity:      types.NewDecimalFromFloat(75.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "INVALID_CLIENT_001",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Test invalid transition: PENDING -> FILLED (should fail)
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusFilled)
	if err == nil {
		t.Error("Expected error for invalid transition from PENDING to FILLED")
	}

	// Test invalid transition: PENDING -> CANCELLED (valid)
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusCancelled)
	if err != nil {
		t.Fatalf("Failed to cancel order: %v", err)
	}

	// Test invalid transition: CANCELLED -> SUBMITTED (should fail)
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusSubmitted)
	if err == nil {
		t.Error("Expected error for invalid transition from CANCELLED to SUBMITTED")
	}
}

func TestOrderManagerProcessFill(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	ctx := context.Background()
	manager := NewOrderManager()

	asset := &domain.Asset{
		Symbol:    "TSLA",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "FILL_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		Price:         types.NewDecimalFromFloat(800.0),
		TimeInForce:   domain.TimeInForceGTC,
		ClientOrderID: "FILL_CLIENT_001",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Update to submitted first
	err = manager.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusSubmitted)
	if err != nil {
		t.Fatalf("Failed to update status to SUBMITTED: %v", err)
	}

	// Process partial fill
	partialFill := &ports.Fill{
		ID:        "FILL_001",
		OrderID:   order.ID,
		Price:     types.NewDecimalFromFloat(799.50),
		Quantity:  types.NewDecimalFromFloat(50.0),
		Timestamp: time.Now(),
		Venue:     "TEST_VENUE",
	}

	err = manager.ProcessFill(ctx, partialFill)
	if err != nil {
		t.Fatalf("Failed to process fill: %v", err)
	}

	// Order should be partially filled
	filledOrder, err := manager.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get filled order: %v", err)
	}

	if filledOrder.Status != domain.OrderStatusPartiallyFilled {
		t.Errorf("Expected status PARTIALLY_FILLED, got %s", filledOrder.Status.String())
	}

	// Process remaining fill
	remainingFill := &ports.Fill{
		ID:        "FILL_002",
		OrderID:   order.ID,
		Price:     types.NewDecimalFromFloat(800.25),
		Quantity:  types.NewDecimalFromFloat(50.0),
		Timestamp: time.Now(),
		Venue:     "TEST_VENUE",
	}

	err = manager.ProcessFill(ctx, remainingFill)
	if err != nil {
		t.Fatalf("Failed to process remaining fill: %v", err)
	}

	// Order should be fully filled
	completeOrder, err := manager.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get complete order: %v", err)
	}

	if completeOrder.Status != domain.OrderStatusFilled {
		t.Errorf("Expected status FILLED, got %s", completeOrder.Status.String())
	}
}

func TestOrderManagerProcessReject(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	ctx := context.Background()
	manager := NewOrderManager()

	asset := &domain.Asset{
		Symbol:    "AMZN",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "REJECT_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(25.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "REJECT_CLIENT_001",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Process rejection
	rejectReason := "Insufficient funds"
	err = manager.ProcessReject(ctx, order.ID, rejectReason)
	if err != nil {
		t.Fatalf("Failed to process reject: %v", err)
	}

	// Order should be rejected
	rejectedOrder, err := manager.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get rejected order: %v", err)
	}

	if rejectedOrder.Status != domain.OrderStatusRejected {
		t.Errorf("Expected status REJECTED, got %s", rejectedOrder.Status.String())
	}
}

func TestOrderManagerGetOrdersByStatus(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	ctx := context.Background()
	manager := NewOrderManager()

	// Submit multiple orders with different statuses
	asset := &domain.Asset{
		Symbol:    "NFLX",
		AssetType: domain.AssetTypeStock,
	}

	// Create pending orders
	for i := 0; i < 3; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("STATUS_TEST_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeLimit,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(10.0),
			Price:         types.NewDecimalFromFloat(500.0),
			TimeInForce:   domain.TimeInForceGTC,
			ClientOrderID: fmt.Sprintf("STATUS_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}

		err := manager.SubmitOrder(ctx, order)
		if err != nil {
			t.Fatalf("Failed to submit order %d: %v", i, err)
		}
	}

	// Update some to submitted
	err := manager.UpdateOrderStatus(ctx, "STATUS_TEST_0", domain.OrderStatusSubmitted)
	if err != nil {
		t.Fatalf("Failed to update order status: %v", err)
	}

	err = manager.UpdateOrderStatus(ctx, "STATUS_TEST_1", domain.OrderStatusSubmitted)
	if err != nil {
		t.Fatalf("Failed to update order status: %v", err)
	}

	// Get pending orders
	pendingOrders, err := manager.GetOrdersByStatus(ctx, domain.OrderStatusPending)
	if err != nil {
		t.Fatalf("Failed to get pending orders: %v", err)
	}

	if len(pendingOrders) != 1 {
		t.Errorf("Expected 1 pending order, got %d", len(pendingOrders))
	}

	// Get submitted orders
	submittedOrders, err := manager.GetOrdersByStatus(ctx, domain.OrderStatusSubmitted)
	if err != nil {
		t.Fatalf("Failed to get submitted orders: %v", err)
	}

	if len(submittedOrders) != 2 {
		t.Errorf("Expected 2 submitted orders, got %d", len(submittedOrders))
	}
}

func TestOrderManagerValidateTransition(t *testing.T) {
	// This test will FAIL - OrderManager doesn't exist yet
	ctx := context.Background()
	manager := NewOrderManager()

	asset := &domain.Asset{
		Symbol:    "NVDA",
		AssetType: domain.AssetTypeStock,
	}

	order := &domain.Order{
		ID:            "VALIDATE_TEST_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(15.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "VALIDATE_CLIENT_001",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	err := manager.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Test valid transitions
	validTransitions := []domain.OrderStatus{
		domain.OrderStatusSubmitted,
		domain.OrderStatusCancelled,
		domain.OrderStatusRejected,
	}

	for _, newStatus := range validTransitions {
		err = manager.ValidateOrderTransition(ctx, order.ID, newStatus)
		if err != nil {
			t.Errorf("Expected valid transition from PENDING to %s, but got error: %v", newStatus.String(), err)
		}
	}

	// Test invalid transitions
	invalidTransitions := []domain.OrderStatus{
		domain.OrderStatusFilled,
		domain.OrderStatusPartiallyFilled,
		domain.OrderStatusExpired,
	}

	for _, newStatus := range invalidTransitions {
		err = manager.ValidateOrderTransition(ctx, order.ID, newStatus)
		if err == nil {
			t.Errorf("Expected invalid transition from PENDING to %s, but no error was returned", newStatus.String())
		}
	}
}
