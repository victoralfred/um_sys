package feature

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFeatureFlag(t *testing.T) {
	ctx := context.Background()

	t.Run("Create and validate feature flag", func(t *testing.T) {
		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "new-dashboard",
			Name:         "New Dashboard UI",
			Description:  "Enable the redesigned dashboard interface",
			Type:         FlagTypeBoolean,
			DefaultValue: false,
			Enabled:      true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		assert.NotNil(t, flag)
		assert.Equal(t, "new-dashboard", flag.Key)
		assert.Equal(t, FlagTypeBoolean, flag.Type)
		assert.False(t, flag.DefaultValue.(bool))
	})

	t.Run("Evaluate flag with targeting rules", func(t *testing.T) {
		evaluator := NewFeatureFlagEvaluator()

		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "premium-features",
			Type:         FlagTypeBoolean,
			DefaultValue: false,
			Enabled:      true,
			Rules: []TargetingRule{
				{
					ID:       uuid.New(),
					Priority: 1,
					Enabled:  true,
					Conditions: []Condition{
						{
							Property: "subscription",
							Operator: OperatorEquals,
							Value:    "premium",
						},
					},
					Value: true,
				},
			},
		}

		// Premium user should get feature
		premiumContext := EvaluationContext{
			UserID: uuid.New(),
			Properties: map[string]interface{}{
				"subscription": "premium",
			},
		}

		result := evaluator.Evaluate(ctx, flag, premiumContext)
		assert.True(t, result.Value.(bool))
		assert.Equal(t, "rule_match", result.Reason)

		// Free user should not get feature
		freeContext := EvaluationContext{
			UserID: uuid.New(),
			Properties: map[string]interface{}{
				"subscription": "free",
			},
		}

		result = evaluator.Evaluate(ctx, flag, freeContext)
		assert.False(t, result.Value.(bool))
		assert.Equal(t, "default", result.Reason)
	})

	t.Run("Percentage rollout", func(t *testing.T) {
		evaluator := NewFeatureFlagEvaluator()

		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "experimental-feature",
			Type:         FlagTypeBoolean,
			DefaultValue: false,
			Enabled:      true,
			RolloutStrategy: &RolloutStrategy{
				Type:       RolloutTypePercentage,
				Percentage: 30, // 30% of users
				Sticky:     true,
			},
		}

		// Test with multiple users
		includedCount := 0
		totalUsers := 100

		userResults := make(map[uuid.UUID]bool)
		for i := 0; i < totalUsers; i++ {
			userID := uuid.New()
			context := EvaluationContext{
				UserID: userID,
			}

			result := evaluator.Evaluate(ctx, flag, context)
			if result.Value.(bool) {
				includedCount++
			}
			userResults[userID] = result.Value.(bool)

			// Verify stickiness - same user always gets same result
			result2 := evaluator.Evaluate(ctx, flag, context)
			assert.Equal(t, result.Value, result2.Value, "Sticky rollout should return same value for same user")
		}

		// Should be approximately 30% (with some tolerance)
		percentage := float64(includedCount) / float64(totalUsers) * 100
		assert.InDelta(t, 30.0, percentage, 10.0, "Rollout percentage should be approximately 30%")
	})

	t.Run("Multi-variant testing", func(t *testing.T) {
		evaluator := NewFeatureFlagEvaluator()

		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "checkout-flow",
			Type:         FlagTypeString,
			DefaultValue: "classic",
			Enabled:      true,
			Variants: []Variant{
				{Key: "classic", Value: "classic", Weight: 50},
				{Key: "streamlined", Value: "streamlined", Weight: 30},
				{Key: "one-click", Value: "one-click", Weight: 20},
			},
		}

		variantCounts := make(map[string]int)
		totalUsers := 100

		for i := 0; i < totalUsers; i++ {
			context := EvaluationContext{
				UserID: uuid.New(),
			}

			result := evaluator.Evaluate(ctx, flag, context)
			variantCounts[result.Value.(string)]++
		}

		// All variants should be represented
		assert.Greater(t, variantCounts["classic"], 0)
		assert.Greater(t, variantCounts["streamlined"], 0)
		assert.Greater(t, variantCounts["one-click"], 0)
	})

	t.Run("Schedule-based activation", func(t *testing.T) {
		evaluator := NewFeatureFlagEvaluator()

		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "holiday-sale",
			Type:         FlagTypeBoolean,
			DefaultValue: false,
			Enabled:      true,
			Schedule: &Schedule{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(1 * time.Hour),
				Timezone:  "UTC",
			},
		}

		context := EvaluationContext{
			UserID:    uuid.New(),
			Timestamp: time.Now(),
		}

		// Within schedule
		result := evaluator.Evaluate(ctx, flag, context)
		assert.True(t, result.Value.(bool))

		// Before schedule
		context.Timestamp = time.Now().Add(-2 * time.Hour)
		result = evaluator.Evaluate(ctx, flag, context)
		assert.False(t, result.Value.(bool))

		// After schedule
		context.Timestamp = time.Now().Add(2 * time.Hour)
		result = evaluator.Evaluate(ctx, flag, context)
		assert.False(t, result.Value.(bool))
	})

	t.Run("User overrides", func(t *testing.T) {
		evaluator := NewFeatureFlagEvaluator()

		userID := uuid.New()
		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "beta-feature",
			Type:         FlagTypeBoolean,
			DefaultValue: false,
			Enabled:      true,
			Overrides: []Override{
				{
					Type:   OverrideTypeUser,
					Target: userID.String(),
					Value:  true,
				},
			},
		}

		// User with override
		context := EvaluationContext{
			UserID: userID,
		}

		result := evaluator.Evaluate(ctx, flag, context)
		assert.True(t, result.Value.(bool))
		assert.Equal(t, "override", result.Reason)

		// Different user without override
		context.UserID = uuid.New()
		result = evaluator.Evaluate(ctx, flag, context)
		assert.False(t, result.Value.(bool))
		assert.Equal(t, "default", result.Reason)
	})

	t.Run("Complex conditions with multiple operators", func(t *testing.T) {
		evaluator := NewFeatureFlagEvaluator()

		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "power-user-feature",
			Type:         FlagTypeBoolean,
			DefaultValue: false,
			Enabled:      true,
			Rules: []TargetingRule{
				{
					Priority: 1,
					Enabled:  true,
					Conditions: []Condition{
						{
							Property: "usage_count",
							Operator: OperatorGreaterThan,
							Value:    100,
						},
						{
							Property: "account_age_days",
							Operator: OperatorGreaterThanOrEqual,
							Value:    30,
						},
						{
							Property: "country",
							Operator: OperatorIn,
							Value:    []string{"US", "CA", "UK"},
						},
					},
					Value: true,
				},
			},
		}

		// User meeting all conditions
		context := EvaluationContext{
			UserID: uuid.New(),
			Properties: map[string]interface{}{
				"usage_count":      150,
				"account_age_days": 45,
				"country":          "US",
			},
		}

		result := evaluator.Evaluate(ctx, flag, context)
		assert.True(t, result.Value.(bool))

		// User not meeting usage condition
		context.Properties["usage_count"] = 50
		result = evaluator.Evaluate(ctx, flag, context)
		assert.False(t, result.Value.(bool))
	})

	t.Run("JSON configuration flag", func(t *testing.T) {
		evaluator := NewFeatureFlagEvaluator()

		defaultConfig := map[string]interface{}{
			"api_timeout":  30,
			"max_retries":  3,
			"batch_size":   100,
			"enable_cache": true,
		}

		premiumConfig := map[string]interface{}{
			"api_timeout":  60,
			"max_retries":  5,
			"batch_size":   500,
			"enable_cache": true,
		}

		flag := &FeatureFlag{
			ID:           uuid.New(),
			Key:          "api-config",
			Type:         FlagTypeJSON,
			DefaultValue: defaultConfig,
			Enabled:      true,
			Rules: []TargetingRule{
				{
					Priority: 1,
					Enabled:  true,
					Conditions: []Condition{
						{
							Property: "tier",
							Operator: OperatorEquals,
							Value:    "premium",
						},
					},
					Value: premiumConfig,
				},
			},
		}

		// Premium user gets enhanced config
		context := EvaluationContext{
			UserID: uuid.New(),
			Properties: map[string]interface{}{
				"tier": "premium",
			},
		}

		result := evaluator.Evaluate(ctx, flag, context)
		config := result.Value.(map[string]interface{})
		assert.Equal(t, 60, config["api_timeout"])
		assert.Equal(t, 5, config["max_retries"])
		assert.Equal(t, 500, config["batch_size"])
	})
}
