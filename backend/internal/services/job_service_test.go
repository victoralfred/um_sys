package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/domain/job"
)

func TestJobService(t *testing.T) {
	ctx := context.Background()

	t.Run("Create and enqueue job", func(t *testing.T) {
		service := NewJobService()

		// Create a job
		payload := map[string]interface{}{
			"user_id": uuid.New().String(),
			"action":  "send_welcome_email",
		}

		createdJob, err := service.CreateJob(ctx, "email", payload, job.PriorityNormal)
		assert.NoError(t, err)
		assert.NotNil(t, createdJob)
		assert.Equal(t, "email", createdJob.GetType())
		assert.Equal(t, job.JobStatusPending, createdJob.GetStatus())

		// Enqueue the job
		err = service.EnqueueJob(ctx, createdJob)
		assert.NoError(t, err)

		// Verify job is queued
		queuedJob, err := service.GetJob(ctx, createdJob.GetID())
		assert.NoError(t, err)
		assert.Equal(t, job.JobStatusQueued, queuedJob.GetStatus())
	})

	t.Run("Process job with handler", func(t *testing.T) {
		service := NewJobService()

		// Register a handler
		handlerCalled := false
		handler := &job.JobHandlerFunc{
			TypeName: "process_data",
			HandlerFunc: func(ctx context.Context, j job.Job) error {
				handlerCalled = true
				return nil
			},
			Timeout: 5 * time.Second,
		}

		err := service.RegisterHandler("process_data", handler)
		assert.NoError(t, err)

		// Create and enqueue job
		payload := map[string]interface{}{"data": "test"}
		j, err := service.CreateJob(ctx, "process_data", payload, job.PriorityNormal)
		require.NoError(t, err)

		err = service.EnqueueJob(ctx, j)
		require.NoError(t, err)

		// Process the job
		err = service.ProcessNextJob(ctx, "process_data")
		assert.NoError(t, err)
		assert.True(t, handlerCalled)

		// Check job status
		processedJob, err := service.GetJob(ctx, j.GetID())
		assert.NoError(t, err)
		assert.Equal(t, job.JobStatusCompleted, processedJob.GetStatus())
	})

	t.Run("Schedule job for future execution", func(t *testing.T) {
		service := NewJobService()

		// Schedule a job for 1 hour from now
		runAt := time.Now().Add(1 * time.Hour)
		payload := map[string]interface{}{"task": "scheduled_task"}

		scheduledJob, err := service.ScheduleJob(ctx, "scheduled", payload, runAt, job.PriorityHigh)
		assert.NoError(t, err)
		assert.NotNil(t, scheduledJob)
		assert.Equal(t, job.JobStatusScheduled, scheduledJob.GetStatus())
		assert.NotNil(t, scheduledJob.GetScheduledFor())
		assert.True(t, scheduledJob.GetScheduledFor().Equal(runAt))

		// Get scheduled jobs
		scheduledJobs, err := service.GetScheduledJobs(ctx)
		assert.NoError(t, err)
		assert.Len(t, scheduledJobs, 1)
		assert.Equal(t, scheduledJob.GetID(), scheduledJobs[0].GetID())
	})

	t.Run("Retry failed job with exponential backoff", func(t *testing.T) {
		service := NewJobService()

		// Set retry strategy with very short delays for testing
		strategy := &job.ExponentialBackoffRetryStrategy{
			MaxRetries:   3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
		}
		service.SetRetryStrategy(strategy)

		// Register a failing handler
		attemptCount := 0
		handler := &job.JobHandlerFunc{
			TypeName: "retry_test",
			HandlerFunc: func(ctx context.Context, j job.Job) error {
				attemptCount++
				if attemptCount < 3 {
					return errors.New("temporary failure")
				}
				return nil
			},
		}

		err := service.RegisterHandler("retry_test", handler)
		require.NoError(t, err)

		// Create and process job
		payload := map[string]interface{}{"data": "retry"}
		j, err := service.CreateJob(ctx, "retry_test", payload, job.PriorityNormal)
		require.NoError(t, err)

		err = service.EnqueueJob(ctx, j)
		require.NoError(t, err)

		// First attempt - should fail and be retried
		err = service.ProcessNextJob(ctx, "retry_test")
		assert.Error(t, err)

		retriedJob, err := service.GetJob(ctx, j.GetID())
		assert.NoError(t, err)
		assert.Equal(t, job.JobStatusRetrying, retriedJob.GetStatus())
		assert.Equal(t, 1, retriedJob.GetRetryCount())

		// Process retries - use a loop with timeout
		maxAttempts := 10
		for i := 0; i < maxAttempts && attemptCount < 3; i++ {
			time.Sleep(20 * time.Millisecond) // Short delay between attempts
			_ = service.ProcessNextJob(ctx, "retry_test")
		}

		// Verify job eventually succeeded
		finalJob, err := service.GetJob(ctx, j.GetID())
		assert.NoError(t, err)
		assert.Equal(t, job.JobStatusCompleted, finalJob.GetStatus())
	})

	t.Run("Monitor job metrics", func(t *testing.T) {
		service := NewJobService()

		// Register handler
		handler := &job.JobHandlerFunc{
			TypeName: "metrics_test",
			HandlerFunc: func(ctx context.Context, j job.Job) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		}

		err := service.RegisterHandler("metrics_test", handler)
		require.NoError(t, err)

		// Process multiple jobs
		for i := 0; i < 5; i++ {
			payload := map[string]interface{}{"index": i}
			j, err := service.CreateJob(ctx, "metrics_test", payload, job.PriorityNormal)
			require.NoError(t, err)

			err = service.EnqueueJob(ctx, j)
			require.NoError(t, err)

			err = service.ProcessNextJob(ctx, "metrics_test")
			require.NoError(t, err)
		}

		// Get metrics
		metrics, err := service.GetJobMetrics(ctx, "metrics_test")
		assert.NoError(t, err)
		assert.Equal(t, int64(5), metrics.TotalExecuted)
		assert.Equal(t, int64(5), metrics.TotalSucceeded)
		assert.Equal(t, int64(0), metrics.TotalFailed)
		assert.True(t, metrics.AverageDuration >= 100*time.Millisecond)
		assert.NotNil(t, metrics.LastExecuted)
		assert.NotNil(t, metrics.LastSuccess)
	})

	t.Run("Cancel scheduled job", func(t *testing.T) {
		service := NewJobService()

		// Schedule a job
		runAt := time.Now().Add(1 * time.Hour)
		payload := map[string]interface{}{"task": "cancellable"}

		scheduledJob, err := service.ScheduleJob(ctx, "cancellable", payload, runAt, job.PriorityNormal)
		require.NoError(t, err)

		// Cancel the job
		err = service.CancelJob(ctx, scheduledJob.GetID())
		assert.NoError(t, err)

		// Verify job is cancelled
		cancelledJob, err := service.GetJob(ctx, scheduledJob.GetID())
		assert.NoError(t, err)
		assert.Equal(t, job.JobStatusCancelled, cancelledJob.GetStatus())

		// Should not appear in scheduled jobs
		scheduledJobs, err := service.GetScheduledJobs(ctx)
		assert.NoError(t, err)
		assert.Len(t, scheduledJobs, 0)
	})

	t.Run("Priority queue processing", func(t *testing.T) {
		service := NewJobService()

		processOrder := []string{}
		handler := &job.JobHandlerFunc{
			TypeName: "priority_test",
			HandlerFunc: func(ctx context.Context, j job.Job) error {
				payload := j.GetPayload().(json.RawMessage)
				var data map[string]interface{}
				_ = json.Unmarshal(payload, &data)
				processOrder = append(processOrder, data["name"].(string))
				return nil
			},
		}

		err := service.RegisterHandler("priority_test", handler)
		require.NoError(t, err)

		// Create jobs with different priorities
		jobs := []struct {
			name     string
			priority job.Priority
		}{
			{"low", job.PriorityLow},
			{"urgent", job.PriorityUrgent},
			{"normal", job.PriorityNormal},
			{"high", job.PriorityHigh},
		}

		for _, j := range jobs {
			payload := map[string]interface{}{"name": j.name}
			createdJob, err := service.CreateJob(ctx, "priority_test", payload, j.priority)
			require.NoError(t, err)
			err = service.EnqueueJob(ctx, createdJob)
			require.NoError(t, err)
		}

		// Process all jobs
		for i := 0; i < 4; i++ {
			err = service.ProcessNextJob(ctx, "priority_test")
			require.NoError(t, err)
		}

		// Verify processing order (urgent -> high -> normal -> low)
		assert.Equal(t, []string{"urgent", "high", "normal", "low"}, processOrder)
	})

	t.Run("Recurring job scheduling", func(t *testing.T) {
		service := NewJobService()

		// Schedule a recurring job (every minute)
		cronExpr := "* * * * *"
		payload := map[string]interface{}{"task": "recurring"}

		schedule, err := service.ScheduleRecurringJob(ctx, "recurring", payload, cronExpr)
		assert.NoError(t, err)
		assert.NotNil(t, schedule)
		assert.Equal(t, cronExpr, schedule.CronExpr)
		assert.True(t, schedule.Enabled)

		// Get recurring schedules
		schedules, err := service.GetRecurringSchedules(ctx)
		assert.NoError(t, err)
		assert.Len(t, schedules, 1)
		assert.Equal(t, schedule.ID, schedules[0].ID)

		// Disable recurring job
		err = service.DisableRecurringJob(ctx, schedule.ID)
		assert.NoError(t, err)

		// Verify it's disabled
		updatedSchedule, err := service.GetRecurringSchedule(ctx, schedule.ID)
		assert.NoError(t, err)
		assert.False(t, updatedSchedule.Enabled)
	})

	t.Run("Job timeout handling", func(t *testing.T) {
		service := NewJobService()

		// Set retry strategy that doesn't retry timeouts
		service.SetRetryStrategy(&job.LinearRetryStrategy{
			MaxRetries: 3,
			Delay:      1 * time.Second,
			RetryableError: func(err error) bool {
				// Don't retry context errors (timeouts/cancellations)
				return !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled)
			},
		})

		// Register handler with short timeout
		handler := &job.JobHandlerFunc{
			TypeName: "timeout_test",
			HandlerFunc: func(ctx context.Context, j job.Job) error {
				select {
				case <-time.After(5 * time.Second):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
			Timeout: 100 * time.Millisecond,
		}

		err := service.RegisterHandler("timeout_test", handler)
		require.NoError(t, err)

		// Create and process job
		payload := map[string]interface{}{"data": "timeout"}
		j, err := service.CreateJob(ctx, "timeout_test", payload, job.PriorityNormal)
		require.NoError(t, err)

		err = service.EnqueueJob(ctx, j)
		require.NoError(t, err)

		// Should timeout
		err = service.ProcessNextJob(ctx, "timeout_test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")

		// Job should be marked as failed
		failedJob, err := service.GetJob(ctx, j.GetID())
		assert.NoError(t, err)
		assert.Equal(t, job.JobStatusFailed, failedJob.GetStatus())
	})

	t.Run("Bulk job operations", func(t *testing.T) {
		service := NewJobService()

		// Create multiple jobs
		for i := 0; i < 10; i++ {
			payload := map[string]interface{}{"index": i}
			j, err := service.CreateJob(ctx, "bulk_test", payload, job.PriorityNormal)
			require.NoError(t, err)

			err = service.EnqueueJob(ctx, j)
			require.NoError(t, err)
		}

		// Get jobs by status
		queuedJobs, err := service.GetJobsByStatus(ctx, job.JobStatusQueued)
		assert.NoError(t, err)
		assert.Len(t, queuedJobs, 10)

		// Clear queue
		err = service.ClearQueue(ctx, "bulk_test")
		assert.NoError(t, err)

		// Verify queue is empty
		queueSize, err := service.GetQueueSize(ctx, "bulk_test")
		assert.NoError(t, err)
		assert.Equal(t, 0, queueSize)
	})
}
