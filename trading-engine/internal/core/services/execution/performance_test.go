package execution

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// TDD Performance Tests - Define performance requirements first

func BenchmarkExecutionServiceSubmitOrder(b *testing.B) {
	// Target: <1ms p99 latency for order submission
	ctx := context.Background()
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}
	
	config := ServiceConfig{
		MaxConcurrentOrders:   100000,
		OrderTimeout:          30 * time.Second,
		EnableMetrics:         true,
		EnableValidation:      true,
		MaxRetryAttempts:      0, // Disable retries for pure performance test
		RetryBackoffDuration:  0,
	}
	
	service := NewExecutionServiceWithConfig(mockEngine, mockValidator, config)
	err := service.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	asset := &domain.Asset{
		Symbol:    "PERF_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		orderID := 0
		for pb.Next() {
			orderID++
			order := &domain.Order{
				ID:            fmt.Sprintf("PERF_ORDER_%d_%d", b.N, orderID),
				Asset:         asset,
				Type:          domain.OrderTypeMarket,
				Side:          domain.OrderSideBuy,
				Quantity:      types.NewDecimalFromFloat(100.0),
				TimeInForce:   domain.TimeInForceIOC,
				ClientOrderID: fmt.Sprintf("PERF_CLIENT_%d", orderID),
				CreatedAt:     time.Now(),
				Status:        domain.OrderStatusPending,
			}
			
			_, err := service.SubmitOrder(ctx, order)
			if err != nil {
				b.Errorf("Failed to submit order: %v", err)
			}
		}
	})
}

func BenchmarkOrderManagerSubmitOrder(b *testing.B) {
	// Target: <100μs p99 latency for order state management
	ctx := context.Background()
	manager := NewOrderManager()
	
	asset := &domain.Asset{
		Symbol:    "OM_PERF_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		orderID := 0
		for pb.Next() {
			orderID++
			order := &domain.Order{
				ID:            fmt.Sprintf("OM_PERF_ORDER_%d_%d", b.N, orderID),
				Asset:         asset,
				Type:          domain.OrderTypeLimit,
				Side:          domain.OrderSideBuy,
				Quantity:      types.NewDecimalFromFloat(50.0),
				Price:         types.NewDecimalFromFloat(100.0),
				TimeInForce:   domain.TimeInForceGTC,
				ClientOrderID: fmt.Sprintf("OM_PERF_CLIENT_%d", orderID),
				CreatedAt:     time.Now(),
				Status:        domain.OrderStatusPending,
			}
			
			err := manager.SubmitOrder(ctx, order)
			if err != nil {
				b.Errorf("Failed to submit order to manager: %v", err)
			}
		}
	})
	
	// Cleanup
	manager.Stop()
}

func BenchmarkSlippageEstimation(b *testing.B) {
	// Target: <50μs p99 latency for slippage estimation
	ctx := context.Background()
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "SLIP_PERF_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(100.00),
		AskPrice:       types.NewDecimalFromFloat(100.05),
		BidSize:        types.NewDecimalFromFloat(1000.0),
		AskSize:        types.NewDecimalFromFloat(800.0),
		LastTradePrice: types.NewDecimalFromFloat(100.02),
		Timestamp:      time.Now(),
	}
	
	order := &domain.Order{
		ID:            "SLIP_PERF_ORDER",
		Asset:         asset,
		Type:          domain.OrderTypeMarket,
		Side:          domain.OrderSideBuy,
		Quantity:      types.NewDecimalFromFloat(200.0),
		TimeInForce:   domain.TimeInForceIOC,
		ClientOrderID: "SLIP_PERF_CLIENT",
		CreatedAt:     time.Now(),
		Status:        domain.OrderStatusPending,
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := estimator.EstimateSlippage(ctx, order, marketData)
			if err != nil {
				b.Errorf("Failed to estimate slippage: %v", err)
			}
		}
	})
}

func TestExecutionSystemThroughput(t *testing.T) {
	// Target: >50,000 orders/second sustained throughput
	ctx := context.Background()
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}
	
	config := ServiceConfig{
		MaxConcurrentOrders:   200000,
		OrderTimeout:          30 * time.Second,
		EnableMetrics:         true,
		EnableValidation:      false, // Disable for max throughput
		MaxRetryAttempts:      0,
		RetryBackoffDuration:  0,
	}
	
	service := NewExecutionServiceWithConfig(mockEngine, mockValidator, config)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	asset := &domain.Asset{
		Symbol:    "THROUGHPUT_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	// Test parameters
	duration := 5 * time.Second
	numGoroutines := runtime.NumCPU() * 4 // Oversubscribe for I/O bound work
	targetThroughput := 50000.0 // orders/second
	
	var totalOrders int64
	var wg sync.WaitGroup
	startTime := time.Now()
	stopChan := make(chan struct{})
	
	// Start time-limited test
	go func() {
		time.Sleep(duration)
		close(stopChan)
	}()
	
	// Launch goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			orderID := 0
			
			for {
				select {
				case <-stopChan:
					return
				default:
					orderID++
					order := &domain.Order{
						ID:            fmt.Sprintf("THROUGHPUT_%d_%d", goroutineID, orderID),
						Asset:         asset,
						Type:          domain.OrderTypeMarket,
						Side:          domain.OrderSideBuy,
						Quantity:      types.NewDecimalFromFloat(100.0),
						TimeInForce:   domain.TimeInForceIOC,
						ClientOrderID: fmt.Sprintf("THROUGHPUT_CLIENT_%d_%d", goroutineID, orderID),
						CreatedAt:     time.Now(),
						Status:        domain.OrderStatusPending,
					}
					
					_, err := service.SubmitOrder(ctx, order)
					if err != nil {
						t.Errorf("Failed to submit order: %v", err)
						return
					}
					totalOrders++
				}
			}
		}(i)
	}
	
	wg.Wait()
	actualDuration := time.Since(startTime)
	
	actualThroughput := float64(totalOrders) / actualDuration.Seconds()
	
	t.Logf("Throughput test results:")
	t.Logf("  Duration: %v", actualDuration)
	t.Logf("  Total orders: %d", totalOrders)
	t.Logf("  Throughput: %.2f orders/second", actualThroughput)
	t.Logf("  Goroutines: %d", numGoroutines)
	
	if actualThroughput < targetThroughput {
		t.Errorf("Throughput below target: got %.2f, want %.2f orders/second", actualThroughput, targetThroughput)
	}
}

func TestMemoryUsageUnderLoad(t *testing.T) {
	// Target: <2GB memory usage for 100k active orders
	ctx := context.Background()
	manager := NewOrderManager()
	defer manager.Stop()
	
	asset := &domain.Asset{
		Symbol:    "MEMORY_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	// Get baseline memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	// Submit 100k orders
	numOrders := 100000
	for i := 0; i < numOrders; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("MEMORY_ORDER_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeLimit,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(100.0),
			Price:         types.NewDecimalFromFloat(50.0),
			TimeInForce:   domain.TimeInForceGTC,
			ClientOrderID: fmt.Sprintf("MEMORY_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}
		
		err := manager.SubmitOrder(ctx, order)
		if err != nil {
			t.Fatalf("Failed to submit order %d: %v", i, err)
		}
		
		// Trigger GC every 10k orders to get accurate measurement
		if i%10000 == 0 && i > 0 {
			runtime.GC()
		}
	}
	
	// Force GC and measure final memory
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	memoryUsed := m2.Alloc - m1.Alloc
	memoryPerOrder := float64(memoryUsed) / float64(numOrders)
	totalMemoryMB := float64(m2.Alloc) / 1024 / 1024
	
	t.Logf("Memory usage results:")
	t.Logf("  Orders submitted: %d", numOrders)
	t.Logf("  Memory used for orders: %.2f MB", float64(memoryUsed)/1024/1024)
	t.Logf("  Memory per order: %.2f bytes", memoryPerOrder)
	t.Logf("  Total memory: %.2f MB", totalMemoryMB)
	t.Logf("  Heap objects: %d", m2.HeapObjects)
	
	// Fail if memory usage is excessive
	maxMemoryMB := 2000.0 // 2GB limit
	if totalMemoryMB > maxMemoryMB {
		t.Errorf("Memory usage too high: %.2f MB > %.2f MB", totalMemoryMB, maxMemoryMB)
	}
	
	maxMemoryPerOrder := 500.0 // 500 bytes per order max
	if memoryPerOrder > maxMemoryPerOrder {
		t.Errorf("Memory per order too high: %.2f bytes > %.2f bytes", memoryPerOrder, maxMemoryPerOrder)
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Target: Handle 1000+ concurrent goroutines without deadlocks
	ctx := context.Background()
	service := NewExecutionService(&MockExecutionEngine{}, &MockOrderValidator{})
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	manager := NewOrderManager()
	defer manager.Stop()
	
	estimator := NewSlippageEstimator()
	
	asset := &domain.Asset{
		Symbol:    "CONCURRENT_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	marketData := &ports.MarketData{
		Asset:          asset,
		BidPrice:       types.NewDecimalFromFloat(100.00),
		AskPrice:       types.NewDecimalFromFloat(100.05),
		BidSize:        types.NewDecimalFromFloat(1000.0),
		AskSize:        types.NewDecimalFromFloat(800.0),
		LastTradePrice: types.NewDecimalFromFloat(100.02),
		Timestamp:      time.Now(),
	}
	
	numGoroutines := 1000
	operationsPerGoroutine := 100
	var wg sync.WaitGroup
	
	startTime := time.Now()
	
	// Launch concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				order := &domain.Order{
					ID:            fmt.Sprintf("CONCURRENT_%d_%d", goroutineID, j),
					Asset:         asset,
					Type:          domain.OrderTypeMarket,
					Side:          domain.OrderSideBuy,
					Quantity:      types.NewDecimalFromFloat(100.0),
					TimeInForce:   domain.TimeInForceIOC,
					ClientOrderID: fmt.Sprintf("CONCURRENT_CLIENT_%d_%d", goroutineID, j),
					CreatedAt:     time.Now(),
					Status:        domain.OrderStatusPending,
				}
				
				// Test service
				_, err := service.SubmitOrder(ctx, order)
				if err != nil {
					t.Errorf("Service submit failed: %v", err)
					return
				}
				
				// Test manager
				err = manager.SubmitOrder(ctx, order)
				if err != nil {
					t.Errorf("Manager submit failed: %v", err)
					return
				}
				
				// Test estimator
				_, err = estimator.EstimateSlippage(ctx, order, marketData)
				if err != nil {
					t.Errorf("Estimator failed: %v", err)
					return
				}
				
				// Test order retrieval
				_, err = manager.GetOrder(ctx, order.ID)
				if err != nil {
					t.Errorf("Order retrieval failed: %v", err)
					return
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	totalOperations := numGoroutines * operationsPerGoroutine * 4 // 4 operations per iteration
	opsPerSecond := float64(totalOperations) / duration.Seconds()
	
	t.Logf("Concurrent access test results:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Operations per goroutine: %d", operationsPerGoroutine)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations per second: %.2f", opsPerSecond)
	
	// Verify no deadlocks occurred (test completed successfully)
	if duration > 30*time.Second {
		t.Errorf("Test took too long, possible deadlock: %v", duration)
	}
}

func TestLatencyDistribution(t *testing.T) {
	// Target: p99 < 1ms, p95 < 500μs, median < 100μs
	ctx := context.Background()
	service := NewExecutionService(&MockExecutionEngine{}, &MockOrderValidator{})
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)
	
	asset := &domain.Asset{
		Symbol:    "LATENCY_TEST",
		AssetType: domain.AssetTypeStock,
	}
	
	numSamples := 10000
	latencies := make([]time.Duration, numSamples)
	
	// Warm up
	for i := 0; i < 1000; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("WARMUP_ORDER_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeMarket,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(100.0),
			TimeInForce:   domain.TimeInForceIOC,
			ClientOrderID: fmt.Sprintf("WARMUP_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}
		service.SubmitOrder(ctx, order)
	}
	
	// Measure latencies
	for i := 0; i < numSamples; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("LATENCY_ORDER_%d", i),
			Asset:         asset,
			Type:          domain.OrderTypeMarket,
			Side:          domain.OrderSideBuy,
			Quantity:      types.NewDecimalFromFloat(100.0),
			TimeInForce:   domain.TimeInForceIOC,
			ClientOrderID: fmt.Sprintf("LATENCY_CLIENT_%d", i),
			CreatedAt:     time.Now(),
			Status:        domain.OrderStatusPending,
		}
		
		start := time.Now()
		_, err := service.SubmitOrder(ctx, order)
		latency := time.Since(start)
		
		if err != nil {
			t.Errorf("Failed to submit order %d: %v", i, err)
		}
		
		latencies[i] = latency
	}
	
	// Calculate percentiles (simple sorting-based approach)
	// For production, would use more efficient percentile calculation
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	
	// Simple bubble sort for demonstration
	n := len(sortedLatencies)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if sortedLatencies[j] > sortedLatencies[j+1] {
				sortedLatencies[j], sortedLatencies[j+1] = sortedLatencies[j+1], sortedLatencies[j]
			}
		}
	}
	
	p50 := sortedLatencies[len(sortedLatencies)/2]
	p95 := sortedLatencies[int(float64(len(sortedLatencies))*0.95)]
	p99 := sortedLatencies[int(float64(len(sortedLatencies))*0.99)]
	
	t.Logf("Latency distribution results:")
	t.Logf("  Samples: %d", numSamples)
	t.Logf("  Median (p50): %v", p50)
	t.Logf("  95th percentile: %v", p95)
	t.Logf("  99th percentile: %v", p99)
	
	// Check targets
	if p99 > time.Millisecond {
		t.Errorf("p99 latency too high: %v > 1ms", p99)
	}
	
	if p95 > 500*time.Microsecond {
		t.Errorf("p95 latency too high: %v > 500μs", p95)
	}
	
	if p50 > 100*time.Microsecond {
		t.Errorf("median latency too high: %v > 100μs", p50)
	}
}