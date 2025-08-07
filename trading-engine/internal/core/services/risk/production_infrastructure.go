package risk

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/trading-engine/pkg/types"
)

// ProductionCache provides thread-safe caching for production calculations
type ProductionCache struct {
	data            map[string]*CacheEntry
	mu              sync.RWMutex
	maxSize         int
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// CacheEntry represents a cached calculation result with metadata
type CacheEntry struct {
	Value        interface{}
	ExpiresAt    time.Time
	HitCount     uint64
	CreatedAt    time.Time
	LastAccessed time.Time
}

// NewProductionCache creates a new production-ready cache
func NewProductionCache(defaultExpiry time.Duration) *ProductionCache {
	cache := &ProductionCache{
		data:            make(map[string]*CacheEntry),
		maxSize:         10000, // Maximum cache entries
		cleanupInterval: time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a value from the cache
func (c *ProductionCache) Get(key string) interface{} {
	c.mu.RLock()
	entry, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return nil
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return nil
	}

	// Update access metadata
	c.mu.Lock()
	entry.HitCount++
	entry.LastAccessed = time.Now()
	c.mu.Unlock()

	return entry.Value
}

// Set stores a value in the cache
func (c *ProductionCache) Set(key string, value interface{}, expiry time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is full
	if len(c.data) >= c.maxSize {
		c.evictLRU()
	}

	c.data[key] = &CacheEntry{
		Value:        value,
		ExpiresAt:    time.Now().Add(expiry),
		HitCount:     0,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
	}
}

// evictLRU removes the least recently used entry
func (c *ProductionCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.data {
		if oldestKey == "" || entry.LastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccessed
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}

// cleanupExpired removes expired entries periodically
func (c *ProductionCache) cleanupExpired() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.data {
				if now.After(entry.ExpiresAt) {
					delete(c.data, key)
				}
			}
			c.mu.Unlock()

		case <-c.stopCleanup:
			return
		}
	}
}

// GetStats returns cache statistics
func (c *ProductionCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalHits := uint64(0)
	for _, entry := range c.data {
		totalHits += entry.HitCount
	}

	return CacheStats{
		Size:      len(c.data),
		MaxSize:   c.maxSize,
		TotalHits: totalHits,
	}
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Size      int    `json:"size"`
	MaxSize   int    `json:"max_size"`
	TotalHits uint64 `json:"total_hits"`
}

// Close stops the cache cleanup goroutine
func (c *ProductionCache) Close() {
	close(c.stopCleanup)
}

// ProductionValidator provides comprehensive input/output validation for production
type ProductionValidator struct {
}

// NewProductionValidator creates a new production validator
func NewProductionValidator() *ProductionValidator {
	return &ProductionValidator{}
}

// ValidateHistoricalReturns validates historical return data for production use
func (v *ProductionValidator) ValidateHistoricalReturns(returns []types.Decimal) error {
	if len(returns) == 0 {
		return NewRiskError(ErrInsufficientData, "Empty returns data", "validation")
	}

	// Check for extreme values that might indicate data corruption
	extremeThreshold := types.NewDecimalFromFloat(1.0) // 100% return threshold

	for i, ret := range returns {
		negThreshold := extremeThreshold.Mul(types.NewDecimalFromFloat(-1.0))
		if ret.Cmp(extremeThreshold) > 0 || ret.Cmp(negThreshold) < 0 {
			return NewRiskError(ErrCorruptedData,
				fmt.Sprintf("Extreme return detected at index %d: %s", i, ret.String()),
				"validation")
		}
	}

	return nil
}

// PerformanceMonitor tracks system performance for production monitoring
type PerformanceMonitor struct {
	mu               sync.RWMutex
	calculationCount uint64
	totalLatency     time.Duration
	latencyHistory   []time.Duration
	errorCount       uint64
	lastResetTime    time.Time
	maxHistorySize   int
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		latencyHistory: make([]time.Duration, 0, 1000),
		lastResetTime:  time.Now(),
		maxHistorySize: 1000,
	}
}

// RecordCalculation records a calculation's performance metrics
func (m *PerformanceMonitor) RecordCalculation(latency time.Duration, dataPoints int, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calculationCount++
	m.totalLatency += latency

	// Maintain latency history for percentile calculations
	m.latencyHistory = append(m.latencyHistory, latency)
	if len(m.latencyHistory) > m.maxHistorySize {
		m.latencyHistory = m.latencyHistory[1:]
	}

	if !success {
		m.errorCount++
	}
}

// GetCurrentMetrics returns current performance metrics
func (m *PerformanceMonitor) GetCurrentMetrics() PerformanceMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	throughput := float64(m.calculationCount) / time.Since(m.lastResetTime).Seconds()

	return PerformanceMetrics{
		CPUUtilization:    getCurrentCPUUsage(),
		MemoryUtilization: int64(memStats.Alloc),
		GoroutineCount:    runtime.NumGoroutine(),
		GCPauseTime:       time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256]),
		AllocationsCount:  memStats.Mallocs,
		ThroughputQPS:     throughput,
		ConcurrencyLevel:  runtime.NumGoroutine(),
		CacheHitRatio:     0.0, // Would be calculated from cache stats
	}
}

// getCurrentCPUUsage provides a simple CPU usage estimate
func getCurrentCPUUsage() float64 {
	// Simplified CPU usage calculation
	// In production, this would use more sophisticated system monitoring
	return float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 0.1
}

// Reset resets all performance counters
func (m *PerformanceMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calculationCount = 0
	m.totalLatency = 0
	m.latencyHistory = m.latencyHistory[:0]
	m.errorCount = 0
	m.lastResetTime = time.Now()
}

// GetPerformanceReport generates a comprehensive performance report
func (m *PerformanceMonitor) GetPerformanceReport() PerformanceReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := PerformanceReport{
		TotalCalculations: m.calculationCount,
		ErrorCount:        m.errorCount,
		ErrorRate:         float64(m.errorCount) / float64(m.calculationCount),
		ReportPeriod:      time.Since(m.lastResetTime),
	}

	if len(m.latencyHistory) > 0 {
		report.LatencyStats = calculateLatencyStats(m.latencyHistory)
	}

	return report
}

// PerformanceReport contains comprehensive performance statistics
type PerformanceReport struct {
	TotalCalculations uint64            `json:"total_calculations"`
	ErrorCount        uint64            `json:"error_count"`
	ErrorRate         float64           `json:"error_rate"`
	ReportPeriod      time.Duration     `json:"report_period"`
	LatencyStats      LatencyStatistics `json:"latency_stats"`
}

// LatencyStatistics contains detailed latency analysis
type LatencyStatistics struct {
	Mean   time.Duration `json:"mean"`
	Median time.Duration `json:"median"`
	P95    time.Duration `json:"p95"`
	P99    time.Duration `json:"p99"`
	Min    time.Duration `json:"min"`
	Max    time.Duration `json:"max"`
}

// calculateLatencyStats computes latency percentiles and statistics
func calculateLatencyStats(latencies []time.Duration) LatencyStatistics {
	if len(latencies) == 0 {
		return LatencyStatistics{}
	}

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	stats := LatencyStatistics{
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Median: sorted[len(sorted)/2],
	}

	// Calculate P95 and P99
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	stats.P95 = sorted[p95Index]
	stats.P99 = sorted[p99Index]

	// Calculate mean
	total := time.Duration(0)
	for _, latency := range latencies {
		total += latency
	}
	stats.Mean = total / time.Duration(len(latencies))

	return stats
}
