#include "order_engine.h"
#include <iostream>
#include <cassert>
#include <cstring>
#include <thread>
#include <chrono>

// Simple test for the C++ execution engine
int main() {
    std::cout << "Testing C++ Execution Engine..." << std::endl;

    // Test 1: Initialize engine
    ExecutionResult result = engine_initialize("{}");
    assert(result == EXEC_SUCCESS);
    std::cout << "✓ Engine initialization successful" << std::endl;

    // Test 2: Start engine
    result = engine_start();
    assert(result == EXEC_SUCCESS);
    std::cout << "✓ Engine start successful" << std::endl;

    // Give market simulator time to initialize prices
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    // Test 3: Check engine health
    int healthy = engine_is_healthy();
    assert(healthy == 1);
    std::cout << "✓ Engine is healthy" << std::endl;

    // Test 3.5: Check order book has data before submitting order
    COrderBook initial_book = {};
    result = engine_get_order_book("AAPL", &initial_book);
    assert(result == EXEC_SUCCESS);
    std::cout << "✓ Initial order book check" << std::endl;
    std::cout << "  Bid Price: " << initial_book.bid_price << std::endl;
    std::cout << "  Ask Price: " << initial_book.ask_price << std::endl;

    // Test 4: Create and submit market order
    COrderRequest request = {};
    strcpy(request.order_id, "TEST_001");
    strcpy(request.symbol, "AAPL");
    strcpy(request.client_id, "TEST_CLIENT");
    request.order_type = ORDER_TYPE_MARKET;
    request.side = ORDER_SIDE_BUY;
    request.quantity = 100.0;
    request.price = 0.0; // Market order - price ignored
    request.time_in_force = TIME_IN_FORCE_IOC;
    request.timestamp_ns = 1691437200000000000LL; // Aug 7, 2024

    COrderResponse response = {};
    result = engine_submit_order(&request, &response);
    if (result != EXEC_SUCCESS) {
        std::cout << "Order submission failed with result: " << result << std::endl;
        std::cout << "Response message: " << response.message << std::endl;
    }
    assert(result == EXEC_SUCCESS);
    assert(response.status == ORDER_STATUS_FILLED || response.status == ORDER_STATUS_PARTIALLY_FILLED);
    std::cout << "✓ Market order execution successful" << std::endl;
    std::cout << "  Order ID: " << response.order_id << std::endl;
    std::cout << "  Status: " << response.status << std::endl;
    std::cout << "  Executed Quantity: " << response.executed_quantity << std::endl;
    std::cout << "  Average Price: " << response.average_price << std::endl;

    // Test 5: Get order book
    COrderBook book = {};
    result = engine_get_order_book("AAPL", &book);
    assert(result == EXEC_SUCCESS);
    std::cout << "✓ Order book retrieval successful" << std::endl;
    std::cout << "  Symbol: " << book.symbol << std::endl;
    std::cout << "  Bid Price: " << book.bid_price << std::endl;
    std::cout << "  Ask Price: " << book.ask_price << std::endl;

    // Test 6: Get engine metrics
    CEngineMetrics metrics = {};
    result = engine_get_metrics(&metrics);
    assert(result == EXEC_SUCCESS);
    assert(metrics.total_orders_processed >= 1);
    std::cout << "✓ Engine metrics retrieval successful" << std::endl;
    std::cout << "  Total Orders Processed: " << metrics.total_orders_processed << std::endl;
    std::cout << "  Successful Executions: " << metrics.successful_executions << std::endl;
    std::cout << "  Average Latency: " << metrics.average_latency_micros << "µs" << std::endl;

    // Test 7: Stop engine
    result = engine_stop();
    assert(result == EXEC_SUCCESS);
    std::cout << "✓ Engine stop successful" << std::endl;

    std::cout << "\n🎉 All C++ execution engine tests passed!" << std::endl;
    return 0;
}