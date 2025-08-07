package job

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Job represents a background job
type Job interface {
	GetID() uuid.UUID
	GetType() string
	GetPayload() interface{}
	GetStatus() JobStatus
	GetPriority() Priority
	GetRetryCount() int
	GetMaxRetries() int
	GetCreatedAt() time.Time
	GetScheduledFor() *time.Time
	GetMetadata() map[string]interface{}
}

// JobQueue manages job queuing (Single Responsibility Principle)
type JobQueue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context, jobType string) (Job, error)
	Peek(ctx context.Context, jobType string) (Job, error)
	Size(ctx context.Context, jobType string) (int, error)
	Clear(ctx context.Context, jobType string) error
}

// JobProcessor processes jobs (Single Responsibility Principle)
type JobProcessor interface {
	Process(ctx context.Context, job Job) error
	RegisterHandler(jobType string, handler JobHandler) error
	UnregisterHandler(jobType string) error
	GetRegisteredTypes() []string
}

// JobHandler handles specific job types
type JobHandler interface {
	Handle(ctx context.Context, job Job) error
	GetType() string
	GetTimeout() time.Duration
}

// JobScheduler schedules jobs (Single Responsibility Principle)
type JobScheduler interface {
	Schedule(ctx context.Context, job Job, runAt time.Time) error
	ScheduleRecurring(ctx context.Context, job Job, cronExpr string) error
	CancelScheduled(ctx context.Context, jobID uuid.UUID) error
	GetScheduledJobs(ctx context.Context) ([]Job, error)
}

// JobRetryStrategy defines retry behavior (Strategy Pattern)
type JobRetryStrategy interface {
	ShouldRetry(job Job, err error) bool
	NextRetryTime(job Job) time.Time
	GetMaxRetries() int
}

// JobMonitor monitors job execution (Observer Pattern)
type JobMonitor interface {
	RecordStart(ctx context.Context, job Job) error
	RecordSuccess(ctx context.Context, job Job, duration time.Duration) error
	RecordFailure(ctx context.Context, job Job, err error, duration time.Duration) error
	RecordRetry(ctx context.Context, job Job, attempt int) error
	GetMetrics(ctx context.Context, jobType string) (*JobMetrics, error)
}

// JobStore persists jobs (Repository Pattern)
type JobStore interface {
	Save(ctx context.Context, job Job) error
	Get(ctx context.Context, jobID uuid.UUID) (Job, error)
	Update(ctx context.Context, job Job) error
	Delete(ctx context.Context, jobID uuid.UUID) error
	FindByStatus(ctx context.Context, status JobStatus) ([]Job, error)
	FindByType(ctx context.Context, jobType string) ([]Job, error)
}

// JobFactory creates jobs (Factory Pattern)
type JobFactory interface {
	CreateJob(jobType string, payload interface{}) (Job, error)
	CreateScheduledJob(jobType string, payload interface{}, runAt time.Time) (Job, error)
	CreateRecurringJob(jobType string, payload interface{}, cronExpr string) (Job, error)
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRetrying  JobStatus = "retrying"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusScheduled JobStatus = "scheduled"
)

// Priority represents job priority
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityNormal Priority = 1
	PriorityHigh   Priority = 2
	PriorityUrgent Priority = 3
)

// JobMetrics contains job execution metrics
type JobMetrics struct {
	TotalExecuted   int64
	TotalSucceeded  int64
	TotalFailed     int64
	TotalRetried    int64
	AverageDuration time.Duration
	LastExecuted    *time.Time
	LastSuccess     *time.Time
	LastFailure     *time.Time
}

// JobError represents a job execution error
type JobError struct {
	JobID   uuid.UUID
	JobType string
	Message string
	Cause   error
	Time    time.Time
}

func (e *JobError) Error() string {
	return e.Message
}

// JobEvent represents a job lifecycle event
type JobEvent struct {
	ID        uuid.UUID
	JobID     uuid.UUID
	Type      JobEventType
	Timestamp time.Time
	Data      map[string]interface{}
}

// JobEventType represents types of job events
type JobEventType string

const (
	JobEventCreated   JobEventType = "created"
	JobEventQueued    JobEventType = "queued"
	JobEventStarted   JobEventType = "started"
	JobEventCompleted JobEventType = "completed"
	JobEventFailed    JobEventType = "failed"
	JobEventRetried   JobEventType = "retried"
	JobEventCancelled JobEventType = "cancelled"
)
