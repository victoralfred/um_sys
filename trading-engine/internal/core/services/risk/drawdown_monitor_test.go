package risk

import (
	"testing"

	"github.com/trading-engine/pkg/types"
)

// TDD: Step 1 - RED - Write failing tests for Drawdown Monitor

func TestDrawdownMonitorPeakTracking(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	monitor := NewDrawdownMonitor()
	if monitor == nil {
		t.Fatal("Expected drawdown monitor to be created")
	}

	// Test initial state
	initialPeak := monitor.GetCurrentPeak()
	expectedInitialPeak := types.Zero()

	if initialPeak.Cmp(expectedInitialPeak) != 0 {
		t.Errorf("Expected initial peak %s, got %s",
			expectedInitialPeak.String(), initialPeak.String())
	}

	// Test updating to new peak
	newValue := types.NewDecimalFromFloat(100000.0)
	err := monitor.UpdateValue(newValue)
	if err != nil {
		t.Fatalf("Failed to update value: %v", err)
	}

	updatedPeak := monitor.GetCurrentPeak()
	if updatedPeak.Cmp(newValue) != 0 {
		t.Errorf("Expected peak to be updated to %s, got %s",
			newValue.String(), updatedPeak.String())
	}

	// Test value higher than current peak
	higherValue := types.NewDecimalFromFloat(120000.0)
	err = monitor.UpdateValue(higherValue)
	if err != nil {
		t.Fatalf("Failed to update to higher value: %v", err)
	}

	newPeak := monitor.GetCurrentPeak()
	if newPeak.Cmp(higherValue) != 0 {
		t.Errorf("Expected peak to be updated to %s, got %s",
			higherValue.String(), newPeak.String())
	}

	// Test value lower than current peak (shouldn't update peak)
	lowerValue := types.NewDecimalFromFloat(110000.0)
	err = monitor.UpdateValue(lowerValue)
	if err != nil {
		t.Fatalf("Failed to update to lower value: %v", err)
	}

	peakAfterLower := monitor.GetCurrentPeak()
	if peakAfterLower.Cmp(higherValue) != 0 {
		t.Errorf("Peak should remain %s after lower value, got %s",
			higherValue.String(), peakAfterLower.String())
	}
}

func TestDrawdownCalculation(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	monitor := NewDrawdownMonitor()

	// Set up initial peak
	initialValue := types.NewDecimalFromFloat(100000.0)
	monitor.UpdateValue(initialValue)

	// Test current drawdown at peak
	currentDrawdown := monitor.GetCurrentDrawdown()
	expectedDrawdown := types.Zero()

	if currentDrawdown.Cmp(expectedDrawdown) != 0 {
		t.Errorf("Expected zero drawdown at peak, got %s", currentDrawdown.String())
	}

	// Test drawdown calculation when value drops
	lowerValue := types.NewDecimalFromFloat(90000.0) // 10% drawdown
	monitor.UpdateValue(lowerValue)

	drawdownAfterDrop := monitor.GetCurrentDrawdown()
	expectedDrawdownAmount := types.NewDecimalFromFloat(10000.0) // 100k - 90k

	if drawdownAfterDrop.Cmp(expectedDrawdownAmount) != 0 {
		t.Errorf("Expected drawdown %s, got %s",
			expectedDrawdownAmount.String(), drawdownAfterDrop.String())
	}

	// Test drawdown percentage calculation
	drawdownPercent := monitor.GetCurrentDrawdownPercent()
	expectedPercent := types.NewDecimalFromFloat(10.0) // 10%

	if drawdownPercent.Cmp(expectedPercent) != 0 {
		t.Errorf("Expected drawdown percentage %s%%, got %s%%",
			expectedPercent.String(), drawdownPercent.String())
	}

	// Test maximum drawdown tracking
	evenLowerValue := types.NewDecimalFromFloat(75000.0) // 25% drawdown
	monitor.UpdateValue(evenLowerValue)

	maxDrawdown := monitor.GetMaxDrawdown()
	expectedMaxDrawdown := types.NewDecimalFromFloat(25000.0) // 100k - 75k

	if maxDrawdown.Cmp(expectedMaxDrawdown) != 0 {
		t.Errorf("Expected max drawdown %s, got %s",
			expectedMaxDrawdown.String(), maxDrawdown.String())
	}

	maxDrawdownPercent := monitor.GetMaxDrawdownPercent()
	expectedMaxPercent := types.NewDecimalFromFloat(25.0)

	if maxDrawdownPercent.Cmp(expectedMaxPercent) != 0 {
		t.Errorf("Expected max drawdown percentage %s%%, got %s%%",
			expectedMaxPercent.String(), maxDrawdownPercent.String())
	}
}

func TestDrawdownRecovery(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	monitor := NewDrawdownMonitor()

	// Set up initial peak and drawdown
	peak := types.NewDecimalFromFloat(100000.0)
	monitor.UpdateValue(peak)

	drawdownValue := types.NewDecimalFromFloat(80000.0) // 20% drawdown
	monitor.UpdateValue(drawdownValue)

	// Verify drawdown exists
	if monitor.GetCurrentDrawdown().IsZero() {
		t.Fatal("Expected drawdown to be recorded")
	}

	// Test recovery to original peak
	monitor.UpdateValue(peak)

	// After recovery, current drawdown should be zero
	recoveryDrawdown := monitor.GetCurrentDrawdown()
	if !recoveryDrawdown.IsZero() {
		t.Errorf("Expected zero drawdown after recovery, got %s",
			recoveryDrawdown.String())
	}

	// Max drawdown should still be preserved
	maxDrawdown := monitor.GetMaxDrawdown()
	expectedMaxDrawdown := types.NewDecimalFromFloat(20000.0)

	if maxDrawdown.Cmp(expectedMaxDrawdown) != 0 {
		t.Errorf("Expected preserved max drawdown %s, got %s",
			expectedMaxDrawdown.String(), maxDrawdown.String())
	}

	// Test recovery beyond original peak (new peak)
	newPeak := types.NewDecimalFromFloat(110000.0)
	monitor.UpdateValue(newPeak)

	newCurrentPeak := monitor.GetCurrentPeak()
	if newCurrentPeak.Cmp(newPeak) != 0 {
		t.Errorf("Expected new peak %s, got %s",
			newPeak.String(), newCurrentPeak.String())
	}
}

func TestDrawdownHistory(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	monitor := NewDrawdownMonitor()

	// Set up scenario with multiple drawdown periods
	values := []float64{100000, 90000, 85000, 95000, 105000, 80000, 75000, 110000}

	for _, value := range values {
		err := monitor.UpdateValue(types.NewDecimalFromFloat(value))
		if err != nil {
			t.Fatalf("Failed to update value %f: %v", value, err)
		}
	}

	// Test getting drawdown history
	history := monitor.GetDrawdownHistory()
	if len(history) == 0 {
		t.Error("Expected drawdown history to contain entries")
	}

	// Test getting statistics
	stats := monitor.GetDrawdownStatistics()

	// Validate statistics contain expected fields
	if stats.MaxDrawdown.IsZero() {
		t.Error("Expected max drawdown to be recorded in statistics")
	}

	if stats.MaxDrawdownPercent.IsZero() {
		t.Error("Expected max drawdown percentage to be recorded")
	}

	if stats.AverageDrawdown.IsNegative() {
		t.Error("Expected average drawdown to be non-negative")
	}

	// Test drawdown duration tracking
	if stats.MaxDrawdownDuration <= 0 {
		t.Error("Expected max drawdown duration to be positive")
	}
}

func TestDrawdownAlerts(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	monitor := NewDrawdownMonitor()

	// Configure alert thresholds
	config := DrawdownConfig{
		AlertThresholds: []types.Decimal{
			types.NewDecimalFromFloat(5.0),  // 5% alert
			types.NewDecimalFromFloat(10.0), // 10% alert
			types.NewDecimalFromFloat(15.0), // 15% alert
		},
		MaxAcceptableDrawdown: types.NewDecimalFromFloat(20.0), // 20% max
	}

	err := monitor.SetConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Set up initial value
	monitor.UpdateValue(types.NewDecimalFromFloat(100000.0))

	// Test alert triggering
	monitor.UpdateValue(types.NewDecimalFromFloat(94000.0)) // 6% drawdown - should trigger 5% alert

	alerts := monitor.GetActiveAlerts()
	if len(alerts) == 0 {
		t.Error("Expected alert to be triggered at 6% drawdown")
	}

	// Check that 5% threshold was triggered but not 10%
	found5Percent := false
	found10Percent := false

	for _, alert := range alerts {
		if alert.ThresholdPercent.Cmp(types.NewDecimalFromFloat(5.0)) == 0 {
			found5Percent = true
		}
		if alert.ThresholdPercent.Cmp(types.NewDecimalFromFloat(10.0)) == 0 {
			found10Percent = true
		}
	}

	if !found5Percent {
		t.Error("Expected 5% drawdown alert to be active")
	}

	if found10Percent {
		t.Error("10% drawdown alert should not be active yet")
	}

	// Test maximum acceptable drawdown breach
	monitor.UpdateValue(types.NewDecimalFromFloat(75000.0)) // 25% drawdown

	if !monitor.IsMaxDrawdownBreached() {
		t.Error("Expected max acceptable drawdown to be breached at 25%")
	}
}

// Helper functions that will need to be implemented

func TestDrawdownMonitorConfiguration(t *testing.T) {
	// This test will FAIL initially - that's expected in TDD RED phase
	monitor := NewDrawdownMonitor()

	// Test default configuration
	defaultConfig := monitor.GetConfig()
	if len(defaultConfig.AlertThresholds) == 0 {
		t.Error("Expected default alert thresholds to be configured")
	}

	// Test custom configuration
	customConfig := DrawdownConfig{
		EnableRealTimeAlerts:  true,
		AlertThresholds:       []types.Decimal{types.NewDecimalFromFloat(8.0)},
		MaxAcceptableDrawdown: types.NewDecimalFromFloat(15.0),
		HistoryRetentionDays:  90,
	}

	err := monitor.SetConfig(customConfig)
	if err != nil {
		t.Fatalf("Failed to set custom config: %v", err)
	}

	// Verify configuration was applied
	appliedConfig := monitor.GetConfig()
	if !appliedConfig.EnableRealTimeAlerts {
		t.Error("Expected real-time alerts to be enabled")
	}

	if appliedConfig.HistoryRetentionDays != 90 {
		t.Errorf("Expected history retention 90 days, got %d", appliedConfig.HistoryRetentionDays)
	}
}

// TDD RED phase - Types and functions are now in drawdown_monitor.go
