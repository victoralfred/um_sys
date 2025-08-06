package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/domain/job"
)

func TestJobConfigurationService(t *testing.T) {
	t.Run("Create job configuration", func(t *testing.T) {
		service := NewJobConfigurationService()
		userID := uuid.New()

		config := &job.JobConfiguration{
			ID:          uuid.New(),
			Name:        "Daily User Cleanup",
			Type:        "soft_delete_cleanup",
			Description: "Remove soft deleted users older than 30 days",
			Enabled:     true,
			Schedule: job.JobScheduleConfig{
				Type:     job.ScheduleTypeDaily,
				Timezone: "UTC",
				TimeOfDay: &job.TimeOfDay{
					Hour:   2,
					Minute: 0,
				},
			},
			Strategy: job.JobStrategyConfig{
				RetryStrategy: job.RetryConfig{
					MaxRetries:   3,
					InitialDelay: 5 * time.Second,
					MaxDelay:     1 * time.Minute,
					BackoffType:  job.BackoffTypeExponential,
				},
				Timeout:        5 * time.Minute,
				Priority:       job.PriorityNormal,
				MaxConcurrency: 1,
			},
			Parameters: map[string]interface{}{
				"entity":           "user",
				"retention_period": "720h", // 30 days
				"batch_size":       100,
			},
			CreatedBy: userID,
			UpdatedBy: userID,
		}

		err := service.CreateConfiguration(config)
		assert.NoError(t, err)

		// Retrieve configuration
		retrieved, err := service.GetConfiguration(config.ID)
		assert.NoError(t, err)
		assert.Equal(t, config.Name, retrieved.Name)
		assert.Equal(t, config.Type, retrieved.Type)
		assert.True(t, retrieved.Enabled)
	})

	t.Run("Update job configuration", func(t *testing.T) {
		service := NewJobConfigurationService()
		userID := uuid.New()

		// Create initial configuration
		config := &job.JobConfiguration{
			ID:          uuid.New(),
			Name:        "Config Cleanup",
			Type:        "soft_delete_cleanup",
			Description: "Remove soft deleted configurations",
			Enabled:     false,
			Schedule: job.JobScheduleConfig{
				Type:     job.ScheduleTypeWeekly,
				Timezone: "UTC",
			},
			Strategy: job.JobStrategyConfig{
				Timeout:  10 * time.Minute,
				Priority: job.PriorityLow,
			},
			CreatedBy: userID,
		}

		err := service.CreateConfiguration(config)
		require.NoError(t, err)

		// Update configuration
		updates := map[string]interface{}{
			"enabled":     true,
			"description": "Updated description",
			"schedule": job.JobScheduleConfig{
				Type:     job.ScheduleTypeDaily,
				Timezone: "UTC",
			},
		}

		err = service.UpdateConfiguration(config.ID, updates)
		assert.NoError(t, err)

		// Verify updates
		updated, err := service.GetConfiguration(config.ID)
		assert.NoError(t, err)
		assert.True(t, updated.Enabled)
		assert.Equal(t, "Updated description", updated.Description)
		assert.Equal(t, job.ScheduleTypeDaily, updated.Schedule.Type)
	})

	t.Run("Enable and disable configuration", func(t *testing.T) {
		service := NewJobConfigurationService()

		config := &job.JobConfiguration{
			ID:      uuid.New(),
			Name:    "Test Config",
			Type:    "test",
			Enabled: false,
		}

		err := service.CreateConfiguration(config)
		require.NoError(t, err)

		// Enable configuration
		err = service.EnableConfiguration(config.ID)
		assert.NoError(t, err)

		retrieved, _ := service.GetConfiguration(config.ID)
		assert.True(t, retrieved.Enabled)

		// Disable configuration
		err = service.DisableConfiguration(config.ID)
		assert.NoError(t, err)

		retrieved, _ = service.GetConfiguration(config.ID)
		assert.False(t, retrieved.Enabled)
	})

	t.Run("List configurations with filters", func(t *testing.T) {
		service := NewJobConfigurationService()

		// Create multiple configurations
		configs := []*job.JobConfiguration{
			{
				ID:      uuid.New(),
				Name:    "User Cleanup",
				Type:    "soft_delete_cleanup",
				Enabled: true,
			},
			{
				ID:      uuid.New(),
				Name:    "Config Cleanup",
				Type:    "soft_delete_cleanup",
				Enabled: false,
			},
			{
				ID:      uuid.New(),
				Name:    "Analytics Processing",
				Type:    "analytics",
				Enabled: true,
			},
		}

		for _, config := range configs {
			err := service.CreateConfiguration(config)
			require.NoError(t, err)
		}

		// Filter by type
		filtered, err := service.ListConfigurations(map[string]interface{}{
			"type": "soft_delete_cleanup",
		})
		assert.NoError(t, err)
		assert.Len(t, filtered, 2)

		// Filter by enabled
		filtered, err = service.ListConfigurations(map[string]interface{}{
			"enabled": true,
		})
		assert.NoError(t, err)
		assert.Len(t, filtered, 2)
	})

	t.Run("Validate job configuration", func(t *testing.T) {
		service := NewJobConfigurationService()

		// Valid configuration
		validConfig := &job.JobConfiguration{
			ID:   uuid.New(),
			Name: "Valid Config",
			Type: "soft_delete_cleanup",
			Schedule: job.JobScheduleConfig{
				Type:     job.ScheduleTypeCron,
				CronExpr: "0 2 * * *", // Valid cron expression
			},
			Strategy: job.JobStrategyConfig{
				Timeout:  5 * time.Minute,
				Priority: job.PriorityNormal,
			},
		}

		err := service.ValidateConfiguration(validConfig)
		assert.NoError(t, err)

		// Invalid configuration - missing cron expression
		invalidConfig := &job.JobConfiguration{
			ID:   uuid.New(),
			Name: "Invalid Config",
			Type: "soft_delete_cleanup",
			Schedule: job.JobScheduleConfig{
				Type: job.ScheduleTypeCron,
				// Missing CronExpr
			},
		}

		err = service.ValidateConfiguration(invalidConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cron expression")
	})

	t.Run("Schedule configured job", func(t *testing.T) {
		service := NewJobConfigurationService()
		jobService := NewJobService()
		service.SetJobService(jobService)

		config := &job.JobConfiguration{
			ID:      uuid.New(),
			Name:    "Scheduled Cleanup",
			Type:    "soft_delete_cleanup",
			Enabled: true,
			Schedule: job.JobScheduleConfig{
				Type:     job.ScheduleTypeInterval,
				Interval: 1 * time.Hour,
			},
			Strategy: job.JobStrategyConfig{
				Priority: job.PriorityHigh,
			},
			Parameters: map[string]interface{}{
				"entity":     "user",
				"batch_size": 50,
			},
		}

		err := service.CreateConfiguration(config)
		require.NoError(t, err)

		// Schedule the job
		err = service.ScheduleConfiguredJob(config.ID)
		assert.NoError(t, err)

		// Verify job was created in job service
		scheduledJobs, err := service.GetScheduledJobs()
		assert.NoError(t, err)
		assert.Len(t, scheduledJobs, 1)
		assert.Equal(t, config.ID, scheduledJobs[0].ID)
	})

	t.Run("Delete job configuration", func(t *testing.T) {
		service := NewJobConfigurationService()

		config := &job.JobConfiguration{
			ID:   uuid.New(),
			Name: "To Delete",
			Type: "test",
		}

		err := service.CreateConfiguration(config)
		require.NoError(t, err)

		// Delete configuration
		err = service.DeleteConfiguration(config.ID)
		assert.NoError(t, err)

		// Should not be found
		_, err = service.GetConfiguration(config.ID)
		assert.Error(t, err)
	})
}

func TestSoftDeleteCleanupHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("Configure soft delete cleanup", func(t *testing.T) {
		handler := NewSoftDeleteCleanupHandler()

		config := &job.SoftDeleteCleanupConfig{
			JobConfiguration: job.JobConfiguration{
				ID:   uuid.New(),
				Name: "User Cleanup",
				Type: "soft_delete_cleanup",
			},
			TargetEntity:    "user",
			RetentionPeriod: 30 * 24 * time.Hour, // 30 days
			BatchSize:       100,
			DryRun:          false,
		}

		err := handler.SetConfiguration(config)
		assert.NoError(t, err)
		assert.Equal(t, "soft_delete_cleanup", handler.GetType())
	})

	t.Run("Get soft deleted items", func(t *testing.T) {
		handler := NewSoftDeleteCleanupHandler()

		// Setup test data - simulate soft deleted users
		testUsers := []interface{}{
			map[string]interface{}{
				"id":         uuid.New().String(),
				"email":      "deleted1@example.com",
				"deleted_at": time.Now().Add(-35 * 24 * time.Hour), // 35 days ago
			},
			map[string]interface{}{
				"id":         uuid.New().String(),
				"email":      "deleted2@example.com",
				"deleted_at": time.Now().Add(-40 * 24 * time.Hour), // 40 days ago
			},
		}

		handler.SetTestData(testUsers)

		// Get items older than 30 days
		cutoffTime := time.Now().Add(-30 * 24 * time.Hour)
		items, err := handler.GetDeletedItems("user", cutoffTime, 10)
		assert.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("Permanently delete items", func(t *testing.T) {
		handler := NewSoftDeleteCleanupHandler()

		items := []interface{}{
			map[string]interface{}{"id": uuid.New().String()},
			map[string]interface{}{"id": uuid.New().String()},
		}

		err := handler.PermanentlyDelete(items)
		assert.NoError(t, err)

		// Verify statistics
		stats, err := handler.GetStatistics()
		assert.NoError(t, err)
		assert.Equal(t, int64(2), stats.TotalDeleted)
	})

	t.Run("Handle soft delete cleanup job", func(t *testing.T) {
		handler := NewSoftDeleteCleanupHandler()
		jobService := NewJobService()

		// Configure handler
		config := &job.SoftDeleteCleanupConfig{
			TargetEntity:    "user",
			RetentionPeriod: 30 * 24 * time.Hour,
			BatchSize:       50,
			DryRun:          false,
		}
		handler.SetConfiguration(config)

		// Register handler
		err := jobService.RegisterHandler("soft_delete_cleanup", handler)
		require.NoError(t, err)

		// Create job
		jobPayload := map[string]interface{}{
			"entity":           "user",
			"retention_period": "720h",
			"batch_size":       50,
		}

		createdJob, err := jobService.CreateJob(ctx, "soft_delete_cleanup", jobPayload, job.PriorityNormal)
		require.NoError(t, err)

		err = jobService.EnqueueJob(ctx, createdJob)
		require.NoError(t, err)

		// Process job
		err = jobService.ProcessNextJob(ctx, "soft_delete_cleanup")
		assert.NoError(t, err)

		// Check job completed
		processedJob, err := jobService.GetJob(ctx, createdJob.GetID())
		assert.NoError(t, err)
		assert.Equal(t, job.JobStatusCompleted, processedJob.GetStatus())
	})

	t.Run("Dry run mode", func(t *testing.T) {
		handler := NewSoftDeleteCleanupHandler()

		config := &job.SoftDeleteCleanupConfig{
			TargetEntity:    "configuration",
			RetentionPeriod: 7 * 24 * time.Hour,
			BatchSize:       10,
			DryRun:          true, // Dry run mode
		}

		handler.SetConfiguration(config)

		// Set test data
		testConfigs := []interface{}{
			map[string]interface{}{
				"id":         uuid.New().String(),
				"name":       "old_config",
				"deleted_at": time.Now().Add(-10 * 24 * time.Hour),
			},
		}
		handler.SetTestData(testConfigs)

		// Run in dry run mode
		j := &job.BackgroundJob{
			ID:      uuid.New(),
			Type:    "soft_delete_cleanup",
			Payload: nil,
		}

		err := handler.Handle(ctx, j)
		assert.NoError(t, err)

		// Verify nothing was actually deleted
		stats, _ := handler.GetStatistics()
		assert.Equal(t, int64(0), stats.TotalDeleted) // Nothing deleted in dry run
	})

	t.Run("Handle multiple entities", func(t *testing.T) {
		handler := NewSoftDeleteCleanupHandler()

		config := &job.SoftDeleteCleanupConfig{
			TargetEntity:    "all", // Clean all entities
			RetentionPeriod: 30 * 24 * time.Hour,
			BatchSize:       100,
			DryRun:          false,
		}

		err := handler.SetConfiguration(config)
		assert.NoError(t, err)

		// Should handle users, configurations, and other entities
		cutoffTime := time.Now().Add(-30 * 24 * time.Hour)

		// Get deleted users
		users, err := handler.GetDeletedItems("user", cutoffTime, 50)
		assert.NoError(t, err)

		// Get deleted configurations
		configs, err := handler.GetDeletedItems("configuration", cutoffTime, 50)
		assert.NoError(t, err)

		// Both should work when entity is "all"
		assert.NotNil(t, users)
		assert.NotNil(t, configs)
	})
}
