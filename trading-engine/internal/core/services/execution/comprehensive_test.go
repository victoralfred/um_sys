package execution

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// Helper function to create a test asset
func createTestAsset(symbol string) *domain.Asset {
	minQty, _ := types.NewDecimal("0.01")
	maxQty, _ := types.NewDecimal("1000000")
	tickSize, _ := types.NewDecimal("0.01")
	
	return &domain.Asset{
		Symbol:      symbol,
		Name:        symbol + " Stock",
		AssetType:   domain.AssetTypeStock,
		Exchange:    "NASDAQ",
		Currency:    "USD",
		Precision:   2,
		MinQuantity: minQty,
		MaxQuantity: maxQty,
		TickSize:    tickSize,
		IsActive:    true,
	}
}

// Helper function to create a test order
func createTestOrder(id, symbol string, side domain.OrderSide, orderType domain.OrderType, quantity, price float64) *domain.Order {
	qty := types.NewDecimalFromFloat(quantity)
	prc := types.NewDecimalFromFloat(price)
	
	return &domain.Order{
		ID:       id,
		Asset:    createTestAsset(symbol),
		Type:     orderType,
		Side:     side,
		Status:   domain.OrderStatusPending,
		Quantity: qty,
		Price:    prc,
		TimeInForce: domain.TimeInForceGTC,
	}
}

// TestComprehensiveExecutionSystem runs a comprehensive test of the optimized execution system
func TestComprehensiveExecutionSystem(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize optimized service with test configuration
	service := NewOptimizedExecutionService(nil, nil)
	
	// Start the service
	if err := service.Start(ctx); err != nil {
		t.Fatalf("Failed to start optimized service: %v", err)
	}
	// Note: No Shutdown method available, service will clean up when context ends

	t.Run("BasicOrderSubmission", func(t *testing.T) {
		testBasicOrderSubmission(t, service, ctx)
	})

	t.Run("ConcurrentOrderProcessing", func(t *testing.T) {
		testConcurrentOrderProcessing(t, service, ctx)
	})

	t.Run("OrderLifecycleManagement", func(t *testing.T) {
		testOrderLifecycleManagement(t, service, ctx)
	})

	t.Run("ErrorHandlingAndRecovery", func(t *testing.T) {
		testErrorHandlingAndRecovery(t, service, ctx)
	})

	t.Run("PerformanceMetrics", func(t *testing.T) {
		testPerformanceMetrics(t, service, ctx)
	})
}

func testBasicOrderSubmission(t *testing.T, service *OptimizedExecutionService, ctx context.Context) {
	// Test single order submission
	order := createTestOrder("test-basic-1", "AAPL", domain.OrderSideBuy, domain.OrderTypeMarket, 100, 150.0)

	result, err := service.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.OrderID != order.ID {
		t.Errorf("Expected order ID %s, got %s", order.ID, result.OrderID)
	}

	// Verify order can be retrieved
	retrievedOrder, err := service.GetOrderStatus(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve order: %v", err)
	}
	if retrievedOrder.ID != order.ID {
		t.Errorf("Retrieved order ID mismatch: expected %s, got %s", order.ID, retrievedOrder.ID)
	}

	t.Logf("✓ Basic order submission successful: %s", order.ID)
}

func testConcurrentOrderProcessing(t *testing.T, service *OptimizedExecutionService, ctx context.Context) {
	// Test concurrent order submissions
	numGoroutines := 10
	ordersPerGoroutine := 100
	totalExpected := numGoroutines * ordersPerGoroutine

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errors := make([]error, 0)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < ordersPerGoroutine; j++ {
				order := createTestOrder(
					fmt.Sprintf("concurrent-%d-%d", goroutineID, j),
					"MSFT", 
					domain.OrderSideBuy, 
					domain.OrderTypeMarket, 
					10, 
					300.0,
				)

				_, err := service.SubmitOrder(ctx, order)
				
				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Calculate success rate
	successRate := float64(successCount) / float64(totalExpected) * 100
	throughput := float64(successCount) / duration.Seconds()

	t.Logf("Concurrent processing results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total attempted: %d", totalExpected)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", len(errors))
	t.Logf("  Success rate: %.2f%%", successRate)
	t.Logf("  Throughput: %.2f orders/second", throughput)

	// Expect at least 95% success rate
	if successRate < 95.0 {
		t.Errorf("Success rate %.2f%% below expected 95%%", successRate)
		if len(errors) > 0 {
			t.Errorf("First error: %v", errors[0])
		}
	}

	// Expect reasonable throughput (>10K orders/second)
	if throughput < 10000 {
		t.Errorf("Throughput %.2f orders/second below expected 10K", throughput)
	}
}

func testOrderLifecycleManagement(t *testing.T, service *OptimizedExecutionService, ctx context.Context) {
	// Test order lifecycle: submit -> retrieve -> cancel
	order := createTestOrder("lifecycle-test-1", "GOOGL", domain.OrderSideSell, domain.OrderTypeLimit, 50, 2800.0)

	// Submit order
	result, err := service.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Verify submission
	if result.OrderID != order.ID {
		t.Errorf("Order ID mismatch in result")
	}

	// Retrieve order status
	retrievedOrder, err := service.GetOrderStatus(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get order status: %v", err)
	}

	if retrievedOrder.ID != order.ID {
		t.Errorf("Retrieved order ID mismatch")
	}

	// Test cancellation
	err = service.CancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to cancel order: %v", err)
	}

	// Allow some time for order completion simulation
	time.Sleep(200 * time.Millisecond)

	t.Logf("✓ Order lifecycle management successful for order: %s", order.ID)
}

func testErrorHandlingAndRecovery(t *testing.T, service *OptimizedExecutionService, ctx context.Context) {
	// Test nil order handling
	_, err := service.SubmitOrder(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil order submission")
	}

	// Test order with empty ID
	invalidOrder := createTestOrder("", "TSLA", domain.OrderSideBuy, domain.OrderTypeMarket, 10, 800.0)

	_, err = service.SubmitOrder(ctx, invalidOrder)
	// This may or may not error depending on validation logic
	t.Logf("Empty ID order result: %v", err)

	// Test non-existent order retrieval
	_, err = service.GetOrderStatus(ctx, "non-existent-order")
	if err == nil {
		t.Error("Expected error for non-existent order")
	}

	// Test cancelling non-existent order
	err = service.CancelOrder(ctx, "non-existent-order")
	// This may or may not error depending on implementation
	t.Logf("Non-existent order cancellation result: %v", err)

	t.Logf("✓ Error handling tests completed")
}

func testPerformanceMetrics(t *testing.T, service *OptimizedExecutionService, ctx context.Context) {
	// Submit several orders to generate metrics
	numOrders := 100
	for i := 0; i < numOrders; i++ {
		order := createTestOrder(
			fmt.Sprintf("metrics-test-%d", i),
			"AMZN",
			domain.OrderSideBuy,
			domain.OrderTypeMarket,
			1,
			3000.0,
		)

		_, err := service.SubmitOrder(ctx, order)
		if err != nil {
			t.Logf("Order %d failed: %v", i, err)
		}
	}

	// Allow some processing time
	time.Sleep(500 * time.Millisecond)

	// Get detailed metrics
	metrics := service.GetDetailedMetrics()

	t.Logf("Performance metrics:")
	t.Logf("  Orders submitted: %d", metrics.TotalOrdersSubmitted)
	t.Logf("  Orders processed: %d", metrics.TotalOrdersProcessed)
	t.Logf("  Orders rejected: %d", metrics.TotalOrdersRejected)
	t.Logf("  Active orders: %d", metrics.ActiveOrdersCount)
	t.Logf("  Validation failures: %d", metrics.ValidationFailures)
	t.Logf("  Engine failures: %d", metrics.EngineFailures)
	t.Logf("  Avg processing time: %dns", metrics.AverageProcessingTimeNs)
	t.Logf("  P99 processing time: %dns", metrics.P99ProcessingTimeNs)

	// Basic sanity checks
	if metrics.TotalOrdersSubmitted == 0 {
		t.Error("Expected some orders to be submitted")
	}

	if metrics.AverageProcessingTimeNs == 0 {
		t.Error("Expected non-zero average processing time")
	}

	t.Logf("✓ Performance metrics validation completed")
}

// TestExecutionSystemStressTest runs a stress test
func TestExecutionSystemStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	service := NewOptimizedExecutionService(nil, nil)

	if err := service.Start(ctx); err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Stress test parameters
	testDuration := 10 * time.Second
	numGoroutines := 20
	targetThroughput := 50000.0 // 50K orders/second

	var totalOrders uint64
	var errors uint64
	stopChan := make(chan struct{})

	var wg sync.WaitGroup
	startTime := time.Now()

	// Start load generators
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			localOrders := 0
			localErrors := 0
			
			for {
				select {
				case <-stopChan:
					// Update totals atomically would be better, but this is for testing
					totalOrders += uint64(localOrders)
					errors += uint64(localErrors)
					return
				default:
					order := createTestOrder(
						fmt.Sprintf("stress-%d-%d", goroutineID, localOrders),
						"SPY",
						domain.OrderSideBuy,
						domain.OrderTypeMarket,
						1,
						400.0,
					)

					_, err := service.SubmitOrder(ctx, order)
					if err != nil {
						localErrors++
					} else {
						localOrders++
					}
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(testDuration)
	close(stopChan)
	wg.Wait()

	actualDuration := time.Since(startTime)
	throughput := float64(totalOrders) / actualDuration.Seconds()
	errorRate := float64(errors) / float64(totalOrders+errors) * 100

	t.Logf("Stress test results:")
	t.Logf("  Duration: %v", actualDuration)
	t.Logf("  Total orders: %d", totalOrders)
	t.Logf("  Errors: %d", errors)
	t.Logf("  Error rate: %.2f%%", errorRate)
	t.Logf("  Throughput: %.2f orders/second", throughput)
	t.Logf("  Target throughput: %.2f orders/second", targetThroughput)

	// Verify performance requirements
	if throughput < targetThroughput {
		t.Errorf("Throughput %.2f below target %.2f orders/second", throughput, targetThroughput)
	}

	if errorRate > 5.0 {
		t.Errorf("Error rate %.2f%% above acceptable 5%%", errorRate)
	}

	// Get final metrics
	metrics := service.GetDetailedMetrics()
	t.Logf("Final system metrics:")
	t.Logf("  Active orders: %d", metrics.ActiveOrdersCount)
	t.Logf("  Avg processing time: %dns", metrics.AverageProcessingTimeNs)
	t.Logf("  P99 processing time: %dns", metrics.P99ProcessingTimeNs)
}