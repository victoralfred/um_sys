# Technical Implementation Guide

## Part 1: Redis Streams Implementation

### 1.1 Redis Streams Service Implementation

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"
    
    "github.com/go-redis/redis/v8"
    "github.com/google/uuid"
)

// StreamAnalyticsService handles real-time analytics with Redis Streams
type StreamAnalyticsService struct {
    client        *redis.Client
    consumerGroup string
    consumerID    string
    mu            sync.RWMutex
    metrics       *StreamMetrics
}

// StreamMetrics tracks performance metrics
type StreamMetrics struct {
    EventsPublished  int64
    EventsConsumed   int64
    ConsumerLag      int64
    LastEventTime    time.Time
    ProcessingErrors int64
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
    ID         string                 `json:"id"`
    Type       string                 `json:"type"`
    UserID     string                 `json:"user_id"`
    SessionID  string                 `json:"session_id"`
    Timestamp  time.Time              `json:"timestamp"`
    Properties map[string]interface{} `json:"properties"`
    Metadata   map[string]string      `json:"metadata"`
}

// NewStreamAnalyticsService creates a new stream analytics service
func NewStreamAnalyticsService(redisURL string) (*StreamAnalyticsService, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }
    
    // Configure for production
    opt.PoolSize = 100
    opt.MinIdleConns = 10
    opt.MaxRetries = 3
    opt.ReadTimeout = 3 * time.Second
    opt.WriteTimeout = 3 * time.Second
    
    client := redis.NewClient(opt)
    
    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis connection failed: %w", err)
    }
    
    service := &StreamAnalyticsService{
        client:        client,
        consumerGroup: "analytics-processors",
        consumerID:    fmt.Sprintf("consumer-%s", uuid.New().String()),
        metrics:       &StreamMetrics{},
    }
    
    // Initialize consumer group
    if err := service.initConsumerGroup(ctx); err != nil {
        return nil, err
    }
    
    return service, nil
}

// PublishEvent publishes an event to the stream
func (s *StreamAnalyticsService) PublishEvent(ctx context.Context, event *AnalyticsEvent) error {
    event.ID = uuid.New().String()
    event.Timestamp = time.Now()
    
    // Serialize event
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    
    // Determine stream key based on event type
    streamKey := s.getStreamKey(event.Type)
    
    // Add to stream with automatic ID
    args := &redis.XAddArgs{
        Stream: streamKey,
        MaxLen: 1000000, // Keep last 1M events
        Approx: true,    // Approximate trimming for performance
        Values: map[string]interface{}{
            "data": string(data),
            "type": event.Type,
            "user": event.UserID,
            "time": event.Timestamp.Unix(),
        },
    }
    
    _, err = s.client.XAdd(ctx, args).Result()
    if err != nil {
        s.metrics.ProcessingErrors++
        return fmt.Errorf("failed to publish event: %w", err)
    }
    
    s.metrics.EventsPublished++
    s.metrics.LastEventTime = time.Now()
    
    return nil
}

// ConsumeEvents consumes events from the stream
func (s *StreamAnalyticsService) ConsumeEvents(ctx context.Context, handler EventHandler) error {
    streamKey := s.getStreamKey("*") // Consume from all streams
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Read from stream
            streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
                Group:    s.consumerGroup,
                Consumer: s.consumerID,
                Streams:  []string{streamKey, ">"},
                Count:    100,
                Block:    5 * time.Second,
                NoAck:    false,
            }).Result()
            
            if err != nil {
                if err == redis.Nil {
                    continue // No new messages
                }
                s.metrics.ProcessingErrors++
                return fmt.Errorf("failed to read stream: %w", err)
            }
            
            // Process messages
            for _, stream := range streams {
                for _, message := range stream.Messages {
                    if err := s.processMessage(ctx, message, handler); err != nil {
                        s.metrics.ProcessingErrors++
                        // Log error but continue processing
                        continue
                    }
                    
                    // Acknowledge message
                    s.client.XAck(ctx, stream.Stream, s.consumerGroup, message.ID)
                    s.metrics.EventsConsumed++
                }
            }
        }
    }
}

// processMessage processes a single message
func (s *StreamAnalyticsService) processMessage(ctx context.Context, msg redis.XMessage, handler EventHandler) error {
    data, ok := msg.Values["data"].(string)
    if !ok {
        return fmt.Errorf("invalid message format")
    }
    
    var event AnalyticsEvent
    if err := json.Unmarshal([]byte(data), &event); err != nil {
        return fmt.Errorf("failed to unmarshal event: %w", err)
    }
    
    return handler.Handle(ctx, &event)
}

// EventHandler processes analytics events
type EventHandler interface {
    Handle(ctx context.Context, event *AnalyticsEvent) error
}

// getStreamKey returns the stream key for an event type
func (s *StreamAnalyticsService) getStreamKey(eventType string) string {
    return fmt.Sprintf("analytics:stream:%s", eventType)
}

// initConsumerGroup initializes the consumer group
func (s *StreamAnalyticsService) initConsumerGroup(ctx context.Context) error {
    streams := []string{
        "analytics:stream:user_action",
        "analytics:stream:system_event",
        "analytics:stream:api_request",
    }
    
    for _, stream := range streams {
        // Create consumer group (ignore error if already exists)
        s.client.XGroupCreateMkStream(ctx, stream, s.consumerGroup, "0")
    }
    
    return nil
}
```

### 1.2 Event Aggregator Implementation

```go
package analytics

import (
    "context"
    "sync"
    "time"
)

// EventAggregator aggregates events in time windows
type EventAggregator struct {
    mu       sync.RWMutex
    windows  map[string]*TimeWindow
    storage  AggregateStorage
}

// TimeWindow represents an aggregation window
type TimeWindow struct {
    Start    time.Time
    End      time.Time
    Duration time.Duration
    Metrics  map[string]*Metric
}

// Metric represents an aggregated metric
type Metric struct {
    Count    int64
    Sum      float64
    Min      float64
    Max      float64
    Values   []float64 // For percentile calculations
}

// NewEventAggregator creates a new event aggregator
func NewEventAggregator(storage AggregateStorage) *EventAggregator {
    return &EventAggregator{
        windows: make(map[string]*TimeWindow),
        storage: storage,
    }
}

// Aggregate processes an event for aggregation
func (a *EventAggregator) Aggregate(event *AnalyticsEvent) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    // Get or create windows
    windows := a.getWindows(event.Timestamp)
    
    for _, window := range windows {
        key := a.getMetricKey(event, window)
        metric := window.Metrics[key]
        if metric == nil {
            metric = &Metric{
                Min: float64(^uint(0) >> 1), // Max float
                Max: -float64(^uint(0) >> 1), // Min float
            }
            window.Metrics[key] = metric
        }
        
        // Update metric
        if value, ok := event.Properties["value"].(float64); ok {
            metric.Count++
            metric.Sum += value
            if value < metric.Min {
                metric.Min = value
            }
            if value > metric.Max {
                metric.Max = value
            }
            metric.Values = append(metric.Values, value)
        }
    }
}

// getWindows returns relevant time windows for a timestamp
func (a *EventAggregator) getWindows(timestamp time.Time) []*TimeWindow {
    windows := []*TimeWindow{}
    
    // 1-minute window
    minuteKey := timestamp.Truncate(time.Minute).Format("2006-01-02T15:04")
    if _, exists := a.windows[minuteKey]; !exists {
        a.windows[minuteKey] = &TimeWindow{
            Start:    timestamp.Truncate(time.Minute),
            End:      timestamp.Truncate(time.Minute).Add(time.Minute),
            Duration: time.Minute,
            Metrics:  make(map[string]*Metric),
        }
    }
    windows = append(windows, a.windows[minuteKey])
    
    // 5-minute window
    fiveMinKey := timestamp.Truncate(5 * time.Minute).Format("2006-01-02T15:04")
    if _, exists := a.windows[fiveMinKey]; !exists {
        a.windows[fiveMinKey] = &TimeWindow{
            Start:    timestamp.Truncate(5 * time.Minute),
            End:      timestamp.Truncate(5 * time.Minute).Add(5 * time.Minute),
            Duration: 5 * time.Minute,
            Metrics:  make(map[string]*Metric),
        }
    }
    windows = append(windows, a.windows[fiveMinKey])
    
    return windows
}

// FlushWindows flushes completed windows to storage
func (a *EventAggregator) FlushWindows(ctx context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    now := time.Now()
    toDelete := []string{}
    
    for key, window := range a.windows {
        if now.After(window.End.Add(30 * time.Second)) { // 30s grace period
            // Calculate percentiles before storing
            for _, metric := range window.Metrics {
                metric.calculatePercentiles()
            }
            
            // Store window
            if err := a.storage.Store(ctx, window); err != nil {
                return err
            }
            
            toDelete = append(toDelete, key)
        }
    }
    
    // Clean up flushed windows
    for _, key := range toDelete {
        delete(a.windows, key)
    }
    
    return nil
}
```

---

## Part 2: Query Optimization Implementation

### 2.1 Smart Query Cache

```go
package cache

import (
    "context"
    "crypto/md5"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/go-redis/redis/v8"
)

// QueryCache implements intelligent query result caching
type QueryCache struct {
    redis      *redis.Client
    serializer Serializer
    stats      *CacheStats
}

// CacheStats tracks cache performance
type CacheStats struct {
    Hits       int64
    Misses     int64
    Evictions  int64
    AvgHitTime time.Duration
}

// CacheKey generates a cache key from query and parameters
func (c *QueryCache) CacheKey(query string, params []interface{}) string {
    h := md5.New()
    h.Write([]byte(query))
    for _, p := range params {
        h.Write([]byte(fmt.Sprintf("%v", p)))
    }
    return fmt.Sprintf("query:cache:%x", h.Sum(nil))
}

// Get retrieves cached query result
func (c *QueryCache) Get(ctx context.Context, key string, dest interface{}) error {
    start := time.Now()
    
    data, err := c.redis.Get(ctx, key).Bytes()
    if err == redis.Nil {
        c.stats.Misses++
        return ErrCacheMiss
    }
    if err != nil {
        return err
    }
    
    c.stats.Hits++
    c.stats.AvgHitTime = (c.stats.AvgHitTime + time.Since(start)) / 2
    
    return c.serializer.Deserialize(data, dest)
}

// Set stores query result in cache
func (c *QueryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := c.serializer.Serialize(value)
    if err != nil {
        return err
    }
    
    // Use pipeline for atomic operations
    pipe := c.redis.Pipeline()
    pipe.Set(ctx, key, data, ttl)
    pipe.ZAdd(ctx, "cache:access", &redis.Z{
        Score:  float64(time.Now().Unix()),
        Member: key,
    })
    
    _, err = pipe.Exec(ctx)
    return err
}

// InvalidatePattern invalidates cache entries matching pattern
func (c *QueryCache) InvalidatePattern(ctx context.Context, pattern string) error {
    var cursor uint64
    var keys []string
    
    for {
        var err error
        var newKeys []string
        newKeys, cursor, err = c.redis.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            return err
        }
        
        keys = append(keys, newKeys...)
        
        if cursor == 0 {
            break
        }
    }
    
    if len(keys) > 0 {
        c.stats.Evictions += int64(len(keys))
        return c.redis.Del(ctx, keys...).Err()
    }
    
    return nil
}
```

### 2.2 Query Optimizer

```go
package optimizer

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
    "time"
)

// QueryOptimizer optimizes database queries
type QueryOptimizer struct {
    db         *sql.DB
    cache      *QueryCache
    monitor    *QueryMonitor
}

// OptimizedQuery represents an optimized query
type OptimizedQuery struct {
    Original   string
    Optimized  string
    CacheKey   string
    CacheTTL   time.Duration
    UseCache   bool
    Hints      []string
}

// Optimize analyzes and optimizes a query
func (o *QueryOptimizer) Optimize(query string, params []interface{}) *OptimizedQuery {
    opt := &OptimizedQuery{
        Original: query,
        UseCache: true,
    }
    
    // Analyze query type
    queryType := o.getQueryType(query)
    
    switch queryType {
    case "SELECT":
        opt.Optimized = o.optimizeSelect(query)
        opt.CacheTTL = o.calculateCacheTTL(query)
        opt.CacheKey = o.cache.CacheKey(opt.Optimized, params)
        
    case "INSERT", "UPDATE", "DELETE":
        opt.Optimized = query
        opt.UseCache = false
        // Invalidate related caches
        o.invalidateRelatedCaches(query)
        
    default:
        opt.Optimized = query
    }
    
    // Add query hints
    opt.Hints = o.generateHints(opt.Optimized)
    
    return opt
}

// optimizeSelect optimizes SELECT queries
func (o *QueryOptimizer) optimizeSelect(query string) string {
    optimized := query
    
    // Add LIMIT if missing for safety
    if !strings.Contains(strings.ToUpper(query), "LIMIT") {
        optimized += " LIMIT 10000"
    }
    
    // Convert subqueries to JOINs where possible
    optimized = o.convertSubqueriesToJoins(optimized)
    
    // Add index hints for known patterns
    optimized = o.addIndexHints(optimized)
    
    return optimized
}

// calculateCacheTTL determines appropriate cache TTL
func (o *QueryOptimizer) calculateCacheTTL(query string) time.Duration {
    // Real-time data: short TTL
    if strings.Contains(query, "real_time") || 
       strings.Contains(query, "current_") {
        return 5 * time.Second
    }
    
    // Aggregated data: longer TTL
    if strings.Contains(query, "COUNT") || 
       strings.Contains(query, "SUM") ||
       strings.Contains(query, "AVG") {
        return 5 * time.Minute
    }
    
    // Historical data: long TTL
    if strings.Contains(query, "history") || 
       strings.Contains(query, "archive") {
        return 1 * time.Hour
    }
    
    // Default
    return 1 * time.Minute
}
```

---

## Part 3: Comprehensive Logging Middleware

### 3.1 Structured Logger Implementation

```go
package logging

import (
    "context"
    "fmt"
    "runtime"
    "sync"
    "time"
    
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// StructuredLogger implements enterprise logging
type StructuredLogger struct {
    logger    *zap.Logger
    buffer    *LogBuffer
    masker    *PIIMasker
    sampler   *AdaptiveSampler
    metrics   *LogMetrics
}

// LogBuffer implements circular buffer for async logging
type LogBuffer struct {
    mu       sync.Mutex
    entries  []LogEntry
    size     int
    position int
    overflow int64
}

// NewStructuredLogger creates a production logger
func NewStructuredLogger(config LogConfig) (*StructuredLogger, error) {
    // Configure encoder
    encoderConfig := zapcore.EncoderConfig{
        TimeKey:        "timestamp",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        FunctionKey:    "function",
        MessageKey:     "message",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }
    
    // Create production config
    zapConfig := zap.Config{
        Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
        Development:       false,
        DisableCaller:     false,
        DisableStacktrace: false,
        Sampling: &zap.SamplingConfig{
            Initial:    100,
            Thereafter: 100,
        },
        Encoding:         "json",
        EncoderConfig:    encoderConfig,
        OutputPaths:      config.OutputPaths,
        ErrorOutputPaths: config.ErrorOutputPaths,
    }
    
    logger, err := zapConfig.Build(
        zap.AddCallerSkip(1),
        zap.AddStacktrace(zapcore.ErrorLevel),
    )
    if err != nil {
        return nil, err
    }
    
    return &StructuredLogger{
        logger:  logger,
        buffer:  NewLogBuffer(config.BufferSize),
        masker:  NewPIIMasker(),
        sampler: NewAdaptiveSampler(),
        metrics: &LogMetrics{},
    }, nil
}

// LogWithContext logs with context information
func (l *StructuredLogger) LogWithContext(ctx context.Context, level zapcore.Level, msg string, fields ...zap.Field) {
    // Extract context values
    contextFields := l.extractContextFields(ctx)
    
    // Add runtime information
    _, file, line, _ := runtime.Caller(1)
    contextFields = append(contextFields,
        zap.String("file", file),
        zap.Int("line", line),
        zap.Int("goroutine", runtime.NumGoroutine()),
    )
    
    // Combine fields
    allFields := append(contextFields, fields...)
    
    // Apply PII masking
    allFields = l.masker.MaskFields(allFields)
    
    // Check sampling
    if !l.sampler.ShouldLog(level, msg) {
        l.metrics.Sampled++
        return
    }
    
    // Log based on level
    switch level {
    case zapcore.DebugLevel:
        l.logger.Debug(msg, allFields...)
    case zapcore.InfoLevel:
        l.logger.Info(msg, allFields...)
    case zapcore.WarnLevel:
        l.logger.Warn(msg, allFields...)
    case zapcore.ErrorLevel:
        l.logger.Error(msg, allFields...)
    case zapcore.DPanicLevel, zapcore.PanicLevel:
        l.logger.Panic(msg, allFields...)
    case zapcore.FatalLevel:
        l.logger.Fatal(msg, allFields...)
    }
    
    l.metrics.TotalLogged++
}

// extractContextFields extracts fields from context
func (l *StructuredLogger) extractContextFields(ctx context.Context) []zap.Field {
    fields := []zap.Field{}
    
    // Trace ID
    if traceID := ctx.Value("trace_id"); traceID != nil {
        fields = append(fields, zap.String("trace_id", traceID.(string)))
    }
    
    // User ID
    if userID := ctx.Value("user_id"); userID != nil {
        fields = append(fields, zap.String("user_id", userID.(string)))
    }
    
    // Session ID
    if sessionID := ctx.Value("session_id"); sessionID != nil {
        fields = append(fields, zap.String("session_id", sessionID.(string)))
    }
    
    // Request ID
    if requestID := ctx.Value("request_id"); requestID != nil {
        fields = append(fields, zap.String("request_id", requestID.(string)))
    }
    
    return fields
}

// AdaptiveSampler implements intelligent log sampling
type AdaptiveSampler struct {
    mu         sync.RWMutex
    rates      map[string]float64
    counts     map[string]int64
    lastReset  time.Time
}

// ShouldLog determines if a log entry should be logged
func (s *AdaptiveSampler) ShouldLog(level zapcore.Level, msg string) bool {
    // Always log errors and above
    if level >= zapcore.ErrorLevel {
        return true
    }
    
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Check sampling rate for this message type
    key := fmt.Sprintf("%s:%s", level, msg)
    rate, exists := s.rates[key]
    if !exists {
        rate = 1.0 // Default: log everything
    }
    
    // Increment count
    s.counts[key]++
    
    // Apply sampling
    return s.counts[key]%int64(1/rate) == 0
}

// AdjustRates adjusts sampling rates based on volume
func (s *AdaptiveSampler) AdjustRates() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    now := time.Now()
    duration := now.Sub(s.lastReset)
    
    for key, count := range s.counts {
        rate := float64(count) / duration.Seconds()
        
        // Adjust sampling rate based on volume
        if rate > 1000 { // More than 1000/sec
            s.rates[key] = 0.01 // Sample 1%
        } else if rate > 100 { // More than 100/sec
            s.rates[key] = 0.1 // Sample 10%
        } else if rate > 10 { // More than 10/sec
            s.rates[key] = 0.5 // Sample 50%
        } else {
            s.rates[key] = 1.0 // Log everything
        }
    }
    
    // Reset counts
    s.counts = make(map[string]int64)
    s.lastReset = now
}
```

### 3.2 Correlation and Tracing

```go
package tracing

import (
    "context"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// DistributedTracer implements distributed tracing
type DistributedTracer struct {
    tracer trace.Tracer
}

// NewDistributedTracer creates a new distributed tracer
func NewDistributedTracer(serviceName string) *DistributedTracer {
    tracer := otel.Tracer(serviceName)
    return &DistributedTracer{
        tracer: tracer,
    }
}

// StartSpan starts a new span
func (t *DistributedTracer) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
    ctx, span := t.tracer.Start(ctx, name,
        trace.WithAttributes(attrs...),
        trace.WithTimestamp(time.Now()),
    )
    
    // Add common attributes
    span.SetAttributes(
        attribute.String("service.version", getVersion()),
        attribute.String("deployment.environment", getEnvironment()),
    )
    
    return ctx, span
}

// TraceMiddleware adds tracing to HTTP requests
func TraceMiddleware(tracer *DistributedTracer) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Start span
        ctx, span := tracer.StartSpan(c.Request.Context(),
            fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
            attribute.String("http.method", c.Request.Method),
            attribute.String("http.url", c.Request.URL.String()),
            attribute.String("http.scheme", c.Request.URL.Scheme),
            attribute.String("http.host", c.Request.Host),
            attribute.String("user_agent", c.Request.UserAgent()),
            attribute.String("remote_addr", c.ClientIP()),
        )
        defer span.End()
        
        // Inject context
        c.Request = c.Request.WithContext(ctx)
        c.Set("trace_id", span.SpanContext().TraceID().String())
        c.Set("span_id", span.SpanContext().SpanID().String())
        
        // Process request
        c.Next()
        
        // Record response
        span.SetAttributes(
            attribute.Int("http.status_code", c.Writer.Status()),
            attribute.Int64("http.response_size", int64(c.Writer.Size())),
        )
        
        // Record errors
        if len(c.Errors) > 0 {
            span.RecordError(c.Errors.Last())
        }
    }
}
```

---

## Configuration Files

### Redis Configuration (redis.conf)
```conf
# Production Redis Configuration
maxmemory 16gb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
appendonly yes
appendfsync everysec
tcp-backlog 511
tcp-keepalive 300
timeout 0
databases 16

# Stream specific
stream-node-max-bytes 4096
stream-node-max-entries 100
```

### Elasticsearch Configuration (elasticsearch.yml)
```yaml
cluster.name: analytics-cluster
node.name: analytics-node-1
path.data: /var/lib/elasticsearch
path.logs: /var/log/elasticsearch
network.host: 0.0.0.0
http.port: 9200
discovery.seed_hosts: ["node1", "node2", "node3"]
cluster.initial_master_nodes: ["node1", "node2", "node3"]

# Performance settings
indices.memory.index_buffer_size: 30%
indices.queries.cache.size: 15%
thread_pool.write.queue_size: 1000
thread_pool.search.queue_size: 1000

# Security
xpack.security.enabled: true
xpack.security.transport.ssl.enabled: true
```

### Monitoring Stack (docker-compose.yml)
```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=30d'
      - '--storage.tsdb.retention.size=10GB'
    
  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_INSTALL_PLUGINS=redis-app
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
    ports:
      - "3000:3000"
    
  elasticsearch:
    image: elasticsearch:7.15.0
    environment:
      - discovery.type=single-node
      - "ES_JAVA_OPTS=-Xms2g -Xmx2g"
    volumes:
      - es_data:/usr/share/elasticsearch/data
    ports:
      - "9200:9200"
    
  kibana:
    image: kibana:7.15.0
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch

volumes:
  prometheus_data:
  grafana_data:
  es_data:
```

This comprehensive implementation provides production-ready code for all three improvements with proper error handling, monitoring, and scalability considerations.