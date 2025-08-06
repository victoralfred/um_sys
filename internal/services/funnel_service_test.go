package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunnelService(t *testing.T) {
	ctx := context.Background()

	t.Run("Create funnel", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		funnel := &FunnelDefinition{
			ID:          uuid.New(),
			Name:        "Checkout Funnel",
			Description: "Track user conversion through checkout process",
			Steps: []FunnelStep{
				{
					Name:      "View Product",
					EventType: "product_view",
					Order:     1,
				},
				{
					Name:      "Add to Cart",
					EventType: "add_to_cart",
					Order:     2,
				},
				{
					Name:      "Start Checkout",
					EventType: "checkout_started",
					Order:     3,
				},
				{
					Name:      "Complete Purchase",
					EventType: "checkout_completed",
					Order:     4,
				},
			},
			TimeWindow: 30 * time.Minute, // User has 30 min to complete funnel
		}

		err := service.CreateFunnel(ctx, funnel)
		assert.NoError(t, err)

		// Verify funnel was created
		retrieved, err := service.GetFunnel(ctx, funnel.ID)
		assert.NoError(t, err)
		assert.Equal(t, funnel.Name, retrieved.Name)
		assert.Len(t, retrieved.Steps, 4)
	})

	t.Run("Analyze funnel conversion", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		// Create funnel
		funnelID := uuid.New()
		funnel := &FunnelDefinition{
			ID:   funnelID,
			Name: "Onboarding Funnel",
			Steps: []FunnelStep{
				{Name: "Sign Up", EventType: "user_signup", Order: 1},
				{Name: "Verify Email", EventType: "email_verified", Order: 2},
				{Name: "Complete Profile", EventType: "profile_completed", Order: 3},
				{Name: "First Action", EventType: "first_action", Order: 4},
			},
			TimeWindow: 24 * time.Hour,
		}

		err := service.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		// Analyze funnel
		params := FunnelAnalysisParams{
			StartTime: time.Now().Add(-7 * 24 * time.Hour),
			EndTime:   time.Now(),
			Segments:  []string{"user_type", "source"},
		}

		analysis, err := service.AnalyzeFunnel(ctx, funnelID, params)
		assert.NoError(t, err)
		assert.NotNil(t, analysis)

		// Verify analysis results
		assert.Equal(t, funnelID, analysis.FunnelID)
		assert.GreaterOrEqual(t, analysis.TotalUsers, int64(0))
		assert.GreaterOrEqual(t, analysis.CompletedUsers, int64(0))
		assert.GreaterOrEqual(t, analysis.ConversionRate, 0.0)
		assert.LessOrEqual(t, analysis.ConversionRate, 100.0)

		// Check step conversions
		assert.Len(t, analysis.StepConversions, 4)
		for i, step := range analysis.StepConversions {
			assert.NotEmpty(t, step.StepName)
			assert.GreaterOrEqual(t, step.UsersReached, int64(0))
			assert.GreaterOrEqual(t, step.ConversionRate, 0.0)

			// Each step should have fewer users than the previous
			if i > 0 {
				assert.LessOrEqual(t, step.UsersReached, analysis.StepConversions[i-1].UsersReached)
			}
		}
	})

	t.Run("Identify drop-off points", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		funnelID := uuid.New()
		analysis := service.AnalyzeFunnel(ctx, funnelID, FunnelAnalysisParams{
			StartTime: time.Now().Add(-30 * 24 * time.Hour),
			EndTime:   time.Now(),
		})

		dropoffs, err := service.GetDropoffPoints(ctx, funnelID)
		assert.NoError(t, err)
		assert.NotEmpty(t, dropoffs)

		for _, dropoff := range dropoffs {
			assert.NotEmpty(t, dropoff.FromStep)
			assert.NotEmpty(t, dropoff.ToStep)
			assert.GreaterOrEqual(t, dropoff.DropoffRate, 0.0)
			assert.LessOrEqual(t, dropoff.DropoffRate, 100.0)
			assert.GreaterOrEqual(t, dropoff.UserCount, int64(0))
		}

		// Find biggest drop-off
		biggestDropoff := service.GetBiggestDropoff(ctx, funnelID)
		assert.NotNil(t, biggestDropoff)
		assert.NotEmpty(t, biggestDropoff.FromStep)
	})

	t.Run("Multi-path funnel analysis", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		// Create funnel with alternative paths
		funnel := &FunnelDefinition{
			ID:   uuid.New(),
			Name: "Flexible Checkout",
			Steps: []FunnelStep{
				{Name: "Landing", EventType: "page_view", Order: 1},
				{Name: "Product or Search", EventTypes: []string{"product_view", "search"}, Order: 2},
				{Name: "Add to Cart", EventType: "add_to_cart", Order: 3},
				{Name: "Checkout", EventType: "checkout_started", Order: 4},
				{Name: "Purchase", EventType: "purchase_completed", Order: 5},
			},
			AllowSkipSteps: true, // Users can skip certain steps
			TimeWindow:     1 * time.Hour,
		}

		err := service.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		// Analyze paths taken
		pathAnalysis, err := service.AnalyzePaths(ctx, funnel.ID, FunnelAnalysisParams{
			StartTime: time.Now().Add(-7 * 24 * time.Hour),
			EndTime:   time.Now(),
		})

		assert.NoError(t, err)
		assert.NotNil(t, pathAnalysis)
		assert.NotEmpty(t, pathAnalysis.Paths)

		// Check different paths
		for _, path := range pathAnalysis.Paths {
			assert.NotEmpty(t, path.Steps)
			assert.GreaterOrEqual(t, path.UserCount, int64(0))
			assert.GreaterOrEqual(t, path.ConversionRate, 0.0)
			assert.GreaterOrEqual(t, path.AverageTime, time.Duration(0))
		}
	})

	t.Run("Funnel comparison", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		// Create two funnels to compare
		funnel1 := &FunnelDefinition{
			ID:   uuid.New(),
			Name: "Old Checkout Flow",
			Steps: []FunnelStep{
				{Name: "Cart", EventType: "view_cart", Order: 1},
				{Name: "Shipping", EventType: "enter_shipping", Order: 2},
				{Name: "Payment", EventType: "enter_payment", Order: 3},
				{Name: "Confirm", EventType: "confirm_order", Order: 4},
			},
		}

		funnel2 := &FunnelDefinition{
			ID:   uuid.New(),
			Name: "New Checkout Flow",
			Steps: []FunnelStep{
				{Name: "Cart", EventType: "view_cart", Order: 1},
				{Name: "Quick Checkout", EventType: "quick_checkout", Order: 2},
				{Name: "Confirm", EventType: "confirm_order", Order: 3},
			},
		}

		err := service.CreateFunnel(ctx, funnel1)
		require.NoError(t, err)

		err = service.CreateFunnel(ctx, funnel2)
		require.NoError(t, err)

		// Compare funnels
		comparison, err := service.CompareFunnels(ctx, funnel1.ID, funnel2.ID, FunnelAnalysisParams{
			StartTime: time.Now().Add(-30 * 24 * time.Hour),
			EndTime:   time.Now(),
		})

		assert.NoError(t, err)
		assert.NotNil(t, comparison)
		assert.NotEqual(t, comparison.Funnel1.ConversionRate, comparison.Funnel2.ConversionRate)
		assert.NotZero(t, comparison.ConversionRateDiff)
		assert.NotEmpty(t, comparison.Winner)
	})

	t.Run("Funnel with properties filter", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		funnelID := uuid.New()

		// Analyze funnel with property filters
		params := FunnelAnalysisParams{
			StartTime: time.Now().Add(-7 * 24 * time.Hour),
			EndTime:   time.Now(),
			Filters: map[string]interface{}{
				"device_type": "mobile",
				"country":     "US",
				"user_type":   "premium",
			},
		}

		analysis, err := service.AnalyzeFunnel(ctx, funnelID, params)
		assert.NoError(t, err)
		assert.NotNil(t, analysis)

		// Results should be filtered
		assert.Contains(t, analysis.AppliedFilters, "device_type")
		assert.Contains(t, analysis.AppliedFilters, "country")
		assert.Contains(t, analysis.AppliedFilters, "user_type")
	})

	t.Run("Time to convert analysis", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		funnelID := uuid.New()

		// Get time to convert for each step
		timeAnalysis, err := service.AnalyzeTimeToConvert(ctx, funnelID, FunnelAnalysisParams{
			StartTime: time.Now().Add(-30 * 24 * time.Hour),
			EndTime:   time.Now(),
		})

		assert.NoError(t, err)
		assert.NotNil(t, timeAnalysis)

		for _, stepTime := range timeAnalysis.StepTimes {
			assert.NotEmpty(t, stepTime.StepName)
			assert.GreaterOrEqual(t, stepTime.MedianTime, time.Duration(0))
			assert.GreaterOrEqual(t, stepTime.AverageTime, time.Duration(0))
			assert.GreaterOrEqual(t, stepTime.P95Time, stepTime.MedianTime)
		}

		assert.GreaterOrEqual(t, timeAnalysis.TotalMedianTime, time.Duration(0))
		assert.GreaterOrEqual(t, timeAnalysis.TotalAverageTime, time.Duration(0))
	})

	t.Run("Funnel attribution", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		funnelID := uuid.New()

		// Analyze attribution (what led users to enter funnel)
		attribution, err := service.AnalyzeAttribution(ctx, funnelID, AttributionParams{
			Model:     "last_touch", // or "first_touch", "linear", "time_decay"
			Lookback:  7 * 24 * time.Hour,
			StartTime: time.Now().Add(-30 * 24 * time.Hour),
			EndTime:   time.Now(),
		})

		assert.NoError(t, err)
		assert.NotNil(t, attribution)
		assert.NotEmpty(t, attribution.Channels)

		totalAttribution := 0.0
		for _, channel := range attribution.Channels {
			assert.NotEmpty(t, channel.Name)
			assert.GreaterOrEqual(t, channel.Users, int64(0))
			assert.GreaterOrEqual(t, channel.Attribution, 0.0)
			totalAttribution += channel.Attribution
		}
		assert.InDelta(t, 100.0, totalAttribution, 0.01) // Should sum to 100%
	})

	t.Run("Export funnel data", func(t *testing.T) {
		service := NewFunnelService(nil, nil)

		funnelID := uuid.New()

		// Export funnel data as CSV
		csvData, err := service.ExportFunnelData(ctx, funnelID, ExportParams{
			Format:           "csv",
			StartTime:        time.Now().Add(-30 * 24 * time.Hour),
			EndTime:          time.Now(),
			IncludeRawEvents: true,
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, csvData)
		assert.Contains(t, string(csvData), "user_id")
		assert.Contains(t, string(csvData), "step_name")
		assert.Contains(t, string(csvData), "timestamp")

		// Export as JSON
		jsonData, err := service.ExportFunnelData(ctx, funnelID, ExportParams{
			Format: "json",
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)
		assert.Contains(t, string(jsonData), "funnel_id")
		assert.Contains(t, string(jsonData), "analysis")
	})
}
