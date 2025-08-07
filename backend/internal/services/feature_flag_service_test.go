package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureFlagService(t *testing.T) {
	ctx := context.Background()

	t.Run("Create and retrieve feature flag", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create a boolean flag
		flagKey := "new-dashboard"
		flagName := "New Dashboard UI"
		flagDescription := "Enable new dashboard design for users"
		defaultValue := false

		createdFlag, err := service.CreateFlag(ctx, flagKey, flagName, flagDescription, defaultValue)
		assert.NoError(t, err)
		assert.NotNil(t, createdFlag)
		assert.Equal(t, flagKey, createdFlag.Key)
		assert.Equal(t, flagName, createdFlag.Name)

		// Retrieve the flag
		retrieved, err := service.GetFlag(ctx, flagKey)
		assert.NoError(t, err)
		assert.Equal(t, flagKey, retrieved.Key)
		assert.Equal(t, flagName, retrieved.Name)
		assert.Equal(t, defaultValue, retrieved.DefaultValue)
	})

	t.Run("Simple boolean flag evaluation", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "feature-x"
		_, err := service.CreateFlag(ctx, flagKey, "Feature X", "Test feature", false)
		require.NoError(t, err)

		// Evaluate for a user
		userID := uuid.New()
		result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
		assert.NoError(t, err)
		assert.Equal(t, false, result.Value)
		assert.Equal(t, "default", result.Reason)
	})

	t.Run("User targeting", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "beta-feature"
		_, err := service.CreateFlag(ctx, flagKey, "Beta Feature", "Beta testing", false)
		require.NoError(t, err)

		// Add user to beta list
		targetUserID := uuid.New()
		err = service.AddUserToFlag(ctx, flagKey, targetUserID)
		require.NoError(t, err)

		// Evaluate for targeted user
		result, err := service.EvaluateForUser(ctx, flagKey, targetUserID, nil)
		assert.NoError(t, err)
		assert.Equal(t, true, result.Value)
		assert.Equal(t, "rule_match", result.Reason)

		// Evaluate for non-targeted user
		otherUserID := uuid.New()
		result, err = service.EvaluateForUser(ctx, flagKey, otherUserID, nil)
		assert.NoError(t, err)
		assert.Equal(t, false, result.Value)
		assert.Equal(t, "default", result.Reason)
	})

	t.Run("Percentage rollout", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag with percentage rollout
		flagKey := "gradual-rollout"
		_, err := service.CreateFlag(ctx, flagKey, "Gradual Rollout", "50% rollout", false)
		require.NoError(t, err)

		// Set percentage rollout
		err = service.SetPercentageRollout(ctx, flagKey, 50)
		require.NoError(t, err)

		// Evaluate for multiple users
		enabledCount := 0
		totalUsers := 1000

		for i := 0; i < totalUsers; i++ {
			userID := uuid.New()
			result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
			assert.NoError(t, err)

			if result.Value.(bool) {
				enabledCount++
			}
		}

		// Should be roughly 50% (with some variance)
		percentage := float64(enabledCount) / float64(totalUsers) * 100
		assert.InDelta(t, 50.0, percentage, 10.0, "Rollout percentage should be close to 50%")
	})

	t.Run("Property-based targeting", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "premium-feature"
		_, err := service.CreateFlag(ctx, flagKey, "Premium Feature", "Premium only", false)
		require.NoError(t, err)

		// Add property rule
		err = service.AddPropertyRule(ctx, flagKey, "subscription_plan", "equals", "premium", true)
		require.NoError(t, err)

		// Evaluate for premium user
		userID := uuid.New()
		properties := map[string]interface{}{
			"subscription_plan": "premium",
		}
		result, err := service.EvaluateForUser(ctx, flagKey, userID, properties)
		assert.NoError(t, err)
		assert.Equal(t, true, result.Value)

		// Evaluate for free user
		freeProperties := map[string]interface{}{
			"subscription_plan": "free",
		}
		result, err = service.EvaluateForUser(ctx, flagKey, userID, freeProperties)
		assert.NoError(t, err)
		assert.Equal(t, false, result.Value)
	})

	t.Run("String flag with variants", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create string flag
		flagKey := "button-color"
		_, err := service.CreateStringFlag(ctx, flagKey, "Button Color", "UI test", "blue")
		require.NoError(t, err)

		// Add variants
		err = service.AddVariant(ctx, flagKey, "variant-a", "green", 33)
		require.NoError(t, err)

		err = service.AddVariant(ctx, flagKey, "variant-b", "red", 33)
		require.NoError(t, err)

		err = service.AddVariant(ctx, flagKey, "control", "blue", 34)
		require.NoError(t, err)

		// Evaluate multiple times
		colors := make(map[string]int)
		for i := 0; i < 1000; i++ {
			userID := uuid.New()
			result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
			assert.NoError(t, err)
			color := result.Value.(string)
			colors[color]++
		}

		// All variants should be present
		assert.Contains(t, colors, "blue")
		assert.Contains(t, colors, "green")
		assert.Contains(t, colors, "red")
	})

	t.Run("JSON configuration flag", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create JSON flag
		defaultConfig := map[string]interface{}{
			"rate_limit":  100,
			"timeout_ms":  5000,
			"retry_count": 3,
		}

		flagKey := "api-config"
		_, err := service.CreateJSONFlag(ctx, flagKey, "API Config", "API configuration", defaultConfig)
		require.NoError(t, err)

		// Evaluate configuration
		userID := uuid.New()
		result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
		assert.NoError(t, err)

		config := result.Value.(map[string]interface{})
		assert.Equal(t, 100, config["rate_limit"])
		assert.Equal(t, 5000, config["timeout_ms"])
		assert.Equal(t, 3, config["retry_count"])
	})

	t.Run("Bulk evaluation", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create multiple flags
		flags := []struct {
			key          string
			defaultValue interface{}
		}{
			{"feature-1", true},
			{"feature-2", "default"},
			{"feature-3", float64(42)},
		}

		for _, f := range flags {
			_, err := service.CreateFlag(ctx, f.key, f.key, "Test flag", f.defaultValue)
			require.NoError(t, err)
		}

		// Bulk evaluate
		userID := uuid.New()
		flagKeys := []string{"feature-1", "feature-2", "feature-3"}
		results, err := service.EvaluateAll(ctx, flagKeys, userID, nil)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, true, results["feature-1"])
		assert.Equal(t, "default", results["feature-2"])
		assert.Equal(t, float64(42), results["feature-3"])
	})

	t.Run("Flag override for specific user", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "override-test"
		_, err := service.CreateFlag(ctx, flagKey, "Override Test", "Test overrides", false)
		require.NoError(t, err)

		// Create override for specific user
		userID := uuid.New()
		err = service.CreateOverride(ctx, flagKey, userID, true, "Testing override")
		assert.NoError(t, err)

		// Evaluate for overridden user
		result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
		assert.NoError(t, err)
		assert.Equal(t, true, result.Value)
		assert.Equal(t, "override", result.Reason)

		// Evaluate for different user
		otherUserID := uuid.New()
		result, err = service.EvaluateForUser(ctx, flagKey, otherUserID, nil)
		assert.NoError(t, err)
		assert.Equal(t, false, result.Value)
	})

	t.Run("Flag disabled returns default", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "disabled-flag"
		_, err := service.CreateFlag(ctx, flagKey, "Disabled Flag", "Test disabled", false)
		require.NoError(t, err)

		// Disable flag
		err = service.DisableFlag(ctx, flagKey)
		require.NoError(t, err)

		// Should return default when disabled
		userID := uuid.New()
		result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
		assert.NoError(t, err)
		assert.Equal(t, false, result.Value)
		assert.Equal(t, "flag_disabled", result.Reason)
	})

	t.Run("Environment-based targeting", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "env-feature"
		_, err := service.CreateFlag(ctx, flagKey, "Env Feature", "Environment specific", false)
		require.NoError(t, err)

		// Enable for production only
		err = service.EnableForEnvironment(ctx, flagKey, "production")
		require.NoError(t, err)

		// Test production environment
		userID := uuid.New()
		result, err := service.EvaluateForUserInEnvironment(ctx, flagKey, userID, "production", nil)
		assert.NoError(t, err)
		assert.Equal(t, true, result.Value)

		// Test development environment
		result, err = service.EvaluateForUserInEnvironment(ctx, flagKey, userID, "development", nil)
		assert.NoError(t, err)
		assert.Equal(t, false, result.Value)
	})

	t.Run("Update flag configuration", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "updateable-flag"
		_, err := service.CreateFlag(ctx, flagKey, "Updateable Flag", "Test updates", false)
		require.NoError(t, err)

		// Update flag
		err = service.UpdateFlag(ctx, flagKey, "Updated Flag", "Updated description", true)
		assert.NoError(t, err)

		// Verify update
		updated, err := service.GetFlag(ctx, flagKey)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Flag", updated.Name)
		assert.Equal(t, "Updated description", updated.Description)
		assert.Equal(t, true, updated.DefaultValue)
	})

	t.Run("Delete feature flag", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "deletable-flag"
		_, err := service.CreateFlag(ctx, flagKey, "Deletable Flag", "Test deletion", false)
		require.NoError(t, err)

		// Delete flag
		err = service.DeleteFlag(ctx, flagKey)
		assert.NoError(t, err)

		// Try to get deleted flag
		_, err = service.GetFlag(ctx, flagKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Flag change history", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag
		flagKey := "tracked-flag"
		_, err := service.CreateFlag(ctx, flagKey, "Tracked Flag", "Track changes", false)
		require.NoError(t, err)

		// Make some changes
		err = service.UpdateFlag(ctx, flagKey, "Updated Tracked Flag", "Updated description", true)
		require.NoError(t, err)

		// Get change history
		history, err := service.GetFlagHistory(ctx, flagKey)
		assert.NoError(t, err)
		assert.NotEmpty(t, history)

		// Should have create and update events
		assert.GreaterOrEqual(t, len(history), 2)
	})

	t.Run("Consistent hash for percentage rollout", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag with percentage
		flagKey := "consistent-rollout"
		_, err := service.CreateFlag(ctx, flagKey, "Consistent Rollout", "Test consistency", false)
		require.NoError(t, err)

		err = service.SetPercentageRollout(ctx, flagKey, 30)
		require.NoError(t, err)

		// Same user should always get same result
		userID := uuid.New()
		firstResult, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
		assert.NoError(t, err)

		// Evaluate multiple times for same user
		for i := 0; i < 10; i++ {
			result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
			assert.NoError(t, err)
			assert.Equal(t, firstResult.Value, result.Value, "Same user should always get same result")
		}
	})

	t.Run("A/B test with multiple variants", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create experiment
		flagKey := "homepage-experiment"
		_, err := service.CreateStringFlag(ctx, flagKey, "Homepage Experiment", "A/B test", "control")
		require.NoError(t, err)

		// Create experiment with variants
		err = service.CreateExperiment(ctx, flagKey, []ExperimentVariant{
			{Name: "control", Value: "original", Weight: 33},
			{Name: "variant-a", Value: "new-design-a", Weight: 33},
			{Name: "variant-b", Value: "new-design-b", Weight: 34},
		})
		require.NoError(t, err)

		// Track variant distribution
		variants := make(map[string]int)
		for i := 0; i < 3000; i++ {
			userID := uuid.New()
			result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
			assert.NoError(t, err)
			value := result.Value.(string)
			variants[value]++
		}

		// All variants should have users
		assert.Greater(t, variants["original"], 0)
		assert.Greater(t, variants["new-design-a"], 0)
		assert.Greater(t, variants["new-design-b"], 0)

		// Distribution should be roughly equal (with some variance)
		for _, count := range variants {
			percentage := float64(count) / 3000.0 * 100
			assert.InDelta(t, 33.3, percentage, 5.0)
		}
	})

	t.Run("Schedule-based feature activation", func(t *testing.T) {
		service := NewFeatureFlagService(nil, nil)

		// Create flag with schedule
		flagKey := "scheduled-feature"
		_, err := service.CreateFlag(ctx, flagKey, "Scheduled Feature", "Time-based", false)
		require.NoError(t, err)

		// Schedule activation
		startTime := time.Now().Add(1 * time.Hour)
		endTime := time.Now().Add(2 * time.Hour)
		err = service.ScheduleFlag(ctx, flagKey, startTime, endTime)
		require.NoError(t, err)

		// Before schedule - should be disabled
		userID := uuid.New()
		result, err := service.EvaluateForUser(ctx, flagKey, userID, nil)
		assert.NoError(t, err)
		assert.Equal(t, false, result.Value)

		// Simulate time progression (in real implementation would check actual time)
		// This is just to show the expected behavior
	})
}
