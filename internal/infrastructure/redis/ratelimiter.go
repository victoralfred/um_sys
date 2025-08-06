package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/victoralfred/um_sys/internal/domain/ratelimit"
)

const (
	rateLimitKeyPrefix = "rate_limit:"
)

// RateLimiter implements rate limiting using Redis sliding window
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new Redis rate limiter
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
	}
}

// Check implements sliding window rate limiting using Redis sorted sets
func (r *RateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (*ratelimit.RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	redisKey := rateLimitKeyPrefix + key

	// Use Lua script for atomic operations
	luaScript := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local ttl = tonumber(ARGV[4])

		-- Remove expired entries
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current requests in window
		local current = redis.call('ZCARD', key)

		-- Check if under limit
		if current < limit then
			-- Add current request with unique score to avoid duplicates
			redis.call('ZADD', key, now, now .. ':' .. math.random())
			-- Set expiration
			redis.call('EXPIRE', key, ttl)
			local new_count = current + 1
			return {1, new_count, limit - new_count}
		else
			-- Get oldest entry for reset time calculation
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local reset_time = now + (ttl * 1000)
			if #oldest > 0 then
				reset_time = tonumber(oldest[2]) + (ttl * 1000)
			end
			return {0, current, 0, reset_time}
		end
	`

	// Convert times to Unix timestamps (milliseconds)
	windowStartMs := windowStart.UnixMilli()
	nowMs := now.UnixMilli()
	ttlSeconds := int(window.Seconds()) + 1

	result, err := r.client.Eval(ctx, luaScript, []string{redisKey}, windowStartMs, nowMs, limit, ttlSeconds).Result()
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	// Parse result
	resultArray, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result format from rate limiter")
	}

	allowed := resultArray[0].(int64) == 1
	remaining := int(resultArray[2].(int64))

	var resetTime time.Time
	var retryAfter time.Duration

	if !allowed && len(resultArray) > 3 {
		if resetTimeMs, ok := resultArray[3].(int64); ok && resetTimeMs > 0 {
			resetTime = time.UnixMilli(resetTimeMs).Add(window)
			retryAfter = time.Until(resetTime)
			if retryAfter < 0 {
				retryAfter = 0
			}
		}
	}

	if allowed {
		resetTime = now.Add(window)
	}

	return &ratelimit.RateLimitResult{
		Allowed:    allowed,
		Limit:      limit,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}

// Reset resets the rate limit for a key
func (r *RateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := rateLimitKeyPrefix + key
	return r.client.Del(ctx, redisKey).Err()
}

// GetStatus returns current rate limit status without updating counters
func (r *RateLimiter) GetStatus(ctx context.Context, key string, limit int, window time.Duration) (*ratelimit.RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	redisKey := rateLimitKeyPrefix + key

	// Remove expired entries first
	err := r.client.ZRemRangeByScore(ctx, redisKey, "-inf", strconv.FormatInt(windowStart.UnixMilli(), 10)).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to clean expired entries: %w", err)
	}

	// Get current count
	current, err := r.client.ZCard(ctx, redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get current count: %w", err)
	}

	remaining := limit - int(current)
	if remaining < 0 {
		remaining = 0
	}

	var resetTime time.Time
	var retryAfter time.Duration

	if current > 0 {
		// Get oldest entry
		oldest, err := r.client.ZRangeWithScores(ctx, redisKey, 0, 0).Result()
		if err == nil && len(oldest) > 0 {
			oldestTime := time.UnixMilli(int64(oldest[0].Score))
			resetTime = oldestTime.Add(window)
			if resetTime.Before(now) {
				resetTime = now.Add(window)
			}
		} else {
			resetTime = now.Add(window)
		}
	} else {
		resetTime = now.Add(window)
	}

	if int(current) >= limit {
		retryAfter = time.Until(resetTime)
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	return &ratelimit.RateLimitResult{
		Allowed:    int(current) < limit,
		Limit:      limit,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}
