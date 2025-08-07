package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

type PostgresMetricRepository struct {
	db *sql.DB
}

func NewPostgresMetricRepository(db *sql.DB) *PostgresMetricRepository {
	return &PostgresMetricRepository{
		db: db,
	}
}

func (r *PostgresMetricRepository) Store(ctx context.Context, metric *analytics.Metric) error {
	query := `
		INSERT INTO analytics_metrics (
			id, name, type, value, labels, timestamp, ttl, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	var labelsJSON, metadataJSON []byte
	var err error

	if metric.Labels != nil {
		labelsJSON, err = json.Marshal(metric.Labels)
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %w", err)
		}
	}

	if metric.Metadata != nil {
		metadataJSON, err = json.Marshal(metric.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	var ttlSeconds *int64
	if metric.TTL != nil {
		seconds := int64(metric.TTL.Seconds())
		ttlSeconds = &seconds
	}

	_, err = r.db.ExecContext(ctx, query,
		metric.ID,
		metric.Name,
		string(metric.Type),
		metric.Value,
		labelsJSON,
		metric.Timestamp,
		ttlSeconds,
		metadataJSON,
		metric.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}

	return nil
}

func (r *PostgresMetricRepository) Get(ctx context.Context, id uuid.UUID) (*analytics.Metric, error) {
	query := `
		SELECT id, name, type, value, labels, timestamp, ttl, metadata, created_at
		FROM analytics_metrics
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	metric := &analytics.Metric{}
	var metricType string
	var labelsJSON, metadataJSON sql.NullString
	var ttlSeconds sql.NullInt64

	err := row.Scan(
		&metric.ID,
		&metric.Name,
		&metricType,
		&metric.Value,
		&labelsJSON,
		&metric.Timestamp,
		&ttlSeconds,
		&metadataJSON,
		&metric.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, analytics.ErrMetricNotFound
		}
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	metric.Type = analytics.MetricType(metricType)

	if labelsJSON.Valid {
		if err := json.Unmarshal([]byte(labelsJSON.String), &metric.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
	}

	if metadataJSON.Valid {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metric.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	if ttlSeconds.Valid {
		ttl := time.Duration(ttlSeconds.Int64) * time.Second
		metric.TTL = &ttl
	}

	return metric, nil
}

func (r *PostgresMetricRepository) List(ctx context.Context, filter analytics.MetricFilter) ([]*analytics.Metric, int64, error) {
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	// Build WHERE conditions
	if len(filter.Names) > 0 {
		nameConditions := make([]string, len(filter.Names))
		for i, name := range filter.Names {
			nameConditions[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, name)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("name IN (%s)", strings.Join(nameConditions, ",")))
	}

	if len(filter.Types) > 0 {
		typeConditions := make([]string, len(filter.Types))
		for i, metricType := range filter.Types {
			typeConditions[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, string(metricType))
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(typeConditions, ",")))
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	// Handle labels filter
	if len(filter.Labels) > 0 {
		for key, value := range filter.Labels {
			conditions = append(conditions, fmt.Sprintf("labels->>'%s' = $%d", key, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM analytics_metrics
		%s`, whereClause)

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count metrics: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT id, name, type, value, labels, timestamp, ttl, metadata, created_at
		FROM analytics_metrics
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list metrics: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	metrics := []*analytics.Metric{}
	for rows.Next() {
		metric := &analytics.Metric{}
		var metricType string
		var labelsJSON, metadataJSON sql.NullString
		var ttlSeconds sql.NullInt64

		err := rows.Scan(
			&metric.ID,
			&metric.Name,
			&metricType,
			&metric.Value,
			&labelsJSON,
			&metric.Timestamp,
			&ttlSeconds,
			&metadataJSON,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan metric: %w", err)
		}

		metric.Type = analytics.MetricType(metricType)

		if labelsJSON.Valid {
			if err := json.Unmarshal([]byte(labelsJSON.String), &metric.Labels); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal labels: %w", err)
			}
		}

		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &metric.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		if ttlSeconds.Valid {
			ttl := time.Duration(ttlSeconds.Int64) * time.Second
			metric.TTL = &ttl
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error during row iteration: %w", err)
	}

	return metrics, total, nil
}

func (r *PostgresMetricRepository) GetByName(ctx context.Context, name string, startTime, endTime time.Time) ([]*analytics.Metric, error) {
	query := `
		SELECT id, name, type, value, labels, timestamp, ttl, metadata, created_at
		FROM analytics_metrics
		WHERE name = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp DESC`

	rows, err := r.db.QueryContext(ctx, query, name, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by name: %w", err)
	}
	defer func() { _ = rows.Close() }()

	metrics := []*analytics.Metric{}
	for rows.Next() {
		metric := &analytics.Metric{}
		var metricType string
		var labelsJSON, metadataJSON sql.NullString
		var ttlSeconds sql.NullInt64

		err := rows.Scan(
			&metric.ID,
			&metric.Name,
			&metricType,
			&metric.Value,
			&labelsJSON,
			&metric.Timestamp,
			&ttlSeconds,
			&metadataJSON,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		metric.Type = analytics.MetricType(metricType)

		if labelsJSON.Valid {
			if err := json.Unmarshal([]byte(labelsJSON.String), &metric.Labels); err != nil {
				return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
			}
		}

		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &metric.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		if ttlSeconds.Valid {
			ttl := time.Duration(ttlSeconds.Int64) * time.Second
			metric.TTL = &ttl
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return metrics, nil
}

func (r *PostgresMetricRepository) GetLatest(ctx context.Context, names []string) ([]*analytics.Metric, error) {
	if len(names) == 0 {
		return []*analytics.Metric{}, nil
	}

	placeholders := make([]string, len(names))
	args := make([]interface{}, len(names))
	for i, name := range names {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = name
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT ON (name) id, name, type, value, labels, timestamp, ttl, metadata, created_at
		FROM analytics_metrics
		WHERE name IN (%s)
		ORDER BY name, timestamp DESC`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	metrics := []*analytics.Metric{}
	for rows.Next() {
		metric := &analytics.Metric{}
		var metricType string
		var labelsJSON, metadataJSON sql.NullString
		var ttlSeconds sql.NullInt64

		err := rows.Scan(
			&metric.ID,
			&metric.Name,
			&metricType,
			&metric.Value,
			&labelsJSON,
			&metric.Timestamp,
			&ttlSeconds,
			&metadataJSON,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		metric.Type = analytics.MetricType(metricType)

		if labelsJSON.Valid {
			if err := json.Unmarshal([]byte(labelsJSON.String), &metric.Labels); err != nil {
				return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
			}
		}

		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &metric.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		if ttlSeconds.Valid {
			ttl := time.Duration(ttlSeconds.Int64) * time.Second
			metric.TTL = &ttl
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return metrics, nil
}

func (r *PostgresMetricRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM analytics_metrics
		WHERE ttl IS NOT NULL
		AND created_at + INTERVAL '1 second' * ttl < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired metrics: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return count, nil
}

func (r *PostgresMetricRepository) Aggregate(ctx context.Context, name string, aggregationType string, startTime, endTime time.Time, groupBy string) (map[string]float64, error) {
	var aggregationFunc string
	switch strings.ToLower(aggregationType) {
	case "sum":
		aggregationFunc = "SUM"
	case "avg", "average":
		aggregationFunc = "AVG"
	case "min":
		aggregationFunc = "MIN"
	case "max":
		aggregationFunc = "MAX"
	case "count":
		aggregationFunc = "COUNT"
	default:
		return nil, fmt.Errorf("unsupported aggregation type: %s", aggregationType)
	}

	var groupByClause, selectClause string
	switch strings.ToLower(groupBy) {
	case "hour":
		groupByClause = "date_trunc('hour', timestamp)"
		selectClause = "date_trunc('hour', timestamp)::text"
	case "day":
		groupByClause = "date_trunc('day', timestamp)"
		selectClause = "date_trunc('day', timestamp)::text"
	case "week":
		groupByClause = "date_trunc('week', timestamp)"
		selectClause = "date_trunc('week', timestamp)::text"
	case "month":
		groupByClause = "date_trunc('month', timestamp)"
		selectClause = "date_trunc('month', timestamp)::text"
	case "":
		groupByClause = "'total'"
		selectClause = "'total'"
	default:
		return nil, fmt.Errorf("unsupported groupBy: %s", groupBy)
	}

	query := fmt.Sprintf(`
		SELECT %s, %s(value)
		FROM analytics_metrics
		WHERE name = $1 AND timestamp >= $2 AND timestamp <= $3
		GROUP BY %s
		ORDER BY %s`, selectClause, aggregationFunc, groupByClause, groupByClause)

	rows, err := r.db.QueryContext(ctx, query, name, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]float64)
	for rows.Next() {
		var key string
		var value float64
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan aggregation: %w", err)
		}
		result[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return result, nil
}
