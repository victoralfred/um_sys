package ports

import (
	"context"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// ExecutionResult represents the result of order execution
type ExecutionResult struct {
	OrderID       string        `json:"order_id"`
	ExecutedAt    time.Time     `json:"executed_at"`
	AveragePrice  types.Decimal `json:"average_price"`
	TotalQuantity types.Decimal `json:"total_quantity"`
	Fills         []Fill        `json:"fills"`
	Status        string        `json:"status"`
	ErrorMessage  string        `json:"error_message,omitempty"`
}

// Fill represents a partial execution of an order
type Fill struct {
	ID           string        `json:"id"`
	OrderID      string        `json:"order_id"`
	Price        types.Decimal `json:"price"`
	Quantity     types.Decimal `json:"quantity"`
	Fee          types.Decimal `json:"fee"`
	Timestamp    time.Time     `json:"timestamp"`
	Venue        string        `json:"venue"`
	TradeID      string        `json:"trade_id"`
	Counterparty string        `json:"counterparty,omitempty"`
}

// OrderExecutor defines the interface for order execution engines
type OrderExecutor interface {
	// SubmitOrder submits an order for execution
	SubmitOrder(ctx context.Context, order *domain.Order) (*ExecutionResult, error)

	// CancelOrder cancels a pending order
	CancelOrder(ctx context.Context, orderID string) error

	// ModifyOrder modifies an existing order
	ModifyOrder(ctx context.Context, orderID string, modification OrderModification) error

	// GetOrderStatus retrieves the current status of an order
	GetOrderStatus(ctx context.Context, orderID string) (*domain.Order, error)

	// GetExecutionHistory retrieves execution history for an order
	GetExecutionHistory(ctx context.Context, orderID string) ([]Fill, error)
}

// OrderModification represents changes to an existing order
type OrderModification struct {
	NewQuantity  *types.Decimal `json:"new_quantity,omitempty"`
	NewPrice     *types.Decimal `json:"new_price,omitempty"`
	NewStopPrice *types.Decimal `json:"new_stop_price,omitempty"`
}

// ExecutionEngine defines the core execution engine interface
type ExecutionEngine interface {
	OrderExecutor

	// Start starts the execution engine
	Start(ctx context.Context) error

	// Stop stops the execution engine gracefully
	Stop(ctx context.Context) error

	// IsHealthy returns the health status of the engine
	IsHealthy() bool

	// GetMetrics returns execution metrics
	GetMetrics() ExecutionMetrics
}

// ExecutionMetrics contains performance and operational metrics
type ExecutionMetrics struct {
	TotalOrdersProcessed uint64        `json:"total_orders_processed"`
	OrdersPerSecond      float64       `json:"orders_per_second"`
	AverageLatency       time.Duration `json:"average_latency"`
	P99Latency           time.Duration `json:"p99_latency"`
	SuccessfulExecutions uint64        `json:"successful_executions"`
	FailedExecutions     uint64        `json:"failed_executions"`
	ActiveOrders         uint64        `json:"active_orders"`
	LastExecutionTime    time.Time     `json:"last_execution_time"`
}

// SlippageEstimator calculates expected slippage for orders
type SlippageEstimator interface {
	// EstimateSlippage calculates expected slippage for an order
	EstimateSlippage(ctx context.Context, order *domain.Order, marketData MarketData) (types.Decimal, error)

	// GetHistoricalSlippage retrieves historical slippage data
	GetHistoricalSlippage(ctx context.Context, asset *domain.Asset, timeRange TimeRange) ([]SlippageData, error)
}

// MarketData represents current market conditions
type MarketData struct {
	Asset          *domain.Asset `json:"asset"`
	BidPrice       types.Decimal `json:"bid_price"`
	AskPrice       types.Decimal `json:"ask_price"`
	BidSize        types.Decimal `json:"bid_size"`
	AskSize        types.Decimal `json:"ask_size"`
	LastTradePrice types.Decimal `json:"last_trade_price"`
	LastTradeSize  types.Decimal `json:"last_trade_size"`
	Volume         types.Decimal `json:"volume"`
	VWAP           types.Decimal `json:"vwap"`
	Volatility     types.Decimal `json:"volatility"`
	Timestamp      time.Time     `json:"timestamp"`
}

// SlippageData represents historical slippage information
type SlippageData struct {
	OrderSize     types.Decimal `json:"order_size"`
	ExpectedPrice types.Decimal `json:"expected_price"`
	ActualPrice   types.Decimal `json:"actual_price"`
	Slippage      types.Decimal `json:"slippage"`
	SlippageBps   types.Decimal `json:"slippage_bps"`
	Timestamp     time.Time     `json:"timestamp"`
}

// TimeRange represents a time period for queries
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// OrderValidator validates orders before execution
type OrderValidator interface {
	// ValidateOrder performs pre-execution validation
	ValidateOrder(ctx context.Context, order *domain.Order) error

	// ValidateRiskLimits checks if order violates risk limits
	ValidateRiskLimits(ctx context.Context, order *domain.Order, portfolio *domain.Portfolio) error

	// ValidateMarketConditions checks if market conditions allow execution
	ValidateMarketConditions(ctx context.Context, order *domain.Order, marketData MarketData) error
}

// ExecutionEventType represents different types of execution events
type ExecutionEventType int

const (
	ExecutionEventOrderSubmitted ExecutionEventType = iota + 1
	ExecutionEventOrderAccepted
	ExecutionEventOrderRejected
	ExecutionEventOrderFilled
	ExecutionEventOrderPartiallyFilled
	ExecutionEventOrderCancelled
	ExecutionEventOrderModified
	ExecutionEventExecutionError
)

func (e ExecutionEventType) String() string {
	switch e {
	case ExecutionEventOrderSubmitted:
		return "ORDER_SUBMITTED"
	case ExecutionEventOrderAccepted:
		return "ORDER_ACCEPTED"
	case ExecutionEventOrderRejected:
		return "ORDER_REJECTED"
	case ExecutionEventOrderFilled:
		return "ORDER_FILLED"
	case ExecutionEventOrderPartiallyFilled:
		return "ORDER_PARTIALLY_FILLED"
	case ExecutionEventOrderCancelled:
		return "ORDER_CANCELLED"
	case ExecutionEventOrderModified:
		return "ORDER_MODIFIED"
	case ExecutionEventExecutionError:
		return "EXECUTION_ERROR"
	default:
		return "UNKNOWN"
	}
}

// ExecutionEvent represents an event in the order execution process
type ExecutionEvent struct {
	ID        string             `json:"id"`
	Type      ExecutionEventType `json:"type"`
	OrderID   string             `json:"order_id"`
	Timestamp time.Time          `json:"timestamp"`
	Data      interface{}        `json:"data"`
	Message   string             `json:"message"`
}

// ExecutionEventHandler handles execution events
type ExecutionEventHandler interface {
	// HandleEvent processes an execution event
	HandleEvent(ctx context.Context, event ExecutionEvent) error
}

// ExecutionEventPublisher publishes execution events
type ExecutionEventPublisher interface {
	// PublishEvent publishes an execution event
	PublishEvent(ctx context.Context, event ExecutionEvent) error
}

// PositionUpdater updates positions based on execution results
type PositionUpdater interface {
	// UpdatePosition updates position from execution result
	UpdatePosition(ctx context.Context, executionResult ExecutionResult) error

	// GetPosition retrieves current position for an asset
	GetPosition(ctx context.Context, assetID string) (*domain.Position, error)

	// GetAllPositions retrieves all open positions
	GetAllPositions(ctx context.Context) ([]*domain.Position, error)
}

// RiskManager provides risk management capabilities for execution
type RiskManager interface {
	// CheckPreTradeRisk validates order against risk limits before execution
	CheckPreTradeRisk(ctx context.Context, order *domain.Order, portfolio *domain.Portfolio) error

	// CheckPostTradeRisk validates portfolio risk after execution
	CheckPostTradeRisk(ctx context.Context, executionResult ExecutionResult, portfolio *domain.Portfolio) error

	// GetRiskMetrics calculates current portfolio risk metrics
	GetRiskMetrics(ctx context.Context, portfolio *domain.Portfolio) (RiskMetrics, error)
}

// RiskMetrics contains portfolio risk measurements
type RiskMetrics struct {
	VaR               types.Decimal `json:"var"`
	CVaR              types.Decimal `json:"cvar"`
	MaxDrawdown       types.Decimal `json:"max_drawdown"`
	Volatility        types.Decimal `json:"volatility"`
	SharpeRatio       types.Decimal `json:"sharpe_ratio"`
	BetaToMarket      types.Decimal `json:"beta_to_market"`
	ConcentrationRisk types.Decimal `json:"concentration_risk"`
	LeverageRatio     types.Decimal `json:"leverage_ratio"`
}

// ExecutionConfig contains configuration for the execution engine
type ExecutionConfig struct {
	MaxOrdersPerSecond   int           `json:"max_orders_per_second"`
	MaxConcurrentOrders  int           `json:"max_concurrent_orders"`
	OrderTimeoutDuration time.Duration `json:"order_timeout_duration"`
	RetryAttempts        int           `json:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay"`
	SlippageThreshold    types.Decimal `json:"slippage_threshold"`
	EnableRiskChecks     bool          `json:"enable_risk_checks"`
	EnableSlippageCheck  bool          `json:"enable_slippage_check"`
	MaxPositionSize      types.Decimal `json:"max_position_size"`
	MaxDailyVolume       types.Decimal `json:"max_daily_volume"`
}
