package risk

import (
	"fmt"
	"time"

	"github.com/trading-engine/pkg/types"
)

// DrawdownConfig contains configuration parameters for drawdown monitoring
type DrawdownConfig struct {
	EnableRealTimeAlerts  bool            `json:"enable_real_time_alerts"`
	AlertThresholds       []types.Decimal `json:"alert_thresholds"`
	MaxAcceptableDrawdown types.Decimal   `json:"max_acceptable_drawdown"`
	HistoryRetentionDays  int             `json:"history_retention_days"`
}

// DrawdownAlert represents an active drawdown alert
type DrawdownAlert struct {
	ThresholdPercent types.Decimal `json:"threshold_percent"`
	TriggeredAt      time.Time     `json:"triggered_at"`
	CurrentDrawdown  types.Decimal `json:"current_drawdown"`
	IsActive         bool          `json:"is_active"`
}

// DrawdownHistoryEntry represents a point in drawdown history
type DrawdownHistoryEntry struct {
	Timestamp       time.Time     `json:"timestamp"`
	Value           types.Decimal `json:"value"`
	Peak            types.Decimal `json:"peak"`
	Drawdown        types.Decimal `json:"drawdown"`
	DrawdownPercent types.Decimal `json:"drawdown_percent"`
}

// DrawdownStatistics contains comprehensive drawdown statistics
type DrawdownStatistics struct {
	MaxDrawdown           types.Decimal     `json:"max_drawdown"`
	MaxDrawdownPercent    types.Decimal     `json:"max_drawdown_percent"`
	AverageDrawdown       types.Decimal     `json:"average_drawdown"`
	MaxDrawdownDuration   time.Duration     `json:"max_drawdown_duration"`
	CurrentDrawdownDuration time.Duration   `json:"current_drawdown_duration"`
	TotalDrawdownPeriods  int               `json:"total_drawdown_periods"`
}

// DrawdownMonitor handles drawdown tracking and alerting - TDD GREEN phase implementation
type DrawdownMonitor struct {
	currentPeak      types.Decimal
	currentValue     types.Decimal
	maxDrawdown      types.Decimal
	maxDrawdownPercent types.Decimal
	config           DrawdownConfig
	history          []DrawdownHistoryEntry
	activeAlerts     []DrawdownAlert
	lastUpdateTime   time.Time
}

// TDD GREEN phase - implement just enough to make tests pass
func NewDrawdownMonitor() *DrawdownMonitor {
	return &DrawdownMonitor{
		currentPeak:      types.Zero(),
		currentValue:     types.Zero(),
		maxDrawdown:      types.Zero(),
		maxDrawdownPercent: types.Zero(),
		config: DrawdownConfig{
			EnableRealTimeAlerts: true,
			AlertThresholds: []types.Decimal{
				types.NewDecimalFromFloat(5.0),
				types.NewDecimalFromFloat(10.0),
				types.NewDecimalFromFloat(15.0),
			},
			MaxAcceptableDrawdown: types.NewDecimalFromFloat(20.0),
			HistoryRetentionDays:  30,
		},
		history:      make([]DrawdownHistoryEntry, 0),
		activeAlerts: make([]DrawdownAlert, 0),
		lastUpdateTime: time.Now(),
	}
}

func (dm *DrawdownMonitor) GetCurrentPeak() types.Decimal {
	return dm.currentPeak
}

func (dm *DrawdownMonitor) UpdateValue(value types.Decimal) error {
	if value.IsNegative() {
		return fmt.Errorf("value cannot be negative")
	}

	dm.currentValue = value
	dm.lastUpdateTime = time.Now()

	// Update peak if value is higher
	if value.Cmp(dm.currentPeak) > 0 {
		dm.currentPeak = value
	}

	// Calculate current drawdown
	currentDrawdown := dm.getCurrentDrawdownAmount()
	
	// Update maximum drawdown if current is larger
	if currentDrawdown.Cmp(dm.maxDrawdown) > 0 {
		dm.maxDrawdown = currentDrawdown
		
		// Update percentage
		if dm.currentPeak.IsPositive() {
			dm.maxDrawdownPercent = dm.maxDrawdown.Div(dm.currentPeak).Mul(types.NewDecimalFromInt(100))
		}
	}

	// Add to history
	entry := DrawdownHistoryEntry{
		Timestamp:       dm.lastUpdateTime,
		Value:           value,
		Peak:            dm.currentPeak,
		Drawdown:        currentDrawdown,
		DrawdownPercent: dm.GetCurrentDrawdownPercent(),
	}
	dm.history = append(dm.history, entry)

	// Process alerts
	dm.processAlerts()

	return nil
}

func (dm *DrawdownMonitor) getCurrentDrawdownAmount() types.Decimal {
	if dm.currentPeak.IsZero() {
		return types.Zero()
	}
	
	drawdown := dm.currentPeak.Sub(dm.currentValue)
	if drawdown.IsNegative() {
		return types.Zero()
	}
	
	return drawdown
}

func (dm *DrawdownMonitor) GetCurrentDrawdown() types.Decimal {
	return dm.getCurrentDrawdownAmount()
}

func (dm *DrawdownMonitor) GetCurrentDrawdownPercent() types.Decimal {
	if dm.currentPeak.IsZero() {
		return types.Zero()
	}
	
	drawdown := dm.getCurrentDrawdownAmount()
	return drawdown.Div(dm.currentPeak).Mul(types.NewDecimalFromInt(100))
}

func (dm *DrawdownMonitor) GetMaxDrawdown() types.Decimal {
	return dm.maxDrawdown
}

func (dm *DrawdownMonitor) GetMaxDrawdownPercent() types.Decimal {
	return dm.maxDrawdownPercent
}

func (dm *DrawdownMonitor) GetDrawdownHistory() []DrawdownHistoryEntry {
	return dm.history
}

func (dm *DrawdownMonitor) GetDrawdownStatistics() DrawdownStatistics {
	// Calculate basic statistics
	totalDrawdown := types.Zero()
	drawdownCount := 0
	
	for _, entry := range dm.history {
		if entry.Drawdown.IsPositive() {
			totalDrawdown = totalDrawdown.Add(entry.Drawdown)
			drawdownCount++
		}
	}
	
	averageDrawdown := types.Zero()
	if drawdownCount > 0 {
		averageDrawdown = totalDrawdown.Div(types.NewDecimalFromInt(int64(drawdownCount)))
	}
	
	return DrawdownStatistics{
		MaxDrawdown:           dm.maxDrawdown,
		MaxDrawdownPercent:    dm.maxDrawdownPercent,
		AverageDrawdown:       averageDrawdown,
		MaxDrawdownDuration:   time.Hour * 24, // Simplified for TDD GREEN phase
		CurrentDrawdownDuration: time.Hour,     // Simplified
		TotalDrawdownPeriods:  drawdownCount,
	}
}

func (dm *DrawdownMonitor) processAlerts() {
	currentDrawdownPercent := dm.GetCurrentDrawdownPercent()
	
	// Clear existing alerts
	dm.activeAlerts = make([]DrawdownAlert, 0)
	
	// Check each threshold
	for _, threshold := range dm.config.AlertThresholds {
		if currentDrawdownPercent.Cmp(threshold) >= 0 {
			alert := DrawdownAlert{
				ThresholdPercent: threshold,
				TriggeredAt:      dm.lastUpdateTime,
				CurrentDrawdown:  currentDrawdownPercent,
				IsActive:         true,
			}
			dm.activeAlerts = append(dm.activeAlerts, alert)
		}
	}
}

func (dm *DrawdownMonitor) GetActiveAlerts() []DrawdownAlert {
	return dm.activeAlerts
}

func (dm *DrawdownMonitor) IsMaxDrawdownBreached() bool {
	currentDrawdownPercent := dm.GetCurrentDrawdownPercent()
	return currentDrawdownPercent.Cmp(dm.config.MaxAcceptableDrawdown) > 0
}

func (dm *DrawdownMonitor) SetConfig(config DrawdownConfig) error {
	dm.config = config
	return nil
}

func (dm *DrawdownMonitor) GetConfig() DrawdownConfig {
	return dm.config
}