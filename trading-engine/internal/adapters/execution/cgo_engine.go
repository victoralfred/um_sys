package execution

/*
#cgo CFLAGS: -I../../../cpp/execution
#cgo LDFLAGS: -L../../../cpp/execution -lorder_engine -lstdc++ -lpthread -lrt -ldl

#include "order_engine_c.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// CGOExecutionEngine implements the ExecutionEngine interface using C++ backend
type CGOExecutionEngine struct {
	initialized bool
	running     bool
}

// NewCGOExecutionEngine creates a new CGO-based execution engine
func NewCGOExecutionEngine() *CGOExecutionEngine {
	return &CGOExecutionEngine{}
}

// Initialize initializes the C++ execution engine
func (e *CGOExecutionEngine) Initialize(config string) error {
	// Allow reinitializing if needed
	if e.initialized && e.running {
		return fmt.Errorf("engine is running, stop first before reinitializing")
	}

	configStr := C.CString(config)
	defer C.free(unsafe.Pointer(configStr))

	result := C.engine_initialize(configStr)
	if result != C.EXEC_SUCCESS {
		return fmt.Errorf("failed to initialize engine: %d", int(result))
	}

	e.initialized = true
	e.running = false // Reset running state
	return nil
}

// Start starts the C++ execution engine (implements ExecutionEngine interface)
func (e *CGOExecutionEngine) Start(ctx context.Context) error {
	if !e.initialized {
		return fmt.Errorf("engine not initialized")
	}

	if e.running {
		return fmt.Errorf("engine already running")
	}

	result := C.engine_start()
	if result != C.EXEC_SUCCESS {
		return fmt.Errorf("failed to start engine: %d", int(result))
	}

	e.running = true
	return nil
}

// Stop stops the C++ execution engine (implements ExecutionEngine interface)
func (e *CGOExecutionEngine) Stop(ctx context.Context) error {
	if !e.initialized && !e.running {
		return nil // Already stopped and cleaned
	}

	// Try to stop even if not marked as running (for cleanup)
	result := C.engine_stop()
	if result != C.EXEC_SUCCESS && e.running {
		return fmt.Errorf("failed to stop engine: %d", int(result))
	}

	e.running = false
	e.initialized = false
	return nil
}

// IsHealthy checks if the engine is healthy
func (e *CGOExecutionEngine) IsHealthy() bool {
	if !e.initialized || !e.running {
		return false
	}

	healthy := C.engine_is_healthy()
	return int(healthy) == 1
}

// SubmitOrder submits an order through the C++ engine (implements OrderExecutor interface)
func (e *CGOExecutionEngine) SubmitOrder(ctx context.Context, order *domain.Order) (*ports.ExecutionResult, error) {
	if !e.running {
		return nil, fmt.Errorf("engine not running")
	}

	// Convert Go order to C request
	request := e.orderToRequest(order)

	var response C.COrderResponse
	result := C.engine_submit_order(&request, &response)

	if result != C.EXEC_SUCCESS {
		return &ports.ExecutionResult{
			OrderID:      order.ID,
			Status:       "REJECTED",
			ErrorMessage: C.GoString(&response.message[0]),
			ExecutedAt:   time.Now(),
		}, nil
	}

	// Convert C response to Go result
	execResult := &ports.ExecutionResult{
		OrderID:       order.ID,
		Status:        e.mapOrderStatus(response.status),
		TotalQuantity: types.NewDecimalFromFloat(float64(response.executed_quantity)),
		AveragePrice:  types.NewDecimalFromFloat(float64(response.average_price)),
		ExecutedAt:    time.Unix(0, int64(response.execution_time_ns)),
	}

	// Add fills if any
	if response.executed_quantity > 0 {
		fill := ports.Fill{
			ID:        fmt.Sprintf("fill_%s_1", order.ID),
			OrderID:   order.ID,
			Price:     execResult.AveragePrice,
			Quantity:  execResult.TotalQuantity,
			Timestamp: execResult.ExecutedAt,
			Venue:     "CGO_ENGINE",
		}
		execResult.Fills = []ports.Fill{fill}
	}

	return execResult, nil
}

// CancelOrder cancels an order in the C++ engine (implements OrderExecutor interface)
func (e *CGOExecutionEngine) CancelOrder(ctx context.Context, orderID string) error {
	if !e.running {
		return fmt.Errorf("engine not running")
	}

	orderIDStr := C.CString(orderID)
	defer C.free(unsafe.Pointer(orderIDStr))

	result := C.engine_cancel_order(orderIDStr)
	if result != C.EXEC_SUCCESS {
		return fmt.Errorf("failed to cancel order %s: %d", orderID, int(result))
	}

	return nil
}

// ModifyOrder modifies an existing order (implements OrderExecutor interface)
func (e *CGOExecutionEngine) ModifyOrder(ctx context.Context, orderID string, modification ports.OrderModification) error {
	// Not implemented in C++ engine yet
	return fmt.Errorf("order modification not yet implemented")
}

// GetOrderStatus retrieves the current status of an order (implements OrderExecutor interface)
func (e *CGOExecutionEngine) GetOrderStatus(ctx context.Context, orderID string) (*domain.Order, error) {
	// Not implemented in C++ engine yet
	return nil, fmt.Errorf("order status retrieval not yet implemented")
}

// GetExecutionHistory retrieves execution history for an order (implements OrderExecutor interface)
func (e *CGOExecutionEngine) GetExecutionHistory(ctx context.Context, orderID string) ([]ports.Fill, error) {
	// Not implemented in C++ engine yet
	return nil, fmt.Errorf("execution history retrieval not yet implemented")
}

// GetOrderBook retrieves order book data from the C++ engine
func (e *CGOExecutionEngine) GetOrderBook(symbol string) (*ports.MarketData, error) {
	if !e.running {
		return nil, fmt.Errorf("engine not running")
	}

	symbolStr := C.CString(symbol)
	defer C.free(unsafe.Pointer(symbolStr))

	var book C.COrderBook
	result := C.engine_get_order_book(symbolStr, &book)
	if result != C.EXEC_SUCCESS {
		return nil, fmt.Errorf("failed to get order book for %s: %d", symbol, int(result))
	}

	return &ports.MarketData{
		Asset:          nil, // Will need to be filled in by caller
		BidPrice:       types.NewDecimalFromFloat(float64(book.bid_price)),
		AskPrice:       types.NewDecimalFromFloat(float64(book.ask_price)),
		BidSize:        types.NewDecimalFromFloat(float64(book.bid_size)),
		AskSize:        types.NewDecimalFromFloat(float64(book.ask_size)),
		LastTradePrice: types.NewDecimalFromFloat(float64(book.last_price)),
		Timestamp:      time.Unix(0, int64(book.timestamp_ns)),
	}, nil
}

// GetMetrics retrieves performance metrics from the C++ engine
func (e *CGOExecutionEngine) GetMetrics() ports.ExecutionMetrics {
	if !e.running {
		// Return zero metrics if not running
		return ports.ExecutionMetrics{}
	}

	var metrics C.CEngineMetrics
	result := C.engine_get_metrics(&metrics)
	if result != C.EXEC_SUCCESS {
		// Return zero metrics on error
		return ports.ExecutionMetrics{}
	}

	return ports.ExecutionMetrics{
		TotalOrdersProcessed: uint64(metrics.total_orders_processed),
		SuccessfulExecutions: uint64(metrics.successful_executions),
		FailedExecutions:     uint64(metrics.failed_executions),
		ActiveOrders:         uint64(metrics.active_orders),
		AverageLatency:       time.Duration(metrics.average_latency_micros) * time.Microsecond,
		P99Latency:           time.Duration(metrics.p99_latency_micros) * time.Microsecond,
		OrdersPerSecond:      float64(metrics.orders_per_second),
	}
}

// Helper functions

func (e *CGOExecutionEngine) orderToRequest(order *domain.Order) C.COrderRequest {
	var request C.COrderRequest

	// Copy order ID (ensure null termination)
	orderID := order.ID
	if len(orderID) > 63 {
		orderID = orderID[:63]
	}
	orderIDCStr := C.CString(orderID)
	defer C.free(unsafe.Pointer(orderIDCStr))
	C.strcpy(&request.order_id[0], orderIDCStr)

	// Copy symbol
	symbol := order.Asset.Symbol
	if len(symbol) > 15 {
		symbol = symbol[:15]
	}
	symbolCStr := C.CString(symbol)
	defer C.free(unsafe.Pointer(symbolCStr))
	C.strcpy(&request.symbol[0], symbolCStr)

	// Copy client ID
	clientID := order.ClientOrderID
	if len(clientID) > 63 {
		clientID = clientID[:63]
	}
	clientIDCStr := C.CString(clientID)
	defer C.free(unsafe.Pointer(clientIDCStr))
	C.strcpy(&request.client_id[0], clientIDCStr)

	// Map order type
	request.order_type = e.mapOrderType(order.Type)
	request.side = e.mapOrderSide(order.Side)
	request.quantity = C.double(order.Quantity.Float64())
	request.price = C.double(order.Price.Float64())
	request.stop_price = C.double(order.StopPrice.Float64())
	request.time_in_force = e.mapTimeInForce(order.TimeInForce)
	request.timestamp_ns = C.int64_t(order.CreatedAt.UnixNano())

	return request
}

func (e *CGOExecutionEngine) mapOrderType(orderType domain.OrderType) C.OrderType {
	switch orderType {
	case domain.OrderTypeMarket:
		return C.ORDER_TYPE_MARKET
	case domain.OrderTypeLimit:
		return C.ORDER_TYPE_LIMIT
	case domain.OrderTypeStop:
		return C.ORDER_TYPE_STOP
	case domain.OrderTypeStopLimit:
		return C.ORDER_TYPE_STOP_LIMIT
	default:
		return C.ORDER_TYPE_MARKET
	}
}

func (e *CGOExecutionEngine) mapOrderSide(side domain.OrderSide) C.OrderSide {
	switch side {
	case domain.OrderSideBuy:
		return C.ORDER_SIDE_BUY
	case domain.OrderSideSell:
		return C.ORDER_SIDE_SELL
	default:
		return C.ORDER_SIDE_BUY
	}
}

func (e *CGOExecutionEngine) mapTimeInForce(tif domain.TimeInForce) C.TimeInForce {
	switch tif {
	case domain.TimeInForceGTC:
		return C.TIME_IN_FORCE_GTC
	case domain.TimeInForceIOC:
		return C.TIME_IN_FORCE_IOC
	case domain.TimeInForceFOK:
		return C.TIME_IN_FORCE_FOK
	case domain.TimeInForceDAY:
		return C.TIME_IN_FORCE_DAY
	default:
		return C.TIME_IN_FORCE_GTC
	}
}

func (e *CGOExecutionEngine) mapOrderStatus(status C.OrderStatus) string {
	switch status {
	case C.ORDER_STATUS_PENDING:
		return "PENDING"
	case C.ORDER_STATUS_SUBMITTED:
		return "SUBMITTED"
	case C.ORDER_STATUS_PARTIALLY_FILLED:
		return "PARTIALLY_FILLED"
	case C.ORDER_STATUS_FILLED:
		return "FILLED"
	case C.ORDER_STATUS_CANCELLED:
		return "CANCELLED"
	case C.ORDER_STATUS_REJECTED:
		return "REJECTED"
	case C.ORDER_STATUS_EXPIRED:
		return "EXPIRED"
	default:
		return "PENDING"
	}
}

func (e *CGOExecutionEngine) mapErrorCode(result C.ExecutionResult) string {
	switch result {
	case C.EXEC_ERROR_INVALID_ORDER:
		return "INVALID_ORDER"
	case C.EXEC_ERROR_INSUFFICIENT_LIQUIDITY:
		return "INSUFFICIENT_LIQUIDITY"
	case C.EXEC_ERROR_RISK_LIMIT_EXCEEDED:
		return "RISK_LIMIT_EXCEEDED"
	case C.EXEC_ERROR_TIMEOUT:
		return "TIMEOUT"
	case C.EXEC_ERROR_SYSTEM_ERROR:
		return "SYSTEM_ERROR"
	case C.EXEC_ERROR_ORDER_NOT_FOUND:
		return "ORDER_NOT_FOUND"
	case C.EXEC_ERROR_MARKET_CLOSED:
		return "MARKET_CLOSED"
	default:
		return "UNKNOWN_ERROR"
	}
}
