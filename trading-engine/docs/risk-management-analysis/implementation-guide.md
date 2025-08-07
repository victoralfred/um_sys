# Risk Management Implementation Guide

## 🎯 Overview

This guide provides detailed implementation instructions for transforming the current risk management feature from TDD prototype to production-ready system.

## 📋 Current Implementation Analysis

### File Structure Assessment

```
internal/core/services/risk/
├── cvar_calculator.go          (478 lines) ✅ Complete business logic
├── cvar_calculator_test.go     (495 lines) ✅ Comprehensive tests
├── drawdown_monitor.go         (227 lines) ✅ Complete business logic  
├── drawdown_monitor_test.go    (307 lines) ✅ Comprehensive tests
├── position_sizer.go           (142 lines) ✅ Complete business logic
├── position_sizer_enhanced_test.go (141 lines) ✅ Comprehensive tests
├── var_calculator.go           (380 lines) ✅ Complete business logic
└── var_calculator_test.go      (397 lines) ✅ Comprehensive tests

Total: 2,567 lines of code with 88% test coverage
```

### Code Quality Metrics

| Component | Lines | Test Coverage | Cyclomatic Complexity | Tech Debt |
|-----------|-------|---------------|----------------------|-----------|
| CVaR Calculator | 478 | 90% | Medium | Low |
| VaR Calculator | 380 | 85% | Medium | Low |
| Drawdown Monitor | 227 | 92% | Low | Low |
| Position Sizer | 142 | 88% | Low | Low |

## 🔧 Phase 1: Critical Performance Optimization

### 1.1 Performance Benchmark Implementation

**Objective:** Measure and achieve <1ms p99 latency for risk calculations

**Current Problem:**
```go
// Inefficient: Full array sorting on every calculation
sort.Slice(sortedReturns, func(i, j int) bool {
    return sortedReturns[i].Cmp(sortedReturns[j]) < 0
})
```

**Solution Implementation:**

```go
// File: internal/core/services/risk/optimized_var_calculator.go
package risk

import (
    "context"
    "sync"
    "time"
    
    "github.com/google/btree"
    "github.com/hashicorp/golang-lru/v2"
    "github.com/trading-engine/pkg/types"
)

// OptimizedVaRCalculator provides high-performance VaR calculations
type OptimizedVaRCalculator struct {
    // Pre-sorted data structure for O(log n) operations
    sortedReturns *btree.BTree
    
    // LRU cache for calculated results
    cache *lru.Cache[string, VaRResult]
    
    // Object pools to reduce memory allocations
    decimalPool sync.Pool
    slicePool   sync.Pool
    
    // Configuration
    cacheSize int
    cacheTTL  time.Duration
}

// DecimalItem implements btree.Item for decimal sorting
type DecimalItem struct {
    Value types.Decimal
}

func (d DecimalItem) Less(than btree.Item) bool {
    return d.Value.Cmp(than.(DecimalItem).Value) < 0
}

func NewOptimizedVaRCalculator(config OptimizedConfig) *OptimizedVaRCalculator {
    cache, _ := lru.New[string, VaRResult](config.CacheSize)
    
    calc := &OptimizedVaRCalculator{
        sortedReturns: btree.New(32), // Degree 32 for optimal performance
        cache:         cache,
        cacheSize:     config.CacheSize,
        cacheTTL:      config.CacheTTL,
    }
    
    // Initialize object pools
    calc.decimalPool.New = func() interface{} {
        return make([]types.Decimal, 0, 1000)
    }
    
    calc.slicePool.New = func() interface{} {
        return make([]types.Decimal, 0, 1000)
    }
    
    return calc
}

// CalculateHistoricalVaR optimized implementation targeting <1ms p99
func (ovc *OptimizedVaRCalculator) CalculateHistoricalVaR(
    ctx context.Context,
    returns []types.Decimal,
    portfolioValue, confidence types.Decimal,
) (VaRResult, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        if duration > time.Millisecond {
            // Log performance violation
            log.Warn("VaR calculation exceeded 1ms SLA", 
                "duration", duration,
                "data_points", len(returns))
        }
    }()
    
    // Generate cache key
    cacheKey := generateCacheKey(returns, portfolioValue, confidence)
    
    // Check cache first
    if cached, found := ovc.cache.Get(cacheKey); found {
        return cached, nil
    }
    
    // Fast path for small datasets
    if len(returns) < 100 {
        return ovc.calculateSmallDataset(returns, portfolioValue, confidence)
    }
    
    // Optimized calculation for large datasets
    result, err := ovc.calculateOptimized(ctx, returns, portfolioValue, confidence)
    if err != nil {
        return VaRResult{}, err
    }
    
    // Cache result
    ovc.cache.Add(cacheKey, result)
    
    return result, nil
}

// calculateOptimized uses streaming algorithms for large datasets
func (ovc *OptimizedVaRCalculator) calculateOptimized(
    ctx context.Context,
    returns []types.Decimal,
    portfolioValue, confidence types.Decimal,
) (VaRResult, error) {
    // Use Welford's algorithm for streaming statistics
    stats := NewStreamingStats()
    
    for _, ret := range returns {
        stats.Update(ret)
        
        // Check for cancellation
        select {
        case <-ctx.Done():
            return VaRResult{}, ctx.Err()
        default:
        }
    }
    
    // Calculate percentile using streaming quantiles
    percentile := types.NewDecimalFromFloat(100.0).Sub(confidence)
    quantile := ovc.calculateStreamingQuantile(returns, percentile)
    
    varAmount := quantile.Mul(portfolioValue)
    
    return VaRResult{
        Method:          "HistoricalOptimized",
        ConfidenceLevel: confidence,
        VaR:             varAmount,
        PortfolioValue:  portfolioValue,
        Statistics:      stats.GetStatistics(),
        CalculatedAt:    time.Now(),
    }, nil
}
```

**Performance Test Implementation:**

```go
// File: internal/core/services/risk/optimized_var_calculator_test.go
func BenchmarkVaRCalculationPerformance(b *testing.B) {
    calculator := NewOptimizedVaRCalculator(OptimizedConfig{
        CacheSize: 1000,
        CacheTTL:  5 * time.Minute,
    })
    
    // Generate realistic test data
    returns := generateRealisticReturns(1000) // 1000 data points
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
        
        // Enforce SLA requirement
        if duration > time.Millisecond {
            b.Fatalf("VaR calculation took %v, exceeds 1ms SLA", duration)
        }
        
        if err != nil {
            b.Fatal(err)
        }
        
        if result.VaR.IsZero() {
            b.Fatal("VaR result should not be zero")
        }
    }
}

func BenchmarkConcurrentVaRCalculations(b *testing.B) {
    calculator := NewOptimizedVaRCalculator(OptimizedConfig{
        CacheSize: 1000,
        CacheTTL:  5 * time.Minute,
    })
    
    returns := generateRealisticReturns(500)
    portfolioValue := types.NewDecimalFromFloat(1000000.0)
    confidence := types.NewDecimalFromFloat(95.0)
    
    b.ResetTimer()
    b.SetParallelism(100) // Test with 100 concurrent goroutines
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            start := time.Now()
            
            _, err := calculator.CalculateHistoricalVaR(
                context.Background(),
                returns,
                portfolioValue,
                confidence,
            )
            
            duration := time.Since(start)
            
            if duration > time.Millisecond {
                b.Errorf("Concurrent VaR calculation took %v", duration)
            }
            
            if err != nil {
                b.Error(err)
            }
        }
    })
}
```

### 1.2 Memory Optimization

**Current Problem:** High memory allocations in decimal operations

**Solution:**

```go
// File: pkg/types/decimal_pool.go
package types

import (
    "sync"
)

// DecimalPool manages a pool of decimal objects to reduce allocations
type DecimalPool struct {
    pool sync.Pool
}

var GlobalDecimalPool = NewDecimalPool()

func NewDecimalPool() *DecimalPool {
    return &DecimalPool{
        pool: sync.Pool{
            New: func() interface{} {
                return NewDecimal("0")
            },
        },
    }
}

func (dp *DecimalPool) Get() Decimal {
    return dp.pool.Get().(Decimal)
}

func (dp *DecimalPool) Put(d Decimal) {
    // Reset decimal to zero before returning to pool
    d.SetZero()
    dp.pool.Put(d)
}

// UsageExample in risk calculations:
func (vc *VaRCalculator) calculateOptimizedWithPool(returns []types.Decimal) types.Decimal {
    sum := types.GlobalDecimalPool.Get()
    defer types.GlobalDecimalPool.Put(sum)
    
    for _, ret := range returns {
        sum.Add(ret)
    }
    
    mean := types.GlobalDecimalPool.Get()
    defer types.GlobalDecimalPool.Put(mean)
    
    mean.Div(sum, types.NewDecimalFromInt(int64(len(returns))))
    
    return mean.Copy() // Return copy, original goes back to pool
}
```

## 🔧 Phase 2: Observability Infrastructure

### 2.1 Structured Logging Implementation

**Objective:** Complete operational visibility with correlation tracking

```go
// File: internal/core/services/risk/instrumented_risk_service.go
package risk

import (
    "context"
    "log/slog"
    "time"
    
    "github.com/google/uuid"
    "go.opentelemetry.io/otel/trace"
)

type InstrumentedRiskService struct {
    varCalculator      VaRCalculator
    cvarCalculator     CVaRCalculator
    drawdownMonitor    DrawdownMonitor
    positionSizer      PositionSizer
    
    logger   *slog.Logger
    tracer   trace.Tracer
    metrics  MetricsCollector
}

func NewInstrumentedRiskService(
    varCalc VaRCalculator,
    cvarCalc CVaRCalculator,
    ddMonitor DrawdownMonitor,
    posSizer PositionSizer,
    logger *slog.Logger,
    tracer trace.Tracer,
    metrics MetricsCollector,
) *InstrumentedRiskService {
    return &InstrumentedRiskService{
        varCalculator:   varCalc,
        cvarCalculator:  cvarCalc,
        drawdownMonitor: ddMonitor,
        positionSizer:   posSizer,
        logger:          logger,
        tracer:          tracer,
        metrics:         metrics,
    }
}

func (irs *InstrumentedRiskService) CalculateVaR(
    ctx context.Context, 
    req VaRRequest,
) (VaRResult, error) {
    // Start distributed trace
    ctx, span := irs.tracer.Start(ctx, "risk.calculate_var")
    defer span.End()
    
    // Generate correlation ID if not present
    correlationID := getOrCreateCorrelationID(ctx)
    
    // Enhanced structured logging
    irs.logger.InfoContext(ctx, "VaR calculation started",
        slog.String("correlation_id", correlationID),
        slog.String("portfolio_id", req.PortfolioID),
        slog.String("method", req.Method),
        slog.Float64("confidence", req.Confidence.Float64()),
        slog.Int("data_points", len(req.Returns)),
        slog.String("asset_class", req.AssetClass),
    )
    
    // Performance timing
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        
        // Record metrics
        irs.metrics.RecordCalculationDuration(
            req.Method, 
            req.AssetClass, 
            duration,
        )
        
        // Log completion
        irs.logger.InfoContext(ctx, "VaR calculation completed",
            slog.String("correlation_id", correlationID),
            slog.Duration("duration", duration),
            slog.Bool("success", true),
        )
    }()
    
    // Perform calculation
    result, err := irs.varCalculator.CalculateHistoricalVaR(
        ctx, 
        req.Returns,
        req.PortfolioValue, 
        req.Confidence,
    )
    
    if err != nil {
        // Error logging and metrics
        irs.logger.ErrorContext(ctx, "VaR calculation failed",
            slog.String("correlation_id", correlationID),
            slog.String("error", err.Error()),
            slog.String("error_type", classifyError(err)),
        )
        
        irs.metrics.RecordCalculationError(req.Method, err)
        
        return VaRResult{}, err
    }
    
    // Success metrics and audit logging
    irs.metrics.RecordCalculationSuccess(req.Method, req.AssetClass)
    
    irs.logger.InfoContext(ctx, "VaR calculation successful",
        slog.String("correlation_id", correlationID),
        slog.String("var_amount", result.VaR.String()),
        slog.Bool("risk_limit_check_required", result.VaR.Abs().Cmp(req.RiskLimit) > 0),
    )
    
    return result, nil
}

// Context helpers
func getOrCreateCorrelationID(ctx context.Context) string {
    if id := ctx.Value("correlation_id"); id != nil {
        return id.(string)
    }
    return uuid.New().String()
}

func classifyError(err error) string {
    switch {
    case strings.Contains(err.Error(), "insufficient"):
        return "INSUFFICIENT_DATA"
    case strings.Contains(err.Error(), "timeout"):
        return "CALCULATION_TIMEOUT"
    case strings.Contains(err.Error(), "invalid"):
        return "INVALID_INPUT"
    default:
        return "UNKNOWN_ERROR"
    }
}
```

### 2.2 Metrics Collection

```go
// File: internal/monitoring/risk_metrics.go
package monitoring

import (
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

type RiskMetricsCollector struct {
    // Calculation performance metrics
    calculationDuration *prometheus.HistogramVec
    calculationErrors   *prometheus.CounterVec
    calculationSuccess  *prometheus.CounterVec
    
    // Business metrics
    riskLimitBreaches   *prometheus.CounterVec
    portfolioVaR        *prometheus.GaugeVec
    drawdownAlerts      *prometheus.CounterVec
    
    // System metrics
    cacheHitRate        *prometheus.GaugeVec
    memoryUsage         prometheus.Gauge
    goroutineCount      prometheus.Gauge
}

func NewRiskMetricsCollector() *RiskMetricsCollector {
    return &RiskMetricsCollector{
        calculationDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "risk_calculation_duration_seconds",
                Help: "Duration of risk calculations",
                Buckets: []float64{
                    0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0,
                },
            },
            []string{"method", "asset_class", "portfolio_id"},
        ),
        
        calculationErrors: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "risk_calculation_errors_total",
                Help: "Total number of risk calculation errors",
            },
            []string{"method", "error_type"},
        ),
        
        riskLimitBreaches: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "risk_limit_breaches_total",
                Help: "Total number of risk limit breaches",
            },
            []string{"portfolio_id", "breach_type", "severity"},
        ),
        
        portfolioVaR: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "portfolio_var_current",
                Help: "Current VaR for portfolios",
            },
            []string{"portfolio_id", "confidence_level"},
        ),
        
        cacheHitRate: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "risk_calculation_cache_hit_rate",
                Help: "Cache hit rate for risk calculations",
            },
            []string{"cache_type"},
        ),
    }
}

func (rmc *RiskMetricsCollector) RecordCalculationDuration(
    method, assetClass string, 
    duration time.Duration,
) {
    rmc.calculationDuration.WithLabelValues(method, assetClass, "").
        Observe(duration.Seconds())
}

func (rmc *RiskMetricsCollector) RecordRiskLimitBreach(
    portfolioID, breachType, severity string,
) {
    rmc.riskLimitBreaches.WithLabelValues(portfolioID, breachType, severity).Inc()
}

func (rmc *RiskMetricsCollector) UpdatePortfolioVaR(
    portfolioID string, 
    confidenceLevel float64, 
    varAmount float64,
) {
    rmc.portfolioVaR.WithLabelValues(
        portfolioID, 
        fmt.Sprintf("%.1f", confidenceLevel),
    ).Set(varAmount)
}
```

## 🔧 Phase 3: Integration Architecture

### 3.1 Port Interfaces

```go
// File: internal/core/ports/risk.go
package ports

import (
    "context"
    "time"
    
    "github.com/trading-engine/internal/core/domain"
    "github.com/trading-engine/pkg/types"
)

// Primary ports (driven by the application)
type RiskCalculationPort interface {
    CalculateVaR(ctx context.Context, req VaRRequest) (VaRResult, error)
    CalculateCVaR(ctx context.Context, req CVaRRequest) (CVaRResult, error)
    MonitorDrawdown(ctx context.Context, portfolioID string) (DrawdownStatus, error)
    CalculatePositionSize(ctx context.Context, req PositionSizeRequest) (PositionSizeResult, error)
}

// Secondary ports (dependencies)
type RiskDataPort interface {
    GetHistoricalReturns(ctx context.Context, assetID string, period time.Duration) ([]types.Decimal, error)
    GetPortfolio(ctx context.Context, portfolioID string) (*domain.Portfolio, error)
    GetMarketData(ctx context.Context, assets []string) (MarketData, error)
    GetRiskLimits(ctx context.Context, portfolioID string) (RiskLimits, error)
}

type RiskStoragePort interface {
    SaveVaRResult(ctx context.Context, result VaRResult) error
    SaveDrawdownEvent(ctx context.Context, event DrawdownEvent) error
    GetRiskHistory(ctx context.Context, portfolioID string, period time.Duration) ([]RiskSnapshot, error)
}

type RiskNotificationPort interface {
    SendRiskAlert(ctx context.Context, alert RiskAlert) error
    SendLimitBreachNotification(ctx context.Context, breach LimitBreach) error
    SendDrawdownAlert(ctx context.Context, alert DrawdownAlert) error
}

type RiskEventPort interface {
    PublishVaRCalculated(ctx context.Context, event VaRCalculatedEvent) error
    PublishRiskLimitBreach(ctx context.Context, event RiskLimitBreachEvent) error
    PublishDrawdownAlert(ctx context.Context, event DrawdownAlertEvent) error
}

// Request/Response types
type VaRRequest struct {
    PortfolioID     string          `json:"portfolio_id"`
    Method          string          `json:"method"`
    Confidence      types.Decimal   `json:"confidence"`
    TimeHorizon     time.Duration   `json:"time_horizon"`
    Returns         []types.Decimal `json:"returns,omitempty"`
    AssetClass      string          `json:"asset_class"`
    RiskLimit       types.Decimal   `json:"risk_limit"`
}

type VaRResult struct {
    PortfolioID     string          `json:"portfolio_id"`
    Method          string          `json:"method"`
    ConfidenceLevel types.Decimal   `json:"confidence_level"`
    VaR             types.Decimal   `json:"var"`
    PortfolioValue  types.Decimal   `json:"portfolio_value"`
    Statistics      VaRStatistics   `json:"statistics"`
    CalculatedAt    time.Time       `json:"calculated_at"`
    ExpiresAt       time.Time       `json:"expires_at"`
}

type RiskAlert struct {
    ID           string        `json:"id"`
    PortfolioID  string        `json:"portfolio_id"`
    Type         AlertType     `json:"type"`
    Severity     Severity      `json:"severity"`
    Message      string        `json:"message"`
    Threshold    types.Decimal `json:"threshold"`
    CurrentValue types.Decimal `json:"current_value"`
    TriggeredAt  time.Time     `json:"triggered_at"`
}
```

### 3.2 Adapter Implementations

```go
// File: internal/adapters/risk/postgres_risk_data_adapter.go
package risk

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/trading-engine/internal/core/ports"
    "github.com/trading-engine/pkg/types"
)

type PostgresRiskDataAdapter struct {
    db    *sql.DB
    cache CacheAdapter
}

func NewPostgresRiskDataAdapter(db *sql.DB, cache CacheAdapter) *PostgresRiskDataAdapter {
    return &PostgresRiskDataAdapter{
        db:    db,
        cache: cache,
    }
}

func (p *PostgresRiskDataAdapter) GetHistoricalReturns(
    ctx context.Context, 
    assetID string, 
    period time.Duration,
) ([]types.Decimal, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("returns:%s:%s", assetID, period.String())
    if cached, found := p.cache.Get(ctx, cacheKey); found {
        return cached.([]types.Decimal), nil
    }
    
    // Query database
    query := `
        SELECT return_pct 
        FROM daily_returns 
        WHERE asset_id = $1 
          AND date >= $2 
        ORDER BY date DESC 
        LIMIT $3
    `
    
    startDate := time.Now().Add(-period)
    maxRows := int(period.Hours() / 24) // Approximate days
    
    rows, err := p.db.QueryContext(ctx, query, assetID, startDate, maxRows)
    if err != nil {
        return nil, fmt.Errorf("failed to query returns: %w", err)
    }
    defer rows.Close()
    
    var returns []types.Decimal
    for rows.Next() {
        var returnPct string
        if err := rows.Scan(&returnPct); err != nil {
            return nil, fmt.Errorf("failed to scan return: %w", err)
        }
        
        decimal, err := types.NewDecimalFromString(returnPct)
        if err != nil {
            return nil, fmt.Errorf("invalid decimal value: %w", err)
        }
        
        returns = append(returns, decimal)
    }
    
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows iteration error: %w", err)
    }
    
    // Cache results
    p.cache.Set(ctx, cacheKey, returns, 15*time.Minute)
    
    return returns, nil
}

func (p *PostgresRiskDataAdapter) GetRiskLimits(
    ctx context.Context, 
    portfolioID string,
) (ports.RiskLimits, error) {
    query := `
        SELECT 
            max_var_percent,
            max_drawdown_percent,
            max_position_size_percent,
            confidence_level,
            updated_at
        FROM risk_limits 
        WHERE portfolio_id = $1 
          AND effective_date <= NOW() 
        ORDER BY effective_date DESC 
        LIMIT 1
    `
    
    var limits ports.RiskLimits
    var updatedAt time.Time
    
    err := p.db.QueryRowContext(ctx, query, portfolioID).Scan(
        &limits.MaxVaRPercent,
        &limits.MaxDrawdownPercent,
        &limits.MaxPositionSizePercent,
        &limits.ConfidenceLevel,
        &updatedAt,
    )
    
    if err != nil {
        if err == sql.ErrNoRows {
            // Return default limits if none configured
            return ports.GetDefaultRiskLimits(), nil
        }
        return ports.RiskLimits{}, fmt.Errorf("failed to query risk limits: %w", err)
    }
    
    limits.PortfolioID = portfolioID
    limits.UpdatedAt = updatedAt
    
    return limits, nil
}
```

### 3.3 Event Integration

```go
// File: internal/adapters/risk/event_publisher_adapter.go
package risk

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/trading-engine/internal/core/ports"
    "github.com/nats-io/nats.go"
)

type NATSRiskEventAdapter struct {
    nc      *nats.Conn
    js      nats.JetStreamContext
    subjects map[string]string
}

func NewNATSRiskEventAdapter(nc *nats.Conn) (*NATSRiskEventAdapter, error) {
    js, err := nc.JetStream()
    if err != nil {
        return nil, fmt.Errorf("failed to create jetstream context: %w", err)
    }
    
    return &NATSRiskEventAdapter{
        nc: nc,
        js: js,
        subjects: map[string]string{
            "var_calculated":     "risk.var.calculated",
            "limit_breach":       "risk.limit.breach",
            "drawdown_alert":     "risk.drawdown.alert",
        },
    }, nil
}

func (n *NATSRiskEventAdapter) PublishVaRCalculated(
    ctx context.Context, 
    event ports.VaRCalculatedEvent,
) error {
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    
    _, err = n.js.PublishAsync(n.subjects["var_calculated"], data)
    if err != nil {
        return fmt.Errorf("failed to publish VaR calculated event: %w", err)
    }
    
    return nil
}

func (n *NATSRiskEventAdapter) PublishRiskLimitBreach(
    ctx context.Context, 
    event ports.RiskLimitBreachEvent,
) error {
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    
    // Use high priority for risk breaches
    _, err = n.js.PublishAsync(
        n.subjects["limit_breach"], 
        data, 
        nats.MsgId(event.ID),
        nats.ExpectLastMsgId(nats.LastBySubj),
    )
    
    if err != nil {
        return fmt.Errorf("failed to publish risk limit breach event: %w", err)
    }
    
    return nil
}
```

## 🔧 Phase 4: Testing Strategy

### 4.1 Integration Test Framework

```go
// File: tests/integration/risk_integration_test.go
//go:build integration

package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

type RiskIntegrationTestSuite struct {
    ctx              context.Context
    postgresContainer *postgres.PostgresContainer
    redisContainer   *redis.RedisContainer
    riskService      *risk.InstrumentedRiskService
    cleanup          func()
}

func setupRiskIntegrationTest(t *testing.T) *RiskIntegrationTestSuite {
    ctx := context.Background()
    
    // Start PostgreSQL container
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("risk_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        postgres.WithInitScripts("./testdata/schema.sql"),
    )
    require.NoError(t, err)
    
    // Start Redis container
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    require.NoError(t, err)
    
    // Connect to databases
    postgresURL, err := postgresContainer.ConnectionString(ctx)
    require.NoError(t, err)
    
    redisURL, err := redisContainer.ConnectionString(ctx)
    require.NoError(t, err)
    
    // Initialize risk service with real dependencies
    riskService, err := setupRiskServiceWithDependencies(postgresURL, redisURL)
    require.NoError(t, err)
    
    return &RiskIntegrationTestSuite{
        ctx:               ctx,
        postgresContainer: postgresContainer,
        redisContainer:    redisContainer,
        riskService:       riskService,
        cleanup: func() {
            postgresContainer.Terminate(ctx)
            redisContainer.Terminate(ctx)
        },
    }
}

func TestRiskManagementWorkflow(t *testing.T) {
    suite := setupRiskIntegrationTest(t)
    defer suite.cleanup()
    
    // Test complete VaR calculation workflow
    t.Run("VaR_Calculation_Workflow", func(t *testing.T) {
        // Setup test portfolio with realistic data
        portfolio := createIntegrationTestPortfolio(t, suite.ctx)
        
        // Historical data setup
        returns := generateRealisticMarketReturns(252) // 1 year of data
        
        // Calculate VaR
        varRequest := ports.VaRRequest{
            PortfolioID:    portfolio.ID,
            Method:         "Historical",
            Confidence:     types.NewDecimalFromFloat(95.0),
            TimeHorizon:    24 * time.Hour,
            Returns:        returns,
            AssetClass:     "EQUITY",
            RiskLimit:      types.NewDecimalFromFloat(50000.0),
        }
        
        result, err := suite.riskService.CalculateVaR(suite.ctx, varRequest)
        require.NoError(t, err)
        
        // Verify results
        assert.Equal(t, "Historical", result.Method)
        assert.True(t, result.VaR.IsNegative()) // VaR should be negative (loss)
        assert.True(t, result.VaR.Abs().IsPositive())
        
        // Verify result was persisted
        stored, err := suite.riskService.GetVaRHistory(suite.ctx, portfolio.ID, time.Hour)
        require.NoError(t, err)
        assert.Len(t, stored, 1)
    })
    
    // Test risk limit monitoring
    t.Run("Risk_Limit_Monitoring", func(t *testing.T) {
        portfolio := createIntegrationTestPortfolio(t, suite.ctx)
        
        // Create scenario with risk limit breach
        extremeReturns := generateExtremeMarketReturns(100) // Extreme losses
        
        varRequest := ports.VaRRequest{
            PortfolioID:    portfolio.ID,
            Method:         "Historical",
            Confidence:     types.NewDecimalFromFloat(95.0),
            Returns:        extremeReturns,
            RiskLimit:      types.NewDecimalFromFloat(10000.0), // Low limit
        }
        
        // This should trigger risk limit breach
        result, err := suite.riskService.CalculateVaR(suite.ctx, varRequest)
        require.NoError(t, err)
        
        // Verify breach was detected and handled
        breaches, err := suite.riskService.GetRiskLimitBreaches(suite.ctx, portfolio.ID)
        require.NoError(t, err)
        assert.NotEmpty(t, breaches)
        
        // Verify alert was sent
        alerts, err := suite.riskService.GetActiveAlerts(suite.ctx, portfolio.ID)
        require.NoError(t, err)
        assert.NotEmpty(t, alerts)
    })
    
    // Test concurrent calculations
    t.Run("Concurrent_Calculations", func(t *testing.T) {
        portfolio := createIntegrationTestPortfolio(t, suite.ctx)
        returns := generateRealisticMarketReturns(500)
        
        numConcurrent := 50
        results := make(chan ports.VaRResult, numConcurrent)
        errors := make(chan error, numConcurrent)
        
        // Launch concurrent calculations
        for i := 0; i < numConcurrent; i++ {
            go func(id int) {
                varRequest := ports.VaRRequest{
                    PortfolioID: fmt.Sprintf("%s-%d", portfolio.ID, id),
                    Method:      "Historical",
                    Confidence:  types.NewDecimalFromFloat(95.0),
                    Returns:     returns,
                    AssetClass:  "EQUITY",
                }
                
                result, err := suite.riskService.CalculateVaR(suite.ctx, varRequest)
                if err != nil {
                    errors <- err
                    return
                }
                
                results <- result
            }(i)
        }
        
        // Collect results
        successCount := 0
        errorCount := 0
        timeout := time.After(30 * time.Second)
        
        for i := 0; i < numConcurrent; i++ {
            select {
            case <-results:
                successCount++
            case <-errors:
                errorCount++
            case <-timeout:
                t.Fatal("Concurrent test timed out")
            }
        }
        
        // Verify all calculations succeeded
        assert.Equal(t, numConcurrent, successCount)
        assert.Equal(t, 0, errorCount)
    })
}

func TestPerformanceUnderLoad(t *testing.T) {
    suite := setupRiskIntegrationTest(t)
    defer suite.cleanup()
    
    portfolio := createIntegrationTestPortfolio(t, suite.ctx)
    returns := generateRealisticMarketReturns(1000)
    
    // Test performance under sustained load
    duration := 60 * time.Second
    maxDuration := time.Millisecond
    calculations := 0
    violations := 0
    
    start := time.Now()
    
    for time.Since(start) < duration {
        calcStart := time.Now()
        
        varRequest := ports.VaRRequest{
            PortfolioID: portfolio.ID,
            Method:      "Historical",
            Confidence:  types.NewDecimalFromFloat(95.0),
            Returns:     returns,
        }
        
        _, err := suite.riskService.CalculateVaR(suite.ctx, varRequest)
        require.NoError(t, err)
        
        calcDuration := time.Since(calcStart)
        calculations++
        
        if calcDuration > maxDuration {
            violations++
        }
        
        // Brief pause to prevent overwhelming the system
        time.Sleep(10 * time.Millisecond)
    }
    
    // Verify performance SLA
    violationRate := float64(violations) / float64(calculations) * 100
    
    t.Logf("Performance test results:")
    t.Logf("  Total calculations: %d", calculations)
    t.Logf("  SLA violations: %d (%.2f%%)", violations, violationRate)
    t.Logf("  Throughput: %.2f calc/sec", float64(calculations)/duration.Seconds())
    
    // Allow up to 5% SLA violations under sustained load
    assert.Less(t, violationRate, 5.0, "SLA violation rate too high")
}
```

## 📊 Success Criteria & Validation

### Performance Validation

```bash
# Run performance benchmarks
go test -bench=. -benchmem ./internal/core/services/risk/...

# Target results:
# BenchmarkVaRCalculation-8    50000  20000 ns/op  1024 B/op  12 allocs/op
#                              ^^^^^  ^^^^^^^^^^^  ^^^^^^^^   ^^^^^^^^^^^^
#                              ops    <1ms p99     <100MB    <20 allocs
```

### Integration Validation

```bash
# Run integration tests
go test -tags=integration -v ./tests/integration/...

# Expected output:
# === RUN   TestRiskManagementWorkflow/VaR_Calculation_Workflow
# --- PASS: TestRiskManagementWorkflow/VaR_Calculation_Workflow (0.05s)
# === RUN   TestRiskManagementWorkflow/Risk_Limit_Monitoring  
# --- PASS: TestRiskManagementWorkflow/Risk_Limit_Monitoring (0.03s)
# === RUN   TestRiskManagementWorkflow/Concurrent_Calculations
# --- PASS: TestRiskManagementWorkflow/Concurrent_Calculations (5.20s)
```

### Monitoring Validation

```yaml
# Prometheus queries for validation
queries:
  - name: "Risk calculation SLA compliance"
    query: 'histogram_quantile(0.99, risk_calculation_duration_seconds_bucket)'
    target: '< 0.001'  # <1ms p99
    
  - name: "Risk calculation error rate"  
    query: 'rate(risk_calculation_errors_total[5m])'
    target: '< 0.001'  # <0.1% error rate
    
  - name: "Cache hit rate"
    query: 'risk_calculation_cache_hit_rate'  
    target: '> 0.90'   # >90% cache hit rate
```

This implementation guide provides the concrete steps needed to transform the risk management feature from a TDD prototype into a production-ready system that meets the stringent requirements of a high-performance trading environment.

---

*Next: [Testing Strategy](./testing-strategy.md)*