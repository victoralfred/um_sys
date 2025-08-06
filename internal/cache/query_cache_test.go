package cache

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestQueryCache(t *testing.T) {
	ctx := context.Background()

	t.Run("Generate cache key from query", func(t *testing.T) {
		cache := NewQueryCache(getTestRedisClient())

		query := "SELECT * FROM users WHERE id = ?"
		params := []interface{}{123}

		key1 := cache.CacheKey(query, params)
		assert.NotEmpty(t, key1)

		// Same query should generate same key
		key2 := cache.CacheKey(query, params)
		assert.Equal(t, key1, key2)

		// Different params should generate different key
		params2 := []interface{}{456}
		key3 := cache.CacheKey(query, params2)
		assert.NotEqual(t, key1, key3)
	})

	t.Run("Cache and retrieve query results", func(t *testing.T) {
		cache := NewQueryCache(getTestRedisClient())

		// Test data
		type User struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		users := []User{
			{ID: 1, Name: "Alice", Email: "alice@example.com"},
			{ID: 2, Name: "Bob", Email: "bob@example.com"},
		}

		key := "test:users:list"

		// Cache miss initially
		var result []User
		err := cache.Get(ctx, key, &result)
		assert.Error(t, err)
		assert.Equal(t, ErrCacheMiss, err)

		// Set cache
		err = cache.Set(ctx, key, users, 5*time.Second)
		assert.NoError(t, err)

		// Cache hit
		var cached []User
		err = cache.Get(ctx, key, &cached)
		assert.NoError(t, err)
		assert.Equal(t, users, cached)

		// Check stats
		stats := cache.GetStats()
		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
	})

	t.Run("Cache invalidation patterns", func(t *testing.T) {
		cache := NewQueryCache(getTestRedisClient())

		// Set multiple cache entries
		_ = cache.Set(ctx, "query:users:1", "user1", 1*time.Minute)
		_ = cache.Set(ctx, "query:users:2", "user2", 1*time.Minute)
		_ = cache.Set(ctx, "query:posts:1", "post1", 1*time.Minute)

		// Invalidate all user queries
		err := cache.InvalidatePattern(ctx, "query:users:*")
		assert.NoError(t, err)

		// User queries should be gone
		var result string
		err = cache.Get(ctx, "query:users:1", &result)
		assert.Equal(t, ErrCacheMiss, err)

		// Post queries should still exist
		err = cache.Get(ctx, "query:posts:1", &result)
		assert.NoError(t, err)
		assert.Equal(t, "post1", result)

		// Check eviction count
		stats := cache.GetStats()
		assert.Equal(t, int64(2), stats.Evictions)
	})

	t.Run("Adaptive TTL based on query type", func(t *testing.T) {
		cache := NewQueryCache(getTestRedisClient())

		testCases := []struct {
			query       string
			expectedTTL time.Duration
		}{
			{"SELECT * FROM real_time_events", 5 * time.Second},
			{"SELECT COUNT(*) FROM users", 5 * time.Minute},
			{"SELECT * FROM history_logs", 1 * time.Hour},
			{"SELECT * FROM products", 1 * time.Minute}, // default
		}

		for _, tc := range testCases {
			ttl := cache.CalculateTTL(tc.query)
			assert.Equal(t, tc.expectedTTL, ttl, "Query: %s", tc.query)
		}
	})

	t.Run("Compression for large results", func(t *testing.T) {
		cache := NewQueryCache(getTestRedisClient())
		cache.EnableCompression(1024) // Compress if > 1KB

		// Create large result
		largeData := make([]map[string]string, 1000)
		for i := range largeData {
			largeData[i] = map[string]string{
				"id":          fmt.Sprintf("%d", i),
				"description": strings.Repeat("x", 100),
			}
		}

		key := "large:result"
		err := cache.Set(ctx, key, largeData, 1*time.Minute)
		assert.NoError(t, err)

		// Retrieve and verify
		var retrieved []map[string]string
		err = cache.Get(ctx, key, &retrieved)
		assert.NoError(t, err)
		assert.Equal(t, len(largeData), len(retrieved))

		// Check if compression was used
		size, err := cache.GetEntrySize(ctx, key)
		assert.NoError(t, err)
		assert.Less(t, size, int64(100*1000)) // Should be compressed
	})
}

func getTestRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   2, // Use test database
	})
}

func getTestDB() *sql.DB {
	// Return nil for testing purposes
	return nil
}

func TestQueryOptimizer(t *testing.T) {
	ctx := context.Background()

	t.Run("Optimize SELECT queries", func(t *testing.T) {
		optimizer := NewQueryOptimizer(nil, NewQueryCache(getTestRedisClient()))

		// Query without LIMIT
		query := "SELECT * FROM users WHERE active = true"
		opt := optimizer.Optimize(query, nil)

		assert.Contains(t, opt.Optimized, "LIMIT")
		assert.True(t, opt.UseCache)
		assert.NotEmpty(t, opt.CacheKey)
		assert.Greater(t, opt.CacheTTL, time.Duration(0))
	})

	t.Run("Convert subqueries to JOINs", func(t *testing.T) {
		optimizer := NewQueryOptimizer(nil, nil)

		query := `
			SELECT u.* FROM users u 
			WHERE u.id IN (SELECT user_id FROM orders WHERE total > 100)
		`

		optimized := optimizer.ConvertSubqueriesToJoins(query)
		assert.Contains(t, optimized, "JOIN")
		assert.NotContains(t, optimized, "IN (SELECT")
	})

	t.Run("Add index hints", func(t *testing.T) {
		optimizer := NewQueryOptimizer(nil, nil)

		query := "SELECT * FROM users WHERE email = ? AND status = ?"
		optimized := optimizer.AddIndexHints(query)

		// Should suggest composite index
		hints := optimizer.GenerateHints(optimized)
		assert.Contains(t, hints, "idx_users_email_status")
	})

	t.Run("Detect and prevent N+1 queries", func(t *testing.T) {
		monitor := NewQueryMonitor()
		optimizer := NewQueryOptimizer(nil, nil)
		optimizer.SetMonitor(monitor)

		// Simulate N+1 pattern
		queries := []string{
			"SELECT * FROM posts WHERE user_id = 1",
			"SELECT * FROM posts WHERE user_id = 2",
			"SELECT * FROM posts WHERE user_id = 3",
		}

		for _, q := range queries {
			monitor.RecordQuery(q, 10*time.Millisecond)
		}

		// Detect N+1
		patterns := monitor.DetectN1Patterns()
		assert.Len(t, patterns, 1)
		assert.Contains(t, patterns[0].Query, "posts")

		// Suggest batch query
		suggestion := optimizer.SuggestBatchQuery(&patterns[0])
		assert.Contains(t, suggestion, "IN")
	})

	t.Run("Query result caching strategy", func(t *testing.T) {
		cache := NewQueryCache(getTestRedisClient())
		optimizer := NewQueryOptimizer(nil, cache)

		// INSERT should invalidate cache
		insertQuery := "INSERT INTO users (name, email) VALUES (?, ?)"
		opt := optimizer.Optimize(insertQuery, []interface{}{"John", "john@example.com"})
		assert.False(t, opt.UseCache)

		// Should trigger cache invalidation
		err := optimizer.Execute(ctx, opt)
		assert.NoError(t, err)

		// Verify related caches were invalidated
		stats := cache.GetStats()
		assert.Greater(t, stats.Evictions, int64(0))
	})

	t.Run("Materialized view suggestions", func(t *testing.T) {
		monitor := NewQueryMonitor()
		optimizer := NewQueryOptimizer(nil, nil)
		optimizer.SetMonitor(monitor)

		// Record expensive aggregation queries
		expensiveQuery := `
			SELECT 
				DATE(created_at) as date,
				COUNT(*) as total,
				SUM(amount) as revenue
			FROM orders
			WHERE created_at >= NOW() - INTERVAL '30 days'
			GROUP BY DATE(created_at)
		`

		// Simulate multiple executions
		for i := 0; i < 100; i++ {
			monitor.RecordQuery(expensiveQuery, 500*time.Millisecond)
		}

		// Should suggest materialized view
		suggestions := optimizer.SuggestMaterializedViews()
		assert.Len(t, suggestions, 1)
		assert.Contains(t, suggestions[0].ViewName, "daily_orders")
		assert.Contains(t, suggestions[0].Definition, "MATERIALIZED VIEW")
	})

	t.Run("Query plan analysis", func(t *testing.T) {
		db := getTestDB()
		optimizer := NewQueryOptimizer(db, nil)

		query := "SELECT * FROM users u JOIN posts p ON u.id = p.user_id WHERE u.active = true"

		plan, err := optimizer.AnalyzeQueryPlan(ctx, query)
		assert.NoError(t, err)
		assert.NotNil(t, plan)

		// Check for performance issues
		issues := plan.GetPerformanceIssues()
		for _, issue := range issues {
			t.Logf("Performance issue: %s", issue)
		}

		// Generate optimization suggestions
		suggestions := optimizer.GenerateOptimizations(plan)
		assert.NotEmpty(t, suggestions)
	})
}

func TestMultiLayerCache(t *testing.T) {
	ctx := context.Background()

	t.Run("L1 (memory) -> L2 (Redis) -> L3 (Database)", func(t *testing.T) {
		mlCache := NewMultiLayerCache()

		// Configure layers
		mlCache.AddLayer("L1", NewMemoryCache(100*MB))
		mlCache.AddLayer("L2", NewRedisCache(getTestRedisClient()))
		mlCache.AddLayer("L3", NewDatabaseCache(getTestDB()))

		key := "user:123"

		// First access - all misses, fetch from DB
		_, layer, err := mlCache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, "L3", layer)

		// Second access - L1 hit
		_, layer, err = mlCache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, "L1", layer)

		// Clear L1
		_ = mlCache.ClearLayer("L1")

		// Third access - L2 hit
		_, layer, err = mlCache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, "L2", layer)

		// Stats
		stats := mlCache.GetStats()
		assert.Equal(t, int64(1), stats.L1Hits)
		assert.Equal(t, int64(1), stats.L2Hits)
		assert.Equal(t, int64(1), stats.L3Hits)
	})
}
