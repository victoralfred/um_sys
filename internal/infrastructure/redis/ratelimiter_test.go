package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	redisImpl "github.com/victoralfred/um_sys/internal/infrastructure/redis"
)

func TestRateLimiter_Check(t *testing.T) {
	limiter := setupRateLimiter(t)
	ctx := context.Background()

	t.Run("allows requests under limit", func(t *testing.T) {
		key := "test-key-1"
		limit := 5
		window := time.Minute

		// First request should be allowed
		result, err := limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, limit, result.Limit)
		assert.Equal(t, limit-1, result.Remaining)

		// Second request should be allowed
		result, err = limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, limit-2, result.Remaining)
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		key := "test-key-2"
		limit := 3
		window := time.Minute

		// Make requests up to limit
		for i := 0; i < limit; i++ {
			result, err := limiter.Check(ctx, key, limit, window)
			require.NoError(t, err)
			assert.True(t, result.Allowed)
		}

		// Next request should be blocked
		result, err := limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, 0, result.Remaining)
		assert.True(t, result.RetryAfter > 0)
	})

	t.Run("sliding window behavior", func(t *testing.T) {
		key := "test-key-3"
		limit := 2
		window := 100 * time.Millisecond

		// Make requests up to limit
		for i := 0; i < limit; i++ {
			result, err := limiter.Check(ctx, key, limit, window)
			require.NoError(t, err)
			assert.True(t, result.Allowed)
		}

		// Should be blocked
		result, err := limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		// Wait for window to slide
		time.Sleep(110 * time.Millisecond)

		// Should be allowed again
		result, err = limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	})

	t.Run("different keys are independent", func(t *testing.T) {
		key1 := "test-key-4a"
		key2 := "test-key-4b"
		limit := 1
		window := time.Minute

		// Use up limit for key1
		result, err := limiter.Check(ctx, key1, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed)

		// key1 should be blocked
		result, err = limiter.Check(ctx, key1, limit, window)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		// key2 should still be allowed
		result, err = limiter.Check(ctx, key2, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	})
}

func TestRateLimiter_GetStatus(t *testing.T) {
	limiter := setupRateLimiter(t)
	ctx := context.Background()

	t.Run("returns status without updating counters", func(t *testing.T) {
		key := "status-key-1"
		limit := 5
		window := time.Minute

		// Make some requests first
		for i := 0; i < 2; i++ {
			_, err := limiter.Check(ctx, key, limit, window)
			require.NoError(t, err)
		}

		// Get status multiple times
		for i := 0; i < 3; i++ {
			result, err := limiter.GetStatus(ctx, key, limit, window)
			require.NoError(t, err)
			assert.True(t, result.Allowed)
			assert.Equal(t, 3, result.Remaining) // Should stay same
		}

		// Verify we can still make requests
		result, err := limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 2, result.Remaining)
	})

	t.Run("returns correct status when over limit", func(t *testing.T) {
		key := "status-key-2"
		limit := 2
		window := time.Minute

		// Use up the limit
		for i := 0; i < limit; i++ {
			_, err := limiter.Check(ctx, key, limit, window)
			require.NoError(t, err)
		}

		// Get status
		result, err := limiter.GetStatus(ctx, key, limit, window)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, 0, result.Remaining)
		assert.True(t, result.RetryAfter > 0)
	})
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := setupRateLimiter(t)
	ctx := context.Background()

	t.Run("resets rate limit for key", func(t *testing.T) {
		key := "reset-key-1"
		limit := 2
		window := time.Minute

		// Use up the limit
		for i := 0; i < limit; i++ {
			result, err := limiter.Check(ctx, key, limit, window)
			require.NoError(t, err)
			assert.True(t, result.Allowed)
		}

		// Should be blocked
		result, err := limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		// Reset the key
		err = limiter.Reset(ctx, key)
		require.NoError(t, err)

		// Should be allowed again
		result, err = limiter.Check(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, limit-1, result.Remaining)
	})

	t.Run("reset non-existent key doesn't error", func(t *testing.T) {
		err := limiter.Reset(ctx, "non-existent-key")
		require.NoError(t, err)
	})
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := setupRateLimiter(t)
	ctx := context.Background()

	key := "concurrent-key"
	limit := 10
	window := time.Second

	// Run concurrent requests
	results := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func() {
			result, err := limiter.Check(ctx, key, limit, window)
			if err != nil {
				results <- false
				return
			}
			results <- result.Allowed
		}()
	}

	// Collect results
	allowedCount := 0
	for i := 0; i < 20; i++ {
		if <-results {
			allowedCount++
		}
	}

	// Should allow exactly the limit
	assert.Equal(t, limit, allowedCount)
}

func setupRateLimiter(t *testing.T) *redisImpl.RateLimiter {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6380",
		DB:   1, // Use different DB for rate limiting tests
	})

	// Clean up test data
	t.Cleanup(func() {
		ctx := context.Background()
		client.FlushDB(ctx)
		_ = client.Close()
	})

	return redisImpl.NewRateLimiter(client)
}
