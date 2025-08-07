package domain

import (
	"fmt"
	"testing"

	"github.com/trading-engine/pkg/types"
)

func createTestPortfolio() *Portfolio {
	portfolio, _ := NewPortfolio("PORT-001", "Test Portfolio", types.NewDecimalFromFloat(100000.0))
	return portfolio
}

func createTestPositionForPortfolio(symbol, txType, quantity, price string) *Position {
	asset, _ := NewAssetBuilder().
		Symbol(symbol).
		Name(symbol + " Test").
		Type(AssetTypeStock).
		Exchange("NASDAQ").
		Currency("USD").
		Build()

	qty, _ := types.NewDecimal(quantity)
	prc, _ := types.NewDecimal(price)

	transaction := PositionTransaction{
		ID:       "TX-001",
		Type:     txType,
		Quantity: qty,
		Price:    prc,
		Fee:      types.NewDecimalFromFloat(1.0),
	}

	position, _ := NewPosition("POS-"+symbol, asset, transaction)
	return position
}

func TestNewPortfolio(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		portfolioName  string
		initialCapital string
		wantErr        bool
	}{
		{
			name:           "valid portfolio",
			id:             "PORT-001",
			portfolioName:  "Test Portfolio",
			initialCapital: "100000.0",
			wantErr:        false,
		},
		{
			name:           "empty ID",
			id:             "",
			portfolioName:  "Test Portfolio",
			initialCapital: "100000.0",
			wantErr:        true,
		},
		{
			name:           "empty name",
			id:             "PORT-002",
			portfolioName:  "",
			initialCapital: "100000.0",
			wantErr:        true,
		},
		{
			name:           "zero initial capital",
			id:             "PORT-003",
			portfolioName:  "Test Portfolio",
			initialCapital: "0",
			wantErr:        true,
		},
		{
			name:           "negative initial capital",
			id:             "PORT-004",
			portfolioName:  "Test Portfolio",
			initialCapital: "-1000.0",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capital, _ := types.NewDecimal(tt.initialCapital)
			portfolio, err := NewPortfolio(tt.id, tt.portfolioName, capital)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewPortfolio() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if portfolio == nil {
					t.Errorf("NewPortfolio() returned nil portfolio")
					return
				}

				if portfolio.Status != PortfolioStatusActive {
					t.Errorf("Expected portfolio status ACTIVE, got %v", portfolio.Status)
				}

				if portfolio.CashBalance.Cmp(capital) != 0 {
					t.Errorf("Expected cash balance %v, got %v",
						capital.String(), portfolio.CashBalance.String())
				}

				if portfolio.Metrics.TotalValue.Cmp(capital) != 0 {
					t.Errorf("Expected total value %v, got %v",
						capital.String(), portfolio.Metrics.TotalValue.String())
				}
			}
		})
	}
}

func TestPortfolioAddPosition(t *testing.T) {
	portfolio := createTestPortfolio()

	// Create test position
	position := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")

	// Test adding valid position
	err := portfolio.AddPosition(position)
	if err != nil {
		t.Errorf("AddPosition() failed: %v", err)
	}

	// Verify position was added
	if len(portfolio.Positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(portfolio.Positions))
	}

	// Verify position can be retrieved
	retrievedPos, exists := portfolio.GetPosition(position.ID)
	if !exists {
		t.Errorf("Position not found after adding")
	}

	if retrievedPos.ID != position.ID {
		t.Errorf("Retrieved position ID mismatch")
	}

	// Verify position can be retrieved by asset
	retrievedByAsset, exists := portfolio.GetPositionByAsset("AAPL")
	if !exists {
		t.Errorf("Position not found by asset after adding")
	}

	if retrievedByAsset.ID != position.ID {
		t.Errorf("Retrieved position by asset ID mismatch")
	}

	// Test adding duplicate position for same asset
	duplicatePosition := createTestPositionForPortfolio("AAPL", "BUY", "50", "155.0")
	err = portfolio.AddPosition(duplicatePosition)
	if err == nil {
		t.Errorf("Should not be able to add duplicate position for same asset")
	}
}

func TestPortfolioRemovePosition(t *testing.T) {
	portfolio := createTestPortfolio()
	position := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")

	// Add position first
	portfolio.AddPosition(position)

	// Remove position
	err := portfolio.RemovePosition(position.ID)
	if err != nil {
		t.Errorf("RemovePosition() failed: %v", err)
	}

	// Verify position was removed
	if len(portfolio.Positions) != 0 {
		t.Errorf("Expected 0 positions after removal, got %d", len(portfolio.Positions))
	}

	// Verify position cannot be retrieved
	_, exists := portfolio.GetPosition(position.ID)
	if exists {
		t.Errorf("Position still found after removal")
	}

	// Test removing non-existent position
	err = portfolio.RemovePosition("NON-EXISTENT")
	if err == nil {
		t.Errorf("Should not be able to remove non-existent position")
	}
}

func TestPortfolioCashManagement(t *testing.T) {
	portfolio := createTestPortfolio()
	initialCash := portfolio.CashBalance

	// Test adding cash
	addAmount := types.NewDecimalFromFloat(5000.0)
	err := portfolio.AddCash(addAmount)
	if err != nil {
		t.Errorf("AddCash() failed: %v", err)
	}

	expectedCash := initialCash.Add(addAmount)
	if portfolio.CashBalance.Cmp(expectedCash) != 0 {
		t.Errorf("Expected cash balance %v, got %v",
			expectedCash.String(), portfolio.CashBalance.String())
	}

	// Test withdrawing cash
	withdrawAmount := types.NewDecimalFromFloat(3000.0)
	err = portfolio.WithdrawCash(withdrawAmount)
	if err != nil {
		t.Errorf("WithdrawCash() failed: %v", err)
	}

	expectedCash = expectedCash.Sub(withdrawAmount)
	if portfolio.CashBalance.Cmp(expectedCash) != 0 {
		t.Errorf("Expected cash balance %v after withdrawal, got %v",
			expectedCash.String(), portfolio.CashBalance.String())
	}

	// Test withdrawing more cash than available
	excessAmount := portfolio.CashBalance.Add(types.NewDecimalFromFloat(1000.0))
	err = portfolio.WithdrawCash(excessAmount)
	if err == nil {
		t.Errorf("Should not be able to withdraw more cash than available")
	}

	// Test adding negative cash
	err = portfolio.AddCash(types.NewDecimalFromFloat(-1000.0))
	if err == nil {
		t.Errorf("Should not be able to add negative cash")
	}

	// Test withdrawing negative cash
	err = portfolio.WithdrawCash(types.NewDecimalFromFloat(-1000.0))
	if err == nil {
		t.Errorf("Should not be able to withdraw negative cash")
	}
}

func TestPortfolioUpdatePositionPrice(t *testing.T) {
	portfolio := createTestPortfolio()
	position := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")
	portfolio.AddPosition(position)

	// Update position price
	newPrice := types.NewDecimalFromFloat(160.0)
	err := portfolio.UpdatePositionPrice("AAPL", newPrice)
	if err != nil {
		t.Errorf("UpdatePositionPrice() failed: %v", err)
	}

	// Verify position price was updated
	updatedPosition, _ := portfolio.GetPositionByAsset("AAPL")
	if updatedPosition.CurrentPrice.Cmp(newPrice) != 0 {
		t.Errorf("Expected position price %v, got %v",
			newPrice.String(), updatedPosition.CurrentPrice.String())
	}

	// Test updating price for non-existent asset
	err = portfolio.UpdatePositionPrice("GOOGL", newPrice)
	if err == nil {
		t.Errorf("Should not be able to update price for non-existent asset")
	}
}

func TestPortfolioUpdateAllPrices(t *testing.T) {
	portfolio := createTestPortfolio()

	// Add multiple positions
	position1 := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")
	position2 := createTestPositionForPortfolio("GOOGL", "BUY", "50", "2500.0")
	portfolio.AddPosition(position1)
	portfolio.AddPosition(position2)

	// Update all prices
	prices := map[string]types.Decimal{
		"AAPL":  types.NewDecimalFromFloat(160.0),
		"GOOGL": types.NewDecimalFromFloat(2600.0),
		"MSFT":  types.NewDecimalFromFloat(300.0), // Non-existent position (should be ignored)
	}

	err := portfolio.UpdateAllPrices(prices)
	if err != nil {
		t.Errorf("UpdateAllPrices() failed: %v", err)
	}

	// Verify prices were updated
	aaplPosition, _ := portfolio.GetPositionByAsset("AAPL")
	if aaplPosition.CurrentPrice.Cmp(prices["AAPL"]) != 0 {
		t.Errorf("AAPL price not updated correctly")
	}

	googlPosition, _ := portfolio.GetPositionByAsset("GOOGL")
	if googlPosition.CurrentPrice.Cmp(prices["GOOGL"]) != 0 {
		t.Errorf("GOOGL price not updated correctly")
	}
}

func TestPortfolioMetricsCalculation(t *testing.T) {
	portfolio := createTestPortfolio()

	// Add positions with different performance
	position1 := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")  // Will be profitable
	position2 := createTestPositionForPortfolio("GOOGL", "BUY", "50", "2500.0") // Will be at loss

	portfolio.AddPosition(position1)
	portfolio.AddPosition(position2)

	// Update prices to create P&L
	portfolio.UpdatePositionPrice("AAPL", types.NewDecimalFromFloat(160.0))   // +$1000 profit
	portfolio.UpdatePositionPrice("GOOGL", types.NewDecimalFromFloat(2400.0)) // -$5000 loss

	metrics := portfolio.Metrics

	// Verify position counts
	if metrics.TotalPositions != 2 {
		t.Errorf("Expected 2 total positions, got %d", metrics.TotalPositions)
	}

	if metrics.LongPositions != 2 {
		t.Errorf("Expected 2 long positions, got %d", metrics.LongPositions)
	}

	if metrics.ShortPositions != 0 {
		t.Errorf("Expected 0 short positions, got %d", metrics.ShortPositions)
	}

	// Verify market value calculation
	// AAPL: 100 * 160 = 16000, GOOGL: 50 * 2400 = 120000, Total = 136000
	expectedMarketValue := types.NewDecimalFromFloat(136000.0)
	if metrics.MarketValue.Cmp(expectedMarketValue) != 0 {
		t.Errorf("Expected market value %v, got %v",
			expectedMarketValue.String(), metrics.MarketValue.String())
	}

	// Verify total value (cash + market value)
	expectedTotalValue := portfolio.CashBalance.Add(expectedMarketValue)
	if metrics.TotalValue.Cmp(expectedTotalValue) != 0 {
		t.Errorf("Expected total value %v, got %v",
			expectedTotalValue.String(), metrics.TotalValue.String())
	}

	// Verify unrealized P&L
	// AAPL: (160-150)*100 = 1000, GOOGL: (2400-2500)*50 = -5000, Total = -4000
	expectedUnrealizedPnL := types.NewDecimalFromFloat(-4000.0)
	if metrics.UnrealizedPnL.Cmp(expectedUnrealizedPnL) != 0 {
		t.Errorf("Expected unrealized P&L %v, got %v",
			expectedUnrealizedPnL.String(), metrics.UnrealizedPnL.String())
	}
}

func TestPortfolioPositionFiltering(t *testing.T) {
	portfolio := createTestPortfolio()

	// Add positions with different asset types
	stockPosition := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")
	stockPosition.Asset.AssetType = AssetTypeStock

	cryptoAsset, _ := NewAssetBuilder().
		Symbol("BTCUSD").
		Name("Bitcoin").
		Type(AssetTypeCrypto).
		Build()

	cryptoTx := PositionTransaction{
		ID:       "TX-CRYPTO",
		Type:     "BUY",
		Quantity: types.NewDecimalFromInt(1),
		Price:    types.NewDecimalFromFloat(50000.0),
		Fee:      types.NewDecimalFromFloat(10.0),
	}
	cryptoPosition, _ := NewPosition("POS-CRYPTO", cryptoAsset, cryptoTx)

	portfolio.AddPosition(stockPosition)
	portfolio.AddPosition(cryptoPosition)

	// Test filtering by asset type
	stockPositions := portfolio.GetPositionsByAssetType(AssetTypeStock)
	if len(stockPositions) != 1 {
		t.Errorf("Expected 1 stock position, got %d", len(stockPositions))
	}

	cryptoPositions := portfolio.GetPositionsByAssetType(AssetTypeCrypto)
	if len(cryptoPositions) != 1 {
		t.Errorf("Expected 1 crypto position, got %d", len(cryptoPositions))
	}

	// Test getting open vs closed positions
	openPositions := portfolio.GetOpenPositions()
	if len(openPositions) != 2 {
		t.Errorf("Expected 2 open positions, got %d", len(openPositions))
	}

	closedPositions := portfolio.GetClosedPositions()
	if len(closedPositions) != 0 {
		t.Errorf("Expected 0 closed positions, got %d", len(closedPositions))
	}

	// Close one position
	stockPosition.Close()
	stockPosition.ForceClose()

	openPositions = portfolio.GetOpenPositions()
	if len(openPositions) != 1 {
		t.Errorf("Expected 1 open position after closing one, got %d", len(openPositions))
	}

	closedPositions = portfolio.GetClosedPositions()
	if len(closedPositions) != 1 {
		t.Errorf("Expected 1 closed position after closing one, got %d", len(closedPositions))
	}
}

func TestPortfolioTopPositions(t *testing.T) {
	portfolio := createTestPortfolio()

	// Add positions with different market values
	position1 := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")  // Market value: 15000
	position2 := createTestPositionForPortfolio("GOOGL", "BUY", "50", "2500.0") // Market value: 125000
	position3 := createTestPositionForPortfolio("MSFT", "BUY", "200", "300.0")  // Market value: 60000

	portfolio.AddPosition(position1)
	portfolio.AddPosition(position2)
	portfolio.AddPosition(position3)

	// Get top 2 positions
	topPositions := portfolio.GetTopPositions(2)

	if len(topPositions) != 2 {
		t.Errorf("Expected 2 top positions, got %d", len(topPositions))
	}

	// Should be ordered by market value descending: GOOGL, MSFT
	if topPositions[0].Asset.Symbol != "GOOGL" {
		t.Errorf("Expected first position to be GOOGL, got %s", topPositions[0].Asset.Symbol)
	}

	if topPositions[1].Asset.Symbol != "MSFT" {
		t.Errorf("Expected second position to be MSFT, got %s", topPositions[1].Asset.Symbol)
	}

	// Test requesting more positions than available
	allPositions := portfolio.GetTopPositions(10)
	if len(allPositions) != 3 {
		t.Errorf("Expected 3 positions when requesting more than available, got %d", len(allPositions))
	}
}

func TestPortfolioRebalance(t *testing.T) {
	portfolio := createTestPortfolio()

	// Add positions to create current allocation
	position1 := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")  // Market value: 15000
	position2 := createTestPositionForPortfolio("GOOGL", "BUY", "20", "2500.0") // Market value: 50000

	portfolio.AddPosition(position1)
	portfolio.AddPosition(position2)

	// Total portfolio value: 100000 (cash) + 15000 + 50000 = 165000
	// Current weights: AAPL ~9.09%, GOOGL ~30.30%, Cash ~60.61%

	// Define target weights
	targetWeights := map[string]types.Decimal{
		"AAPL":  types.NewDecimalFromFloat(20.0), // Want 20%
		"GOOGL": types.NewDecimalFromFloat(30.0), // Want 30%
		"MSFT":  types.NewDecimalFromFloat(50.0), // New position 50%
	}

	instructions, err := portfolio.Rebalance(targetWeights)
	if err != nil {
		t.Errorf("Rebalance() failed: %v", err)
	}

	if len(instructions) == 0 {
		t.Errorf("Expected rebalancing instructions, got none")
	}

	// Verify instructions are sorted by magnitude
	for i := 1; i < len(instructions); i++ {
		prevMagnitude := instructions[i-1].ValueDiff.Abs()
		currMagnitude := instructions[i].ValueDiff.Abs()
		if prevMagnitude.Cmp(currMagnitude) < 0 {
			t.Errorf("Instructions not sorted by magnitude")
		}
	}

	// Test invalid target weights (don't sum to 100%)
	invalidWeights := map[string]types.Decimal{
		"AAPL":  types.NewDecimalFromFloat(60.0),
		"GOOGL": types.NewDecimalFromFloat(60.0), // Total 120%
	}

	_, err = portfolio.Rebalance(invalidWeights)
	if err == nil {
		t.Errorf("Should reject target weights that don't sum to 100%%")
	}
}

func TestPortfolioStatusManagement(t *testing.T) {
	portfolio := createTestPortfolio()

	// Test initial status
	if !portfolio.IsActive() {
		t.Errorf("Portfolio should be active initially")
	}

	// Test suspend
	err := portfolio.Suspend()
	if err != nil {
		t.Errorf("Suspend() failed: %v", err)
	}

	if !portfolio.IsSuspended() {
		t.Errorf("Portfolio should be suspended")
	}

	// Test adding position to suspended portfolio
	position := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")
	err = portfolio.AddPosition(position)
	if err == nil {
		t.Errorf("Should not be able to add position to suspended portfolio")
	}

	// Test resume
	err = portfolio.Resume()
	if err != nil {
		t.Errorf("Resume() failed: %v", err)
	}

	if !portfolio.IsActive() {
		t.Errorf("Portfolio should be active after resume")
	}

	// Test close
	err = portfolio.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if !portfolio.IsClosed() {
		t.Errorf("Portfolio should be closed")
	}

	if portfolio.ClosedAt == nil {
		t.Errorf("ClosedAt should not be nil")
	}

	// Test invalid state transitions
	err = portfolio.Resume() // Can't resume closed portfolio
	if err == nil {
		t.Errorf("Should not be able to resume closed portfolio")
	}

	err = portfolio.Suspend() // Can't suspend closed portfolio
	if err == nil {
		t.Errorf("Should not be able to suspend closed portfolio")
	}
}

func TestPortfolioUtilityMethods(t *testing.T) {
	portfolio := createTestPortfolio()

	// Test age calculation
	age := portfolio.GetAge()
	if age <= 0 {
		t.Errorf("Portfolio age should be positive")
	}

	// Add positions for diversification test
	position1 := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")
	position2 := createTestPositionForPortfolio("GOOGL", "BUY", "20", "2500.0")
	position3 := createTestPositionForPortfolio("MSFT", "BUY", "100", "300.0")

	portfolio.AddPosition(position1)
	portfolio.AddPosition(position2)
	portfolio.AddPosition(position3)

	// Test diversification ratio
	diversificationRatio := portfolio.GetDiversificationRatio()
	if diversificationRatio.IsNegative() {
		t.Errorf("Diversification ratio should not be negative")
	}

	// More positions should generally mean better diversification
	// (though this is a simplified test)
	if diversificationRatio.IsZero() && len(portfolio.Positions) > 1 {
		t.Errorf("Diversification ratio should be positive with multiple positions")
	}
}

func TestPortfolioValidation(t *testing.T) {
	// Test valid portfolio
	portfolio := createTestPortfolio()
	err := portfolio.Validate()
	if err != nil {
		t.Errorf("Valid portfolio should pass validation: %v", err)
	}

	tests := []struct {
		name     string
		modifier func(*Portfolio)
		wantErr  bool
	}{
		{
			name: "empty portfolio ID",
			modifier: func(p *Portfolio) {
				p.ID = ""
			},
			wantErr: true,
		},
		{
			name: "empty portfolio name",
			modifier: func(p *Portfolio) {
				p.Name = ""
			},
			wantErr: true,
		},
		{
			name: "negative cash balance",
			modifier: func(p *Portfolio) {
				p.CashBalance = types.NewDecimalFromFloat(-1000.0)
			},
			wantErr: true,
		},
		{
			name: "zero initial capital",
			modifier: func(p *Portfolio) {
				p.InitialCapital = types.Zero()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPortfolio := portfolio.Clone()
			tt.modifier(testPortfolio)

			err := testPortfolio.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPortfolioClone(t *testing.T) {
	portfolio := createTestPortfolio()
	position := createTestPositionForPortfolio("AAPL", "BUY", "100", "150.0")
	portfolio.AddPosition(position)

	// Clone portfolio
	cloned := portfolio.Clone()

	// Verify clone has same data
	if cloned.ID != portfolio.ID {
		t.Errorf("Cloned portfolio ID mismatch")
	}

	if len(cloned.Positions) != len(portfolio.Positions) {
		t.Errorf("Cloned portfolio positions count mismatch")
	}

	// Verify it's a deep copy (modifying clone doesn't affect original)
	cloned.Name = "Modified Clone"
	if portfolio.Name == "Modified Clone" {
		t.Errorf("Clone modification affected original portfolio")
	}

	// Verify positions are also cloned
	for id := range portfolio.Positions {
		originalPos := portfolio.Positions[id]
		clonedPos := cloned.Positions[id]

		if originalPos == clonedPos {
			t.Errorf("Position was not deep copied")
		}

		if originalPos.ID != clonedPos.ID {
			t.Errorf("Cloned position data mismatch")
		}
	}
}

func TestPortfolioStringMethods(t *testing.T) {
	tests := []struct {
		status   PortfolioStatus
		expected string
	}{
		{PortfolioStatusActive, "ACTIVE"},
		{PortfolioStatusSuspended, "SUSPENDED"},
		{PortfolioStatusClosed, "CLOSED"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.status.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.status.String())
			}
		})
	}
}

func BenchmarkPortfolioMetricsCalculation(b *testing.B) {
	portfolio := createTestPortfolio()

	// Add multiple positions
	for i := 0; i < 10; i++ {
		symbol := fmt.Sprintf("TEST%d", i)
		position := createTestPositionForPortfolio(symbol, "BUY", "100", "100.0")
		portfolio.AddPosition(position)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		portfolio.calculateMetrics()
	}
}

func BenchmarkPortfolioRebalance(b *testing.B) {
	portfolio := createTestPortfolio()

	// Add positions
	symbols := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA"}
	for _, symbol := range symbols {
		position := createTestPositionForPortfolio(symbol, "BUY", "100", "100.0")
		portfolio.AddPosition(position)
	}

	targetWeights := map[string]types.Decimal{
		"AAPL":  types.NewDecimalFromFloat(20.0),
		"GOOGL": types.NewDecimalFromFloat(20.0),
		"MSFT":  types.NewDecimalFromFloat(20.0),
		"AMZN":  types.NewDecimalFromFloat(20.0),
		"TSLA":  types.NewDecimalFromFloat(20.0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = portfolio.Rebalance(targetWeights)
	}
}
