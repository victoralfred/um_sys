package execution

import (
	"context"
	"testing"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// TDD REFACTOR phase tests for enhanced SlippageEstimator functionality

func TestSlippageEstimatorMetricsTracking(t *testing.T) {
	ctx := context.Background()
	estimator := NewSlippageEstimator()

	asset := &domain.Asset{
		Symbol:    "METRICS_TEST",
		AssetType: domain.AssetTypeStock,
	}

	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(100.00),
		AskPrice:       types.NewDecimalFromFloat(100.05),
		BidSize:        types.NewDecimalFromFloat(500.0),
		AskSize:        types.NewDecimalFromFloat(400.0),
		LastTradePrice: types.NewDecimalFromFloat(100.02),
		Timestamp:      time.Now(),
	}

	// Perform multiple estimations
	for i := 0; i < 5; i++ {
		order := &domain.Order{
			ID:            "METRICS_ORDER_" + string(rune(i+'0')),
			Asset:         asset,
			Type:          domain.OrderTypeMarket,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(100.0),
			TimeInForce:   domain.TimeInForceIOC,
			ClientOrderID: "METRICS_CLIENT",
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}

		_, err := estimator.EstimateSlippage(ctx, order, marketData)
		if err != nil {
			t.Fatalf("Failed to estimate slippage for order %d: %v", i, err)
		}
	}

	// Check metrics
	metrics := estimator.GetMetrics()
	if metrics.TotalEstimations != 5 {
		t.Errorf("Expected 5 estimations, got %d", metrics.TotalEstimations)
	}

	if metrics.EstimationsPerSymbol["METRICS_TEST"] != 5 {
		t.Errorf("Expected 5 estimations for METRICS_TEST, got %d", metrics.EstimationsPerSymbol["METRICS_TEST"])
	}

	if metrics.AverageEstimationTime == 0 {
		t.Error("Expected non-zero average estimation time")
	}

	// Test metrics reset
	estimator.ResetMetrics()
	resetMetrics := estimator.GetMetrics()
	if resetMetrics.TotalEstimations != 0 {
		t.Errorf("Expected 0 estimations after reset, got %d", resetMetrics.TotalEstimations)
	}
}

func TestSlippageEstimatorLiquidityProfileTracking(t *testing.T) {
	estimator := NewSlippageEstimator()
	symbol := "LIQUIDITY_TEST"

	// Update liquidity profile multiple times
	bidSize := types.NewDecimalFromFloat(1000.0)
	askSize := types.NewDecimalFromFloat(800.0)
	spread := types.NewDecimalFromFloat(0.05)

	estimator.UpdateLiquidityProfile(symbol, bidSize, askSize, spread)

	// Retrieve profile
	profile, exists := estimator.GetLiquidityProfile(symbol)
	if !exists {
		t.Fatal("Expected liquidity profile to exist")
	}

	if profile.AverageBidSize.Cmp(bidSize) != 0 {
		t.Errorf("Expected average bid size %s, got %s", bidSize, profile.AverageBidSize)
	}

	if profile.LiquidityScore == 0 {
		t.Error("Expected non-zero liquidity score")
	}

	// Update with different values
	newBidSize := types.NewDecimalFromFloat(1200.0)
	newAskSize := types.NewDecimalFromFloat(900.0)
	newSpread := types.NewDecimalFromFloat(0.06)

	estimator.UpdateLiquidityProfile(symbol, newBidSize, newAskSize, newSpread)

	// Profile should be updated with moving average
	updatedProfile, _ := estimator.GetLiquidityProfile(symbol)
	if updatedProfile.AverageBidSize.Cmp(bidSize) == 0 {
		t.Error("Expected liquidity profile to be updated with moving average")
	}
}

func TestSlippageEstimatorMarketRegimeAnalysis(t *testing.T) {
	estimator := NewSlippageEstimator()

	// Test normal market conditions
	estimator.UpdateMarketRegime(1.0, 0.1, 0.2)
	regime := estimator.GetMarketRegime()

	if regime.RegimeType != "normal" {
		t.Errorf("Expected normal regime, got %s", regime.RegimeType)
	}

	if regime.TrendDirection != "sideways" {
		t.Errorf("Expected sideways trend, got %s", regime.TrendDirection)
	}

	// Test volatile market conditions
	estimator.UpdateMarketRegime(2.5, 0.4, 0.8)
	volatileRegime := estimator.GetMarketRegime()

	if volatileRegime.RegimeType != "volatile" {
		t.Errorf("Expected volatile regime, got %s", volatileRegime.RegimeType)
	}

	if volatileRegime.TrendDirection != "up" {
		t.Errorf("Expected up trend, got %s", volatileRegime.TrendDirection)
	}

	if volatileRegime.MarketStress != 0.8 {
		t.Errorf("Expected market stress 0.8, got %f", volatileRegime.MarketStress)
	}
}

func TestSlippageEstimatorImpactModelCalibration(t *testing.T) {
	estimator := NewSlippageEstimator()
	symbol := "CALIBRATION_TEST"

	// Create synthetic execution data
	executionData := []ExecutionDataPoint{
		{OrderSizeRatio: 0.1, ActualSlippage: 2.0, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.2, ActualSlippage: 4.5, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.3, ActualSlippage: 7.2, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.4, ActualSlippage: 10.1, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.5, ActualSlippage: 13.5, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.6, ActualSlippage: 17.2, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.7, ActualSlippage: 21.3, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.8, ActualSlippage: 25.8, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.9, ActualSlippage: 30.7, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 1.0, ActualSlippage: 36.1, MarketCondition: "normal", Timestamp: time.Now()},
	}

	// Calibrate impact model
	err := estimator.CalibrateImpactModel(symbol, executionData)
	if err != nil {
		t.Fatalf("Failed to calibrate impact model: %v", err)
	}

	// Check that model was created
	estimator.mu.RLock()
	model, exists := estimator.impactModels[symbol]
	estimator.mu.RUnlock()

	if !exists {
		t.Fatal("Expected impact model to be created")
	}

	if model.LinearImpact == 0 {
		t.Error("Expected non-zero linear impact")
	}

	if model.ModelConfidence == 0 {
		t.Error("Expected non-zero model confidence")
	}

	if model.SampleSize != len(executionData) {
		t.Errorf("Expected sample size %d, got %d", len(executionData), model.SampleSize)
	}

	t.Logf("Calibrated model - Linear Impact: %f, Confidence: %f", model.LinearImpact, model.ModelConfidence)
}

func TestSlippageEstimatorModelBasedEstimation(t *testing.T) {
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	symbol := "MODEL_TEST"

	asset := &domain.Asset{
		Symbol:    symbol,
		AssetType: domain.AssetTypeStock,
	}

	// First calibrate a model
	executionData := []ExecutionDataPoint{
		{OrderSizeRatio: 0.1, ActualSlippage: 1.5, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.2, ActualSlippage: 3.2, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.3, ActualSlippage: 5.1, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.4, ActualSlippage: 7.3, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.5, ActualSlippage: 9.8, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.6, ActualSlippage: 12.7, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.7, ActualSlippage: 16.1, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.8, ActualSlippage: 19.9, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.9, ActualSlippage: 24.2, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 1.0, ActualSlippage: 29.1, MarketCondition: "normal", Timestamp: time.Now()},
	}

	err := estimator.CalibrateImpactModel(symbol, executionData)
	if err != nil {
		t.Fatalf("Failed to calibrate model: %v", err)
	}

	// Create market data and order
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(200.00),
		AskPrice:       types.NewDecimalFromFloat(200.10),
		BidSize:        types.NewDecimalFromFloat(1000.0),
		AskSize:        types.NewDecimalFromFloat(900.0),
		LastTradePrice: types.NewDecimalFromFloat(200.05),
		Timestamp:      time.Now(),
	}

	order := &domain.Order{
		ID:            "MODEL_ORDER_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(500.0), // 50% of average volume (assuming 1000)
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "MODEL_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	// Test model-based estimation
	averageVolume := types.NewDecimalFromFloat(1000.0)
	modelSlippage, err := estimator.EstimateSlippageWithModel(ctx, order, marketData, averageVolume)
	if err != nil {
		t.Fatalf("Failed to estimate slippage with model: %v", err)
	}

	// Test generic estimation for comparison
	genericSlippage, err := estimator.EstimateSlippage(ctx, order, marketData)
	if err != nil {
		t.Fatalf("Failed to estimate generic slippage: %v", err)
	}

	t.Logf("Model-based slippage: %s bps, Generic slippage: %s bps", modelSlippage, genericSlippage)

	// Model-based should be non-zero
	if modelSlippage.IsZero() {
		t.Error("Expected non-zero model-based slippage")
	}
}

func TestSlippageEstimatorRegimeAdjustments(t *testing.T) {
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	symbol := "REGIME_TEST"

	asset := &domain.Asset{
		Symbol:    symbol,
		AssetType: domain.AssetTypeStock,
	}

	// Calibrate model first
	executionData := []ExecutionDataPoint{
		{OrderSizeRatio: 0.2, ActualSlippage: 5.0, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.4, ActualSlippage: 10.0, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.6, ActualSlippage: 15.0, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.8, ActualSlippage: 20.0, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 1.0, ActualSlippage: 25.0, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.3, ActualSlippage: 7.5, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.5, ActualSlippage: 12.5, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.7, ActualSlippage: 17.5, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.9, ActualSlippage: 22.5, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.1, ActualSlippage: 2.5, MarketCondition: "normal", Timestamp: time.Now()},
	}

	err := estimator.CalibrateImpactModel(symbol, executionData)
	if err != nil {
		t.Fatalf("Failed to calibrate model: %v", err)
	}

	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(150.00),
		AskPrice:       types.NewDecimalFromFloat(150.10),
		BidSize:        types.NewDecimalFromFloat(800.0),
		AskSize:        types.NewDecimalFromFloat(700.0),
		LastTradePrice: types.NewDecimalFromFloat(150.05),
		Timestamp:      time.Now(),
	}

	order := &domain.Order{
		ID:            "REGIME_ORDER_001",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(200.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "REGIME_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}

	averageVolume := types.NewDecimalFromFloat(1000.0)

	// Test normal regime
	estimator.UpdateMarketRegime(1.0, 0.0, 0.1)
	normalSlippage, err := estimator.EstimateSlippageWithModel(ctx, order, marketData, averageVolume)
	if err != nil {
		t.Fatalf("Failed to estimate normal regime slippage: %v", err)
	}

	// Test volatile regime
	estimator.UpdateMarketRegime(2.5, 0.0, 0.8)
	volatileSlippage, err := estimator.EstimateSlippageWithModel(ctx, order, marketData, averageVolume)
	if err != nil {
		t.Fatalf("Failed to estimate volatile regime slippage: %v", err)
	}

	// Volatile regime should have higher slippage
	if volatileSlippage.Cmp(normalSlippage) <= 0 {
		t.Errorf("Expected volatile regime slippage (%s) > normal regime slippage (%s)", volatileSlippage, normalSlippage)
	}

	t.Logf("Normal regime: %s bps, Volatile regime: %s bps", normalSlippage, volatileSlippage)
}

func TestSlippageEstimatorInsufficientDataHandling(t *testing.T) {
	estimator := NewSlippageEstimator()
	symbol := "INSUFFICIENT_DATA"

	// Try to calibrate with insufficient data
	insufficientData := []ExecutionDataPoint{
		{OrderSizeRatio: 0.1, ActualSlippage: 1.0, MarketCondition: "normal", Timestamp: time.Now()},
		{OrderSizeRatio: 0.2, ActualSlippage: 2.0, MarketCondition: "normal", Timestamp: time.Now()},
	}

	err := estimator.CalibrateImpactModel(symbol, insufficientData)
	if err == nil {
		t.Error("Expected error for insufficient calibration data")
	}

	// Model should not exist
	estimator.mu.RLock()
	_, exists := estimator.impactModels[symbol]
	estimator.mu.RUnlock()

	if exists {
		t.Error("Expected no impact model to be created with insufficient data")
	}
}
