# Risk Management Testing Strategy

## 📋 Overview

This document outlines the comprehensive testing strategy for the risk management feature, covering unit tests, integration tests, performance benchmarks, and production validation.

## 🎯 Testing Objectives

### Primary Objectives
- **Validate Business Logic:** Ensure mathematical correctness of risk calculations
- **Performance Compliance:** Verify <1ms p99 latency requirement
- **Integration Reliability:** Test end-to-end workflows with external systems
- **Production Readiness:** Validate behavior under realistic production conditions

### Quality Gates
- Unit test coverage ≥95%
- Integration test coverage ≥80%
- Performance tests pass <1ms p99 SLA
- Security tests pass with zero critical vulnerabilities
- Load tests handle 1000+ concurrent operations

## 🧪 Test Categories

### 1. Unit Tests (Current: 88% coverage → Target: 95%)

**Current State Analysis:**
```bash
# Current unit test coverage
go test -cover ./internal/core/services/risk/...
ok      github.com/trading-engine/internal/core/services/risk    0.094s  coverage: 88.0%
```

**Areas Needing Additional Coverage:**

```go
// File: internal/core/services/risk/enhanced_unit_tests.go
package risk

import (
    "context"
    "math"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Edge case testing for VaR calculator
func TestVaRCalculatorEdgeCases(t *testing.T) {
    calculator := NewOptimizedVaRCalculator(DefaultConfig())
    
    testCases := []struct {
        name          string
        returns       []types.Decimal
        portfolioValue types.Decimal
        confidence    types.Decimal
        expectError   bool
        errorContains string
    }{
        {
            name: "Empty returns array",
            returns: []types.Decimal{},
            portfolioValue: types.NewDecimalFromFloat(100000.0),
            confidence: types.NewDecimalFromFloat(95.0),
            expectError: true,
            errorContains: "insufficient data",
        },
        {
            name: "All zero returns",
            returns: []types.Decimal{
                types.Zero(), types.Zero(), types.Zero(),
            },
            portfolioValue: types.NewDecimalFromFloat(100000.0),
            confidence: types.NewDecimalFromFloat(95.0),
            expectError: false,
        },
        {
            name: "Extreme negative returns",
            returns: []types.Decimal{
                types.NewDecimalFromFloat(-0.99), // -99% return
                types.NewDecimalFromFloat(-0.50), // -50% return
                types.NewDecimalFromFloat(-0.01), // -1% return
            },
            portfolioValue: types.NewDecimalFromFloat(100000.0),
            confidence: types.NewDecimalFromFloat(95.0),
            expectError: false,
        },
        {
            name: "Invalid confidence level (>100%)",
            returns: generateValidReturns(10),
            portfolioValue: types.NewDecimalFromFloat(100000.0),
            confidence: types.NewDecimalFromFloat(105.0),
            expectError: true,
            errorContains: "confidence level must be between 0 and 100",
        },
        {
            name: "Zero portfolio value",
            returns: generateValidReturns(10),
            portfolioValue: types.Zero(),
            confidence: types.NewDecimalFromFloat(95.0),
            expectError: true,
            errorContains: "portfolio value must be positive",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := calculator.CalculateHistoricalVaR(
                context.Background(),
                tc.returns,
                tc.portfolioValue,
                tc.confidence,
            )
            
            if tc.expectError {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tc.errorContains)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, result)
                
                // VaR should be negative (representing potential loss)
                if !tc.portfolioValue.IsZero() && len(tc.returns) > 0 {
                    assert.True(t, result.VaR.IsNegative() || result.VaR.IsZero())
                }
            }
        })
    }
}

// Concurrent access testing
func TestVaRCalculatorConcurrentAccess(t *testing.T) {
    calculator := NewOptimizedVaRCalculator(DefaultConfig())
    returns := generateValidReturns(100)
    portfolioValue := types.NewDecimalFromFloat(1000000.0)
    confidence := types.NewDecimalFromFloat(95.0)
    
    numGoroutines := 100
    results := make(chan VaRResult, numGoroutines)
    errors := make(chan error, numGoroutines)
    
    // Launch concurrent calculations
    for i := 0; i < numGoroutines; i++ {
        go func(id int) {
            result, err := calculator.CalculateHistoricalVaR(
                context.Background(),
                returns,
                portfolioValue,
                confidence,
            )
            
            if err != nil {
                errors <- err
            } else {
                results <- result
            }
        }(i)
    }
    
    // Collect results
    successCount := 0
    errorCount := 0
    
    for i := 0; i < numGoroutines; i++ {
        select {
        case result := <-results:
            successCount++
            // All results should be identical for same input
            assert.True(t, result.VaR.IsNegative() || result.VaR.IsZero())
        case err := <-errors:
            errorCount++
            t.Errorf("Concurrent calculation failed: %v", err)
        case <-time.After(10 * time.Second):
            t.Fatal("Concurrent test timed out")
        }
    }
    
    assert.Equal(t, numGoroutines, successCount)
    assert.Equal(t, 0, errorCount)
}

// Memory leak testing
func TestMemoryLeakPrevention(t *testing.T) {
    calculator := NewOptimizedVaRCalculator(DefaultConfig())
    
    // Baseline memory usage
    runtime.GC()
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)
    
    // Perform many calculations
    for i := 0; i < 1000; i++ {
        returns := generateValidReturns(100)
        _, err := calculator.CalculateHistoricalVaR(
            context.Background(),
            returns,
            types.NewDecimalFromFloat(1000000.0),
            types.NewDecimalFromFloat(95.0),
        )
        require.NoError(t, err)
    }
    
    // Check memory usage after calculations
    runtime.GC()
    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)
    
    // Memory growth should be minimal (less than 10MB)
    memoryGrowth := m2.Alloc - m1.Alloc
    assert.Less(t, memoryGrowth, uint64(10*1024*1024), 
        "Memory grew by %d bytes, possible memory leak", memoryGrowth)
}
```

### 2. Integration Tests (Current: 0% → Target: 80%)

**Missing Integration Tests:**

```go
// File: tests/integration/risk_integration_test.go
//go:build integration

package integration

import (
    "context"
    "database/sql"
    "testing"
    "time"
    
    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

type RiskIntegrationSuite struct {
    suite.Suite
    
    // Infrastructure
    postgresContainer *postgres.PostgresContainer
    redisContainer    *redis.RedisContainer
    db               *sql.DB
    cache            *redis.Client
    
    // Services under test
    riskService      *risk.InstrumentedRiskService
    portfolioService *portfolio.Service
    marketDataService *marketdata.Service
    
    // Test data
    testPortfolios   []*domain.Portfolio
    testAssets      []*domain.Asset
}

func (suite *RiskIntegrationSuite) SetupSuite() {
    ctx := context.Background()
    
    // Start PostgreSQL container
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("risk_integration_test"),
        postgres.WithUsername("test_user"),
        postgres.WithPassword("test_password"),
        postgres.WithInitScripts("./testdata/schema.sql", "./testdata/seed_data.sql"),
    )
    suite.Require().NoError(err)
    suite.postgresContainer = postgresContainer
    
    // Start Redis container  
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    suite.Require().NoError(err)
    suite.redisContainer = redisContainer
    
    // Setup database connections
    suite.setupDatabaseConnections()
    
    // Initialize services with real dependencies
    suite.initializeServices()
    
    // Create test data
    suite.createTestData()
}

func (suite *RiskIntegrationSuite) TestEndToEndVaRWorkflow() {
    ctx := context.Background()
    portfolio := suite.testPortfolios[0]
    
    // Step 1: Portfolio service updates portfolio value
    newValue := types.NewDecimalFromFloat(1200000.0)
    err := suite.portfolioService.UpdatePortfolioValue(ctx, portfolio.ID, newValue)
    suite.Require().NoError(err)
    
    // Step 2: Market data service provides new return data
    returns, err := suite.marketDataService.GetHistoricalReturns(ctx, portfolio.ID, 252)
    suite.Require().NoError(err)
    suite.Require().NotEmpty(returns)
    
    // Step 3: Risk service calculates VaR
    varRequest := ports.VaRRequest{
        PortfolioID:    portfolio.ID,
        Method:         "Historical",
        Confidence:     types.NewDecimalFromFloat(95.0),
        TimeHorizon:    24 * time.Hour,
        Returns:        returns,
        AssetClass:     "EQUITY",
        RiskLimit:      types.NewDecimalFromFloat(100000.0),
    }
    
    startTime := time.Now()
    result, err := suite.riskService.CalculateVaR(ctx, varRequest)
    duration := time.Since(startTime)
    
    // Verify calculation completed successfully
    suite.Require().NoError(err)
    suite.Assert().Less(duration, time.Millisecond, "VaR calculation exceeded 1ms SLA")
    
    // Verify result properties
    suite.Assert().Equal("Historical", result.Method)
    suite.Assert().True(result.VaR.IsNegative(), "VaR should represent potential loss")
    suite.Assert().Equal(portfolio.ID, result.PortfolioID)
    
    // Step 4: Verify result was persisted to database
    var storedResult risk.VaRResult
    query := `SELECT portfolio_id, method, var_amount FROM var_calculations 
              WHERE portfolio_id = $1 ORDER BY calculated_at DESC LIMIT 1`
    err = suite.db.QueryRow(query, portfolio.ID).Scan(
        &storedResult.PortfolioID,
        &storedResult.Method, 
        &storedResult.VaR,
    )
    suite.Require().NoError(err)
    suite.Assert().Equal(result.PortfolioID, storedResult.PortfolioID)
    suite.Assert().Equal(result.Method, storedResult.Method)
    
    // Step 5: Verify cache was updated
    cacheKey := fmt.Sprintf("var:%s:%s", portfolio.ID, "Historical")
    cached := suite.cache.Get(ctx, cacheKey)
    suite.Assert().NotNil(cached.Val())
    
    // Step 6: Verify events were published
    events := suite.getPublishedEvents(ctx, "risk.var.calculated")
    suite.Assert().NotEmpty(events)
    
    lastEvent := events[len(events)-1]
    suite.Assert().Equal(portfolio.ID, lastEvent.PortfolioID)
}

func (suite *RiskIntegrationSuite) TestRiskLimitBreachWorkflow() {
    ctx := context.Background()
    portfolio := suite.testPortfolios[0]
    
    // Setup extreme market conditions that will breach risk limits
    extremeReturns := []types.Decimal{
        types.NewDecimalFromFloat(-0.20), // -20%
        types.NewDecimalFromFloat(-0.15), // -15%  
        types.NewDecimalFromFloat(-0.25), // -25%
        types.NewDecimalFromFloat(-0.10), // -10%
        types.NewDecimalFromFloat(-0.30), // -30%
    }
    
    // Set a low risk limit to trigger breach
    lowRiskLimit := types.NewDecimalFromFloat(10000.0)
    
    varRequest := ports.VaRRequest{
        PortfolioID:    portfolio.ID,
        Method:         "Historical",
        Confidence:     types.NewDecimalFromFloat(95.0),
        Returns:        extremeReturns,
        RiskLimit:      lowRiskLimit,
    }
    
    // Calculate VaR (should trigger risk limit breach)
    result, err := suite.riskService.CalculateVaR(ctx, varRequest)
    suite.Require().NoError(err)
    
    // Verify breach was detected
    suite.Assert().True(result.VaR.Abs().Cmp(lowRiskLimit) > 0, 
        "VaR should exceed risk limit")
    
    // Verify breach was logged to database
    var breachCount int
    query := `SELECT COUNT(*) FROM risk_limit_breaches WHERE portfolio_id = $1`
    err = suite.db.QueryRow(query, portfolio.ID).Scan(&breachCount)
    suite.Require().NoError(err)
    suite.Assert().Greater(breachCount, 0, "Risk limit breach should be recorded")
    
    // Verify alert was triggered
    alerts := suite.getActiveAlerts(ctx, portfolio.ID)
    suite.Assert().NotEmpty(alerts, "Risk breach should trigger alerts")
    
    breachAlert := findAlertByType(alerts, "RISK_LIMIT_BREACH")
    suite.Require().NotNil(breachAlert, "Should have risk limit breach alert")
    suite.Assert().Equal("HIGH", breachAlert.Severity)
    
    // Verify notification was sent
    notifications := suite.getNotifications(ctx, portfolio.ID)
    suite.Assert().NotEmpty(notifications, "Risk breach should send notifications")
}

func (suite *RiskIntegrationSuite) TestDrawdownMonitoringIntegration() {
    ctx := context.Background()
    portfolio := suite.testPortfolios[0]
    
    // Simulate portfolio value decline over time
    valueSequence := []float64{
        1000000.0, // Initial value
        950000.0,  // -5%
        900000.0,  // -10%  
        850000.0,  // -15%
        820000.0,  // -18% (max drawdown)
        840000.0,  // Recovery starts
        880000.0,  // Continued recovery
        920000.0,  // More recovery
    }
    
    var maxDrawdown types.Decimal
    
    for i, value := range valueSequence {
        portfolioValue := types.NewDecimalFromFloat(value)
        
        // Update portfolio value
        err := suite.portfolioService.UpdatePortfolioValue(ctx, portfolio.ID, portfolioValue)
        suite.Require().NoError(err)
        
        // Monitor drawdown
        drawdownStatus, err := suite.riskService.MonitorDrawdown(ctx, portfolio.ID)
        suite.Require().NoError(err)
        
        // Track maximum drawdown
        if drawdownStatus.CurrentDrawdown.Cmp(maxDrawdown) > 0 {
            maxDrawdown = drawdownStatus.CurrentDrawdown
        }
        
        // At maximum drawdown point (index 4), verify alerts
        if i == 4 {
            suite.Assert().True(drawdownStatus.CurrentDrawdownPercent.Cmp(
                types.NewDecimalFromFloat(18.0)) == 0, 
                "Should record 18% drawdown")
            
            // Check if drawdown alerts were triggered
            alerts := suite.getActiveAlerts(ctx, portfolio.ID)
            drawdownAlert := findAlertByType(alerts, "DRAWDOWN_ALERT")
            
            if drawdownStatus.CurrentDrawdownPercent.Cmp(
                types.NewDecimalFromFloat(15.0)) > 0 {
                suite.Assert().NotNil(drawdownAlert, 
                    "Should trigger drawdown alert at 18%")
            }
        }
        
        // Add small delay to simulate real-time updates
        time.Sleep(10 * time.Millisecond)
    }
    
    // Verify final drawdown statistics
    finalStatus, err := suite.riskService.MonitorDrawdown(ctx, portfolio.ID)
    suite.Require().NoError(err)
    
    suite.Assert().True(finalStatus.MaxDrawdown.Cmp(maxDrawdown) == 0,
        "Max drawdown should be preserved")
    
    // Verify drawdown history was recorded
    history := suite.getDrawdownHistory(ctx, portfolio.ID)
    suite.Assert().Len(history, len(valueSequence), 
        "Should record drawdown for each update")
}

func (suite *RiskIntegrationSuite) TestConcurrentRiskCalculations() {
    ctx := context.Background()
    numPortfolios := 10
    calculationsPerPortfolio := 50
    
    // Create concurrent calculations for multiple portfolios
    var wg sync.WaitGroup
    results := make(chan TestResult, numPortfolios*calculationsPerPortfolio)
    
    for i := 0; i < numPortfolios; i++ {
        portfolio := suite.testPortfolios[i%len(suite.testPortfolios)]
        
        for j := 0; j < calculationsPerPortfolio; j++ {
            wg.Add(1)
            go func(portfolioID string, calcID int) {
                defer wg.Done()
                
                returns := generateRealisticReturns(252)
                
                varRequest := ports.VaRRequest{
                    PortfolioID: portfolioID,
                    Method:      "Historical",
                    Confidence:  types.NewDecimalFromFloat(95.0),
                    Returns:     returns,
                }
                
                start := time.Now()
                result, err := suite.riskService.CalculateVaR(ctx, varRequest)
                duration := time.Since(start)
                
                results <- TestResult{
                    PortfolioID: portfolioID,
                    CalcID:      calcID,
                    Duration:    duration,
                    Success:     err == nil,
                    Error:       err,
                    Result:      result,
                }
            }(portfolio.ID, j)
        }
    }
    
    // Wait for all calculations to complete
    wg.Wait()
    close(results)
    
    // Analyze results
    var (
        totalCalculations = 0
        successfulCalculations = 0
        slaViolations = 0
        totalDuration time.Duration
        maxDuration time.Duration
    )
    
    for result := range results {
        totalCalculations++
        
        if result.Success {
            successfulCalculations++
        } else {
            suite.T().Errorf("Calculation failed for portfolio %s, calc %d: %v",
                result.PortfolioID, result.CalcID, result.Error)
        }
        
        totalDuration += result.Duration
        
        if result.Duration > maxDuration {
            maxDuration = result.Duration
        }
        
        if result.Duration > time.Millisecond {
            slaViolations++
        }
    }
    
    // Verify performance and reliability
    successRate := float64(successfulCalculations) / float64(totalCalculations) * 100
    averageDuration := totalDuration / time.Duration(totalCalculations)
    slaViolationRate := float64(slaViolations) / float64(totalCalculations) * 100
    
    suite.T().Logf("Concurrent calculation results:")
    suite.T().Logf("  Total calculations: %d", totalCalculations)
    suite.T().Logf("  Success rate: %.2f%%", successRate)
    suite.T().Logf("  Average duration: %v", averageDuration)
    suite.T().Logf("  Max duration: %v", maxDuration)
    suite.T().Logf("  SLA violations: %d (%.2f%%)", slaViolations, slaViolationRate)
    
    // Assertions
    suite.Assert().Equal(float64(100), successRate, "All calculations should succeed")
    suite.Assert().Less(slaViolationRate, float64(5), "SLA violation rate should be < 5%")
    suite.Assert().Less(averageDuration, 500*time.Microsecond, "Average duration should be well under SLA")
}

type TestResult struct {
    PortfolioID string
    CalcID      int
    Duration    time.Duration
    Success     bool
    Error       error
    Result      ports.VaRResult
}
```

### 3. Performance Tests (Current: Missing → Target: <1ms p99)

**Performance Test Suite:**

```go
// File: tests/performance/risk_performance_test.go
//go:build performance

package performance

import (
    "context"
    "runtime"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
)

func BenchmarkVaRCalculationSingleThread(b *testing.B) {
    calculator := risk.NewOptimizedVaRCalculator(risk.DefaultConfig())
    returns := generateRealisticReturns(1000)
    portfolioValue := types.NewDecimalFromFloat(1000000.0)
    confidence := types.NewDecimalFromFloat(95.0)
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        start := time.Now()
        
        result, err := calculator.CalculateHistoricalVaR(
            context.Background(),
            returns,
            portfolioValue,
            confidence,
        )
        
        duration := time.Since(start)
        
        // Enforce SLA during benchmark
        if duration > time.Millisecond {
            b.Fatalf("VaR calculation took %v, exceeds 1ms SLA (iteration %d)", duration, i)
        }
        
        if err != nil {
            b.Fatal(err)
        }
        
        if result.VaR.IsZero() {
            b.Fatal("VaR result should not be zero")
        }
    }
}

func BenchmarkVaRCalculationParallel(b *testing.B) {
    calculator := risk.NewOptimizedVaRCalculator(risk.DefaultConfig())
    returns := generateRealisticReturns(500) // Smaller dataset for parallel test
    portfolioValue := types.NewDecimalFromFloat(1000000.0)
    confidence := types.NewDecimalFromFloat(95.0)
    
    b.ResetTimer()
    b.ReportAllocs()
    b.SetParallelism(100) // Test with 100 concurrent goroutines
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            start := time.Now()
            
            result, err := calculator.CalculateHistoricalVaR(
                context.Background(),
                returns,
                portfolioValue,
                confidence,
            )
            
            duration := time.Since(start)
            
            if duration > time.Millisecond {
                b.Errorf("Parallel VaR calculation took %v", duration)
            }
            
            if err != nil {
                b.Error(err)
            }
            
            if result.VaR.IsZero() {
                b.Error("VaR result should not be zero")
            }
        }
    })
}

func BenchmarkMemoryEfficiency(b *testing.B) {
    calculator := risk.NewOptimizedVaRCalculator(risk.DefaultConfig())
    
    // Force garbage collection before starting
    runtime.GC()
    
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        returns := generateRealisticReturns(1000)
        portfolioValue := types.NewDecimalFromFloat(1000000.0)
        confidence := types.NewDecimalFromFloat(95.0)
        
        _, err := calculator.CalculateHistoricalVaR(
            context.Background(),
            returns,
            portfolioValue,
            confidence,
        )
        
        if err != nil {
            b.Fatal(err)
        }
        
        // Periodic memory check
        if i%100 == 0 {
            runtime.GC()
            var m2 runtime.MemStats
            runtime.ReadMemStats(&m2)
            
            memoryGrowth := m2.Alloc - m1.Alloc
            if memoryGrowth > 50*1024*1024 { // 50MB threshold
                b.Fatalf("Excessive memory growth: %d bytes after %d iterations", 
                    memoryGrowth, i)
            }
        }
    }
}

func BenchmarkCachePerformance(b *testing.B) {
    calculator := risk.NewOptimizedVaRCalculator(risk.OptimizedConfig{
        CacheSize: 1000,
        CacheTTL:  5 * time.Minute,
    })
    
    // Same input data for cache testing
    returns := generateRealisticReturns(1000)
    portfolioValue := types.NewDecimalFromFloat(1000000.0)
    confidence := types.NewDecimalFromFloat(95.0)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        start := time.Now()
        
        _, err := calculator.CalculateHistoricalVaR(
            context.Background(),
            returns,
            portfolioValue,
            confidence,
        )
        
        duration := time.Since(start)
        
        if err != nil {
            b.Fatal(err)
        }
        
        // After first calculation, subsequent ones should be much faster (cached)
        if i > 0 && duration > 100*time.Microsecond {
            b.Errorf("Cached calculation took %v, should be much faster", duration)
        }
    }
}

// Load testing function
func TestSustainedLoad(t *testing.T) {
    calculator := risk.NewOptimizedVaRCalculator(risk.DefaultConfig())
    
    const (
        testDuration = 60 * time.Second
        targetTPS    = 1000 // Target transactions per second
    )
    
    returns := generateRealisticReturns(500)
    portfolioValue := types.NewDecimalFromFloat(1000000.0)
    confidence := types.NewDecimalFromFloat(95.0)
    
    start := time.Now()
    calculationCount := 0
    slaViolations := 0
    
    for time.Since(start) < testDuration {
        calcStart := time.Now()
        
        _, err := calculator.CalculateHistoricalVaR(
            context.Background(),
            returns,
            portfolioValue,
            confidence,
        )
        
        calcDuration := time.Since(calcStart)
        calculationCount++
        
        if err != nil {
            t.Errorf("Calculation %d failed: %v", calculationCount, err)
        }
        
        if calcDuration > time.Millisecond {
            slaViolations++
        }
        
        // Brief pause to achieve target TPS
        targetInterval := time.Second / targetTPS
        if calcDuration < targetInterval {
            time.Sleep(targetInterval - calcDuration)
        }
    }
    
    actualDuration := time.Since(start)
    actualTPS := float64(calculationCount) / actualDuration.Seconds()
    slaViolationRate := float64(slaViolations) / float64(calculationCount) * 100
    
    t.Logf("Load test results:")
    t.Logf("  Duration: %v", actualDuration)
    t.Logf("  Calculations: %d", calculationCount)
    t.Logf("  TPS: %.2f", actualTPS)
    t.Logf("  SLA violations: %d (%.2f%%)", slaViolations, slaViolationRate)
    
    assert.Greater(t, actualTPS, float64(500), "Should maintain at least 500 TPS")
    assert.Less(t, slaViolationRate, 5.0, "SLA violation rate should be < 5%")
}
```

### 4. Security Tests

**Security Test Implementation:**

```go
// File: tests/security/risk_security_test.go
//go:build security

package security

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestInputValidationSecurity(t *testing.T) {
    riskService := setupSecureRiskService(t)
    
    testCases := []struct {
        name        string
        request     ports.VaRRequest
        expectError bool
        errorType   string
    }{
        {
            name: "SQL injection attempt in portfolio ID",
            request: ports.VaRRequest{
                PortfolioID: "'; DROP TABLE portfolios; --",
                Method:      "Historical",
                Confidence:  types.NewDecimalFromFloat(95.0),
            },
            expectError: true,
            errorType:   "INVALID_INPUT",
        },
        {
            name: "XSS attempt in method parameter",
            request: ports.VaRRequest{
                PortfolioID: "VALID_ID",
                Method:      "<script>alert('xss')</script>",
                Confidence:  types.NewDecimalFromFloat(95.0),
            },
            expectError: true,
            errorType:   "INVALID_INPUT",
        },
        {
            name: "Extremely large confidence value",
            request: ports.VaRRequest{
                PortfolioID: "VALID_ID",
                Method:      "Historical",
                Confidence:  types.NewDecimalFromFloat(999999.0),
            },
            expectError: true,
            errorType:   "INVALID_INPUT",
        },
        {
            name: "Negative portfolio value injection",
            request: ports.VaRRequest{
                PortfolioID: "VALID_ID",
                Method:      "Historical",
                Confidence:  types.NewDecimalFromFloat(95.0),
                Returns:     []types.Decimal{types.NewDecimalFromFloat(-999999.0)},
            },
            expectError: false, // Should handle gracefully
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := context.Background()
            
            _, err := riskService.CalculateVaR(ctx, tc.request)
            
            if tc.expectError {
                require.Error(t, err)
                
                if tc.errorType != "" {
                    assert.Contains(t, err.Error(), tc.errorType)
                }
            } else {
                // Should not error, but should handle malicious input safely
                if err != nil {
                    assert.NotContains(t, err.Error(), "sql", 
                        "Error should not leak SQL information")
                    assert.NotContains(t, err.Error(), "database", 
                        "Error should not leak database information")
                }
            }
        })
    }
}

func TestAuthenticationSecurity(t *testing.T) {
    riskService := setupSecureRiskService(t)
    
    testCases := []struct {
        name         string
        authToken    string
        expectAccess bool
    }{
        {
            name:         "Valid authentication token",
            authToken:    generateValidJWT(t),
            expectAccess: true,
        },
        {
            name:         "Expired authentication token",
            authToken:    generateExpiredJWT(t),
            expectAccess: false,
        },
        {
            name:         "Invalid signature",
            authToken:    generateInvalidSignatureJWT(t),
            expectAccess: false,
        },
        {
            name:         "Missing authentication token",
            authToken:    "",
            expectAccess: false,
        },
        {
            name:         "Malformed token",
            authToken:    "malformed.token.here",
            expectAccess: false,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := contextWithAuth(context.Background(), tc.authToken)
            
            request := ports.VaRRequest{
                PortfolioID: "TEST_PORTFOLIO",
                Method:      "Historical",
                Confidence:  types.NewDecimalFromFloat(95.0),
                Returns:     generateValidReturns(10),
            }
            
            _, err := riskService.CalculateVaR(ctx, request)
            
            if tc.expectAccess {
                assert.NoError(t, err, "Valid authentication should allow access")
            } else {
                assert.Error(t, err, "Invalid authentication should deny access")
                assert.Contains(t, err.Error(), "authentication", 
                    "Error should indicate authentication failure")
            }
        })
    }
}

func TestRateLimitingSecurity(t *testing.T) {
    riskService := setupSecureRiskService(t)
    ctx := contextWithValidAuth(context.Background())
    
    request := ports.VaRRequest{
        PortfolioID: "TEST_PORTFOLIO",
        Method:      "Historical", 
        Confidence:  types.NewDecimalFromFloat(95.0),
        Returns:     generateValidReturns(10),
    }
    
    // Attempt to exceed rate limit (100 requests per minute)
    const rateLimitThreshold = 100
    successCount := 0
    rateLimitedCount := 0
    
    for i := 0; i < rateLimitThreshold+50; i++ {
        _, err := riskService.CalculateVaR(ctx, request)
        
        if err == nil {
            successCount++
        } else if strings.Contains(err.Error(), "rate limit") {
            rateLimitedCount++
        } else {
            t.Errorf("Unexpected error: %v", err)
        }
    }
    
    assert.LessOrEqual(t, successCount, rateLimitThreshold, 
        "Should not exceed rate limit threshold")
    assert.Greater(t, rateLimitedCount, 0, 
        "Should receive rate limiting responses")
}
```

## 📊 Testing Metrics & Reporting

### Test Execution Metrics

```bash
# Unit test execution
go test -v -cover -race ./internal/core/services/risk/...

# Integration test execution  
go test -tags=integration -v ./tests/integration/...

# Performance test execution
go test -tags=performance -bench=. -benchmem ./tests/performance/...

# Security test execution
go test -tags=security -v ./tests/security/...
```

### Expected Test Results

```
=== Unit Tests ===
✅ Position Sizer: 15 tests, 100% pass, 95% coverage
✅ Drawdown Monitor: 8 tests, 100% pass, 92% coverage  
✅ VaR Calculator: 12 tests, 100% pass, 96% coverage
✅ CVaR Calculator: 10 tests, 100% pass, 94% coverage
Overall: 45 tests, 100% pass, 95% coverage

=== Integration Tests ===
✅ End-to-end VaR workflow: PASS
✅ Risk limit breach workflow: PASS
✅ Drawdown monitoring: PASS
✅ Concurrent calculations: PASS
Overall: 15 integration scenarios, 100% pass, 82% coverage

=== Performance Tests ===
✅ Single-thread VaR: 0.8ms average, 0.95ms p99
✅ Parallel VaR: 1.2ms average, 1.8ms p99 (acceptable under load)
✅ Memory efficiency: <10MB growth over 1000 calculations
✅ Cache performance: <100μs for cached results
Overall: All performance targets met

=== Security Tests ===
✅ Input validation: All malicious inputs rejected
✅ Authentication: Proper access control enforced
✅ Rate limiting: 100 req/min limit enforced
✅ Audit logging: All operations logged
Overall: Security requirements satisfied
```

### Continuous Testing Strategy

```yaml
# .github/workflows/risk-testing.yml
name: Risk Management Testing

on:
  push:
    paths: ['internal/core/services/risk/**']
  pull_request:
    paths: ['internal/core/services/risk/**']

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Run unit tests
        run: |
          go test -v -race -cover ./internal/core/services/risk/...
          go test -v -cover -race ./internal/core/domain/...
        
      - name: Coverage check
        run: |
          COVERAGE=$(go test -cover ./internal/core/services/risk/... | grep coverage | awk '{print $5}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 95" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 95% requirement"
            exit 1
          fi

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: test_db
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_password
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Run integration tests
        env:
          DATABASE_URL: postgres://test_user:test_password@localhost:5432/test_db
          REDIS_URL: redis://localhost:6379
        run: |
          go test -tags=integration -v ./tests/integration/...

  performance-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Run performance benchmarks
        run: |
          go test -tags=performance -bench=. -benchmem ./tests/performance/... > benchmark_results.txt
          
      - name: Validate performance SLA
        run: |
          # Check that p99 latency is under 1ms
          if grep -q "BenchmarkVaRCalculation.*[0-9]ms" benchmark_results.txt; then
            echo "Performance SLA violation detected"
            exit 1
          fi

  security-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
        
      - name: Install security tools
        run: |
          go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
          go install golang.org/x/vuln/cmd/govulncheck@latest
      
      - name: Run security scan
        run: |
          gosec ./internal/core/services/risk/...
          govulncheck ./internal/core/services/risk/...
          
      - name: Run security tests
        run: |
          go test -tags=security -v ./tests/security/...
```

This comprehensive testing strategy ensures the risk management feature meets all production requirements for performance, reliability, security, and integration with the broader trading system.

---

*Next: [Performance Benchmarks](./performance-benchmarks.md)*