package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/job"
	"github.com/victoralfred/um_sys/internal/services"
)

// JobConfigHandler handles job configuration HTTP requests
type JobConfigHandler struct {
	configService *services.JobConfigurationService
	jobService    *services.JobService
}

// NewJobConfigHandler creates a new job configuration handler
func NewJobConfigHandler(configService *services.JobConfigurationService, jobService *services.JobService) *JobConfigHandler {
	return &JobConfigHandler{
		configService: configService,
		jobService:    jobService,
	}
}

// CreateJobConfigRequest represents a request to create a job configuration
type CreateJobConfigRequest struct {
	Name        string                    `json:"name" binding:"required"`
	Type        string                    `json:"type" binding:"required"`
	Description string                    `json:"description"`
	Enabled     bool                      `json:"enabled"`
	Schedule    job.JobScheduleConfig     `json:"schedule"`
	Strategy    job.JobStrategyConfig     `json:"strategy"`
	Parameters  map[string]interface{}    `json:"parameters"`
}

// UpdateJobConfigRequest represents a request to update a job configuration
type UpdateJobConfigRequest struct {
	Name        *string                   `json:"name,omitempty"`
	Description *string                   `json:"description,omitempty"`
	Enabled     *bool                     `json:"enabled,omitempty"`
	Schedule    *job.JobScheduleConfig    `json:"schedule,omitempty"`
	Strategy    *job.JobStrategyConfig    `json:"strategy,omitempty"`
	Parameters  map[string]interface{}    `json:"parameters,omitempty"`
}

// CreateConfiguration creates a new job configuration
// @Summary Create job configuration
// @Description Create a new job configuration for scheduled tasks
// @Tags Jobs
// @Accept json
// @Produce json
// @Param request body CreateJobConfigRequest true "Job configuration"
// @Success 201 {object} job.JobConfiguration
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/jobs/configurations [post]
func (h *JobConfigHandler) CreateConfiguration(c *gin.Context) {
	var req CreateJobConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (would come from auth middleware)
	userID := uuid.New() // TODO: Get from auth context

	config := &job.JobConfiguration{
		ID:          uuid.New(),
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Enabled:     req.Enabled,
		Schedule:    req.Schedule,
		Strategy:    req.Strategy,
		Parameters:  req.Parameters,
		CreatedBy:   userID,
		UpdatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.configService.CreateConfiguration(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetConfiguration retrieves a job configuration
// @Summary Get job configuration
// @Description Get a job configuration by ID
// @Tags Jobs
// @Produce json
// @Param id path string true "Configuration ID"
// @Success 200 {object} job.JobConfiguration
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/jobs/configurations/{id} [get]
func (h *JobConfigHandler) GetConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	config, err := h.configService.GetConfiguration(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfiguration updates a job configuration
// @Summary Update job configuration
// @Description Update an existing job configuration
// @Tags Jobs
// @Accept json
// @Produce json
// @Param id path string true "Configuration ID"
// @Param request body UpdateJobConfigRequest true "Update data"
// @Success 200 {object} job.JobConfiguration
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/jobs/configurations/{id} [put]
func (h *JobConfigHandler) UpdateConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req UpdateJobConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Schedule != nil {
		updates["schedule"] = *req.Schedule
	}
	if req.Strategy != nil {
		updates["strategy"] = *req.Strategy
	}
	if req.Parameters != nil {
		updates["parameters"] = req.Parameters
	}

	if err := h.configService.UpdateConfiguration(id, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, _ := h.configService.GetConfiguration(id)
	c.JSON(http.StatusOK, config)
}

// DeleteConfiguration deletes a job configuration
// @Summary Delete job configuration
// @Description Delete a job configuration
// @Tags Jobs
// @Param id path string true "Configuration ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/jobs/configurations/{id} [delete]
func (h *JobConfigHandler) DeleteConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if err := h.configService.DeleteConfiguration(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListConfigurations lists job configurations
// @Summary List job configurations
// @Description List all job configurations with optional filters
// @Tags Jobs
// @Produce json
// @Param type query string false "Filter by type"
// @Param enabled query bool false "Filter by enabled status"
// @Success 200 {array} job.JobConfiguration
// @Router /api/v1/jobs/configurations [get]
func (h *JobConfigHandler) ListConfigurations(c *gin.Context) {
	filters := make(map[string]interface{})
	
	if jobType := c.Query("type"); jobType != "" {
		filters["type"] = jobType
	}
	if enabled := c.Query("enabled"); enabled != "" {
		filters["enabled"] = enabled == "true"
	}

	configs, err := h.configService.ListConfigurations(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// EnableConfiguration enables a job configuration
// @Summary Enable job configuration
// @Description Enable a job configuration to start scheduling
// @Tags Jobs
// @Param id path string true "Configuration ID"
// @Success 200 {object} job.JobConfiguration
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/jobs/configurations/{id}/enable [post]
func (h *JobConfigHandler) EnableConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if err := h.configService.EnableConfiguration(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Schedule the job if enabled
	h.configService.ScheduleConfiguredJob(id)

	config, _ := h.configService.GetConfiguration(id)
	c.JSON(http.StatusOK, config)
}

// DisableConfiguration disables a job configuration
// @Summary Disable job configuration
// @Description Disable a job configuration to stop scheduling
// @Tags Jobs
// @Param id path string true "Configuration ID"
// @Success 200 {object} job.JobConfiguration
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/jobs/configurations/{id}/disable [post]
func (h *JobConfigHandler) DisableConfiguration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if err := h.configService.DisableConfiguration(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	config, _ := h.configService.GetConfiguration(id)
	c.JSON(http.StatusOK, config)
}

// TriggerJob manually triggers a configured job
// @Summary Trigger job manually
// @Description Manually trigger a configured job to run immediately
// @Tags Jobs
// @Param id path string true "Configuration ID"
// @Success 202 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/jobs/configurations/{id}/trigger [post]
func (h *JobConfigHandler) TriggerJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if err := h.configService.ScheduleConfiguredJob(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Job triggered successfully"})
}

// GetJobStatistics gets statistics for a job configuration
// @Summary Get job statistics
// @Description Get execution statistics for a job configuration
// @Tags Jobs
// @Param id path string true "Configuration ID"
// @Success 200 {object} job.SoftDeleteStats
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/jobs/configurations/{id}/stats [get]
func (h *JobConfigHandler) GetJobStatistics(c *gin.Context) {
	idStr := c.Param("id")
	configID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	config, err := h.configService.GetConfiguration(configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Get metrics from job service
	ctx := c.Request.Context()
	metrics, err := h.jobService.GetJobMetrics(ctx, config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// RegisterRoutes registers the job configuration routes
func (h *JobConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	jobs := router.Group("/jobs")
	{
		configs := jobs.Group("/configurations")
		{
			configs.POST("", h.CreateConfiguration)
			configs.GET("", h.ListConfigurations)
			configs.GET("/:id", h.GetConfiguration)
			configs.PUT("/:id", h.UpdateConfiguration)
			configs.DELETE("/:id", h.DeleteConfiguration)
			configs.POST("/:id/enable", h.EnableConfiguration)
			configs.POST("/:id/disable", h.DisableConfiguration)
			configs.POST("/:id/trigger", h.TriggerJob)
			configs.GET("/:id/stats", h.GetJobStatistics)
		}
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}