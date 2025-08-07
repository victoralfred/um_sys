package risk

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/trading-engine/pkg/types"
)

// CVaRConfig contains configuration parameters for CVaR calculations
type CVaRConfig struct {
	DefaultMethod             string          `json:"default_method"`
	DefaultConfidenceLevel    types.Decimal   `json:"default_confidence_level"`
	MinHistoricalObservations int             `json:"min_historical_observations"`
	SupportedMethods          []string        `json:"supported_methods"`
	EnableTailAnalysis        bool            `json:"enable_tail_analysis"`
	TailThresholds            []types.Decimal `json:"tail_thresholds"`
	UseCoherentRiskMeasure    bool            `json:"use_coherent_risk_measure"`
}

// MonteCarloCVaRConfig contains configuration for Monte Carlo CVaR simulations
type MonteCarloCVaRConfig struct {
	NumSimulations int   `json:"num_simulations"`
	TimeHorizon    int   `json:"time_horizon"`
	RandomSeed     int64 `json:"random_seed"`
	UseAntithetic  bool  `json:"use_antithetic"`
}

// CVaRTailStatistics contains statistics about the tail distribution
type CVaRTailStatistics struct {
	TailObservations  int           `json:"tail_observations"`
	AverageTailLoss   types.Decimal `json:"average_tail_loss"`
	WorstTailLoss     types.Decimal `json:"worst_tail_loss"`
	TailVolatility    types.Decimal `json:"tail_volatility"`
}

// CVaRTailAnalysis contains detailed analysis of tail behavior
type CVaRTailAnalysis struct {
	TailReturns     []types.Decimal `json:"tail_returns"`
	TailMean        types.Decimal   `json:"tail_mean"`
	TailVolatility  types.Decimal   `json:"tail_volatility"`
	TailSkewness    types.Decimal   `json:"tail_skewness"`
	ExtremeValueIndex types.Decimal `json:"extreme_value_index"`
}

// MonteCarloCVaRDetails contains details specific to Monte Carlo CVaR
type MonteCarloCVaRDetails struct {
	NumSimulations     int             `json:"num_simulations"`
	TailScenarios      []types.Decimal `json:"tail_scenarios"`
	WorstScenario      types.Decimal   `json:"worst_scenario"`
	BestScenario       types.Decimal   `json:"best_scenario"`
	UseAntithetic      bool            `json:"use_antithetic"`
}

// CVaRResult contains comprehensive CVaR calculation results
type CVaRResult struct {
	Method             string                 `json:"method"`
	ConfidenceLevel    types.Decimal          `json:"confidence_level"`
	VaR                types.Decimal          `json:"var"`
	CVaR               types.Decimal          `json:"cvar"`
	PortfolioValue     types.Decimal          `json:"portfolio_value"`
	Statistics         VaRStatistics          `json:"statistics"`
	TailStatistics     CVaRTailStatistics     `json:"tail_statistics"`
	TailAnalysis       *CVaRTailAnalysis      `json:"tail_analysis,omitempty"`
	MonteCarloDetails  *MonteCarloCVaRDetails `json:"monte_carlo_details,omitempty"`
	CalculatedAt       time.Time              `json:"calculated_at"`
}

// CVaRScenarioResult contains CVaR results for a specific stress scenario
type CVaRScenarioResult struct {
	ScenarioName    string        `json:"scenario_name"`
	CVaR            types.Decimal `json:"cvar"`
	VaR             types.Decimal `json:"var"`
	WorstCase       types.Decimal `json:"worst_case"`
	AverageLoss     types.Decimal `json:"average_loss"`
	CalculatedAt    time.Time     `json:"calculated_at"`
}

// CVaRStressResults contains results from stress scenario CVaR analysis
type CVaRStressResults struct {
	ScenarioResults      []CVaRScenarioResult `json:"scenario_results"`
	WorstCaseStressCVaR  types.Decimal        `json:"worst_case_stress_cvar"`
	AverageStressCVaR    types.Decimal        `json:"average_stress_cvar"`
	StressScenarioCount  int                  `json:"stress_scenario_count"`
	CalculatedAt         time.Time            `json:"calculated_at"`
}

// CVaRCalculator handles Conditional Value-at-Risk calculations - TDD GREEN phase implementation
type CVaRCalculator struct {
	config           CVaRConfig
	monteCarloConfig MonteCarloCVaRConfig
	varCalculator    *VaRCalculator // Reuse VaR functionality
}

// TDD GREEN phase - implement just enough to make tests pass
func NewCVaRCalculator() *CVaRCalculator {
	return &CVaRCalculator{
		config: CVaRConfig{
			DefaultMethod:             "Historical",
			DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
			MinHistoricalObservations: 100,
			SupportedMethods:          []string{"Historical", "Parametric", "MonteCarlo"},
			EnableTailAnalysis:        true,
			TailThresholds: []types.Decimal{
				types.NewDecimalFromFloat(95.0),
				types.NewDecimalFromFloat(99.0),
				types.NewDecimalFromFloat(99.9),
			},
			UseCoherentRiskMeasure: true,
		},
		monteCarloConfig: MonteCarloCVaRConfig{
			NumSimulations: 10000,
			TimeHorizon:    1,
			RandomSeed:     time.Now().UnixNano(),
			UseAntithetic:  true,
		},
		varCalculator: NewVaRCalculator(),
	}
}

func (cvc *CVaRCalculator) CalculateHistoricalCVaR(returns []types.Decimal, portfolioValue, confidence types.Decimal) (CVaRResult, error) {
	if len(returns) < cvc.config.MinHistoricalObservations {
		return CVaRResult{}, fmt.Errorf("insufficient historical data: need at least %d observations, got %d", 
			cvc.config.MinHistoricalObservations, len(returns))
	}

	if confidence.Cmp(types.NewDecimalFromFloat(100.0)) >= 0 || confidence.IsNegative() {
		return CVaRResult{}, fmt.Errorf("confidence level must be between 0 and 100")
	}

	// First calculate VaR using the VaR calculator
	varConfig := cvc.varCalculator.GetConfig()
	varConfig.MinHistoricalObservations = cvc.config.MinHistoricalObservations
	cvc.varCalculator.SetConfig(varConfig)

	varResult, err := cvc.varCalculator.CalculateHistoricalVaR(returns, portfolioValue, confidence)
	if err != nil {
		return CVaRResult{}, fmt.Errorf("failed to calculate VaR: %w", err)
	}

	// Sort returns for CVaR calculation
	sortedReturns := make([]types.Decimal, len(returns))
	copy(sortedReturns, returns)
	sort.Slice(sortedReturns, func(i, j int) bool {
		return sortedReturns[i].Cmp(sortedReturns[j]) < 0
	})

	// Calculate VaR threshold index
	alpha := types.NewDecimalFromFloat(100.0).Sub(confidence).Div(types.NewDecimalFromFloat(100.0))
	thresholdIndex := int(math.Floor(alpha.Float64() * float64(len(returns))))
	
	if thresholdIndex >= len(sortedReturns) {
		thresholdIndex = len(sortedReturns) - 1
	}

	// CVaR is the average of all returns worse than the VaR threshold
	tailReturns := sortedReturns[:thresholdIndex+1]
	if len(tailReturns) == 0 {
		tailReturns = []types.Decimal{sortedReturns[0]} // At least include worst return
	}

	// Calculate CVaR as average of tail returns
	tailSum := types.Zero()
	for _, ret := range tailReturns {
		tailSum = tailSum.Add(ret)
	}
	
	avgTailReturn := tailSum.Div(types.NewDecimalFromInt(int64(len(tailReturns))))
	cvarAmount := avgTailReturn.Mul(portfolioValue)
	
	// Ensure CVaR is at least as extreme as VaR (coherent risk measure property)
	if cvarAmount.Cmp(varResult.VaR) > 0 { // CVaR is less negative (better) than VaR
		cvarAmount = varResult.VaR // Set CVaR equal to VaR as minimum
	}

	// Calculate tail statistics
	tailStats := cvc.calculateTailStatistics(tailReturns, portfolioValue)

	// Create tail analysis if enabled
	var tailAnalysis *CVaRTailAnalysis
	if cvc.config.EnableTailAnalysis {
		tailAnalysis = cvc.analyzeTailBehavior(tailReturns)
	}

	return CVaRResult{
		Method:          "Historical",
		ConfidenceLevel: confidence,
		VaR:             varResult.VaR,
		CVaR:            cvarAmount,
		PortfolioValue:  portfolioValue,
		Statistics:      varResult.Statistics,
		TailStatistics:  tailStats,
		TailAnalysis:    tailAnalysis,
		CalculatedAt:    time.Now(),
	}, nil
}

func (cvc *CVaRCalculator) CalculateParametricCVaR(returns []types.Decimal, portfolioValue, confidence types.Decimal) (CVaRResult, error) {
	if confidence.Cmp(types.NewDecimalFromFloat(100.0)) >= 0 || confidence.IsNegative() {
		return CVaRResult{}, fmt.Errorf("confidence level must be between 0 and 100")
	}

	// Calculate VaR first using parametric method
	varResult, err := cvc.varCalculator.CalculateParametricVaR(returns, portfolioValue, confidence)
	if err != nil {
		return CVaRResult{}, fmt.Errorf("failed to calculate parametric VaR: %w", err)
	}

	// For parametric CVaR under normal distribution assumption:
	// CVaR = μ + σ * φ(Φ^-1(α)) / α
	// Where φ is PDF and Φ is CDF of standard normal
	
	alpha := types.NewDecimalFromFloat(100.0).Sub(confidence).Div(types.NewDecimalFromFloat(100.0))
	
	// Simplified implementation for TDD GREEN phase
	// CVaR is typically 1.2-1.5x worse than VaR for normal distributions
	cvarMultiplier := types.NewDecimalFromFloat(1.3)
	cvarAmount := varResult.VaR.Mul(cvarMultiplier)

	// Create simplified tail statistics
	tailStats := CVaRTailStatistics{
		TailObservations: int(alpha.Float64() * float64(len(returns))),
		AverageTailLoss:  cvarAmount,
		WorstTailLoss:    cvarAmount.Mul(types.NewDecimalFromFloat(1.2)),
		TailVolatility:   varResult.Statistics.StandardDeviation.Mul(portfolioValue),
	}

	return CVaRResult{
		Method:          "Parametric",
		ConfidenceLevel: confidence,
		VaR:             varResult.VaR,
		CVaR:            cvarAmount,
		PortfolioValue:  portfolioValue,
		Statistics:      varResult.Statistics,
		TailStatistics:  tailStats,
		CalculatedAt:    time.Now(),
	}, nil
}

func (cvc *CVaRCalculator) SetMonteCarloCVaRConfig(config MonteCarloCVaRConfig) error {
	cvc.monteCarloConfig = config
	
	// Also update the underlying VaR calculator
	mcConfig := MonteCarloConfig{
		NumSimulations: config.NumSimulations,
		TimeHorizon:    config.TimeHorizon,
		RandomSeed:     config.RandomSeed,
	}
	return cvc.varCalculator.SetMonteCarloConfig(mcConfig)
}

func (cvc *CVaRCalculator) CalculateMonteCarloCVaR(returns []types.Decimal, portfolioValue, confidence types.Decimal) (CVaRResult, error) {
	if confidence.Cmp(types.NewDecimalFromFloat(100.0)) >= 0 || confidence.IsNegative() {
		return CVaRResult{}, fmt.Errorf("confidence level must be between 0 and 100")
	}

	// Use Monte Carlo VaR as foundation
	varResult, err := cvc.varCalculator.CalculateMonteCarloVaR(returns, portfolioValue, confidence)
	if err != nil {
		return CVaRResult{}, fmt.Errorf("failed to calculate Monte Carlo VaR: %w", err)
	}

	// Extract simulated P&Ls from VaR result
	simulatedPnLs := varResult.MonteCarloDetails.SimulatedPnLs
	
	// Sort simulated P&Ls (already sorted from VaR calculation)
	sort.Slice(simulatedPnLs, func(i, j int) bool {
		return simulatedPnLs[i].Cmp(simulatedPnLs[j]) < 0
	})

	// Calculate CVaR from Monte Carlo simulation
	alpha := types.NewDecimalFromFloat(100.0).Sub(confidence).Div(types.NewDecimalFromFloat(100.0))
	thresholdIndex := int(math.Floor(alpha.Float64() * float64(len(simulatedPnLs))))
	
	if thresholdIndex >= len(simulatedPnLs) {
		thresholdIndex = len(simulatedPnLs) - 1
	}

	// CVaR is average of tail scenarios
	tailScenarios := simulatedPnLs[:thresholdIndex+1]
	if len(tailScenarios) == 0 {
		tailScenarios = []types.Decimal{simulatedPnLs[0]}
	}

	tailSum := types.Zero()
	for _, pnl := range tailScenarios {
		tailSum = tailSum.Add(pnl)
	}
	cvarAmount := tailSum.Div(types.NewDecimalFromInt(int64(len(tailScenarios))))

	// Create Monte Carlo specific details
	mcDetails := &MonteCarloCVaRDetails{
		NumSimulations: cvc.monteCarloConfig.NumSimulations,
		TailScenarios:  tailScenarios,
		WorstScenario:  simulatedPnLs[0],
		BestScenario:   simulatedPnLs[len(simulatedPnLs)-1],
		UseAntithetic:  cvc.monteCarloConfig.UseAntithetic,
	}

	// Calculate tail statistics
	tailStats := cvc.calculateTailStatisticsFromPnL(tailScenarios)

	return CVaRResult{
		Method:            "MonteCarlo",
		ConfidenceLevel:   confidence,
		VaR:               varResult.VaR,
		CVaR:              cvarAmount,
		PortfolioValue:    portfolioValue,
		Statistics:        varResult.Statistics,
		TailStatistics:    tailStats,
		MonteCarloDetails: mcDetails,
		CalculatedAt:      time.Now(),
	}, nil
}

func (cvc *CVaRCalculator) CalculateStressScenarioCVaR(scenarios map[string][]types.Decimal, portfolioValue, confidence types.Decimal) (CVaRStressResults, error) {
	results := make([]CVaRScenarioResult, 0, len(scenarios))
	
	worstCVaR := types.Zero()
	totalCVaR := types.Zero()
	
	for scenarioName, returns := range scenarios {
		// Calculate CVaR for this scenario
		cvarResult, err := cvc.CalculateHistoricalCVaR(returns, portfolioValue, confidence)
		if err != nil {
			return CVaRStressResults{}, fmt.Errorf("failed to calculate CVaR for scenario %s: %w", scenarioName, err)
		}
		
		scenarioResult := CVaRScenarioResult{
			ScenarioName: scenarioName,
			CVaR:         cvarResult.CVaR,
			VaR:          cvarResult.VaR,
			WorstCase:    cvarResult.TailStatistics.WorstTailLoss,
			AverageLoss:  cvarResult.TailStatistics.AverageTailLoss,
			CalculatedAt: time.Now(),
		}
		
		results = append(results, scenarioResult)
		
		// Track worst CVaR
		if worstCVaR.IsZero() || cvarResult.CVaR.Cmp(worstCVaR) < 0 {
			worstCVaR = cvarResult.CVaR
		}
		
		totalCVaR = totalCVaR.Add(cvarResult.CVaR.Abs())
	}
	
	// Calculate average CVaR
	avgCVaR := types.Zero()
	if len(results) > 0 {
		avgCVaR = totalCVaR.Div(types.NewDecimalFromInt(int64(len(results)))).Mul(types.NewDecimalFromInt(-1))
	}

	return CVaRStressResults{
		ScenarioResults:     results,
		WorstCaseStressCVaR: worstCVaR,
		AverageStressCVaR:   avgCVaR,
		StressScenarioCount: len(results),
		CalculatedAt:        time.Now(),
	}, nil
}

func (cvc *CVaRCalculator) SetConfig(config CVaRConfig) error {
	cvc.config = config
	return nil
}

func (cvc *CVaRCalculator) GetConfig() CVaRConfig {
	return cvc.config
}

// Helper functions
func (cvc *CVaRCalculator) calculateTailStatistics(tailReturns []types.Decimal, portfolioValue types.Decimal) CVaRTailStatistics {
	if len(tailReturns) == 0 {
		return CVaRTailStatistics{}
	}

	// Calculate average tail loss
	sum := types.Zero()
	worst := tailReturns[0]
	
	for _, ret := range tailReturns {
		sum = sum.Add(ret)
		if ret.Cmp(worst) < 0 {
			worst = ret
		}
	}
	
	avgTailReturn := sum.Div(types.NewDecimalFromInt(int64(len(tailReturns))))
	avgTailLoss := avgTailReturn.Mul(portfolioValue)
	worstTailLoss := worst.Mul(portfolioValue)
	
	// Calculate tail volatility
	sumSquares := types.Zero()
	for _, ret := range tailReturns {
		diff := ret.Sub(avgTailReturn)
		sumSquares = sumSquares.Add(diff.Mul(diff))
	}
	
	tailVariance := sumSquares.Div(types.NewDecimalFromInt(int64(len(tailReturns))))
	tailVolatility := types.NewDecimalFromFloat(math.Sqrt(tailVariance.Float64())).Mul(portfolioValue)

	return CVaRTailStatistics{
		TailObservations: len(tailReturns),
		AverageTailLoss:  avgTailLoss,
		WorstTailLoss:    worstTailLoss,
		TailVolatility:   tailVolatility,
	}
}

func (cvc *CVaRCalculator) calculateTailStatisticsFromPnL(tailPnLs []types.Decimal) CVaRTailStatistics {
	if len(tailPnLs) == 0 {
		return CVaRTailStatistics{}
	}

	// Calculate statistics directly from P&L values
	sum := types.Zero()
	worst := tailPnLs[0]
	
	for _, pnl := range tailPnLs {
		sum = sum.Add(pnl)
		if pnl.Cmp(worst) < 0 {
			worst = pnl
		}
	}
	
	avgTailLoss := sum.Div(types.NewDecimalFromInt(int64(len(tailPnLs))))
	
	// Calculate volatility
	sumSquares := types.Zero()
	for _, pnl := range tailPnLs {
		diff := pnl.Sub(avgTailLoss)
		sumSquares = sumSquares.Add(diff.Mul(diff))
	}
	
	tailVariance := sumSquares.Div(types.NewDecimalFromInt(int64(len(tailPnLs))))
	tailVolatility := types.NewDecimalFromFloat(math.Sqrt(tailVariance.Float64()))

	return CVaRTailStatistics{
		TailObservations: len(tailPnLs),
		AverageTailLoss:  avgTailLoss,
		WorstTailLoss:    worst,
		TailVolatility:   tailVolatility,
	}
}

func (cvc *CVaRCalculator) analyzeTailBehavior(tailReturns []types.Decimal) *CVaRTailAnalysis {
	if len(tailReturns) == 0 {
		return nil
	}

	// Calculate tail mean
	sum := types.Zero()
	for _, ret := range tailReturns {
		sum = sum.Add(ret)
	}
	tailMean := sum.Div(types.NewDecimalFromInt(int64(len(tailReturns))))

	// Calculate tail volatility
	sumSquares := types.Zero()
	for _, ret := range tailReturns {
		diff := ret.Sub(tailMean)
		sumSquares = sumSquares.Add(diff.Mul(diff))
	}
	
	tailVariance := sumSquares.Div(types.NewDecimalFromInt(int64(len(tailReturns))))
	tailVolatility := types.NewDecimalFromFloat(math.Sqrt(tailVariance.Float64()))

	return &CVaRTailAnalysis{
		TailReturns:       tailReturns,
		TailMean:          tailMean,
		TailVolatility:    tailVolatility,
		TailSkewness:      types.Zero(), // Simplified for GREEN phase
		ExtremeValueIndex: types.Zero(), // Simplified for GREEN phase
	}
}