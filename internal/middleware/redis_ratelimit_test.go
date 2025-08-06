package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/victoralfred/um_sys/internal/domain/ratelimit"
	"github.com/victoralfred/um_sys/internal/middleware"
)

// mockRateLimiter implements ratelimit.RateLimiter for testing
type mockRateLimiter struct {
	results map[string]*ratelimit.RateLimitResult
	errors  map[string]error
}

func newMockRateLimiter() *mockRateLimiter {
	return &mockRateLimiter{
		results: make(map[string]*ratelimit.RateLimitResult),
		errors:  make(map[string]error),
	}
}

func (m *mockRateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (*ratelimit.RateLimitResult, error) {
	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	if result, exists := m.results[key]; exists {
		return result, nil
	}

	// Default: allow with full remaining
	return &ratelimit.RateLimitResult{
		Allowed:    true,
		Limit:      limit,
		Remaining:  limit - 1,
		ResetTime:  time.Now().Add(window),
		RetryAfter: 0,
	}, nil
}

func (m *mockRateLimiter) Reset(ctx context.Context, key string) error {
	delete(m.results, key)
	delete(m.errors, key)
	return nil
}

func (m *mockRateLimiter) GetStatus(ctx context.Context, key string, limit int, window time.Duration) (*ratelimit.RateLimitResult, error) {
	return m.Check(ctx, key, limit, window)
}

func (m *mockRateLimiter) setResult(key string, result *ratelimit.RateLimitResult) {
	m.results[key] = result
}

func (m *mockRateLimiter) setError(key string, err error) {
	m.errors[key] = err
}

func TestRedisRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &ratelimit.RateLimitConfig{
		Global: struct {
			Limit  int
			Window time.Duration
		}{Limit: 1000, Window: time.Hour},
		PerUser: struct {
			Limit  int
			Window time.Duration
		}{Limit: 100, Window: time.Hour},
		PerIP: struct {
			Limit  int
			Window time.Duration
		}{Limit: 60, Window: time.Hour},
	}

	t.Run("allows request under limit", func(t *testing.T) {
		limiter := newMockRateLimiter()

		router := gin.New()
		router.Use(middleware.RedisRateLimit(limiter, config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("X-RateLimit-Limit"), "60") // IP limit
		assert.Contains(t, w.Header().Get("X-RateLimit-Remaining"), "59")
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	})

	t.Run("blocks request over global limit", func(t *testing.T) {
		limiter := newMockRateLimiter()

		// Set global limit exceeded
		limiter.setResult("global", &ratelimit.RateLimitResult{
			Allowed:    false,
			Limit:      1000,
			Remaining:  0,
			ResetTime:  time.Now().Add(time.Hour),
			RetryAfter: 30 * time.Second,
		})

		router := gin.New()
		router.Use(middleware.RedisRateLimit(limiter, config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "GLOBAL_RATE_LIMIT_EXCEEDED")
		assert.Equal(t, "30", w.Header().Get("Retry-After"))
	})

	t.Run("blocks request over IP limit", func(t *testing.T) {
		limiter := newMockRateLimiter()

		// Set IP limit exceeded
		clientIP := "192.168.1.1"
		limiter.setResult("ip:"+clientIP, &ratelimit.RateLimitResult{
			Allowed:    false,
			Limit:      60,
			Remaining:  0,
			ResetTime:  time.Now().Add(time.Hour),
			RetryAfter: 60 * time.Second,
		})

		router := gin.New()
		router.Use(middleware.RedisRateLimit(limiter, config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", clientIP)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w.Body.String(), "60") // limit
		assert.Equal(t, "60", w.Header().Get("Retry-After"))
	})

	t.Run("uses user limits when authenticated", func(t *testing.T) {
		limiter := newMockRateLimiter()

		router := gin.New()
		// Simulate authentication middleware that sets user_id
		router.Use(func(c *gin.Context) {
			c.Set("user_id", "user-123")
			c.Next()
		})
		router.Use(middleware.RedisRateLimit(limiter, config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("X-RateLimit-Limit"), "100") // User limit
	})

	t.Run("handles rate limiter errors gracefully", func(t *testing.T) {
		limiter := newMockRateLimiter()
		limiter.setError("global", assert.AnError)

		router := gin.New()
		router.Use(middleware.RedisRateLimit(limiter, config))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "RATE_LIMIT_ERROR")
	})
}

func TestPerEndpointRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	endpointLimits := map[string]struct {
		Limit  int
		Window time.Duration
	}{
		"/auth/login": {Limit: 5, Window: time.Minute},
		"/api/upload": {Limit: 10, Window: time.Hour},
	}

	t.Run("applies endpoint-specific limits", func(t *testing.T) {
		limiter := newMockRateLimiter()

		router := gin.New()
		router.Use(middleware.PerEndpointRateLimit(limiter, endpointLimits))
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("POST", "/auth/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("X-RateLimit-Limit"), "5")
	})

	t.Run("blocks requests over endpoint limit", func(t *testing.T) {
		limiter := newMockRateLimiter()

		// Set endpoint limit exceeded
		clientIP := "192.168.1.1"
		endpointKey := "endpoint:/auth/login:ip:" + clientIP
		limiter.setResult(endpointKey, &ratelimit.RateLimitResult{
			Allowed:    false,
			Limit:      5,
			Remaining:  0,
			ResetTime:  time.Now().Add(time.Minute),
			RetryAfter: 30 * time.Second,
		})

		router := gin.New()
		router.Use(middleware.PerEndpointRateLimit(limiter, endpointLimits))
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.Header.Set("X-Forwarded-For", clientIP)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "ENDPOINT_RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w.Body.String(), "/auth/login")
	})

	t.Run("skips endpoints without specific limits", func(t *testing.T) {
		limiter := newMockRateLimiter()

		router := gin.New()
		router.Use(middleware.PerEndpointRateLimit(limiter, endpointLimits))
		router.GET("/unmonitored", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/unmonitored", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Should not have rate limit headers
		assert.Empty(t, w.Header().Get("X-RateLimit-Limit"))
	})

	t.Run("uses user-specific keys when authenticated", func(t *testing.T) {
		limiter := newMockRateLimiter()

		router := gin.New()
		// Simulate authentication middleware
		router.Use(func(c *gin.Context) {
			c.Set("user_id", "user-456")
			c.Next()
		})
		router.Use(middleware.PerEndpointRateLimit(limiter, endpointLimits))
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("POST", "/auth/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// The mock limiter would have been called with key "endpoint:/auth/login:user:user-456"
	})
}
