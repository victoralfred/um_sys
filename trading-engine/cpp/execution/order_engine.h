#ifndef ORDER_ENGINE_H
#define ORDER_ENGINE_H

#include <memory>
#include <vector>
#include <unordered_map>
#include <atomic>
#include <chrono>
#include <string>
#include <mutex>
#include <shared_mutex>
#include <condition_variable>
#include <thread>
#include <queue>
#include <cstring>

// C-compatible exports for CGO bindings
#ifdef __cplusplus
extern "C" {
#endif

// Forward declarations removed - structs defined below

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

#ifdef __cplusplus

#include <functional>
#include <array>

namespace trading {
namespace execution {

// High-performance constants
constexpr size_t MAX_CONCURRENT_ORDERS = 10000;
constexpr size_t ORDER_BOOK_DEPTH = 20;
constexpr size_t MEMORY_POOL_SIZE = 1024 * 1024; // 1MB
constexpr int64_t ORDER_TIMEOUT_NS = 30000000000LL; // 30 seconds
constexpr size_t MAX_SYMBOLS = 1000;

// Forward declarations
class OrderBookLevel;
class Order;
class ExecutionEngine;

// OrderFill struct for C++ (different from COrderFill)
struct OrderFill {
    std::string fill_id;
    std::string order_id;
    double price;
    double quantity;
    double fee;
    int64_t timestamp_ns;
    std::string venue;
};

// Price representation with fixed-point arithmetic for precision
class Price {
public:
    explicit Price(double value = 0.0);
    explicit Price(int64_t ticks);
    
    Price operator+(const Price& other) const;
    Price operator-(const Price& other) const;
    Price operator*(double factor) const;
    bool operator<(const Price& other) const;
    bool operator>(const Price& other) const;
    bool operator<=(const Price& other) const;
    bool operator>=(const Price& other) const;
    bool operator==(const Price& other) const;
    
    double to_double() const;
    int64_t ticks() const;
    
private:
    static constexpr int64_t TICK_PRECISION = 100000; // 5 decimal places
    int64_t ticks_;
};

// Memory pool for high-frequency allocations
template<typename T>
class MemoryPool {
public:
    explicit MemoryPool(size_t size);
    ~MemoryPool();
    
    T* acquire();
    void release(T* ptr);
    
private:
    std::vector<T> pool_;
    std::queue<T*> available_;
    std::mutex mutex_;
};

// Lock-free order book level
class OrderBookLevel {
public:
    OrderBookLevel() = default;
    
    void set_price_size(const Price& price, double size);
    Price get_price() const;
    double get_size() const;
    bool is_valid() const;
    
private:
    std::atomic<int64_t> price_ticks_{0};
    std::atomic<double> size_{0.0};
};

// High-performance order book
class OrderBook {
public:
    explicit OrderBook(const std::string& symbol);
    
    void update_bid(const Price& price, double size, size_t level = 0);
    void update_ask(const Price& price, double size, size_t level = 0);
    
    Price get_best_bid() const;
    Price get_best_ask() const;
    double get_bid_size(size_t level = 0) const;
    double get_ask_size(size_t level = 0) const;
    
    Price get_mid_price() const;
    double get_spread() const;
    
    bool has_sufficient_liquidity(OrderSide side, double quantity, const Price& limit_price) const;
    std::vector<std::pair<Price, double>> get_fills_for_market_order(OrderSide side, double quantity) const;
    
    int64_t get_last_update_time() const;
    void set_last_update_time(int64_t timestamp_ns);
    
private:
    std::string symbol_;
    std::array<OrderBookLevel, ORDER_BOOK_DEPTH> bids_;
    std::array<OrderBookLevel, ORDER_BOOK_DEPTH> asks_;
    std::atomic<int64_t> last_update_time_{0};
    mutable std::shared_mutex mutex_;
};

// Order representation
class Order {
public:
    Order() = default;
    Order(const COrderRequest& request);
    
    // Getters
    const std::string& get_order_id() const { return order_id_; }
    const std::string& get_symbol() const { return symbol_; }
    OrderType get_type() const { return type_; }
    OrderSide get_side() const { return side_; }
    double get_quantity() const { return quantity_; }
    Price get_price() const { return price_; }
    Price get_stop_price() const { return stop_price_; }
    TimeInForce get_time_in_force() const { return time_in_force_; }
    OrderStatus get_status() const { return status_.load(); }
    double get_filled_quantity() const { return filled_quantity_.load(); }
    Price get_average_fill_price() const;
    int64_t get_timestamp() const { return timestamp_ns_; }
    
    // State management
    void set_status(OrderStatus status);
    void add_fill(const Price& price, double quantity, double fee = 0.0);
    bool is_fully_filled() const;
    bool is_active() const;
    bool is_expired(int64_t current_time_ns) const;
    
    // Risk validation
    bool validate() const;
    double get_remaining_quantity() const;
    
private:
    std::string order_id_;
    std::string symbol_;
    OrderType type_;
    OrderSide side_;
    double quantity_;
    Price price_;
    Price stop_price_;
    TimeInForce time_in_force_;
    std::atomic<OrderStatus> status_{ORDER_STATUS_PENDING};
    std::atomic<double> filled_quantity_{0.0};
    std::atomic<int64_t> average_fill_price_ticks_{0};
    int64_t timestamp_ns_;
    std::string client_id_;
    
    mutable std::mutex fill_mutex_;
    std::vector<std::pair<Price, double>> fills_;
};

// Performance metrics collection
class PerformanceMetrics {
public:
    PerformanceMetrics();
    
    void record_order_processed(int64_t latency_micros, bool success);
    void record_fill(double quantity, const Price& price);
    void record_memory_usage(size_t bytes);
    void record_cpu_usage(double percentage);
    
    CEngineMetrics get_metrics() const;
    void reset();
    
private:
    std::atomic<uint64_t> total_orders_processed_{0};
    std::atomic<uint64_t> successful_executions_{0};
    std::atomic<uint64_t> failed_executions_{0};
    std::atomic<uint64_t> active_orders_{0};
    std::atomic<uint64_t> total_latency_micros_{0};
    std::atomic<double> total_volume_{0.0};
    std::atomic<size_t> memory_usage_bytes_{0};
    std::atomic<double> cpu_usage_percent_{0.0};
    
    // For P99 latency calculation
    mutable std::mutex latency_mutex_;
    std::vector<int64_t> latency_samples_;
    
    std::chrono::steady_clock::time_point start_time_;
};

// Main execution engine
class ExecutionEngine {
public:
    ExecutionEngine();
    ~ExecutionEngine();
    
    ExecutionResult initialize(const std::string& config_json);
    ExecutionResult start();
    ExecutionResult stop();
    
    ExecutionResult submit_order(const COrderRequest& request, COrderResponse& response);
    ExecutionResult cancel_order(const std::string& order_id);
    ExecutionResult get_order_book(const std::string& symbol, OrderBook& book);
    
    bool is_healthy() const;
    CEngineMetrics get_metrics() const;
    
    // Callback registration
    void register_fill_callback(std::function<void(const OrderFill&)> callback);
    void register_status_callback(std::function<void(const std::string&, OrderStatus, const std::string&)> callback);
    
    // Market data updates
    ExecutionResult update_order_book(const std::string& symbol, const OrderBook& book);
    
private:
    // Core execution logic
    ExecutionResult execute_market_order(std::shared_ptr<Order> order);
    ExecutionResult execute_limit_order(std::shared_ptr<Order> order);
    ExecutionResult execute_stop_order(std::shared_ptr<Order> order);
    
    // Order management
    void process_order_queue();
    void check_stop_orders();
    void expire_orders();
    
    // Market simulation (for testing)
    void simulate_market_data();
    Price get_simulated_price(const std::string& symbol) const;
    
    // Configuration and state
    struct Config {
        size_t max_concurrent_orders = MAX_CONCURRENT_ORDERS;
        int64_t order_timeout_ns = ORDER_TIMEOUT_NS;
        bool enable_risk_checks = true;
        double max_position_size = 1000000.0;
        bool enable_simulation = true;
        int worker_thread_count = 4;
    } config_;
    
    // Thread management
    std::vector<std::thread> worker_threads_;
    std::thread order_processor_thread_;
    std::thread market_simulator_thread_;
    std::atomic<bool> running_{false};
    
    // Order storage and management
    std::unordered_map<std::string, std::shared_ptr<Order>> active_orders_;
    std::unordered_map<std::string, std::unique_ptr<OrderBook>> order_books_;
    std::queue<std::shared_ptr<Order>> order_queue_;
    std::mutex order_mutex_;
    std::condition_variable order_cv_;
    
    // Memory pools
    std::unique_ptr<MemoryPool<Order>> order_pool_;
    std::unique_ptr<MemoryPool<OrderFill>> fill_pool_;
    
    // Performance monitoring
    std::unique_ptr<PerformanceMetrics> metrics_;
    
    // Callbacks
    std::function<void(const OrderFill&)> fill_callback_;
    std::function<void(const std::string&, OrderStatus, const std::string&)> status_callback_;
    
    // Thread synchronization
    mutable std::shared_mutex engine_mutex_;
    std::atomic<bool> initialized_{false};
    std::atomic<bool> healthy_{false};
};

} // namespace execution
} // namespace trading

#endif // __cplusplus

#endif // ORDER_ENGINE_H