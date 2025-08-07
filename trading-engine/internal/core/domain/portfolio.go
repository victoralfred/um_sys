package domain

import (
	"fmt"
	"sort"
	"time"

	"github.com/trading-engine/pkg/types"
)

// PortfolioStatus represents the current state of the portfolio
type PortfolioStatus int

const (
	PortfolioStatusActive PortfolioStatus = iota + 1
	PortfolioStatusSuspended
	PortfolioStatusClosed
)

func (ps PortfolioStatus) String() string {
	switch ps {
	case PortfolioStatusActive:
		return "ACTIVE"
	case PortfolioStatusSuspended:
		return "SUSPENDED"
	case PortfolioStatusClosed:
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}

// PortfolioMetrics represents performance and risk metrics for the portfolio
type PortfolioMetrics struct {
	TotalValue          types.Decimal `json:"total_value"`
	CashBalance         types.Decimal `json:"cash_balance"`
	MarketValue         types.Decimal `json:"market_value"`
	TotalCost           types.Decimal `json:"total_cost"`
	TotalPnL            types.Decimal `json:"total_pnl"`
	RealizedPnL         types.Decimal `json:"realized_pnl"`
	UnrealizedPnL       types.Decimal `json:"unrealized_pnl"`
	TotalFees           types.Decimal `json:"total_fees"`
	NetPnL              types.Decimal `json:"net_pnl"`
	ReturnPercentage    types.Decimal `json:"return_percentage"`
	
	// Risk Metrics
	MaxDrawdown         types.Decimal `json:"max_drawdown"`
	MaxDrawdownPercent  types.Decimal `json:"max_drawdown_percent"`
	PeakValue           types.Decimal `json:"peak_value"`
	CurrentDrawdown     types.Decimal `json:"current_drawdown"`
	
	// Position Metrics
	TotalPositions      int           `json:"total_positions"`
	LongPositions       int           `json:"long_positions"`
	ShortPositions      int           `json:"short_positions"`
	ProfitablePositions int           `json:"profitable_positions"`
	LosingPositions     int           `json:"losing_positions"`
	
	// Concentration Risk
	MaxPositionWeight   types.Decimal `json:"max_position_weight"`
	ConcentrationRisk   types.Decimal `json:"concentration_risk"`
	
	UpdatedAt           time.Time     `json:"updated_at"`
}

// Portfolio represents a collection of positions and cash
type Portfolio struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Status          PortfolioStatus          `json:"status"`
	Positions       map[string]*Position     `json:"positions"`        // keyed by position ID
	AssetPositions  map[string]*Position     `json:"asset_positions"`  // keyed by asset symbol for quick lookup
	CashBalance     types.Decimal            `json:"cash_balance"`
	InitialCapital  types.Decimal            `json:"initial_capital"`
	Metrics         PortfolioMetrics         `json:"metrics"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	ClosedAt        *time.Time               `json:"closed_at,omitempty"`
}

// NewPortfolio creates a new portfolio with initial cash balance
func NewPortfolio(id, name string, initialCapital types.Decimal) (*Portfolio, error) {
	if id == "" {
		return nil, fmt.Errorf("portfolio ID cannot be empty")
	}
	
	if name == "" {
		return nil, fmt.Errorf("portfolio name cannot be empty")
	}
	
	if !initialCapital.IsPositive() {
		return nil, fmt.Errorf("initial capital must be positive")
	}
	
	now := time.Now()
	portfolio := &Portfolio{
		ID:             id,
		Name:           name,
		Status:         PortfolioStatusActive,
		Positions:      make(map[string]*Position),
		AssetPositions: make(map[string]*Position),
		CashBalance:    initialCapital,
		InitialCapital: initialCapital,
		CreatedAt:      now,
		UpdatedAt:      now,
		Metrics: PortfolioMetrics{
			TotalValue:    initialCapital,
			CashBalance:   initialCapital,
			PeakValue:     initialCapital,
			UpdatedAt:     now,
		},
	}
	
	return portfolio, nil
}

// AddPosition adds a position to the portfolio
func (p *Portfolio) AddPosition(position *Position) error {
	if p.Status != PortfolioStatusActive {
		return fmt.Errorf("cannot add position to %s portfolio", p.Status.String())
	}
	
	if position == nil {
		return fmt.Errorf("position cannot be nil")
	}
	
	if err := position.Validate(); err != nil {
		return fmt.Errorf("invalid position: %w", err)
	}
	
	// Check for existing position with same asset
	if existingPos, exists := p.AssetPositions[position.Asset.Symbol]; exists && existingPos.IsOpen() {
		return fmt.Errorf("portfolio already has an open position for asset %s", position.Asset.Symbol)
	}
	
	// Add position to both maps
	p.Positions[position.ID] = position
	p.AssetPositions[position.Asset.Symbol] = position
	
	p.UpdatedAt = time.Now()
	p.calculateMetrics()
	
	return nil
}

// RemovePosition removes a position from the portfolio
func (p *Portfolio) RemovePosition(positionID string) error {
	position, exists := p.Positions[positionID]
	if !exists {
		return fmt.Errorf("position %s not found in portfolio", positionID)
	}
	
	// Remove from both maps
	delete(p.Positions, positionID)
	delete(p.AssetPositions, position.Asset.Symbol)
	
	p.UpdatedAt = time.Now()
	p.calculateMetrics()
	
	return nil
}

// GetPosition retrieves a position by ID
func (p *Portfolio) GetPosition(positionID string) (*Position, bool) {
	position, exists := p.Positions[positionID]
	return position, exists
}

// GetPositionByAsset retrieves a position by asset symbol
func (p *Portfolio) GetPositionByAsset(assetSymbol string) (*Position, bool) {
	position, exists := p.AssetPositions[assetSymbol]
	return position, exists
}

// UpdatePositionPrice updates the market price for a specific asset
func (p *Portfolio) UpdatePositionPrice(assetSymbol string, newPrice types.Decimal) error {
	position, exists := p.AssetPositions[assetSymbol]
	if !exists {
		return fmt.Errorf("no position found for asset %s", assetSymbol)
	}
	
	if err := position.UpdateMarketPrice(newPrice); err != nil {
		return fmt.Errorf("failed to update position price: %w", err)
	}
	
	p.UpdatedAt = time.Now()
	p.calculateMetrics()
	
	return nil
}

// UpdateAllPrices updates market prices for all positions
func (p *Portfolio) UpdateAllPrices(prices map[string]types.Decimal) error {
	var errors []string
	
	for assetSymbol, price := range prices {
		if position, exists := p.AssetPositions[assetSymbol]; exists {
			if err := position.UpdateMarketPrice(price); err != nil {
				errors = append(errors, fmt.Sprintf("failed to update %s: %v", assetSymbol, err))
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("price update errors: %v", errors)
	}
	
	p.UpdatedAt = time.Now()
	p.calculateMetrics()
	
	return nil
}

// AddCash adds cash to the portfolio
func (p *Portfolio) AddCash(amount types.Decimal) error {
	if p.Status != PortfolioStatusActive {
		return fmt.Errorf("cannot add cash to %s portfolio", p.Status.String())
	}
	
	if !amount.IsPositive() {
		return fmt.Errorf("cash amount must be positive")
	}
	
	p.CashBalance = p.CashBalance.Add(amount)
	p.UpdatedAt = time.Now()
	p.calculateMetrics()
	
	return nil
}

// WithdrawCash removes cash from the portfolio
func (p *Portfolio) WithdrawCash(amount types.Decimal) error {
	if p.Status != PortfolioStatusActive {
		return fmt.Errorf("cannot withdraw cash from %s portfolio", p.Status.String())
	}
	
	if !amount.IsPositive() {
		return fmt.Errorf("withdrawal amount must be positive")
	}
	
	if amount.Cmp(p.CashBalance) > 0 {
		return fmt.Errorf("insufficient cash balance: have %s, need %s", 
			p.CashBalance.String(), amount.String())
	}
	
	p.CashBalance = p.CashBalance.Sub(amount)
	p.UpdatedAt = time.Now()
	p.calculateMetrics()
	
	return nil
}

// GetOpenPositions returns all open positions
func (p *Portfolio) GetOpenPositions() []*Position {
	var openPositions []*Position
	
	for _, position := range p.Positions {
		if position.IsOpen() {
			openPositions = append(openPositions, position)
		}
	}
	
	return openPositions
}

// GetClosedPositions returns all closed positions
func (p *Portfolio) GetClosedPositions() []*Position {
	var closedPositions []*Position
	
	for _, position := range p.Positions {
		if position.IsClosed() {
			closedPositions = append(closedPositions, position)
		}
	}
	
	return closedPositions
}

// GetPositionsByAssetType returns positions filtered by asset type
func (p *Portfolio) GetPositionsByAssetType(assetType AssetType) []*Position {
	var filteredPositions []*Position
	
	for _, position := range p.Positions {
		if position.Asset.AssetType == assetType {
			filteredPositions = append(filteredPositions, position)
		}
	}
	
	return filteredPositions
}

// GetTopPositions returns the largest positions by market value
func (p *Portfolio) GetTopPositions(count int) []*Position {
	openPositions := p.GetOpenPositions()
	
	// Sort by market value descending
	sort.Slice(openPositions, func(i, j int) bool {
		return openPositions[i].MarketValue.Cmp(openPositions[j].MarketValue) > 0
	})
	
	if count > len(openPositions) {
		count = len(openPositions)
	}
	
	return openPositions[:count]
}

// Suspend suspends the portfolio (no new positions allowed)
func (p *Portfolio) Suspend() error {
	if p.Status == PortfolioStatusClosed {
		return fmt.Errorf("cannot suspend closed portfolio")
	}
	
	p.Status = PortfolioStatusSuspended
	p.UpdatedAt = time.Now()
	
	return nil
}

// Resume resumes a suspended portfolio
func (p *Portfolio) Resume() error {
	if p.Status != PortfolioStatusSuspended {
		return fmt.Errorf("can only resume suspended portfolio, current status: %s", p.Status.String())
	}
	
	p.Status = PortfolioStatusActive
	p.UpdatedAt = time.Now()
	
	return nil
}

// Close closes the portfolio
func (p *Portfolio) Close() error {
	if p.Status == PortfolioStatusClosed {
		return fmt.Errorf("portfolio is already closed")
	}
	
	p.Status = PortfolioStatusClosed
	now := time.Now()
	p.ClosedAt = &now
	p.UpdatedAt = now
	
	return nil
}

// calculateMetrics recalculates all portfolio metrics
func (p *Portfolio) calculateMetrics() {
	now := time.Now()
	
	// Reset metrics
	metrics := PortfolioMetrics{
		CashBalance: p.CashBalance,
		UpdatedAt:   now,
	}
	
	// Calculate position-based metrics
	var totalMarketValue types.Decimal = types.Zero()
	var totalCost types.Decimal = types.Zero()
	var totalRealizedPnL types.Decimal = types.Zero()
	var totalUnrealizedPnL types.Decimal = types.Zero()
	var totalFees types.Decimal = types.Zero()
	
	var totalPositions, longPositions, shortPositions int
	var profitablePositions, losingPositions int
	var maxPositionValue types.Decimal = types.Zero()
	
	for _, position := range p.Positions {
		// Count positions
		if position.IsOpen() {
			totalPositions++
			if position.IsLong() {
				longPositions++
			} else if position.IsShort() {
				shortPositions++
			}
			
			// Market value and cost
			totalMarketValue = totalMarketValue.Add(position.MarketValue)
			totalCost = totalCost.Add(position.CostBasis)
			
			// Track largest position for concentration risk
			if position.MarketValue.Cmp(maxPositionValue) > 0 {
				maxPositionValue = position.MarketValue
			}
		}
		
		// P&L metrics (include closed positions)
		totalRealizedPnL = totalRealizedPnL.Add(position.RealizedPnL)
		totalUnrealizedPnL = totalUnrealizedPnL.Add(position.UnrealizedPnL)
		totalFees = totalFees.Add(position.TotalFees)
		
		// Profitable/losing positions
		totalPnL := position.TotalPnL()
		if totalPnL.IsPositive() {
			profitablePositions++
		} else if totalPnL.IsNegative() {
			losingPositions++
		}
	}
	
	// Portfolio totals
	metrics.MarketValue = totalMarketValue
	metrics.TotalCost = totalCost
	metrics.TotalValue = p.CashBalance.Add(totalMarketValue)
	metrics.RealizedPnL = totalRealizedPnL
	metrics.UnrealizedPnL = totalUnrealizedPnL
	metrics.TotalPnL = totalRealizedPnL.Add(totalUnrealizedPnL)
	metrics.TotalFees = totalFees
	metrics.NetPnL = metrics.TotalPnL.Sub(totalFees)
	
	// Return percentage
	if p.InitialCapital.IsPositive() {
		metrics.ReturnPercentage = metrics.NetPnL.Div(p.InitialCapital).Mul(types.NewDecimalFromInt(100))
	}
	
	// Position counts
	metrics.TotalPositions = totalPositions
	metrics.LongPositions = longPositions
	metrics.ShortPositions = shortPositions
	metrics.ProfitablePositions = profitablePositions
	metrics.LosingPositions = losingPositions
	
	// Concentration risk
	if totalMarketValue.IsPositive() {
		metrics.MaxPositionWeight = maxPositionValue.Div(totalMarketValue).Mul(types.NewDecimalFromInt(100))
		// Concentration risk is the sum of squares of position weights
		concentrationRisk := types.Zero()
		for _, position := range p.Positions {
			if position.IsOpen() && totalMarketValue.IsPositive() {
				weight := position.MarketValue.Div(totalMarketValue)
				concentrationRisk = concentrationRisk.Add(weight.Mul(weight))
			}
		}
		metrics.ConcentrationRisk = concentrationRisk.Mul(types.NewDecimalFromInt(100))
	}
	
	// Update peak value and drawdown
	if p.Metrics.PeakValue.IsZero() {
		metrics.PeakValue = metrics.TotalValue
	} else {
		if metrics.TotalValue.Cmp(p.Metrics.PeakValue) > 0 {
			metrics.PeakValue = metrics.TotalValue
		} else {
			metrics.PeakValue = p.Metrics.PeakValue
		}
	}
	
	// Calculate drawdown
	if metrics.PeakValue.IsPositive() {
		metrics.CurrentDrawdown = metrics.PeakValue.Sub(metrics.TotalValue)
		
		// Initialize MaxDrawdown if it's zero (first calculation)
		if p.Metrics.MaxDrawdown.IsZero() {
			metrics.MaxDrawdown = metrics.CurrentDrawdown
		} else {
			metrics.MaxDrawdown = p.Metrics.MaxDrawdown
			if metrics.CurrentDrawdown.Cmp(metrics.MaxDrawdown) > 0 {
				metrics.MaxDrawdown = metrics.CurrentDrawdown
			}
		}
		
		metrics.MaxDrawdownPercent = metrics.MaxDrawdown.Div(metrics.PeakValue).Mul(types.NewDecimalFromInt(100))
	}
	
	p.Metrics = metrics
}

// Rebalance performs portfolio rebalancing based on target weights
func (p *Portfolio) Rebalance(targetWeights map[string]types.Decimal) ([]RebalanceInstruction, error) {
	if p.Status != PortfolioStatusActive {
		return nil, fmt.Errorf("cannot rebalance %s portfolio", p.Status.String())
	}
	
	// Validate target weights sum to 100%
	totalWeight := types.Zero()
	for _, weight := range targetWeights {
		totalWeight = totalWeight.Add(weight)
	}
	
	hundredPercent := types.NewDecimalFromInt(100)
	tolerance := types.NewDecimalFromFloat(0.01) // 0.01% tolerance
	diff := totalWeight.Sub(hundredPercent).Abs()
	
	if diff.Cmp(tolerance) > 0 {
		return nil, fmt.Errorf("target weights must sum to 100%%, got %s", totalWeight.String())
	}
	
	// Calculate current weights
	currentWeights := make(map[string]types.Decimal)
	totalValue := p.Metrics.TotalValue
	
	if totalValue.IsZero() || totalValue.IsNegative() {
		return nil, fmt.Errorf("portfolio value must be positive for rebalancing")
	}
	
	for symbol, position := range p.AssetPositions {
		if position.IsOpen() {
			currentWeight := position.MarketValue.Div(totalValue).Mul(hundredPercent)
			currentWeights[symbol] = currentWeight
		}
	}
	
	// Generate rebalance instructions
	var instructions []RebalanceInstruction
	
	for symbol, targetWeight := range targetWeights {
		currentWeight, exists := currentWeights[symbol]
		if !exists {
			currentWeight = types.Zero()
		}
		weightDiff := targetWeight.Sub(currentWeight)
		
		// Only create instruction if difference is significant
		minDiff := types.NewDecimalFromFloat(0.5) // 0.5% minimum difference
		if weightDiff.Abs().Cmp(minDiff) > 0 {
			targetValue := totalValue.Mul(targetWeight).Div(hundredPercent)
			
			var currentValue types.Decimal = types.Zero()
			if position, exists := p.AssetPositions[symbol]; exists && position.IsOpen() {
				currentValue = position.MarketValue
			}
			
			valueDiff := targetValue.Sub(currentValue)
			
			instruction := RebalanceInstruction{
				AssetSymbol:   symbol,
				CurrentWeight: currentWeight,
				TargetWeight:  targetWeight,
				WeightDiff:    weightDiff,
				CurrentValue:  currentValue,
				TargetValue:   targetValue,
				ValueDiff:     valueDiff,
				Action:        determineRebalanceAction(valueDiff),
			}
			
			instructions = append(instructions, instruction)
		}
	}
	
	// Sort by absolute value difference (largest first)
	sort.Slice(instructions, func(i, j int) bool {
		return instructions[i].ValueDiff.Abs().Cmp(instructions[j].ValueDiff.Abs()) > 0
	})
	
	return instructions, nil
}

// RebalanceInstruction represents an instruction for portfolio rebalancing
type RebalanceInstruction struct {
	AssetSymbol   string        `json:"asset_symbol"`
	CurrentWeight types.Decimal `json:"current_weight"`
	TargetWeight  types.Decimal `json:"target_weight"`
	WeightDiff    types.Decimal `json:"weight_diff"`
	CurrentValue  types.Decimal `json:"current_value"`
	TargetValue   types.Decimal `json:"target_value"`
	ValueDiff     types.Decimal `json:"value_diff"`
	Action        string        `json:"action"`
}

func determineRebalanceAction(valueDiff types.Decimal) string {
	if valueDiff.IsPositive() {
		return "BUY"
	} else if valueDiff.IsNegative() {
		return "SELL"
	}
	return "HOLD"
}

// Utility methods

// IsActive returns true if the portfolio is active
func (p *Portfolio) IsActive() bool {
	return p.Status == PortfolioStatusActive
}

// IsSuspended returns true if the portfolio is suspended
func (p *Portfolio) IsSuspended() bool {
	return p.Status == PortfolioStatusSuspended
}

// IsClosed returns true if the portfolio is closed
func (p *Portfolio) IsClosed() bool {
	return p.Status == PortfolioStatusClosed
}

// GetAge returns how long the portfolio has existed
func (p *Portfolio) GetAge() time.Duration {
	endTime := time.Now()
	if p.ClosedAt != nil {
		endTime = *p.ClosedAt
	}
	return endTime.Sub(p.CreatedAt)
}

// GetDiversificationRatio calculates the portfolio diversification ratio
func (p *Portfolio) GetDiversificationRatio() types.Decimal {
	openPositions := p.GetOpenPositions()
	numPositions := len(openPositions)
	
	if numPositions <= 1 {
		return types.Zero()
	}
	
	// Simple diversification: 1 - HHI (Herfindahl-Hirschman Index)
	// HHI = sum of squared weights
	hhi := p.Metrics.ConcentrationRisk.Div(types.NewDecimalFromInt(100))
	return types.NewDecimalFromInt(1).Sub(hhi)
}

// Validate performs comprehensive portfolio validation
func (p *Portfolio) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("portfolio ID is required")
	}
	
	if p.Name == "" {
		return fmt.Errorf("portfolio name is required")
	}
	
	if p.CashBalance.IsNegative() {
		return fmt.Errorf("cash balance cannot be negative")
	}
	
	if !p.InitialCapital.IsPositive() {
		return fmt.Errorf("initial capital must be positive")
	}
	
	// Validate all positions
	for _, position := range p.Positions {
		if err := position.Validate(); err != nil {
			return fmt.Errorf("invalid position %s: %w", position.ID, err)
		}
	}
	
	// Check consistency between position maps
	for positionID, position := range p.Positions {
		assetPosition, exists := p.AssetPositions[position.Asset.Symbol]
		if !exists {
			return fmt.Errorf("position %s not found in asset positions map", positionID)
		}
		
		if assetPosition.ID != position.ID {
			return fmt.Errorf("position ID mismatch in asset positions map")
		}
	}
	
	return nil
}

// Clone creates a deep copy of the portfolio
func (p *Portfolio) Clone() *Portfolio {
	clone := *p
	
	// Deep copy positions maps
	clone.Positions = make(map[string]*Position)
	clone.AssetPositions = make(map[string]*Position)
	
	for id, position := range p.Positions {
		clonedPosition := position.Clone()
		clone.Positions[id] = clonedPosition
		clone.AssetPositions[position.Asset.Symbol] = clonedPosition
	}
	
	// Copy time pointers
	if p.ClosedAt != nil {
		closedAt := *p.ClosedAt
		clone.ClosedAt = &closedAt
	}
	
	return &clone
}