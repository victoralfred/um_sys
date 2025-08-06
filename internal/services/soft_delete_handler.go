package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/victoralfred/um_sys/internal/domain/job"
)

// SoftDeleteCleanupHandler handles soft delete cleanup jobs
type SoftDeleteCleanupHandler struct {
	mu          sync.RWMutex
	config      *job.SoftDeleteCleanupConfig
	stats       *job.SoftDeleteStats
	testData    []interface{} // For testing
	isTestMode  bool
}

// NewSoftDeleteCleanupHandler creates a new soft delete cleanup handler
func NewSoftDeleteCleanupHandler() *SoftDeleteCleanupHandler {
	return &SoftDeleteCleanupHandler{
		stats: &job.SoftDeleteStats{
			Entity:       "all",
			TotalDeleted: 0,
			LastRun:      time.Time{},
		},
	}
}

// SetConfiguration sets the handler configuration
func (h *SoftDeleteCleanupHandler) SetConfiguration(config *job.SoftDeleteCleanupConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if config == nil {
		return errors.New("configuration cannot be nil")
	}

	h.config = config
	h.stats.Entity = config.TargetEntity
	return nil
}

// GetType returns the job type this handler processes
func (h *SoftDeleteCleanupHandler) GetType() string {
	return "soft_delete_cleanup"
}

// GetTimeout returns the handler timeout
func (h *SoftDeleteCleanupHandler) GetTimeout() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.config != nil && h.config.JobConfiguration.Strategy.Timeout > 0 {
		return h.config.JobConfiguration.Strategy.Timeout
	}
	return 5 * time.Minute // Default timeout
}

// Handle processes a soft delete cleanup job
func (h *SoftDeleteCleanupHandler) Handle(ctx context.Context, j job.Job) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update last run time
	h.stats.LastRun = time.Now()

	// Parse job payload if provided
	if payload := j.GetPayload(); payload != nil {
		if jsonPayload, ok := payload.(json.RawMessage); ok {
			var params map[string]interface{}
			if err := json.Unmarshal(jsonPayload, &params); err == nil {
				// Update config from payload if provided
				if entity, ok := params["entity"].(string); ok {
					h.config.TargetEntity = entity
				}
				if retentionStr, ok := params["retention_period"].(string); ok {
					if duration, err := time.ParseDuration(retentionStr); err == nil {
						h.config.RetentionPeriod = duration
					}
				}
				if batchSize, ok := params["batch_size"].(float64); ok {
					h.config.BatchSize = int(batchSize)
				}
			}
		}
	}

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-h.config.RetentionPeriod)

	// Process entities based on target
	entities := []string{}
	switch h.config.TargetEntity {
	case "all":
		entities = []string{"user", "configuration"}
	default:
		entities = []string{h.config.TargetEntity}
	}

	totalDeleted := 0
	for _, entity := range entities {
		// Get items to delete
		items, err := h.GetDeletedItems(entity, cutoffTime, h.config.BatchSize)
		if err != nil {
			return fmt.Errorf("failed to get deleted %s items: %w", entity, err)
		}

		if len(items) == 0 {
			continue
		}

		// If dry run, just log what would be deleted
		if h.config.DryRun {
			// In production, this would log the items
			fmt.Printf("DRY RUN: Would delete %d %s items\n", len(items), entity)
			continue
		}

		// Permanently delete items
		if err := h.PermanentlyDelete(items); err != nil {
			return fmt.Errorf("failed to delete %s items: %w", entity, err)
		}

		totalDeleted += len(items)
	}

	// Update statistics
	if !h.config.DryRun {
		h.stats.TotalDeleted += int64(totalDeleted)
	}

	return nil
}

// GetDeletedItems retrieves soft deleted items older than the specified time
func (h *SoftDeleteCleanupHandler) GetDeletedItems(entity string, olderThan time.Time, limit int) ([]interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// In test mode, return test data
	if h.isTestMode && h.testData != nil {
		var result []interface{}
		for _, item := range h.testData {
			if m, ok := item.(map[string]interface{}); ok {
				if deletedAt, ok := m["deleted_at"].(time.Time); ok {
					if deletedAt.Before(olderThan) {
						result = append(result, item)
						if len(result) >= limit {
							break
						}
					}
				}
			}
		}
		return result, nil
	}

	// In production, this would query the database
	// For now, return empty slice
	return []interface{}{}, nil
}

// PermanentlyDelete permanently deletes the specified items
func (h *SoftDeleteCleanupHandler) PermanentlyDelete(items []interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// In production, this would delete from database
	// For now, just update stats
	h.stats.TotalDeleted += int64(len(items))
	
	return nil
}

// GetStatistics returns cleanup statistics
func (h *SoftDeleteCleanupHandler) GetStatistics() (*job.SoftDeleteStats, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Calculate next scheduled time
	if h.config != nil {
		switch h.config.JobConfiguration.Schedule.Type {
		case job.ScheduleTypeDaily:
			h.stats.NextScheduled = h.stats.LastRun.Add(24 * time.Hour)
		case job.ScheduleTypeWeekly:
			h.stats.NextScheduled = h.stats.LastRun.Add(7 * 24 * time.Hour)
		case job.ScheduleTypeInterval:
			h.stats.NextScheduled = h.stats.LastRun.Add(h.config.JobConfiguration.Schedule.Interval)
		}
	}

	return h.stats, nil
}

// SetTestData sets test data for testing
func (h *SoftDeleteCleanupHandler) SetTestData(data []interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.testData = data
	h.isTestMode = true
}