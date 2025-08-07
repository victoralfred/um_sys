#ifndef ORDER_ENGINE_C_H
#define ORDER_ENGINE_C_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// Order types
typedef enum {
    ORDER_TYPE_MARKET = 1,
    ORDER_TYPE_LIMIT = 2,
    ORDER_TYPE_STOP = 3,
    ORDER_TYPE_STOP_LIMIT = 4,
    ORDER_TYPE_TRAILING_STOP = 5
} OrderType;

// Order sides
typedef enum {
    ORDER_SIDE_BUY = 1,
    ORDER_SIDE_SELL = 2
} OrderSide;

// Order status
typedef enum {
    ORDER_STATUS_PENDING = 1,
    ORDER_STATUS_SUBMITTED = 2,
    ORDER_STATUS_PARTIALLY_FILLED = 3,
    ORDER_STATUS_FILLED = 4,
    ORDER_STATUS_CANCELLED = 5,
    ORDER_STATUS_REJECTED = 6,
    ORDER_STATUS_EXPIRED = 7
} OrderStatus;

// Time in force
typedef enum {
    TIME_IN_FORCE_GTC = 1,  // Good Till Cancelled
    TIME_IN_FORCE_IOC = 2,  // Immediate Or Cancel
    TIME_IN_FORCE_FOK = 3,  // Fill Or Kill
    TIME_IN_FORCE_DAY = 4,  // Day
    TIME_IN_FORCE_GTD = 5   // Good Till Date
} TimeInForce;

// Execution result codes
typedef enum {
    EXEC_SUCCESS = 0,
    EXEC_ERROR_INVALID_ORDER = 1,
    EXEC_ERROR_INSUFFICIENT_LIQUIDITY = 2,
    EXEC_ERROR_RISK_LIMIT_EXCEEDED = 3,
    EXEC_ERROR_TIMEOUT = 4,
    EXEC_ERROR_SYSTEM_ERROR = 5,
    EXEC_ERROR_ORDER_NOT_FOUND = 6,
    EXEC_ERROR_MARKET_CLOSED = 7
} ExecutionResult;

// C-compatible structures
typedef struct COrderRequest {
    char order_id[64];
    char symbol[16];
    OrderType order_type;
    OrderSide side;
    double quantity;
    double price;
    double stop_price;
    TimeInForce time_in_force;
    int64_t timestamp_ns;
    char client_id[64];
} COrderRequest;

typedef struct COrderResponse {
    char order_id[64];
    ExecutionResult result;
    OrderStatus status;
    char message[256];
    double executed_quantity;
    double average_price;
    int64_t execution_time_ns;
    int64_t latency_micros;
} COrderResponse;

typedef struct COrderFill {
    char fill_id[64];
    char order_id[64];
    double price;
    double quantity;
    double fee;
    int64_t timestamp_ns;
    char venue[32];
} COrderFill;

typedef struct COrderBook {
    char symbol[16];
    int64_t timestamp_ns;
    double bid_price;
    double ask_price;
    double bid_size;
    double ask_size;
    double last_price;
    double last_size;
} COrderBook;

typedef struct CEngineMetrics {
    uint64_t total_orders_processed;
    uint64_t successful_executions;
    uint64_t failed_executions;
    uint64_t active_orders;
    double average_latency_micros;
    double p99_latency_micros;
    double orders_per_second;
    uint64_t memory_usage_bytes;
    double cpu_usage_percent;
    int64_t uptime_seconds;
} CEngineMetrics;

// C API functions
ExecutionResult engine_initialize(const char* config_json);
ExecutionResult engine_start(void);
ExecutionResult engine_stop(void);
ExecutionResult engine_submit_order(const COrderRequest* request, COrderResponse* response);
ExecutionResult engine_cancel_order(const char* order_id);
ExecutionResult engine_get_order_book(const char* symbol, COrderBook* book);
ExecutionResult engine_get_metrics(CEngineMetrics* metrics);
int engine_is_healthy(void);

// Callback function types
typedef void (*FillCallback)(const COrderFill* fill);
typedef void (*StatusCallback)(const char* order_id, OrderStatus status, const char* message);

// Register callbacks
ExecutionResult engine_register_fill_callback(FillCallback callback);
ExecutionResult engine_register_status_callback(StatusCallback callback);

#ifdef __cplusplus
}
#endif

#endif // ORDER_ENGINE_C_H