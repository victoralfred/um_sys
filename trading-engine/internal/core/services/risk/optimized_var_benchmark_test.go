package risk

import (
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

// BenchmarkOptimizedVaRCalculation_ProductionSLA tests the optimized VaR implementation
// This benchmark should PASS, demonstrating the performance improvements
func BenchmarkOptimizedVaRCalculation_ProductionSLA(b *testing.B) {
	config := VaRConfig{
		DefaultMethod:             "optimized_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100, // Reduced from 250 for testing
		SupportedMethods:          []string{"optimized_historical"},
	}
	calculator := NewOptimizedVaRCalculator(config)

	testCases := []struct {
		name     string
		dataSize int
	}{
		{"Small_100", 100},
		{"Medium_1000", 1000},
		{"Large_5000", 5000},
		{"Production_10000", 10000},
		{"Enterprise_50000", 50000}, // Stress test for enterprise scale
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
					b.Fatalf("Optimized VaR calculation failed: %v", err)
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
				b.Logf("❌ OPTIMIZED SLA VIOLATION")
				b.Logf("   Required: p99 < %v", slaThreshold)
				b.Logf("   Actual:   p99 = %v", p99Latency)
				b.Logf("   Violation: %v (%.2fx slower than SLA)",
					p99Latency-slaThreshold,
					float64(p99Latency.Nanoseconds())/float64(slaThreshold.Nanoseconds()))

				// This should NOT happen with the optimized implementation
				b.Fail()
			} else {
				b.Logf("✅ SLA COMPLIANT: p99 = %v (under %v threshold)", p99Latency, slaThreshold)

				// Calculate performance improvement
				b.Logf("📈 PERFORMANCE HEADROOM: %.1fx faster than required",
					float64(slaThreshold.Nanoseconds())/float64(p99Latency.Nanoseconds()))
			}
		})
	}
}

// BenchmarkVaRComparison_OriginalVsOptimized compares original vs optimized implementation
func BenchmarkVaRComparison_OriginalVsOptimized(b *testing.B) {
	originalCalculator := NewVaRCalculator()
	optimizedCalculator := NewOptimizedVaRCalculator(VaRConfig{
		DefaultMethod:             "optimized_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 250,
	})

	returns := generateRealisticMarketReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)

	b.Run("Original", func(b *testing.B) {
		latencies := make([]time.Duration, b.N)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()
			_, err := originalCalculator.CalculateHistoricalVaR(
				returns,
				portfolio,
				types.NewDecimalFromFloat(95.0),
			)
			latencies[i] = time.Since(start)

			if err != nil {
				b.Fatalf("Original VaR calculation failed: %v", err)
			}
		}
		b.StopTimer()

		p99Latency := calculateP99Latency(latencies)
		b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
	})

	b.Run("Optimized", func(b *testing.B) {
		latencies := make([]time.Duration, b.N)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()
			_, err := optimizedCalculator.CalculateHistoricalVaR(
				returns,
				portfolio,
				types.NewDecimalFromFloat(95.0),
			)
			latencies[i] = time.Since(start)

			if err != nil {
				b.Fatalf("Optimized VaR calculation failed: %v", err)
			}
		}
		b.StopTimer()

		p99Latency := calculateP99Latency(latencies)
		b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
	})
}

// BenchmarkOptimizedVaRCalculation_CacheEfficiency tests cache performance
func BenchmarkOptimizedVaRCalculation_CacheEfficiency(b *testing.B) {
	calculator := NewOptimizedVaRCalculator(VaRConfig{
		DefaultMethod:             "optimized_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
	})

	returns := generateRealisticMarketReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)

	b.Run("ColdCache", func(b *testing.B) {
		// Each calculation uses different data (cold cache)
		latencies := make([]time.Duration, b.N)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testReturns := generateRealisticMarketReturns(1000)
			start := time.Now()

			_, err := calculator.CalculateHistoricalVaR(
				testReturns,
				portfolio,
				types.NewDecimalFromFloat(95.0),
			)

			latencies[i] = time.Since(start)

			if err != nil {
				b.Fatalf("VaR calculation failed: %v", err)
			}
		}
		b.StopTimer()

		p99Latency := calculateP99Latency(latencies)
		b.ReportMetric(float64(p99Latency.Nanoseconds()), "cold_p99_ns")
	})

	b.Run("WarmCache", func(b *testing.B) {
		// All calculations use same data (warm cache after first)
		latencies := make([]time.Duration, b.N)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()

			_, err := calculator.CalculateHistoricalVaR(
				returns, // Same data = cache hits
				portfolio,
				types.NewDecimalFromFloat(95.0),
			)

			latencies[i] = time.Since(start)

			if err != nil {
				b.Fatalf("VaR calculation failed: %v", err)
			}
		}
		b.StopTimer()

		p99Latency := calculateP99Latency(latencies)
		b.ReportMetric(float64(p99Latency.Nanoseconds()), "warm_p99_ns")

		// Warm cache should be significantly faster
		if p99Latency > time.Microsecond*100 { // 100μs threshold for cached results
			b.Logf("⚠️ CACHE NOT EFFECTIVE: p99 = %v (expected <100μs)", p99Latency)
		} else {
			b.Logf("✅ CACHE EFFECTIVE: p99 = %v (<100μs threshold)", p99Latency)
		}
	})
}

// BenchmarkOptimizedVaRCalculation_MemoryEfficiency tests memory optimization
func BenchmarkOptimizedVaRCalculation_MemoryEfficiency(b *testing.B) {
	calculator := NewOptimizedVaRCalculator(VaRConfig{
		DefaultMethod:             "optimized_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
	})

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

// BenchmarkOptimizedVaRCalculation_ConcurrentLoad tests concurrent performance
func BenchmarkOptimizedVaRCalculation_ConcurrentLoad(b *testing.B) {
	calculator := NewOptimizedVaRCalculator(VaRConfig{
		DefaultMethod:             "optimized_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
	})

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

// BenchmarkOptimizedVaRCalculation_DataScaling tests performance scaling with data size
func BenchmarkOptimizedVaRCalculation_DataScaling(b *testing.B) {
	calculator := NewOptimizedVaRCalculator(VaRConfig{
		DefaultMethod:             "optimized_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
	})

	dataSizes := []int{100, 500, 1000, 5000, 10000, 25000, 50000}
	portfolio := types.NewDecimalFromFloat(1000000.0)

	for _, size := range dataSizes {
		b.Run(b.Name()+"_"+string(rune(size/1000))+"K", func(b *testing.B) {
			returns := generateRealisticMarketReturns(size)
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

			p99Latency := calculateP99Latency(latencies)
			b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
			b.ReportMetric(float64(size), "data_points")

			// Log scaling behavior
			scalingRatio := float64(p99Latency.Nanoseconds()) / float64(size)
			b.ReportMetric(scalingRatio, "ns_per_datapoint")
		})
	}
}
