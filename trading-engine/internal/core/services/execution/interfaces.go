package execution

import (
	"context"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// ExecutionService provides high-level execution capabilities
type ExecutionService interface {
	// ExecuteOrder executes an order with full validation and risk checks
	ExecuteOrder(ctx context.Context, order *domain.Order) (*ports.ExecutionResult, error)

	// ExecuteOrderWithSlippageLimit executes order with maximum allowable slippage
	ExecuteOrderWithSlippageLimit(ctx context.Context, order *domain.Order, maxSlippageBps types.Decimal) (*ports.ExecutionResult, error)

	// CancelOrder cancels an active order
	CancelOrder(ctx context.Context, orderID string) error

	// ModifyOrder modifies an existing order
	ModifyOrder(ctx context.Context, orderID string, modification *ports.OrderModification) error

	// GetActiveOrders returns all active orders
	GetActiveOrders(ctx context.Context) ([]*domain.Order, error)

	// GetOrderHistory returns execution history for an order
	GetOrderHistory(ctx context.Context, orderID string) ([]ports.Fill, error)

	// GetExecutionMetrics returns current execution performance metrics
	GetExecutionMetrics(ctx context.Context) (*ports.ExecutionMetrics, error)
}

// OrderManager manages order lifecycle and state transitions
type OrderManager interface {
	// SubmitOrder submits a new order to the execution system
	SubmitOrder(ctx context.Context, order *domain.Order) error

	// ProcessFill processes a fill notification
	ProcessFill(ctx context.Context, fill *ports.Fill) error

	// ProcessReject processes an order rejection
	ProcessReject(ctx context.Context, orderID string, reason string) error

	// UpdateOrderStatus updates the status of an order
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error

	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)

	// GetOrdersByStatus retrieves orders by status
	GetOrdersByStatus(ctx context.Context, status domain.OrderStatus) ([]*domain.Order, error)

	// ValidateOrderTransition validates a status transition
	ValidateOrderTransition(ctx context.Context, orderID string, newStatus domain.OrderStatus) error
}

// ExecutionEngine defines the low-level execution engine interface
type ExecutionEngine interface {
	// Initialize initializes the execution engine
	Initialize(ctx context.Context, config *ExecutionEngineConfig) error

	// Start starts the execution engine
	Start(ctx context.Context) error

	// Stop stops the execution engine gracefully
	Stop(ctx context.Context) error

	// SubmitOrder submits an order for execution
	SubmitOrder(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error)

	// CancelOrder cancels a pending order
	CancelOrder(ctx context.Context, orderID string) error

	// GetOrderBook retrieves the current order book state
	GetOrderBook(ctx context.Context, symbol string) (*OrderBook, error)

	// GetEngineStatus returns the current engine status
	GetEngineStatus(ctx context.Context) (*EngineStatus, error)

	// RegisterFillHandler registers a fill event handler
	RegisterFillHandler(handler FillHandler)

	// RegisterStatusHandler registers a status update handler
	RegisterStatusHandler(handler StatusHandler)
}

// ExecutionRequest represents a request to execute an order
type ExecutionRequest struct {
	Order           *domain.Order      `json:"order"`
	MaxSlippageBps  types.Decimal      `json:"max_slippage_bps"`
	TimeInForce     domain.TimeInForce `json:"time_in_force"`
	ExecutionType   ExecutionType      `json:"execution_type"`
	PostOnly        bool               `json:"post_only"`
	ReduceOnly      bool               `json:"reduce_only"`
	ClientRequestID string             `json:"client_request_id"`
	Timestamp       time.Time          `json:"timestamp"`
}

// ExecutionResponse represents the response from execution request
type ExecutionResponse struct {
	RequestID     string          `json:"request_id"`
	OrderID       string          `json:"order_id"`
	Status        ExecutionStatus `json:"status"`
	Message       string          `json:"message"`
	EstimatedFill *EstimatedFill  `json:"estimated_fill,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
	LatencyMicros int64           `json:"latency_micros"`
}

// ExecutionType defines different execution algorithms
type ExecutionType int

const (
	ExecutionTypeImmediate ExecutionType = iota + 1 // Immediate execution
	ExecutionTypeTWAP                               // Time-Weighted Average Price
	ExecutionTypeVWAP                               // Volume-Weighted Average Price
	ExecutionTypePOV                                // Percentage of Volume
	ExecutionTypeIceberg                            // Iceberg orders
	ExecutionTypeSniper                             // Aggressive liquidity taking
	ExecutionTypeMaker                              // Passive liquidity provision
)

func (e ExecutionType) String() string {
	switch e {
	case ExecutionTypeImmediate:
		return "IMMEDIATE"
	case ExecutionTypeTWAP:
		return "TWAP"
	case ExecutionTypeVWAP:
		return "VWAP"
	case ExecutionTypePOV:
		return "POV"
	case ExecutionTypeIceberg:
		return "ICEBERG"
	case ExecutionTypeSniper:
		return "SNIPER"
	case ExecutionTypeMaker:
		return "MAKER"
	default:
		return "UNKNOWN"
	}
}

// ExecutionStatus represents the status of an execution request
type ExecutionStatus int

const (
	ExecutionStatusPending ExecutionStatus = iota + 1
	ExecutionStatusAccepted
	ExecutionStatusRejected
	ExecutionStatusPartiallyFilled
	ExecutionStatusFilled
	ExecutionStatusCancelled
	ExecutionStatusExpired
	ExecutionStatusError
)

func (e ExecutionStatus) String() string {
	switch e {
	case ExecutionStatusPending:
		return "PENDING"
	case ExecutionStatusAccepted:
		return "ACCEPTED"
	case ExecutionStatusRejected:
		return "REJECTED"
	case ExecutionStatusPartiallyFilled:
		return "PARTIALLY_FILLED"
	case ExecutionStatusFilled:
		return "FILLED"
	case ExecutionStatusCancelled:
		return "CANCELLED"
	case ExecutionStatusExpired:
		return "EXPIRED"
	case ExecutionStatusError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// EstimatedFill represents an estimated execution result
type EstimatedFill struct {
	EstimatedPrice    types.Decimal `json:"estimated_price"`
	EstimatedQuantity types.Decimal `json:"estimated_quantity"`
	EstimatedFee      types.Decimal `json:"estimated_fee"`
	EstimatedSlippage types.Decimal `json:"estimated_slippage"`
	Confidence        float64       `json:"confidence"`
}

// OrderBook represents the current state of the order book
type OrderBook struct {
	Symbol    string      `json:"symbol"`
	Timestamp time.Time   `json:"timestamp"`
	Bids      []BookLevel `json:"bids"`
	Asks      []BookLevel `json:"asks"`
	LastTrade *LastTrade  `json:"last_trade,omitempty"`
}

// BookLevel represents a price level in the order book
type BookLevel struct {
	Price    types.Decimal `json:"price"`
	Quantity types.Decimal `json:"quantity"`
	Orders   int           `json:"orders"`
}

// LastTrade represents the most recent trade
type LastTrade struct {
	Price     types.Decimal `json:"price"`
	Quantity  types.Decimal `json:"quantity"`
	Timestamp time.Time     `json:"timestamp"`
	Side      string        `json:"side"`
}

// EngineStatus represents the current status of the execution engine
type EngineStatus struct {
	IsRunning       bool          `json:"is_running"`
	Uptime          time.Duration `json:"uptime"`
	ProcessedOrders uint64        `json:"processed_orders"`
	ActiveOrders    uint64        `json:"active_orders"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	MemoryUsage     uint64        `json:"memory_usage"`
	CPUUsage        float64       `json:"cpu_usage"`
	LastHealthCheck time.Time     `json:"last_health_check"`
	ConnectedVenues []string      `json:"connected_venues"`
}

// FillHandler handles fill notifications from the execution engine
type FillHandler interface {
	HandleFill(ctx context.Context, fill *ports.Fill) error
}

// StatusHandler handles status update notifications
type StatusHandler interface {
	HandleStatusUpdate(ctx context.Context, orderID string, status ExecutionStatus, message string) error
}

// ExecutionEngineConfig contains configuration for the execution engine
type ExecutionEngineConfig struct {
	// Performance settings
	MaxConcurrentOrders  int           `json:"max_concurrent_orders"`
	MaxOrdersPerSecond   int           `json:"max_orders_per_second"`
	OrderTimeoutDuration time.Duration `json:"order_timeout_duration"`
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`

	// Risk settings
	EnableRiskChecks     bool          `json:"enable_risk_checks"`
	MaxOrderSize         types.Decimal `json:"max_order_size"`
	MaxPositionSize      types.Decimal `json:"max_position_size"`
	MaxDailyVolume       types.Decimal `json:"max_daily_volume"`
	DefaultSlippageLimit types.Decimal `json:"default_slippage_limit"`

	// Execution settings
	DefaultExecutionType ExecutionType `json:"default_execution_type"`
	EnableSmartRouting   bool          `json:"enable_smart_routing"`
	EnableDarkPools      bool          `json:"enable_dark_pools"`
	MinFillSize          types.Decimal `json:"min_fill_size"`

	// Technical settings
	WorkerPoolSize  int  `json:"worker_pool_size"`
	OrderBookDepth  int  `json:"order_book_depth"`
	MemoryPoolSize  int  `json:"memory_pool_size"`
	EnableNUMAAware bool `json:"enable_numa_aware"`

	// Monitoring settings
	MetricsEnabled      bool          `json:"metrics_enabled"`
	MetricsInterval     time.Duration `json:"metrics_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	EnableTracing       bool          `json:"enable_tracing"`

	// Venues configuration
	EnabledVenues []string               `json:"enabled_venues"`
	VenueConfigs  map[string]interface{} `json:"venue_configs"`
}

// AlgoExecutor defines the interface for algorithmic execution strategies
type AlgoExecutor interface {
	// Execute executes an order using the specified algorithm
	Execute(ctx context.Context, order *domain.Order, params AlgoParams) (*ports.ExecutionResult, error)

	// GetSupportedAlgos returns the list of supported algorithms
	GetSupportedAlgos() []ExecutionType

	// ValidateParams validates algorithm parameters
	ValidateParams(algoType ExecutionType, params AlgoParams) error
}

// AlgoParams contains parameters for algorithmic execution
type AlgoParams struct {
	// TWAP parameters
	TWAPDuration  *time.Duration `json:"twap_duration,omitempty"`
	TWAPSliceSize *types.Decimal `json:"twap_slice_size,omitempty"`

	// VWAP parameters
	VWAPEndTime       *time.Time `json:"vwap_end_time,omitempty"`
	VWAPParticipation *float64   `json:"vwap_participation,omitempty"`

	// POV parameters
	POVRate    *float64       `json:"pov_rate,omitempty"`
	POVMaxSize *types.Decimal `json:"pov_max_size,omitempty"`

	// Iceberg parameters
	IcebergSliceSize *types.Decimal `json:"iceberg_slice_size,omitempty"`
	IcebergRandomize *bool          `json:"iceberg_randomize,omitempty"`

	// General parameters
	MaxSlippage *types.Decimal `json:"max_slippage,omitempty"`
	Urgency     *float64       `json:"urgency,omitempty"` // 0.0 = patient, 1.0 = aggressive
	StartTime   *time.Time     `json:"start_time,omitempty"`
	EndTime     *time.Time     `json:"end_time,omitempty"`
}

// PerformanceMonitor monitors execution performance
type PerformanceMonitor interface {
	// RecordExecution records an execution event
	RecordExecution(ctx context.Context, orderID string, latency time.Duration, success bool)

	// RecordSlippage records slippage for an execution
	RecordSlippage(ctx context.Context, orderID string, expectedPrice, actualPrice types.Decimal)

	// GetMetrics returns current performance metrics
	GetMetrics(ctx context.Context) (*ExecutionPerformanceMetrics, error)

	// Reset resets performance metrics
	Reset(ctx context.Context) error
}

// ExecutionPerformanceMetrics contains detailed performance metrics
type ExecutionPerformanceMetrics struct {
	// Latency metrics
	AverageLatency time.Duration `json:"average_latency"`
	MedianLatency  time.Duration `json:"median_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	MaxLatency     time.Duration `json:"max_latency"`

	// Throughput metrics
	OrdersPerSecond float64       `json:"orders_per_second"`
	FillsPerSecond  float64       `json:"fills_per_second"`
	TotalVolume     types.Decimal `json:"total_volume"`

	// Quality metrics
	FillRate        float64       `json:"fill_rate"`
	AverageSlippage types.Decimal `json:"average_slippage"`
	SlippageStdDev  types.Decimal `json:"slippage_std_dev"`
	SuccessRate     float64       `json:"success_rate"`

	// Error metrics
	TimeoutRate float64 `json:"timeout_rate"`
	RejectRate  float64 `json:"reject_rate"`
	ErrorRate   float64 `json:"error_rate"`

	// Resource metrics
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    uint64  `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`

	// Time range
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	SampleCount int64     `json:"sample_count"`
}
