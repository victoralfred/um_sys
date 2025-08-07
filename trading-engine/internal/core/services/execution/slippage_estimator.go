package execution

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// SlippageEstimatorConfig contains configuration for slippage estimation
type SlippageEstimatorConfig struct {
	BaseSlippageBps       float64 `json:"base_slippage_bps"`
	VolatilityMultiplier  float64 `json:"volatility_multiplier"`
	LiquidityImpactFactor float64 `json:"liquidity_impact_factor"`
	TimeDecayFactor       float64 `json:"time_decay_factor"`
	MaxHistoryWindow      int     `json:"max_history_window"`
	MinLiquidityThreshold float64 `json:"min_liquidity_threshold"`
}

// DefaultSlippageEstimatorConfig returns reasonable default configuration
func DefaultSlippageEstimatorConfig() SlippageEstimatorConfig {
	return SlippageEstimatorConfig{
		BaseSlippageBps:       2.0,  // 2 bps base slippage
		VolatilityMultiplier:  1.5,  // 1.5x volatility impact
		LiquidityImpactFactor: 1.2,  // 1.2x liquidity impact
		TimeDecayFactor:       0.05, // 5% per minute decay
		MaxHistoryWindow:      50,   // Keep 50 price points
		MinLiquidityThreshold: 10.0, // Minimum 10 shares liquidity
	}
}

// DefaultSlippageEstimator implements sophisticated slippage estimation algorithms
// TDD REFACTOR phase - enhanced production-ready implementation
type DefaultSlippageEstimator struct {
	mu              sync.RWMutex
	config          SlippageEstimatorConfig
	priceHistory    map[string][]PricePoint   // Historical prices by symbol
	volatilityCache map[string]VolatilityData // Cached volatility calculations
	metrics         SlippageEstimatorMetrics  // Performance metrics

	// Advanced features
	liquidityProfile  map[string]LiquidityProfile   // Liquidity patterns by symbol
	correlationMatrix map[string]map[string]float64 // Asset correlations
	marketRegime      MarketRegimeData              // Current market conditions

	// Model parameters learned from historical data
	impactModels    map[string]ImpactModel // Symbol-specific impact models
	lastModelUpdate time.Time              // When models were last updated
}

// PricePoint represents a historical price observation
type PricePoint struct {
	Price     types.Decimal
	Timestamp time.Time
}

// VolatilityData contains volatility calculation results
type VolatilityData struct {
	Volatility   types.Decimal
	CalculatedAt time.Time
	SampleCount  int
}

// SlippageEstimatorMetrics tracks estimator performance
type SlippageEstimatorMetrics struct {
	TotalEstimations      uint64            `json:"total_estimations"`
	AverageEstimationTime time.Duration     `json:"average_estimation_time"`
	CacheHitRate          float64           `json:"cache_hit_rate"`
	ModelAccuracy         float64           `json:"model_accuracy"`
	LastUpdated           time.Time         `json:"last_updated"`
	EstimationsPerSymbol  map[string]uint64 `json:"estimations_per_symbol"`
}

// LiquidityProfile contains liquidity pattern analysis
type LiquidityProfile struct {
	AverageBidSize    types.Decimal   `json:"average_bid_size"`
	AverageAskSize    types.Decimal   `json:"average_ask_size"`
	TypicalSpread     types.Decimal   `json:"typical_spread"`
	LiquidityScore    float64         `json:"liquidity_score"` // 0-1 score
	TimeOfDayPatterns map[int]float64 `json:"time_patterns"`   // Hour -> liquidity factor
	LastUpdated       time.Time       `json:"last_updated"`
}

// MarketRegimeData contains current market condition analysis
type MarketRegimeData struct {
	RegimeType       string    `json:"regime_type"`       // "normal", "volatile", "trending"
	VolatilityRegime float64   `json:"volatility_regime"` // Current vol vs historical
	TrendDirection   string    `json:"trend_direction"`   // "up", "down", "sideways"
	MarketStress     float64   `json:"market_stress"`     // 0-1 stress indicator
	LastUpdated      time.Time `json:"last_updated"`
}

// ImpactModel contains symbol-specific impact parameters
type ImpactModel struct {
	LinearImpact     float64   `json:"linear_impact"`    // Linear impact coefficient
	SquareRootImpact float64   `json:"sqrt_impact"`      // Square root impact coefficient
	TemporaryImpact  float64   `json:"temporary_impact"` // Temporary impact decay
	PermanentImpact  float64   `json:"permanent_impact"` // Permanent impact
	ModelConfidence  float64   `json:"confidence"`       // 0-1 confidence in model
	SampleSize       int       `json:"sample_size"`      // Number of observations
	LastCalibrated   time.Time `json:"last_calibrated"`
}

// NewSlippageEstimator creates a new slippage estimator with default configuration
func NewSlippageEstimator() *DefaultSlippageEstimator {
	return &DefaultSlippageEstimator{
		config:          DefaultSlippageEstimatorConfig(),
		priceHistory:    make(map[string][]PricePoint),
		volatilityCache: make(map[string]VolatilityData),
		metrics: SlippageEstimatorMetrics{
			EstimationsPerSymbol: make(map[string]uint64),
			LastUpdated:          time.Now(),
		},
		liquidityProfile:  make(map[string]LiquidityProfile),
		correlationMatrix: make(map[string]map[string]float64),
		marketRegime: MarketRegimeData{
			RegimeType:     "normal",
			TrendDirection: "sideways",
			LastUpdated:    time.Now(),
		},
		impactModels:    make(map[string]ImpactModel),
		lastModelUpdate: time.Now(),
	}
}

// NewSlippageEstimatorWithConfig creates a slippage estimator with custom configuration
func NewSlippageEstimatorWithConfig(config SlippageEstimatorConfig) *DefaultSlippageEstimator {
	return &DefaultSlippageEstimator{
		config:          config,
		priceHistory:    make(map[string][]PricePoint),
		volatilityCache: make(map[string]VolatilityData),
		metrics: SlippageEstimatorMetrics{
			EstimationsPerSymbol: make(map[string]uint64),
			LastUpdated:          time.Now(),
		},
		liquidityProfile:  make(map[string]LiquidityProfile),
		correlationMatrix: make(map[string]map[string]float64),
		marketRegime: MarketRegimeData{
			RegimeType:     "normal",
			TrendDirection: "sideways",
			LastUpdated:    time.Now(),
		},
		impactModels:    make(map[string]ImpactModel),
		lastModelUpdate: time.Now(),
	}
}

// EstimateSlippage estimates slippage for an order given market conditions with enhanced analytics
func (e *DefaultSlippageEstimator) EstimateSlippage(ctx context.Context, order *domain.Order, marketData *ports.MarketData) (types.Decimal, error) {
	startTime := time.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	// Update metrics
	e.metrics.TotalEstimations++
	symbol := order.Asset.Symbol
	e.metrics.EstimationsPerSymbol[symbol]++

	// Validate inputs
	if order == nil || marketData == nil {
		return types.NewDecimalFromFloat(0), fmt.Errorf("order and market data cannot be nil")
	}

	// Check for zero liquidity
	if marketData.BidSize.IsZero() && marketData.AskSize.IsZero() {
		return types.NewDecimalFromFloat(0), fmt.Errorf("no liquidity available")
	}

	// Calculate base slippage
	baseSlippage := types.NewDecimalFromFloat(e.config.BaseSlippageBps / 10000.0) // Convert bps to decimal

	// Get liquidity impact
	liquidityImpact := e.calculateLiquidityImpact(order, marketData)

	// Get volatility adjustment
	volatilityAdjustment := e.calculateVolatilityAdjustment(order.Asset)

	// Get time decay adjustment
	timeAdjustment := e.calculateTimeDecay(marketData.Timestamp)

	// Calculate order type specific adjustments
	orderTypeAdjustment := e.calculateOrderTypeAdjustment(order, marketData)

	// Combine all factors
	totalSlippage := baseSlippage.
		Add(liquidityImpact).
		Add(volatilityAdjustment).
		Add(timeAdjustment).
		Add(orderTypeAdjustment)

	// Convert to basis points for return
	slippageBps := totalSlippage.Mul(types.NewDecimalFromFloat(10000.0))

	// Ensure non-negative slippage (unless market is crossed)
	if slippageBps.IsNegative() && !e.isMarketCrossed(marketData) {
		slippageBps = types.NewDecimalFromFloat(0)
	}

	// Update performance metrics
	duration := time.Since(startTime)
	if e.metrics.TotalEstimations == 1 {
		e.metrics.AverageEstimationTime = duration
	} else {
		// Exponential moving average
		alpha := 0.1
		e.metrics.AverageEstimationTime = time.Duration(
			float64(e.metrics.AverageEstimationTime)*(1-alpha) +
				float64(duration)*alpha,
		)
	}
	e.metrics.LastUpdated = time.Now()

	return slippageBps, nil
}

// UpdateHistoricalData updates historical price data for volatility calculations
func (e *DefaultSlippageEstimator) UpdateHistoricalData(ctx context.Context, asset *domain.Asset, price types.Decimal, timestamp time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	symbol := asset.Symbol

	// Add new price point
	pricePoint := PricePoint{
		Price:     price,
		Timestamp: timestamp,
	}

	history := e.priceHistory[symbol]
	history = append(history, pricePoint)

	// Maintain window size
	if len(history) > e.config.MaxHistoryWindow {
		history = history[len(history)-e.config.MaxHistoryWindow:]
	}

	e.priceHistory[symbol] = history

	// Invalidate volatility cache for this symbol
	delete(e.volatilityCache, symbol)

	return nil
}

// GetVolatility calculates and returns volatility for an asset
func (e *DefaultSlippageEstimator) GetVolatility(ctx context.Context, asset *domain.Asset) (types.Decimal, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	symbol := asset.Symbol

	// Check cache first (valid for 1 minute)
	if cached, exists := e.volatilityCache[symbol]; exists {
		if time.Since(cached.CalculatedAt) < time.Minute {
			return cached.Volatility, nil
		}
	}

	// Calculate volatility from price history
	history, exists := e.priceHistory[symbol]
	if !exists || len(history) < 2 {
		// No sufficient history, return default volatility
		defaultVol := types.NewDecimalFromFloat(1.0) // 100 bps
		return defaultVol, nil
	}

	volatility := e.calculateVolatilityFromHistory(history)

	// Cache the result
	e.volatilityCache[symbol] = VolatilityData{
		Volatility:   volatility,
		CalculatedAt: time.Now(),
		SampleCount:  len(history),
	}

	return volatility, nil
}

// GetConfig returns current configuration
func (e *DefaultSlippageEstimator) GetConfig() SlippageEstimatorConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// Helper methods for slippage calculation

// calculateLiquidityImpact estimates slippage based on order size vs available liquidity
func (e *DefaultSlippageEstimator) calculateLiquidityImpact(order *domain.Order, marketData *ports.MarketData) types.Decimal {
	var availableLiquidity types.Decimal

	// Determine available liquidity based on order side
	if order.Side == domain.OrderSideBuy {
		availableLiquidity = marketData.AskSize
	} else {
		availableLiquidity = marketData.BidSize
	}

	// If no liquidity, return high impact
	if availableLiquidity.IsZero() {
		return types.NewDecimalFromFloat(e.config.BaseSlippageBps * 5 / 10000.0) // 5x base slippage
	}

	// Calculate impact ratio
	impactRatio := order.Quantity.Div(availableLiquidity)

	// Apply impact function: impact increases non-linearly with size
	if impactRatio.Cmp(types.NewDecimalFromFloat(0.5)) <= 0 {
		// Small order: minimal impact
		return types.NewDecimalFromFloat(0.1 / 10000.0) // 0.1 bps
	} else if impactRatio.Cmp(types.NewDecimalFromFloat(1.0)) <= 0 {
		// Medium order: moderate impact
		impact := impactRatio.Mul(types.NewDecimalFromFloat(e.config.LiquidityImpactFactor * e.config.BaseSlippageBps))
		return impact.Div(types.NewDecimalFromFloat(10000.0))
	} else {
		// Large order: high impact (exceeds available liquidity)
		impact := impactRatio.Mul(types.NewDecimalFromFloat(e.config.LiquidityImpactFactor * e.config.BaseSlippageBps * 2))
		return impact.Div(types.NewDecimalFromFloat(10000.0))
	}
}

// calculateVolatilityAdjustment adjusts slippage based on asset volatility
func (e *DefaultSlippageEstimator) calculateVolatilityAdjustment(asset *domain.Asset) types.Decimal {
	// Get cached volatility or use default
	volatilityData, exists := e.volatilityCache[asset.Symbol]
	if !exists {
		// Use default moderate volatility
		return types.NewDecimalFromFloat(0.5 / 10000.0) // 0.5 bps
	}

	// Apply volatility multiplier
	adjustment := volatilityData.Volatility.Mul(types.NewDecimalFromFloat(e.config.VolatilityMultiplier))
	return adjustment.Div(types.NewDecimalFromFloat(10000.0))
}

// calculateTimeDecay adjusts slippage based on market data age
func (e *DefaultSlippageEstimator) calculateTimeDecay(marketDataTime time.Time) types.Decimal {
	age := time.Since(marketDataTime)
	ageMinutes := age.Minutes()

	// Apply exponential decay: older data = higher uncertainty = higher slippage
	if ageMinutes <= 1.0 {
		return types.NewDecimalFromFloat(0) // Fresh data, no adjustment
	}

	decayFactor := math.Pow(1.0+e.config.TimeDecayFactor, ageMinutes-1.0) - 1.0
	adjustment := types.NewDecimalFromFloat(decayFactor * e.config.BaseSlippageBps / 10000.0)

	return adjustment
}

// calculateOrderTypeAdjustment adjusts slippage based on order type
func (e *DefaultSlippageEstimator) calculateOrderTypeAdjustment(order *domain.Order, marketData *ports.MarketData) types.Decimal {
	switch order.Type {
	case domain.OrderTypeMarket:
		// Market orders cross the spread
		spread := e.calculateSpread(marketData)
		return spread.Div(types.NewDecimalFromFloat(2.0)) // Half spread impact

	case domain.OrderTypeLimit:
		// Limit order adjustment based on aggressiveness
		return e.calculateLimitOrderAdjustment(order, marketData)

	default:
		return types.NewDecimalFromFloat(0)
	}
}

// calculateLimitOrderAdjustment calculates adjustment for limit orders
func (e *DefaultSlippageEstimator) calculateLimitOrderAdjustment(order *domain.Order, marketData *ports.MarketData) types.Decimal {
	if order.Price.IsZero() {
		return types.NewDecimalFromFloat(0)
	}

	var referencePrice types.Decimal

	if order.Side == domain.OrderSideBuy {
		referencePrice = marketData.AskPrice
		// Aggressive if limit price > ask price
		if order.Price.Cmp(referencePrice) > 0 {
			priceDiff := order.Price.Sub(referencePrice)
			return priceDiff.Div(referencePrice) // Percentage difference as slippage
		}
	} else {
		referencePrice = marketData.BidPrice
		// Aggressive if limit price < bid price
		if order.Price.Cmp(referencePrice) < 0 {
			priceDiff := referencePrice.Sub(order.Price)
			return priceDiff.Div(referencePrice) // Percentage difference as slippage
		}
	}

	// Passive limit order - minimal slippage
	return types.NewDecimalFromFloat(0.05 / 10000.0) // 0.05 bps
}

// calculateSpread calculates bid-ask spread
func (e *DefaultSlippageEstimator) calculateSpread(marketData *ports.MarketData) types.Decimal {
	spread := marketData.AskPrice.Sub(marketData.BidPrice)
	midPrice := marketData.BidPrice.Add(marketData.AskPrice).Div(types.NewDecimalFromFloat(2.0))

	if midPrice.IsZero() {
		return types.NewDecimalFromFloat(0)
	}

	return spread.Div(midPrice) // Relative spread
}

// isMarketCrossed checks if the market is crossed (bid > ask)
func (e *DefaultSlippageEstimator) isMarketCrossed(marketData *ports.MarketData) bool {
	return marketData.BidPrice.Cmp(marketData.AskPrice) > 0
}

// calculateVolatilityFromHistory calculates volatility from price history
func (e *DefaultSlippageEstimator) calculateVolatilityFromHistory(history []PricePoint) types.Decimal {
	if len(history) < 2 {
		return types.NewDecimalFromFloat(1.0) // Default 100 bps
	}

	// Calculate returns
	returns := make([]float64, len(history)-1)
	for i := 1; i < len(history); i++ {
		if history[i-1].Price.IsZero() {
			continue
		}

		returnValue := history[i].Price.Div(history[i-1].Price).Sub(types.NewDecimalFromFloat(1.0))
		returns[i-1] = returnValue.Float64()
	}

	if len(returns) == 0 {
		return types.NewDecimalFromFloat(1.0)
	}

	// Calculate standard deviation of returns
	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += math.Pow(ret-mean, 2)
	}
	variance /= float64(len(returns))

	volatility := math.Sqrt(variance)

	// Convert to basis points and annualize (assuming daily data)
	annualizedVolatility := volatility * math.Sqrt(252) * 10000 // 252 trading days

	return types.NewDecimalFromFloat(annualizedVolatility)
}

// Enhanced REFACTOR methods for production-ready functionality

// GetMetrics returns current estimator performance metrics
func (e *DefaultSlippageEstimator) GetMetrics() SlippageEstimatorMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metrics
}

// UpdateLiquidityProfile updates liquidity profile for a symbol
func (e *DefaultSlippageEstimator) UpdateLiquidityProfile(symbol string, bidSize, askSize, spread types.Decimal) {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile, exists := e.liquidityProfile[symbol]
	if !exists {
		profile = LiquidityProfile{
			TimeOfDayPatterns: make(map[int]float64),
			LastUpdated:       time.Now(),
		}
	}

	// Update running averages (simple exponential moving average)
	alpha := 0.1
	if profile.AverageBidSize.IsZero() {
		profile.AverageBidSize = bidSize
		profile.AverageAskSize = askSize
		profile.TypicalSpread = spread
	} else {
		profile.AverageBidSize = profile.AverageBidSize.Mul(types.NewDecimalFromFloat(1 - alpha)).Add(bidSize.Mul(types.NewDecimalFromFloat(alpha)))
		profile.AverageAskSize = profile.AverageAskSize.Mul(types.NewDecimalFromFloat(1 - alpha)).Add(askSize.Mul(types.NewDecimalFromFloat(alpha)))
		profile.TypicalSpread = profile.TypicalSpread.Mul(types.NewDecimalFromFloat(1 - alpha)).Add(spread.Mul(types.NewDecimalFromFloat(alpha)))
	}

	// Calculate liquidity score (0-1, higher is better)
	totalLiquidity := bidSize.Add(askSize)
	referenceSize := types.NewDecimalFromFloat(1000.0) // Reference size for scoring
	profile.LiquidityScore = math.Min(1.0, totalLiquidity.Div(referenceSize).Float64())

	profile.LastUpdated = time.Now()
	e.liquidityProfile[symbol] = profile
}

// UpdateMarketRegime analyzes and updates current market conditions
func (e *DefaultSlippageEstimator) UpdateMarketRegime(overallVolatility float64, trendStrength float64, stressLevel float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Classify volatility regime
	if overallVolatility > 2.0 {
		e.marketRegime.VolatilityRegime = overallVolatility
		e.marketRegime.RegimeType = "volatile"
	} else if overallVolatility > 1.5 {
		e.marketRegime.VolatilityRegime = overallVolatility
		e.marketRegime.RegimeType = "elevated"
	} else {
		e.marketRegime.VolatilityRegime = overallVolatility
		e.marketRegime.RegimeType = "normal"
	}

	// Classify trend direction
	if trendStrength > 0.3 {
		e.marketRegime.TrendDirection = "up"
	} else if trendStrength < -0.3 {
		e.marketRegime.TrendDirection = "down"
	} else {
		e.marketRegime.TrendDirection = "sideways"
	}

	e.marketRegime.MarketStress = math.Max(0.0, math.Min(1.0, stressLevel))
	e.marketRegime.LastUpdated = time.Now()
}

// CalibrateImpactModel calibrates symbol-specific impact models from execution data
func (e *DefaultSlippageEstimator) CalibrateImpactModel(symbol string, executionData []ExecutionDataPoint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(executionData) < 10 {
		return fmt.Errorf("insufficient data points for calibration: need at least 10, got %d", len(executionData))
	}

	// Simple linear regression for impact model
	// Real implementation would use more sophisticated econometric methods
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(executionData))

	for _, point := range executionData {
		x := point.OrderSizeRatio // Independent variable: order size / average volume
		y := point.ActualSlippage // Dependent variable: realized slippage

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Linear impact coefficient: beta = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX^2)
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return fmt.Errorf("cannot calibrate model: singular matrix")
	}

	linearImpact := (n*sumXY - sumX*sumY) / denominator

	// Square root impact (simplified estimation)
	sqrtImpact := linearImpact * 0.6 // Empirical adjustment

	model := ImpactModel{
		LinearImpact:     linearImpact,
		SquareRootImpact: sqrtImpact,
		TemporaryImpact:  linearImpact * 0.7, // 70% temporary
		PermanentImpact:  linearImpact * 0.3, // 30% permanent
		ModelConfidence:  e.calculateModelConfidence(executionData, linearImpact),
		SampleSize:       len(executionData),
		LastCalibrated:   time.Now(),
	}

	e.impactModels[symbol] = model
	e.lastModelUpdate = time.Now()

	return nil
}

// EstimateSlippageWithModel uses calibrated models for more accurate estimation
func (e *DefaultSlippageEstimator) EstimateSlippageWithModel(ctx context.Context, order *domain.Order, marketData *ports.MarketData, averageVolume types.Decimal) (types.Decimal, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	symbol := order.Asset.Symbol

	// Get calibrated impact model if available
	model, hasModel := e.impactModels[symbol]
	if !hasModel || model.ModelConfidence < 0.3 {
		// Fall back to generic estimation (prevent infinite loop by using basic calculation)
		baseSlippage := types.NewDecimalFromFloat(e.config.BaseSlippageBps)
		return baseSlippage, nil
	}

	// Calculate order size ratio
	sizeRatio := order.Quantity.Div(averageVolume).Float64()

	// Apply sophisticated impact model
	linearImpact := model.LinearImpact * sizeRatio
	sqrtImpact := model.SquareRootImpact * math.Sqrt(sizeRatio)

	// Combine impacts with regime adjustments
	baseImpact := linearImpact + sqrtImpact

	// Apply market regime adjustments
	regimeAdjustment := 1.0
	switch e.marketRegime.RegimeType {
	case "volatile":
		regimeAdjustment = 1.5
	case "elevated":
		regimeAdjustment = 1.2
	default:
		regimeAdjustment = 1.0
	}

	adjustedImpact := baseImpact * regimeAdjustment * (1.0 + e.marketRegime.MarketStress*0.5)

	// Convert to basis points
	slippageBps := types.NewDecimalFromFloat(adjustedImpact * 10000.0)

	return slippageBps, nil
}

// ResetMetrics resets performance metrics
func (e *DefaultSlippageEstimator) ResetMetrics() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metrics = SlippageEstimatorMetrics{
		EstimationsPerSymbol: make(map[string]uint64),
		LastUpdated:          time.Now(),
	}
}

// GetLiquidityProfile returns liquidity profile for a symbol
func (e *DefaultSlippageEstimator) GetLiquidityProfile(symbol string) (LiquidityProfile, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	profile, exists := e.liquidityProfile[symbol]
	return profile, exists
}

// GetMarketRegime returns current market regime analysis
func (e *DefaultSlippageEstimator) GetMarketRegime() MarketRegimeData {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.marketRegime
}

// Helper types and methods

// ExecutionDataPoint represents historical execution data for model calibration
type ExecutionDataPoint struct {
	OrderSizeRatio  float64 // Order size / average volume
	ActualSlippage  float64 // Realized slippage in bps
	MarketCondition string  // "normal", "volatile", etc.
	Timestamp       time.Time
}

// calculateModelConfidence estimates confidence in the calibrated model
func (e *DefaultSlippageEstimator) calculateModelConfidence(data []ExecutionDataPoint, predictedImpact float64) float64 {
	if len(data) < 5 {
		return 0.1 // Low confidence with limited data
	}

	// Calculate R-squared (coefficient of determination)
	var totalSumSquares, residualSumSquares float64

	// Calculate mean of actual values
	var meanActual float64
	for _, point := range data {
		meanActual += point.ActualSlippage
	}
	meanActual /= float64(len(data))

	// Calculate sum of squares
	for _, point := range data {
		predicted := predictedImpact * point.OrderSizeRatio
		residualSumSquares += math.Pow(point.ActualSlippage-predicted, 2)
		totalSumSquares += math.Pow(point.ActualSlippage-meanActual, 2)
	}

	if totalSumSquares == 0 {
		return 0.5 // Moderate confidence if no variance
	}

	rSquared := 1 - (residualSumSquares / totalSumSquares)

	// Adjust confidence based on sample size
	sizeAdjustment := math.Min(1.0, float64(len(data))/50.0) // Full confidence with 50+ samples

	return math.Max(0.1, math.Min(0.95, rSquared*sizeAdjustment))
}
