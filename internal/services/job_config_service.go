package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/job"
)

// JobConfigurationService manages job configurations
type JobConfigurationService struct {
	mu             sync.RWMutex
	configurations map[uuid.UUID]*job.JobConfiguration
	jobService     *JobService
}

// NewJobConfigurationService creates a new job configuration service
func NewJobConfigurationService() *JobConfigurationService {
	return &JobConfigurationService{
		configurations: make(map[uuid.UUID]*job.JobConfiguration),
	}
}

// SetJobService sets the job service for scheduling
func (s *JobConfigurationService) SetJobService(jobService *JobService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobService = jobService
}

// CreateConfiguration creates a new job configuration
func (s *JobConfigurationService) CreateConfiguration(config *job.JobConfiguration) error {
	if err := s.ValidateConfiguration(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	s.configurations[config.ID] = config
	return nil
}

// UpdateConfiguration updates an existing job configuration
func (s *JobConfigurationService) UpdateConfiguration(id uuid.UUID, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, exists := s.configurations[id]
	if !exists {
		return errors.New("configuration not found")
	}

	// Apply updates
	if enabled, ok := updates["enabled"].(bool); ok {
		config.Enabled = enabled
	}
	if description, ok := updates["description"].(string); ok {
		config.Description = description
	}
	if schedule, ok := updates["schedule"].(job.JobScheduleConfig); ok {
		config.Schedule = schedule
	}
	if strategy, ok := updates["strategy"].(job.JobStrategyConfig); ok {
		config.Strategy = strategy
	}
	if parameters, ok := updates["parameters"].(map[string]interface{}); ok {
		config.Parameters = parameters
	}

	config.UpdatedAt = time.Now()
	return nil
}

// DeleteConfiguration deletes a job configuration
func (s *JobConfigurationService) DeleteConfiguration(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configurations[id]; !exists {
		return errors.New("configuration not found")
	}

	delete(s.configurations, id)
	return nil
}

// GetConfiguration retrieves a job configuration by ID
func (s *JobConfigurationService) GetConfiguration(id uuid.UUID) (*job.JobConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.configurations[id]
	if !exists {
		return nil, errors.New("configuration not found")
	}

	return config, nil
}

// ListConfigurations lists configurations with optional filters
func (s *JobConfigurationService) ListConfigurations(filters map[string]interface{}) ([]*job.JobConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*job.JobConfiguration

	for _, config := range s.configurations {
		match := true

		// Apply filters
		if jobType, ok := filters["type"].(string); ok && config.Type != jobType {
			match = false
		}
		if enabled, ok := filters["enabled"].(bool); ok && config.Enabled != enabled {
			match = false
		}

		if match {
			result = append(result, config)
		}
	}

	return result, nil
}

// EnableConfiguration enables a job configuration
func (s *JobConfigurationService) EnableConfiguration(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, exists := s.configurations[id]
	if !exists {
		return errors.New("configuration not found")
	}

	config.Enabled = true
	config.UpdatedAt = time.Now()
	return nil
}

// DisableConfiguration disables a job configuration
func (s *JobConfigurationService) DisableConfiguration(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, exists := s.configurations[id]
	if !exists {
		return errors.New("configuration not found")
	}

	config.Enabled = false
	config.UpdatedAt = time.Now()
	return nil
}

// ValidateConfiguration validates a job configuration
func (s *JobConfigurationService) ValidateConfiguration(config *job.JobConfiguration) error {
	if config.Name == "" {
		return errors.New("name is required")
	}
	if config.Type == "" {
		return errors.New("type is required")
	}

	// Validate schedule
	switch config.Schedule.Type {
	case job.ScheduleTypeCron:
		if config.Schedule.CronExpr == "" {
			return errors.New("cron expression is required for cron schedule")
		}
		// TODO: Validate cron expression format
	case job.ScheduleTypeInterval:
		if config.Schedule.Interval <= 0 {
			return errors.New("interval must be positive")
		}
	case job.ScheduleTypeDaily, job.ScheduleTypeWeekly, job.ScheduleTypeMonthly:
		// These are valid
	case "":
		// Default to once if not specified
		config.Schedule.Type = job.ScheduleTypeOnce
	default:
		return fmt.Errorf("invalid schedule type: %s", config.Schedule.Type)
	}

	// Validate strategy
	if config.Strategy.Timeout <= 0 {
		config.Strategy.Timeout = 5 * time.Minute // Default timeout
	}
	if config.Strategy.Priority < job.PriorityLow || config.Strategy.Priority > job.PriorityUrgent {
		config.Strategy.Priority = job.PriorityNormal
	}
	if config.Strategy.MaxConcurrency <= 0 {
		config.Strategy.MaxConcurrency = 1
	}

	return nil
}

// ScheduleConfiguredJob schedules a configured job for execution
func (s *JobConfigurationService) ScheduleConfiguredJob(configID uuid.UUID) error {
	s.mu.RLock()
	config, exists := s.configurations[configID]
	s.mu.RUnlock()

	if !exists {
		return errors.New("configuration not found")
	}

	if !config.Enabled {
		return errors.New("configuration is disabled")
	}

	if s.jobService == nil {
		return errors.New("job service not configured")
	}

	// Create job based on configuration
	ctx := context.Background()

	// Determine next run time based on schedule
	var runAt time.Time
	switch config.Schedule.Type {
	case job.ScheduleTypeOnce:
		runAt = time.Now()
	case job.ScheduleTypeInterval:
		runAt = time.Now().Add(config.Schedule.Interval)
	case job.ScheduleTypeDaily:
		// Schedule for next occurrence at specified time
		now := time.Now()
		runAt = time.Date(now.Year(), now.Month(), now.Day()+1,
			config.Schedule.TimeOfDay.Hour, config.Schedule.TimeOfDay.Minute, 0, 0, now.Location())
	case job.ScheduleTypeCron:
		// For cron, we'd need a cron parser - for now just schedule immediately
		runAt = time.Now().Add(1 * time.Minute)
	default:
		runAt = time.Now()
	}

	// Schedule the job
	_, err := s.jobService.ScheduleJob(ctx, config.Type, config.Parameters, runAt, config.Strategy.Priority)
	return err
}

// GetScheduledJobs returns all scheduled job configurations
func (s *JobConfigurationService) GetScheduledJobs() ([]*job.JobConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var scheduled []*job.JobConfiguration
	for _, config := range s.configurations {
		if config.Enabled {
			scheduled = append(scheduled, config)
		}
	}

	return scheduled, nil
}
