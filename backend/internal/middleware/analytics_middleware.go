package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

// AnalyticsMiddleware creates middleware to automatically track API calls
func AnalyticsMiddleware(analyticsService analytics.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate response time
		responseTime := time.Since(start)

		// Extract user information from context if available
		var userID *uuid.UUID
		if userIDStr, exists := c.Get("user_id"); exists {
			if uid, ok := userIDStr.(string); ok {
				if parsedID, err := uuid.Parse(uid); err == nil {
					userID = &parsedID
				}
			} else if uid, ok := userIDStr.(uuid.UUID); ok {
				userID = &uid
			}
		}

		// Extract session ID if available
		var sessionID *string
		if sid, exists := c.Get("session_id"); exists {
			if sidStr, ok := sid.(string); ok {
				sessionID = &sidStr
			}
		}

		// Track the API call in the background
		go func() {
			_ = analyticsService.TrackAPICall(
				c.Request.Context(),
				c.Request.Method,
				c.Request.URL.Path,
				c.Writer.Status(),
				responseTime,
				userID,
				sessionID,
			)
			// Errors are ignored in background processing
		}()
	}
}

// TrackEvent is a helper function to manually track events from handlers
func TrackEvent(c *gin.Context, analyticsService analytics.AnalyticsService, eventType analytics.EventType, properties map[string]interface{}) {
	// Extract user information from context if available
	var userID *uuid.UUID
	if userIDStr, exists := c.Get("user_id"); exists {
		if uid, ok := userIDStr.(string); ok {
			if parsedID, err := uuid.Parse(uid); err == nil {
				userID = &parsedID
			}
		} else if uid, ok := userIDStr.(uuid.UUID); ok {
			userID = &uid
		}
	}

	// Extract session ID if available
	var sessionID *string
	if sid, exists := c.Get("session_id"); exists {
		if sidStr, ok := sid.(string); ok {
			sessionID = &sidStr
		}
	}

	// Create event
	event := &analytics.Event{
		ID:         uuid.New(),
		Type:       eventType,
		UserID:     userID,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
		Properties: properties,
		Context: &analytics.EventContext{
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			Referrer:  c.GetHeader("Referer"),
			Path:      c.Request.URL.Path,
			Method:    c.Request.Method,
		},
		CreatedAt: time.Now(),
	}

	// Track event in background
	go func() {
		_ = analyticsService.TrackEvent(c.Request.Context(), event)
		// Errors are ignored in background processing
	}()
}

// IncrementCounter is a helper function to increment counter metrics
func IncrementCounter(analyticsService analytics.AnalyticsService, name string, labels map[string]string, value float64) {
	go func() {
		_ = analyticsService.IncrementCounter(context.TODO(), name, labels, value)
		// Errors are ignored in background processing
	}()
}

// SetGauge is a helper function to set gauge metrics
func SetGauge(analyticsService analytics.AnalyticsService, name string, labels map[string]string, value float64) {
	go func() {
		_ = analyticsService.SetGauge(context.TODO(), name, labels, value)
		// Errors are ignored in background processing
	}()
}

// RecordHistogram is a helper function to record histogram metrics
func RecordHistogram(analyticsService analytics.AnalyticsService, name string, labels map[string]string, value float64) {
	go func() {
		_ = analyticsService.RecordHistogram(context.TODO(), name, labels, value)
		// Errors are ignored in background processing
	}()
}
