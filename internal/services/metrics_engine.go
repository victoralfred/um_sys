package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// MetricsEngine manages custom metric definitions and calculations
type MetricsEngine struct {
	mu              sync.RWMutex
	definitions     map[string]*CustomMetricDefinition
	cache           *MetricCache
	scheduler       *cron.Cron
	schedules       map[string]cron.EntryID
	dependencies    map[string][]string
	alerts          map[string]*AlertConfiguration
	simulatedValues map[string]float64
	updateChannels  map[string][]chan MetricUpdate
}

// CustomMetricDefinition defines a custom metric
type CustomMetricDefinition struct {
	ID              uuid.UUID
	Name            string
	Description     string
	Formula         string
	Unit            string
	Category        string
	AggregationType string
	Dimensions      []string
	TimeWindow      time.Duration
	CacheTTL        time.Duration
	Dependencies    []string
	RealTime        bool
	UpdateInterval  time.Duration
}

// CalculationParams parameters for metric calculation
type CalculationParams struct {
	StartTime time.Time
	EndTime   time.Time
	GroupBy   []string
	Filters   map[string]interface{}
}

// MetricResult represents a calculated metric result
type MetricResult struct {
	MetricName string
	Value      float64
	Timestamp  time.Time
	Dimensions map[string]string
	FromCache  bool
}

// CronSchedule defines a cron schedule for metric calculation
type CronSchedule struct {
	Expression string
	Enabled    bool
	LastRun    time.Time
	NextRun    time.Time
}

// AlertConfiguration defines alert rules for a metric
type AlertConfiguration struct {
	MetricName string
	Condition  string // above, below, equals, increase_by, decrease_by
	Threshold  float64
	Window     time.Duration
	Actions    []AlertAction
	triggered  bool
	lastValue  float64
}

// AlertAction defines an action to take when alert triggers
type AlertAction struct {
	Type   string // email, webhook, slack
	Target string
}

// MetricCache provides caching for metric results
type MetricCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedResult
}

// CachedResult represents a cached metric result
type CachedResult struct {
	Result    *MetricResult
	ExpiresAt time.Time
}

// MetricUpdate represents a real-time metric update
type MetricUpdate struct {
	MetricName string
	Value      float64
	Timestamp  time.Time
}

// NewMetricsEngine creates a new metrics engine
func NewMetricsEngine() *MetricsEngine {
	return &MetricsEngine{
		definitions:     make(map[string]*CustomMetricDefinition),
		cache:           &MetricCache{cache: make(map[string]*CachedResult)},
		scheduler:       cron.New(),
		schedules:       make(map[string]cron.EntryID),
		dependencies:    make(map[string][]string),
		alerts:          make(map[string]*AlertConfiguration),
		simulatedValues: make(map[string]float64),
		updateChannels:  make(map[string][]chan MetricUpdate),
	}
}

// Define defines a new custom metric
func (e *MetricsEngine) Define(metric *CustomMetricDefinition) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if metric.Name == "" {
		return errors.New("metric name is required")
	}

	if _, exists := e.definitions[metric.Name]; exists {
		return fmt.Errorf("metric %s already defined", metric.Name)
	}

	// Validate formula
	if err := e.ValidateFormula(metric.Formula); err != nil {
		return fmt.Errorf("invalid formula: %w", err)
	}

	e.definitions[metric.Name] = metric

	// Track dependencies
	if len(metric.Dependencies) > 0 {
		e.dependencies[metric.Name] = metric.Dependencies
	}

	// Start real-time updates if enabled
	if metric.RealTime {
		e.startRealTimeUpdates(metric)
	}

	return nil
}

// ValidateFormula validates a metric formula
func (e *MetricsEngine) ValidateFormula(formula string) error {
	if formula == "" {
		return errors.New("formula cannot be empty")
	}

	// Check for basic syntax errors
	if strings.Count(formula, "(") != strings.Count(formula, ")") {
		return errors.New("syntax error: unmatched parentheses")
	}

	// Check for supported functions
	supportedFunctions := []string{
		"count", "sum", "avg", "min", "max", "percentile",
		"stddev", "variance", "median", "moving_avg", "corr",
		"distinct",
	}

	// Check if formula contains undefined functions
	for _, part := range strings.Fields(formula) {
		if strings.Contains(part, "(") {
			funcName := strings.Split(part, "(")[0]
			if funcName != "" && !e.isSupportedFunction(funcName, supportedFunctions) {
				// Check if it's not a metric reference
				if !strings.HasPrefix(funcName, "metric.") {
					return fmt.Errorf("undefined function: %s", funcName)
				}
			}
		}
	}

	return nil
}

// GetDefinition retrieves a metric definition
func (e *MetricsEngine) GetDefinition(name string) (*CustomMetricDefinition, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	def, exists := e.definitions[name]
	if !exists {
		return nil, fmt.Errorf("metric %s not found", name)
	}

	return def, nil
}

// Calculate calculates a metric value
func (e *MetricsEngine) Calculate(ctx context.Context, metricName string, params CalculationParams) (*MetricResult, error) {
	e.mu.RLock()
	definition, exists := e.definitions[metricName]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("metric %s not found", metricName)
	}

	// Check cache first
	cacheKey := e.getCacheKey(metricName, params)
	if cached := e.cache.Get(cacheKey); cached != nil {
		cached.FromCache = true
		return cached, nil
	}

	// Calculate dependencies first
	if len(definition.Dependencies) > 0 {
		for _, dep := range definition.Dependencies {
			if _, err := e.Calculate(ctx, dep, params); err != nil {
				return nil, fmt.Errorf("failed to calculate dependency %s: %w", dep, err)
			}
		}
	}

	// Simulate calculation for testing
	var value float64
	if simValue, exists := e.simulatedValues[metricName]; exists {
		value = simValue
	} else {
		// In real implementation, this would execute the formula against the database
		value = e.simulateCalculation(definition.Formula, params)
	}

	result := &MetricResult{
		MetricName: metricName,
		Value:      value,
		Timestamp:  time.Now(),
		Dimensions: make(map[string]string),
		FromCache:  false,
	}

	// Cache result if TTL is set
	if definition.CacheTTL > 0 {
		e.cache.Set(cacheKey, result, definition.CacheTTL)
	}

	return result, nil
}

// CalculateWithDimensions calculates metric with dimension breakdown
func (e *MetricsEngine) CalculateWithDimensions(ctx context.Context, metricName string, params CalculationParams) ([]*MetricResult, error) {
	if len(params.GroupBy) == 0 {
		result, err := e.Calculate(ctx, metricName, params)
		if err != nil {
			return nil, err
		}
		return []*MetricResult{result}, nil
	}

	// Simulate dimension breakdown
	results := make([]*MetricResult, 0)
	dimensions := []string{"category1", "category2", "category3"}

	for _, dim := range dimensions {
		result := &MetricResult{
			MetricName: metricName,
			Value:      e.simulateCalculation("", params),
			Timestamp:  time.Now(),
			Dimensions: map[string]string{
				params.GroupBy[0]: dim,
			},
			FromCache: false,
		}
		results = append(results, result)
	}

	return results, nil
}

// Schedule schedules periodic calculation of a metric
func (e *MetricsEngine) Schedule(metricName string, schedule CronSchedule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.definitions[metricName]; !exists {
		return fmt.Errorf("metric %s not found", metricName)
	}

	// Remove existing schedule if any
	if entryID, exists := e.schedules[metricName]; exists {
		e.scheduler.Remove(entryID)
	}

	if !schedule.Enabled {
		return nil
	}

	// Add new schedule
	entryID, err := e.scheduler.AddFunc(schedule.Expression, func() {
		ctx := context.Background()
		params := CalculationParams{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		}
		e.Calculate(ctx, metricName, params)
	})

	if err != nil {
		return fmt.Errorf("failed to schedule metric: %w", err)
	}

	e.schedules[metricName] = entryID
	e.scheduler.Start()

	return nil
}

// GetSchedule retrieves the schedule for a metric
func (e *MetricsEngine) GetSchedule(metricName string) (*CronSchedule, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	entryID, exists := e.schedules[metricName]
	if !exists {
		return nil, fmt.Errorf("no schedule found for metric %s", metricName)
	}

	entry := e.scheduler.Entry(entryID)
	if entry.ID == 0 {
		return nil, fmt.Errorf("schedule entry not found")
	}

	return &CronSchedule{
		Expression: "0 * * * *", // Stored expression would be retrieved from definition
		Enabled:    true,
		LastRun:    entry.Prev,
		NextRun:    entry.Next,
	}, nil
}

// SubscribeToUpdates subscribes to real-time metric updates
func (e *MetricsEngine) SubscribeToUpdates(ctx context.Context, metricName string) (<-chan MetricUpdate, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	definition, exists := e.definitions[metricName]
	if !exists {
		return nil, fmt.Errorf("metric %s not found", metricName)
	}

	if !definition.RealTime {
		return nil, fmt.Errorf("metric %s is not real-time enabled", metricName)
	}

	ch := make(chan MetricUpdate, 100)
	e.updateChannels[metricName] = append(e.updateChannels[metricName], ch)

	// Send initial update
	go func() {
		ch <- MetricUpdate{
			MetricName: metricName,
			Value:      e.simulateCalculation(definition.Formula, CalculationParams{}),
			Timestamp:  time.Now(),
		}
	}()

	return ch, nil
}

// SetAlert sets an alert configuration for a metric
func (e *MetricsEngine) SetAlert(alert AlertConfiguration) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.definitions[alert.MetricName]; !exists {
		return fmt.Errorf("metric %s not found", alert.MetricName)
	}

	e.alerts[alert.MetricName] = &alert
	return nil
}

// IsAlertTriggered checks if an alert is triggered
func (e *MetricsEngine) IsAlertTriggered(metricName string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	alert, exists := e.alerts[metricName]
	if !exists {
		return false, fmt.Errorf("no alert configured for metric %s", metricName)
	}

	return alert.triggered, nil
}

// SimulateValue simulates a metric value for testing
func (e *MetricsEngine) SimulateValue(metricName string, value float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.simulatedValues[metricName] = value

	// Check alerts
	if alert, exists := e.alerts[metricName]; exists {
		switch alert.Condition {
		case "above":
			alert.triggered = value > alert.Threshold
		case "below":
			alert.triggered = value < alert.Threshold
		case "equals":
			alert.triggered = value == alert.Threshold
		}
		alert.lastValue = value
	}
}

// Helper methods

func (e *MetricsEngine) isSupportedFunction(funcName string, supported []string) bool {
	for _, s := range supported {
		if strings.EqualFold(funcName, s) {
			return true
		}
	}
	return false
}

func (e *MetricsEngine) getCacheKey(metricName string, params CalculationParams) string {
	return fmt.Sprintf("%s_%d_%d_%v",
		metricName,
		params.StartTime.Unix(),
		params.EndTime.Unix(),
		params.GroupBy)
}

func (e *MetricsEngine) simulateCalculation(formula string, params CalculationParams) float64 {
	// Simulate different values based on formula
	if strings.Contains(formula, "count") {
		return float64(100 + time.Now().Unix()%100)
	} else if strings.Contains(formula, "sum") {
		return float64(1000 + time.Now().Unix()%1000)
	} else if strings.Contains(formula, "avg") {
		return float64(50 + time.Now().Unix()%50)
	} else if strings.Contains(formula, "percentile") {
		return float64(95)
	}
	return float64(time.Now().Unix() % 100)
}

func (e *MetricsEngine) startRealTimeUpdates(metric *CustomMetricDefinition) {
	go func() {
		ticker := time.NewTicker(metric.UpdateInterval)
		defer ticker.Stop()

		for range ticker.C {
			e.mu.RLock()
			channels := e.updateChannels[metric.Name]
			e.mu.RUnlock()

			if len(channels) > 0 {
				update := MetricUpdate{
					MetricName: metric.Name,
					Value:      e.simulateCalculation(metric.Formula, CalculationParams{}),
					Timestamp:  time.Now(),
				}

				for _, ch := range channels {
					select {
					case ch <- update:
					default:
						// Channel full, skip
					}
				}
			}
		}
	}()
}

// MetricCache methods

func (c *MetricCache) Get(key string) *MetricResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.cache[key]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return nil
	}

	return cached.Result
}

func (c *MetricCache) Set(key string, result *MetricResult, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &CachedResult{
		Result:    result,
		ExpiresAt: time.Now().Add(ttl),
	}
}
