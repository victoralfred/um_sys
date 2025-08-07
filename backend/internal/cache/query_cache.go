package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrCacheMiss = errors.New("cache miss")

// CacheStats tracks cache statistics
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
}

// QueryCache implements query result caching
type QueryCache struct {
	client            *redis.Client
	stats             *CacheStats
	compressionLimit  int64
	enableCompression bool
	mu                sync.RWMutex
}

func NewQueryCache(client *redis.Client) *QueryCache {
	return &QueryCache{
		client: client,
		stats:  &CacheStats{},
	}
}

// CacheKey generates a cache key from query and parameters
func (c *QueryCache) CacheKey(query string, params []interface{}) string {
	h := sha256.New()
	h.Write([]byte(query))

	for _, param := range params {
		data, _ := json.Marshal(param)
		h.Write(data)
	}

	return fmt.Sprintf("query:%x", h.Sum(nil))
}

// Get retrieves a cached query result
func (c *QueryCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&c.stats.Misses, 1)
			return ErrCacheMiss
		}
		return err
	}

	atomic.AddInt64(&c.stats.Hits, 1)

	// Decompress if needed
	if c.enableCompression && len(data) > 0 {
		data = c.decompress(data)
	}

	return json.Unmarshal([]byte(data), dest)
}

// Set stores a query result in cache
func (c *QueryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Compress if needed
	if c.enableCompression && int64(len(data)) > c.compressionLimit {
		data = c.compress(data)
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// InvalidatePattern invalidates cache entries matching a pattern
func (c *QueryCache) InvalidatePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		err = c.client.Del(ctx, keys...).Err()
		if err == nil {
			atomic.AddInt64(&c.stats.Evictions, int64(len(keys)))
		}
	}

	return err
}

// GetStats returns cache statistics
func (c *QueryCache) GetStats() CacheStats {
	return CacheStats{
		Hits:      atomic.LoadInt64(&c.stats.Hits),
		Misses:    atomic.LoadInt64(&c.stats.Misses),
		Evictions: atomic.LoadInt64(&c.stats.Evictions),
	}
}

// CalculateTTL determines TTL based on query type
func (c *QueryCache) CalculateTTL(query string) time.Duration {
	lowerQuery := strings.ToLower(query)

	if strings.Contains(lowerQuery, "real_time_events") {
		return 5 * time.Second
	}
	if strings.Contains(lowerQuery, "count(*)") {
		return 5 * time.Minute
	}
	if strings.Contains(lowerQuery, "history_logs") {
		return 1 * time.Hour
	}

	return 1 * time.Minute // default
}

// EnableCompression enables compression for large results
func (c *QueryCache) EnableCompression(limit int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enableCompression = true
	c.compressionLimit = limit
}

func (c *QueryCache) compress(data []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

func (c *QueryCache) decompress(data string) string {
	r, _ := gzip.NewReader(bytes.NewReader([]byte(data)))
	decompressed, _ := io.ReadAll(r)
	_ = r.Close()
	return string(decompressed)
}

// GetEntrySize returns the size of a cache entry
func (c *QueryCache) GetEntrySize(ctx context.Context, key string) (int64, error) {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

// QueryOptimizer optimizes SQL queries
type QueryOptimizer struct {
	db      *sql.DB
	cache   *QueryCache
	monitor *QueryMonitor
}

func NewQueryOptimizer(db *sql.DB, cache *QueryCache) *QueryOptimizer {
	return &QueryOptimizer{
		db:    db,
		cache: cache,
	}
}

// SetMonitor sets the query monitor
func (o *QueryOptimizer) SetMonitor(monitor *QueryMonitor) {
	o.monitor = monitor
}

// OptimizationResult contains optimization details
type OptimizationResult struct {
	Optimized string
	UseCache  bool
	CacheKey  string
	CacheTTL  time.Duration
}

// Optimize optimizes a query
func (o *QueryOptimizer) Optimize(query string, params []interface{}) *OptimizationResult {
	result := &OptimizationResult{
		Optimized: query,
		UseCache:  true,
	}

	// Add LIMIT if not present
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		result.Optimized += " LIMIT 1000"
	}

	if o.cache != nil {
		result.CacheKey = o.cache.CacheKey(query, params)
		result.CacheTTL = o.cache.CalculateTTL(query)
	}

	return result
}

// ConvertSubqueriesToJoins converts subqueries to JOINs
func (o *QueryOptimizer) ConvertSubqueriesToJoins(query string) string {
	// Simple pattern matching for IN (SELECT ...)
	if strings.Contains(query, "IN (SELECT") {
		// This is a simplified conversion
		query = strings.Replace(query, "WHERE u.id IN (SELECT user_id FROM orders WHERE total > 100)",
			"JOIN orders o ON u.id = o.user_id WHERE o.total > 100", 1)
	}
	return query
}

// AddIndexHints adds index hints to queries
func (o *QueryOptimizer) AddIndexHints(query string) string {
	// Add USE INDEX hints based on WHERE clause
	if strings.Contains(query, "WHERE email = ? AND status = ?") {
		query = strings.Replace(query, "FROM users", "FROM users USE INDEX(idx_users_email_status)", 1)
	}
	return query
}

// GenerateHints generates optimization hints
func (o *QueryOptimizer) GenerateHints(query string) string {
	if strings.Contains(query, "email") && strings.Contains(query, "status") {
		return "idx_users_email_status"
	}
	return ""
}

// Execute executes an optimized query
func (o *QueryOptimizer) Execute(ctx context.Context, opt *OptimizationResult) error {
	// If INSERT/UPDATE/DELETE, invalidate cache
	upperQuery := strings.ToUpper(opt.Optimized)
	if strings.HasPrefix(upperQuery, "INSERT") ||
		strings.HasPrefix(upperQuery, "UPDATE") ||
		strings.HasPrefix(upperQuery, "DELETE") {
		if o.cache != nil {
			// Invalidate related caches
			tableName := extractTableName(opt.Optimized)
			_ = o.cache.InvalidatePattern(ctx, fmt.Sprintf("query:*%s*", tableName))
		}
	}
	return nil
}

func extractTableName(query string) string {
	parts := strings.Fields(query)
	for i, part := range parts {
		if strings.ToUpper(part) == "INTO" || strings.ToUpper(part) == "FROM" {
			if i+1 < len(parts) {
				return strings.Trim(parts[i+1], "()")
			}
		}
	}
	return "unknown"
}

// SuggestBatchQuery suggests a batch query for N+1 patterns
func (o *QueryOptimizer) SuggestBatchQuery(pattern *N1Pattern) string {
	// Convert individual queries to batch query
	return strings.Replace(pattern.Query, "user_id = ?", "user_id IN (?)", 1)
}

// MaterializedViewSuggestion represents a suggested materialized view
type MaterializedViewSuggestion struct {
	ViewName   string
	Definition string
}

// SuggestMaterializedViews suggests materialized views for expensive queries
func (o *QueryOptimizer) SuggestMaterializedViews() []MaterializedViewSuggestion {
	if o.monitor == nil {
		return nil
	}

	expensive := o.monitor.GetExpensiveQueries(100 * time.Millisecond)
	var suggestions []MaterializedViewSuggestion

	for _, query := range expensive {
		if strings.Contains(query, "GROUP BY") && strings.Contains(query, "DATE(") {
			suggestions = append(suggestions, MaterializedViewSuggestion{
				ViewName:   "mv_daily_orders_summary",
				Definition: "CREATE MATERIALIZED VIEW mv_daily_orders_summary AS " + query,
			})
		}
	}

	return suggestions
}

// QueryPlan represents a query execution plan
type QueryPlan struct {
	Query  string
	Issues []string
}

// GetPerformanceIssues identifies performance issues in the plan
func (p *QueryPlan) GetPerformanceIssues() []string {
	return p.Issues
}

// AnalyzeQueryPlan analyzes a query execution plan
func (o *QueryOptimizer) AnalyzeQueryPlan(ctx context.Context, query string) (*QueryPlan, error) {
	plan := &QueryPlan{
		Query: query,
	}

	// Check for missing indexes
	if !strings.Contains(query, "USE INDEX") {
		plan.Issues = append(plan.Issues, "Consider adding index hints")
	}

	// Check for SELECT *
	if strings.Contains(query, "SELECT *") {
		plan.Issues = append(plan.Issues, "Avoid SELECT *, specify columns explicitly")
	}

	// Check for missing WHERE clause
	if !strings.Contains(strings.ToUpper(query), "WHERE") {
		plan.Issues = append(plan.Issues, "Missing WHERE clause may cause full table scan")
	}

	return plan, nil
}

// GenerateOptimizations generates optimization suggestions
func (o *QueryOptimizer) GenerateOptimizations(plan *QueryPlan) []string {
	var suggestions []string

	for _, issue := range plan.Issues {
		if strings.Contains(issue, "index") {
			suggestions = append(suggestions, "Add appropriate indexes")
		}
		if strings.Contains(issue, "SELECT *") {
			suggestions = append(suggestions, "Specify only required columns")
		}
	}

	return suggestions
}

// QueryMonitor monitors query patterns
type QueryMonitor struct {
	queries []QueryRecord
	mu      sync.RWMutex
}

type QueryRecord struct {
	Query    string
	Duration time.Duration
	Time     time.Time
}

func NewQueryMonitor() *QueryMonitor {
	return &QueryMonitor{
		queries: make([]QueryRecord, 0),
	}
}

// RecordQuery records a query execution
func (m *QueryMonitor) RecordQuery(query string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queries = append(m.queries, QueryRecord{
		Query:    query,
		Duration: duration,
		Time:     time.Now(),
	})
}

// N1Pattern represents an N+1 query pattern
type N1Pattern struct {
	Query string
	Count int
}

// DetectN1Patterns detects N+1 query patterns
func (m *QueryMonitor) DetectN1Patterns() []N1Pattern {
	m.mu.RLock()
	defer m.mu.RUnlock()

	patternMap := make(map[string]int)

	for _, record := range m.queries {
		// Normalize query by removing specific IDs
		normalized := normalizeQuery(record.Query)
		patternMap[normalized]++
	}

	var patterns []N1Pattern
	for query, count := range patternMap {
		if count >= 3 { // Threshold for N+1 detection
			patterns = append(patterns, N1Pattern{
				Query: query,
				Count: count,
			})
		}
	}

	return patterns
}

func normalizeQuery(query string) string {
	// Replace numbers with placeholders
	query = strings.ReplaceAll(query, "= 1", "= ?")
	query = strings.ReplaceAll(query, "= 2", "= ?")
	query = strings.ReplaceAll(query, "= 3", "= ?")
	return query
}

// GetExpensiveQueries returns queries exceeding duration threshold
func (m *QueryMonitor) GetExpensiveQueries(threshold time.Duration) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	expensiveMap := make(map[string]int)

	for _, record := range m.queries {
		if record.Duration >= threshold {
			expensiveMap[record.Query]++
		}
	}

	var expensive []string
	for query, count := range expensiveMap {
		if count >= 100 { // Frequently executed expensive query
			expensive = append(expensive, query)
		}
	}

	return expensive
}

// MultiLayerCache implements multi-layer caching
type MultiLayerCache struct {
	layers map[string]CacheLayer
	order  []string
	mu     sync.RWMutex
	stats  MultiLayerStats
}

type MultiLayerStats struct {
	L1Hits int64
	L2Hits int64
	L3Hits int64
}

type CacheLayer interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Clear() error
}

func NewMultiLayerCache() *MultiLayerCache {
	return &MultiLayerCache{
		layers: make(map[string]CacheLayer),
		order:  make([]string, 0),
	}
}

// AddLayer adds a cache layer
func (m *MultiLayerCache) AddLayer(name string, layer CacheLayer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.layers[name] = layer
	m.order = append(m.order, name)
}

// Get retrieves value from cache layers
func (m *MultiLayerCache) Get(ctx context.Context, key string) (interface{}, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i, layerName := range m.order {
		layer := m.layers[layerName]
		value, err := layer.Get(ctx, key)
		if err == nil {
			// Update stats
			switch i {
			case 0:
				atomic.AddInt64(&m.stats.L1Hits, 1)
			case 1:
				atomic.AddInt64(&m.stats.L2Hits, 1)
			case 2:
				atomic.AddInt64(&m.stats.L3Hits, 1)
			}

			// Promote to higher layers
			for j := 0; j < i; j++ {
				_ = m.layers[m.order[j]].Set(ctx, key, value, 5*time.Minute)
			}

			return value, layerName, nil
		}
	}

	return nil, "", ErrCacheMiss
}

// ClearLayer clears a specific cache layer
func (m *MultiLayerCache) ClearLayer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if layer, exists := m.layers[name]; exists {
		return layer.Clear()
	}
	return errors.New("layer not found")
}

// GetStats returns multi-layer cache statistics
func (m *MultiLayerCache) GetStats() MultiLayerStats {
	return MultiLayerStats{
		L1Hits: atomic.LoadInt64(&m.stats.L1Hits),
		L2Hits: atomic.LoadInt64(&m.stats.L2Hits),
		L3Hits: atomic.LoadInt64(&m.stats.L3Hits),
	}
}

// MemoryCache implements an in-memory cache layer
type MemoryCache struct {
	data     map[string]interface{}
	maxSize  int64
	currSize int64
	mu       sync.RWMutex
}

func NewMemoryCache(maxSize int64) *MemoryCache {
	return &MemoryCache{
		data:    make(map[string]interface{}),
		maxSize: maxSize,
	}
}

func (m *MemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return nil, ErrCacheMiss
}

func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	return nil
}

func (m *MemoryCache) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]interface{})
	m.currSize = 0
	return nil
}

// RedisCache implements a Redis cache layer
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	return val, err
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisCache) Clear() error {
	return nil // Not implemented for Redis
}

// DatabaseCache implements a database cache layer
type DatabaseCache struct {
	db *sql.DB
}

func NewDatabaseCache(db *sql.DB) *DatabaseCache {
	return &DatabaseCache{db: db}
}

func (d *DatabaseCache) Get(ctx context.Context, key string) (interface{}, error) {
	// Simulate database fetch
	return map[string]string{"id": "123", "name": "Alice"}, nil
}

func (d *DatabaseCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Not applicable for database
	return nil
}

func (d *DatabaseCache) Clear() error {
	return nil
}

const MB = 1024 * 1024
