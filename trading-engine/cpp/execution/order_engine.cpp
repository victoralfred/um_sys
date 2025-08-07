#include "order_engine.h"
#include <algorithm>
#include <random>
#include <cmath>
#include <sstream>
#include <iomanip>
#include <chrono>
#include <thread>
#include <shared_mutex>

using namespace trading::execution;

// Global engine instance for C API
static std::unique_ptr<ExecutionEngine> g_engine = nullptr;
static std::mutex g_engine_mutex;

//==============================================================================
// Price Implementation
//==============================================================================

Price::Price(double value) : ticks_(static_cast<int64_t>(value * TICK_PRECISION)) {}

Price::Price(int64_t ticks) : ticks_(ticks) {}

Price Price::operator+(const Price& other) const {
    return Price(ticks_ + other.ticks_);
}

Price Price::operator-(const Price& other) const {
    return Price(ticks_ - other.ticks_);
}

Price Price::operator*(double factor) const {
    return Price(static_cast<int64_t>(ticks_ * factor));
}

bool Price::operator<(const Price& other) const {
    return ticks_ < other.ticks_;
}

bool Price::operator>(const Price& other) const {
    return ticks_ > other.ticks_;
}

bool Price::operator<=(const Price& other) const {
    return ticks_ <= other.ticks_;
}

bool Price::operator>=(const Price& other) const {
    return ticks_ >= other.ticks_;
}

bool Price::operator==(const Price& other) const {
    return ticks_ == other.ticks_;
}

double Price::to_double() const {
    return static_cast<double>(ticks_) / TICK_PRECISION;
}

int64_t Price::ticks() const {
    return ticks_;
}

//==============================================================================
// MemoryPool Implementation
//==============================================================================

template<typename T>
MemoryPool<T>::MemoryPool(size_t size) : pool_(size) {
    for (auto& item : pool_) {
        available_.push(&item);
    }
}

template<typename T>
MemoryPool<T>::~MemoryPool() = default;

template<typename T>
T* MemoryPool<T>::acquire() {
    std::lock_guard<std::mutex> lock(mutex_);
    if (available_.empty()) {
        return nullptr; // Pool exhausted
    }
    T* ptr = available_.front();
    available_.pop();
    return ptr;
}

template<typename T>
void MemoryPool<T>::release(T* ptr) {
    std::lock_guard<std::mutex> lock(mutex_);
    available_.push(ptr);
}

// Explicit template instantiations
template class trading::execution::MemoryPool<trading::execution::Order>;
template class trading::execution::MemoryPool<trading::execution::OrderFill>;

//==============================================================================
// OrderBookLevel Implementation  
//==============================================================================

void OrderBookLevel::set_price_size(const Price& price, double size) {
    price_ticks_.store(price.ticks());
    size_.store(size);
}

Price OrderBookLevel::get_price() const {
    return Price(price_ticks_.load());
}

double OrderBookLevel::get_size() const {
    return size_.load();
}

bool OrderBookLevel::is_valid() const {
    return price_ticks_.load() > 0 && size_.load() > 0;
}

//==============================================================================
// OrderBook Implementation
//==============================================================================

trading::execution::OrderBook::OrderBook(const std::string& symbol) : symbol_(symbol) {}

void trading::execution::OrderBook::update_bid(const Price& price, double size, size_t level) {
    if (level >= ORDER_BOOK_DEPTH) return;
    
    std::unique_lock<std::shared_mutex> lock(mutex_);
    bids_[level].set_price_size(price, size);
    set_last_update_time(std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::steady_clock::now().time_since_epoch()).count());
}

void trading::execution::OrderBook::update_ask(const Price& price, double size, size_t level) {
    if (level >= ORDER_BOOK_DEPTH) return;
    
    std::unique_lock<std::shared_mutex> lock(mutex_);
    asks_[level].set_price_size(price, size);
    set_last_update_time(std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::steady_clock::now().time_since_epoch()).count());
}

Price trading::execution::OrderBook::get_best_bid() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    return bids_[0].get_price();
}

Price trading::execution::OrderBook::get_best_ask() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    return asks_[0].get_price();
}

double trading::execution::OrderBook::get_bid_size(size_t level) const {
    if (level >= ORDER_BOOK_DEPTH) return 0.0;
    std::shared_lock<std::shared_mutex> lock(mutex_);
    return bids_[level].get_size();
}

double trading::execution::OrderBook::get_ask_size(size_t level) const {
    if (level >= ORDER_BOOK_DEPTH) return 0.0;
    std::shared_lock<std::shared_mutex> lock(mutex_);
    return asks_[level].get_size();
}

Price trading::execution::OrderBook::get_mid_price() const {
    Price bid = get_best_bid();
    Price ask = get_best_ask();
    return Price((bid.ticks() + ask.ticks()) / 2);
}

double trading::execution::OrderBook::get_spread() const {
    return (get_best_ask() - get_best_bid()).to_double();
}

bool trading::execution::OrderBook::has_sufficient_liquidity(OrderSide side, double quantity, const Price& limit_price) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    double available_quantity = 0.0;
    
    if (side == ORDER_SIDE_BUY) {
        // Check ask side
        for (const auto& level : asks_) {
            if (!level.is_valid()) break;
            if (level.get_price() > limit_price) break;
            
            available_quantity += level.get_size();
            if (available_quantity >= quantity) {
                return true;
            }
        }
    } else {
        // Check bid side
        for (const auto& level : bids_) {
            if (!level.is_valid()) break;
            if (level.get_price() < limit_price) break;
            
            available_quantity += level.get_size();
            if (available_quantity >= quantity) {
                return true;
            }
        }
    }
    
    return false;
}

std::vector<std::pair<Price, double>> trading::execution::OrderBook::get_fills_for_market_order(OrderSide side, double quantity) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    std::vector<std::pair<Price, double>> fills;
    
    double remaining_quantity = quantity;
    
    if (side == ORDER_SIDE_BUY) {
        // Take from ask side
        for (const auto& level : asks_) {
            if (!level.is_valid() || remaining_quantity <= 0) break;
            
            double fill_quantity = std::min(remaining_quantity, level.get_size());
            fills.emplace_back(level.get_price(), fill_quantity);
            remaining_quantity -= fill_quantity;
        }
    } else {
        // Take from bid side
        for (const auto& level : bids_) {
            if (!level.is_valid() || remaining_quantity <= 0) break;
            
            double fill_quantity = std::min(remaining_quantity, level.get_size());
            fills.emplace_back(level.get_price(), fill_quantity);
            remaining_quantity -= fill_quantity;
        }
    }
    
    return fills;
}

int64_t trading::execution::OrderBook::get_last_update_time() const {
    return last_update_time_.load();
}

void trading::execution::OrderBook::set_last_update_time(int64_t timestamp_ns) {
    last_update_time_.store(timestamp_ns);
}

//==============================================================================
// Order Implementation
//==============================================================================

Order::Order(const COrderRequest& request) 
    : order_id_(request.order_id)
    , symbol_(request.symbol) 
    , type_(request.order_type)
    , side_(request.side)
    , quantity_(request.quantity)
    , price_(request.price)
    , stop_price_(request.stop_price)
    , time_in_force_(request.time_in_force)
    , timestamp_ns_(request.timestamp_ns)
    , client_id_(request.client_id) {
}

void Order::set_status(OrderStatus status) {
    status_.store(status);
}

void Order::add_fill(const Price& price, double quantity, double fee) {
    std::lock_guard<std::mutex> lock(fill_mutex_);
    
    fills_.emplace_back(price, quantity);
    
    double old_filled = filled_quantity_.load();
    double new_filled = old_filled + quantity;
    filled_quantity_.store(new_filled);
    
    // Update average fill price using weighted average
    int64_t old_avg_ticks = average_fill_price_ticks_.load();
    int64_t new_avg_ticks = ((old_avg_ticks * old_filled) + (price.ticks() * quantity)) / new_filled;
    average_fill_price_ticks_.store(new_avg_ticks);
    
    // Update status
    if (new_filled >= quantity_) {
        set_status(ORDER_STATUS_FILLED);
    } else {
        set_status(ORDER_STATUS_PARTIALLY_FILLED);
    }
}

bool Order::is_fully_filled() const {
    return filled_quantity_.load() >= quantity_;
}

bool Order::is_active() const {
    OrderStatus status = status_.load();
    return status == ORDER_STATUS_SUBMITTED || status == ORDER_STATUS_PARTIALLY_FILLED;
}

bool Order::is_expired(int64_t current_time_ns) const {
    if (time_in_force_ == TIME_IN_FORCE_DAY) {
        // Check if it's past market close (simplified - assume 4 PM ET)
        // In real implementation, would check market hours
        return false; // Simplified for now
    }
    
    // Check order timeout
    return (current_time_ns - timestamp_ns_) > ORDER_TIMEOUT_NS;
}

Price Order::get_average_fill_price() const {
    return Price(average_fill_price_ticks_.load());
}

bool Order::validate() const {
    if (order_id_.empty() || symbol_.empty()) return false;
    if (quantity_ <= 0) return false;
    if (type_ == ORDER_TYPE_LIMIT && price_.ticks() <= 0) return false;
    if (type_ == ORDER_TYPE_STOP && stop_price_.ticks() <= 0) return false;
    return true;
}

double Order::get_remaining_quantity() const {
    return quantity_ - filled_quantity_.load();
}

//==============================================================================
// PerformanceMetrics Implementation
//==============================================================================

PerformanceMetrics::PerformanceMetrics() 
    : start_time_(std::chrono::steady_clock::now()) {
}

void PerformanceMetrics::record_order_processed(int64_t latency_micros, bool success) {
    total_orders_processed_.fetch_add(1);
    total_latency_micros_.fetch_add(latency_micros);
    
    if (success) {
        successful_executions_.fetch_add(1);
    } else {
        failed_executions_.fetch_add(1);
    }
    
    // Store latency sample for P99 calculation
    std::lock_guard<std::mutex> lock(latency_mutex_);
    latency_samples_.push_back(latency_micros);
    
    // Keep only recent samples (last 10000)
    if (latency_samples_.size() > 10000) {
        latency_samples_.erase(latency_samples_.begin(), 
                              latency_samples_.begin() + latency_samples_.size() - 10000);
    }
}

void PerformanceMetrics::record_fill(double quantity, const Price& price) {
    double current = total_volume_.load();
    double new_value = current + (quantity * price.to_double());
    total_volume_.store(new_value);
}

void PerformanceMetrics::record_memory_usage(size_t bytes) {
    memory_usage_bytes_.store(bytes);
}

void PerformanceMetrics::record_cpu_usage(double percentage) {
    cpu_usage_percent_.store(percentage);
}

CEngineMetrics PerformanceMetrics::get_metrics() const {
    CEngineMetrics metrics = {};
    
    metrics.total_orders_processed = total_orders_processed_.load();
    metrics.successful_executions = successful_executions_.load();
    metrics.failed_executions = failed_executions_.load();
    metrics.active_orders = active_orders_.load();
    metrics.memory_usage_bytes = memory_usage_bytes_.load();
    metrics.cpu_usage_percent = cpu_usage_percent_.load();
    
    // Calculate uptime
    auto now = std::chrono::steady_clock::now();
    auto uptime = std::chrono::duration_cast<std::chrono::seconds>(now - start_time_);
    metrics.uptime_seconds = uptime.count();
    
    // Calculate average latency
    uint64_t total_orders = total_orders_processed_.load();
    if (total_orders > 0) {
        metrics.average_latency_micros = static_cast<double>(total_latency_micros_.load()) / total_orders;
        
        // Calculate orders per second
        if (uptime.count() > 0) {
            metrics.orders_per_second = static_cast<double>(total_orders) / uptime.count();
        }
    }
    
    // Calculate P99 latency
    std::lock_guard<std::mutex> lock(latency_mutex_);
    if (!latency_samples_.empty()) {
        std::vector<int64_t> sorted_samples = latency_samples_;
        std::sort(sorted_samples.begin(), sorted_samples.end());
        
        size_t p99_index = static_cast<size_t>(sorted_samples.size() * 0.99);
        if (p99_index < sorted_samples.size()) {
            metrics.p99_latency_micros = sorted_samples[p99_index];
        }
    }
    
    return metrics;
}

void PerformanceMetrics::reset() {
    total_orders_processed_.store(0);
    successful_executions_.store(0);
    failed_executions_.store(0);
    active_orders_.store(0);
    total_latency_micros_.store(0);
    total_volume_.store(0);
    memory_usage_bytes_.store(0);
    cpu_usage_percent_.store(0);
    
    std::lock_guard<std::mutex> lock(latency_mutex_);
    latency_samples_.clear();
    
    start_time_ = std::chrono::steady_clock::now();
}

//==============================================================================
// ExecutionEngine Implementation
//==============================================================================

ExecutionEngine::ExecutionEngine() {
    metrics_ = std::make_unique<PerformanceMetrics>();
    order_pool_ = std::make_unique<MemoryPool<Order>>(MAX_CONCURRENT_ORDERS);
    fill_pool_ = std::make_unique<MemoryPool<OrderFill>>(MAX_CONCURRENT_ORDERS * 10);
}

ExecutionEngine::~ExecutionEngine() {
    stop();
}

ExecutionResult ExecutionEngine::initialize(const std::string& config_json) {
    std::unique_lock<std::shared_mutex> lock(engine_mutex_);
    
    try {
        // Simple JSON parsing alternative - just use defaults for now
        // In production, would use proper JSON library or custom parser
        if (!config_json.empty()) {
            // For now, just validate that config is provided
            // TODO: Implement proper JSON parsing or use simpler config format
        }
        
        // Initialize order books for common symbols
        std::vector<std::string> symbols = {"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"};
        for (const auto& symbol : symbols) {
            order_books_[symbol] = std::make_unique<OrderBook>(symbol);
        }
        
        initialized_.store(true);
        return EXEC_SUCCESS;
        
    } catch (const std::exception& e) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
}

ExecutionResult ExecutionEngine::start() {
    if (!initialized_.load()) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    std::unique_lock<std::shared_mutex> lock(engine_mutex_);
    
    if (running_.load()) {
        return EXEC_SUCCESS; // Already running
    }
    
    running_.store(true);
    healthy_.store(true);
    
    // Start worker threads
    worker_threads_.reserve(config_.worker_thread_count);
    for (int i = 0; i < config_.worker_thread_count; ++i) {
        worker_threads_.emplace_back([this]() { process_order_queue(); });
    }
    
    // Start market simulator if enabled
    if (config_.enable_simulation) {
        market_simulator_thread_ = std::thread([this]() { simulate_market_data(); });
    }
    
    return EXEC_SUCCESS;
}

ExecutionResult ExecutionEngine::stop() {
    running_.store(false);
    healthy_.store(false);
    
    // Notify all waiting threads
    order_cv_.notify_all();
    
    // Join all threads
    for (auto& thread : worker_threads_) {
        if (thread.joinable()) {
            thread.join();
        }
    }
    worker_threads_.clear();
    
    if (market_simulator_thread_.joinable()) {
        market_simulator_thread_.join();
    }
    
    return EXEC_SUCCESS;
}

ExecutionResult ExecutionEngine::submit_order(const COrderRequest& request, COrderResponse& response) {
    auto start_time = std::chrono::steady_clock::now();
    
    // Initialize response
    strncpy(response.order_id, request.order_id, sizeof(response.order_id) - 1);
    response.result = EXEC_SUCCESS;
    response.status = ORDER_STATUS_SUBMITTED;
    response.executed_quantity = 0.0;
    response.average_price = 0.0;
    response.execution_time_ns = 0;
    
    try {
        // Create order
        auto order = std::make_shared<Order>(request);
        
        // Validate order
        if (!order->validate()) {
            response.result = EXEC_ERROR_INVALID_ORDER;
            strncpy(response.message, "Invalid order parameters", sizeof(response.message) - 1);
            return EXEC_ERROR_INVALID_ORDER;
        }
        
        // Risk checks
        if (config_.enable_risk_checks) {
            if (order->get_quantity() > config_.max_position_size) {
                response.result = EXEC_ERROR_RISK_LIMIT_EXCEEDED;
                strncpy(response.message, "Order size exceeds risk limits", sizeof(response.message) - 1);
                return EXEC_ERROR_RISK_LIMIT_EXCEEDED;
            }
        }
        
        // Add to active orders
        {
            std::lock_guard<std::mutex> lock(order_mutex_);
            active_orders_[order->get_order_id()] = order;
            order_queue_.push(order);
        }
        
        order_cv_.notify_one();
        
        // For immediate execution (simplified)
        ExecutionResult exec_result = EXEC_SUCCESS;
        
        switch (order->get_type()) {
            case ORDER_TYPE_MARKET:
                exec_result = execute_market_order(order);
                break;
            case ORDER_TYPE_LIMIT:
                exec_result = execute_limit_order(order);
                break;
            case ORDER_TYPE_STOP:
                exec_result = execute_stop_order(order);
                break;
            default:
                exec_result = EXEC_ERROR_INVALID_ORDER;
                break;
        }
        
        // Update response
        response.result = exec_result;
        response.status = order->get_status();
        response.executed_quantity = order->get_filled_quantity();
        response.average_price = order->get_average_fill_price().to_double();
        
        auto end_time = std::chrono::steady_clock::now();
        auto latency = std::chrono::duration_cast<std::chrono::microseconds>(end_time - start_time);
        response.latency_micros = latency.count();
        response.execution_time_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(end_time.time_since_epoch()).count();
        
        // Record metrics
        metrics_->record_order_processed(latency.count(), exec_result == EXEC_SUCCESS);
        
        if (exec_result == EXEC_SUCCESS && response.executed_quantity > 0) {
            metrics_->record_fill(response.executed_quantity, Price(response.average_price));
        }
        
        return exec_result;
        
    } catch (const std::exception& e) {
        response.result = EXEC_ERROR_SYSTEM_ERROR;
        strncpy(response.message, "System error", sizeof(response.message) - 1);
        return EXEC_ERROR_SYSTEM_ERROR;
    }
}

ExecutionResult ExecutionEngine::cancel_order(const std::string& order_id) {
    std::lock_guard<std::mutex> lock(order_mutex_);
    
    auto it = active_orders_.find(order_id);
    if (it == active_orders_.end()) {
        return EXEC_ERROR_ORDER_NOT_FOUND;
    }
    
    auto order = it->second;
    if (!order->is_active()) {
        return EXEC_ERROR_INVALID_ORDER; // Already filled or cancelled
    }
    
    order->set_status(ORDER_STATUS_CANCELLED);
    active_orders_.erase(it);
    
    // Notify status callback
    if (status_callback_) {
        status_callback_(order_id, ORDER_STATUS_CANCELLED, "Order cancelled");
    }
    
    return EXEC_SUCCESS;
}

ExecutionResult ExecutionEngine::get_order_book(const std::string& symbol, OrderBook& book) {
    auto it = order_books_.find(symbol);
    if (it == order_books_.end()) {
        return EXEC_ERROR_INVALID_ORDER;
    }
    
    // Cannot copy OrderBook due to atomic and mutex members
    // Instead, update the provided book with current data
    auto& source = *it->second;
    
    // Copy price levels - just get best bid/ask for simplicity
    if (source.get_best_bid().ticks() > 0) {
        book.update_bid(source.get_best_bid(), source.get_bid_size(0), 0);
    }
    if (source.get_best_ask().ticks() > 0) {
        book.update_ask(source.get_best_ask(), source.get_ask_size(0), 0);
    }
    
    return EXEC_SUCCESS;
}

bool ExecutionEngine::is_healthy() const {
    return healthy_.load() && running_.load();
}

CEngineMetrics ExecutionEngine::get_metrics() const {
    auto metrics = metrics_->get_metrics();
    
    // Update active orders count - need to cast away const for mutex
    std::lock_guard<std::mutex> lock(const_cast<std::mutex&>(order_mutex_));
    metrics.active_orders = active_orders_.size();
    
    return metrics;
}

void ExecutionEngine::register_fill_callback(std::function<void(const OrderFill&)> callback) {
    fill_callback_ = callback;
}

void ExecutionEngine::register_status_callback(std::function<void(const std::string&, OrderStatus, const std::string&)> callback) {
    status_callback_ = callback;
}

ExecutionResult ExecutionEngine::execute_market_order(std::shared_ptr<Order> order) {
    auto book_it = order_books_.find(order->get_symbol());
    if (book_it == order_books_.end()) {
        return EXEC_ERROR_INVALID_ORDER;
    }
    
    auto& book = book_it->second;
    auto fills = book->get_fills_for_market_order(order->get_side(), order->get_quantity());
    
    if (fills.empty()) {
        return EXEC_ERROR_INSUFFICIENT_LIQUIDITY;
    }
    
    // Process fills
    for (const auto& fill : fills) {
        order->add_fill(fill.first, fill.second);
        
        // Notify fill callback
        if (fill_callback_) {
            // Create C++ OrderFill for internal callback
            OrderFill cpp_fill = {};
            cpp_fill.fill_id = "fill_123";
            cpp_fill.order_id = order->get_order_id();
            cpp_fill.price = fill.first.to_double();
            cpp_fill.quantity = fill.second;
            cpp_fill.fee = fill.second * 0.001; // 0.1% fee
            cpp_fill.timestamp_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
                std::chrono::steady_clock::now().time_since_epoch()).count();
            cpp_fill.venue = "SIM";
            
            fill_callback_(cpp_fill);
        }
    }
    
    return EXEC_SUCCESS;
}

ExecutionResult ExecutionEngine::execute_limit_order(std::shared_ptr<Order> order) {
    auto book_it = order_books_.find(order->get_symbol());
    if (book_it == order_books_.end()) {
        return EXEC_ERROR_INVALID_ORDER;
    }
    
    auto& book = book_it->second;
    
    // For limit orders, check if they can be immediately filled
    Price best_price = (order->get_side() == ORDER_SIDE_BUY) ? 
                       book->get_best_ask() : book->get_best_bid();
    
    bool can_fill = (order->get_side() == ORDER_SIDE_BUY && order->get_price() >= best_price) ||
                    (order->get_side() == ORDER_SIDE_SELL && order->get_price() <= best_price);
    
    if (can_fill && book->has_sufficient_liquidity(order->get_side(), order->get_quantity(), order->get_price())) {
        // Immediate fill
        order->add_fill(order->get_price(), order->get_quantity());
        return EXEC_SUCCESS;
    }
    
    // Order remains open (would be added to order book in real implementation)
    order->set_status(ORDER_STATUS_SUBMITTED);
    return EXEC_SUCCESS;
}

ExecutionResult ExecutionEngine::execute_stop_order(std::shared_ptr<Order> order) {
    // Simplified stop order logic - in real implementation would monitor price levels
    auto book_it = order_books_.find(order->get_symbol());
    if (book_it == order_books_.end()) {
        return EXEC_ERROR_INVALID_ORDER;
    }
    
    auto& book = book_it->second;
    Price current_price = book->get_mid_price();
    
    // Check if stop price is triggered
    bool triggered = (order->get_side() == ORDER_SIDE_BUY && current_price >= order->get_stop_price()) ||
                     (order->get_side() == ORDER_SIDE_SELL && current_price <= order->get_stop_price());
    
    if (triggered) {
        // Convert to market order
        order->add_fill(current_price, order->get_quantity());
        return EXEC_SUCCESS;
    }
    
    // Order remains pending
    order->set_status(ORDER_STATUS_SUBMITTED);
    return EXEC_SUCCESS;
}

void ExecutionEngine::process_order_queue() {
    while (running_.load()) {
        std::unique_lock<std::mutex> lock(order_mutex_);
        order_cv_.wait(lock, [this] { return !order_queue_.empty() || !running_.load(); });
        
        if (!running_.load()) break;
        
        while (!order_queue_.empty()) {
            auto order = order_queue_.front();
            order_queue_.pop();
            
            // Process order (already handled in submit_order for simplicity)
            // In real implementation, would have more sophisticated order processing
        }
    }
}

void ExecutionEngine::simulate_market_data() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_real_distribution<> price_change(-0.01, 0.01); // ±1% price change
    
    // Initialize market prices
    std::unordered_map<std::string, Price> prices = {
        {"AAPL", Price(150.0)},
        {"GOOGL", Price(2500.0)},
        {"MSFT", Price(300.0)},
        {"TSLA", Price(800.0)},
        {"AMZN", Price(3000.0)}
    };
    
    while (running_.load()) {
        for (auto& [symbol, price] : prices) {
            // Simulate price movement
            double change = price_change(gen);
            price = price * (1.0 + change);
            
            // Update order book
            auto book_it = order_books_.find(symbol);
            if (book_it != order_books_.end()) {
                auto& book = book_it->second;
                
                // Create simple bid/ask spread (0.1%)
                Price bid = price * 0.999;
                Price ask = price * 1.001;
                
                book->update_bid(bid, 1000.0, 0);
                book->update_ask(ask, 1000.0, 0);
            }
        }
        
        std::this_thread::sleep_for(std::chrono::milliseconds(100)); // 10Hz updates
    }
}

Price ExecutionEngine::get_simulated_price(const std::string& symbol) const {
    auto book_it = order_books_.find(symbol);
    if (book_it != order_books_.end()) {
        return book_it->second->get_mid_price();
    }
    return Price(100.0); // Default price
}

//==============================================================================
// C API Implementation
//==============================================================================

ExecutionResult engine_initialize(const char* config_json) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine) {
        g_engine = std::make_unique<ExecutionEngine>();
    }
    
    std::string config = config_json ? config_json : "";
    return g_engine->initialize(config);
}

ExecutionResult engine_start(void) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    return g_engine->start();
}

ExecutionResult engine_stop(void) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine) {
        return EXEC_SUCCESS;
    }
    
    ExecutionResult result = g_engine->stop();
    g_engine.reset();
    return result;
}

ExecutionResult engine_submit_order(const COrderRequest* request, COrderResponse* response) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine || !request || !response) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    return g_engine->submit_order(*request, *response);
}

ExecutionResult engine_cancel_order(const char* order_id) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine || !order_id) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    return g_engine->cancel_order(order_id);
}

ExecutionResult engine_get_order_book(const char* symbol, COrderBook* book) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine || !symbol || !book) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    trading::execution::OrderBook cpp_book(symbol);
    ExecutionResult result = g_engine->get_order_book(symbol, cpp_book);
    
    if (result == EXEC_SUCCESS) {
        strncpy(book->symbol, symbol, sizeof(book->symbol) - 1);
        book->timestamp_ns = cpp_book.get_last_update_time();
        book->bid_price = cpp_book.get_best_bid().to_double();
        book->ask_price = cpp_book.get_best_ask().to_double();
        book->bid_size = cpp_book.get_bid_size();
        book->ask_size = cpp_book.get_ask_size();
        book->last_price = cpp_book.get_mid_price().to_double();
        book->last_size = 0.0; // Not tracked in this implementation
    }
    
    return result;
}

ExecutionResult engine_get_metrics(CEngineMetrics* metrics) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine || !metrics) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    *metrics = g_engine->get_metrics();
    return EXEC_SUCCESS;
}

int engine_is_healthy(void) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine) {
        return 0;
    }
    
    return g_engine->is_healthy() ? 1 : 0;
}

ExecutionResult engine_register_fill_callback(FillCallback callback) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    if (callback) {
        g_engine->register_fill_callback([callback](const OrderFill& fill) {
            // Convert C++ OrderFill to COrderFill for C callback
            COrderFill c_fill = {};
            strncpy(c_fill.fill_id, fill.fill_id.c_str(), sizeof(c_fill.fill_id) - 1);
            strncpy(c_fill.order_id, fill.order_id.c_str(), sizeof(c_fill.order_id) - 1);
            c_fill.price = fill.price;
            c_fill.quantity = fill.quantity;
            c_fill.fee = fill.fee;
            c_fill.timestamp_ns = fill.timestamp_ns;
            strncpy(c_fill.venue, fill.venue.c_str(), sizeof(c_fill.venue) - 1);
            callback(&c_fill);
        });
    }
    
    return EXEC_SUCCESS;
}

ExecutionResult engine_register_status_callback(StatusCallback callback) {
    std::lock_guard<std::mutex> lock(g_engine_mutex);
    
    if (!g_engine) {
        return EXEC_ERROR_SYSTEM_ERROR;
    }
    
    if (callback) {
        g_engine->register_status_callback([callback](const std::string& order_id, OrderStatus status, const std::string& message) {
            callback(order_id.c_str(), status, message.c_str());
        });
    }
    
    return EXEC_SUCCESS;
}