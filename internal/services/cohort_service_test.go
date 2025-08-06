package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCohortService(t *testing.T) {
	ctx := context.Background()

	t.Run("Define cohort", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		cohort := &CohortDefinition{
			ID:          uuid.New(),
			Name:        "January 2024 Signups",
			Description: "Users who signed up in January 2024",
			Type:        "behavioral", // behavioral, demographic, technographic
			Criteria: CohortCriteria{
				EventType: "user_registration",
				TimeRange: TimeRange{
					Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
				},
				Properties: map[string]interface{}{
					"source": "organic",
				},
			},
		}

		err := service.DefineCohort(ctx, cohort)
		assert.NoError(t, err)

		// Verify cohort was created
		retrieved, err := service.GetCohort(ctx, cohort.ID)
		assert.NoError(t, err)
		assert.Equal(t, cohort.Name, retrieved.Name)
		assert.Equal(t, cohort.Type, retrieved.Type)
	})

	t.Run("Calculate cohort size", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		cohortID := uuid.New()
		cohort := &CohortDefinition{
			ID:   cohortID,
			Name: "High-Value Users",
			Type: "behavioral",
			Criteria: CohortCriteria{
				EventType: "purchase_completed",
				Properties: map[string]interface{}{
					"total_amount": map[string]interface{}{
						"$gte": 100.0,
					},
				},
			},
		}

		err := service.DefineCohort(ctx, cohort)
		require.NoError(t, err)

		// Calculate cohort size
		size, err := service.CalculateCohortSize(ctx, cohortID)
		assert.NoError(t, err)
		assert.Greater(t, size, int64(0))
	})

	t.Run("Cohort retention analysis", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		cohortID := uuid.New()

		// Analyze retention over time
		retention, err := service.AnalyzeRetention(ctx, cohortID, RetentionParams{
			RetentionEvent: "user_activity",
			Intervals:      []string{"Day 1", "Day 7", "Day 14", "Day 30"},
			StartDate:      time.Now().Add(-30 * 24 * time.Hour),
			EndDate:        time.Now(),
		})

		assert.NoError(t, err)
		assert.NotNil(t, retention)
		assert.NotEmpty(t, retention.Intervals)

		// Check retention rates decrease over time
		for i := 1; i < len(retention.Intervals); i++ {
			assert.LessOrEqual(t, retention.Intervals[i].RetentionRate, retention.Intervals[i-1].RetentionRate)
		}
	})

	t.Run("Cohort comparison", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		// Create two cohorts
		cohort1 := &CohortDefinition{
			ID:   uuid.New(),
			Name: "Q1 2024 Users",
			Type: "temporal",
			Criteria: CohortCriteria{
				EventType: "user_registration",
				TimeRange: TimeRange{
					Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC),
				},
			},
		}

		cohort2 := &CohortDefinition{
			ID:   uuid.New(),
			Name: "Q2 2024 Users",
			Type: "temporal",
			Criteria: CohortCriteria{
				EventType: "user_registration",
				TimeRange: TimeRange{
					Start: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
				},
			},
		}

		err := service.DefineCohort(ctx, cohort1)
		require.NoError(t, err)

		err = service.DefineCohort(ctx, cohort2)
		require.NoError(t, err)

		// Compare cohorts
		comparison, err := service.CompareCohorts(ctx, cohort1.ID, cohort2.ID, ComparisonParams{
			Metrics: []string{"retention_rate", "revenue", "engagement_score"},
			Period:  "30_days",
		})

		assert.NoError(t, err)
		assert.NotNil(t, comparison)
		assert.NotEmpty(t, comparison.MetricComparisons)

		for _, metric := range comparison.MetricComparisons {
			assert.NotEmpty(t, metric.MetricName)
			assert.GreaterOrEqual(t, metric.Cohort1Value, 0.0)
			assert.GreaterOrEqual(t, metric.Cohort2Value, 0.0)
			assert.NotZero(t, metric.Difference)
		}
	})

	t.Run("Behavioral cohort segmentation", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		// Define behavioral cohort based on multiple events
		cohort := &CohortDefinition{
			ID:   uuid.New(),
			Name: "Power Users",
			Type: "behavioral",
			Criteria: CohortCriteria{
				MultipleEvents: []EventCriteria{
					{
						EventType: "feature_usage",
						MinCount:  10,
						Properties: map[string]interface{}{
							"feature": "advanced_analytics",
						},
					},
					{
						EventType: "user_activity",
						MinCount:  20,
					},
				},
				TimeWindow: 7 * 24 * time.Hour, // Active in last 7 days
			},
		}

		err := service.DefineCohort(ctx, cohort)
		assert.NoError(t, err)

		// Get cohort members
		members, err := service.GetCohortMembers(ctx, cohort.ID, MemberParams{
			Limit:  100,
			Offset: 0,
		})

		assert.NoError(t, err)
		assert.NotNil(t, members)
		assert.NotEmpty(t, members.Users)

		for _, user := range members.Users {
			assert.NotEqual(t, uuid.Nil, user.UserID)
			assert.NotZero(t, user.JoinedAt)
			assert.NotEmpty(t, user.Attributes)
		}
	})

	t.Run("Cohort lifecycle analysis", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		cohortID := uuid.New()

		// Analyze cohort lifecycle stages
		lifecycle, err := service.AnalyzeLifecycle(ctx, cohortID, LifecycleParams{
			Stages: []string{"new", "active", "at_risk", "churned", "reactivated"},
			Window: 30 * 24 * time.Hour,
		})

		assert.NoError(t, err)
		assert.NotNil(t, lifecycle)
		assert.Len(t, lifecycle.StageDistribution, 5)

		totalPercentage := 0.0
		for _, stage := range lifecycle.StageDistribution {
			assert.NotEmpty(t, stage.Name)
			assert.GreaterOrEqual(t, stage.UserCount, int64(0))
			assert.GreaterOrEqual(t, stage.Percentage, 0.0)
			assert.LessOrEqual(t, stage.Percentage, 100.0)
			totalPercentage += stage.Percentage
		}
		assert.InDelta(t, 100.0, totalPercentage, 0.01)
	})

	t.Run("Revenue cohort analysis", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		cohortID := uuid.New()

		// Analyze revenue metrics for cohort
		revenue, err := service.AnalyzeRevenue(ctx, cohortID, RevenueParams{
			StartDate: time.Now().Add(-90 * 24 * time.Hour),
			EndDate:   time.Now(),
			GroupBy:   "month",
		})

		assert.NoError(t, err)
		assert.NotNil(t, revenue)
		assert.Greater(t, revenue.TotalRevenue, 0.0)
		assert.Greater(t, revenue.AverageRevenue, 0.0)
		assert.Greater(t, revenue.MedianRevenue, 0.0)
		assert.NotEmpty(t, revenue.RevenueByPeriod)

		for _, period := range revenue.RevenueByPeriod {
			assert.NotEmpty(t, period.Period)
			assert.GreaterOrEqual(t, period.Revenue, 0.0)
			assert.GreaterOrEqual(t, period.UserCount, int64(0))
			assert.GreaterOrEqual(t, period.ARPU, 0.0)
		}
	})

	t.Run("Predictive cohort analysis", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		cohortID := uuid.New()

		// Predict future behavior
		prediction, err := service.PredictBehavior(ctx, cohortID, PredictionParams{
			PredictionType: "churn_probability",
			TimeHorizon:    30 * 24 * time.Hour,
			Features: []string{
				"days_since_last_activity",
				"total_sessions",
				"total_revenue",
				"feature_adoption_rate",
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, prediction)
		assert.NotEmpty(t, prediction.Predictions)

		for _, pred := range prediction.Predictions {
			assert.NotEqual(t, uuid.Nil, pred.UserID)
			assert.GreaterOrEqual(t, pred.Probability, 0.0)
			assert.LessOrEqual(t, pred.Probability, 1.0)
			assert.GreaterOrEqual(t, pred.Confidence, 0.0)
			assert.LessOrEqual(t, pred.Confidence, 1.0)
		}

		assert.GreaterOrEqual(t, prediction.ModelAccuracy, 0.0)
		assert.LessOrEqual(t, prediction.ModelAccuracy, 1.0)
	})

	t.Run("Export cohort data", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		cohortID := uuid.New()

		// Export cohort data as CSV
		csvData, err := service.ExportCohort(ctx, cohortID, ExportParams{
			Format:         "csv",
			IncludeEvents:  true,
			IncludeMetrics: true,
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, csvData)
		assert.Contains(t, string(csvData), "user_id")
		assert.Contains(t, string(csvData), "joined_date")

		// Export as JSON
		jsonData, err := service.ExportCohort(ctx, cohortID, ExportParams{
			Format: "json",
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)
		assert.Contains(t, string(jsonData), "cohort_id")
		assert.Contains(t, string(jsonData), "users")
	})

	t.Run("Dynamic cohort with real-time updates", func(t *testing.T) {
		service := NewCohortService(nil, nil)

		// Create dynamic cohort that updates in real-time
		cohort := &CohortDefinition{
			ID:      uuid.New(),
			Name:    "Currently Active Users",
			Type:    "dynamic",
			Dynamic: true,
			Criteria: CohortCriteria{
				EventType:  "user_activity",
				TimeWindow: 5 * time.Minute, // Active in last 5 minutes
			},
		}

		err := service.DefineCohort(ctx, cohort)
		assert.NoError(t, err)

		// Subscribe to cohort changes
		updates, err := service.SubscribeToCohortUpdates(ctx, cohort.ID)
		assert.NoError(t, err)
		assert.NotNil(t, updates)

		// Should receive update
		select {
		case update := <-updates:
			assert.Equal(t, cohort.ID, update.CohortID)
			assert.NotEmpty(t, update.ChangeType) // "user_added" or "user_removed"
			assert.NotEqual(t, uuid.Nil, update.UserID)
		case <-time.After(10 * time.Second):
			t.Fatal("No cohort update received")
		}
	})
}
