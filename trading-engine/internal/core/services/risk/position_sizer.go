package risk

import (
	"fmt"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// ConservativeScaling represents different conservative scaling factors for Kelly Criterion
type ConservativeScaling int

const (
	ScalingConservative ConservativeScaling = iota
	ScalingModerate
	ScalingStandard
	ScalingAggressive
)

// SizingConfig contains configuration parameters for position sizing
type SizingConfig struct {
	ConservativeScaling ConservativeScaling `json:"conservative_scaling"`
	ConfidenceFactor    types.Decimal       `json:"confidence_factor"`
	MaxPositionSize     types.Decimal       `json:"max_position_size"`
	MinPositionSize     types.Decimal       `json:"min_position_size"`
	UseStopLossScaling  bool                `json:"use_stop_loss_scaling"`
}

// OptimalSizeResult contains comprehensive position sizing analysis
type OptimalSizeResult struct {
	PrimarySizingMethod string        `json:"primary_sizing_method"`
	RecommendedSize     types.Decimal `json:"recommended_size"`
	PositionValue       types.Decimal `json:"position_value"`
	StopLossSize        types.Decimal `json:"stop_loss_size"`
	KellySize           types.Decimal `json:"kelly_size"`
}

// PositionSizer handles position size calculations - GREEN phase implementation
type PositionSizer struct {
	config  SizingConfig
	scaling ConservativeScaling
}

// TDD GREEN phase - implement just enough to make tests pass
func NewPositionSizer() *PositionSizer {
	return &PositionSizer{
		config: SizingConfig{
			ConservativeScaling: ScalingConservative,
			ConfidenceFactor:    types.NewDecimalFromFloat(0.80),
			MaxPositionSize:     types.NewDecimalFromFloat(10.0),
			MinPositionSize:     types.NewDecimalFromFloat(0.5),
			UseStopLossScaling:  true,
		},
	}
}

func NewPositionSizerWithConservativeKelly(scaling ConservativeScaling) *PositionSizer {
	sizer := NewPositionSizer()
	sizer.config.ConservativeScaling = scaling
	sizer.scaling = scaling
	return sizer
}

func (ps *PositionSizer) GetConservativeScalingFactor() types.Decimal {
	switch ps.config.ConservativeScaling {
	case ScalingConservative:
		return types.NewDecimalFromFloat(0.10)
	case ScalingModerate:
		return types.NewDecimalFromFloat(0.15)
	case ScalingStandard:
		return types.NewDecimalFromFloat(0.25)
	case ScalingAggressive:
		return types.NewDecimalFromFloat(0.50)
	default:
		return types.NewDecimalFromFloat(0.10)
	}
}

func (ps *PositionSizer) SetConfidenceFactor(factor types.Decimal) error {
	// Validate confidence factor is between 0 and 1
	if factor.IsNegative() || factor.Cmp(types.NewDecimalFromInt(1)) > 0 {
		return fmt.Errorf("confidence factor must be between 0.0 and 1.0")
	}
	ps.config.ConfidenceFactor = factor
	return nil
}

func (ps *PositionSizer) GetConfig() SizingConfig {
	return ps.config
}

func (ps *PositionSizer) UpdateConfig(config SizingConfig) {
	ps.config = config
}

func (ps *PositionSizer) ApplyPositionLimits(size, portfolioValue, price types.Decimal) (types.Decimal, error) {
	if size.IsZero() || size.IsNegative() {
		return types.Zero(), nil
	}

	// Calculate position value and percentage
	positionValue := size.Mul(price)
	positionPercent := positionValue.Div(portfolioValue).Mul(types.NewDecimalFromInt(100))

	// Apply maximum position size limit
	if positionPercent.Cmp(ps.config.MaxPositionSize) > 0 {
		maxPositionValue := portfolioValue.Mul(ps.config.MaxPositionSize).Div(types.NewDecimalFromInt(100))
		return maxPositionValue.Div(price), nil
	}

	// Apply minimum position size limit
	if positionPercent.Cmp(ps.config.MinPositionSize) < 0 {
		minPositionValue := portfolioValue.Mul(ps.config.MinPositionSize).Div(types.NewDecimalFromInt(100))
		return minPositionValue.Div(price), nil
	}

	return size, nil
}

func (ps *PositionSizer) CalculateOptimalSize(portfolio *domain.Portfolio, asset *domain.Asset, entryPrice, stopPrice, riskPerTrade, confidenceFactor types.Decimal) (OptimalSizeResult, error) {
	result := OptimalSizeResult{}

	// Simple implementation to pass tests
	if !stopPrice.IsZero() {
		result.PrimarySizingMethod = "StopLoss"

		// Simple stop-loss based calculation
		portfolioValue := portfolio.Metrics.TotalValue
		riskAmount := portfolioValue.Mul(riskPerTrade).Div(types.NewDecimalFromInt(100))
		riskPerShare := entryPrice.Sub(stopPrice).Abs()

		if !riskPerShare.IsZero() {
			quantity := riskAmount.Div(riskPerShare).Mul(confidenceFactor)
			result.RecommendedSize = quantity
			result.StopLossSize = quantity
			result.PositionValue = quantity.Mul(entryPrice)
		}
	}

	return result, nil
}
