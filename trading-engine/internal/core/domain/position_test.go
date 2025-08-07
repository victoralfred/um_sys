package domain

import (
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

func createTestTransaction(txType string, quantity, price string) PositionTransaction {
	qty, _ := types.NewDecimal(quantity)
	prc, _ := types.NewDecimal(price)
	
	return PositionTransaction{
		ID:        "TX-001",
		Type:      txType,
		Quantity:  qty,
		Price:     prc,
		Fee:       types.NewDecimalFromFloat(1.0),
		Timestamp: time.Now(),
	}
}

func TestNewPosition(t *testing.T) {
	asset := createTestAsset()

	tests := []struct {
		name        string
		id          string
		asset       *Asset
		transaction PositionTransaction
		wantErr     bool
		wantSide    PositionSide
	}{
		{
			name:        "valid long position",
			id:          "POS-001",
			asset:       asset,
			transaction: createTestTransaction("BUY", "100", "150.00"),
			wantErr:     false,
			wantSide:    PositionSideLong,
		},
		{
			name:        "valid short position",
			id:          "POS-002",
			asset:       asset,
			transaction: createTestTransaction("SELL", "100", "150.00"),
			wantErr:     false,
			wantSide:    PositionSideShort,
		},
		{
			name:        "empty position ID",
			id:          "",
			asset:       asset,
			transaction: createTestTransaction("BUY", "100", "150.00"),
			wantErr:     true,
		},
		{
			name:        "nil asset",
			id:          "POS-003",
			asset:       nil,
			transaction: createTestTransaction("BUY", "100", "150.00"),
			wantErr:     true,
		},
		{
			name:        "zero quantity transaction",
			id:          "POS-004",
			asset:       asset,
			transaction: createTestTransaction("BUY", "0", "150.00"),
			wantErr:     true,
		},
		{
			name:        "zero price transaction",
			id:          "POS-005",
			asset:       asset,
			transaction: createTestTransaction("BUY", "100", "0"),
			wantErr:     true,
		},
		{
			name:        "invalid transaction type",
			id:          "POS-006",
			asset:       asset,
			transaction: PositionTransaction{
				ID:       "TX-001",
				Type:     "INVALID",
				Quantity: types.NewDecimalFromInt(100),
				Price:    types.NewDecimalFromFloat(150.00),
				Fee:      types.NewDecimalFromFloat(1.0),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			position, err := NewPosition(tt.id, tt.asset, tt.transaction)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPosition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if position == nil {
					t.Errorf("NewPosition() returned nil position")
					return
				}
				
				if position.Side != tt.wantSide {
					t.Errorf("Expected position side %v, got %v", tt.wantSide, position.Side)
				}
				
				if position.Status != PositionStatusOpen {
					t.Errorf("Expected position status OPEN, got %v", position.Status)
				}
				
				if position.Quantity.Cmp(tt.transaction.Quantity) != 0 {
					t.Errorf("Expected quantity %v, got %v", 
						tt.transaction.Quantity.String(), position.Quantity.String())
				}
				
				if position.AvgEntryPrice.Cmp(tt.transaction.Price) != 0 {
					t.Errorf("Expected avg entry price %v, got %v", 
						tt.transaction.Price.String(), position.AvgEntryPrice.String())
				}
			}
		})
	}
}

func TestPositionIncreasePosition(t *testing.T) {
	asset := createTestAsset()
	
	// Create initial long position
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, _ := NewPosition("POS-001", asset, initialTx)
	
	// Add more to the position
	additionalTx := createTestTransaction("BUY", "50", "160.00")
	err := position.AddTransaction(additionalTx)
	
	if err != nil {
		t.Errorf("AddTransaction() failed: %v", err)
	}
	
	// Verify quantity increased
	expectedQty := types.NewDecimalFromInt(150)
	if position.Quantity.Cmp(expectedQty) != 0 {
		t.Errorf("Expected quantity %v, got %v", expectedQty.String(), position.Quantity.String())
	}
	
	// Verify average entry price calculation
	// (100 * 150.00 + 50 * 160.00) / 150 = 153.33
	expectedAvgPrice := types.NewDecimalFromFloat(153.33)
	tolerance := types.NewDecimalFromFloat(0.01)
	diff := position.AvgEntryPrice.Sub(expectedAvgPrice).Abs()
	
	if diff.Cmp(tolerance) > 0 {
		t.Errorf("Expected avg entry price approximately %v, got %v", 
			expectedAvgPrice.String(), position.AvgEntryPrice.String())
	}
	
	// Verify cost basis includes fees
	// (100 * 150.00 + 1.00) + (50 * 160.00 + 1.00) = 23002.00
	expectedCostBasis := types.NewDecimalFromFloat(23002.00)
	if position.CostBasis.Cmp(expectedCostBasis) != 0 {
		t.Errorf("Expected cost basis %v, got %v", 
			expectedCostBasis.String(), position.CostBasis.String())
	}
}

func TestPositionReducePosition(t *testing.T) {
	asset := createTestAsset()
	
	// Create initial long position
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, _ := NewPosition("POS-001", asset, initialTx)
	
	// Partially close the position at a profit
	closeTx := createTestTransaction("SELL", "40", "160.00")
	err := position.AddTransaction(closeTx)
	
	if err != nil {
		t.Errorf("AddTransaction() failed: %v", err)
	}
	
	// Verify quantity reduced
	expectedQty := types.NewDecimalFromInt(60)
	if position.Quantity.Cmp(expectedQty) != 0 {
		t.Errorf("Expected quantity %v, got %v", expectedQty.String(), position.Quantity.String())
	}
	
	// Verify realized P&L calculation
	// (160.00 - 150.00) * 40 - 1.00 (fee) = 399.00
	expectedRealizedPnL := types.NewDecimalFromFloat(399.00)
	if position.RealizedPnL.Cmp(expectedRealizedPnL) != 0 {
		t.Errorf("Expected realized P&L %v, got %v", 
			expectedRealizedPnL.String(), position.RealizedPnL.String())
	}
	
	// Position should still be open
	if position.Status != PositionStatusOpen {
		t.Errorf("Expected position to remain open")
	}
	
	if !position.IsLong() {
		t.Errorf("Expected position to remain long")
	}
}

func TestPositionFullClose(t *testing.T) {
	asset := createTestAsset()
	
	// Create initial long position
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, _ := NewPosition("POS-001", asset, initialTx)
	
	// Fully close the position
	closeTx := createTestTransaction("SELL", "100", "160.00")
	err := position.AddTransaction(closeTx)
	
	if err != nil {
		t.Errorf("AddTransaction() failed: %v", err)
	}
	
	// Verify position is closed
	if position.Status != PositionStatusClosed {
		t.Errorf("Expected position status CLOSED, got %v", position.Status)
	}
	
	if position.Side != PositionSideFlat {
		t.Errorf("Expected position side FLAT, got %v", position.Side)
	}
	
	if !position.Quantity.IsZero() {
		t.Errorf("Expected zero quantity, got %v", position.Quantity.String())
	}
	
	if position.ClosedAt == nil {
		t.Errorf("ClosedAt should not be nil")
	}
	
	// Verify realized P&L
	// (160.00 - 150.00) * 100 - 1.00 (fee) = 999.00
	expectedRealizedPnL := types.NewDecimalFromFloat(999.00)
	if position.RealizedPnL.Cmp(expectedRealizedPnL) != 0 {
		t.Errorf("Expected realized P&L %v, got %v", 
			expectedRealizedPnL.String(), position.RealizedPnL.String())
	}
	
	// Unrealized P&L should be zero
	if !position.UnrealizedPnL.IsZero() {
		t.Errorf("Expected zero unrealized P&L, got %v", position.UnrealizedPnL.String())
	}
}

func TestPositionShortPosition(t *testing.T) {
	asset := createTestAsset()
	
	// Create initial short position
	initialTx := createTestTransaction("SELL", "100", "150.00")
	position, _ := NewPosition("POS-001", asset, initialTx)
	
	if !position.IsShort() {
		t.Errorf("Expected short position")
	}
	
	// Update market price (price went down - profit for short)
	err := position.UpdateMarketPrice(types.NewDecimalFromFloat(140.00))
	if err != nil {
		t.Errorf("UpdateMarketPrice() failed: %v", err)
	}
	
	// Verify unrealized P&L for short position
	// (150.00 - 140.00) * 100 = 1000.00
	expectedUnrealizedPnL := types.NewDecimalFromFloat(1000.00)
	if position.UnrealizedPnL.Cmp(expectedUnrealizedPnL) != 0 {
		t.Errorf("Expected unrealized P&L %v, got %v", 
			expectedUnrealizedPnL.String(), position.UnrealizedPnL.String())
	}
	
	// Cover the short position
	coverTx := createTestTransaction("BUY", "100", "140.00")
	err = position.AddTransaction(coverTx)
	if err != nil {
		t.Errorf("AddTransaction() failed: %v", err)
	}
	
	// Verify position is closed with profit
	if position.Status != PositionStatusClosed {
		t.Errorf("Expected position status CLOSED")
	}
	
	// Realized P&L: (150.00 - 140.00) * 100 - 1.00 (fee) = 999.00
	expectedRealizedPnL := types.NewDecimalFromFloat(999.00)
	if position.RealizedPnL.Cmp(expectedRealizedPnL) != 0 {
		t.Errorf("Expected realized P&L %v, got %v", 
			expectedRealizedPnL.String(), position.RealizedPnL.String())
	}
}

func TestPositionUpdateMarketPrice(t *testing.T) {
	asset := createTestAsset()
	
	// Create long position template
	initialTx := createTestTransaction("BUY", "100", "150.00")
	
	tests := []struct {
		name              string
		newPrice          string
		expectedUnrealized string
		wantErr           bool
	}{
		{
			name:              "price increase - profit",
			newPrice:          "160.00",
			expectedUnrealized: "1000", // (160 - 150) * 100
			wantErr:           false,
		},
		{
			name:              "price decrease - loss",
			newPrice:          "140.00",
			expectedUnrealized: "-1000", // (140 - 150) * 100
			wantErr:           false,
		},
		{
			name:              "same price - no change",
			newPrice:          "150.00",
			expectedUnrealized: "0", // (150 - 150) * 100
			wantErr:           false,
		},
		{
			name:     "negative price",
			newPrice: "-10.00",
			wantErr:  true,
		},
		{
			name:     "zero price",
			newPrice: "0",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset position for each test
			testPosition, _ := NewPosition("TEST-POS", asset, initialTx)
			
			newPrice, _ := types.NewDecimal(tt.newPrice)
			err := testPosition.UpdateMarketPrice(newPrice)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateMarketPrice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				expectedUnrealized, _ := types.NewDecimal(tt.expectedUnrealized)
				if testPosition.UnrealizedPnL.Cmp(expectedUnrealized) != 0 {
					t.Errorf("Expected unrealized P&L %v, got %v", 
						expectedUnrealized.String(), testPosition.UnrealizedPnL.String())
				}
				
				// Verify current price was updated
				if testPosition.CurrentPrice.Cmp(newPrice) != 0 {
					t.Errorf("Expected current price %v, got %v", 
						newPrice.String(), testPosition.CurrentPrice.String())
				}
			}
		})
	}
}

func TestPositionRiskMetrics(t *testing.T) {
	asset := createTestAsset()
	
	// Create long position
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, _ := NewPosition("POS-001", asset, initialTx)
	
	// Simulate price movements to test risk metrics
	priceUpdates := []string{"160.00", "170.00", "165.00", "155.00", "145.00"}
	
	for _, priceStr := range priceUpdates {
		price, _ := types.NewDecimal(priceStr)
		position.UpdateMarketPrice(price)
	}
	
	// Check max unrealized P&L (should be at 170.00)
	expectedMaxPnL := types.NewDecimalFromFloat(2000.00) // (170 - 150) * 100
	if position.MaxUnrealizedPnL.Cmp(expectedMaxPnL) != 0 {
		t.Errorf("Expected max unrealized P&L %v, got %v", 
			expectedMaxPnL.String(), position.MaxUnrealizedPnL.String())
	}
	
	// Check min unrealized P&L (should be at 145.00)
	expectedMinPnL := types.NewDecimalFromFloat(-500.00) // (145 - 150) * 100
	if position.MinUnrealizedPnL.Cmp(expectedMinPnL) != 0 {
		t.Errorf("Expected min unrealized P&L %v, got %v", 
			expectedMinPnL.String(), position.MinUnrealizedPnL.String())
	}
	
	// Check max drawdown (from peak of 2000 to current -500 = 2500)
	expectedMaxDrawdown := types.NewDecimalFromFloat(2500.00)
	if position.MaxDrawdown.Cmp(expectedMaxDrawdown) != 0 {
		t.Errorf("Expected max drawdown %v, got %v", 
			expectedMaxDrawdown.String(), position.MaxDrawdown.String())
	}
}

func TestPositionUtilityMethods(t *testing.T) {
	asset := createTestAsset()
	
	// Create position with some transactions
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, err := NewPosition("POS-001", asset, initialTx)
	if err != nil {
		t.Fatalf("Failed to create position: %v", err)
	}
	
	// Add partial close for realized P&L
	closeTx := createTestTransaction("SELL", "40", "160.00")
	position.AddTransaction(closeTx)
	
	// Update market price for unrealized P&L
	position.UpdateMarketPrice(types.NewDecimalFromFloat(155.00))
	
	// Test utility methods
	totalPnL := position.TotalPnL()
	expectedTotalPnL := position.RealizedPnL.Add(position.UnrealizedPnL)
	if totalPnL.Cmp(expectedTotalPnL) != 0 {
		t.Errorf("TotalPnL() mismatch")
	}
	
	netPnL := position.NetPnL()
	expectedNetPnL := totalPnL.Sub(position.TotalFees)
	if netPnL.Cmp(expectedNetPnL) != 0 {
		t.Errorf("NetPnL() mismatch")
	}
	
	// Test boolean methods
	if !position.IsLong() {
		t.Errorf("Should be long position")
	}
	
	if position.IsShort() {
		t.Errorf("Should not be short position")
	}
	
	if position.IsFlat() {
		t.Errorf("Should not be flat position")
	}
	
	if !position.IsOpen() {
		t.Errorf("Should be open position")
	}
	
	if position.IsClosed() {
		t.Errorf("Should not be closed position")
	}
	
	// Test holding period
	holdingPeriod := position.GetHoldingPeriod()
	if holdingPeriod <= 0 {
		t.Errorf("Holding period should be positive")
	}
	
	// Test P&L percentage
	pnlPercentage := position.GetPnLPercentage()
	if pnlPercentage.IsZero() {
		t.Errorf("P&L percentage should not be zero")
	}
}

func TestPositionValidation(t *testing.T) {
	asset := createTestAsset()
	
	// Create valid position
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, _ := NewPosition("POS-001", asset, initialTx)
	
	// Test valid position
	err := position.Validate()
	if err != nil {
		t.Errorf("Valid position should pass validation: %v", err)
	}
	
	// Test invalid positions
	tests := []struct {
		name     string
		modifier func(*Position)
		wantErr  bool
	}{
		{
			name: "empty position ID",
			modifier: func(p *Position) {
				p.ID = ""
			},
			wantErr: true,
		},
		{
			name: "nil asset",
			modifier: func(p *Position) {
				p.Asset = nil
			},
			wantErr: true,
		},
		{
			name: "negative quantity in open position",
			modifier: func(p *Position) {
				p.Quantity = types.NewDecimalFromFloat(-10.0)
			},
			wantErr: true,
		},
		{
			name: "closed position with non-zero quantity",
			modifier: func(p *Position) {
				p.Status = PositionStatusClosed
				p.Quantity = types.NewDecimalFromInt(50)
			},
			wantErr: true,
		},
		{
			name: "negative average entry price",
			modifier: func(p *Position) {
				p.AvgEntryPrice = types.NewDecimalFromFloat(-10.0)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPosition := position.Clone()
			tt.modifier(testPosition)
			
			err := testPosition.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPositionForceClose(t *testing.T) {
	asset := createTestAsset()
	
	// Create position
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, _ := NewPosition("POS-001", asset, initialTx)
	
	// Update market price to create unrealized P&L
	position.UpdateMarketPrice(types.NewDecimalFromFloat(160.00))
	
	// Force close
	err := position.ForceClose()
	if err != nil {
		t.Errorf("ForceClose() failed: %v", err)
	}
	
	// Verify position is closed
	if position.Status != PositionStatusClosed {
		t.Errorf("Expected position status CLOSED")
	}
	
	if position.Side != PositionSideFlat {
		t.Errorf("Expected position side FLAT")
	}
	
	if !position.Quantity.IsZero() {
		t.Errorf("Expected zero quantity")
	}
	
	// Verify unrealized P&L was moved to realized P&L
	expectedRealizedPnL := types.NewDecimalFromFloat(1000.00) // (160 - 150) * 100
	if position.RealizedPnL.Cmp(expectedRealizedPnL) != 0 {
		t.Errorf("Expected realized P&L %v, got %v", 
			expectedRealizedPnL.String(), position.RealizedPnL.String())
	}
	
	if !position.UnrealizedPnL.IsZero() {
		t.Errorf("Expected zero unrealized P&L after force close")
	}
}

func TestPositionStringMethods(t *testing.T) {
	tests := []struct {
		side     PositionSide
		expected string
	}{
		{PositionSideLong, "LONG"},
		{PositionSideShort, "SHORT"},
		{PositionSideFlat, "FLAT"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.side.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.side.String())
			}
		})
	}

	statusTests := []struct {
		status   PositionStatus
		expected string
	}{
		{PositionStatusOpen, "OPEN"},
		{PositionStatusClosed, "CLOSED"},
		{PositionStatusClosing, "CLOSING"},
	}

	for _, tt := range statusTests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.status.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.status.String())
			}
		})
	}
}

func BenchmarkPositionCreation(b *testing.B) {
	asset := createTestAsset()
	initialTx := createTestTransaction("BUY", "100", "150.00")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewPosition("BENCH-POS", asset, initialTx)
	}
}

func BenchmarkPositionPriceUpdate(b *testing.B) {
	asset := createTestAsset()
	initialTx := createTestTransaction("BUY", "100", "150.00")
	position, _ := NewPosition("BENCH-POS", asset, initialTx)
	
	newPrice := types.NewDecimalFromFloat(155.00)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = position.UpdateMarketPrice(newPrice)
	}
}

func BenchmarkPositionAddTransaction(b *testing.B) {
	asset := createTestAsset()
	initialTx := createTestTransaction("BUY", "100", "150.00")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		position, _ := NewPosition("BENCH-POS", asset, initialTx)
		additionalTx := createTestTransaction("BUY", "50", "155.00")
		_ = position.AddTransaction(additionalTx)
	}
}