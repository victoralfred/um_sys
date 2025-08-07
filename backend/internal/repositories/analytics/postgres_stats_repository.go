package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

type PostgresStatsRepository struct {
	db *sql.DB
}

func NewPostgresStatsRepository(db *sql.DB) *PostgresStatsRepository {
	return &PostgresStatsRepository{
		db: db,
	}
}

func (r *PostgresStatsRepository) GenerateUsageStats(ctx context.Context, filter analytics.StatsFilter) (*analytics.UsageStats, error) {
	stats := &analytics.UsageStats{
		Period:    filter.Period,
		StartTime: filter.StartTime,
		EndTime:   filter.EndTime,
	}

	// Get total events
	totalEvents, err := r.getTotalEvents(ctx, filter.StartTime, filter.EndTime, filter.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}
	stats.TotalEvents = totalEvents

	// Get unique users
	uniqueUsers, err := r.getUniqueUsers(ctx, filter.StartTime, filter.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique users: %w", err)
	}
	stats.UniqueUsers = uniqueUsers

	// Get total sessions
	totalSessions, err := r.getTotalSessions(ctx, filter.StartTime, filter.EndTime, filter.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total sessions: %w", err)
	}
	stats.TotalSessions = totalSessions

	// Get average session time
	avgSessionTime, err := r.getAverageSessionTime(ctx, filter.StartTime, filter.EndTime, filter.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get average session time: %w", err)
	}
	stats.AvgSessionTime = avgSessionTime

	// Get events by type
	eventsByType, err := r.getEventsByType(ctx, filter.StartTime, filter.EndTime, filter.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}
	stats.EventsByType = eventsByType

	// Get top pages
	topPages, err := r.GetTopPages(ctx, filter.StartTime, filter.EndTime, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top pages: %w", err)
	}
	stats.TopPages = topPages

	// Get top features
	topFeatures, err := r.GetTopFeatures(ctx, filter.StartTime, filter.EndTime, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top features: %w", err)
	}
	stats.TopFeatures = topFeatures

	// Get user growth
	userGrowth, err := r.GetUserGrowth(ctx, filter.StartTime, filter.EndTime, filter.Period)
	if err != nil {
		return nil, fmt.Errorf("failed to get user growth: %w", err)
	}
	stats.UserGrowth = userGrowth

	return stats, nil
}

func (r *PostgresStatsRepository) GetUserGrowth(ctx context.Context, startTime, endTime time.Time, interval string) ([]analytics.UserGrowthStats, error) {
	var dateTrunc string
	switch interval {
	case "hourly":
		dateTrunc = "hour"
	case "daily":
		dateTrunc = "day"
	case "weekly":
		dateTrunc = "week"
	case "monthly":
		dateTrunc = "month"
	default:
		dateTrunc = "day"
	}

	query := fmt.Sprintf(`
		WITH date_series AS (
			SELECT date_trunc('%s', generate_series($1::timestamp, $2::timestamp, '1 %s'::interval)) as date
		),
		user_registrations AS (
			SELECT 
				date_trunc('%s', timestamp) as date,
				COUNT(DISTINCT user_id) as new_users
			FROM analytics_events
			WHERE type = 'user_registration'
			AND timestamp >= $1 AND timestamp <= $2
			GROUP BY date_trunc('%s', timestamp)
		),
		active_users AS (
			SELECT 
				date_trunc('%s', timestamp) as date,
				COUNT(DISTINCT user_id) as active_users
			FROM analytics_events
			WHERE user_id IS NOT NULL
			AND timestamp >= $1 AND timestamp <= $2
			GROUP BY date_trunc('%s', timestamp)
		)
		SELECT 
			ds.date,
			COALESCE(ur.new_users, 0) as new_users,
			COALESCE(au.active_users, 0) as active_users,
			0.0 as churn_rate
		FROM date_series ds
		LEFT JOIN user_registrations ur ON ds.date = ur.date
		LEFT JOIN active_users au ON ds.date = au.date
		ORDER BY ds.date`, dateTrunc, dateTrunc, dateTrunc, dateTrunc, dateTrunc, dateTrunc)

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get user growth: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var growth []analytics.UserGrowthStats
	for rows.Next() {
		var stat analytics.UserGrowthStats
		if err := rows.Scan(&stat.Date, &stat.NewUsers, &stat.ActiveUsers, &stat.ChurnRate); err != nil {
			return nil, fmt.Errorf("failed to scan user growth: %w", err)
		}
		growth = append(growth, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return growth, nil
}

func (r *PostgresStatsRepository) GetTopPages(ctx context.Context, startTime, endTime time.Time, limit int) ([]analytics.PageStats, error) {
	query := `
		SELECT 
			context->>'path' as path,
			COUNT(*) as views,
			COUNT(DISTINCT user_id) as unique_views,
			EXTRACT(EPOCH FROM AVG(
				CASE 
					WHEN context->>'response_time_ms' IS NOT NULL 
					THEN INTERVAL '1 millisecond' * (context->>'response_time_ms')::bigint
					ELSE INTERVAL '0'
				END
			))::bigint as avg_time_seconds
		FROM analytics_events
		WHERE type = 'page_view'
		AND context->>'path' IS NOT NULL
		AND timestamp >= $1 AND timestamp <= $2
		GROUP BY context->>'path'
		ORDER BY views DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top pages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var pages []analytics.PageStats
	for rows.Next() {
		var page analytics.PageStats
		var avgTimeSeconds int64
		if err := rows.Scan(&page.Path, &page.Views, &page.UniqueViews, &avgTimeSeconds); err != nil {
			return nil, fmt.Errorf("failed to scan page stats: %w", err)
		}
		page.AvgTime = time.Duration(avgTimeSeconds) * time.Second
		pages = append(pages, page)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return pages, nil
}

func (r *PostgresStatsRepository) GetTopFeatures(ctx context.Context, startTime, endTime time.Time, limit int) ([]analytics.FeatureStats, error) {
	query := `
		SELECT 
			properties->>'feature' as feature,
			COUNT(*) as usage,
			COUNT(DISTINCT user_id) as users
		FROM analytics_events
		WHERE type = 'feature_usage'
		AND properties->>'feature' IS NOT NULL
		AND timestamp >= $1 AND timestamp <= $2
		GROUP BY properties->>'feature'
		ORDER BY usage DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top features: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var features []analytics.FeatureStats
	for rows.Next() {
		var feature analytics.FeatureStats
		if err := rows.Scan(&feature.Feature, &feature.Usage, &feature.Users); err != nil {
			return nil, fmt.Errorf("failed to scan feature stats: %w", err)
		}
		features = append(features, feature)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return features, nil
}

func (r *PostgresStatsRepository) GetActiveUsers(ctx context.Context, startTime, endTime time.Time, interval string) (map[string]int64, error) {
	var dateTrunc string
	switch interval {
	case "hourly":
		dateTrunc = "hour"
	case "daily":
		dateTrunc = "day"
	case "weekly":
		dateTrunc = "week"
	case "monthly":
		dateTrunc = "month"
	default:
		dateTrunc = "day"
	}

	query := fmt.Sprintf(`
		SELECT 
			date_trunc('%s', timestamp)::text as period,
			COUNT(DISTINCT user_id) as active_users
		FROM analytics_events
		WHERE user_id IS NOT NULL
		AND timestamp >= $1 AND timestamp <= $2
		GROUP BY date_trunc('%s', timestamp)
		ORDER BY date_trunc('%s', timestamp)`, dateTrunc, dateTrunc, dateTrunc)

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	activeUsers := make(map[string]int64)
	for rows.Next() {
		var period string
		var count int64
		if err := rows.Scan(&period, &count); err != nil {
			return nil, fmt.Errorf("failed to scan active users: %w", err)
		}
		activeUsers[period] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return activeUsers, nil
}

// Helper methods

func (r *PostgresStatsRepository) getTotalEvents(ctx context.Context, startTime, endTime time.Time, userID *uuid.UUID) (int64, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `SELECT COUNT(*) FROM analytics_events WHERE timestamp >= $1 AND timestamp <= $2 AND user_id = $3`
		args = []interface{}{startTime, endTime, *userID}
	} else {
		query = `SELECT COUNT(*) FROM analytics_events WHERE timestamp >= $1 AND timestamp <= $2`
		args = []interface{}{startTime, endTime}
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total events: %w", err)
	}

	return count, nil
}

func (r *PostgresStatsRepository) getUniqueUsers(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	query := `SELECT COUNT(DISTINCT user_id) FROM analytics_events WHERE timestamp >= $1 AND timestamp <= $2 AND user_id IS NOT NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, startTime, endTime).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unique users: %w", err)
	}

	return count, nil
}

func (r *PostgresStatsRepository) getTotalSessions(ctx context.Context, startTime, endTime time.Time, userID *uuid.UUID) (int64, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `SELECT COUNT(DISTINCT session_id) FROM analytics_events WHERE timestamp >= $1 AND timestamp <= $2 AND session_id IS NOT NULL AND user_id = $3`
		args = []interface{}{startTime, endTime, *userID}
	} else {
		query = `SELECT COUNT(DISTINCT session_id) FROM analytics_events WHERE timestamp >= $1 AND timestamp <= $2 AND session_id IS NOT NULL`
		args = []interface{}{startTime, endTime}
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total sessions: %w", err)
	}

	return count, nil
}

func (r *PostgresStatsRepository) getAverageSessionTime(ctx context.Context, startTime, endTime time.Time, userID *uuid.UUID) (time.Duration, error) {
	// This is a simplified calculation - in a real system, you'd want to track session start/end events
	// For now, we'll estimate based on time between first and last events in a session
	var query string
	var args []interface{}

	baseQuery := `
		SELECT AVG(session_duration)::bigint
		FROM (
			SELECT 
				session_id,
				EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) as session_duration
			FROM analytics_events
			WHERE timestamp >= $1 AND timestamp <= $2 
			AND session_id IS NOT NULL`

	if userID != nil {
		query = baseQuery + ` AND user_id = $3
			GROUP BY session_id
			HAVING COUNT(*) > 1
		) sessions`
		args = []interface{}{startTime, endTime, *userID}
	} else {
		query = baseQuery + `
			GROUP BY session_id
			HAVING COUNT(*) > 1
		) sessions`
		args = []interface{}{startTime, endTime}
	}

	var avgSeconds sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&avgSeconds)
	if err != nil {
		return 0, fmt.Errorf("failed to get average session time: %w", err)
	}

	if !avgSeconds.Valid {
		return 0, nil
	}

	return time.Duration(avgSeconds.Int64) * time.Second, nil
}

func (r *PostgresStatsRepository) getEventsByType(ctx context.Context, startTime, endTime time.Time, userID *uuid.UUID) (map[analytics.EventType]int64, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `SELECT type, COUNT(*) FROM analytics_events WHERE timestamp >= $1 AND timestamp <= $2 AND user_id = $3 GROUP BY type`
		args = []interface{}{startTime, endTime, *userID}
	} else {
		query = `SELECT type, COUNT(*) FROM analytics_events WHERE timestamp >= $1 AND timestamp <= $2 GROUP BY type`
		args = []interface{}{startTime, endTime}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}
	defer func() { _ = rows.Close() }()

	eventsByType := make(map[analytics.EventType]int64)
	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan event type: %w", err)
		}
		eventsByType[analytics.EventType(eventType)] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return eventsByType, nil
}
