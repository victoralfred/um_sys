package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BackgroundJob is the concrete implementation of Job interface
type BackgroundJob struct {
	ID           uuid.UUID              `json:"id"`
	Type         string                 `json:"type"`
	Payload      json.RawMessage        `json:"payload"`
	Status       JobStatus              `json:"status"`
	Priority     Priority               `json:"priority"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ScheduledFor *time.Time             `json:"scheduled_for,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	FailedAt     *time.Time             `json:"failed_at,omitempty"`
	LastError    string                 `json:"last_error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Result       json.RawMessage        `json:"result,omitempty"`
}

// GetID returns the job ID
func (j *BackgroundJob) GetID() uuid.UUID {
	return j.ID
}

// GetType returns the job type
func (j *BackgroundJob) GetType() string {
	return j.Type
}

// GetPayload returns the job payload
func (j *BackgroundJob) GetPayload() interface{} {
	return j.Payload
}

// GetStatus returns the job status
func (j *BackgroundJob) GetStatus() JobStatus {
	return j.Status
}

// GetPriority returns the job priority
func (j *BackgroundJob) GetPriority() Priority {
	return j.Priority
}

// GetRetryCount returns the retry count
func (j *BackgroundJob) GetRetryCount() int {
	return j.RetryCount
}

// GetMaxRetries returns the max retries
func (j *BackgroundJob) GetMaxRetries() int {
	return j.MaxRetries
}

// GetCreatedAt returns the creation time
func (j *BackgroundJob) GetCreatedAt() time.Time {
	return j.CreatedAt
}

// GetScheduledFor returns the scheduled time
func (j *BackgroundJob) GetScheduledFor() *time.Time {
	return j.ScheduledFor
}

// GetMetadata returns the job metadata
func (j *BackgroundJob) GetMetadata() map[string]interface{} {
	return j.Metadata
}

// JobHandlerFunc is an adapter to allow regular functions to be used as JobHandlers
type JobHandlerFunc struct {
	TypeName    string
	HandlerFunc func(ctx context.Context, job Job) error
	Timeout     time.Duration
}

// Handle executes the handler function
func (h *JobHandlerFunc) Handle(ctx context.Context, job Job) error {
	return h.HandlerFunc(ctx, job)
}

// GetType returns the job type this handler processes
func (h *JobHandlerFunc) GetType() string {
	return h.TypeName
}

// GetTimeout returns the handler timeout
func (h *JobHandlerFunc) GetTimeout() time.Duration {
	if h.Timeout == 0 {
		return 30 * time.Second // Default timeout
	}
	return h.Timeout
}

// ExponentialBackoffRetryStrategy implements exponential backoff retry
type ExponentialBackoffRetryStrategy struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	RetryableError func(error) bool
}

// ShouldRetry determines if a job should be retried
func (s *ExponentialBackoffRetryStrategy) ShouldRetry(job Job, err error) bool {
	if job.GetRetryCount() >= s.MaxRetries {
		return false
	}
	if s.RetryableError != nil {
		return s.RetryableError(err)
	}
	return true // Retry all errors by default
}

// NextRetryTime calculates the next retry time
func (s *ExponentialBackoffRetryStrategy) NextRetryTime(job Job) time.Time {
	retryCount := job.GetRetryCount()
	delay := s.InitialDelay

	for i := 0; i < retryCount; i++ {
		delay = time.Duration(float64(delay) * s.Multiplier)
		if delay > s.MaxDelay {
			delay = s.MaxDelay
			break
		}
	}

	return time.Now().Add(delay)
}

// GetMaxRetries returns the maximum number of retries
func (s *ExponentialBackoffRetryStrategy) GetMaxRetries() int {
	return s.MaxRetries
}

// LinearRetryStrategy implements linear retry with fixed delay
type LinearRetryStrategy struct {
	MaxRetries     int
	Delay          time.Duration
	RetryableError func(error) bool
}

// ShouldRetry determines if a job should be retried
func (s *LinearRetryStrategy) ShouldRetry(job Job, err error) bool {
	if job.GetRetryCount() >= s.MaxRetries {
		return false
	}
	if s.RetryableError != nil {
		return s.RetryableError(err)
	}
	return true
}

// NextRetryTime calculates the next retry time
func (s *LinearRetryStrategy) NextRetryTime(job Job) time.Time {
	return time.Now().Add(s.Delay)
}

// GetMaxRetries returns the maximum number of retries
func (s *LinearRetryStrategy) GetMaxRetries() int {
	return s.MaxRetries
}

// JobResult represents the result of a job execution
type JobResult struct {
	JobID     uuid.UUID       `json:"job_id"`
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	Duration  time.Duration   `json:"duration"`
	Timestamp time.Time       `json:"timestamp"`
}

// RecurringJobSchedule represents a recurring job schedule
type RecurringJobSchedule struct {
	ID        uuid.UUID              `json:"id"`
	JobType   string                 `json:"job_type"`
	CronExpr  string                 `json:"cron_expr"`
	Payload   json.RawMessage        `json:"payload"`
	Enabled   bool                   `json:"enabled"`
	LastRun   *time.Time             `json:"last_run,omitempty"`
	NextRun   *time.Time             `json:"next_run,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
