package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

type AnalyticsHandler struct {
	analyticsService analytics.AnalyticsService
}

func NewAnalyticsHandler(analyticsService analytics.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// TrackEvent tracks an analytics event
func (h *AnalyticsHandler) TrackEvent(c *gin.Context) {
	var event analytics.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	if err := h.analyticsService.TrackEvent(c.Request.Context(), &event); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "TRACK_EVENT_FAILED",
			Message: "Failed to track event",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Event tracked successfully",
	})
}

// RecordMetric records a metric value
func (h *AnalyticsHandler) RecordMetric(c *gin.Context) {
	var metric analytics.Metric
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	if err := h.analyticsService.RecordMetric(c.Request.Context(), &metric); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "RECORD_METRIC_FAILED",
			Message: "Failed to record metric",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Metric recorded successfully",
	})
}

// GetEvents retrieves events based on filter criteria
func (h *AnalyticsHandler) GetEvents(c *gin.Context) {
	filter := analytics.EventFilter{}

	// Parse user_id
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_USER_ID",
				Message: "Invalid user_id",
				Details: "user_id must be a valid UUID",
			})
			return
		}
		filter.UserID = &userID
	}

	// Parse session_id
	if sessionID := c.Query("session_id"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	// Parse types
	if types := c.QueryArray("types"); len(types) > 0 {
		eventTypes := make([]analytics.EventType, len(types))
		for i, t := range types {
			eventTypes[i] = analytics.EventType(t)
		}
		filter.Types = eventTypes
	}

	// Parse time range
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_START_TIME",
				Message: "Invalid start_time",
				Details: "start_time must be in ISO 8601 format",
			})
			return
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_END_TIME",
				Message: "Invalid end_time",
				Details: "end_time must be in ISO 8601 format",
			})
			return
		}
		filter.EndTime = &endTime
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_LIMIT",
				Message: "Invalid limit",
				Details: "limit must be a positive integer",
			})
			return
		}
		filter.Limit = limit
	} else {
		filter.Limit = 100
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_OFFSET",
				Message: "Invalid offset",
				Details: "offset must be a non-negative integer",
			})
			return
		}
		filter.Offset = offset
	}

	events, total, err := h.analyticsService.GetEvents(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "GET_EVENTS_FAILED",
			Message: "Failed to get events",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetUsageStats retrieves usage statistics
func (h *AnalyticsHandler) GetUsageStats(c *gin.Context) {
	filter := analytics.StatsFilter{
		Period: c.DefaultQuery("period", "daily"),
	}

	// Parse user_id
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_USER_ID",
				Message: "Invalid user_id",
				Details: "user_id must be a valid UUID",
			})
			return
		}
		filter.UserID = &userID
	}

	// Parse time range - default to last 7 days if not provided
	now := time.Now()
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_START_TIME",
				Message: "Invalid start_time",
				Details: "start_time must be in ISO 8601 format",
			})
			return
		}
		filter.StartTime = startTime
	} else {
		filter.StartTime = now.AddDate(0, 0, -7)
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_END_TIME",
				Message: "Invalid end_time",
				Details: "end_time must be in ISO 8601 format",
			})
			return
		}
		filter.EndTime = endTime
	} else {
		filter.EndTime = now
	}

	stats, err := h.analyticsService.GetUsageStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "GET_USAGE_STATS_FAILED",
			Message: "Failed to get usage statistics",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetDashboard retrieves dashboard data
func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	period := c.DefaultQuery("period", "daily")

	dashboardData, err := h.analyticsService.GetDashboardData(c.Request.Context(), period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "GET_DASHBOARD_FAILED",
			Message: "Failed to get dashboard data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dashboardData)
}

// GetMetrics retrieves metrics based on filter criteria
func (h *AnalyticsHandler) GetMetrics(c *gin.Context) {
	filter := analytics.MetricFilter{}

	// Parse names
	if names := c.QueryArray("names"); len(names) > 0 {
		filter.Names = names
	}

	// Parse types
	if types := c.QueryArray("types"); len(types) > 0 {
		metricTypes := make([]analytics.MetricType, len(types))
		for i, t := range types {
			metricTypes[i] = analytics.MetricType(t)
		}
		filter.Types = metricTypes
	}

	// Parse time range
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_START_TIME",
				Message: "Invalid start_time",
				Details: "start_time must be in ISO 8601 format",
			})
			return
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_END_TIME",
				Message: "Invalid end_time",
				Details: "end_time must be in ISO 8601 format",
			})
			return
		}
		filter.EndTime = &endTime
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_LIMIT",
				Message: "Invalid limit",
				Details: "limit must be a positive integer",
			})
			return
		}
		filter.Limit = limit
	} else {
		filter.Limit = 100
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_OFFSET",
				Message: "Invalid offset",
				Details: "offset must be a non-negative integer",
			})
			return
		}
		filter.Offset = offset
	}

	metrics, total, err := h.analyticsService.GetMetrics(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "GET_METRICS_FAILED",
			Message: "Failed to get metrics",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
		"total":   total,
		"limit":   filter.Limit,
		"offset":  filter.Offset,
	})
}

// ExportData exports analytics data
func (h *AnalyticsHandler) ExportData(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "csv" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_FORMAT",
			Message: "Invalid format",
			Details: "format must be either 'json' or 'csv'",
		})
		return
	}

	filter := analytics.EventFilter{}

	// Parse types
	if types := c.QueryArray("types"); len(types) > 0 {
		eventTypes := make([]analytics.EventType, len(types))
		for i, t := range types {
			eventTypes[i] = analytics.EventType(t)
		}
		filter.Types = eventTypes
	}

	// Parse time range - default to last 30 days
	now := time.Now()
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_START_TIME",
				Message: "Invalid start_time",
				Details: "start_time must be in ISO 8601 format",
			})
			return
		}
		filter.StartTime = &startTime
	} else {
		startTime := now.AddDate(0, 0, -30)
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_END_TIME",
				Message: "Invalid end_time",
				Details: "end_time must be in ISO 8601 format",
			})
			return
		}
		filter.EndTime = &endTime
	} else {
		filter.EndTime = &now
	}

	data, err := h.analyticsService.ExportData(c.Request.Context(), filter, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "EXPORT_DATA_FAILED",
			Message: "Failed to export data",
			Details: err.Error(),
		})
		return
	}

	// Set appropriate content type
	contentType := "application/json"
	if format == "csv" {
		contentType = "text/csv"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename=analytics_export."+format)
	c.Data(http.StatusOK, contentType, data)
}
