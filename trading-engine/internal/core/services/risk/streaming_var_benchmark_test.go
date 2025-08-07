package risk

import (
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

// BenchmarkStreamingVaRCalculation_ProductionSLA tests the streaming VaR implementation
// This benchmark should PASS for all dataset sizes, demonstrating O(n) performance
func BenchmarkStreamingVaRCalculation_ProductionSLA(b *testing.B) {
	config := VaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
		SupportedMethods:          []string{"streaming_historical"},
	}
	calculator := NewStreamingVaRCalculator(config)

	testCases := []struct {
		name     string
		dataSize int
	}{
		{"Small_100", 100},
		{"Medium_1000", 1000},
		{"Large_5000", 5000},
		{"Production_10000", 10000},
		{"Enterprise_50000", 50000},
		{"Massive_100000", 100000}, // Stress test for massive datasets
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
					b.Fatalf("Streaming VaR calculation failed: %v", err)
				}
			}
			b.StopTimer()

			// Calculate p99 latency
			p99Latency := calculateP99Latency(latencies)

			// SLA requirement: <1ms p99
			slaThreshold := time.Millisecond

			b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
			b.ReportMetric(float64(p99Latency.Microseconds()), "p99_latency_μs")
			b.ReportMetric(float64(tc.dataSize), "data_points")

			// Calculate throughput metrics
			avgLatency := time.Duration(calculateAverageLatency(latencies))
			pointsPerSecond := float64(tc.dataSize) / avgLatency.Seconds()
			b.ReportMetric(pointsPerSecond, "points_per_second")

			if p99Latency > slaThreshold {
				b.Logf("❌ STREAMING SLA VIOLATION")
				b.Logf("   Required: p99 < %v", slaThreshold)
				b.Logf("   Actual:   p99 = %v", p99Latency)
				b.Logf("   Violation: %v (%.2fx slower than SLA)",
					p99Latency-slaThreshold,
					float64(p99Latency.Nanoseconds())/float64(slaThreshold.Nanoseconds()))

				// Streaming should achieve SLA for all dataset sizes
				b.Fail()
			} else {
				b.Logf("✅ SLA COMPLIANT: p99 = %v (under %v threshold)", p99Latency, slaThreshold)

				// Calculate performance metrics
				headroom := float64(slaThreshold.Nanoseconds()) / float64(p99Latency.Nanoseconds())
				b.Logf("📈 PERFORMANCE HEADROOM: %.1fx faster than required", headroom)
				b.Logf("🚀 THROUGHPUT: %.0f data points/second", pointsPerSecond)

				// Log scaling behavior for analysis
				nsPerPoint := float64(p99Latency.Nanoseconds()) / float64(tc.dataSize)
				b.Logf("⚡ SCALING: %.2f ns/point", nsPerPoint)
			}
		})
	}
}

// BenchmarkAllVaRImplementations_Comparison compares all three implementations
func BenchmarkAllVaRImplementations_Comparison(b *testing.B) {
	dataSize := 1000
	returns := generateRealisticMarketReturns(dataSize)
	portfolio := types.NewDecimalFromFloat(1000000.0)

	b.Run("Original", func(b *testing.B) {
		calculator := NewVaRCalculator()
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
				b.Fatalf("Original VaR calculation failed: %v", err)
			}
		}
		b.StopTimer()

		p99Latency := calculateP99Latency(latencies)
		b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
		b.ReportMetric(float64(p99Latency.Microseconds()), "p99_latency_μs")

		slaCompliant := p99Latency <= time.Millisecond
		if slaCompliant {
			b.Logf("✅ Original: SLA compliant (p99=%v)", p99Latency)
		} else {
			b.Logf("❌ Original: SLA violation (p99=%v)", p99Latency)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		config := VaRConfig{
			DefaultMethod:             "optimized_historical",
			DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
			MinHistoricalObservations: 250,
		}
		calculator := NewOptimizedVaRCalculator(config)
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

		p99Latency := calculateP99Latency(latencies)
		b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
		b.ReportMetric(float64(p99Latency.Microseconds()), "p99_latency_μs")

		slaCompliant := p99Latency <= time.Millisecond
		if slaCompliant {
			b.Logf("✅ Optimized: SLA compliant (p99=%v)", p99Latency)
		} else {
			b.Logf("❌ Optimized: SLA violation (p99=%v)", p99Latency)
		}
	})

	b.Run("Streaming", func(b *testing.B) {
		config := VaRConfig{
			DefaultMethod:             "streaming_historical",
			DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
			MinHistoricalObservations: 100,
		}
		calculator := NewStreamingVaRCalculator(config)
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
				b.Fatalf("Streaming VaR calculation failed: %v", err)
			}
		}
		b.StopTimer()

		p99Latency := calculateP99Latency(latencies)
		b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
		b.ReportMetric(float64(p99Latency.Microseconds()), "p99_latency_μs")

		slaCompliant := p99Latency <= time.Millisecond
		if slaCompliant {
			b.Logf("✅ Streaming: SLA compliant (p99=%v)", p99Latency)
		} else {
			b.Logf("❌ Streaming: SLA violation (p99=%v)", p99Latency)
		}
	})
}

// BenchmarkStreamingVaRCalculation_ScalingAnalysis tests O(n) scaling behavior
func BenchmarkStreamingVaRCalculation_ScalingAnalysis(b *testing.B) {
	config := VaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
	}
	calculator := NewStreamingVaRCalculator(config)
	portfolio := types.NewDecimalFromFloat(1000000.0)

	dataSizes := []int{100, 500, 1000, 5000, 10000, 50000, 100000}

	for _, size := range dataSizes {
		b.Run(benchNameFromSize(size), func(b *testing.B) {
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
			avgLatency := time.Duration(calculateAverageLatency(latencies))

			b.ReportMetric(float64(p99Latency.Nanoseconds()), "p99_latency_ns")
			b.ReportMetric(float64(size), "data_points")

			// Calculate scaling metrics
			nsPerPoint := float64(p99Latency.Nanoseconds()) / float64(size)
			throughput := float64(size) / avgLatency.Seconds()

			b.ReportMetric(nsPerPoint, "ns_per_datapoint")
			b.ReportMetric(throughput, "points_per_second")

			// Verify O(n) scaling (should be roughly constant ns/point)
			if nsPerPoint > 100.0 { // 100ns per point threshold
				b.Logf("⚠️ SCALING CONCERN: %.2f ns/point (>100ns threshold)", nsPerPoint)
			} else {
				b.Logf("✅ GOOD SCALING: %.2f ns/point", nsPerPoint)
			}
		})
	}
}

// Helper functions
func calculateAverageLatency(latencies []time.Duration) int64 {
	if len(latencies) == 0 {
		return 0
	}

	sum := int64(0)
	for _, latency := range latencies {
		sum += latency.Nanoseconds()
	}
	return sum / int64(len(latencies))
}

func benchNameFromSize(size int) string {
	if size < 1000 {
		return "Size_" + string(rune('0'+size/100)) + "H" // Hundreds
	} else if size < 10000 {
		return "Size_" + string(rune('0'+size/1000)) + "K" // Thousands
	} else {
		return "Size_" + string(rune('0'+size/10000)) + "0K" // Ten-thousands
	}
}
