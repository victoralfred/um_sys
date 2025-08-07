package risk

import (
	"testing"

	"github.com/trading-engine/pkg/types"
)

// TDD: Step 1 - RED - Write failing tests for CVaR (Conditional Value-at-Risk) Calculator

func TestHistoricalCVaRCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewCVaRCalculator()
	if calculator == nil {
		t.Fatal("Expected CVaR calculator to be created")
	}

	// Reduce minimum observations for testing
	config := calculator.GetConfig()
	config.MinHistoricalObservations = 5
	calculator.SetConfig(config)

	// Test historical CVaR calculation with sample returns
	returns := []types.Decimal{
		types.NewDecimalFromFloat(-0.10), // -10% (worst)
		types.NewDecimalFromFloat(-0.05), // -5%
		types.NewDecimalFromFloat(-0.02), // -2%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(-0.08), // -8%
		types.NewDecimalFromFloat(0.04),  // 4%
		types.NewDecimalFromFloat(-0.03), // -3%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(-0.06), // -6%
	}

	portfolioValue := types.NewDecimalFromFloat(1000000.0) // $1M portfolio
	confidence := types.NewDecimalFromFloat(95.0)           // 95% confidence level

	cvarResult, err := calculator.CalculateHistoricalCVaR(returns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateHistoricalCVaR failed: %v", err)
	}

	// Verify CVaR result structure
	if cvarResult.Method != "Historical" {
		t.Errorf("Expected method 'Historical', got %s", cvarResult.Method)
	}

	if cvarResult.ConfidenceLevel.Cmp(confidence) != 0 {
		t.Errorf("Expected confidence level %s, got %s", 
			confidence.String(), cvarResult.ConfidenceLevel.String())
	}

	// CVaR should be more negative than VaR (representing worse expected loss)
	if cvarResult.CVaR.IsZero() {
		t.Error("Expected non-zero CVaR value")
	}

	if cvarResult.CVaR.IsPositive() {
		t.Error("CVaR should be negative (representing potential loss)")
	}

	// CVaR should be at least as bad as VaR
	if cvarResult.VaR.IsZero() {
		t.Error("Expected VaR to be calculated alongside CVaR")
	}

	if cvarResult.CVaR.Cmp(cvarResult.VaR) > 0 {
		t.Errorf("CVaR (%s) should be more negative than or equal to VaR (%s)",
			cvarResult.CVaR.String(), cvarResult.VaR.String())
	}

	// Verify tail loss statistics
	if cvarResult.TailStatistics.TailObservations <= 0 {
		t.Error("Expected positive number of tail observations")
	}

	if cvarResult.TailStatistics.AverageTailLoss.IsZero() {
		t.Error("Expected non-zero average tail loss")
	}

	if cvarResult.TailStatistics.WorstTailLoss.IsZero() {
		t.Error("Expected worst tail loss to be recorded")
	}
}

func TestParametricCVaRCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewCVaRCalculator()

	// Test parametric CVaR calculation using normal distribution assumption
	returns := []types.Decimal{
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(-0.01), // -1%
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(-0.02), // -2%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.03), // -3%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(-0.01), // -1%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(-0.02), // -2%
	}

	portfolioValue := types.NewDecimalFromFloat(2000000.0) // $2M portfolio
	confidence := types.NewDecimalFromFloat(99.0)          // 99% confidence level

	cvarResult, err := calculator.CalculateParametricCVaR(returns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateParametricCVaR failed: %v", err)
	}

	// Verify parametric CVaR result
	if cvarResult.Method != "Parametric" {
		t.Errorf("Expected method 'Parametric', got %s", cvarResult.Method)
	}

	if cvarResult.ConfidenceLevel.Cmp(confidence) != 0 {
		t.Errorf("Expected confidence level %s, got %s", 
			confidence.String(), cvarResult.ConfidenceLevel.String())
	}

	if cvarResult.CVaR.IsZero() {
		t.Error("Expected non-zero parametric CVaR value")
	}

	// Should include statistical parameters
	if cvarResult.Statistics.Mean.IsZero() && cvarResult.Statistics.StandardDeviation.IsZero() {
		t.Error("Expected statistical measures to be calculated")
	}

	// Higher confidence level should result in higher CVaR (more negative)
	confidence95 := types.NewDecimalFromFloat(95.0)
	cvarResult95, err := calculator.CalculateParametricCVaR(returns, portfolioValue, confidence95)
	if err != nil {
		t.Fatalf("CalculateParametricCVaR failed for 95%%: %v", err)
	}

	// 99% CVaR should be more negative than 95% CVaR
	if cvarResult.CVaR.Cmp(cvarResult95.CVaR) > 0 {
		t.Errorf("99%% CVaR (%s) should be more negative than 95%% CVaR (%s)",
			cvarResult.CVaR.String(), cvarResult95.CVaR.String())
	}
}

func TestMonteCarloCVaRCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewCVaRCalculator()

	// Configure Monte Carlo parameters for CVaR
	config := MonteCarloCVaRConfig{
		NumSimulations: 10000,
		TimeHorizon:    1, // 1 day
		RandomSeed:     67890,
		UseAntithetic:  true, // Use antithetic variance reduction
	}

	err := calculator.SetMonteCarloCVaRConfig(config)
	if err != nil {
		t.Fatalf("Failed to set Monte Carlo CVaR config: %v", err)
	}

	// Sample historical data for Monte Carlo CVaR
	returns := []types.Decimal{
		types.NewDecimalFromFloat(0.020),  // 2.0%
		types.NewDecimalFromFloat(-0.035), // -3.5%
		types.NewDecimalFromFloat(0.015),  // 1.5%
		types.NewDecimalFromFloat(-0.025), // -2.5%
		types.NewDecimalFromFloat(0.030),  // 3.0%
		types.NewDecimalFromFloat(-0.040), // -4.0%
		types.NewDecimalFromFloat(0.010),  // 1.0%
		types.NewDecimalFromFloat(-0.015), // -1.5%
		types.NewDecimalFromFloat(0.025),  // 2.5%
		types.NewDecimalFromFloat(-0.030), // -3.0%
	}

	portfolioValue := types.NewDecimalFromFloat(5000000.0) // $5M portfolio
	confidence := types.NewDecimalFromFloat(95.0)

	cvarResult, err := calculator.CalculateMonteCarloCVaR(returns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateMonteCarloCVaR failed: %v", err)
	}

	// Verify Monte Carlo CVaR result
	if cvarResult.Method != "MonteCarlo" {
		t.Errorf("Expected method 'MonteCarlo', got %s", cvarResult.Method)
	}

	if cvarResult.MonteCarloDetails.NumSimulations != config.NumSimulations {
		t.Errorf("Expected %d simulations, got %d", 
			config.NumSimulations, cvarResult.MonteCarloDetails.NumSimulations)
	}

	if cvarResult.CVaR.IsZero() {
		t.Error("Expected non-zero Monte Carlo CVaR value")
	}

	// Monte Carlo should provide detailed simulation statistics
	if cvarResult.MonteCarloDetails.TailScenarios == nil || len(cvarResult.MonteCarloDetails.TailScenarios) == 0 {
		t.Error("Expected tail scenarios to be captured")
	}

	if cvarResult.MonteCarloDetails.WorstScenario.IsZero() {
		t.Error("Expected worst scenario to be recorded")
	}

	// CVaR should be reasonable relative to worst scenario
	if cvarResult.CVaR.Cmp(cvarResult.MonteCarloDetails.WorstScenario) < 0 {
		t.Errorf("CVaR (%s) should not be worse than worst scenario (%s)",
			cvarResult.CVaR.String(), cvarResult.MonteCarloDetails.WorstScenario.String())
	}
}

func TestCVaRVsVaRComparison(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewCVaRCalculator()
	
	// Reduce minimum observations for testing
	config := calculator.GetConfig()
	config.MinHistoricalObservations = 5
	calculator.SetConfig(config)

	// Test with asymmetric returns (more extreme negative returns)
	returns := []types.Decimal{
		types.NewDecimalFromFloat(-0.15), // -15% (extreme loss)
		types.NewDecimalFromFloat(-0.10), // -10% (severe loss)
		types.NewDecimalFromFloat(-0.05), // -5%
		types.NewDecimalFromFloat(-0.02), // -2%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(0.04),  // 4%
		types.NewDecimalFromFloat(0.05),  // 5%
		types.NewDecimalFromFloat(0.06),  // 6%
	}

	portfolioValue := types.NewDecimalFromFloat(1000000.0)
	confidence := types.NewDecimalFromFloat(90.0) // 90% confidence

	cvarResult, err := calculator.CalculateHistoricalCVaR(returns, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("Failed to calculate CVaR: %v", err)
	}

	// CVaR should capture tail risk better than VaR
	varCVaRRatio := cvarResult.CVaR.Div(cvarResult.VaR).Abs()
	expectedMinRatio := types.NewDecimalFromFloat(1.0) // CVaR should be at least as bad as VaR

	if varCVaRRatio.Cmp(expectedMinRatio) < 0 {
		t.Errorf("CVaR should be at least as extreme as VaR, ratio: %s", varCVaRRatio.String())
	}

	// For asymmetric distributions, CVaR should be notably worse than VaR
	// This captures the "tail risk" beyond the VaR threshold
	if cvarResult.TailStatistics.TailObservations > 1 {
		expectedMaxRatio := types.NewDecimalFromFloat(3.0) // CVaR shouldn't be more than 3x VaR
		if varCVaRRatio.Cmp(expectedMaxRatio) > 0 {
			t.Errorf("CVaR seems excessively worse than VaR, ratio: %s", varCVaRRatio.String())
		}
	}
}

func TestCVaRTailRiskAnalysis(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewCVaRCalculator()

	// Configure for detailed tail risk analysis
	config := calculator.GetConfig()
	config.MinHistoricalObservations = 5
	config.EnableTailAnalysis = true
	config.TailThresholds = []types.Decimal{
		types.NewDecimalFromFloat(95.0), // 95% threshold
		types.NewDecimalFromFloat(99.0), // 99% threshold
		types.NewDecimalFromFloat(99.9), // 99.9% threshold
	}
	calculator.SetConfig(config)

	// Create data with clear tail structure
	returns := []types.Decimal{
		types.NewDecimalFromFloat(-0.20), // -20% (extreme tail)
		types.NewDecimalFromFloat(-0.12), // -12% (severe tail)
		types.NewDecimalFromFloat(-0.08), // -8% (moderate tail)
		types.NewDecimalFromFloat(-0.05), // -5%
		types.NewDecimalFromFloat(-0.03), // -3%
		types.NewDecimalFromFloat(-0.01), // -1%
		types.NewDecimalFromFloat(0.01),  // 1%
		types.NewDecimalFromFloat(0.02),  // 2%
		types.NewDecimalFromFloat(0.03),  // 3%
		types.NewDecimalFromFloat(0.04),  // 4%
		types.NewDecimalFromFloat(0.05),  // 5%
		types.NewDecimalFromFloat(0.06),  // 6%
	}

	portfolioValue := types.NewDecimalFromFloat(1000000.0)

	// Test multiple confidence levels for tail analysis
	for _, confidence := range []float64{95.0, 99.0, 99.5} {
		confLevel := types.NewDecimalFromFloat(confidence)
		
		cvarResult, err := calculator.CalculateHistoricalCVaR(returns, portfolioValue, confLevel)
		if err != nil {
			t.Fatalf("Failed to calculate CVaR at %s%% confidence: %v", confLevel.String(), err)
		}

		// Verify tail analysis results
		tailAnalysis := cvarResult.TailAnalysis
		if tailAnalysis == nil {
			t.Errorf("Expected tail analysis at %s%% confidence", confLevel.String())
			continue
		}

		if len(tailAnalysis.TailReturns) == 0 {
			t.Errorf("Expected tail returns to be captured at %s%% confidence", confLevel.String())
		}

		// Higher confidence levels should capture fewer but more extreme observations
		expectedTailObs := int((100.0 - confidence) / 100.0 * float64(len(returns)))
		actualTailObs := len(tailAnalysis.TailReturns)
		
		// Allow some tolerance in the expected count
		tolerance := 2
		if actualTailObs < expectedTailObs-tolerance || actualTailObs > expectedTailObs+tolerance {
			t.Logf("At %s%% confidence: expected ~%d tail observations, got %d", 
				confLevel.String(), expectedTailObs, actualTailObs)
		}

		// Verify tail characteristics
		if tailAnalysis.TailMean.IsPositive() {
			t.Errorf("Tail mean should be negative at %s%% confidence", confLevel.String())
		}

		if tailAnalysis.TailVolatility.IsNegative() {
			t.Errorf("Tail volatility should be positive at %s%% confidence", confLevel.String())
		}
	}
}

func TestCVaRStressScenarios(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewCVaRCalculator()

	// Test CVaR under different stress scenarios
	scenarios := map[string][]types.Decimal{
		"NormalMarket": {
			types.NewDecimalFromFloat(0.02),
			types.NewDecimalFromFloat(-0.01),
			types.NewDecimalFromFloat(0.015),
			types.NewDecimalFromFloat(-0.005),
			types.NewDecimalFromFloat(0.01),
		},
		"VolatileMarket": {
			types.NewDecimalFromFloat(0.05),
			types.NewDecimalFromFloat(-0.04),
			types.NewDecimalFromFloat(0.03),
			types.NewDecimalFromFloat(-0.06),
			types.NewDecimalFromFloat(0.02),
		},
		"CrisisMarket": {
			types.NewDecimalFromFloat(-0.15),
			types.NewDecimalFromFloat(-0.08),
			types.NewDecimalFromFloat(-0.12),
			types.NewDecimalFromFloat(-0.05),
			types.NewDecimalFromFloat(-0.20),
		},
	}

	portfolioValue := types.NewDecimalFromFloat(1000000.0)
	confidence := types.NewDecimalFromFloat(95.0)

	// Reduce minimum observations for testing
	config := calculator.GetConfig()
	config.MinHistoricalObservations = 3
	calculator.SetConfig(config)

	stressResults, err := calculator.CalculateStressScenarioCVaR(scenarios, portfolioValue, confidence)
	if err != nil {
		t.Fatalf("CalculateStressScenarioCVaR failed: %v", err)
	}

	// Verify stress scenario results
	if len(stressResults.ScenarioResults) != len(scenarios) {
		t.Errorf("Expected %d scenario results, got %d", 
			len(scenarios), len(stressResults.ScenarioResults))
	}

	// Crisis scenario should have worst CVaR
	var normalCVaR, crisisCVaR types.Decimal
	found := make(map[string]bool)

	for _, result := range stressResults.ScenarioResults {
		found[result.ScenarioName] = true
		
		if result.ScenarioName == "NormalMarket" {
			normalCVaR = result.CVaR
		} else if result.ScenarioName == "CrisisMarket" {
			crisisCVaR = result.CVaR
		}
		
		if result.CVaR.IsZero() {
			t.Errorf("Scenario %s should have non-zero CVaR", result.ScenarioName)
		}
	}

	// Verify all scenarios were processed
	for scenario := range scenarios {
		if !found[scenario] {
			t.Errorf("Missing result for scenario: %s", scenario)
		}
	}

	// Crisis CVaR should be worse (more negative) than normal CVaR
	if !normalCVaR.IsZero() && !crisisCVaR.IsZero() {
		if crisisCVaR.Cmp(normalCVaR) > 0 {
			t.Errorf("Crisis CVaR (%s) should be worse than normal CVaR (%s)",
				crisisCVaR.String(), normalCVaR.String())
		}
	}

	// Overall stress CVaR should be reported
	if stressResults.WorstCaseStressCVaR.IsZero() {
		t.Error("Expected worst-case stress CVaR to be calculated")
	}
}

func TestCVaRConfigurationAndValidation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	calculator := NewCVaRCalculator()

	// Test default configuration
	defaultConfig := calculator.GetConfig()
	if defaultConfig.DefaultMethod == "" {
		t.Error("Expected default method to be configured")
	}

	if len(defaultConfig.SupportedMethods) == 0 {
		t.Error("Expected supported methods to be configured")
	}

	// Test custom configuration
	customConfig := CVaRConfig{
		DefaultMethod:             "Historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(99.0),
		MinHistoricalObservations: 100,
		SupportedMethods:          []string{"Historical", "Parametric", "MonteCarlo"},
		EnableTailAnalysis:        true,
		TailThresholds: []types.Decimal{
			types.NewDecimalFromFloat(95.0),
			types.NewDecimalFromFloat(99.0),
		},
		UseCoherentRiskMeasure: true,
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

	if appliedConfig.MinHistoricalObservations != 100 {
		t.Errorf("Expected min observations 100, got %d", appliedConfig.MinHistoricalObservations)
	}

	if !appliedConfig.UseCoherentRiskMeasure {
		t.Error("Expected coherent risk measure to be enabled")
	}

	// Test validation with invalid confidence level
	insufficientReturns := []types.Decimal{
		types.NewDecimalFromFloat(0.01),
		types.NewDecimalFromFloat(-0.02),
	}

	portfolioValue := types.NewDecimalFromFloat(1000000.0)
	invalidConfidence := types.NewDecimalFromFloat(101.0) // > 100%

	_, err = calculator.CalculateHistoricalCVaR(insufficientReturns, portfolioValue, invalidConfidence)
	if err == nil {
		t.Error("Should reject invalid confidence level")
	}

	// Test validation with insufficient data
	validConfidence := types.NewDecimalFromFloat(95.0)
	_, err = calculator.CalculateHistoricalCVaR(insufficientReturns, portfolioValue, validConfidence)
	if err == nil {
		t.Error("Should reject CVaR calculation with insufficient data")
	}
}

// TDD RED phase - Types and functions will be implemented in cvar_calculator.go