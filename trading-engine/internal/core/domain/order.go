package domain

import (
	"fmt"
	"time"

	"github.com/trading-engine/pkg/types"
)

// OrderType represents the type of order
type OrderType int

const (
	OrderTypeMarket OrderType = iota + 1
	OrderTypeLimit
	OrderTypeStop
	OrderTypeStopLimit
	OrderTypeTrailingStop
)

func (ot OrderType) String() string {
	switch ot {
	case OrderTypeMarket:
		return "MARKET"
	case OrderTypeLimit:
		return "LIMIT"
	case OrderTypeStop:
		return "STOP"
	case OrderTypeStopLimit:
		return "STOP_LIMIT"
	case OrderTypeTrailingStop:
		return "TRAILING_STOP"
	default:
		return "UNKNOWN"
	}
}

// OrderSide represents whether the order is buying or selling
type OrderSide int

const (
	OrderSideBuy OrderSide = iota + 1
	OrderSideSell
)

func (os OrderSide) String() string {
	switch os {
	case OrderSideBuy:
		return "BUY"
	case OrderSideSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

// OrderStatus represents the current state of the order
type OrderStatus int

const (
	OrderStatusPending OrderStatus = iota + 1
	OrderStatusSubmitted
	OrderStatusPartiallyFilled
	OrderStatusFilled
	OrderStatusCancelled
	OrderStatusRejected
	OrderStatusExpired
)

func (os OrderStatus) String() string {
	switch os {
	case OrderStatusPending:
		return "PENDING"
	case OrderStatusSubmitted:
		return "SUBMITTED"
	case OrderStatusPartiallyFilled:
		return "PARTIALLY_FILLED"
	case OrderStatusFilled:
		return "FILLED"
	case OrderStatusCancelled:
		return "CANCELLED"
	case OrderStatusRejected:
		return "REJECTED"
	case OrderStatusExpired:
		return "EXPIRED"
	default:
		return "UNKNOWN"
	}
}

// TimeInForce represents how long the order remains active
type TimeInForce int

const (
	TimeInForceGTC TimeInForce = iota + 1 // Good Till Cancelled
	TimeInForceIOC                        // Immediate Or Cancel
	TimeInForceFOK                        // Fill Or Kill
	TimeInForceDAY                        // Day
	TimeInForceGTD                        // Good Till Date
)

func (tif TimeInForce) String() string {
	switch tif {
	case TimeInForceGTC:
		return "GTC"
	case TimeInForceIOC:
		return "IOC"
	case TimeInForceFOK:
		return "FOK"
	case TimeInForceDAY:
		return "DAY"
	case TimeInForceGTD:
		return "GTD"
	default:
		return "UNKNOWN"
	}
}

// OrderFill represents a partial or complete fill of an order
type OrderFill struct {
	ID        string        `json:"id"`
	OrderID   string        `json:"order_id"`
	Price     types.Decimal `json:"price"`
	Quantity  types.Decimal `json:"quantity"`
	Fee       types.Decimal `json:"fee"`
	Timestamp time.Time     `json:"timestamp"`
}

// Order represents a trading order with complete state management
type Order struct {
	ID             string        `json:"id"`
	ClientOrderID  string        `json:"client_order_id"`
	Asset          *Asset        `json:"asset"`
	Type           OrderType     `json:"type"`
	Side           OrderSide     `json:"side"`
	Status         OrderStatus   `json:"status"`
	Quantity       types.Decimal `json:"quantity"`
	Price          types.Decimal `json:"price,omitempty"`           // Not used for market orders
	StopPrice      types.Decimal `json:"stop_price,omitempty"`      // Used for stop orders
	TrailingAmount types.Decimal `json:"trailing_amount,omitempty"` // Used for trailing stop
	FilledQuantity types.Decimal `json:"filled_quantity"`
	AvgFillPrice   types.Decimal `json:"avg_fill_price"`
	TimeInForce    TimeInForce   `json:"time_in_force"`
	ExpiresAt      *time.Time    `json:"expires_at,omitempty"`
	Fills          []OrderFill   `json:"fills"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	SubmittedAt    *time.Time    `json:"submitted_at,omitempty"`
	FilledAt       *time.Time    `json:"filled_at,omitempty"`
	CancelledAt    *time.Time    `json:"cancelled_at,omitempty"`
}

// OrderBuilder provides a fluent interface for creating orders
type OrderBuilder struct {
	order Order
	err   error
}

// NewOrderBuilder creates a new order builder
func NewOrderBuilder() *OrderBuilder {
	return &OrderBuilder{
		order: Order{
			Status:         OrderStatusPending,
			FilledQuantity: types.Zero(),
			AvgFillPrice:   types.Zero(),
			TimeInForce:    TimeInForceGTC,
			Fills:          make([]OrderFill, 0),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

// ID sets the order ID
func (b *OrderBuilder) ID(id string) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if id == "" {
		b.err = fmt.Errorf("order ID cannot be empty")
		return b
	}

	b.order.ID = id
	return b
}

// ClientOrderID sets the client order ID
func (b *OrderBuilder) ClientOrderID(clientID string) *OrderBuilder {
	if b.err != nil {
		return b
	}

	b.order.ClientOrderID = clientID
	return b
}

// Asset sets the asset for the order
func (b *OrderBuilder) Asset(asset *Asset) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if asset == nil {
		b.err = fmt.Errorf("asset cannot be nil")
		return b
	}

	if err := asset.Validate(); err != nil {
		b.err = fmt.Errorf("invalid asset: %w", err)
		return b
	}

	b.order.Asset = asset
	return b
}

// Type sets the order type
func (b *OrderBuilder) Type(orderType OrderType) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if orderType < OrderTypeMarket || orderType > OrderTypeTrailingStop {
		b.err = fmt.Errorf("invalid order type: %d", orderType)
		return b
	}

	b.order.Type = orderType
	return b
}

// Side sets the order side
func (b *OrderBuilder) Side(side OrderSide) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if side != OrderSideBuy && side != OrderSideSell {
		b.err = fmt.Errorf("invalid order side: %d", side)
		return b
	}

	b.order.Side = side
	return b
}

// Quantity sets the order quantity
func (b *OrderBuilder) Quantity(qty types.Decimal) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if !qty.IsPositive() {
		b.err = fmt.Errorf("quantity must be positive")
		return b
	}

	b.order.Quantity = qty
	return b
}

// Price sets the limit price
func (b *OrderBuilder) Price(price types.Decimal) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if !price.IsPositive() {
		b.err = fmt.Errorf("price must be positive")
		return b
	}

	b.order.Price = price
	return b
}

// StopPrice sets the stop price
func (b *OrderBuilder) StopPrice(stopPrice types.Decimal) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if !stopPrice.IsPositive() {
		b.err = fmt.Errorf("stop price must be positive")
		return b
	}

	b.order.StopPrice = stopPrice
	return b
}

// TrailingAmount sets the trailing stop amount
func (b *OrderBuilder) TrailingAmount(amount types.Decimal) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if !amount.IsPositive() {
		b.err = fmt.Errorf("trailing amount must be positive")
		return b
	}

	b.order.TrailingAmount = amount
	return b
}

// TimeInForce sets the time in force
func (b *OrderBuilder) TimeInForce(tif TimeInForce) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if tif < TimeInForceGTC || tif > TimeInForceGTD {
		b.err = fmt.Errorf("invalid time in force: %d", tif)
		return b
	}

	b.order.TimeInForce = tif
	return b
}

// ExpiresAt sets the expiration time
func (b *OrderBuilder) ExpiresAt(expiresAt time.Time) *OrderBuilder {
	if b.err != nil {
		return b
	}

	if expiresAt.Before(time.Now()) {
		b.err = fmt.Errorf("expiration time must be in the future")
		return b
	}

	b.order.ExpiresAt = &expiresAt
	return b
}

// Build creates the order with validation
func (b *OrderBuilder) Build() (*Order, error) {
	if b.err != nil {
		return nil, b.err
	}

	// Validate required fields
	if b.order.ID == "" {
		return nil, fmt.Errorf("order ID is required")
	}

	if b.order.Asset == nil {
		return nil, fmt.Errorf("asset is required")
	}

	if b.order.Quantity.IsZero() || b.order.Quantity.IsNegative() {
		return nil, fmt.Errorf("quantity must be positive")
	}

	// Validate quantity constraints for the asset
	if !b.order.Asset.IsValidQuantity(b.order.Quantity) {
		return nil, fmt.Errorf("quantity %s is not valid for asset %s",
			b.order.Quantity.String(), b.order.Asset.Symbol)
	}

	// Type-specific validations
	switch b.order.Type {
	case OrderTypeLimit:
		if b.order.Price.IsZero() {
			return nil, fmt.Errorf("limit orders require a price")
		}
		// Round price to asset's tick size
		b.order.Price = b.order.Asset.RoundPrice(b.order.Price)

	case OrderTypeStopLimit:
		if b.order.Price.IsZero() {
			return nil, fmt.Errorf("stop limit orders require a price")
		}
		if b.order.StopPrice.IsZero() {
			return nil, fmt.Errorf("stop limit orders require a stop price")
		}
		b.order.Price = b.order.Asset.RoundPrice(b.order.Price)
		b.order.StopPrice = b.order.Asset.RoundPrice(b.order.StopPrice)

	case OrderTypeStop:
		if b.order.StopPrice.IsZero() {
			return nil, fmt.Errorf("stop orders require a stop price")
		}
		b.order.StopPrice = b.order.Asset.RoundPrice(b.order.StopPrice)

	case OrderTypeTrailingStop:
		if b.order.TrailingAmount.IsZero() {
			return nil, fmt.Errorf("trailing stop orders require a trailing amount")
		}

	case OrderTypeMarket:
		// Market orders don't need price validation

	default:
		return nil, fmt.Errorf("unknown order type: %s", b.order.Type.String())
	}

	// Time in force validations
	if b.order.TimeInForce == TimeInForceGTD && b.order.ExpiresAt == nil {
		return nil, fmt.Errorf("GTD orders require an expiration time")
	}

	if b.order.TimeInForce != TimeInForceGTD && b.order.ExpiresAt != nil {
		return nil, fmt.Errorf("only GTD orders can have expiration time")
	}

	return &b.order, nil
}

// State transition methods

// Submit transitions the order from pending to submitted
func (o *Order) Submit() error {
	if o.Status != OrderStatusPending {
		return fmt.Errorf("cannot submit order in status %s", o.Status.String())
	}

	now := time.Now()
	o.Status = OrderStatusSubmitted
	o.SubmittedAt = &now
	o.UpdatedAt = now

	return nil
}

// Fill processes a fill for the order
func (o *Order) Fill(fill OrderFill) error {
	if o.Status != OrderStatusSubmitted && o.Status != OrderStatusPartiallyFilled {
		return fmt.Errorf("cannot fill order in status %s", o.Status.String())
	}

	if !fill.Quantity.IsPositive() {
		return fmt.Errorf("fill quantity must be positive")
	}

	if !fill.Price.IsPositive() {
		return fmt.Errorf("fill price must be positive")
	}

	// Check that fill doesn't exceed remaining quantity
	remainingQty := o.Quantity.Sub(o.FilledQuantity)
	if fill.Quantity.Cmp(remainingQty) > 0 {
		return fmt.Errorf("fill quantity %s exceeds remaining quantity %s",
			fill.Quantity.String(), remainingQty.String())
	}

	// Calculate new average fill price
	totalValue := o.AvgFillPrice.Mul(o.FilledQuantity).Add(fill.Price.Mul(fill.Quantity))
	newFilledQty := o.FilledQuantity.Add(fill.Quantity)

	// Add fill to order
	fill.OrderID = o.ID
	if fill.Timestamp.IsZero() {
		fill.Timestamp = time.Now()
	}
	o.Fills = append(o.Fills, fill)

	// Update order state
	o.FilledQuantity = newFilledQty
	o.AvgFillPrice = totalValue.Div(newFilledQty)
	o.UpdatedAt = time.Now()

	// Update status
	if o.FilledQuantity.Cmp(o.Quantity) == 0 {
		o.Status = OrderStatusFilled
		now := time.Now()
		o.FilledAt = &now
	} else {
		o.Status = OrderStatusPartiallyFilled
	}

	return nil
}

// Cancel transitions the order to cancelled status
func (o *Order) Cancel() error {
	if o.Status == OrderStatusFilled {
		return fmt.Errorf("cannot cancel filled order")
	}

	if o.Status == OrderStatusCancelled {
		return fmt.Errorf("order is already cancelled")
	}

	if o.Status == OrderStatusRejected {
		return fmt.Errorf("cannot cancel rejected order")
	}

	now := time.Now()
	o.Status = OrderStatusCancelled
	o.CancelledAt = &now
	o.UpdatedAt = now

	return nil
}

// Reject transitions the order to rejected status
func (o *Order) Reject(reason string) error {
	if o.Status != OrderStatusPending && o.Status != OrderStatusSubmitted {
		return fmt.Errorf("cannot reject order in status %s", o.Status.String())
	}

	o.Status = OrderStatusRejected
	o.UpdatedAt = time.Now()

	return nil
}

// Expire transitions the order to expired status
func (o *Order) Expire() error {
	if o.Status == OrderStatusFilled || o.Status == OrderStatusCancelled {
		return fmt.Errorf("cannot expire order in status %s", o.Status.String())
	}

	o.Status = OrderStatusExpired
	o.UpdatedAt = time.Now()

	return nil
}

// Utility methods

// IsActive returns true if the order is still active
func (o *Order) IsActive() bool {
	return o.Status == OrderStatusSubmitted || o.Status == OrderStatusPartiallyFilled
}

// IsClosed returns true if the order is in a terminal state
func (o *Order) IsClosed() bool {
	return o.Status == OrderStatusFilled ||
		o.Status == OrderStatusCancelled ||
		o.Status == OrderStatusRejected ||
		o.Status == OrderStatusExpired
}

// RemainingQuantity returns the unfilled quantity
func (o *Order) RemainingQuantity() types.Decimal {
	return o.Quantity.Sub(o.FilledQuantity)
}

// TotalFees returns the total fees paid for all fills
func (o *Order) TotalFees() types.Decimal {
	total := types.Zero()
	for _, fill := range o.Fills {
		total = total.Add(fill.Fee)
	}
	return total
}

// IsExpired checks if the order has expired
func (o *Order) IsExpired() bool {
	if o.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*o.ExpiresAt)
}

// Validate performs comprehensive order validation
func (o *Order) Validate() error {
	if o.ID == "" {
		return fmt.Errorf("order ID is required")
	}

	if o.Asset == nil {
		return fmt.Errorf("asset is required")
	}

	if err := o.Asset.Validate(); err != nil {
		return fmt.Errorf("invalid asset: %w", err)
	}

	if o.Quantity.IsZero() || o.Quantity.IsNegative() {
		return fmt.Errorf("quantity must be positive")
	}

	if !o.Asset.IsValidQuantity(o.Quantity) {
		return fmt.Errorf("quantity %s is not valid for asset %s",
			o.Quantity.String(), o.Asset.Symbol)
	}

	if o.FilledQuantity.IsNegative() {
		return fmt.Errorf("filled quantity cannot be negative")
	}

	if o.FilledQuantity.Cmp(o.Quantity) > 0 {
		return fmt.Errorf("filled quantity cannot exceed order quantity")
	}

	// Type-specific validations
	switch o.Type {
	case OrderTypeLimit:
		if o.Price.IsZero() || o.Price.IsNegative() {
			return fmt.Errorf("limit orders require a positive price")
		}

	case OrderTypeStopLimit:
		if o.Price.IsZero() || o.Price.IsNegative() {
			return fmt.Errorf("stop limit orders require a positive price")
		}
		if o.StopPrice.IsZero() || o.StopPrice.IsNegative() {
			return fmt.Errorf("stop limit orders require a positive stop price")
		}

	case OrderTypeStop:
		if o.StopPrice.IsZero() || o.StopPrice.IsNegative() {
			return fmt.Errorf("stop orders require a positive stop price")
		}

	case OrderTypeTrailingStop:
		if o.TrailingAmount.IsZero() || o.TrailingAmount.IsNegative() {
			return fmt.Errorf("trailing stop orders require a positive trailing amount")
		}
	}

	// Validate fills consistency
	totalFilled := types.Zero()
	for _, fill := range o.Fills {
		if fill.OrderID != o.ID {
			return fmt.Errorf("fill order ID %s does not match order ID %s",
				fill.OrderID, o.ID)
		}
		totalFilled = totalFilled.Add(fill.Quantity)
	}

	if totalFilled.Cmp(o.FilledQuantity) != 0 {
		return fmt.Errorf("sum of fills %s does not match filled quantity %s",
			totalFilled.String(), o.FilledQuantity.String())
	}

	return nil
}
