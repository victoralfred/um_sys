package services

import (
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/job"
)

// JobService manages background jobs
type JobService struct {
	mu               sync.RWMutex
	jobs             map[uuid.UUID]*job.BackgroundJob
	queues           map[string]*PriorityQueue
	handlers         map[string]job.JobHandler
	retryStrategy    job.JobRetryStrategy
	metrics          map[string]*job.JobMetrics
	schedules        map[uuid.UUID]*job.RecurringJobSchedule
	scheduledJobs    map[uuid.UUID]*job.BackgroundJob
	processingCtx    context.Context
	processingCancel context.CancelFunc
}

// NewJobService creates a new job service
func NewJobService() *JobService {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobService{
		jobs:          make(map[uuid.UUID]*job.BackgroundJob),
		queues:        make(map[string]*PriorityQueue),
		handlers:      make(map[string]job.JobHandler),
		metrics:       make(map[string]*job.JobMetrics),
		schedules:     make(map[uuid.UUID]*job.RecurringJobSchedule),
		scheduledJobs: make(map[uuid.UUID]*job.BackgroundJob),
		retryStrategy: &job.LinearRetryStrategy{
			MaxRetries: 3,
			Delay:      5 * time.Second,
		},
		processingCtx:    ctx,
		processingCancel: cancel,
	}
}

// CreateJob creates a new job
func (s *JobService) CreateJob(ctx context.Context, jobType string, payload interface{}, priority job.Priority) (job.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	j := &job.BackgroundJob{
		ID:         uuid.New(),
		Type:       jobType,
		Payload:    payloadBytes,
		Status:     job.JobStatusPending,
		Priority:   priority,
		RetryCount: 0,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	s.jobs[j.ID] = j
	return j, nil
}

// EnqueueJob adds a job to the queue
func (s *JobService) EnqueueJob(ctx context.Context, j job.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bgJob, ok := s.jobs[j.GetID()]
	if !ok {
		return errors.New("job not found")
	}

	// Get or create queue for job type
	queue, exists := s.queues[j.GetType()]
	if !exists {
		queue = NewPriorityQueue()
		s.queues[j.GetType()] = queue
	}

	// Update job status
	bgJob.Status = job.JobStatusQueued
	bgJob.UpdatedAt = time.Now()

	// Add to queue
	heap.Push(queue, &PriorityQueueItem{
		Job:      bgJob,
		Priority: int(bgJob.Priority),
	})

	return nil
}

// GetJob retrieves a job by ID
func (s *JobService) GetJob(ctx context.Context, jobID uuid.UUID) (job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	j, exists := s.jobs[jobID]
	if !exists {
		return nil, errors.New("job not found")
	}

	return j, nil
}

// RegisterHandler registers a job handler
func (s *JobService) RegisterHandler(jobType string, handler job.JobHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.handlers[jobType]; exists {
		return fmt.Errorf("handler already registered for type: %s", jobType)
	}

	s.handlers[jobType] = handler
	return nil
}

// ProcessNextJob processes the next job in the queue
func (s *JobService) ProcessNextJob(ctx context.Context, jobType string) error {
	s.mu.Lock()
	queue, exists := s.queues[jobType]
	if !exists || queue.Len() == 0 {
		s.mu.Unlock()
		return errors.New("no jobs in queue")
	}

	// Check for jobs that are ready to be processed (retry delay)
	var jobToProcess *job.BackgroundJob
	now := time.Now()
	checkedJobs := 0
	totalJobs := queue.Len()

	for checkedJobs < totalJobs && queue.Len() > 0 {
		item := heap.Pop(queue).(*PriorityQueueItem)
		j := item.Job
		checkedJobs++

		// Check if job is scheduled for future
		if j.ScheduledFor != nil && j.ScheduledFor.After(now) {
			// Put it back and continue
			heap.Push(queue, item)
			continue
		}

		// Check if it's a retry that's not ready yet
		if j.Status == job.JobStatusRetrying && j.ScheduledFor != nil && j.ScheduledFor.After(now) {
			// Put it back and continue
			heap.Push(queue, item)
			continue
		}

		jobToProcess = j
		break
	}

	if jobToProcess == nil {
		s.mu.Unlock()
		return errors.New("no jobs ready to process")
	}

	handler, exists := s.handlers[jobType]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("no handler registered for type: %s", jobType)
	}

	// Update job status
	jobToProcess.Status = job.JobStatusRunning
	jobToProcess.StartedAt = &now
	jobToProcess.UpdatedAt = now
	s.mu.Unlock()

	// Create context with timeout
	timeout := handler.GetTimeout()
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	processCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Record start in metrics
	s.recordMetricStart(jobType)
	startTime := time.Now()

	// Process the job
	err := handler.Handle(processCtx, jobToProcess)
	duration := time.Since(startTime)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		// Store the original error for retry strategy
		originalErr := err

		// Check if it's a timeout
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("job timeout after %v", timeout)
		}

		// Handle failure
		jobToProcess.LastError = err.Error()
		jobToProcess.FailedAt = &now

		// Check if should retry (use original error for strategy)
		if s.retryStrategy != nil && s.retryStrategy.ShouldRetry(jobToProcess, originalErr) {
			jobToProcess.RetryCount++
			jobToProcess.Status = job.JobStatusRetrying
			nextRetry := s.retryStrategy.NextRetryTime(jobToProcess)
			jobToProcess.ScheduledFor = &nextRetry

			// Re-queue for retry
			heap.Push(queue, &PriorityQueueItem{
				Job:      jobToProcess,
				Priority: int(jobToProcess.Priority),
			})

			s.recordMetricRetry(jobType)
		} else {
			jobToProcess.Status = job.JobStatusFailed
			s.recordMetricFailure(jobType, duration)
		}

		jobToProcess.UpdatedAt = time.Now()
		return err
	}

	// Success
	completedAt := time.Now()
	jobToProcess.Status = job.JobStatusCompleted
	jobToProcess.CompletedAt = &completedAt
	jobToProcess.UpdatedAt = completedAt

	s.recordMetricSuccess(jobType, duration)

	return nil
}

// ScheduleJob schedules a job for future execution
func (s *JobService) ScheduleJob(ctx context.Context, jobType string, payload interface{}, runAt time.Time, priority job.Priority) (job.Job, error) {
	j, err := s.CreateJob(ctx, jobType, payload, priority)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bgJob := s.jobs[j.GetID()]
	bgJob.Status = job.JobStatusScheduled
	bgJob.ScheduledFor = &runAt
	bgJob.UpdatedAt = time.Now()

	s.scheduledJobs[bgJob.ID] = bgJob

	return bgJob, nil
}

// GetScheduledJobs returns all scheduled jobs
func (s *JobService) GetScheduledJobs(ctx context.Context) ([]job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var scheduled []job.Job
	for _, j := range s.scheduledJobs {
		if j.Status == job.JobStatusScheduled {
			scheduled = append(scheduled, j)
		}
	}

	return scheduled, nil
}

// SetRetryStrategy sets the retry strategy
func (s *JobService) SetRetryStrategy(strategy job.JobRetryStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retryStrategy = strategy
}

// GetJobMetrics returns metrics for a job type
func (s *JobService) GetJobMetrics(ctx context.Context, jobType string) (*job.JobMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics, exists := s.metrics[jobType]
	if !exists {
		return &job.JobMetrics{}, nil
	}

	return metrics, nil
}

// CancelJob cancels a job
func (s *JobService) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, exists := s.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}

	j.Status = job.JobStatusCancelled
	j.UpdatedAt = time.Now()

	// Remove from scheduled jobs if present
	delete(s.scheduledJobs, jobID)

	return nil
}

// GetJobsByStatus returns jobs with the specified status
func (s *JobService) GetJobsByStatus(ctx context.Context, status job.JobStatus) ([]job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var jobs []job.Job
	for _, j := range s.jobs {
		if j.Status == status {
			jobs = append(jobs, j)
		}
	}

	return jobs, nil
}

// ClearQueue clears all jobs from a queue
func (s *JobService) ClearQueue(ctx context.Context, jobType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, exists := s.queues[jobType]
	if !exists {
		return nil
	}

	// Clear the queue
	for queue.Len() > 0 {
		heap.Pop(queue)
	}

	return nil
}

// GetQueueSize returns the size of a queue
func (s *JobService) GetQueueSize(ctx context.Context, jobType string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queue, exists := s.queues[jobType]
	if !exists {
		return 0, nil
	}

	return queue.Len(), nil
}

// ScheduleRecurringJob schedules a recurring job
func (s *JobService) ScheduleRecurringJob(ctx context.Context, jobType string, payload interface{}, cronExpr string) (*job.RecurringJobSchedule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	schedule := &job.RecurringJobSchedule{
		ID:        uuid.New(),
		JobType:   jobType,
		CronExpr:  cronExpr,
		Payload:   payloadBytes,
		Enabled:   true,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.schedules[schedule.ID] = schedule
	return schedule, nil
}

// GetRecurringSchedules returns all recurring schedules
func (s *JobService) GetRecurringSchedules(ctx context.Context) ([]*job.RecurringJobSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var schedules []*job.RecurringJobSchedule
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// GetRecurringSchedule returns a specific recurring schedule
func (s *JobService) GetRecurringSchedule(ctx context.Context, scheduleID uuid.UUID) (*job.RecurringJobSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return nil, errors.New("schedule not found")
	}

	return schedule, nil
}

// DisableRecurringJob disables a recurring job
func (s *JobService) DisableRecurringJob(ctx context.Context, scheduleID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return errors.New("schedule not found")
	}

	schedule.Enabled = false
	schedule.UpdatedAt = time.Now()

	return nil
}

// Metric recording helpers
func (s *JobService) recordMetricStart(jobType string) {
	if _, exists := s.metrics[jobType]; !exists {
		s.metrics[jobType] = &job.JobMetrics{}
	}
}

func (s *JobService) recordMetricSuccess(jobType string, duration time.Duration) {
	metrics := s.metrics[jobType]
	if metrics == nil {
		metrics = &job.JobMetrics{}
		s.metrics[jobType] = metrics
	}

	metrics.TotalExecuted++
	metrics.TotalSucceeded++
	now := time.Now()
	metrics.LastExecuted = &now
	metrics.LastSuccess = &now

	// Update average duration
	if metrics.AverageDuration == 0 {
		metrics.AverageDuration = duration
	} else {
		// Simple moving average
		metrics.AverageDuration = (metrics.AverageDuration + duration) / 2
	}
}

func (s *JobService) recordMetricFailure(jobType string, duration time.Duration) {
	metrics := s.metrics[jobType]
	if metrics == nil {
		metrics = &job.JobMetrics{}
		s.metrics[jobType] = metrics
	}

	metrics.TotalExecuted++
	metrics.TotalFailed++
	now := time.Now()
	metrics.LastExecuted = &now
	metrics.LastFailure = &now
}

func (s *JobService) recordMetricRetry(jobType string) {
	metrics := s.metrics[jobType]
	if metrics == nil {
		metrics = &job.JobMetrics{}
		s.metrics[jobType] = metrics
	}

	metrics.TotalRetried++
}

// PriorityQueue implements a priority queue for jobs
type PriorityQueue []*PriorityQueueItem

type PriorityQueueItem struct {
	Job      *job.BackgroundJob
	Priority int
	index    int
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{}
	heap.Init(pq)
	return pq
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Higher priority comes first
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}
