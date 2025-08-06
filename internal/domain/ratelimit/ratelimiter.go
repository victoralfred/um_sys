package ratelimit

import (
	"context"
	"time"
)

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed    bool
	Limit      int
	Remaining  int
	ResetTime  time.Time
	RetryAfter time.Duration
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Check checks if a request should be allowed and updates counters
	Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)

	// Reset resets the rate limit for a key
	Reset(ctx context.Context, key string) error

	// GetStatus returns current rate limit status without updating counters
	GetStatus(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Global struct {
		Limit  int
		Window time.Duration
	}
	PerUser struct {
		Limit  int
		Window time.Duration
	}
	PerIP struct {
		Limit  int
		Window time.Duration
	}
}

// DefaultConfig returns default rate limiting configuration
func DefaultConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Global: struct {
			Limit  int
			Window time.Duration
		}{
			Limit:  1000,
			Window: time.Hour,
		},
		PerUser: struct {
			Limit  int
			Window time.Duration
		}{
			Limit:  100,
			Window: time.Hour,
		},
		PerIP: struct {
			Limit  int
			Window time.Duration
		}{
			Limit:  60,
			Window: time.Minute,
		},
	}
}
