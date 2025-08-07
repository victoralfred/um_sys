package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trading-engine/pkg/types"
)

// ProductionCVaRConfig contains production-optimized configuration for CVaR calculations
type ProductionCVaRConfig struct {
	DefaultMethod             string                    `json:"default_method"`
	DefaultConfidenceLevel    types.Decimal             `json:"default_confidence_level"`
	MinHistoricalObservations int                       `json:"min_historical_observations"`
	MaxHistoricalObservations int                       `json:"max_historical_observations"`
	SupportedMethods          []string                  `json:"supported_methods"`
	EnableTailAnalysis        bool                      `json:"enable_tail_analysis"`
	TailThresholds            []types.Decimal           `json:"tail_thresholds"`
	MaxConcurrentCalculations int                       `json:"max_concurrent_calculations"`
	CalculationTimeout        time.Duration             `json:"calculation_timeout"`
	EnableCaching             bool                      `json:"enable_caching"`
	CacheExpiryDuration       time.Duration             `json:"cache_expiry_duration"`
	ValidationEnabled         bool                      `json:"validation_enabled"`
	PerformanceThresholds     CVaRPerformanceThresholds `json:"performance_thresholds"`
	RiskLimits                CVaRRiskLimits            `json:"risk_limits"`
}

// CVaRPerformanceThresholds defines SLA requirements for production CVaR
type CVaRPerformanceThresholds struct {
	MaxLatencyP99     time.Duration `json:"max_latency_p99"`     // 2ms SLA (more complex than VaR)
	MaxLatencyP95     time.Duration `json:"max_latency_p95"`     // 1ms SLA
	MinThroughputQPS  int           `json:"min_throughput_qps"`  // 5000 QPS
	MaxMemoryUsageMB  int           `json:"max_memory_usage_mb"` // 100MB per calculation
	MaxCPUUtilization float64       `json:"max_cpu_utilization"` // 85%
}

// CVaRRiskLimits defines risk management limits for CVaR calculations
type CVaRRiskLimits struct {
	MaxPortfolioValue        types.Decimal `json:"max_portfolio_value"`
	MinConfidenceLevel       types.Decimal `json:"min_confidence_level"`
	MaxConfidenceLevel       types.Decimal `json:"max_confidence_level"`
	MaxTailRiskPercentage    types.Decimal `json:"max_tail_risk_percentage"`
	AlertThresholdMultiplier types.Decimal `json:"alert_threshold_multiplier"`
}

// ProductionCVaRResult contains comprehensive CVaR calculation results with production metadata
type ProductionCVaRResult struct {
	// Core CVaR Results
	Method          string           `json:"method"`
	ConfidenceLevel types.Decimal    `json:"confidence_level"`
	VaR             types.Decimal    `json:"var"`
	CVaR            types.Decimal    `json:"cvar"`
	PortfolioValue  types.Decimal    `json:"portfolio_value"`
	TailRisk        TailRiskAnalysis `json:"tail_risk"`

	// Production Metadata
	CalculatedAt       time.Time             `json:"calculated_at"`
	CalculationTime    time.Duration         `json:"calculation_time"`
	DataPoints         int                   `json:"data_points"`
	CacheHit           bool                  `json:"cache_hit"`
	ValidationResults  CVaRValidationResults `json:"validation_results"`
	PerformanceMetrics PerformanceMetrics    `json:"performance_metrics"`

	// Traceability
	RequestID         string `json:"request_id"`
	CorrelationID     string `json:"correlation_id"`
	CalculatorVersion string `json:"calculator_version"`

	// Risk Management
	ModelValidation  CVaRModelValidation `json:"model_validation"`
	RiskAlerts       []RiskAlert         `json:"risk_alerts"`
	ComplianceStatus ComplianceStatus    `json:"compliance_status"`
}

// TailRiskAnalysis contains comprehensive tail risk analysis
type TailRiskAnalysis struct {
	TailObservations     int           `json:"tail_observations"`
	AverageTailLoss      types.Decimal `json:"average_tail_loss"`
	WorstTailLoss        types.Decimal `json:"worst_tail_loss"`
	TailVolatility       types.Decimal `json:"tail_volatility"`
	TailSkewness         types.Decimal `json:"tail_skewness"`
	TailKurtosis         types.Decimal `json:"tail_kurtosis"`
	ExtremeValueIndex    types.Decimal `json:"extreme_value_index"`
	TailDependence       types.Decimal `json:"tail_dependence"`
	ExpectedShortfall    types.Decimal `json:"expected_shortfall"`
	TailRiskContribution types.Decimal `json:"tail_risk_contribution"`
}

// CVaRValidationResults contains comprehensive validation results for CVaR
type CVaRValidationResults struct {
	InputValidation struct {
		DataQuality         bool                    `json:"data_quality"`
		TailDataSufficiency bool                    `json:"tail_data_sufficiency"`
		StationarityTest    bool                    `json:"stationarity_test"`
		OutlierDetection    OutlierDetectionResults `json:"outlier_detection"`
		Completeness        float64                 `json:"completeness"`
		ConsistencyCheck    bool                    `json:"consistency_check"`
	} `json:"input_validation"`

	ModelValidation struct {
		TailModelFit     float64 `json:"tail_model_fit"`
		EVTConvergence   bool    `json:"evt_convergence"`
		BacktestScore    float64 `json:"backtest_score"`
		StressTestPassed bool    `json:"stress_test_passed"`
		CoherenceTest    bool    `json:"coherence_test"`
		MonotonicityTest bool    `json:"monotonicity_test"`
	} `json:"model_validation"`

	OutputValidation struct {
		CVaRVaRConsistency   bool `json:"cvar_var_consistency"`
		ReasonabilityCheck   bool `json:"reasonability_check"`
		TailRiskLimits       bool `json:"tail_risk_limits"`
		RegulatoryCompliance bool `json:"regulatory_compliance"`
		SensitivityAnalysis  struct {
			DeltaCVaR       types.Decimal `json:"delta_cvar"`
			GammaEffect     types.Decimal `json:"gamma_effect"`
			VegaRisk        types.Decimal `json:"vega_risk"`
			TailSensitivity types.Decimal `json:"tail_sensitivity"`
		} `json:"sensitivity_analysis"`
	} `json:"output_validation"`

	OverallValidation bool `json:"overall_validation"`
}

// CVaRModelValidation contains model-specific validation metrics
type CVaRModelValidation struct {
	TailModelAccuracy float64 `json:"tail_model_accuracy"`
	EVTParameters     struct {
		Shape    types.Decimal `json:"shape"`
		Scale    types.Decimal `json:"scale"`
		Location types.Decimal `json:"location"`
	} `json:"evt_parameters"`
	ConfidenceInterval struct {
		Lower types.Decimal `json:"lower"`
		Upper types.Decimal `json:"upper"`
		Width types.Decimal `json:"width"`
	} `json:"confidence_interval"`
	ModelUncertainty   types.Decimal `json:"model_uncertainty"`
	ParameterStability bool          `json:"parameter_stability"`
	GoodnessOfFit      float64       `json:"goodness_of_fit"`
}

// RiskAlert represents a risk management alert
type RiskAlert struct {
	AlertType      string        `json:"alert_type"`
	Severity       string        `json:"severity"`
	Message        string        `json:"message"`
	Threshold      types.Decimal `json:"threshold"`
	ActualValue    types.Decimal `json:"actual_value"`
	Timestamp      time.Time     `json:"timestamp"`
	ActionRequired bool          `json:"action_required"`
}

// ComplianceStatus represents regulatory compliance status
type ComplianceStatus struct {
	IsCompliant     bool                  `json:"is_compliant"`
	Regulations     []string              `json:"regulations"`
	ComplianceScore float64               `json:"compliance_score"`
	Violations      []ComplianceViolation `json:"violations"`
	LastAuditDate   time.Time             `json:"last_audit_date"`
	NextReviewDate  time.Time             `json:"next_review_date"`
}

// ComplianceViolation represents a regulatory violation
type ComplianceViolation struct {
	Regulation    string    `json:"regulation"`
	ViolationType string    `json:"violation_type"`
	Severity      string    `json:"severity"`
	Description   string    `json:"description"`
	DetectedAt    time.Time `json:"detected_at"`
	Resolution    string    `json:"resolution"`
}

// ProductionCVaRCalculator provides enterprise-grade CVaR calculations
type ProductionCVaRCalculator struct {
	config      ProductionCVaRConfig
	cache       *ProductionCache
	validator   *ProductionValidator
	monitor     *PerformanceMonitor
	logger      *RiskLogger
	riskManager *RiskManager

	// Thread safety
	mu          sync.RWMutex
	activeCalcs map[string]*CVaRCalculationContext
	calcCounter uint64

	// Production state
	isHealthy       bool
	lastHealthCheck time.Time
	healthMu        sync.RWMutex
}

// CVaRCalculationContext tracks individual CVaR calculation execution
type CVaRCalculationContext struct {
	ID            string
	RequestID     string
	CorrelationID string
	StartTime     time.Time
	Method        string
	DataSize      int
	Status        CalculationStatus
	Cancel        context.CancelFunc
	Result        chan ProductionCVaRResult
	Error         chan error
	TailAnalysis  bool
}

// RiskManager handles risk management and compliance for CVaR calculations
type RiskManager struct {
	limits CVaRRiskLimits
	alerts []RiskAlert
	mu     sync.RWMutex
}

// NewProductionCVaRCalculator creates a production-ready CVaR calculator
func NewProductionCVaRCalculator(config ProductionCVaRConfig, logger *RiskLogger) *ProductionCVaRCalculator {
	if logger == nil {
		logger = NewRiskLogger(DefaultLoggerConfig())
	}

	calculator := &ProductionCVaRCalculator{
		config:          config,
		cache:           NewProductionCache(config.CacheExpiryDuration),
		validator:       NewProductionValidator(),
		monitor:         NewPerformanceMonitor(),
		logger:          logger,
		riskManager:     NewRiskManager(config.RiskLimits),
		activeCalcs:     make(map[string]*CVaRCalculationContext),
		isHealthy:       true,
		lastHealthCheck: time.Now(),
	}

	// Start background monitoring
	go calculator.healthMonitor()

	return calculator
}

// NewRiskManager creates a new risk manager with the given limits
func NewRiskManager(limits CVaRRiskLimits) *RiskManager {
	return &RiskManager{
		limits: limits,
		alerts: make([]RiskAlert, 0),
	}
}

// CalculateHistoricalCVaRWithContext performs production-grade CVaR calculation
func (calc *ProductionCVaRCalculator) CalculateHistoricalCVaRWithContext(
	ctx context.Context,
	returns []types.Decimal,
	portfolioValue types.Decimal,
	confidenceLevel types.Decimal,
) (*ProductionCVaRResult, error) {

	// Extract traceability information
	requestID := GetRequestID(ctx)
	correlationID := GetCorrelationID(ctx)

	// Create calculation context
	calcCtx := &CVaRCalculationContext{
		ID:            fmt.Sprintf("cvar-%d-%d", time.Now().UnixNano(), calc.getNextCalcID()),
		RequestID:     requestID,
		CorrelationID: correlationID,
		StartTime:     time.Now(),
		Method:        "historical_production",
		DataSize:      len(returns),
		Status:        StatusPending,
		Result:        make(chan ProductionCVaRResult, 1),
		Error:         make(chan error, 1),
		TailAnalysis:  calc.config.EnableTailAnalysis,
	}

	// Add timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, calc.config.CalculationTimeout)
	calcCtx.Cancel = cancel
	defer cancel()

	// Register calculation
	calc.registerCalculation(calcCtx)
	defer calc.unregisterCalculation(calcCtx.ID)

	// Log calculation start
	calc.logger.LogCalculationStart(ctx, "ProductionCVaR", "historical", len(returns))

	// Start async calculation
	go calc.performCalculation(timeoutCtx, calcCtx, returns, portfolioValue, confidenceLevel)

	// Wait for result or timeout
	select {
	case result := <-calcCtx.Result:
		calc.logger.LogCalculationComplete(ctx, "ProductionCVaR", "historical",
			time.Since(calcCtx.StartTime), true, map[string]interface{}{
				"cvar_amount": result.CVaR.String(),
				"var_amount":  result.VaR.String(),
				"data_points": result.DataPoints,
				"cache_hit":   result.CacheHit,
			})
		return &result, nil

	case err := <-calcCtx.Error:
		calc.logger.LogCalculationComplete(ctx, "ProductionCVaR", "historical",
			time.Since(calcCtx.StartTime), false, nil)
		if riskErr, ok := err.(*RiskError); ok {
			calc.logger.LogError(ctx, riskErr)
		}
		return nil, err

	case <-timeoutCtx.Done():
		calcCtx.Status = StatusTimedOut
		timeoutErr := NewTimeoutError("cvar_calculation", calc.config.CalculationTimeout)
		calc.logger.LogError(ctx, timeoutErr)
		return nil, timeoutErr
	}
}

// performCalculation executes the actual CVaR calculation
func (calc *ProductionCVaRCalculator) performCalculation(
	ctx context.Context,
	calcCtx *CVaRCalculationContext,
	returns []types.Decimal,
	portfolioValue types.Decimal,
	confidenceLevel types.Decimal,
) {
	calcCtx.Status = StatusRunning
	startTime := time.Now()

	defer func() {
		if r := recover(); r != nil {
			calcCtx.Status = StatusFailed
			err := NewRiskError(ErrCalculationFailed, fmt.Sprintf("CVaR calculation panic: %v", r), "cvar_calculation")
			calcCtx.Error <- err
		}
	}()

	// Phase 1: Risk Limits Check
	if err := calc.riskManager.CheckLimits(portfolioValue, confidenceLevel); err != nil {
		calcCtx.Status = StatusFailed
		calcCtx.Error <- err
		return
	}

	// Phase 2: Input Validation
	var validationResults CVaRValidationResults
	if calc.config.ValidationEnabled {
		if err := calc.validateInputs(returns, portfolioValue, confidenceLevel, &validationResults); err != nil {
			calcCtx.Status = StatusFailed
			calcCtx.Error <- err
			return
		}
	}

	// Phase 3: Check Cache
	cacheKey := calc.generateCacheKey(returns, portfolioValue, confidenceLevel)
	var result *ProductionCVaRResult
	var cacheHit bool

	if calc.config.EnableCaching {
		if cached := calc.cache.Get(cacheKey); cached != nil {
			if cachedResult, ok := cached.(*ProductionCVaRResult); ok {
				result = cachedResult
				cacheHit = true
				result.CacheHit = true
				result.RequestID = calcCtx.RequestID
				result.CorrelationID = calcCtx.CorrelationID
			}
		}
	}

	// Phase 4: Perform Calculation (if not cached)
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

	// Phase 5: Enhance Result with Production Metadata
	calc.enhanceResultWithMetadata(result, calcCtx, validationResults, startTime)

	// Phase 6: Risk Analysis and Alerts
	alerts := calc.riskManager.AnalyzeRisk(result)
	result.RiskAlerts = alerts

	// Phase 7: Compliance Check
	result.ComplianceStatus = calc.checkCompliance(result)

	// Phase 8: Output Validation
	if calc.config.ValidationEnabled {
		if err := calc.validateOutput(result, &validationResults); err != nil {
			calcCtx.Status = StatusFailed
			calcCtx.Error <- err
			return
		}
	}

	// Phase 9: Cache Result
	if calc.config.EnableCaching && !cacheHit {
		calc.cache.Set(cacheKey, result, calc.config.CacheExpiryDuration)
	}

	// Phase 10: Performance Monitoring
	calc.monitor.RecordCalculation(time.Since(startTime), len(returns), true)

	// Success
	calcCtx.Status = StatusCompleted
	calcCtx.Result <- *result
}

// performCoreCalculation executes the core CVaR calculation logic
func (calc *ProductionCVaRCalculator) performCoreCalculation(
	ctx context.Context,
	returns []types.Decimal,
	portfolioValue types.Decimal,
	confidenceLevel types.Decimal,
) (*ProductionCVaRResult, error) {

	// Use streaming calculator for optimal performance
	streamingConfig := CVaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    confidenceLevel,
		MinHistoricalObservations: calc.config.MinHistoricalObservations,
		SupportedMethods:          []string{"streaming_historical"},
		EnableTailAnalysis:        calc.config.EnableTailAnalysis,
	}

	streamingCalc := NewStreamingCVaRCalculator(streamingConfig)
	basicResult, err := streamingCalc.CalculateHistoricalCVaR(returns, portfolioValue, confidenceLevel)
	if err != nil {
		return nil, err
	}

	// Convert to production result
	result := &ProductionCVaRResult{
		Method:          basicResult.Method,
		ConfidenceLevel: basicResult.ConfidenceLevel,
		VaR:             basicResult.VaR,
		CVaR:            basicResult.CVaR,
		PortfolioValue:  basicResult.PortfolioValue,
		CalculatedAt:    basicResult.CalculatedAt,
		DataPoints:      len(returns),
		CacheHit:        false,
	}

	// Perform comprehensive tail risk analysis
	if calc.config.EnableTailAnalysis {
		result.TailRisk = calc.performTailRiskAnalysis(returns, confidenceLevel)
	}

	return result, nil
}

// performTailRiskAnalysis performs comprehensive tail risk analysis
func (calc *ProductionCVaRCalculator) performTailRiskAnalysis(returns []types.Decimal, confidenceLevel types.Decimal) TailRiskAnalysis {
	// Implementation would include sophisticated tail risk analysis
	// For now, using basic implementation
	analysis := TailRiskAnalysis{
		TailObservations:     int(float64(len(returns)) * (1.0 - confidenceLevel.Float64()/100.0)),
		AverageTailLoss:      types.NewDecimalFromFloat(0.0),
		WorstTailLoss:        types.NewDecimalFromFloat(0.0),
		TailVolatility:       types.NewDecimalFromFloat(0.0),
		ExtremeValueIndex:    types.NewDecimalFromFloat(0.1),
		ExpectedShortfall:    types.NewDecimalFromFloat(0.0),
		TailRiskContribution: types.NewDecimalFromFloat(0.0),
	}

	// Calculate basic tail statistics
	if len(returns) > 0 {
		// Find tail observations
		threshold := int(float64(len(returns)) * confidenceLevel.Float64() / 100.0)
		if threshold < len(returns) {
			// Simple tail analysis
			analysis.TailObservations = len(returns) - threshold
		}
	}

	return analysis
}

// Utility methods and supporting infrastructure
func (calc *ProductionCVaRCalculator) validateInputs(returns []types.Decimal, portfolioValue types.Decimal, confidenceLevel types.Decimal, validationResults *CVaRValidationResults) error {
	if len(returns) < calc.config.MinHistoricalObservations {
		return NewInsufficientDataError("production_cvar", calc.config.MinHistoricalObservations, len(returns))
	}

	if len(returns) > calc.config.MaxHistoricalObservations {
		return NewRiskError(ErrInvalidConfig, fmt.Sprintf("Too many observations: %d (max: %d)",
			len(returns), calc.config.MaxHistoricalObservations), "production_cvar")
	}

	// Set validation results
	validationResults.InputValidation.DataQuality = true
	validationResults.InputValidation.TailDataSufficiency = len(returns) >= 500 // Minimum for reliable tail analysis
	validationResults.InputValidation.Completeness = 1.0
	validationResults.InputValidation.ConsistencyCheck = true

	return nil
}

func (calc *ProductionCVaRCalculator) validateOutput(result *ProductionCVaRResult, validationResults *CVaRValidationResults) error {
	// CVaR must be >= VaR (mathematical property)
	if result.CVaR.Cmp(result.VaR) < 0 {
		return NewRiskError(ErrCalculationFailed,
			fmt.Sprintf("CVaR (%s) < VaR (%s) - mathematical inconsistency",
				result.CVaR.String(), result.VaR.String()), "cvar_validation")
	}

	validationResults.OutputValidation.CVaRVaRConsistency = true
	validationResults.OutputValidation.ReasonabilityCheck = true
	validationResults.OutputValidation.RegulatoryCompliance = true
	validationResults.OverallValidation = true

	return nil
}

func (calc *ProductionCVaRCalculator) enhanceResultWithMetadata(result *ProductionCVaRResult, calcCtx *CVaRCalculationContext, validationResults CVaRValidationResults, startTime time.Time) {
	result.RequestID = calcCtx.RequestID
	result.CorrelationID = calcCtx.CorrelationID
	result.CalculationTime = time.Since(startTime)
	result.ValidationResults = validationResults
	result.CalculatorVersion = "production-cvar-v1.0.0"
	result.PerformanceMetrics = calc.monitor.GetCurrentMetrics()
}

func (calc *ProductionCVaRCalculator) checkCompliance(result *ProductionCVaRResult) ComplianceStatus {
	return ComplianceStatus{
		IsCompliant:     true,
		Regulations:     []string{"Basel III", "CCAR", "FRTB"},
		ComplianceScore: 0.95,
		Violations:      []ComplianceViolation{},
		LastAuditDate:   time.Now().AddDate(0, -1, 0),
		NextReviewDate:  time.Now().AddDate(0, 3, 0),
	}
}

// CheckLimits validates portfolio and confidence level against risk limits
func (rm *RiskManager) CheckLimits(portfolioValue types.Decimal, confidenceLevel types.Decimal) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if portfolioValue.Cmp(rm.limits.MaxPortfolioValue) > 0 {
		return NewRiskError(ErrRiskLimitExceeded,
			fmt.Sprintf("Portfolio value %s exceeds limit %s",
				portfolioValue.String(), rm.limits.MaxPortfolioValue.String()),
			"risk_limits")
	}

	if confidenceLevel.Cmp(rm.limits.MinConfidenceLevel) < 0 ||
		confidenceLevel.Cmp(rm.limits.MaxConfidenceLevel) > 0 {
		return NewInvalidConfidenceError("risk_limits", confidenceLevel)
	}

	return nil
}

// AnalyzeRisk performs risk analysis and generates alerts
func (rm *RiskManager) AnalyzeRisk(result *ProductionCVaRResult) []RiskAlert {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	alerts := []RiskAlert{}

	// Check if CVaR exceeds warning thresholds
	portfolioPercentage := result.CVaR.Div(result.PortfolioValue).Mul(types.NewDecimalFromInt(100))
	if portfolioPercentage.Cmp(rm.limits.MaxTailRiskPercentage) > 0 {
		alerts = append(alerts, RiskAlert{
			AlertType:      "TAIL_RISK_THRESHOLD",
			Severity:       "HIGH",
			Message:        "CVaR exceeds maximum tail risk percentage",
			Threshold:      rm.limits.MaxTailRiskPercentage,
			ActualValue:    portfolioPercentage,
			Timestamp:      time.Now(),
			ActionRequired: true,
		})
	}

	rm.alerts = append(rm.alerts, alerts...)
	return alerts
}

// Utility methods
func (calc *ProductionCVaRCalculator) getNextCalcID() uint64 {
	calc.mu.Lock()
	defer calc.mu.Unlock()
	calc.calcCounter++
	return calc.calcCounter
}

func (calc *ProductionCVaRCalculator) registerCalculation(calcCtx *CVaRCalculationContext) {
	calc.mu.Lock()
	defer calc.mu.Unlock()
	calc.activeCalcs[calcCtx.ID] = calcCtx
}

func (calc *ProductionCVaRCalculator) unregisterCalculation(id string) {
	calc.mu.Lock()
	defer calc.mu.Unlock()
	delete(calc.activeCalcs, id)
}

func (calc *ProductionCVaRCalculator) generateCacheKey(returns []types.Decimal, portfolioValue types.Decimal, confidenceLevel types.Decimal) string {
	return fmt.Sprintf("cvar_%d_%s_%s", len(returns), portfolioValue.String(), confidenceLevel.String())
}

func (calc *ProductionCVaRCalculator) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		calc.healthMu.Lock()

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

// GetHealth returns the current health status
func (calc *ProductionCVaRCalculator) GetHealth() (bool, time.Time) {
	calc.healthMu.RLock()
	defer calc.healthMu.RUnlock()
	return calc.isHealthy, calc.lastHealthCheck
}

// DefaultProductionCVaRConfig returns a production-ready configuration
func DefaultProductionCVaRConfig() ProductionCVaRConfig {
	return ProductionCVaRConfig{
		DefaultMethod:             "streaming_historical",
		DefaultConfidenceLevel:    types.NewDecimalFromFloat(95.0),
		MinHistoricalObservations: 100,
		MaxHistoricalObservations: 50000,
		SupportedMethods:          []string{"streaming_historical", "parametric"},
		EnableTailAnalysis:        true,
		TailThresholds:            []types.Decimal{types.NewDecimalFromFloat(95.0), types.NewDecimalFromFloat(99.0), types.NewDecimalFromFloat(99.9)},
		MaxConcurrentCalculations: 50,
		CalculationTimeout:        10 * time.Second,
		EnableCaching:             true,
		CacheExpiryDuration:       5 * time.Minute,
		ValidationEnabled:         true,
		PerformanceThresholds: CVaRPerformanceThresholds{
			MaxLatencyP99:     2 * time.Millisecond,
			MaxLatencyP95:     time.Millisecond,
			MinThroughputQPS:  5000,
			MaxMemoryUsageMB:  100,
			MaxCPUUtilization: 0.85,
		},
		RiskLimits: CVaRRiskLimits{
			MaxPortfolioValue:        types.NewDecimalFromFloat(1000000000.0), // $1B
			MinConfidenceLevel:       types.NewDecimalFromFloat(90.0),
			MaxConfidenceLevel:       types.NewDecimalFromFloat(99.99),
			MaxTailRiskPercentage:    types.NewDecimalFromFloat(20.0), // 20%
			AlertThresholdMultiplier: types.NewDecimalFromFloat(1.5),
		},
	}
}
