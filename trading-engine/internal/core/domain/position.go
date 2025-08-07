package domain

import (
	"fmt"
	"time"

	"github.com/trading-engine/pkg/types"
)

// PositionSide represents whether the position is long or short
type PositionSide int

const (
	PositionSideLong PositionSide = iota + 1
	PositionSideShort
	PositionSideFlat // No position
)

func (ps PositionSide) String() string {
	switch ps {
	case PositionSideLong:
		return "LONG"
	case PositionSideShort:
		return "SHORT"
	case PositionSideFlat:
		return "FLAT"
	default:
		return "UNKNOWN"
	}
}

// PositionStatus represents the current state of the position
type PositionStatus int

const (
	PositionStatusOpen PositionStatus = iota + 1
	PositionStatusClosed
	PositionStatusClosing
)

func (ps PositionStatus) String() string {
	switch ps {
	case PositionStatusOpen:
		return "OPEN"
	case PositionStatusClosed:
		return "CLOSED"
	case PositionStatusClosing:
		return "CLOSING"
	default:
		return "UNKNOWN"
	}
}

// PositionTransaction represents a transaction that affects position
type PositionTransaction struct {
	ID          string        `json:"id"`
	PositionID  string        `json:"position_id"`
	OrderID     string        `json:"order_id"`
	OrderFillID string        `json:"order_fill_id"`
	Type        string        `json:"type"` // "BUY", "SELL"
	Quantity    types.Decimal `json:"quantity"`
	Price       types.Decimal `json:"price"`
	Fee         types.Decimal `json:"fee"`
	Timestamp   time.Time     `json:"timestamp"`
	RealizedPnL types.Decimal `json:"realized_pnl,omitempty"`
}

// Position represents a trading position with P&L tracking
type Position struct {
	ID            string                `json:"id"`
	Asset         *Asset                `json:"asset"`
	Side          PositionSide          `json:"side"`
	Status        PositionStatus        `json:"status"`
	Quantity      types.Decimal         `json:"quantity"`
	AvgEntryPrice types.Decimal         `json:"avg_entry_price"`
	CurrentPrice  types.Decimal         `json:"current_price"`
	UnrealizedPnL types.Decimal         `json:"unrealized_pnl"`
	RealizedPnL   types.Decimal         `json:"realized_pnl"`
	TotalFees     types.Decimal         `json:"total_fees"`
	MarketValue   types.Decimal         `json:"market_value"`
	CostBasis     types.Decimal         `json:"cost_basis"`
	Transactions  []PositionTransaction `json:"transactions"`
	OpenedAt      time.Time             `json:"opened_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	ClosedAt      *time.Time            `json:"closed_at,omitempty"`

	// Risk metrics
	MaxUnrealizedPnL types.Decimal `json:"max_unrealized_pnl"`
	MinUnrealizedPnL types.Decimal `json:"min_unrealized_pnl"`
	MaxDrawdown      types.Decimal `json:"max_drawdown"`
}

// NewPosition creates a new position from an initial transaction
func NewPosition(id string, asset *Asset, initialTransaction PositionTransaction) (*Position, error) {
	if id == "" {
		return nil, fmt.Errorf("position ID cannot be empty")
	}

	if asset == nil {
		return nil, fmt.Errorf("asset cannot be nil")
	}

	if err := asset.Validate(); err != nil {
		return nil, fmt.Errorf("invalid asset: %w", err)
	}

	if !initialTransaction.Quantity.IsPositive() {
		return nil, fmt.Errorf("initial transaction quantity must be positive")
	}

	if !initialTransaction.Price.IsPositive() {
		return nil, fmt.Errorf("initial transaction price must be positive")
	}

	// Determine position side from transaction type
	var side PositionSide
	switch initialTransaction.Type {
	case "BUY":
		side = PositionSideLong
	case "SELL":
		side = PositionSideShort
	default:
		return nil, fmt.Errorf("invalid transaction type: %s", initialTransaction.Type)
	}

	now := time.Now()
	initialTransaction.PositionID = id
	if initialTransaction.Timestamp.IsZero() {
		initialTransaction.Timestamp = now
	}

	position := &Position{
		ID:               id,
		Asset:            asset,
		Side:             side,
		Status:           PositionStatusOpen,
		Quantity:         initialTransaction.Quantity,
		AvgEntryPrice:    initialTransaction.Price,
		CurrentPrice:     initialTransaction.Price,
		UnrealizedPnL:    types.Zero(),
		RealizedPnL:      types.Zero(),
		TotalFees:        initialTransaction.Fee,
		CostBasis:        initialTransaction.Price.Mul(initialTransaction.Quantity).Add(initialTransaction.Fee),
		MarketValue:      initialTransaction.Price.Mul(initialTransaction.Quantity),
		Transactions:     []PositionTransaction{initialTransaction},
		OpenedAt:         now,
		UpdatedAt:        now,
		MaxUnrealizedPnL: types.Zero(),
		MinUnrealizedPnL: types.Zero(),
		MaxDrawdown:      types.Zero(),
	}

	// Update market value and unrealized P&L
	position.UpdateMarketPrice(initialTransaction.Price)

	return position, nil
}

// AddTransaction processes a new transaction for the position
func (p *Position) AddTransaction(transaction PositionTransaction) error {
	if p.Status == PositionStatusClosed {
		return fmt.Errorf("cannot add transaction to closed position")
	}

	if !transaction.Quantity.IsPositive() {
		return fmt.Errorf("transaction quantity must be positive")
	}

	if !transaction.Price.IsPositive() {
		return fmt.Errorf("transaction price must be positive")
	}

	if transaction.Type != "BUY" && transaction.Type != "SELL" {
		return fmt.Errorf("invalid transaction type: %s", transaction.Type)
	}

	// Set transaction metadata
	transaction.PositionID = p.ID
	if transaction.Timestamp.IsZero() {
		transaction.Timestamp = time.Now()
	}

	// Process transaction based on type and current position
	if err := p.processTransaction(transaction); err != nil {
		return err
	}

	// Add to transaction history
	p.Transactions = append(p.Transactions, transaction)
	p.TotalFees = p.TotalFees.Add(transaction.Fee)
	p.UpdatedAt = time.Now()

	return nil
}

// processTransaction handles the transaction logic
func (p *Position) processTransaction(transaction PositionTransaction) error {
	isIncreasingPosition := (p.Side == PositionSideLong && transaction.Type == "BUY") ||
		(p.Side == PositionSideShort && transaction.Type == "SELL")

	if isIncreasingPosition {
		// Adding to position - update average entry price
		return p.increasePosition(transaction)
	} else {
		// Reducing position - calculate realized P&L
		return p.reducePosition(transaction)
	}
}

// increasePosition adds to the existing position
func (p *Position) increasePosition(transaction PositionTransaction) error {
	// Calculate new average entry price using weighted average
	currentValue := p.AvgEntryPrice.Mul(p.Quantity)
	newValue := transaction.Price.Mul(transaction.Quantity)
	totalValue := currentValue.Add(newValue)
	totalQuantity := p.Quantity.Add(transaction.Quantity)

	p.AvgEntryPrice = totalValue.Div(totalQuantity)
	p.Quantity = totalQuantity
	p.CostBasis = p.CostBasis.Add(transaction.Price.Mul(transaction.Quantity)).Add(transaction.Fee)

	return nil
}

// reducePosition reduces the existing position and calculates realized P&L
func (p *Position) reducePosition(transaction PositionTransaction) error {
	if transaction.Quantity.Cmp(p.Quantity) > 0 {
		return fmt.Errorf("cannot reduce position by %s shares - only %s shares available",
			transaction.Quantity.String(), p.Quantity.String())
	}

	// Calculate realized P&L for the portion being closed
	var realizedPnL types.Decimal
	if p.Side == PositionSideLong {
		// Long position: PnL = (sell_price - avg_entry_price) * quantity_sold - fees
		pnlPerShare := transaction.Price.Sub(p.AvgEntryPrice)
		realizedPnL = pnlPerShare.Mul(transaction.Quantity).Sub(transaction.Fee)
	} else {
		// Short position: PnL = (avg_entry_price - buy_price) * quantity_covered - fees
		pnlPerShare := p.AvgEntryPrice.Sub(transaction.Price)
		realizedPnL = pnlPerShare.Mul(transaction.Quantity).Sub(transaction.Fee)
	}

	transaction.RealizedPnL = realizedPnL
	p.RealizedPnL = p.RealizedPnL.Add(realizedPnL)
	p.Quantity = p.Quantity.Sub(transaction.Quantity)

	// Adjust cost basis proportionally
	costBasisReduction := p.CostBasis.Mul(transaction.Quantity).Div(p.Quantity.Add(transaction.Quantity))
	p.CostBasis = p.CostBasis.Sub(costBasisReduction)

	// Check if position is fully closed
	if p.Quantity.IsZero() {
		p.Status = PositionStatusClosed
		p.Side = PositionSideFlat
		now := time.Now()
		p.ClosedAt = &now
		p.UnrealizedPnL = types.Zero()
		p.MarketValue = types.Zero()
	}

	return nil
}

// UpdateMarketPrice updates the current market price and recalculates metrics
func (p *Position) UpdateMarketPrice(newPrice types.Decimal) error {
	if !newPrice.IsPositive() {
		return fmt.Errorf("market price must be positive")
	}

	p.CurrentPrice = newPrice

	if p.Status == PositionStatusClosed {
		p.MarketValue = types.Zero()
		p.UnrealizedPnL = types.Zero()
		return nil
	}

	// Update market value
	p.MarketValue = p.CurrentPrice.Mul(p.Quantity)

	// Calculate unrealized P&L
	if p.Side == PositionSideLong {
		// Long: (current_price - avg_entry_price) * quantity
		pnlPerShare := p.CurrentPrice.Sub(p.AvgEntryPrice)
		p.UnrealizedPnL = pnlPerShare.Mul(p.Quantity)
	} else if p.Side == PositionSideShort {
		// Short: (avg_entry_price - current_price) * quantity
		pnlPerShare := p.AvgEntryPrice.Sub(p.CurrentPrice)
		p.UnrealizedPnL = pnlPerShare.Mul(p.Quantity)
	}

	// Update risk metrics
	p.updateRiskMetrics()
	p.UpdatedAt = time.Now()

	return nil
}

// updateRiskMetrics updates position risk tracking metrics
func (p *Position) updateRiskMetrics() {
	// Track maximum and minimum unrealized P&L
	if p.UnrealizedPnL.Cmp(p.MaxUnrealizedPnL) > 0 {
		p.MaxUnrealizedPnL = p.UnrealizedPnL
	}

	if p.UnrealizedPnL.Cmp(p.MinUnrealizedPnL) < 0 {
		p.MinUnrealizedPnL = p.UnrealizedPnL
	}

	// Calculate max drawdown from peak
	if p.MaxUnrealizedPnL.IsPositive() {
		drawdown := p.MaxUnrealizedPnL.Sub(p.UnrealizedPnL)
		if drawdown.Cmp(p.MaxDrawdown) > 0 {
			p.MaxDrawdown = drawdown
		}
	}
}

// Close initiates position closing process
func (p *Position) Close() error {
	if p.Status == PositionStatusClosed {
		return fmt.Errorf("position is already closed")
	}

	if p.Status == PositionStatusClosing {
		return fmt.Errorf("position is already closing")
	}

	p.Status = PositionStatusClosing
	p.UpdatedAt = time.Now()

	return nil
}

// ForceClose immediately closes the position (for emergencies)
func (p *Position) ForceClose() error {
	p.Status = PositionStatusClosed
	p.Side = PositionSideFlat
	now := time.Now()
	p.ClosedAt = &now
	p.UpdatedAt = now

	// Move unrealized P&L to realized P&L
	p.RealizedPnL = p.RealizedPnL.Add(p.UnrealizedPnL)
	p.UnrealizedPnL = types.Zero()
	p.MarketValue = types.Zero()
	p.Quantity = types.Zero()

	return nil
}

// Utility methods

// TotalPnL returns the total P&L (realized + unrealized)
func (p *Position) TotalPnL() types.Decimal {
	return p.RealizedPnL.Add(p.UnrealizedPnL)
}

// NetPnL returns the total P&L minus fees
func (p *Position) NetPnL() types.Decimal {
	return p.TotalPnL().Sub(p.TotalFees)
}

// IsLong returns true if the position is long
func (p *Position) IsLong() bool {
	return p.Side == PositionSideLong
}

// IsShort returns true if the position is short
func (p *Position) IsShort() bool {
	return p.Side == PositionSideShort
}

// IsFlat returns true if there is no position
func (p *Position) IsFlat() bool {
	return p.Side == PositionSideFlat || p.Quantity.IsZero()
}

// IsOpen returns true if the position is currently open
func (p *Position) IsOpen() bool {
	return p.Status == PositionStatusOpen && !p.IsFlat()
}

// IsClosed returns true if the position is closed
func (p *Position) IsClosed() bool {
	return p.Status == PositionStatusClosed
}

// GetHoldingPeriod returns how long the position has been held
func (p *Position) GetHoldingPeriod() time.Duration {
	endTime := time.Now()
	if p.ClosedAt != nil {
		endTime = *p.ClosedAt
	}
	return endTime.Sub(p.OpenedAt)
}

// GetPnLPercentage returns the P&L as a percentage of cost basis
func (p *Position) GetPnLPercentage() types.Decimal {
	if p.CostBasis.IsZero() {
		return types.Zero()
	}

	return p.TotalPnL().Div(p.CostBasis).Mul(types.NewDecimalFromInt(100))
}

// GetROI returns the return on investment percentage
func (p *Position) GetROI() types.Decimal {
	if p.CostBasis.IsZero() {
		return types.Zero()
	}

	return p.NetPnL().Div(p.CostBasis).Mul(types.NewDecimalFromInt(100))
}

// GetDrawdownPercentage returns the maximum drawdown as percentage
func (p *Position) GetDrawdownPercentage() types.Decimal {
	if p.MaxUnrealizedPnL.IsZero() || p.MaxUnrealizedPnL.IsNegative() {
		return types.Zero()
	}

	return p.MaxDrawdown.Div(p.MaxUnrealizedPnL).Mul(types.NewDecimalFromInt(100))
}

// Validate performs comprehensive position validation
func (p *Position) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("position ID is required")
	}

	if p.Asset == nil {
		return fmt.Errorf("asset is required")
	}

	if err := p.Asset.Validate(); err != nil {
		return fmt.Errorf("invalid asset: %w", err)
	}

	if p.Status == PositionStatusOpen && p.Quantity.IsNegative() {
		return fmt.Errorf("position quantity cannot be negative")
	}

	if p.Status == PositionStatusClosed && !p.Quantity.IsZero() {
		return fmt.Errorf("closed position must have zero quantity")
	}

	if !p.AvgEntryPrice.IsZero() && p.AvgEntryPrice.IsNegative() {
		return fmt.Errorf("average entry price cannot be negative")
	}

	if !p.CurrentPrice.IsZero() && p.CurrentPrice.IsNegative() {
		return fmt.Errorf("current price cannot be negative")
	}

	if p.TotalFees.IsNegative() {
		return fmt.Errorf("total fees cannot be negative")
	}

	if p.CostBasis.IsNegative() {
		return fmt.Errorf("cost basis cannot be negative")
	}

	if p.MarketValue.IsNegative() {
		return fmt.Errorf("market value cannot be negative")
	}

	// Validate transaction consistency
	for _, tx := range p.Transactions {
		if tx.PositionID != p.ID {
			return fmt.Errorf("transaction position ID %s does not match position ID %s",
				tx.PositionID, p.ID)
		}
	}

	return nil
}

// Clone creates a deep copy of the position
func (p *Position) Clone() *Position {
	clone := *p

	// Deep copy transactions slice
	clone.Transactions = make([]PositionTransaction, len(p.Transactions))
	copy(clone.Transactions, p.Transactions)

	// Copy time pointers
	if p.ClosedAt != nil {
		closedAt := *p.ClosedAt
		clone.ClosedAt = &closedAt
	}

	return &clone
}
