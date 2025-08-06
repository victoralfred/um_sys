package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

func TestAnalyticsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	t.Run("End-to-end event processing pipeline", func(t *testing.T) {
		// Initialize services
		eventRegistry := NewEventTypeRegistry()
		metricsEngine := NewMetricsEngine()
		funnelService := NewFunnelService(nil, nil)
		cohortService := NewCohortService(nil, nil)

		// Step 1: Register custom event type
		eventSchema := &EventTypeSchema{
			Name:     "checkout_completed",
			Category: "e-commerce",
			RequiredFields: []FieldDefinition{
				{Name: "order_id", Type: "string", Required: true},
				{Name: "amount", Type: "number", Required: true},
				{Name: "items_count", Type: "integer", Required: true},
			},
			Validators: []Validator{
				&RangeValidator{Field: "amount", Min: 0.01, Max: 1000000},
			},
		}

		err := eventRegistry.Register("checkout_completed", eventSchema)
		require.NoError(t, err)

		// Step 2: Create and validate event
		event := &analytics.Event{
			ID:   uuid.New(),
			Type: "checkout_completed",
			Properties: map[string]interface{}{
				"order_id":    "ORD-12345",
				"amount":      299.99,
				"items_count": 3,
			},
			Timestamp: time.Now(),
		}

		err = eventRegistry.Validate(event)
		assert.NoError(t, err)

		// Step 3: Define custom metric based on event
		metric := &CustomMetricDefinition{
			ID:         uuid.New(),
			Name:       "average_order_value",
			Formula:    "sum(events.properties.amount) / count(events)",
			Category:   "e-commerce",
			CacheTTL:   5 * time.Minute,
			TimeWindow: 24 * time.Hour,
		}

		err = metricsEngine.Define(metric)
		require.NoError(t, err)

		// Step 4: Calculate metric
		metricResult, err := metricsEngine.Calculate(ctx, "average_order_value", CalculationParams{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		})
		assert.NoError(t, err)
		assert.Greater(t, metricResult.Value, 0.0)

		// Step 5: Define and analyze funnel
		funnel := &FunnelDefinition{
			ID:   uuid.New(),
			Name: "Purchase Funnel",
			Steps: []FunnelStep{
				{Name: "Product View", EventType: "product_view", Order: 1},
				{Name: "Add to Cart", EventType: "add_to_cart", Order: 2},
				{Name: "Checkout", EventType: "checkout_started", Order: 3},
				{Name: "Purchase", EventType: "checkout_completed", Order: 4},
			},
			TimeWindow: 30 * time.Minute,
		}

		err = funnelService.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		funnelAnalysis, err := funnelService.AnalyzeFunnel(ctx, funnel.ID, FunnelAnalysisParams{
			StartTime: time.Now().Add(-7 * 24 * time.Hour),
			EndTime:   time.Now(),
		})
		assert.NoError(t, err)
		assert.NotNil(t, funnelAnalysis)
		assert.GreaterOrEqual(t, funnelAnalysis.TotalUsers, int64(0))

		// Step 6: Define and analyze cohort
		cohort := &CohortDefinition{
			ID:   uuid.New(),
			Name: "High-Value Customers",
			Type: "behavioral",
			Criteria: CohortCriteria{
				EventType: "checkout_completed",
				Properties: map[string]interface{}{
					"amount": map[string]interface{}{
						"$gte": 100.0,
					},
				},
			},
		}

		err = cohortService.DefineCohort(ctx, cohort)
		require.NoError(t, err)

		cohortSize, err := cohortService.CalculateCohortSize(ctx, cohort.ID)
		assert.NoError(t, err)
		assert.Greater(t, cohortSize, int64(0))
	})

	t.Run("Cross-service metric dependencies", func(t *testing.T) {
		metricsEngine := NewMetricsEngine()
		funnelService := NewFunnelService(nil, nil)

		// Create funnel
		funnel := &FunnelDefinition{
			ID:   uuid.New(),
			Name: "Onboarding",
			Steps: []FunnelStep{
				{Name: "Sign Up", EventType: "signup", Order: 1},
				{Name: "Verify", EventType: "email_verified", Order: 2},
				{Name: "Complete", EventType: "profile_completed", Order: 3},
			},
		}

		err := funnelService.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		// Define metric based on funnel
		metric := &CustomMetricDefinition{
			ID:           uuid.New(),
			Name:         "onboarding_completion_rate",
			Formula:      "funnel.conversion_rate",
			Dependencies: []string{"funnel_analysis"},
		}

		err = metricsEngine.Define(metric)
		assert.NoError(t, err)
	})

	t.Run("Event schema evolution", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		// Register v1 schema
		v1Schema := &EventTypeSchema{
			Name:    "user_action",
			Version: 1,
			RequiredFields: []FieldDefinition{
				{Name: "action", Type: "string", Required: true},
			},
		}

		err := registry.RegisterVersion("user_action", 1, v1Schema)
		require.NoError(t, err)

		// Register v2 schema with additional fields
		v2Schema := &EventTypeSchema{
			Name:    "user_action",
			Version: 2,
			RequiredFields: []FieldDefinition{
				{Name: "action", Type: "string", Required: true},
				{Name: "context", Type: "object", Required: true},
			},
			MigrationRules: []MigrationRule{
				{
					FromVersion: 1,
					ToVersion:   2,
					Transform: func(props map[string]interface{}) map[string]interface{} {
						if _, ok := props["context"]; !ok {
							props["context"] = map[string]interface{}{
								"source": "legacy",
							}
						}
						return props
					},
				},
			},
		}

		err = registry.RegisterVersion("user_action", 2, v2Schema)
		require.NoError(t, err)

		// Test migration
		v1Event := &analytics.Event{
			ID:      uuid.New(),
			Type:    "user_action",
			Version: 1,
			Properties: map[string]interface{}{
				"action": "click",
			},
		}

		migratedEvent, err := registry.Migrate(v1Event, 2)
		assert.NoError(t, err)
		assert.Equal(t, 2, migratedEvent.Version)
		assert.NotNil(t, migratedEvent.Properties["context"])
	})

	t.Run("Cohort retention over time", func(t *testing.T) {
		cohortService := NewCohortService(nil, nil)

		// Create cohort
		cohort := &CohortDefinition{
			ID:   uuid.New(),
			Name: "January Users",
			Type: "temporal",
			Criteria: CohortCriteria{
				EventType: "user_registration",
				TimeRange: TimeRange{
					Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
				},
			},
		}

		err := cohortService.DefineCohort(ctx, cohort)
		require.NoError(t, err)

		// Analyze retention
		retention, err := cohortService.AnalyzeRetention(ctx, cohort.ID, RetentionParams{
			RetentionEvent: "user_activity",
			Intervals:      []string{"Day 1", "Day 7", "Day 30", "Day 60"},
			StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		})

		assert.NoError(t, err)
		assert.NotNil(t, retention)

		// Verify retention decreases over time
		var prevRate float64 = 100.0
		for _, interval := range retention.Intervals {
			assert.LessOrEqual(t, interval.RetentionRate, prevRate)
			prevRate = interval.RetentionRate
		}
	})

	t.Run("Funnel with custom event types", func(t *testing.T) {
		registry := NewEventTypeRegistry()
		funnelService := NewFunnelService(nil, nil)

		// Register custom events for funnel steps
		eventTypes := []string{"tutorial_start", "tutorial_step_1", "tutorial_step_2", "tutorial_complete"}
		for _, eventType := range eventTypes {
			schema := &EventTypeSchema{
				Name:     eventType,
				Category: "onboarding",
				RequiredFields: []FieldDefinition{
					{Name: "user_id", Type: "string", Required: true},
					{Name: "timestamp", Type: "string", Required: true},
				},
			}
			err := registry.Register(eventType, schema)
			require.NoError(t, err)
		}

		// Create funnel with custom events
		funnel := &FunnelDefinition{
			ID:   uuid.New(),
			Name: "Tutorial Completion",
			Steps: []FunnelStep{
				{Name: "Start", EventType: "tutorial_start", Order: 1},
				{Name: "Step 1", EventType: "tutorial_step_1", Order: 2},
				{Name: "Step 2", EventType: "tutorial_step_2", Order: 3},
				{Name: "Complete", EventType: "tutorial_complete", Order: 4},
			},
			TimeWindow: 1 * time.Hour,
		}

		err := funnelService.CreateFunnel(ctx, funnel)
		assert.NoError(t, err)

		// Validate events and analyze funnel
		for _, eventType := range eventTypes {
			event := &analytics.Event{
				ID:   uuid.New(),
				Type: analytics.EventType(eventType),
				Properties: map[string]interface{}{
					"user_id":   uuid.New().String(),
					"timestamp": time.Now().Format(time.RFC3339),
				},
			}
			err := registry.Validate(event)
			assert.NoError(t, err)
		}
	})

	t.Run("Metric alert triggering", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define metric
		metric := &CustomMetricDefinition{
			ID:      uuid.New(),
			Name:    "error_rate",
			Formula: "count(events WHERE type = 'error') / count(events) * 100",
		}

		err := engine.Define(metric)
		require.NoError(t, err)

		// Set alert
		alert := AlertConfiguration{
			MetricName: "error_rate",
			Condition:  "above",
			Threshold:  5.0,
			Window:     5 * time.Minute,
			Actions: []AlertAction{
				{Type: "email", Target: "ops@example.com"},
			},
		}

		err = engine.SetAlert(alert)
		assert.NoError(t, err)

		// Simulate high error rate
		engine.SimulateValue("error_rate", 10.0)

		// Check if alert triggered
		triggered, err := engine.IsAlertTriggered("error_rate")
		assert.NoError(t, err)
		assert.True(t, triggered)

		// Simulate normal error rate
		engine.SimulateValue("error_rate", 2.0)

		triggered, err = engine.IsAlertTriggered("error_rate")
		assert.NoError(t, err)
		assert.False(t, triggered)
	})

	t.Run("Complex funnel paths analysis", func(t *testing.T) {
		funnelService := NewFunnelService(nil, nil)

		// Create multi-path funnel
		funnel := &FunnelDefinition{
			ID:   uuid.New(),
			Name: "Flexible Purchase Path",
			Steps: []FunnelStep{
				{Name: "Entry", EventTypes: []string{"direct_visit", "ad_click", "organic_search"}, Order: 1},
				{Name: "Browse", EventTypes: []string{"category_view", "search", "recommendation_click"}, Order: 2},
				{Name: "Engage", EventType: "product_view", Order: 3},
				{Name: "Convert", EventType: "purchase", Order: 4},
			},
			AllowSkipSteps: true,
			TimeWindow:     2 * time.Hour,
		}

		err := funnelService.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		// Analyze different paths
		paths, err := funnelService.AnalyzePaths(ctx, funnel.ID, FunnelAnalysisParams{
			StartTime: time.Now().Add(-30 * 24 * time.Hour),
			EndTime:   time.Now(),
		})

		assert.NoError(t, err)
		assert.NotNil(t, paths)
		assert.NotEmpty(t, paths.Paths)

		// Verify path variations
		uniquePaths := make(map[string]bool)
		for _, path := range paths.Paths {
			pathKey := ""
			for _, step := range path.Steps {
				pathKey += step + "->"
			}
			uniquePaths[pathKey] = true
		}
		assert.Greater(t, len(uniquePaths), 1, "Should have multiple unique paths")
	})

	t.Run("Cohort lifecycle progression", func(t *testing.T) {
		cohortService := NewCohortService(nil, nil)

		// Create cohort
		cohort := &CohortDefinition{
			ID:   uuid.New(),
			Name: "Q1 Users",
			Type: "temporal",
		}

		err := cohortService.DefineCohort(ctx, cohort)
		require.NoError(t, err)

		// Analyze lifecycle stages
		lifecycle, err := cohortService.AnalyzeLifecycle(ctx, cohort.ID, LifecycleParams{
			Stages: []string{"new", "active", "engaged", "at_risk", "churned"},
			Window: 30 * 24 * time.Hour,
		})

		assert.NoError(t, err)
		assert.NotNil(t, lifecycle)
		assert.Len(t, lifecycle.StageDistribution, 5)

		// Verify stage distribution sums to 100%
		var totalPercentage float64
		for _, stage := range lifecycle.StageDistribution {
			totalPercentage += stage.Percentage
			assert.NotEmpty(t, stage.Name)
			assert.GreaterOrEqual(t, stage.UserCount, int64(0))
		}
		assert.InDelta(t, 100.0, totalPercentage, 0.01)
	})
}

func TestAnalyticsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test")
	}

	ctx := context.Background()

	t.Run("High-volume event processing", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		// Register event type
		schema := &EventTypeSchema{
			Name:     "api_request",
			Category: "system",
			RequiredFields: []FieldDefinition{
				{Name: "endpoint", Type: "string", Required: true},
				{Name: "response_time", Type: "number", Required: true},
			},
		}

		err := registry.Register("api_request", schema)
		require.NoError(t, err)

		// Process many events
		start := time.Now()
		eventCount := 10000

		for i := 0; i < eventCount; i++ {
			event := &analytics.Event{
				ID:   uuid.New(),
				Type: "api_request",
				Properties: map[string]interface{}{
					"endpoint":      "/api/users",
					"response_time": float64(50 + i%100),
				},
			}

			err := registry.Validate(event)
			assert.NoError(t, err)
		}

		elapsed := time.Since(start)
		eventsPerSecond := float64(eventCount) / elapsed.Seconds()

		t.Logf("Processed %d events in %v (%.0f events/sec)", eventCount, elapsed, eventsPerSecond)
		assert.Greater(t, eventsPerSecond, float64(1000), "Should process at least 1000 events/sec")
	})

	t.Run("Concurrent metric calculations", func(t *testing.T) {
		engine := NewMetricsEngine()

		// Define multiple metrics
		metricNames := []string{}
		for i := 0; i < 10; i++ {
			metric := &CustomMetricDefinition{
				ID:       uuid.New(),
				Name:     fmt.Sprintf("metric_%d", i),
				Formula:  "count(events) * 1.5",
				CacheTTL: 1 * time.Minute,
			}
			err := engine.Define(metric)
			require.NoError(t, err)
			metricNames = append(metricNames, metric.Name)
		}

		// Calculate metrics concurrently
		start := time.Now()
		results := make(chan *MetricResult, len(metricNames))
		errors := make(chan error, len(metricNames))

		for _, name := range metricNames {
			go func(metricName string) {
				result, err := engine.Calculate(ctx, metricName, CalculationParams{
					StartTime: time.Now().Add(-24 * time.Hour),
					EndTime:   time.Now(),
				})
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(name)
		}

		// Collect results
		for i := 0; i < len(metricNames); i++ {
			select {
			case err := <-errors:
				t.Fatalf("Error calculating metric: %v", err)
			case result := <-results:
				assert.NotNil(t, result)
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for metric calculation")
			}
		}

		elapsed := time.Since(start)
		t.Logf("Calculated %d metrics concurrently in %v", len(metricNames), elapsed)
		assert.Less(t, elapsed, 2*time.Second, "Concurrent calculations should complete quickly")
	})
}

func TestAnalyticsDataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping data integrity test")
	}

	ctx := context.Background()

	t.Run("Schema validation prevents invalid data", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		// Register strict schema
		schema := &EventTypeSchema{
			Name:     "transaction",
			Category: "financial",
			RequiredFields: []FieldDefinition{
				{Name: "transaction_id", Type: "string", Required: true},
				{Name: "amount", Type: "number", Required: true},
				{Name: "currency", Type: "string", Required: true},
			},
			Validators: []Validator{
				&RangeValidator{Field: "amount", Min: 0, Max: 1000000},
				&RegexValidator{Field: "currency", Pattern: "^[A-Z]{3}$"},
			},
		}

		err := registry.Register("transaction", schema)
		require.NoError(t, err)

		// Test various invalid events
		invalidEvents := []struct {
			name  string
			event *analytics.Event
			error string
		}{
			{
				name: "missing required field",
				event: &analytics.Event{
					ID:   uuid.New(),
					Type: "transaction",
					Properties: map[string]interface{}{
						"transaction_id": "TXN-123",
						"amount":         100.0,
						// missing currency
					},
				},
				error: "missing required field: currency",
			},
			{
				name: "invalid type",
				event: &analytics.Event{
					ID:   uuid.New(),
					Type: "transaction",
					Properties: map[string]interface{}{
						"transaction_id": "TXN-123",
						"amount":         "not a number",
						"currency":       "USD",
					},
				},
				error: "invalid type for field amount",
			},
			{
				name: "validation failure",
				event: &analytics.Event{
					ID:   uuid.New(),
					Type: "transaction",
					Properties: map[string]interface{}{
						"transaction_id": "TXN-123",
						"amount":         -100.0,
						"currency":       "USD",
					},
				},
				error: "value out of range",
			},
			{
				name: "regex validation failure",
				event: &analytics.Event{
					ID:   uuid.New(),
					Type: "transaction",
					Properties: map[string]interface{}{
						"transaction_id": "TXN-123",
						"amount":         100.0,
						"currency":       "US",
					},
				},
				error: "invalid format",
			},
		}

		for _, tc := range invalidEvents {
			t.Run(tc.name, func(t *testing.T) {
				err := registry.Validate(tc.event)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.error)
			})
		}
	})

	t.Run("Cohort membership consistency", func(t *testing.T) {
		cohortService := NewCohortService(nil, nil)

		// Create overlapping cohorts
		cohort1 := &CohortDefinition{
			ID:   uuid.New(),
			Name: "All Users",
			Type: "behavioral",
		}

		cohort2 := &CohortDefinition{
			ID:   uuid.New(),
			Name: "Active Users",
			Type: "behavioral",
			Criteria: CohortCriteria{
				EventType: "user_activity",
			},
		}

		err := cohortService.DefineCohort(ctx, cohort1)
		require.NoError(t, err)

		err = cohortService.DefineCohort(ctx, cohort2)
		require.NoError(t, err)

		// Verify cohort sizes
		size1, err := cohortService.CalculateCohortSize(ctx, cohort1.ID)
		assert.NoError(t, err)

		size2, err := cohortService.CalculateCohortSize(ctx, cohort2.ID)
		assert.NoError(t, err)

		// Active users should be subset of all users
		assert.GreaterOrEqual(t, size1, size2)
	})
}
