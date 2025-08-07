package risk

import (
	"math/rand"
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

// BenchmarkCVaRCalculation_ProductionSLA tests the current CVaR implementation against production SLA
// This benchmark is expected to FAIL initially, demonstrating the performance gap
func BenchmarkCVaRCalculation_ProductionSLA(b *testing.B) {
	calculator := NewCVaRCalculator()

	testCases := []struct {
		name     string
		dataSize int
	}{
		{"Small_100", 100},
		{"Medium_1000", 1000},
		{"Large_5000", 5000},
		{"Production_10000", 10000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			returns := generateRealisticMarketReturns(tc.dataSize)
			portfolio := types.NewDecimalFromFloat(1000000.0) // $1M portfolio
			
			// Track individual operation latencies for p99 calculation
			latencies := make([]time.Duration, b.N)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()
				
				_, err := calculator.CalculateHistoricalCVaR(
					returns,
					portfolio,
					types.NewDecimalFromFloat(95.0),
				)
				
				latencies[i] = time.Since(start)
				
				if err != nil {
					b.Fatalf("CVaR calculation failed: %v", err)
				}
			}
			b.StopTimer()
			
			// Calculate p99 latency
			p99Latency := calculateP99Latency(latencies)
			
			// SLA requirement: <1ms p99 (same as VaR)
			slaThreshold := time.Millisecond
			
			b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
			b.ReportMetric(float64(p99Latency.Microseconds()), "p99_latency_μs")
			
			if p99Latency > slaThreshold {
				b.Logf("❌ CVAR PERFORMANCE SLA VIOLATION")
				b.Logf("   Required: p99 < %v", slaThreshold)
				b.Logf("   Actual:   p99 = %v", p99Latency)
				b.Logf("   Violation: %v (%.2fx slower than SLA)",
					p99Latency-slaThreshold,
					float64(p99Latency.Nanoseconds())/float64(slaThreshold.Nanoseconds()))
				
				// This will cause the benchmark to fail when we implement SLA validation
				// For now, we just report the violation
			} else {
				b.Logf("✅ SLA COMPLIANT: p99 = %v (under %v threshold)", p99Latency, slaThreshold)
			}
		})
	}
}

// BenchmarkCVaRCalculation_MemoryUsage benchmarks memory allocations
func BenchmarkCVaRCalculation_MemoryUsage(b *testing.B) {
	calculator := NewCVaRCalculator()

	returns := generateRealisticMarketReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := calculator.CalculateHistoricalCVaR(
			returns,
			portfolio,
			types.NewDecimalFromFloat(95.0),
		)
		if err != nil {
			b.Fatalf("CVaR calculation failed: %v", err)
		}
	}
}

// BenchmarkCVaRCalculation_ConcurrentLoad tests concurrent calculation performance
func BenchmarkCVaRCalculation_ConcurrentLoad(b *testing.B) {
	calculator := NewCVaRCalculator()

	returns := generateRealisticMarketReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := calculator.CalculateHistoricalCVaR(
				returns,
				portfolio,
				types.NewDecimalFromFloat(95.0),
			)
			if err != nil {
				b.Fatalf("CVaR calculation failed: %v", err)
			}
		}
	})
}

// BenchmarkCVaRCalculation_WorstCaseData benchmarks with market crash scenario data
func BenchmarkCVaRCalculation_WorstCaseData(b *testing.B) {
	calculator := NewCVaRCalculator()

	// Generate worst-case scenario data (market crash pattern)
	returns := generateMarketCrashReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)
	
	latencies := make([]time.Duration, b.N)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		
		_, err := calculator.CalculateHistoricalCVaR(
			returns,
			portfolio,
			types.NewDecimalFromFloat(95.0),
		)
		
		latencies[i] = time.Since(start)
		
		if err != nil {
			b.Fatalf("CVaR calculation failed: %v", err)
		}
	}
	b.StopTimer()
	
	p99Latency := calculateP99Latency(latencies)
	b.ReportMetric(float64(p99Latency.Nanoseconds()), "crash_scenario_p99_ns")
	
	if p99Latency > time.Millisecond {
		b.Logf("❌ CRASH SCENARIO SLA VIOLATION: p99 = %v", p99Latency)
	}
}

// Helper functions are reused from var_calculator_benchmark_test.go

// generateMarketCrashReturns simulates a market crash scenario with extreme negative returns
func generateMarketCrashReturns(size int) []types.Decimal {
	returns := make([]types.Decimal, size)
	rng := rand.New(rand.NewSource(54321))
	
	for i := 0; i < size; i++ {
		var dailyReturn float64
		
		// Simulate market crash with high negative correlation
		if i < size/10 { // First 10% - crash period
			dailyReturn = -0.05 + rng.NormFloat64()*0.03 // -5% ± 3%
		} else if i < size/3 { // Next 23% - recovery volatility
			dailyReturn = rng.NormFloat64() * 0.04 // High volatility
		} else { // Remaining - normal market
			dailyReturn = rng.NormFloat64()*0.015 + 0.0001
		}
		
		returns[i] = types.NewDecimalFromFloat(dailyReturn)
	}
	
	return returns
}