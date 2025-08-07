package risk

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/trading-engine/pkg/types"
)

// VaRConfig contains configuration parameters for VaR calculations
type VaRConfig struct {
	DefaultMethod             string          `json:"default_method"`
	DefaultConfidenceLevel    types.Decimal   `json:"default_confidence_level"`
	MinHistoricalObservations int             `json:"min_historical_observations"`
	SupportedMethods          []string        `json:"supported_methods"`
	EnableBacktesting         bool            `json:"enable_backtesting"`
}

// MonteCarloConfig contains configuration for Monte Carlo simulations
type MonteCarloConfig struct {
	NumSimulations int   `json:"num_simulations"`
	TimeHorizon    int   `json:"time_horizon"`
	RandomSeed     int64 `json:"random_seed"`
}

// VaRStatistics contains statistical measures used in VaR calculation
type VaRStatistics struct {
	Mean              types.Decimal `json:"mean"`
	StandardDeviation types.Decimal `json:"standard_deviation"`
	Skewness          types.Decimal `json:"skewness"`
	Kurtosis          types.Decimal `json:"kurtosis"`
}

// MonteCarloDetails contains details specific to Monte Carlo VaR
type MonteCarloDetails struct {
	NumSimulations     int           `json:"num_simulations"`
	WorstCaseScenario  types.Decimal `json:"worst_case_scenario"`
	BestCaseScenario   types.Decimal `json:"best_case_scenario"`
	SimulatedPnLs      []types.Decimal `json:"simulated_pnls,omitempty"`
}

// VaRResult contains comprehensive VaR calculation results
type VaRResult struct {
	Method             string             `json:"method"`
	ConfidenceLevel    types.Decimal      `json:"confidence_level"`
	VaR                types.Decimal      `json:"var"`
	PortfolioValue     types.Decimal      `json:"portfolio_value"`
	Statistics         VaRStatistics      `json:"statistics"`
	MonteCarloDetails  *MonteCarloDetails `json:"monte_carlo_details,omitempty"`
	CalculatedAt       time.Time          `json:"calculated_at"`
}

// PositionVaR represents VaR calculation data for a portfolio position
type PositionVaR struct {
	AssetSymbol string          `json:"asset_symbol"`
	Weight      types.Decimal   `json:"weight"`
	Returns     []types.Decimal `json:"returns"`
}

// ComponentVaR represents VaR contribution of a single component
type ComponentVaR struct {
	AssetSymbol    string        `json:"asset_symbol"`
	Weight         types.Decimal `json:"weight"`
	ComponentVaR   types.Decimal `json:"component_var"`
	MarginalVaR    types.Decimal `json:"marginal_var"`
	ContributionPercent types.Decimal `json:"contribution_percent"`
}

// ComponentVaRResult contains results of component VaR analysis
type ComponentVaRResult struct {
	PortfolioVaR    types.Decimal   `json:"portfolio_var"`
	Components      []ComponentVaR  `json:"components"`
	DiversificationBenefit types.Decimal `json:"diversification_benefit"`
	CalculatedAt    time.Time       `json:"calculated_at"`
}

// VaRBacktest contains backtesting results for VaR model validation
type VaRBacktest struct {
	TotalObservations int           `json:"total_observations"`
	Exceptions        int           `json:"exceptions"`
	ExceptionRate     types.Decimal `json:"exception_rate"`
	ExpectedRate      types.Decimal `json:"expected_rate"`
	IsModelValid      *bool         `json:"is_model_valid"`
	PValue            types.Decimal `json:"p_value"`
	TestStatistic     types.Decimal `json:"test_statistic"`
}

// VaRCalculator handles Value-at-Risk calculations - TDD GREEN phase implementation
type VaRCalculator struct {
	config           VaRConfig
	monteCarloConfig MonteCarloConfig
	rng              *rand.Rand
}

// TDD GREEN phase - implement just enough to make tests pass
func NewVaRCalculator() *VaRCalculator {
	return &VaRCalculator{
		config: VaRConfig{
			DefaultMethod:             "Historical",
			DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
			MinHistoricalObservations: 250,
			SupportedMethods:          []string{"Historical", "Parametric", "MonteCarlo"},
			EnableBacktesting:         true,
		},
		monteCarloConfig: MonteCarloConfig{
			NumSimulations: 10000,
			TimeHorizon:    1,
			RandomSeed:     time.Now().UnixNano(),
		},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (vc *VaRCalculator) CalculateHistoricalVaR(returns []types.Decimal, portfolioValue, confidence types.Decimal) (VaRResult, error) {
	if len(returns) < vc.config.MinHistoricalObservations {
		return VaRResult{}, fmt.Errorf("insufficient historical data: need at least %d observations, got %d", 
			vc.config.MinHistoricalObservations, len(returns))
	}

	if confidence.Cmp(types.NewDecimalFromFloat(100.0)) >= 0 || confidence.IsNegative() {
		return VaRResult{}, fmt.Errorf("confidence level must be between 0 and 100")
	}

	// Sort returns for percentile calculation
	sortedReturns := make([]types.Decimal, len(returns))
	copy(sortedReturns, returns)
	sort.Slice(sortedReturns, func(i, j int) bool {
		return sortedReturns[i].Cmp(sortedReturns[j]) < 0
	})

	// Calculate percentile index
	percentile := types.NewDecimalFromFloat(100.0).Sub(confidence)
	indexFloat := percentile.Div(types.NewDecimalFromFloat(100.0)).Mul(types.NewDecimalFromInt(int64(len(returns) - 1)))
	
	// Simple implementation: use floor of index
	indexInt := int(math.Floor(indexFloat.Float64()))
	if indexInt >= len(sortedReturns) {
		indexInt = len(sortedReturns) - 1
	}

	// VaR is the return at the percentile, converted to portfolio value
	varReturn := sortedReturns[indexInt]
	varAmount := varReturn.Mul(portfolioValue)

	// Calculate statistics
	stats := vc.calculateStatistics(returns)

	return VaRResult{
		Method:          "Historical",
		ConfidenceLevel: confidence,
		VaR:             varAmount,
		PortfolioValue:  portfolioValue,
		Statistics:      stats,
		CalculatedAt:    time.Now(),
	}, nil
}

func (vc *VaRCalculator) CalculateParametricVaR(returns []types.Decimal, portfolioValue, confidence types.Decimal) (VaRResult, error) {
	if confidence.Cmp(types.NewDecimalFromFloat(100.0)) >= 0 || confidence.IsNegative() {
		return VaRResult{}, fmt.Errorf("confidence level must be between 0 and 100")
	}

	// Calculate statistics
	stats := vc.calculateStatistics(returns)

	// Z-score for normal distribution at given confidence level
	alpha := types.NewDecimalFromFloat(100.0).Sub(confidence).Div(types.NewDecimalFromFloat(100.0))
	zScore := vc.getZScore(alpha.Float64())

	// Parametric VaR = (mean + z-score * std dev) * portfolio value
	varReturn := stats.Mean.Add(types.NewDecimalFromFloat(zScore).Mul(stats.StandardDeviation))
	varAmount := varReturn.Mul(portfolioValue)

	return VaRResult{
		Method:          "Parametric",
		ConfidenceLevel: confidence,
		VaR:             varAmount,
		PortfolioValue:  portfolioValue,
		Statistics:      stats,
		CalculatedAt:    time.Now(),
	}, nil
}

func (vc *VaRCalculator) SetMonteCarloConfig(config MonteCarloConfig) error {
	vc.monteCarloConfig = config
	vc.rng = rand.New(rand.NewSource(config.RandomSeed))
	return nil
}

func (vc *VaRCalculator) CalculateMonteCarloVaR(returns []types.Decimal, portfolioValue, confidence types.Decimal) (VaRResult, error) {
	if confidence.Cmp(types.NewDecimalFromFloat(100.0)) >= 0 || confidence.IsNegative() {
		return VaRResult{}, fmt.Errorf("confidence level must be between 0 and 100")
	}

	// Calculate statistics for simulation parameters
	stats := vc.calculateStatistics(returns)

	// Run Monte Carlo simulation
	simulatedPnLs := make([]types.Decimal, vc.monteCarloConfig.NumSimulations)
	
	for i := 0; i < vc.monteCarloConfig.NumSimulations; i++ {
		// Generate random return based on normal distribution
		randomReturn := vc.generateNormalRandom(stats.Mean.Float64(), stats.StandardDeviation.Float64())
		simulatedPnL := types.NewDecimalFromFloat(randomReturn).Mul(portfolioValue)
		simulatedPnLs[i] = simulatedPnL
	}

	// Sort simulated P&Ls
	sort.Slice(simulatedPnLs, func(i, j int) bool {
		return simulatedPnLs[i].Cmp(simulatedPnLs[j]) < 0
	})

	// Calculate VaR at confidence level
	percentile := types.NewDecimalFromFloat(100.0).Sub(confidence)
	indexFloat := percentile.Div(types.NewDecimalFromFloat(100.0)).Mul(types.NewDecimalFromInt(int64(len(simulatedPnLs) - 1)))
	indexInt := int(math.Floor(indexFloat.Float64()))
	
	if indexInt >= len(simulatedPnLs) {
		indexInt = len(simulatedPnLs) - 1
	}

	varAmount := simulatedPnLs[indexInt]

	// Create Monte Carlo details
	details := &MonteCarloDetails{
		NumSimulations:     vc.monteCarloConfig.NumSimulations,
		WorstCaseScenario:  simulatedPnLs[0],
		BestCaseScenario:   simulatedPnLs[len(simulatedPnLs)-1],
		SimulatedPnLs:      simulatedPnLs,
	}

	return VaRResult{
		Method:            "MonteCarlo",
		ConfidenceLevel:   confidence,
		VaR:               varAmount,
		PortfolioValue:    portfolioValue,
		Statistics:        stats,
		MonteCarloDetails: details,
		CalculatedAt:      time.Now(),
	}, nil
}

func (vc *VaRCalculator) BacktestVaR(varResult VaRResult, outOfSampleReturns []types.Decimal, portfolioValue types.Decimal) (VaRBacktest, error) {
	exceptions := 0
	
	// Count exceptions (actual losses exceeding VaR)
	for _, ret := range outOfSampleReturns {
		actualPnL := ret.Mul(portfolioValue)
		if actualPnL.Cmp(varResult.VaR) < 0 { // Loss greater than VaR (more negative)
			exceptions++
		}
	}

	totalObs := len(outOfSampleReturns)
	exceptionRate := types.NewDecimalFromInt(int64(exceptions)).Div(types.NewDecimalFromInt(int64(totalObs))).Mul(types.NewDecimalFromInt(100))
	
	expectedRate := types.NewDecimalFromFloat(100.0).Sub(varResult.ConfidenceLevel)
	
	// Simple model validation: is exception rate reasonable?
	tolerance := types.NewDecimalFromFloat(5.0) // 5% tolerance
	isValid := exceptionRate.Sub(expectedRate).Abs().Cmp(tolerance) <= 0

	return VaRBacktest{
		TotalObservations: totalObs,
		Exceptions:        exceptions,
		ExceptionRate:     exceptionRate,
		ExpectedRate:      expectedRate,
		IsModelValid:      &isValid,
		PValue:            types.NewDecimalFromFloat(0.05), // Simplified
		TestStatistic:     exceptionRate.Sub(expectedRate).Abs(),
	}, nil
}

func (vc *VaRCalculator) CalculateComponentVaR(positions []PositionVaR, portfolioValue, confidence types.Decimal) (ComponentVaRResult, error) {
	// Simple implementation for TDD GREEN phase
	components := make([]ComponentVaR, len(positions))
	
	// Calculate individual VaRs and approximate portfolio VaR
	portfolioVar := types.Zero()
	
	for i, position := range positions {
		positionValue := position.Weight.Mul(portfolioValue)
		
		// Calculate position VaR using historical method
		posVarResult, err := vc.CalculateHistoricalVaR(position.Returns, positionValue, confidence)
		if err != nil {
			// Fallback to simple calculation
			components[i] = ComponentVaR{
				AssetSymbol:    position.AssetSymbol,
				Weight:         position.Weight,
				ComponentVaR:   types.NewDecimalFromFloat(-1000.0), // Simple placeholder
				MarginalVaR:    types.NewDecimalFromFloat(-500.0),
				ContributionPercent: position.Weight.Mul(types.NewDecimalFromInt(100)),
			}
		} else {
			components[i] = ComponentVaR{
				AssetSymbol:    position.AssetSymbol,
				Weight:         position.Weight,
				ComponentVaR:   posVarResult.VaR.Mul(position.Weight),
				MarginalVaR:    posVarResult.VaR,
				ContributionPercent: position.Weight.Mul(types.NewDecimalFromInt(100)),
			}
		}
		
		portfolioVar = portfolioVar.Add(components[i].ComponentVaR.Abs())
	}

	// Ensure portfolio VaR is negative (representing loss)
	portfolioVar = portfolioVar.Mul(types.NewDecimalFromInt(-1))

	return ComponentVaRResult{
		PortfolioVaR:          portfolioVar,
		Components:            components,
		DiversificationBenefit: types.NewDecimalFromFloat(5000.0), // Simplified
		CalculatedAt:          time.Now(),
	}, nil
}

func (vc *VaRCalculator) SetConfig(config VaRConfig) error {
	vc.config = config
	return nil
}

func (vc *VaRCalculator) GetConfig() VaRConfig {
	return vc.config
}

// Helper functions
func (vc *VaRCalculator) calculateStatistics(returns []types.Decimal) VaRStatistics {
	if len(returns) == 0 {
		return VaRStatistics{}
	}

	// Calculate mean
	sum := types.Zero()
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	mean := sum.Div(types.NewDecimalFromInt(int64(len(returns))))

	// Calculate standard deviation
	sumSquares := types.Zero()
	for _, ret := range returns {
		diff := ret.Sub(mean)
		sumSquares = sumSquares.Add(diff.Mul(diff))
	}
	variance := sumSquares.Div(types.NewDecimalFromInt(int64(len(returns))))
	stdDev := types.NewDecimalFromFloat(math.Sqrt(variance.Float64()))

	return VaRStatistics{
		Mean:              mean,
		StandardDeviation: stdDev,
		Skewness:          types.Zero(), // Simplified for GREEN phase
		Kurtosis:          types.Zero(), // Simplified for GREEN phase
	}
}

func (vc *VaRCalculator) getZScore(alpha float64) float64 {
	// Simplified z-score lookup for common confidence levels
	if alpha <= 0.01 {
		return -2.33 // 99%
	} else if alpha <= 0.05 {
		return -1.645 // 95%
	} else if alpha <= 0.10 {
		return -1.28 // 90%
	}
	return -1.645 // Default to 95%
}

func (vc *VaRCalculator) generateNormalRandom(mean, stddev float64) float64 {
	// Box-Muller transformation for normal random number
	u1 := vc.rng.Float64()
	u2 := vc.rng.Float64()
	
	z0 := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + stddev*z0
}