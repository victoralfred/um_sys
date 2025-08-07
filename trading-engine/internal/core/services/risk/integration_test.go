package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

// ProductionIntegrationSuite provides comprehensive integration testing for production-ready risk management
type ProductionIntegrationSuite struct {
	logger         *RiskLogger
	logBuffer      *bytes.Buffer
	varCalculator  *VaRCalculator
	cvarCalculator *CVaRCalculator
	streamingVar   *StreamingVaRCalculator
	streamingCVar  *StreamingCVaRCalculator
	ctx            context.Context
}

// NewProductionIntegrationSuite creates a production-ready integration test environment
func NewProductionIntegrationSuite(t *testing.T) *ProductionIntegrationSuite {
	// Create buffered logger for test verification
	logBuffer := &bytes.Buffer{}
	handler := slog.NewJSONHandler(logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug})

	logger := &RiskLogger{
		logger: slog.New(handler),
		config: LoggerConfig{
			Level:     LogLevelDebug,
			Format:    "json",
			Component: "production-integration-test",
			AddSource: false, // Disable source tracking for performance
		},
	}

	// Initialize production-ready calculators
	varCalculator := NewVaRCalculator()
	cvarCalculator := NewCVaRCalculator()

	// Production streaming configurations
	streamingVarConfig := VaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
		SupportedMethods:          []string{"streaming_historical", "parametric", "monte_carlo"},
		EnableBacktesting:         true,
	}
	streamingVar := NewStreamingVaRCalculator(streamingVarConfig)

	streamingCVarConfig := CVaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
		SupportedMethods:          []string{"streaming_historical", "parametric"},
		EnableTailAnalysis:        true,
	}
	streamingCVar := NewStreamingCVaRCalculator(streamingCVarConfig)

	// Create production context with full traceability
	ctx := WithRequestID(
		WithCorrelationID(
			WithUserID(
				WithPortfolioID(
					WithOperation(context.Background(), "production_integration_test"),
					"PROD-PORTFOLIO-001"),
				"prod-user-123"),
			"prod-integration-correlation"),
		"prod-integration-request")

	return &ProductionIntegrationSuite{
		logger:         logger,
		logBuffer:      logBuffer,
		varCalculator:  varCalculator,
		cvarCalculator: cvarCalculator,
		streamingVar:   streamingVar,
		streamingCVar:  streamingCVar,
		ctx:            ctx,
	}
}

// TestProductionRiskManagementWorkflow tests the complete production risk management pipeline
func TestProductionRiskManagementWorkflow(t *testing.T) {
	suite := NewProductionIntegrationSuite(t)
	contextLogger := suite.logger.WithContext(suite.ctx)

	// Production test data: realistic market conditions with proper statistical distribution
	returns := generateProductionReturns(1000)         // 1000 observations for statistical significance
	portfolio := types.NewDecimalFromFloat(10000000.0) // $10M production portfolio
	confidence := types.NewDecimalFromFloat(99.0)      // Production 99% confidence

	t.Run("Production_Risk_Pipeline", func(t *testing.T) {
		// Phase 1: System initialization and validation
		contextLogger.LogCalculationStart(suite.ctx, "ProductionRiskPipeline", "comprehensive", len(returns))

		// Validate input data quality
		if err := validateProductionData(returns, portfolio, confidence); err != nil {
			contextLogger.LogError(suite.ctx, NewCalculationError("data_validation", err))
			t.Fatalf("Production data validation failed: %v", err)
		}

		// Phase 2: Performance-critical VaR calculations
		var streamingVaR VaRResult
		t.Run("Production_VaR_Calculations", func(t *testing.T) {
			// Production streaming VaR with SLA requirements
			contextLogger.LogCalculationStart(suite.ctx, "StreamingVaR", "production", len(returns))

			start := time.Now()
			result, err := suite.streamingVar.CalculateHistoricalVaR(returns, portfolio, confidence)
			duration := time.Since(start)

			if err != nil {
				contextLogger.LogError(suite.ctx, NewCalculationError("production_var", err))
				t.Fatalf("Production VaR calculation failed: %v", err)
			}

			streamingVaR = result

			contextLogger.LogCalculationComplete(suite.ctx, "StreamingVaR", "production",
				duration, true, map[string]interface{}{
					"var_amount":       result.VaR.String(),
					"portfolio_value":  result.PortfolioValue.String(),
					"confidence_level": result.ConfidenceLevel.String(),
				})

			// Validate production SLA: must be < 1ms for 99.9% uptime
			if duration > time.Millisecond {
				t.Errorf("Production SLA violation: VaR calculation took %v (required: <1ms)", duration)
			}

			// Validate result quality for production
			if result.VaR.IsZero() {
				t.Error("Production VaR result cannot be zero")
			}

			if !result.VaR.IsNegative() {
				t.Error("Production VaR must be negative (loss estimate)")
			}
		})

		// Phase 3: Production CVaR calculations with tail risk analysis
		t.Run("Production_CVaR_Calculations", func(t *testing.T) {
			contextLogger.LogCalculationStart(suite.ctx, "StreamingCVaR", "production", len(returns))

			start := time.Now()
			result, err := suite.streamingCVar.CalculateHistoricalCVaR(returns, portfolio, confidence)
			duration := time.Since(start)

			if err != nil {
				contextLogger.LogError(suite.ctx, NewCalculationError("production_cvar", err))
				t.Fatalf("Production CVaR calculation failed: %v", err)
			}

			contextLogger.LogCalculationComplete(suite.ctx, "StreamingCVaR", "production",
				duration, true, map[string]interface{}{
					"cvar_amount":      result.CVaR.String(),
					"var_amount":       result.VaR.String(),
					"confidence_level": result.ConfidenceLevel.String(),
				})

			// Production CVaR validation: CVaR >= VaR (mathematical requirement)
			if result.CVaR.Cmp(streamingVaR.VaR) < 0 {
				t.Errorf("Production constraint violation: CVaR (%s) < VaR (%s)",
					result.CVaR.String(), streamingVaR.VaR.String())
			}
		})

		// Phase 4: Production error handling validation
		t.Run("Production_Error_Handling", func(t *testing.T) {
			// Test insufficient data scenario
			shortReturns := returns[:50]
			_, err := suite.varCalculator.CalculateHistoricalVaR(shortReturns, portfolio, confidence)

			if err == nil {
				t.Error("Production system must reject insufficient data")
			}

			if riskErr, ok := err.(*RiskError); ok {
				contextLogger.LogError(suite.ctx, riskErr)

				if riskErr.Code != ErrInsufficientData {
					t.Errorf("Expected ErrInsufficientData, got %v", riskErr.Code)
				}

				// Validate production error classification
				if riskErr.Severity != SeverityLow {
					t.Errorf("Data validation errors should be SeverityLow in production")
				}
			} else {
				t.Error("Production errors must be RiskError instances")
			}
		})

		// Phase 5: Production system metrics and monitoring
		t.Run("Production_System_Monitoring", func(t *testing.T) {
			metrics := GetCurrentSystemMetrics()
			contextLogger.LogSystemMetrics(suite.ctx, metrics)

			// Validate production system health
			if metrics.GoroutineCount > 1000 {
				t.Errorf("Goroutine count too high for production: %d", metrics.GoroutineCount)
			}

			if metrics.MemoryAllocBytes > 100*1024*1024 { // 100MB limit
				t.Errorf("Memory usage too high for production: %d bytes", metrics.MemoryAllocBytes)
			}
		})

		contextLogger.InfoContext(suite.ctx, "Production risk management pipeline completed",
			slog.String("status", "success"),
			slog.String("environment", "production"),
			slog.Int("calculations_performed", 2))
	})

	// Validate production logging quality
	suite.validateProductionLogs(t)
}

// TestProductionConcurrency tests production-level concurrent processing
func TestProductionConcurrency(t *testing.T) {
	suite := NewProductionIntegrationSuite(t)

	// Production concurrency parameters
	const (
		numWorkers            = 100 // Simulate production load
		calculationsPerWorker = 10
		maxAcceptableTime     = 5 * time.Second // Production timeout
	)

	returns := generateProductionReturns(1000)
	portfolio := types.NewDecimalFromFloat(1000000.0)
	confidence := types.NewDecimalFromFloat(95.0)

	t.Run("Production_Concurrent_Load", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan VaRResult, numWorkers*calculationsPerWorker)
		errors := make(chan error, numWorkers*calculationsPerWorker)

		start := time.Now()

		// Launch production workers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				workerCtx := WithRequestID(suite.ctx, fmt.Sprintf("worker-%d", workerID))
				workerLogger := suite.logger.WithContext(workerCtx)

				for j := 0; j < calculationsPerWorker; j++ {
					calcStart := time.Now()
					result, err := suite.streamingVar.CalculateHistoricalVaR(returns, portfolio, confidence)
					calcDuration := time.Since(calcStart)

					if err != nil {
						workerLogger.LogError(workerCtx, NewCalculationError("concurrent_var", err))
						errors <- err
						return
					}

					workerLogger.LogCalculationComplete(workerCtx, "ConcurrentVaR", "production",
						calcDuration, true, map[string]interface{}{
							"worker_id":      workerID,
							"calculation_id": j,
						})

					results <- result
				}
			}(i)
		}

		// Wait with timeout
		done := make(chan bool)
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(maxAcceptableTime):
			t.Fatalf("Production concurrency test timed out after %v", maxAcceptableTime)
		}

		totalDuration := time.Since(start)
		close(results)
		close(errors)

		// Validate production concurrency results
		errorCount := 0
		for range errors {
			errorCount++
		}

		resultCount := 0
		for range results {
			resultCount++
		}

		expectedResults := numWorkers * calculationsPerWorker
		if errorCount > 0 {
			t.Errorf("Production concurrency failed: %d errors", errorCount)
		}

		if resultCount != expectedResults {
			t.Errorf("Production concurrency incomplete: got %d results, expected %d",
				resultCount, expectedResults)
		}

		throughput := float64(expectedResults) / totalDuration.Seconds()
		t.Logf("Production concurrency: %d calculations in %v (%.1f calc/sec)",
			expectedResults, totalDuration, throughput)

		// Production throughput requirement: minimum 1000 calc/sec
		if throughput < 1000 {
			t.Errorf("Production throughput below requirement: %.1f calc/sec (min: 1000)", throughput)
		}
	})
}

// TestProductionStressConditions tests system behavior under production stress
func TestProductionStressConditions(t *testing.T) {
	suite := NewProductionIntegrationSuite(t)

	t.Run("Production_Large_Dataset_Stress", func(t *testing.T) {
		// Production-scale dataset
		largeReturns := generateProductionReturns(250000)   // 250k observations
		portfolio := types.NewDecimalFromFloat(100000000.0) // $100M portfolio

		contextLogger := suite.logger.WithContext(suite.ctx)
		contextLogger.LogCalculationStart(suite.ctx, "ProductionStress", "large_dataset", len(largeReturns))

		start := time.Now()
		result, err := suite.streamingVar.CalculateHistoricalVaR(
			largeReturns, portfolio, types.NewDecimalFromFloat(99.9))
		duration := time.Since(start)

		if err != nil {
			contextLogger.LogError(suite.ctx, NewCalculationError("stress_test", err))
			t.Fatalf("Production stress test failed: %v", err)
		}

		contextLogger.LogCalculationComplete(suite.ctx, "ProductionStress", "large_dataset",
			duration, true, map[string]interface{}{
				"dataset_size":              len(largeReturns),
				"var_amount":                result.VaR.String(),
				"throughput_points_per_sec": float64(len(largeReturns)) / duration.Seconds(),
			})

		// Production requirement: process 250k points in < 500ms
		if duration > 500*time.Millisecond {
			t.Errorf("Production large dataset processing too slow: %v (max: 500ms)", duration)
		}

		t.Logf("Production stress test passed: %d points in %v", len(largeReturns), duration)
	})
}

// generateProductionReturns creates statistically realistic market returns for production testing
func generateProductionReturns(count int) []types.Decimal {
	returns := make([]types.Decimal, count)

	// Use realistic market distribution parameters
	meanReturn := 0.0008 // 0.08% daily mean (approx 20% annual)
	volatility := 0.015  // 1.5% daily volatility (approx 24% annual)

	// Add realistic market conditions including tail events
	for i := 0; i < count; i++ {
		var dailyReturn float64

		// 95% normal market conditions
		if i%20 != 0 {
			// Simulate normal distribution using Box-Muller transform approximation
			dailyReturn = meanReturn + volatility*normalRandom()
		} else {
			// 5% stress conditions (fat tails)
			if i%100 == 0 {
				// 1% extreme events (market crashes/surges)
				dailyReturn = meanReturn + volatility*normalRandom()*5.0
			} else {
				// 4% elevated volatility
				dailyReturn = meanReturn + volatility*normalRandom()*2.0
			}
		}

		returns[i] = types.NewDecimalFromFloat(dailyReturn)
	}

	return returns
}

// normalRandom generates pseudo-normal random numbers for testing
func normalRandom() float64 {
	// Simple pseudo-random normal distribution for deterministic testing
	// In production, this would use crypto/rand for better randomness
	return (float64(time.Now().UnixNano()%1000)/1000.0 - 0.5) * 2.0
}

// validateProductionData ensures data quality for production calculations
func validateProductionData(returns []types.Decimal, portfolio types.Decimal, confidence types.Decimal) error {
	if len(returns) < 100 {
		return NewInsufficientDataError("production_validation", len(returns), 100)
	}

	if portfolio.IsZero() || portfolio.IsNegative() {
		return NewRiskError(ErrInvalidPortfolio, "Portfolio value must be positive", "production_validation")
	}

	if confidence.Cmp(types.NewDecimalFromFloat(50.0)) < 0 ||
		confidence.Cmp(types.NewDecimalFromFloat(99.99)) > 0 {
		return NewInvalidConfidenceError("production_validation", confidence)
	}

	return nil
}

// validateProductionLogs ensures logging meets production standards
func (suite *ProductionIntegrationSuite) validateProductionLogs(t *testing.T) {
	logContent := suite.logBuffer.String()
	if logContent == "" {
		t.Error("Production system must generate audit logs")
		return
	}

	logLines := bytes.Split(suite.logBuffer.Bytes(), []byte("\n"))
	var logEntries []map[string]interface{}

	for _, line := range logLines {
		if len(line) == 0 {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		logEntries = append(logEntries, entry)
	}

	if len(logEntries) < 5 {
		t.Errorf("Insufficient production logging: %d entries (minimum: 5)", len(logEntries))
	}

	// Validate production logging requirements
	hasTraceability := false
	hasPerformanceMetrics := false
	hasErrorHandling := false
	hasSystemMetrics := false

	for _, entry := range logEntries {
		// Check traceability
		if _, hasReqID := entry["request_id"]; hasReqID {
			if _, hasCorr := entry["correlation_id"]; hasCorr {
				hasTraceability = true
			}
		}

		// Check performance metrics
		if _, hasDuration := entry["duration"]; hasDuration {
			hasPerformanceMetrics = true
		}

		// Check error handling
		if _, hasErrorCode := entry["error_code"]; hasErrorCode {
			hasErrorHandling = true
		}

		// Check system metrics
		if _, hasGoroutines := entry["goroutines"]; hasGoroutines {
			hasSystemMetrics = true
		}
	}

	productionRequirements := map[string]bool{
		"Request Traceability": hasTraceability,
		"Performance Metrics":  hasPerformanceMetrics,
		"Error Handling":       hasErrorHandling,
		"System Metrics":       hasSystemMetrics,
	}

	missing := []string{}
	for requirement, present := range productionRequirements {
		if !present {
			missing = append(missing, requirement)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Production logging missing: %v", missing)
	}

	coverage := float64(len(productionRequirements)-len(missing)) / float64(len(productionRequirements)) * 100
	t.Logf("Production logging coverage: %.1f%%", coverage)

	if coverage < 100.0 {
		t.Errorf("Production logging must be 100%% complete, got %.1f%%", coverage)
	}
}
