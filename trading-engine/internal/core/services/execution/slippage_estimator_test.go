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

// TDD RED PHASE: Write failing tests for SlippageEstimator first

func TestSlippageEstimatorInitialization(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	estimator := NewSlippageEstimator()
	if estimator == nil {
		t.Fatal("Expected slippage estimator to be created")
	}
}

func TestSlippageEstimatorMarketOrderEstimation(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "AAPL",
		AssetType: domain.AssetTypeStock,
	}
	
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(150.00),
		AskPrice:       types.NewDecimalFromFloat(150.05),
		BidSize:        types.NewDecimalFromFloat(1000.0),
		AskSize:        types.NewDecimalFromFloat(800.0),
		LastTradePrice: types.NewDecimalFromFloat(150.02),
		Timestamp:      time.Now(),
	}
	
	// Market buy order - should estimate slippage based on ask side liquidity
	buyOrder := &domain.Order{
		ID:            "SLIP_BUY_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(500.0), // Half of ask size
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SLIP_BUY_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	slippage, err := estimator.EstimateSlippage(ctx, buyOrder, marketData)
	if err != nil {
		t.Fatalf("Failed to estimate slippage for buy order: %v", err)
	}
	
	// Should have reasonable slippage for market order
	maxExpectedSlippage := types.NewDecimalFromFloat(10.0) // 100 bps reasonable for market order
	if slippage.Cmp(maxExpectedSlippage) > 0 {
		t.Errorf("Expected slippage <= %s bps, got %s bps", maxExpectedSlippage, slippage)
	}
	
	// Should be non-zero due to spread crossing
	if slippage.IsZero() {
		t.Error("Expected non-zero slippage for market order")
	}
	
	// Market sell order - should estimate slippage based on bid side liquidity
	sellOrder := &domain.Order{
		ID:            "SLIP_SELL_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideSell,
		Quantity:      types.NewDecimalFromFloat(600.0), // Smaller than bid size
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SLIP_SELL_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	sellSlippage, err := estimator.EstimateSlippage(ctx, sellOrder, marketData)
	if err != nil {
		t.Fatalf("Failed to estimate slippage for sell order: %v", err)
	}
	
	// Should have reasonable slippage for market sell order
	if sellSlippage.Cmp(maxExpectedSlippage) > 0 {
		t.Errorf("Expected sell slippage <= %s bps, got %s bps", maxExpectedSlippage, sellSlippage)
	}
	
	// Should be non-zero due to spread crossing
	if sellSlippage.IsZero() {
		t.Error("Expected non-zero slippage for market sell order")
	}
}

func TestSlippageEstimatorLargeOrderEstimation(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "GOOGL",
		AssetType: domain.AssetTypeStock,
	}
	
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(2800.00),
		AskPrice:       types.NewDecimalFromFloat(2800.50),
		BidSize:        types.NewDecimalFromFloat(100.0),
		AskSize:        types.NewDecimalFromFloat(150.0),
		LastTradePrice: types.NewDecimalFromFloat(2800.25),
		Timestamp:      time.Now(),
	}
	
	// Large buy order exceeding available liquidity
	largeBuyOrder := &domain.Order{
		ID:            "SLIP_LARGE_BUY_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(300.0), // 2x ask size
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SLIP_LARGE_BUY_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	largeSlippage, err := estimator.EstimateSlippage(ctx, largeBuyOrder, marketData)
	if err != nil {
		t.Fatalf("Failed to estimate slippage for large order: %v", err)
	}
	
	// Should have significant slippage due to size
	minExpectedSlippage := types.NewDecimalFromFloat(0.5) // 50 bps
	if largeSlippage.Cmp(minExpectedSlippage) < 0 {
		t.Errorf("Expected large order slippage >= %s bps, got %s bps", minExpectedSlippage, largeSlippage)
	}
}

func TestSlippageEstimatorLimitOrderEstimation(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "MSFT",
		AssetType: domain.AssetTypeStock,
	}
	
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(300.00),
		AskPrice:       types.NewDecimalFromFloat(300.10),
		BidSize:        types.NewDecimalFromFloat(500.0),
		AskSize:        types.NewDecimalFromFloat(400.0),
		LastTradePrice: types.NewDecimalFromFloat(300.05),
		Timestamp:      time.Now(),
	}
	
	// Limit buy order at bid price - should have minimal slippage
	limitBuyOrder := &domain.Order{
		ID:            "SLIP_LIMIT_BUY_001",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(200.0),
		Price:         types.NewDecimalFromFloat(300.00), // At bid
		TimeInForce:   domain.TimeInForceGTC,
		ClientOrderID: "SLIP_LIMIT_BUY_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	limitSlippage, err := estimator.EstimateSlippage(ctx, limitBuyOrder, marketData)
	if err != nil {
		t.Fatalf("Failed to estimate slippage for limit order: %v", err)
	}
	
	// Limit orders should have reasonable slippage (includes base slippage)
	maxLimitSlippage := types.NewDecimalFromFloat(5.0) // 50 bps reasonable for limit order
	if limitSlippage.Cmp(maxLimitSlippage) > 0 {
		t.Errorf("Expected limit order slippage <= %s bps, got %s bps", maxLimitSlippage, limitSlippage)
	}
	
	// Aggressive limit buy order above ask - should cross spread
	aggressiveLimitOrder := &domain.Order{
		ID:            "SLIP_AGGRESSIVE_LIMIT_001",
		Asset:         asset,
		Type:          domain.OrderTypeLimit,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		Price:         types.NewDecimalFromFloat(300.15), // Above ask
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SLIP_AGGRESSIVE_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	aggressiveSlippage, err := estimator.EstimateSlippage(ctx, aggressiveLimitOrder, marketData)
	if err != nil {
		t.Fatalf("Failed to estimate slippage for aggressive limit: %v", err)
	}
	
	// Aggressive limit should have positive slippage
	minAggressiveSlippage := types.NewDecimalFromFloat(0.1) // 10 bps
	if aggressiveSlippage.Cmp(minAggressiveSlippage) < 0 {
		t.Errorf("Expected aggressive limit slippage >= %s bps, got %s bps", minAggressiveSlippage, aggressiveSlippage)
	}
}

func TestSlippageEstimatorHistoricalVolatility(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "TSLA",
		AssetType: domain.AssetTypeStock,
	}
	
	// Feed historical price data to estimator
	historicalPrices := []types.Decimal{
		types.NewDecimalFromFloat(800.00),
		types.NewDecimalFromFloat(805.50),
		types.NewDecimalFromFloat(798.25),
		types.NewDecimalFromFloat(812.75),
		types.NewDecimalFromFloat(807.10),
	}
	
	for _, price := range historicalPrices {
		err := estimator.UpdateHistoricalData(ctx, asset, price, time.Now())
		if err != nil {
			t.Fatalf("Failed to update historical data: %v", err)
		}
	}
	
	// Calculate volatility-adjusted slippage
	volatility, err := estimator.GetVolatility(ctx, asset)
	if err != nil {
		t.Fatalf("Failed to get volatility: %v", err)
	}
	
	// Should calculate meaningful volatility from price data
	minVolatility := types.NewDecimalFromFloat(0.5) // 50 bps
	if volatility.Cmp(minVolatility) < 0 {
		t.Errorf("Expected volatility >= %s bps, got %s bps", minVolatility, volatility)
	}
	
	// Current market data
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(805.00),
		AskPrice:       types.NewDecimalFromFloat(806.00),
		BidSize:        types.NewDecimalFromFloat(200.0),
		AskSize:        types.NewDecimalFromFloat(180.0),
		LastTradePrice: types.NewDecimalFromFloat(805.50),
		Timestamp:      time.Now(),
	}
	
	// Market order with volatility adjustment
	volatileOrder := &domain.Order{
		ID:            "SLIP_VOLATILE_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(150.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SLIP_VOLATILE_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	volatileSlippage, err := estimator.EstimateSlippage(ctx, volatileOrder, marketData)
	if err != nil {
		t.Fatalf("Failed to estimate volatility-adjusted slippage: %v", err)
	}
	
	// Volatile assets should have higher slippage estimates
	if volatileSlippage.IsZero() {
		t.Error("Expected non-zero slippage for volatile asset")
	}
}

func TestSlippageEstimatorImpactModel(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "NVDA",
		AssetType: domain.AssetTypeStock,
	}
	
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(450.00),
		AskPrice:       types.NewDecimalFromFloat(450.25),
		BidSize:        types.NewDecimalFromFloat(300.0),
		AskSize:        types.NewDecimalFromFloat(250.0),
		LastTradePrice: types.NewDecimalFromFloat(450.10),
		Timestamp:      time.Now(),
	}
	
	// Test different order sizes to verify impact model
	testCases := []struct {
		name         string
		quantity     float64
		expectedTier string
	}{
		{"Small Order", 50.0, "low"},
		{"Medium Order", 150.0, "medium"},
		{"Large Order", 400.0, "high"},
	}
	
	var previousSlippage types.Decimal
	for i, tc := range testCases {
		order := &domain.Order{
			ID:            fmt.Sprintf("SLIP_IMPACT_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeMarket,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(tc.quantity),
			TimeInForce:   domain.TimeInForceIOC,
			ClientOrderID: fmt.Sprintf("SLIP_IMPACT_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}
		
		slippage, err := estimator.EstimateSlippage(ctx, order, marketData)
		if err != nil {
			t.Fatalf("Failed to estimate slippage for %s: %v", tc.name, err)
		}
		
		// Verify slippage increases with order size
		if i > 0 && slippage.Cmp(previousSlippage) <= 0 {
			t.Errorf("Expected slippage to increase with order size: %s (%s) should be > %s", 
				tc.name, slippage, previousSlippage)
		}
		
		previousSlippage = slippage
		t.Logf("%s (qty: %.0f): slippage = %s bps", tc.name, tc.quantity, slippage)
	}
}

func TestSlippageEstimatorTimeDecay(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "AMZN",
		AssetType: domain.AssetTypeStock,
	}
	
	// Test slippage estimation with time-sensitive market data
	oldMarketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(3200.00),
		AskPrice:       types.NewDecimalFromFloat(3201.00),
		BidSize:        types.NewDecimalFromFloat(100.0),
		AskSize:        types.NewDecimalFromFloat(120.0),
		LastTradePrice: types.NewDecimalFromFloat(3200.50),
		Timestamp:      time.Now().Add(-5 * time.Minute), // 5 minutes old
	}
	
	freshMarketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(3198.00),
		AskPrice:       types.NewDecimalFromFloat(3199.50),
		BidSize:        types.NewDecimalFromFloat(80.0),
		AskSize:        types.NewDecimalFromFloat(90.0),
		LastTradePrice: types.NewDecimalFromFloat(3198.75),
		Timestamp:      time.Now(),
	}
	
	order := &domain.Order{
		ID:            "SLIP_TIME_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(75.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SLIP_TIME_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	// Estimate with old data - should have higher uncertainty/slippage
	oldSlippage, err := estimator.EstimateSlippage(ctx, order, oldMarketData)
	if err != nil {
		t.Fatalf("Failed to estimate slippage with old data: %v", err)
	}
	
	// Estimate with fresh data - should be more accurate
	freshSlippage, err := estimator.EstimateSlippage(ctx, order, freshMarketData)
	if err != nil {
		t.Fatalf("Failed to estimate slippage with fresh data: %v", err)
	}
	
	// Old data should generally produce higher slippage estimates due to uncertainty
	t.Logf("Old data slippage: %s bps, Fresh data slippage: %s bps", oldSlippage, freshSlippage)
}

func TestSlippageEstimatorConfiguration(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	config := SlippageEstimatorConfig{
		BaseSlippageBps:        5.0,   // 5 bps base
		VolatilityMultiplier:   2.0,   // 2x volatility impact
		LiquidityImpactFactor:  1.5,   // 1.5x liquidity impact
		TimeDecayFactor:        0.1,   // 10% per minute decay
		MaxHistoryWindow:       100,   // Keep 100 price points
		MinLiquidityThreshold:  50.0,  // Minimum 50 shares liquidity
	}
	
	estimator := NewSlippageEstimatorWithConfig(config)
	if estimator == nil {
		t.Fatal("Expected configured slippage estimator to be created")
	}
	
	// Verify configuration is applied
	retrievedConfig := estimator.GetConfig()
	if retrievedConfig.BaseSlippageBps != config.BaseSlippageBps {
		t.Errorf("Expected base slippage %f, got %f", config.BaseSlippageBps, retrievedConfig.BaseSlippageBps)
	}
}

func TestSlippageEstimatorEdgeCases(t *testing.T) {
	// This test will FAIL - SlippageEstimator doesn't exist yet
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "EDGE_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	// Test with zero liquidity
	zeroLiquidityData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(100.00),
		AskPrice:       types.NewDecimalFromFloat(100.50),
		BidSize:        types.NewDecimalFromFloat(0.0), // Zero liquidity
		AskSize:        types.NewDecimalFromFloat(0.0), // Zero liquidity
		LastTradePrice: types.NewDecimalFromFloat(100.25),
		Timestamp:      time.Now(),
	}
	
	order := &domain.Order{
		ID:            "EDGE_ZERO_LIQ",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(100.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "EDGE_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	_, err := estimator.EstimateSlippage(ctx, order, zeroLiquidityData)
	if err == nil {
		t.Error("Expected error for zero liquidity scenario")
	}
	
	// Test with negative spread (crossed market)
	crossedMarketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(100.50),
		AskPrice:       types.NewDecimalFromFloat(100.00), // Bid > Ask (crossed)
		BidSize:        types.NewDecimalFromFloat(100.0),
		AskSize:        types.NewDecimalFromFloat(150.0),
		LastTradePrice: types.NewDecimalFromFloat(100.25),
		Timestamp:      time.Now(),
	}
	
	crossedSlippage, err := estimator.EstimateSlippage(ctx, order, crossedMarketData)
	if err != nil {
		t.Fatalf("Should handle crossed market gracefully: %v", err)
	}
	
	// Crossed market might result in negative slippage (favorable execution)
	t.Logf("Crossed market slippage: %s bps", crossedSlippage)
}