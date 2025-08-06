package job

import (
	"time"

	"github.com/google/uuid"
)

// JobConfiguration represents a configurable job
type JobConfiguration struct {
	ID          uuid.UUID                 `json:"id"`
	Name        string                    `json:"name"`
	Type        string                    `json:"type"`
	Description string                    `json:"description"`
	Enabled     bool                      `json:"enabled"`
	Schedule    JobScheduleConfig         `json:"schedule"`
	Strategy    JobStrategyConfig         `json:"strategy"`
	Parameters  map[string]interface{}    `json:"parameters"`
	Metadata    map[string]interface{}    `json:"metadata,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	CreatedBy   uuid.UUID                 `json:"created_by"`
	UpdatedBy   uuid.UUID                 `json:"updated_by"`
}

// TimeOfDay represents a specific time of day
type TimeOfDay struct {
	Hour   int `json:"hour"`   // 0-23
	Minute int `json:"minute"` // 0-59
}

// JobScheduleConfig defines when and how often a job runs
type JobScheduleConfig struct {
	Type       ScheduleType   `json:"type"`
	CronExpr   string         `json:"cron_expr,omitempty"`
	Interval   time.Duration  `json:"interval,omitempty"`
	StartTime  *time.Time     `json:"start_time,omitempty"`
	EndTime    *time.Time     `json:"end_time,omitempty"`
	Timezone   string         `json:"timezone,omitempty"`
	DaysOfWeek []int          `json:"days_of_week,omitempty"`
	TimeOfDay  *TimeOfDay     `json:"time_of_day,omitempty"`
}

// JobStrategyConfig defines how a job should be executed
type JobStrategyConfig struct {
	RetryStrategy    RetryConfig      `json:"retry"`
	Timeout          time.Duration    `json:"timeout"`
	Priority         Priority         `json:"priority"`
	MaxConcurrency   int              `json:"max_concurrency"`
	RateLimitPerSec  int              `json:"rate_limit_per_sec,omitempty"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries       int           `json:"max_retries"`
	InitialDelay     time.Duration `json:"initial_delay"`
	MaxDelay         time.Duration `json:"max_delay"`
	BackoffType      BackoffType   `json:"backoff_type"`
	RetryableErrors  []string      `json:"retryable_errors,omitempty"`
}

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	ScheduleTypeOnce      ScheduleType = "once"
	ScheduleTypeCron      ScheduleType = "cron"
	ScheduleTypeInterval  ScheduleType = "interval"
	ScheduleTypeDaily     ScheduleType = "daily"
	ScheduleTypeWeekly    ScheduleType = "weekly"
	ScheduleTypeMonthly   ScheduleType = "monthly"
)

// BackoffType represents the backoff strategy
type BackoffType string

const (
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeExponential BackoffType = "exponential"
	BackoffTypeFixed       BackoffType = "fixed"
)

// SoftDeleteCleanupConfig specific configuration for soft delete cleanup jobs
type SoftDeleteCleanupConfig struct {
	JobConfiguration
	TargetEntity    string        `json:"target_entity"`    // "user", "configuration", "all"
	RetentionPeriod time.Duration `json:"retention_period"`  // How long to keep soft deleted items
	BatchSize       int           `json:"batch_size"`        // Number of items to process per batch
	DryRun          bool          `json:"dry_run"`           // If true, only report what would be deleted
}

// JobConfigurationRepository handles persistence of job configurations
type JobConfigurationRepository interface {
	Create(config *JobConfiguration) error
	Update(config *JobConfiguration) error
	Delete(id uuid.UUID) error
	GetByID(id uuid.UUID) (*JobConfiguration, error)
	GetByType(jobType string) ([]*JobConfiguration, error)
	GetEnabled() ([]*JobConfiguration, error)
	GetAll() ([]*JobConfiguration, error)
}

// JobConfigurationService manages job configurations
type JobConfigurationService interface {
	CreateConfiguration(config *JobConfiguration) error
	UpdateConfiguration(id uuid.UUID, updates map[string]interface{}) error
	DeleteConfiguration(id uuid.UUID) error
	GetConfiguration(id uuid.UUID) (*JobConfiguration, error)
	ListConfigurations(filters map[string]interface{}) ([]*JobConfiguration, error)
	EnableConfiguration(id uuid.UUID) error
	DisableConfiguration(id uuid.UUID) error
	ValidateConfiguration(config *JobConfiguration) error
	ScheduleConfiguredJob(configID uuid.UUID) error
	GetScheduledJobs() ([]*JobConfiguration, error)
}

// SoftDeleteHandler handles soft delete cleanup
type SoftDeleteHandler interface {
	JobHandler
	SetConfiguration(config *SoftDeleteCleanupConfig) error
	GetDeletedItems(entity string, olderThan time.Time, limit int) ([]interface{}, error)
	PermanentlyDelete(items []interface{}) error
	GetStatistics() (*SoftDeleteStats, error)
}

// SoftDeleteStats contains statistics about soft delete cleanup
type SoftDeleteStats struct {
	Entity         string    `json:"entity"`
	TotalDeleted   int64     `json:"total_deleted"`
	LastRun        time.Time `json:"last_run"`
	NextScheduled  time.Time `json:"next_scheduled"`
	PendingCount   int64     `json:"pending_count"`
	OldestPending  time.Time `json:"oldest_pending"`
}