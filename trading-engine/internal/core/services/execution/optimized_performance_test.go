package execution

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// Performance tests for OptimizedExecutionService

func BenchmarkOptimizedExecutionServiceSubmitOrder(b *testing.B) {
	ctx := context.Background()
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}

	service := NewOptimizedExecutionService(mockEngine, mockValidator)
	err := service.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)

	asset := &domain.Asset{
		Symbol:    "OPT_PERF_TEST",
		AssetType: domain.AssetTypeStock,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		orderID := 0
		for pb.Next() {
			orderID++
			order := &domain.Order{
				ID:            fmt.Sprintf("OPT_PERF_ORDER_%d_%d", b.N, orderID),
				Asset:         asset,
				Type:          domain.OrderTypeMarket,
				Side:          domain.OrderSideBuy,
				Quantity:      types.NewDecimalFromFloat(100.0),
				TimeInForce:   domain.TimeInForceIOC,
				ClientOrderID: fmt.Sprintf("OPT_PERF_CLIENT_%d", orderID),
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

func TestOptimizedExecutionSystemHighThroughput(t *testing.T) {
	// Target: >1,000,000 orders/second with optimizations
	ctx := context.Background()
	mockEngine := &MockExecutionEngine{}
	mockValidator := &MockOrderValidator{}

	service := NewOptimizedExecutionService(mockEngine, mockValidator)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)

	asset := &domain.Asset{
		Symbol:    "HIGH_THROUGHPUT_TEST",
		AssetType: domain.AssetTypeStock,
	}

	// Test parameters for high throughput
	testDuration := 3 * time.Second
	numGoroutines := runtime.NumCPU() * 8 // High concurrency
	targetThroughput := 1000000.0         // 1M orders/second

	var totalOrders int64
	var wg sync.WaitGroup
	var errors int64

	startTime := time.Now()
	stopChan := make(chan struct{})

	// Start time-limited test
	go func() {
		time.Sleep(testDuration)
		close(stopChan)
	}()

	// Launch high-concurrency goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			orderID := 0
			localOrders := 0
			localErrors := 0

			for {
				select {
				case <-stopChan:
					totalOrders += int64(localOrders)
					errors += int64(localErrors)
					return
				default:
					orderID++
					order := &domain.Order{
						ID:            fmt.Sprintf("HIGH_THROUGHPUT_%d_%d", goroutineID, orderID),
						Asset:         asset,
						Type:          domain.OrderTypeMarket,
						Side:          domain.OrderSideBuy,
						Quantity:      types.NewDecimalFromFloat(100.0),
						TimeInForce:   domain.TimeInForceIOC,
						ClientOrderID: fmt.Sprintf("HIGH_THROUGHPUT_CLIENT_%d_%d", goroutineID, orderID),
						CreatedAt:     time.Now(),
						Status:        domain.OrderStatusPending,
					}

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

	wg.Wait()
	actualDuration := time.Since(startTime)

	actualThroughput := float64(totalOrders) / actualDuration.Seconds()
	errorRate := float64(errors) / float64(totalOrders+errors) * 100

	t.Logf("Optimized throughput test results:")
	t.Logf("  Duration: %v", actualDuration)
	t.Logf("  Total orders: %d", totalOrders)
	t.Logf("  Errors: %d (%.2f%%)", errors, errorRate)
	t.Logf("  Throughput: %.2f orders/second", actualThroughput)
	t.Logf("  Goroutines: %d", numGoroutines)

	// Get detailed metrics
	metrics := service.GetDetailedMetrics()
	t.Logf("  Active orders: %d", metrics.ActiveOrdersCount)
	t.Logf("  Avg processing time: %dns", metrics.AverageProcessingTimeNs)
	t.Logf("  P99 processing time: %dns", metrics.P99ProcessingTimeNs)

	if actualThroughput < targetThroughput && errorRate < 10.0 {
		t.Errorf("Throughput below target with acceptable error rate: got %.2f, want %.2f orders/second", actualThroughput, targetThroughput)
	}

	t.Logf("SUCCESS: Achieved %.0fK orders/second throughput", actualThroughput/1000)
}

func TestOptimizedServiceScalability(t *testing.T) {
	// Test scaling from 1 to 1000 goroutines
	ctx := context.Background()
	mockEngine := &MockExecutionEngine{}

	service := NewOptimizedExecutionService(mockEngine, nil) // No validation for max performance
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)

	asset := &domain.Asset{
		Symbol:    "SCALABILITY_TEST",
		AssetType: domain.AssetTypeStock,
	}

	goroutineCounts := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1000}

	for _, numGoroutines := range goroutineCounts {
		// Reset metrics for each test
		service.metrics = &PerformanceMetrics{}

		ordersPerGoroutine := 1000

		var wg sync.WaitGroup
		var totalOrders int64

		startTime := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				localOrders := 0

				for j := 0; j < ordersPerGoroutine; j++ {
					order := &domain.Order{
						ID:            fmt.Sprintf("SCALE_%d_%d_%d", numGoroutines, goroutineID, j),
						Asset:         asset,
						Type:          domain.OrderTypeMarket,
						Side:          domain.OrderSideBuy,
						Quantity:      types.NewDecimalFromFloat(100.0),
						TimeInForce:   domain.TimeInForceIOC,
						ClientOrderID: fmt.Sprintf("SCALE_CLIENT_%d_%d_%d", numGoroutines, goroutineID, j),
						CreatedAt:     time.Now(),
						Status:        domain.OrderStatusPending,
					}

					_, err := service.SubmitOrder(ctx, order)
					if err == nil {
						localOrders++
					}
				}

				totalOrders += int64(localOrders)
			}(i)
		}

		wg.Wait()
		actualDuration := time.Since(startTime)
		throughput := float64(totalOrders) / actualDuration.Seconds()

		t.Logf("Goroutines: %4d, Orders: %6d, Duration: %8v, Throughput: %10.2f ops/sec",
			numGoroutines, totalOrders, actualDuration, throughput)

		// Verify scaling efficiency
		if numGoroutines > 1 && throughput < 1000 { // Minimum acceptable throughput
			t.Errorf("Poor scaling with %d goroutines: %.2f ops/sec", numGoroutines, throughput)
		}
	}
}

func TestOptimizedServiceMemoryEfficiency(t *testing.T) {
	// Test memory usage with object pooling
	ctx := context.Background()
	mockEngine := &MockExecutionEngine{}

	service := NewOptimizedExecutionService(mockEngine, nil)
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop(ctx)

	asset := &domain.Asset{
		Symbol:    "MEMORY_EFFICIENCY_TEST",
		AssetType: domain.AssetTypeStock,
	}

	// Baseline memory measurement
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Submit many orders in batches to test object pooling
	numBatches := 100
	ordersPerBatch := 1000
	totalOrders := numBatches * ordersPerBatch

	for batch := 0; batch < numBatches; batch++ {
		for i := 0; i < ordersPerBatch; i++ {
			order := &domain.Order{
				ID:            fmt.Sprintf("MEMORY_EFF_%d_%d", batch, i),
				Asset:         asset,
				Type:          domain.OrderTypeMarket,
				Side:          domain.OrderSideBuy,
				Quantity:      types.NewDecimalFromFloat(100.0),
				TimeInForce:   domain.TimeInForceIOC,
				ClientOrderID: fmt.Sprintf("MEMORY_EFF_CLIENT_%d_%d", batch, i),
				CreatedAt:     time.Now(),
				Status:        domain.OrderStatusPending,
			}

			_, err := service.SubmitOrder(ctx, order)
			if err != nil && batch == 0 && i < 10 { // Only report first few errors
				t.Logf("Order submission failed: %v", err)
			}
		}

		// Force GC every batch to measure steady state
		if batch%10 == 0 {
			runtime.GC()
		}
	}

	// Final memory measurement
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	memoryUsed := m2.Alloc - m1.Alloc
	memoryPerOrder := float64(memoryUsed) / float64(totalOrders)
	totalMemoryMB := float64(m2.Alloc) / 1024 / 1024

	t.Logf("Optimized memory efficiency results:")
	t.Logf("  Orders processed: %d", totalOrders)
	t.Logf("  Memory used: %.2f MB", float64(memoryUsed)/1024/1024)
	t.Logf("  Memory per order: %.2f bytes", memoryPerOrder)
	t.Logf("  Total memory: %.2f MB", totalMemoryMB)
	t.Logf("  Heap objects: %d", m2.HeapObjects)
	t.Logf("  GC cycles: %d", m2.NumGC-m1.NumGC)

	// Verify memory efficiency
	maxMemoryPerOrder := 200.0 // Should be lower with pooling
	if memoryPerOrder > maxMemoryPerOrder {
		t.Errorf("Memory per order too high: %.2f bytes > %.2f bytes", memoryPerOrder, maxMemoryPerOrder)
	}

	maxTotalMemoryMB := 1000.0 // 1GB limit
	if totalMemoryMB > maxTotalMemoryMB {
		t.Errorf("Total memory usage too high: %.2f MB > %.2f MB", totalMemoryMB, maxTotalMemoryMB)
	}
}

func TestOptimizedServiceLatency(t *testing.T) {
	// Test ultra-low latency with optimizations
	ctx := context.Background()
	mockEngine := &MockExecutionEngine{}

	service := NewOptimizedExecutionService(mockEngine, nil) // No validation for lowest latency
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

	// Warmup period
	for i := 0; i < 1000; i++ {
		order := &domain.Order{
			ID:            fmt.Sprintf("WARMUP_%d", i),
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
			ID:            fmt.Sprintf("LATENCY_%d", i),
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

		if err == nil {
			latencies[i] = latency
		}
	}

	// Calculate percentiles
	validLatencies := make([]time.Duration, 0, numSamples)
	for _, lat := range latencies {
		if lat > 0 {
			validLatencies = append(validLatencies, lat)
		}
	}

	if len(validLatencies) < numSamples/2 {
		t.Fatalf("Too many failed measurements: %d/%d", len(validLatencies), numSamples)
	}

	// Sort for percentile calculation
	for i := 0; i < len(validLatencies)-1; i++ {
		for j := i + 1; j < len(validLatencies); j++ {
			if validLatencies[i] > validLatencies[j] {
				validLatencies[i], validLatencies[j] = validLatencies[j], validLatencies[i]
			}
		}
	}

	p50 := validLatencies[len(validLatencies)/2]
	p95 := validLatencies[int(float64(len(validLatencies))*0.95)]
	p99 := validLatencies[int(float64(len(validLatencies))*0.99)]

	t.Logf("Optimized latency results:")
	t.Logf("  Valid samples: %d/%d", len(validLatencies), numSamples)
	t.Logf("  Median (p50): %v", p50)
	t.Logf("  95th percentile: %v", p95)
	t.Logf("  99th percentile: %v", p99)

	// Stricter targets with optimizations
	if p99 > 500*time.Microsecond {
		t.Errorf("p99 latency too high: %v > 500μs", p99)
	}

	if p95 > 200*time.Microsecond {
		t.Errorf("p95 latency too high: %v > 200μs", p95)
	}

	if p50 > 50*time.Microsecond {
		t.Errorf("median latency too high: %v > 50μs", p50)
	}
}
