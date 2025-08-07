package risk

import (
	"testing"

	"github.com/trading-engine/pkg/types"
)

// TDD: Step 1 - RED - Write failing tests for VaR Calculator

func TestHistoricalVaRCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewVaRCalculator()
	if calculator == nil {
		t.Fatal("Expected VaR calculator to be created")
	}

	// Reduce minimum observations for this test
	config := calculator.GetConfig()
	config.MinHistoricalObservations = 5
	calculator.SetConfig(config)

	// Test historical VaR calculation with sample returns
	returns := []types.Decimal{
		types.NewDecimalFromFloat(-0.05), // -5%
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(-0.02), // -2%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.08), // -8%
		types.NewDecimalFromFloat(0.04),  // 4%
		types.NewDecimalFromFloat(-0.01), // -1%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(-0.03), // -3%
		types.NewDecimalFromFloat(0.05),  // 5%
	}

	portfolioValue := types.NewDecimalFromFloat(1000000.0) // $1M portfolio
	confidence := types.NewDecimalFromFloat(95.0)           // 95% confidence level

	varResult, err := calculator.CalculateHistoricalVaR(returns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateHistoricalVaR failed: %v", err)
	}

	// Verify VaR result structure
	if varResult.Method != "Historical" {
		t.Errorf("Expected method 'Historical', got %s", varResult.Method)
	}

	if varResult.ConfidenceLevel.Cmp(confidence) != 0 {
		t.Errorf("Expected confidence level %s, got %s", 
			confidence.String(), varResult.ConfidenceLevel.String())
	}

	if varResult.VaR.IsZero() {
		t.Error("Expected non-zero VaR value")
	}

	if varResult.VaR.IsPositive() {
		t.Error("VaR should be negative (representing potential loss)")
	}

	// VaR should be reasonable for the given data (between -8% and 0% for this dataset)
	varPercent := varResult.VaR.Div(portfolioValue).Mul(types.NewDecimalFromInt(100))
	minExpected := types.NewDecimalFromFloat(-10.0) // Should not be worse than -10%
	maxExpected := types.NewDecimalFromFloat(0.0)   // Should not be positive

	if varPercent.Cmp(minExpected) < 0 || varPercent.Cmp(maxExpected) > 0 {
		t.Errorf("VaR percentage %s%% outside expected range [%s%%, %s%%]", 
			varPercent.String(), minExpected.String(), maxExpected.String())
	}
}

func TestParametricVaRCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewVaRCalculator()

	// Test parametric VaR calculation
	returns := []types.Decimal{
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.02), // -2%
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(-0.01), // -1%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(-0.03), // -3%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.01), // -1%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(-0.02), // -2%
	}

	portfolioValue := types.NewDecimalFromFloat(500000.0) // $500k portfolio  
	confidence := types.NewDecimalFromFloat(99.0)         // 99% confidence level

	varResult, err := calculator.CalculateParametricVaR(returns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateParametricVaR failed: %v", err)
	}

	// Verify parametric VaR result
	if varResult.Method != "Parametric" {
		t.Errorf("Expected method 'Parametric', got %s", varResult.Method)
	}

	if varResult.ConfidenceLevel.Cmp(confidence) != 0 {
		t.Errorf("Expected confidence level %s, got %s", 
			confidence.String(), varResult.ConfidenceLevel.String())
	}

	if varResult.VaR.IsZero() {
		t.Error("Expected non-zero parametric VaR value")
	}

	// Should include statistical metrics
	if varResult.Statistics.Mean.IsZero() && varResult.Statistics.StandardDeviation.IsZero() {
		t.Error("Expected statistical measures to be calculated")
	}

	// Higher confidence level should result in higher VaR (more negative)
	confidence95 := types.NewDecimalFromFloat(95.0)
	varResult95, err := calculator.CalculateParametricVaR(returns, portfolioValue, confidence95)
	if err != nil {
		t.Fatalf("CalculateParametricVaR failed for 95%%: %v", err)
	}

	// 99% VaR should be more negative than 95% VaR
	if varResult.VaR.Cmp(varResult95.VaR) > 0 {
		t.Errorf("99%% VaR (%s) should be more negative than 95%% VaR (%s)",
			varResult.VaR.String(), varResult95.VaR.String())
	}
}

func TestMonteCarloVaRCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewVaRCalculator()

	// Configure Monte Carlo parameters
	config := MonteCarloConfig{
		NumSimulations: 10000,
		TimeHorizon:    1, // 1 day
		RandomSeed:     12345,
	}

	err := calculator.SetMonteCarloConfig(config)
	if err != nil {
		t.Fatalf("Failed to set Monte Carlo config: %v", err)
	}

	// Sample historical data for Monte Carlo
	returns := []types.Decimal{
		types.NewDecimalFromFloat(0.015),  // 1.5%
		types.NewDecimalFromFloat(-0.025), // -2.5%
		types.NewDecimalFromFloat(0.031),  // 3.1%
		types.NewDecimalFromFloat(-0.018), // -1.8%
		types.NewDecimalFromFloat(0.022),  // 2.2%
		types.NewDecimalFromFloat(-0.035), // -3.5%
		types.NewDecimalFromFloat(0.012),  // 1.2%
		types.NewDecimalFromFloat(-0.009), // -0.9%
		types.NewDecimalFromFloat(0.028),  // 2.8%
		types.NewDecimalFromFloat(-0.021), // -2.1%
	}

	portfolioValue := types.NewDecimalFromFloat(2000000.0) // $2M portfolio
	confidence := types.NewDecimalFromFloat(95.0)

	varResult, err := calculator.CalculateMonteCarloVaR(returns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateMonteCarloVaR failed: %v", err)
	}

	// Verify Monte Carlo VaR result
	if varResult.Method != "MonteCarlo" {
		t.Errorf("Expected method 'MonteCarlo', got %s", varResult.Method)
	}

	if varResult.MonteCarloDetails.NumSimulations != config.NumSimulations {
		t.Errorf("Expected %d simulations, got %d", 
			config.NumSimulations, varResult.MonteCarloDetails.NumSimulations)
	}

	if varResult.VaR.IsZero() {
		t.Error("Expected non-zero Monte Carlo VaR value")
	}

	// Monte Carlo should provide simulation statistics
	if varResult.MonteCarloDetails.WorstCaseScenario.IsZero() {
		t.Error("Expected worst case scenario to be calculated")
	}

	if varResult.MonteCarloDetails.BestCaseScenario.IsZero() {
		t.Error("Expected best case scenario to be calculated")
	}
}

func TestVaRModelValidation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewVaRCalculator()
	
	// Reduce minimum observations for this test
	config := calculator.GetConfig()
	config.MinHistoricalObservations = 5
	calculator.SetConfig(config)

	// Test backtesting functionality
	historicalReturns := []types.Decimal{
		types.NewDecimalFromFloat(-0.01), // -1%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(-0.03), // -3%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.04), // -4%
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(-0.02), // -2%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.05), // -5%
		types.NewDecimalFromFloat(0.02),  // 2%
	}

	outOfSampleReturns := []types.Decimal{
		types.NewDecimalFromFloat(-0.02), // -2%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.06), // -6% (exception)
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(-0.01), // -1%
	}

	portfolioValue := types.NewDecimalFromFloat(1000000.0)
	confidence := types.NewDecimalFromFloat(95.0)

	// Calculate VaR using historical method
	varResult, err := calculator.CalculateHistoricalVaR(historicalReturns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("Failed to calculate VaR for validation: %v", err)
	}

	// Perform backtesting
	backtest, err := calculator.BacktestVaR(varResult, outOfSampleReturns, portfolioValue)
	if err != nil {
		t.Fatalf("BacktestVaR failed: %v", err)
	}

	// Verify backtesting results
	if backtest.TotalObservations != len(outOfSampleReturns) {
		t.Errorf("Expected %d observations, got %d", 
			len(outOfSampleReturns), backtest.TotalObservations)
	}

	if backtest.Exceptions < 0 {
		t.Error("Number of exceptions cannot be negative")
	}

	if backtest.ExceptionRate.IsNegative() {
		t.Error("Exception rate cannot be negative")
	}

	// Exception rate should be reasonable (typically < 25% for small samples)
	maxExpectedExceptionRate := types.NewDecimalFromFloat(25.0)
	if backtest.ExceptionRate.Cmp(maxExpectedExceptionRate) > 0 {
		t.Errorf("Exception rate %s%% seems unusually high", backtest.ExceptionRate.String())
	}

	// Should include model validation metrics
	if backtest.IsModelValid == nil {
		t.Error("Expected model validation result to be determined")
	}
}

func TestVaRComponentAnalysis(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewVaRCalculator()

	// Test component VaR calculation for portfolio positions
	positions := []PositionVaR{
		{
			AssetSymbol: "AAPL",
			Weight:      types.NewDecimalFromFloat(0.4), // 40%
			Returns: []types.Decimal{
				types.NewDecimalFromFloat(-0.02),
				types.NewDecimalFromFloat(0.03),
				types.NewDecimalFromFloat(-0.01),
			},
		},
		{
			AssetSymbol: "GOOGL", 
			Weight:      types.NewDecimalFromFloat(0.35), // 35%
			Returns: []types.Decimal{
				types.NewDecimalFromFloat(-0.03),
				types.NewDecimalFromFloat(0.02),
				types.NewDecimalFromFloat(-0.02),
			},
		},
		{
			AssetSymbol: "MSFT",
			Weight:      types.NewDecimalFromFloat(0.25), // 25%
			Returns: []types.Decimal{
				types.NewDecimalFromFloat(-0.01),
				types.NewDecimalFromFloat(0.04),
				types.NewDecimalFromFloat(-0.015),
			},
		},
	}

	portfolioValue := types.NewDecimalFromFloat(1500000.0)
	confidence := types.NewDecimalFromFloat(95.0)

	componentResult, err := calculator.CalculateComponentVaR(positions, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateComponentVaR failed: %v", err)
	}

	// Verify component analysis
	if len(componentResult.Components) != len(positions) {
		t.Errorf("Expected %d components, got %d", 
			len(positions), len(componentResult.Components))
	}

	// Total component VaR should sum approximately to portfolio VaR
	totalComponentVaR := types.Zero()
	for _, component := range componentResult.Components {
		if component.AssetSymbol == "" {
			t.Error("Component asset symbol should not be empty")
		}
		
		if component.ComponentVaR.IsZero() {
			t.Errorf("Component VaR for %s should not be zero", component.AssetSymbol)
		}
		
		totalComponentVaR = totalComponentVaR.Add(component.ComponentVaR.Abs())
	}

	// Component contributions should be meaningful
	portfolioVaR := componentResult.PortfolioVaR.Abs()
	difference := totalComponentVaR.Sub(portfolioVaR).Abs()
	tolerance := portfolioVaR.Mul(types.NewDecimalFromFloat(0.1)) // 10% tolerance

	if difference.Cmp(tolerance) > 0 {
		t.Errorf("Component VaR sum (%s) differs significantly from portfolio VaR (%s)",
			totalComponentVaR.String(), portfolioVaR.String())
	}
}

func TestVaRConfigurationAndValidation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase  
	calculator := NewVaRCalculator()

	// Test default configuration
	defaultConfig := calculator.GetConfig()
	if len(defaultConfig.SupportedMethods) == 0 {
		t.Error("Expected default supported methods to be configured")
	}

	// Test custom configuration
	customConfig := VaRConfig{
		DefaultMethod:           "Historical",
		DefaultConfidenceLevel:  types.NewDecimalFromFloat(99.0),
		MinHistoricalObservations: 250,
		SupportedMethods:        []string{"Historical", "Parametric", "MonteCarlo"},
		EnableBacktesting:       true,
	}

	err := calculator.SetConfig(customConfig)
	if err != nil {
		t.Fatalf("Failed to set custom config: %v", err)
	}

	// Verify configuration was applied
	appliedConfig := calculator.GetConfig()
	if appliedConfig.DefaultMethod != "Historical" {
		t.Errorf("Expected default method 'Historical', got %s", appliedConfig.DefaultMethod)
	}

	if appliedConfig.MinHistoricalObservations != 250 {
		t.Errorf("Expected min observations 250, got %d", appliedConfig.MinHistoricalObservations)
	}

	// Test validation with insufficient data
	insufficientReturns := []types.Decimal{
		types.NewDecimalFromFloat(0.01),
		types.NewDecimalFromFloat(-0.02),
		types.NewDecimalFromFloat(0.015),
	} // Only 3 observations

	portfolioValue := types.NewDecimalFromFloat(1000000.0)
	confidence := types.NewDecimalFromFloat(95.0)

	_, err = calculator.CalculateHistoricalVaR(insufficientReturns, portfolioValue, confidence)
	if err == nil {
		t.Error("Should reject historical VaR calculation with insufficient data")
	}

	// Test invalid confidence level
	invalidConfidence := types.NewDecimalFromFloat(105.0) // > 100%
	_, err = calculator.CalculateHistoricalVaR(insufficientReturns, portfolioValue, invalidConfidence)
	if err == nil {
		t.Error("Should reject invalid confidence level")
	}
}

// TDD RED phase - Types and functions will be implemented in var_calculator.go