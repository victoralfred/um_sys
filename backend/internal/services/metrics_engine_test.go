package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsEngine(t *testing.T) {
	ctx := context.Background()

	t.Run("Define custom metric", func(t *testing.T) {
		engine := NewMetricsEngine()

		metric := &CustomMetricDefinition{
			ID:              uuid.New(),
			Name:            "conversion_rate",
			Description:     "Percentage of users who complete checkout",
			Formula:         "count(events.checkout_completed) / count(events.checkout_started) * 100",
			Unit:            "percentage",
			Category:        "e-commerce",
			AggregationType: "average",
			Dimensions:      []string{"product_category", "user_segment"},
			TimeWindow:      24 * time.Hour,
			CacheTTL:        5 * time.Minute,
		}

		err := engine.Define(metric)
		assert.NoError(t, err)

		// Verify metric was defined
		defined, err := engine.GetDefinition("conversion_rate")
		assert.NoError(t, err)
		assert.Equal(t, metric.Name, defined.Name)
		assert.Equal(t, metric.Formula, defined.Formula)
	})

	t.Run("Parse and validate formula", func(t *testing.T) {
		engine := NewMetricsEngine()

		testCases := []struct {
			name     string
			formula  string
			valid    bool
			errorMsg string
		}{
			{
				name:    "valid simple formula",
				formula: "count(events.page_view)",
				valid:   true,
			},
			{
				name:    "valid complex formula",
				formula: "(sum(metrics.revenue) / count(distinct users.id)) * 100",
				valid:   true,
			},
			{
				name:    "valid with conditions",
				formula: "count(events.click WHERE properties.button_id = 'buy-now')",
				valid:   true,
			},
			{
				name:     "invalid syntax",
				formula:  "count(events.click WHERE",
				valid:    false,
				errorMsg: "syntax error",
			},
			{
				name:     "undefined function",
				formula:  "undefined_func(events.click)",
				valid:    false,
				errorMsg: "undefined function",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := engine.ValidateFormula(tc.formula)
				if tc.valid {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			})
		}
	})

	t.Run("Calculate metric value", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define a simple count metric
		metric := &CustomMetricDefinition{
			ID:         uuid.New(),
			Name:       "daily_active_users",
			Formula:    "count(distinct events.user_id WHERE events.type = 'user_activity')",
			TimeWindow: 24 * time.Hour,
		}

		err := engine.Define(metric)
		require.NoError(t, err)

		// Calculate metric
		params := CalculationParams{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		}

		result, err := engine.Calculate(ctx, "daily_active_users", params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Value, 0.0)
		assert.Equal(t, "daily_active_users", result.MetricName)
	})

	t.Run("Calculate with dimensions", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define metric with dimensions
		metric := &CustomMetricDefinition{
			ID:         uuid.New(),
			Name:       "revenue_by_category",
			Formula:    "sum(events.properties.amount WHERE events.type = 'purchase')",
			Dimensions: []string{"product_category"},
			TimeWindow: 7 * 24 * time.Hour,
		}

		err := engine.Define(metric)
		require.NoError(t, err)

		// Calculate with dimension breakdown
		params := CalculationParams{
			StartTime: time.Now().Add(-7 * 24 * time.Hour),
			EndTime:   time.Now(),
			GroupBy:   []string{"product_category"},
		}

		results, err := engine.CalculateWithDimensions(ctx, "revenue_by_category", params)
		assert.NoError(t, err)
		assert.NotEmpty(t, results)

		for _, result := range results {
			assert.NotEmpty(t, result.Dimensions["product_category"])
			assert.GreaterOrEqual(t, result.Value, 0.0)
		}
	})

	t.Run("Scheduled metric calculation", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define metric
		metric := &CustomMetricDefinition{
			ID:         uuid.New(),
			Name:       "hourly_error_rate",
			Formula:    "count(events WHERE type = 'error') / count(events) * 100",
			TimeWindow: 1 * time.Hour,
		}

		err := engine.Define(metric)
		require.NoError(t, err)

		// Schedule calculation every hour
		schedule := CronSchedule{
			Expression: "0 * * * *", // Every hour
			Enabled:    true,
		}

		err = engine.Schedule("hourly_error_rate", schedule)
		assert.NoError(t, err)

		// Verify schedule was created
		scheduled, err := engine.GetSchedule("hourly_error_rate")
		assert.NoError(t, err)
		assert.Equal(t, schedule.Expression, scheduled.Expression)
		assert.True(t, scheduled.Enabled)
	})

	t.Run("Cached results", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define metric with cache TTL
		metric := &CustomMetricDefinition{
			ID:       uuid.New(),
			Name:     "expensive_calculation",
			Formula:  "count(events) + sum(events.value) / avg(events.value)",
			CacheTTL: 10 * time.Minute,
		}

		err := engine.Define(metric)
		require.NoError(t, err)

		params := CalculationParams{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		}

		// First calculation should compute
		start := time.Now()
		result1, err := engine.Calculate(ctx, "expensive_calculation", params)
		assert.NoError(t, err)
		duration1 := time.Since(start)

		// Second calculation should use cache (much faster)
		start = time.Now()
		result2, err := engine.Calculate(ctx, "expensive_calculation", params)
		assert.NoError(t, err)
		duration2 := time.Since(start)

		assert.Equal(t, result1.Value, result2.Value)
		// Cache should be faster (or at least not slower)
		assert.True(t, duration2 <= duration1 || result2.FromCache)
		assert.True(t, result2.FromCache)
	})

	t.Run("Complex aggregations", func(t *testing.T) {
		engine := NewMetricsEngine()

		testCases := []struct {
			name    string
			formula string
		}{
			{
				name:    "percentile calculation",
				formula: "percentile(events.properties.response_time, 95)",
			},
			{
				name:    "moving average",
				formula: "moving_avg(sum(events.properties.value), 7)",
			},
			{
				name:    "standard deviation",
				formula: "stddev(events.properties.amount)",
			},
			{
				name:    "correlation",
				formula: "corr(events.properties.price, events.properties.quantity)",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				metric := &CustomMetricDefinition{
					ID:      uuid.New(),
					Name:    tc.name,
					Formula: tc.formula,
				}

				err := engine.Define(metric)
				assert.NoError(t, err)

				result, err := engine.Calculate(ctx, tc.name, CalculationParams{
					StartTime: time.Now().Add(-24 * time.Hour),
					EndTime:   time.Now(),
				})
				assert.NoError(t, err)
				assert.NotNil(t, result)
			})
		}
	})

	t.Run("Metric dependencies", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define base metrics
		baseMetric1 := &CustomMetricDefinition{
			ID:      uuid.New(),
			Name:    "total_users",
			Formula: "count(distinct events.user_id)",
		}

		baseMetric2 := &CustomMetricDefinition{
			ID:      uuid.New(),
			Name:    "total_revenue",
			Formula: "sum(events.properties.amount WHERE type = 'purchase')",
		}

		err := engine.Define(baseMetric1)
		require.NoError(t, err)

		err = engine.Define(baseMetric2)
		require.NoError(t, err)

		// Define derived metric that depends on base metrics
		derivedMetric := &CustomMetricDefinition{
			ID:           uuid.New(),
			Name:         "arpu", // Average Revenue Per User
			Formula:      "metric.total_revenue / metric.total_users",
			Dependencies: []string{"total_users", "total_revenue"},
		}

		err = engine.Define(derivedMetric)
		assert.NoError(t, err)

		// Calculate derived metric (should automatically calculate dependencies)
		result, err := engine.Calculate(ctx, "arpu", CalculationParams{
			StartTime: time.Now().Add(-30 * 24 * time.Hour),
			EndTime:   time.Now(),
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, result.Value, 0.0)
	})

	t.Run("Real-time metric updates", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define real-time metric
		metric := &CustomMetricDefinition{
			ID:             uuid.New(),
			Name:           "current_active_users",
			Formula:        "count(distinct events.user_id)",
			RealTime:       true,
			UpdateInterval: 5 * time.Second,
		}

		err := engine.Define(metric)
		require.NoError(t, err)

		// Subscribe to real-time updates
		updates, err := engine.SubscribeToUpdates(ctx, "current_active_users")
		assert.NoError(t, err)
		assert.NotNil(t, updates)

		// Should receive updates
		select {
		case update := <-updates:
			assert.Equal(t, "current_active_users", update.MetricName)
			assert.GreaterOrEqual(t, update.Value, 0.0)
		case <-time.After(10 * time.Second):
			t.Fatal("No real-time update received")
		}
	})

	t.Run("Metric alerts", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define metric
		metric := &CustomMetricDefinition{
			ID:      uuid.New(),
			Name:    "error_rate",
			Formula: "count(events WHERE type = 'error') / count(events) * 100",
		}

		err := engine.Define(metric)
		require.NoError(t, err)

		// Set alert threshold
		alert := AlertConfiguration{
			MetricName: "error_rate",
			Condition:  "above",
			Threshold:  5.0, // Alert if error rate > 5%
			Window:     5 * time.Minute,
			Actions: []AlertAction{
				{Type: "email", Target: "ops@example.com"},
				{Type: "webhook", Target: "https://alerts.example.com"},
			},
		}

		err = engine.SetAlert(alert)
		assert.NoError(t, err)

		// Simulate metric exceeding threshold
		engine.SimulateValue("error_rate", 10.0)

		// Check if alert was triggered
		triggered, err := engine.IsAlertTriggered("error_rate")
		assert.NoError(t, err)
		assert.True(t, triggered)
	})
}
