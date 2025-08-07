package domain

import (
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

func createTestAsset() *Asset {
	asset, _ := NewAssetBuilder().
		Symbol("AAPL").
		Name("Apple Inc.").
		Type(AssetTypeStock).
		Exchange("NASDAQ").
		Currency("USD").
		Precision(2).
		MinQuantity(types.NewDecimalFromInt(1)).
		MaxQuantity(types.NewDecimalFromInt(10000)).
		TickSize(types.NewDecimalFromFloat(0.01)).
		Build()
	return asset
}

func TestOrderBuilder(t *testing.T) {
	asset := createTestAsset()

	tests := []struct {
		name    string
		builder func() *OrderBuilder
		wantErr bool
	}{
		{
			name: "valid market buy order",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-001").
					Asset(asset).
					Type(OrderTypeMarket).
					Side(OrderSideBuy).
					Quantity(types.NewDecimalFromInt(100))
			},
			wantErr: false,
		},
		{
			name: "valid limit sell order",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-002").
					Asset(asset).
					Type(OrderTypeLimit).
					Side(OrderSideSell).
					Quantity(types.NewDecimalFromInt(50)).
					Price(types.NewDecimalFromFloat(150.25))
			},
			wantErr: false,
		},
		{
			name: "valid stop loss order",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-003").
					Asset(asset).
					Type(OrderTypeStop).
					Side(OrderSideSell).
					Quantity(types.NewDecimalFromInt(75)).
					StopPrice(types.NewDecimalFromFloat(140.00))
			},
			wantErr: false,
		},
		{
			name: "empty order ID",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("").
					Asset(asset).
					Type(OrderTypeMarket).
					Side(OrderSideBuy).
					Quantity(types.NewDecimalFromInt(100))
			},
			wantErr: true,
		},
		{
			name: "nil asset",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-004").
					Asset(nil).
					Type(OrderTypeMarket).
					Side(OrderSideBuy).
					Quantity(types.NewDecimalFromInt(100))
			},
			wantErr: true,
		},
		{
			name: "zero quantity",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-005").
					Asset(asset).
					Type(OrderTypeMarket).
					Side(OrderSideBuy).
					Quantity(types.Zero())
			},
			wantErr: true,
		},
		{
			name: "limit order without price",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-006").
					Asset(asset).
					Type(OrderTypeLimit).
					Side(OrderSideBuy).
					Quantity(types.NewDecimalFromInt(100))
			},
			wantErr: true,
		},
		{
			name: "stop order without stop price",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-007").
					Asset(asset).
					Type(OrderTypeStop).
					Side(OrderSideSell).
					Quantity(types.NewDecimalFromInt(100))
			},
			wantErr: true,
		},
		{
			name: "trailing stop without trailing amount",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-008").
					Asset(asset).
					Type(OrderTypeTrailingStop).
					Side(OrderSideSell).
					Quantity(types.NewDecimalFromInt(100))
			},
			wantErr: true,
		},
		{
			name: "GTD order with expiration",
			builder: func() *OrderBuilder {
				expiresAt := time.Now().Add(24 * time.Hour)
				return NewOrderBuilder().
					ID("ORDER-009").
					Asset(asset).
					Type(OrderTypeLimit).
					Side(OrderSideBuy).
					Quantity(types.NewDecimalFromInt(100)).
					Price(types.NewDecimalFromFloat(150.00)).
					TimeInForce(TimeInForceGTD).
					ExpiresAt(expiresAt)
			},
			wantErr: false,
		},
		{
			name: "GTD order without expiration",
			builder: func() *OrderBuilder {
				return NewOrderBuilder().
					ID("ORDER-010").
					Asset(asset).
					Type(OrderTypeLimit).
					Side(OrderSideBuy).
					Quantity(types.NewDecimalFromInt(100)).
					Price(types.NewDecimalFromFloat(150.00)).
					TimeInForce(TimeInForceGTD)
			},
			wantErr: true,
		},
		{
			name: "non-GTD order with expiration",
			builder: func() *OrderBuilder {
				expiresAt := time.Now().Add(24 * time.Hour)
				return NewOrderBuilder().
					ID("ORDER-011").
					Asset(asset).
					Type(OrderTypeLimit).
					Side(OrderSideBuy).
					Quantity(types.NewDecimalFromInt(100)).
					Price(types.NewDecimalFromFloat(150.00)).
					TimeInForce(TimeInForceGTC).
					ExpiresAt(expiresAt)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := tt.builder().Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderBuilder.Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && order == nil {
				t.Errorf("OrderBuilder.Build() returned nil order")
			}
		})
	}
}

func TestOrderStateTransitions(t *testing.T) {
	asset := createTestAsset()
	
	order, _ := NewOrderBuilder().
		ID("TEST-ORDER").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	// Test initial state
	if order.Status != OrderStatusPending {
		t.Errorf("Expected initial status to be PENDING, got %s", order.Status.String())
	}

	// Test submit transition
	err := order.Submit()
	if err != nil {
		t.Errorf("Submit() failed: %v", err)
	}
	if order.Status != OrderStatusSubmitted {
		t.Errorf("Expected status to be SUBMITTED, got %s", order.Status.String())
	}
	if order.SubmittedAt == nil {
		t.Errorf("SubmittedAt should not be nil")
	}

	// Test invalid submit transition
	err = order.Submit()
	if err == nil {
		t.Errorf("Should not be able to submit order twice")
	}

	// Test partial fill
	fill := OrderFill{
		ID:       "FILL-001",
		Price:    types.NewDecimalFromFloat(150.00),
		Quantity: types.NewDecimalFromInt(50),
		Fee:      types.NewDecimalFromFloat(0.50),
	}

	err = order.Fill(fill)
	if err != nil {
		t.Errorf("Fill() failed: %v", err)
	}
	if order.Status != OrderStatusPartiallyFilled {
		t.Errorf("Expected status to be PARTIALLY_FILLED, got %s", order.Status.String())
	}
	if order.FilledQuantity.Cmp(types.NewDecimalFromInt(50)) != 0 {
		t.Errorf("Expected filled quantity to be 50, got %s", order.FilledQuantity.String())
	}
	if order.AvgFillPrice.Cmp(types.NewDecimalFromFloat(150.00)) != 0 {
		t.Errorf("Expected avg fill price to be 150.00, got %s", order.AvgFillPrice.String())
	}

	// Test complete fill
	fill2 := OrderFill{
		ID:       "FILL-002",
		Price:    types.NewDecimalFromFloat(150.10),
		Quantity: types.NewDecimalFromInt(50),
		Fee:      types.NewDecimalFromFloat(0.50),
	}

	err = order.Fill(fill2)
	if err != nil {
		t.Errorf("Fill() failed: %v", err)
	}
	if order.Status != OrderStatusFilled {
		t.Errorf("Expected status to be FILLED, got %s", order.Status.String())
	}
	if order.FilledQuantity.Cmp(types.NewDecimalFromInt(100)) != 0 {
		t.Errorf("Expected filled quantity to be 100, got %s", order.FilledQuantity.String())
	}
	if order.FilledAt == nil {
		t.Errorf("FilledAt should not be nil")
	}

	// Calculate expected average price: (50 * 150.00 + 50 * 150.10) / 100 = 150.05
	expectedAvgPrice := types.NewDecimalFromFloat(150.05)
	diff := order.AvgFillPrice.Sub(expectedAvgPrice).Abs()
	tolerance := types.NewDecimalFromFloat(0.01) // 1 cent tolerance
	if diff.Cmp(tolerance) > 0 {
		t.Errorf("Expected avg fill price to be approximately 150.05, got %s", order.AvgFillPrice.String())
	}

	// Test invalid cancel on filled order
	err = order.Cancel()
	if err == nil {
		t.Errorf("Should not be able to cancel filled order")
	}
}

func TestOrderCancelTransition(t *testing.T) {
	asset := createTestAsset()
	
	order, _ := NewOrderBuilder().
		ID("TEST-ORDER").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	// Submit and then cancel
	order.Submit()
	
	err := order.Cancel()
	if err != nil {
		t.Errorf("Cancel() failed: %v", err)
	}
	if order.Status != OrderStatusCancelled {
		t.Errorf("Expected status to be CANCELLED, got %s", order.Status.String())
	}
	if order.CancelledAt == nil {
		t.Errorf("CancelledAt should not be nil")
	}

	// Test double cancel
	err = order.Cancel()
	if err == nil {
		t.Errorf("Should not be able to cancel order twice")
	}
}

func TestOrderRejectTransition(t *testing.T) {
	asset := createTestAsset()
	
	order, _ := NewOrderBuilder().
		ID("TEST-ORDER").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	err := order.Reject("Insufficient funds")
	if err != nil {
		t.Errorf("Reject() failed: %v", err)
	}
	if order.Status != OrderStatusRejected {
		t.Errorf("Expected status to be REJECTED, got %s", order.Status.String())
	}

	// Test cancel after reject
	err = order.Cancel()
	if err == nil {
		t.Errorf("Should not be able to cancel rejected order")
	}
}

func TestOrderFillValidation(t *testing.T) {
	asset := createTestAsset()
	
	order, _ := NewOrderBuilder().
		ID("TEST-ORDER").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	order.Submit()

	tests := []struct {
		name    string
		fill    OrderFill
		wantErr bool
	}{
		{
			name: "valid fill",
			fill: OrderFill{
				ID:       "FILL-001",
				Price:    types.NewDecimalFromFloat(150.00),
				Quantity: types.NewDecimalFromInt(50),
				Fee:      types.NewDecimalFromFloat(0.50),
			},
			wantErr: false,
		},
		{
			name: "negative quantity",
			fill: OrderFill{
				ID:       "FILL-002",
				Price:    types.NewDecimalFromFloat(150.00),
				Quantity: types.NewDecimalFromFloat(-10),
				Fee:      types.Zero(),
			},
			wantErr: true,
		},
		{
			name: "zero price",
			fill: OrderFill{
				ID:       "FILL-003",
				Price:    types.Zero(),
				Quantity: types.NewDecimalFromInt(10),
				Fee:      types.Zero(),
			},
			wantErr: true,
		},
		{
			name: "quantity exceeds remaining",
			fill: OrderFill{
				ID:       "FILL-004",
				Price:    types.NewDecimalFromFloat(150.00),
				Quantity: types.NewDecimalFromInt(150), // More than order quantity
				Fee:      types.Zero(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh order for each test
			testOrder, _ := NewOrderBuilder().
				ID("TEST-ORDER-" + tt.name).
				Asset(asset).
				Type(OrderTypeLimit).
				Side(OrderSideBuy).
				Quantity(types.NewDecimalFromInt(100)).
				Price(types.NewDecimalFromFloat(150.00)).
				Build()
			testOrder.Submit()

			err := testOrder.Fill(tt.fill)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fill() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderUtilityMethods(t *testing.T) {
	asset := createTestAsset()
	
	order, _ := NewOrderBuilder().
		ID("TEST-ORDER").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	// Test initial state
	if order.IsActive() {
		t.Errorf("Order should not be active initially")
	}
	if order.IsClosed() {
		t.Errorf("Order should not be closed initially")
	}

	// Submit order
	order.Submit()
	if !order.IsActive() {
		t.Errorf("Order should be active after submit")
	}
	if order.IsClosed() {
		t.Errorf("Order should not be closed after submit")
	}

	// Test remaining quantity
	remainingQty := order.RemainingQuantity()
	if remainingQty.Cmp(types.NewDecimalFromInt(100)) != 0 {
		t.Errorf("Expected remaining quantity to be 100, got %s", remainingQty.String())
	}

	// Partial fill
	fill := OrderFill{
		ID:       "FILL-001",
		Price:    types.NewDecimalFromFloat(150.00),
		Quantity: types.NewDecimalFromInt(30),
		Fee:      types.NewDecimalFromFloat(0.30),
	}
	order.Fill(fill)

	remainingQty = order.RemainingQuantity()
	if remainingQty.Cmp(types.NewDecimalFromInt(70)) != 0 {
		t.Errorf("Expected remaining quantity to be 70, got %s", remainingQty.String())
	}

	// Test total fees
	totalFees := order.TotalFees()
	if totalFees.Cmp(types.NewDecimalFromFloat(0.30)) != 0 {
		t.Errorf("Expected total fees to be 0.30, got %s", totalFees.String())
	}

	// Cancel order
	order.Cancel()
	if order.IsActive() {
		t.Errorf("Order should not be active after cancel")
	}
	if !order.IsClosed() {
		t.Errorf("Order should be closed after cancel")
	}
}

func TestOrderExpiration(t *testing.T) {
	asset := createTestAsset()
	
	// Test non-expiring order
	order1, _ := NewOrderBuilder().
		ID("TEST-ORDER-1").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	if order1.IsExpired() {
		t.Errorf("Order without expiration should not be expired")
	}

	// Test future expiration
	futureTime := time.Now().Add(1 * time.Hour)
	order2, _ := NewOrderBuilder().
		ID("TEST-ORDER-2").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		TimeInForce(TimeInForceGTD).
		ExpiresAt(futureTime).
		Build()

	if order2.IsExpired() {
		t.Errorf("Order with future expiration should not be expired")
	}

	// Test past expiration (simulate by setting ExpiresAt manually after build)
	order3, _ := NewOrderBuilder().
		ID("TEST-ORDER-3").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()
	
	// Set expiration time to past manually
	pastTime := time.Now().Add(-1 * time.Hour)
	order3.ExpiresAt = &pastTime

	if !order3.IsExpired() {
		t.Errorf("Order with past expiration should be expired")
	}
}

func TestOrderTypeString(t *testing.T) {
	tests := []struct {
		orderType OrderType
		expected  string
	}{
		{OrderTypeMarket, "MARKET"},
		{OrderTypeLimit, "LIMIT"},
		{OrderTypeStop, "STOP"},
		{OrderTypeStopLimit, "STOP_LIMIT"},
		{OrderTypeTrailingStop, "TRAILING_STOP"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.orderType.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.orderType.String())
			}
		})
	}
}

func TestOrderSideString(t *testing.T) {
	tests := []struct {
		side     OrderSide
		expected string
	}{
		{OrderSideBuy, "BUY"},
		{OrderSideSell, "SELL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.side.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.side.String())
			}
		})
	}
}

func TestTimeInForceString(t *testing.T) {
	tests := []struct {
		tif      TimeInForce
		expected string
	}{
		{TimeInForceGTC, "GTC"},
		{TimeInForceIOC, "IOC"},
		{TimeInForceFOK, "FOK"},
		{TimeInForceDAY, "DAY"},
		{TimeInForceGTD, "GTD"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.tif.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.tif.String())
			}
		})
	}
}

func TestOrderValidation(t *testing.T) {
	asset := createTestAsset()
	
	validOrder, _ := NewOrderBuilder().
		ID("VALID-ORDER").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(100)).
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	// Test valid order
	err := validOrder.Validate()
	if err != nil {
		t.Errorf("Valid order should pass validation: %v", err)
	}

	// Test order with inconsistent fills
	validOrder.FilledQuantity = types.NewDecimalFromInt(50)
	validOrder.Fills = []OrderFill{
		{
			ID:       "FILL-001",
			OrderID:  validOrder.ID,
			Price:    types.NewDecimalFromFloat(150.00),
			Quantity: types.NewDecimalFromInt(30), // Inconsistent with FilledQuantity
			Fee:      types.Zero(),
		},
	}

	err = validOrder.Validate()
	if err == nil {
		t.Errorf("Order with inconsistent fills should fail validation")
	}
}

func BenchmarkOrderCreation(b *testing.B) {
	asset := createTestAsset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewOrderBuilder().
			ID("BENCH-ORDER").
			Asset(asset).
			Type(OrderTypeLimit).
			Side(OrderSideBuy).
			Quantity(types.NewDecimalFromInt(100)).
			Price(types.NewDecimalFromFloat(150.00)).
			Build()
	}
}

func BenchmarkOrderFill(b *testing.B) {
	asset := createTestAsset()
	
	order, _ := NewOrderBuilder().
		ID("BENCH-ORDER").
		Asset(asset).
		Type(OrderTypeLimit).
		Side(OrderSideBuy).
		Quantity(types.NewDecimalFromInt(1000000)). // Large quantity for many fills
		Price(types.NewDecimalFromFloat(150.00)).
		Build()

	order.Submit()

	fill := OrderFill{
		ID:       "FILL-001",
		Price:    types.NewDecimalFromFloat(150.00),
		Quantity: types.NewDecimalFromInt(1),
		Fee:      types.Zero(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create new order for each iteration to avoid overfill errors
		testOrder, _ := NewOrderBuilder().
			ID("BENCH-ORDER").
			Asset(asset).
			Type(OrderTypeLimit).
			Side(OrderSideBuy).
			Quantity(types.NewDecimalFromInt(1000000)).
			Price(types.NewDecimalFromFloat(150.00)).
			Build()
		testOrder.Submit()

		_ = testOrder.Fill(fill)
	}
}