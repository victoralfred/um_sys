package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/victoralfred/um_sys/internal/domain/ratelimit"
)

// RedisRateLimit creates a Redis-based rate limiting middleware
func RedisRateLimit(limiter ratelimit.RateLimiter, config *ratelimit.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Determine rate limiting key and limits based on context
		var key string
		var limit int
		var window time.Duration

		// Get user ID from context if authenticated
		userID, hasUser := c.Get("user_id")
		clientIP := c.ClientIP()

		// Use different limits based on authentication status
		if hasUser {
			key = fmt.Sprintf("user:%v", userID)
			limit = config.PerUser.Limit
			window = config.PerUser.Window
		} else {
			key = fmt.Sprintf("ip:%s", clientIP)
			limit = config.PerIP.Limit
			window = config.PerIP.Window
		}

		// Also check global rate limit
		globalKey := "global"
		globalResult, err := limiter.Check(c.Request.Context(), globalKey, config.Global.Limit, config.Global.Window)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_ERROR",
					"message": "Rate limiting service unavailable",
				},
			})
			c.Abort()
			return
		}

		if !globalResult.Allowed {
			setRateLimitHeaders(c, globalResult)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "GLOBAL_RATE_LIMIT_EXCEEDED",
					"message":     "Global rate limit exceeded",
					"retry_after": int(globalResult.RetryAfter.Seconds()),
				},
			})
			c.Abort()
			return
		}

		// Check specific rate limit (user or IP)
		result, err := limiter.Check(c.Request.Context(), key, limit, window)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_ERROR",
					"message": "Rate limiting service unavailable",
				},
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		setRateLimitHeaders(c, result)

		// Store in context for rate limit status endpoint
		c.Set("rate_limit", result.Limit)
		c.Set("rate_remaining", result.Remaining)
		c.Set("rate_reset", result.ResetTime.Unix())

		if !result.Allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "RATE_LIMIT_EXCEEDED",
					"message":     "Rate limit exceeded",
					"limit":       result.Limit,
					"remaining":   result.Remaining,
					"reset_at":    result.ResetTime.Unix(),
					"retry_after": int(result.RetryAfter.Seconds()),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PerEndpointRateLimit creates endpoint-specific rate limiting
func PerEndpointRateLimit(limiter ratelimit.RateLimiter, endpointLimits map[string]struct {
	Limit  int
	Window time.Duration
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		endpoint := c.FullPath()
		endpointConfig, exists := endpointLimits[endpoint]
		if !exists {
			c.Next()
			return
		}

		// Create endpoint-specific key
		var key string
		if userID, hasUser := c.Get("user_id"); hasUser {
			key = fmt.Sprintf("endpoint:%s:user:%v", endpoint, userID)
		} else {
			key = fmt.Sprintf("endpoint:%s:ip:%s", endpoint, c.ClientIP())
		}

		result, err := limiter.Check(c.Request.Context(), key, endpointConfig.Limit, endpointConfig.Window)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_ERROR",
					"message": "Rate limiting service unavailable",
				},
			})
			c.Abort()
			return
		}

		setRateLimitHeaders(c, result)

		if !result.Allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "ENDPOINT_RATE_LIMIT_EXCEEDED",
					"message":     fmt.Sprintf("Rate limit exceeded for endpoint %s", endpoint),
					"endpoint":    endpoint,
					"limit":       result.Limit,
					"remaining":   result.Remaining,
					"reset_at":    result.ResetTime.Unix(),
					"retry_after": int(result.RetryAfter.Seconds()),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func setRateLimitHeaders(c *gin.Context, result *ratelimit.RateLimitResult) {
	c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))

	if result.RetryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	}
}
