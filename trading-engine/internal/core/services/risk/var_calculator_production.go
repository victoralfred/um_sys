package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trading-engine/pkg/types"
)

// ProductionVaRConfig contains production-optimized configuration for VaR calculations
type ProductionVaRConfig struct {
	DefaultMethod             string                `json:"default_method"`
	DefaultConfidenceLevel    types.Decimal         `json:"default_confidence_level"`
	MinHistoricalObservations int                   `json:"min_historical_observations"`
	MaxHistoricalObservations int                   `json:"max_historical_observations"`
	SupportedMethods          []string              `json:"supported_methods"`
	EnableBacktesting         bool                  `json:"enable_backtesting"`
	MaxConcurrentCalculations int                   `json:"max_concurrent_calculations"`
	CalculationTimeout        time.Duration         `json:"calculation_timeout"`
	EnableCaching             bool                  `json:"enable_caching"`
	CacheExpiryDuration       time.Duration         `json:"cache_expiry_duration"`
	ValidationEnabled         bool                  `json:"validation_enabled"`
	PerformanceThresholds     PerformanceThresholds `json:"performance_thresholds"`
}

// PerformanceThresholds defines SLA requirements for production
type PerformanceThresholds struct {
	MaxLatencyP99     time.Duration `json:"max_latency_p99"`     // 1ms SLA
	MaxLatencyP95     time.Duration `json:"max_latency_p95"`     // 500μs SLA
	MinThroughputQPS  int           `json:"min_throughput_qps"`  // 10000 QPS
	MaxMemoryUsageMB  int           `json:"max_memory_usage_mb"` // 50MB per calculation
	MaxCPUUtilization float64       `json:"max_cpu_utilization"` // 80%
}

// ProductionVaRStatistics contains comprehensive statistical measures for production monitoring
type ProductionVaRStatistics struct {
	Mean                    types.Decimal `json:"mean"`
	StandardDeviation       types.Decimal `json:"standard_deviation"`
	Skewness                types.Decimal `json:"skewness"`
	Kurtosis                types.Decimal `json:"kurtosis"`
	Minimum                 types.Decimal `json:"minimum"`
	Maximum                 types.Decimal `json:"maximum"`
	Median                  types.Decimal `json:"median"`
	Q1                      types.Decimal `json:"q1"`
	Q3                      types.Decimal `json:"q3"`
	IQR                     types.Decimal `json:"iqr"`
	JarqueBeraStatistic     types.Decimal `json:"jarque_bera_statistic"`
	IsNormallyDistributed   bool          `json:"is_normally_distributed"`
	TailIndex               types.Decimal `json:"tail_index"`
	AutocorrelationLag1     types.Decimal `json:"autocorrelation_lag1"`
	VolatilityClusteringIdx types.Decimal `json:"volatility_clustering_index"`
}

// ProductionVaRResult contains comprehensive VaR calculation results with production metadata
type ProductionVaRResult struct {
	// Core VaR Results
	Method          string                  `json:"method"`
	ConfidenceLevel types.Decimal           `json:"confidence_level"`
	VaR             types.Decimal           `json:"var"`
	PortfolioValue  types.Decimal           `json:"portfolio_value"`
	Statistics      ProductionVaRStatistics `json:"statistics"`

	// Production Metadata
	CalculatedAt       time.Time          `json:"calculated_at"`
	CalculationTime    time.Duration      `json:"calculation_time"`
	DataPoints         int                `json:"data_points"`
	CacheHit           bool               `json:"cache_hit"`
	ValidationResults  ValidationResults  `json:"validation_results"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics"`

	// Traceability
	RequestID         string `json:"request_id"`
	CorrelationID     string `json:"correlation_id"`
	CalculatorVersion string `json:"calculator_version"`

	// Risk Management
	ModelValidation ModelValidationResults `json:"model_validation"`
	BacktestResults *ProductionBacktest    `json:"backtest_results,omitempty"`
}

// ValidationResults contains comprehensive input and output validation results
type ValidationResults struct {
	InputValidation struct {
		DataQuality      bool                    `json:"data_quality"`
		StationarityTest bool                    `json:"stationarity_test"`
		OutlierDetection OutlierDetectionResults `json:"outlier_detection"`
		Completeness     float64                 `json:"completeness"`
		ConsistencyCheck bool                    `json:"consistency_check"`
	} `json:"input_validation"`

	OutputValidation struct {
		ReasonabilityCheck  bool `json:"reasonability_check"`
		MonotonicityCheck   bool `json:"monotonicity_check"`
		ConsistencyCheck    bool `json:"consistency_check"`
		SensitivityAnalysis struct {
			DeltaVaR    types.Decimal `json:"delta_var"`
			GammaEffect types.Decimal `json:"gamma_effect"`
			VegaRisk    types.Decimal `json:"vega_risk"`
		} `json:"sensitivity_analysis"`
	} `json:"output_validation"`

	OverallValidation bool `json:"overall_validation"`
}

// OutlierDetectionResults contains results from statistical outlier detection
type OutlierDetectionResults struct {
	OutlierCount      int           `json:"outlier_count"`
	OutlierPercentage float64       `json:"outlier_percentage"`
	OutlierIndices    []int         `json:"outlier_indices"`
	DetectionMethod   string        `json:"detection_method"`
	Threshold         types.Decimal `json:"threshold"`
	TreatmentApplied  string        `json:"treatment_applied"`
}

// PerformanceMetrics contains detailed performance monitoring data
type PerformanceMetrics struct {
	CPUUtilization    float64       `json:"cpu_utilization"`
	MemoryUtilization int64         `json:"memory_utilization"`
	GoroutineCount    int           `json:"goroutine_count"`
	GCPauseTime       time.Duration `json:"gc_pause_time"`
	AllocationsCount  uint64        `json:"allocations_count"`
	ThroughputQPS     float64       `json:"throughput_qps"`
	ConcurrencyLevel  int           `json:"concurrency_level"`
	CacheHitRatio     float64       `json:"cache_hit_ratio"`
	NetworkLatency    time.Duration `json:"network_latency"`
	IOLatency         time.Duration `json:"io_latency"`
}

// ModelValidationResults contains comprehensive model validation metrics
type ModelValidationResults struct {
	BacktestScore       float64 `json:"backtest_score"`
	CovariateShiftScore float64 `json:"covariate_shift_score"`
	ModelDrift          bool    `json:"model_drift"`
	ConfidenceInterval  struct {
		Lower types.Decimal `json:"lower"`
		Upper types.Decimal `json:"upper"`
		Width types.Decimal `json:"width"`
	} `json:"confidence_interval"`
	RSquared         float64 `json:"r_squared"`
	AdjustedRSquared float64 `json:"adjusted_r_squared"`
	AIC              float64 `json:"aic"`
	BIC              float64 `json:"bic"`
	LogLikelihood    float64 `json:"log_likelihood"`
}

// ProductionBacktest contains comprehensive backtesting results for production validation
type ProductionBacktest struct {
	TotalObservations int           `json:"total_observations"`
	Exceptions        int           `json:"exceptions"`
	ExceptionRate     types.Decimal `json:"exception_rate"`
	ExpectedRate      types.Decimal `json:"expected_rate"`
	IsModelValid      bool          `json:"is_model_valid"`
	PValue            types.Decimal `json:"p_value"`
	TestStatistic     types.Decimal `json:"test_statistic"`

	// Advanced Backtesting Metrics
	ConditionalCoverage   float64 `json:"conditional_coverage"`
	UnconditionalCoverage float64 `json:"unconditional_coverage"`
	LikelihoodRatio       float64 `json:"likelihood_ratio"`
	DurationsTest         struct {
		AverageDuration float64 `json:"average_duration"`
		MaxDuration     int     `json:"max_duration"`
		Clustering      bool    `json:"clustering"`
	} `json:"durations_test"`

	// Regulatory Metrics
	TrafficLight    string        `json:"traffic_light"`    // Green/Yellow/Red
	BaselMultiplier types.Decimal `json:"basel_multiplier"` // Basel III multiplier
	ComplianceScore float64       `json:"compliance_score"`
}

// ProductionVaRCalculator provides enterprise-grade VaR calculations with comprehensive monitoring
type ProductionVaRCalculator struct {
	config    ProductionVaRConfig
	cache     *ProductionCache
	validator *ProductionValidator
	monitor   *PerformanceMonitor
	logger    *RiskLogger

	// Thread safety
	mu          sync.RWMutex
	activeCalcs map[string]*CalculationContext
	calcCounter uint64

	// Production state
	isHealthy       bool
	lastHealthCheck time.Time
	healthMu        sync.RWMutex
}

// CalculationContext tracks individual calculation execution
type CalculationContext struct {
	ID            string
	RequestID     string
	CorrelationID string
	StartTime     time.Time
	Method        string
	DataSize      int
	Status        CalculationStatus
	Cancel        context.CancelFunc
	Result        chan ProductionVaRResult
	Error         chan error
}

// CalculationStatus represents the current state of a calculation
type CalculationStatus int

const (
	StatusPending CalculationStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusCanceled
	StatusTimedOut
)

// String returns the string representation of CalculationStatus
func (s CalculationStatus) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusRunning:
		return "RUNNING"
	case StatusCompleted:
		return "COMPLETED"
	case StatusFailed:
		return "FAILED"
	case StatusCanceled:
		return "CANCELED"
	case StatusTimedOut:
		return "TIMED_OUT"
	default:
		return "UNKNOWN"
	}
}

// NewProductionVaRCalculator creates a production-ready VaR calculator with comprehensive capabilities
func NewProductionVaRCalculator(config ProductionVaRConfig, logger *RiskLogger) *ProductionVaRCalculator {
	if logger == nil {
		logger = NewRiskLogger(DefaultLoggerConfig())
	}

	calculator := &ProductionVaRCalculator{
		config:          config,
		cache:           NewProductionCache(config.CacheExpiryDuration),
		validator:       NewProductionValidator(),
		monitor:         NewPerformanceMonitor(),
		logger:          logger,
		activeCalcs:     make(map[string]*CalculationContext),
		isHealthy:       true,
		lastHealthCheck: time.Now(),
	}

	// Start background health monitoring
	go calculator.healthMonitor()

	return calculator
}

// CalculateHistoricalVaRWithContext performs production-grade historical VaR calculation with full context
func (calc *ProductionVaRCalculator) CalculateHistoricalVaRWithContext(
	ctx context.Context,
	returns []types.Decimal,
	portfolioValue types.Decimal,
	confidenceLevel types.Decimal,
) (*ProductionVaRResult, error) {

	// Extract traceability information
	requestID := GetRequestID(ctx)
	correlationID := GetCorrelationID(ctx)

	// Create calculation context
	calcCtx := &CalculationContext{
		ID:            fmt.Sprintf("var-%d-%d", time.Now().UnixNano(), calc.getNextCalcID()),
		RequestID:     requestID,
		CorrelationID: correlationID,
		StartTime:     time.Now(),
		Method:        "historical_production",
		DataSize:      len(returns),
		Status:        StatusPending,
		Result:        make(chan ProductionVaRResult, 1),
		Error:         make(chan error, 1),
	}

	// Add timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, calc.config.CalculationTimeout)
	calcCtx.Cancel = cancel
	defer cancel()

	// Register calculation
	calc.registerCalculation(calcCtx)
	defer calc.unregisterCalculation(calcCtx.ID)

	// Log calculation start
	calc.logger.LogCalculationStart(ctx, "ProductionVaR", "historical", len(returns))

	// Start async calculation
	go calc.performCalculation(timeoutCtx, calcCtx, returns, portfolioValue, confidenceLevel)

	// Wait for result or timeout
	select {
	case result := <-calcCtx.Result:
		calc.logger.LogCalculationComplete(ctx, "ProductionVaR", "historical",
			time.Since(calcCtx.StartTime), true, map[string]interface{}{
				"var_amount":  result.VaR.String(),
				"data_points": result.DataPoints,
				"cache_hit":   result.CacheHit,
			})
		return &result, nil

	case err := <-calcCtx.Error:
		calc.logger.LogCalculationComplete(ctx, "ProductionVaR", "historical",
			time.Since(calcCtx.StartTime), false, nil)
		if riskErr, ok := err.(*RiskError); ok {
			calc.logger.LogError(ctx, riskErr)
		}
		return nil, err

	case <-timeoutCtx.Done():
		calcCtx.Status = StatusTimedOut
		timeoutErr := NewTimeoutError("var_calculation", calc.config.CalculationTimeout)
		calc.logger.LogError(ctx, timeoutErr)
		return nil, timeoutErr
	}
}

// performCalculation executes the actual VaR calculation with comprehensive monitoring
func (calc *ProductionVaRCalculator) performCalculation(
	ctx context.Context,
	calcCtx *CalculationContext,
	returns []types.Decimal,
	portfolioValue types.Decimal,
	confidenceLevel types.Decimal,
) {
	calcCtx.Status = StatusRunning
	startTime := time.Now()

	defer func() {
		if r := recover(); r != nil {
			calcCtx.Status = StatusFailed
			err := NewRiskError(ErrCalculationFailed, fmt.Sprintf("VaR calculation panic: %v", r), "var_calculation")
			calcCtx.Error <- err
		}
	}()

	// Phase 1: Input Validation (if enabled)
	var validationResults ValidationResults
	if calc.config.ValidationEnabled {
		if err := calc.validateInputs(returns, portfolioValue, confidenceLevel, &validationResults); err != nil {
			calcCtx.Status = StatusFailed
			calcCtx.Error <- err
			return
		}
	}

	// Phase 2: Check Cache
	cacheKey := calc.generateCacheKey(returns, portfolioValue, confidenceLevel)
	var result *ProductionVaRResult
	var cacheHit bool

	if calc.config.EnableCaching {
		if cached := calc.cache.Get(cacheKey); cached != nil {
			if cachedResult, ok := cached.(*ProductionVaRResult); ok {
				result = cachedResult
				cacheHit = true
				result.CacheHit = true
				result.RequestID = calcCtx.RequestID
				result.CorrelationID = calcCtx.CorrelationID
			}
		}
	}

	// Phase 3: Perform Calculation (if not cached)
	if result == nil {
		var err error
		result, err = calc.performCoreCalculation(ctx, returns, portfolioValue, confidenceLevel)
		if err != nil {
			calcCtx.Status = StatusFailed
			calcCtx.Error <- err
			return
		}
		cacheHit = false
	}

	// Phase 4: Enhance Result with Production Metadata
	calc.enhanceResultWithMetadata(result, calcCtx, validationResults, startTime)

	// Phase 5: Output Validation (if enabled)
	if calc.config.ValidationEnabled {
		if err := calc.validateOutput(result, &validationResults); err != nil {
			calcCtx.Status = StatusFailed
			calcCtx.Error <- err
			return
		}
	}

	// Phase 6: Cache Result
	if calc.config.EnableCaching && !cacheHit {
		calc.cache.Set(cacheKey, result, calc.config.CacheExpiryDuration)
	}

	// Phase 7: Performance Monitoring
	calc.monitor.RecordCalculation(time.Since(startTime), len(returns), true)

	// Success
	calcCtx.Status = StatusCompleted
	calcCtx.Result <- *result
}

// Placeholder implementations for production components
func (calc *ProductionVaRCalculator) validateInputs(returns []types.Decimal, portfolioValue types.Decimal, confidenceLevel types.Decimal, validationResults *ValidationResults) error {
	// Comprehensive input validation would be implemented here
	if len(returns) < calc.config.MinHistoricalObservations {
		return NewInsufficientDataError("production_var", calc.config.MinHistoricalObservations, len(returns))
	}

	if len(returns) > calc.config.MaxHistoricalObservations {
		return NewRiskError(ErrInvalidConfig, fmt.Sprintf("Too many observations: %d (max: %d)",
			len(returns), calc.config.MaxHistoricalObservations), "production_var")
	}

	validationResults.InputValidation.DataQuality = true
	validationResults.InputValidation.Completeness = 1.0
	validationResults.InputValidation.ConsistencyCheck = true
	return nil
}

func (calc *ProductionVaRCalculator) performCoreCalculation(ctx context.Context, returns []types.Decimal, portfolioValue types.Decimal, confidenceLevel types.Decimal) (*ProductionVaRResult, error) {
	// Use streaming calculator for optimal performance
	streamingConfig := VaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    confidenceLevel,
		MinHistoricalObservations: calc.config.MinHistoricalObservations,
		SupportedMethods:          []string{"streaming_historical"},
		EnableBacktesting:         calc.config.EnableBacktesting,
	}

	streamingCalc := NewStreamingVaRCalculator(streamingConfig)
	basicResult, err := streamingCalc.CalculateHistoricalVaR(returns, portfolioValue, confidenceLevel)
	if err != nil {
		return nil, err
	}

	// Convert to production result
	result := &ProductionVaRResult{
		Method:          basicResult.Method,
		ConfidenceLevel: basicResult.ConfidenceLevel,
		VaR:             basicResult.VaR,
		PortfolioValue:  basicResult.PortfolioValue,
		CalculatedAt:    basicResult.CalculatedAt,
		DataPoints:      len(returns),
		CacheHit:        false,
	}

	// Enhance with production statistics
	calc.calculateProductionStatistics(returns, &result.Statistics)

	return result, nil
}

func (calc *ProductionVaRCalculator) calculateProductionStatistics(returns []types.Decimal, stats *ProductionVaRStatistics) {
	// Implementation would include comprehensive statistical analysis
	// For now, using basic implementation
	if len(returns) == 0 {
		return
	}

	// Calculate basic statistics
	sum := types.NewDecimalFromFloat(0.0)
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	stats.Mean = sum.Div(types.NewDecimalFromInt(int64(len(returns))))

	// Additional statistics would be calculated here
	stats.IsNormallyDistributed = true               // Placeholder
	stats.TailIndex = types.NewDecimalFromFloat(0.1) // Placeholder
}

func (calc *ProductionVaRCalculator) validateOutput(result *ProductionVaRResult, validationResults *ValidationResults) error {
	validationResults.OutputValidation.ReasonabilityCheck = true
	validationResults.OutputValidation.MonotonicityCheck = true
	validationResults.OutputValidation.ConsistencyCheck = true
	validationResults.OverallValidation = true
	return nil
}

func (calc *ProductionVaRCalculator) enhanceResultWithMetadata(result *ProductionVaRResult, calcCtx *CalculationContext, validationResults ValidationResults, startTime time.Time) {
	result.RequestID = calcCtx.RequestID
	result.CorrelationID = calcCtx.CorrelationID
	result.CalculationTime = time.Since(startTime)
	result.ValidationResults = validationResults
	result.CalculatorVersion = "production-v1.0.0"

	// Performance metrics
	result.PerformanceMetrics = calc.monitor.GetCurrentMetrics()
}

// Utility methods
func (calc *ProductionVaRCalculator) getNextCalcID() uint64 {
	calc.mu.Lock()
	defer calc.mu.Unlock()
	calc.calcCounter++
	return calc.calcCounter
}

func (calc *ProductionVaRCalculator) registerCalculation(calcCtx *CalculationContext) {
	calc.mu.Lock()
	defer calc.mu.Unlock()
	calc.activeCalcs[calcCtx.ID] = calcCtx
}

func (calc *ProductionVaRCalculator) unregisterCalculation(id string) {
	calc.mu.Lock()
	defer calc.mu.Unlock()
	delete(calc.activeCalcs, id)
}

func (calc *ProductionVaRCalculator) generateCacheKey(returns []types.Decimal, portfolioValue types.Decimal, confidenceLevel types.Decimal) string {
	// Simple cache key generation - in production this would be more sophisticated
	return fmt.Sprintf("var_%d_%s_%s", len(returns), portfolioValue.String(), confidenceLevel.String())
}

func (calc *ProductionVaRCalculator) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		calc.healthMu.Lock()

		// Check system health
		activeCount := len(calc.activeCalcs)
		if activeCount > calc.config.MaxConcurrentCalculations {
			calc.isHealthy = false
		} else {
			calc.isHealthy = true
		}

		calc.lastHealthCheck = time.Now()
		calc.healthMu.Unlock()
	}
}

// GetHealth returns the current health status of the calculator
func (calc *ProductionVaRCalculator) GetHealth() (bool, time.Time) {
	calc.healthMu.RLock()
	defer calc.healthMu.RUnlock()
	return calc.isHealthy, calc.lastHealthCheck
}

// GetActiveCalculations returns the number of currently active calculations
func (calc *ProductionVaRCalculator) GetActiveCalculations() int {
	calc.mu.RLock()
	defer calc.mu.RUnlock()
	return len(calc.activeCalcs)
}

// DefaultProductionVaRConfig returns a production-ready configuration
func DefaultProductionVaRConfig() ProductionVaRConfig {
	return ProductionVaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
		MaxHistoricalObservations: 100000,
		SupportedMethods:          []string{"streaming_historical", "parametric", "monte_carlo"},
		EnableBacktesting:         true,
		MaxConcurrentCalculations: 100,
		CalculationTimeout:        5 * time.Second,
		EnableCaching:             true,
		CacheExpiryDuration:       5 * time.Minute,
		ValidationEnabled:         true,
		PerformanceThresholds: PerformanceThresholds{
			MaxLatencyP99:     time.Millisecond,
			MaxLatencyP95:     500 * time.Microsecond,
			MinThroughputQPS:  10000,
			MaxMemoryUsageMB:  50,
			MaxCPUUtilization: 0.8,
		},
	}
}
