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

type PostgresEventRepository struct {
	db *sql.DB
}

func NewPostgresEventRepository(db *sql.DB) *PostgresEventRepository {
	return &PostgresEventRepository{
		db: db,
	}
}

func (r *PostgresEventRepository) Store(ctx context.Context, event *analytics.Event) error {
	query := `
		INSERT INTO analytics_events (
			id, type, user_id, session_id, timestamp, properties, context, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	var propertiesJSON, contextJSON interface{}
	var err error

	if event.Properties != nil {
		propertiesJSON, err = json.Marshal(event.Properties)
		if err != nil {
			return fmt.Errorf("failed to marshal properties: %w", err)
		}
	} else {
		propertiesJSON = nil
	}

	if event.Context != nil {
		contextJSON, err = json.Marshal(event.Context)
		if err != nil {
			return fmt.Errorf("failed to marshal context: %w", err)
		}
	} else {
		contextJSON = nil
	}

	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		string(event.Type),
		event.UserID,
		event.SessionID,
		event.Timestamp,
		propertiesJSON,
		contextJSON,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	return nil
}

func (r *PostgresEventRepository) Get(ctx context.Context, id uuid.UUID) (*analytics.Event, error) {
	query := `
		SELECT id, type, user_id, session_id, timestamp, properties, context, created_at
		FROM analytics_events
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	event := &analytics.Event{}
	var eventType string
	var propertiesJSON, contextJSON sql.NullString

	err := row.Scan(
		&event.ID,
		&eventType,
		&event.UserID,
		&event.SessionID,
		&event.Timestamp,
		&propertiesJSON,
		&contextJSON,
		&event.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, analytics.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event.Type = analytics.EventType(eventType)

	if propertiesJSON.Valid {
		if err := json.Unmarshal([]byte(propertiesJSON.String), &event.Properties); err != nil {
			return nil, fmt.Errorf("failed to unmarshal properties: %w", err)
		}
	}

	if contextJSON.Valid {
		if err := json.Unmarshal([]byte(contextJSON.String), &event.Context); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	return event, nil
}

func (r *PostgresEventRepository) List(ctx context.Context, filter analytics.EventFilter) ([]*analytics.Event, int64, error) {
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	// Build WHERE conditions
	if len(filter.Types) > 0 {
		typeConditions := make([]string, len(filter.Types))
		for i, eventType := range filter.Types {
			typeConditions[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, string(eventType))
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(typeConditions, ",")))
	}

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.SessionID != nil {
		conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIndex))
		args = append(args, *filter.SessionID)
		argIndex++
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

	if filter.Path != nil {
		conditions = append(conditions, fmt.Sprintf("context->>'path' = $%d", argIndex))
		args = append(args, *filter.Path)
		argIndex++
	}

	if filter.IPAddress != nil {
		conditions = append(conditions, fmt.Sprintf("context->>'ip_address' = $%d", argIndex))
		args = append(args, *filter.IPAddress)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM analytics_events
		%s`, whereClause)

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT id, type, user_id, session_id, timestamp, properties, context, created_at
		FROM analytics_events
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
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	events := []*analytics.Event{}
	for rows.Next() {
		event := &analytics.Event{}
		var eventType string
		var propertiesJSON, contextJSON sql.NullString

		err := rows.Scan(
			&event.ID,
			&eventType,
			&event.UserID,
			&event.SessionID,
			&event.Timestamp,
			&propertiesJSON,
			&contextJSON,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}

		event.Type = analytics.EventType(eventType)

		if propertiesJSON.Valid {
			if err := json.Unmarshal([]byte(propertiesJSON.String), &event.Properties); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal properties: %w", err)
			}
		}

		if contextJSON.Valid {
			if err := json.Unmarshal([]byte(contextJSON.String), &event.Context); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal context: %w", err)
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error during row iteration: %w", err)
	}

	return events, total, nil
}

func (r *PostgresEventRepository) GetUserEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*analytics.Event, int64, error) {
	filter := analytics.EventFilter{
		UserID: &userID,
		Limit:  limit,
		Offset: offset,
	}
	return r.List(ctx, filter)
}

func (r *PostgresEventRepository) GetSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]*analytics.Event, int64, error) {
	filter := analytics.EventFilter{
		SessionID: &sessionID,
		Limit:     limit,
		Offset:    offset,
	}
	return r.List(ctx, filter)
}

func (r *PostgresEventRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM analytics_events WHERE timestamp < $1`

	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return count, nil
}

func (r *PostgresEventRepository) GetEventCounts(ctx context.Context, startTime, endTime time.Time, groupBy string) (map[string]int64, error) {
	var query string
	switch groupBy {
	case "type":
		query = `
			SELECT type, COUNT(*)
			FROM analytics_events
			WHERE timestamp >= $1 AND timestamp <= $2
			GROUP BY type`
	case "hour":
		query = `
			SELECT date_trunc('hour', timestamp)::text, COUNT(*)
			FROM analytics_events
			WHERE timestamp >= $1 AND timestamp <= $2
			GROUP BY date_trunc('hour', timestamp)
			ORDER BY date_trunc('hour', timestamp)`
	case "day":
		query = `
			SELECT date_trunc('day', timestamp)::text, COUNT(*)
			FROM analytics_events
			WHERE timestamp >= $1 AND timestamp <= $2
			GROUP BY date_trunc('day', timestamp)
			ORDER BY date_trunc('day', timestamp)`
	default:
		return nil, fmt.Errorf("unsupported groupBy: %s", groupBy)
	}

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get event counts: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	counts := make(map[string]int64)
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[key] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return counts, nil
}
