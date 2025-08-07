package risk

import (
	"testing"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// TDD: Step 1 - RED - Write failing tests for enhanced Position Sizer

func TestConservativeKellyScaling(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	sizer := NewPositionSizerWithConservativeKelly(ScalingConservative)
	if sizer == nil {
		t.Fatal("Expected position sizer to be created")
	}

	// Test that conservative scaling factor is applied
	scalingFactor := sizer.GetConservativeScalingFactor()
	expectedFactor := types.NewDecimalFromFloat(0.10) // 10% of Kelly optimal

	if scalingFactor.Cmp(expectedFactor) != 0 {
		t.Errorf("Expected conservative scaling factor %s, got %s",
			expectedFactor.String(), scalingFactor.String())
	}
}

func TestConfidenceBasedScaling(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	sizer := NewPositionSizer()

	// Test setting confidence factor
	confidenceFactor := types.NewDecimalFromFloat(0.75) // 75% confidence
	err := sizer.SetConfidenceFactor(confidenceFactor)
	if err != nil {
		t.Fatalf("Failed to set confidence factor: %v", err)
	}

	// Test retrieving confidence factor
	config := sizer.GetConfig()
	if config.ConfidenceFactor.Cmp(confidenceFactor) != 0 {
		t.Errorf("Expected confidence factor %s, got %s",
			confidenceFactor.String(), config.ConfidenceFactor.String())
	}

	// Test invalid confidence factors
	invalidFactors := []types.Decimal{
		types.NewDecimalFromFloat(-0.1),
		types.NewDecimalFromFloat(1.5),
	}

	for _, factor := range invalidFactors {
		err := sizer.SetConfidenceFactor(factor)
		if err == nil {
			t.Errorf("Expected error for invalid confidence factor %s", factor.String())
		}
	}
}

func TestPositionSizeLimits(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	sizer := NewPositionSizer()

	// Configure position limits
	config := SizingConfig{
		MaxPositionSize: types.NewDecimalFromFloat(10.0), // 10% max
		MinPositionSize: types.NewDecimalFromFloat(0.5),  // 0.5% min
	}
	sizer.UpdateConfig(config)

	portfolioValue := types.NewDecimalFromFloat(100000.0)
	price := types.NewDecimalFromFloat(100.0)

	// Test size exceeding maximum - should be capped
	largeSize := types.NewDecimalFromFloat(200) // 20% of portfolio
	cappedSize, err := sizer.ApplyPositionLimits(largeSize, portfolioValue, price)
	if err != nil {
		t.Fatalf("ApplyPositionLimits failed: %v", err)
	}

	cappedValue := cappedSize.Mul(price)
	cappedPercent := cappedValue.Div(portfolioValue).Mul(types.NewDecimalFromInt(100))

	if cappedPercent.Cmp(config.MaxPositionSize) > 0 {
		t.Errorf("Position size not properly capped: %s%% > %s%%",
			cappedPercent.String(), config.MaxPositionSize.String())
	}
}

func TestOptimalSizeCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	portfolio := createTestPortfolio()
	asset := createTestAsset("AAPL")
	sizer := NewPositionSizer()

	entryPrice := types.NewDecimalFromFloat(100.0)
	stopPrice := types.NewDecimalFromFloat(95.0)
	riskPerTrade := types.NewDecimalFromFloat(2.0)
	confidenceFactor := types.NewDecimalFromFloat(0.80)

	result, err := sizer.CalculateOptimalSize(portfolio, asset, entryPrice, stopPrice, riskPerTrade, confidenceFactor)
	if err != nil {
		t.Fatalf("CalculateOptimalSize failed: %v", err)
	}

	// Validate comprehensive analysis was performed
	if result.PrimarySizingMethod == "" {
		t.Error("Expected primary sizing method to be determined")
	}

	if result.RecommendedSize.IsZero() {
		t.Error("Expected non-zero recommended size")
	}

	if result.PositionValue.IsZero() {
		t.Error("Expected non-zero position value calculation")
	}

	// Should prefer stop-loss based sizing when stop price is provided
	if result.PrimarySizingMethod != "StopLoss" {
		t.Errorf("Expected StopLoss as primary method with stop price, got %s", result.PrimarySizingMethod)
	}
}

// Helper functions that will need to be implemented
func createTestPortfolio() *domain.Portfolio {
	portfolio, _ := domain.NewPortfolio("TEST-001", "Test Portfolio", types.NewDecimalFromFloat(100000.0))
	return portfolio
}

func createTestAsset(symbol string) *domain.Asset {
	asset, _ := domain.NewAssetBuilder().
		Symbol(symbol).
		Name(symbol + " Test").
		Type(domain.AssetTypeStock).
		Build()
	return asset
}

// TDD RED phase - Types and functions are now in position_sizer.go
