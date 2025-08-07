package risk

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

// BenchmarkVaRCalculation_ProductionSLA tests the current VaR implementation against production SLA
// This benchmark is expected to FAIL initially, demonstrating the performance gap
func BenchmarkVaRCalculation_ProductionSLA(b *testing.B) {
	calculator := NewVaRCalculator()

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

				_, err := calculator.CalculateHistoricalVaR(
					returns,
					portfolio,
					types.NewDecimalFromFloat(95.0),
				)

				latencies[i] = time.Since(start)

				if err != nil {
					b.Fatalf("VaR calculation failed: %v", err)
				}
			}
			b.StopTimer()

			// Calculate p99 latency
			p99Latency := calculateP99Latency(latencies)

			// SLA requirement: <1ms p99
			slaThreshold := time.Millisecond

			b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
			b.ReportMetric(float64(p99Latency.Microseconds()), "p99_latency_μs")

			if p99Latency > slaThreshold {
				b.Logf("❌ PERFORMANCE SLA VIOLATION")
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

// BenchmarkVaRCalculation_MemoryUsage benchmarks memory allocations
func BenchmarkVaRCalculation_MemoryUsage(b *testing.B) {
	calculator := NewVaRCalculator()

	returns := generateRealisticMarketReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := calculator.CalculateHistoricalVaR(
			returns,
			portfolio,
			types.NewDecimalFromFloat(95.0),
		)
		if err != nil {
			b.Fatalf("VaR calculation failed: %v", err)
		}
	}
}

// BenchmarkVaRCalculation_ConcurrentLoad tests concurrent calculation performance
func BenchmarkVaRCalculation_ConcurrentLoad(b *testing.B) {
	calculator := NewVaRCalculator()

	returns := generateRealisticMarketReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := calculator.CalculateHistoricalVaR(
				returns,
				portfolio,
				types.NewDecimalFromFloat(95.0),
			)
			if err != nil {
				b.Fatalf("VaR calculation failed: %v", err)
			}
		}
	})
}

// generateRealisticMarketReturns creates synthetic market return data with realistic characteristics
func generateRealisticMarketReturns(size int) []types.Decimal {
	returns := make([]types.Decimal, size)

	// Use seeded random for reproducible benchmarks
	rng := rand.New(rand.NewSource(12345))

	// Generate returns with realistic market characteristics:
	// - Mean close to zero (0.01% daily return)
	// - Standard deviation ~1.5% (realistic daily volatility)
	// - Some fat-tail events (occasional large moves)

	for i := 0; i < size; i++ {
		var dailyReturn float64

		// 95% normal market conditions
		if rng.Float64() < 0.95 {
			dailyReturn = rng.NormFloat64()*0.015 + 0.0001 // 1.5% vol, 0.01% drift
		} else {
			// 5% fat-tail events (market stress)
			sign := 1.0
			if rng.Float64() < 0.6 { // 60% chance of negative tail event
				sign = -1.0
			}
			dailyReturn = sign * (0.03 + rng.Float64()*0.05) // 3-8% moves
		}

		returns[i] = types.NewDecimalFromFloat(dailyReturn)
	}

	return returns
}

// calculateP99Latency calculates the 99th percentile latency from a slice of durations
func calculateP99Latency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Sort latencies to find percentiles
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate p99 index (99th percentile)
	p99Index := int(float64(len(sorted)) * 0.99)
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	return sorted[p99Index]
}
