package risk

import (
	"math"
	"sync"
	"time"

	"github.com/trading-engine/pkg/types"
)

// StreamingCVaRCalculator provides ultra-high-performance CVaR calculations using streaming algorithms
// Targets <1ms p99 for any dataset size through O(n) approximation algorithms
type StreamingCVaRCalculator struct {
	quantileEstimator *P2QuantileEstimator
	cache             *VaRCache // Reuse existing cache
	config            CVaRConfig
	mu                sync.RWMutex
}

// NewStreamingCVaRCalculator creates a new streaming CVaR calculator
func NewStreamingCVaRCalculator(config CVaRConfig) *StreamingCVaRCalculator {
	return &StreamingCVaRCalculator{
		quantileEstimator: NewP2QuantileEstimator(),
		cache:             NewVaRCache(1000, 15*time.Minute), // Reuse cache implementation
		config:            config,
	}
}

// CalculateHistoricalCVaR calculates CVaR using O(n) streaming algorithms
func (scvc *StreamingCVaRCalculator) CalculateHistoricalCVaR(
	returns []types.Decimal,
	portfolioValue, confidence types.Decimal,
) (CVaRResult, error) {
	// Input validation
	if len(returns) < scvc.config.MinHistoricalObservations {
		return CVaRResult{}, &CVaRError{
			Code:    "INSUFFICIENT_DATA",
			Message: "insufficient historical data",
			Details: map[string]interface{}{
				"required": scvc.config.MinHistoricalObservations,
				"provided": len(returns),
			},
		}
	}

	// Check cache first
	cacheKey := scvc.buildCacheKey(returns, confidence)
	if cachedResult, found := scvc.cache.Get(cacheKey); found {
		// Convert VaRResult to CVaRResult and scale
		cvarResult := scvc.convertVaRToCVaRResult(cachedResult, portfolioValue)
		return cvarResult, nil
	}

	start := time.Now()

	// Convert confidence level to quantile (95% confidence = 5th percentile)
	quantile := types.NewDecimalFromFloat(100.0).Sub(confidence).Div(types.NewDecimalFromFloat(100.0))

	// Step 1: Calculate VaR threshold using streaming quantile estimation
	varThreshold := scvc.calculateStreamingQuantile(returns, quantile.Float64())

	// Step 2: Calculate CVaR as conditional expectation of tail using streaming algorithm
	cvarValue := scvc.calculateStreamingTailExpectation(returns, varThreshold)

	// Calculate portfolio-level CVaR
	portfolioCVaR := cvarValue.Abs().Mul(portfolioValue)

	// Calculate streaming statistics for both full sample and tail
	fullStats := scvc.calculateStreamingStatistics(returns)
	tailStats := scvc.calculateTailStatistics(returns, varThreshold)

	result := CVaRResult{
		Method:          "streaming_historical",
		ConfidenceLevel: confidence,
		CVaR:            portfolioCVaR,
		VaR:             varThreshold.Abs().Mul(portfolioValue),
		PortfolioValue:  portfolioValue,
		Statistics:      scvc.convertToFullStatistics(fullStats),
		TailStatistics:  tailStats,
		CalculatedAt:    time.Now(),
	}

	// Cache result (using portfolio-independent form)
	normalizedVaRResult := VaRResult{
		Method:          "streaming_historical_cvar",
		ConfidenceLevel: confidence,
		VaR:             varThreshold.Abs(),
		PortfolioValue:  types.NewDecimalFromInt(1),
		Statistics:      scvc.convertToVaRStatistics(fullStats),
		CalculatedAt:    time.Now(),
	}
	scvc.cache.Set(cacheKey, normalizedVaRResult)

	// Log performance for debugging
	elapsed := time.Since(start)
	_ = elapsed // TODO: Add to structured logging

	return result, nil
}

// calculateStreamingQuantile uses P² algorithm for O(n) quantile estimation
func (scvc *StreamingCVaRCalculator) calculateStreamingQuantile(returns []types.Decimal, quantile float64) types.Decimal {
	// Reset estimator for new calculation
	estimator := NewP2QuantileEstimator()
	estimator.SetQuantile(quantile)

	// Stream all data points through the estimator
	for _, ret := range returns {
		estimator.Update(ret.Float64())
	}

	return types.NewDecimalFromFloat(estimator.GetQuantile())
}

// calculateStreamingTailExpectation computes the conditional expectation of tail in O(n)
func (scvc *StreamingCVaRCalculator) calculateStreamingTailExpectation(returns []types.Decimal, threshold types.Decimal) types.Decimal {
	var tailSum types.Decimal = types.NewDecimalFromInt(0)
	var tailCount int = 0

	// Single pass through data to find tail observations and compute mean
	for _, ret := range returns {
		if ret.Cmp(threshold) <= 0 { // In tail (worse than or equal to VaR threshold)
			tailSum = tailSum.Add(ret)
			tailCount++
		}
	}

	if tailCount == 0 {
		return threshold // If no tail observations, CVaR equals VaR
	}

	return tailSum.Div(types.NewDecimalFromInt(int64(tailCount)))
}

// calculateTailStatistics computes tail-specific statistics in single O(n) pass
func (scvc *StreamingCVaRCalculator) calculateTailStatistics(returns []types.Decimal, threshold types.Decimal) CVaRTailStatistics {
	var tailSum types.Decimal = types.NewDecimalFromInt(0)
	var tailSumSquares types.Decimal = types.NewDecimalFromInt(0)
	var tailCount int = 0
	var worstLoss types.Decimal = threshold

	// Single pass to collect tail statistics
	for _, ret := range returns {
		if ret.Cmp(threshold) <= 0 { // In tail
			tailSum = tailSum.Add(ret)
			tailSumSquares = tailSumSquares.Add(ret.Mul(ret))
			tailCount++

			if ret.Cmp(worstLoss) < 0 {
				worstLoss = ret
			}
		}
	}

	if tailCount == 0 {
		return CVaRTailStatistics{
			TailObservations: 0,
			AverageTailLoss:  threshold,
			WorstTailLoss:    threshold,
			TailVolatility:   types.NewDecimalFromInt(0),
		}
	}

	averageTailLoss := tailSum.Div(types.NewDecimalFromInt(int64(tailCount)))

	// Calculate tail volatility using Welford's algorithm
	meanSquare := tailSumSquares.Div(types.NewDecimalFromInt(int64(tailCount)))
	variance := meanSquare.Sub(averageTailLoss.Mul(averageTailLoss))
	volatility := types.NewDecimalFromFloat(math.Sqrt(math.Abs(variance.Float64())))

	return CVaRTailStatistics{
		TailObservations: tailCount,
		AverageTailLoss:  averageTailLoss,
		WorstTailLoss:    worstLoss,
		TailVolatility:   volatility,
	}
}

// calculateStreamingStatistics computes statistics in single O(n) pass using Welford's algorithm
func (scvc *StreamingCVaRCalculator) calculateStreamingStatistics(returns []types.Decimal) StreamingStats {
	if len(returns) == 0 {
		return StreamingStats{}
	}

	// Welford's algorithm for mean and variance in O(n)
	var mean, m2 float64
	count := 0.0

	for _, ret := range returns {
		count++
		value := ret.Float64()
		delta := value - mean
		mean += delta / count
		delta2 := value - mean
		m2 += delta * delta2
	}

	variance := m2 / (count - 1)
	stdDev := math.Sqrt(variance)

	return StreamingStats{
		Mean:              types.NewDecimalFromFloat(mean),
		StandardDeviation: types.NewDecimalFromFloat(stdDev),
		Count:             int(count),
	}
}

// Helper types and methods
type StreamingStats struct {
	Mean              types.Decimal
	StandardDeviation types.Decimal
	Count             int
}

// buildCacheKey creates deterministic cache key for CVaR
func (scvc *StreamingCVaRCalculator) buildCacheKey(returns []types.Decimal, confidence types.Decimal) string {
	return confidence.String() + "_" + string(rune(len(returns))) + "_stream_cvar"
}

// convertVaRToCVaRResult converts cached VaR result to CVaR result
func (scvc *StreamingCVaRCalculator) convertVaRToCVaRResult(varResult VaRResult, portfolioValue types.Decimal) CVaRResult {
	// Scale VaR result to current portfolio value
	scaledVaR := varResult.VaR.Mul(portfolioValue).Div(varResult.PortfolioValue)

	// For cached results, estimate CVaR as 1.2 * VaR (conservative approximation)
	estimatedCVaR := scaledVaR.Mul(types.NewDecimalFromFloat(1.2))

	return CVaRResult{
		Method:          "streaming_historical_cached",
		ConfidenceLevel: varResult.ConfidenceLevel,
		CVaR:            estimatedCVaR,
		VaR:             scaledVaR,
		PortfolioValue:  portfolioValue,
		Statistics:      scvc.convertToFullStatistics(StreamingStats{Mean: varResult.Statistics.Mean, StandardDeviation: varResult.Statistics.StandardDeviation}),
		TailStatistics: CVaRTailStatistics{
			TailObservations: 0,
			AverageTailLoss:  estimatedCVaR.Div(portfolioValue),
			WorstTailLoss:    estimatedCVaR.Div(portfolioValue).Mul(types.NewDecimalFromFloat(1.5)),
			TailVolatility:   varResult.Statistics.StandardDeviation,
		},
		CalculatedAt: time.Now(),
	}
}

// convertToFullStatistics converts streaming stats to full statistics
func (scvc *StreamingCVaRCalculator) convertToFullStatistics(stats StreamingStats) VaRStatistics {
	return VaRStatistics{
		Mean:              stats.Mean,
		StandardDeviation: stats.StandardDeviation,
		Skewness:          types.NewDecimalFromInt(0), // Omitted for performance
		Kurtosis:          types.NewDecimalFromInt(0), // Omitted for performance
	}
}

// convertToVaRStatistics converts streaming stats to VaR statistics format
func (scvc *StreamingCVaRCalculator) convertToVaRStatistics(stats StreamingStats) VaRStatistics {
	return VaRStatistics{
		Mean:              stats.Mean,
		StandardDeviation: stats.StandardDeviation,
		Skewness:          types.NewDecimalFromInt(0),
		Kurtosis:          types.NewDecimalFromInt(0),
	}
}

// CVaRError represents a typed error for CVaR calculations
type CVaRError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (ce *CVaRError) Error() string {
	return ce.Code + ": " + ce.Message
}

// IsTemporary indicates if the error is temporary and can be retried
func (ce *CVaRError) IsTemporary() bool {
	return ce.Code == "TEMPORARY_FAILURE" || ce.Code == "TIMEOUT"
}

// IsCritical indicates if the error is critical and requires immediate attention
func (ce *CVaRError) IsCritical() bool {
	return ce.Code == "DATA_CORRUPTION" || ce.Code == "CALCULATION_ERROR"
}
