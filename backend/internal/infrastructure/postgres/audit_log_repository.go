package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victoralfred/um_sys/internal/domain/audit"
)

type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{
		db: db,
	}
}

func (r *AuditLogRepository) Create(ctx context.Context, entry *audit.LogEntry) error {
	query := `
		INSERT INTO audit_logs (
			id, timestamp, event_type, severity, user_id, actor_id, entity_type, 
			entity_id, action, description, ip_address, user_agent, metadata, 
			changes, request_id, session_id, trace_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`

	var metadataJSON, changesJSON []byte
	var err error

	if entry.Metadata != nil {
		metadataJSON, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	if entry.Changes != nil {
		changesJSON, err = json.Marshal(entry.Changes)
		if err != nil {
			return fmt.Errorf("failed to marshal changes: %w", err)
		}
	}

	// Handle empty IP address
	var ipAddress interface{}
	if entry.IPAddress == "" {
		ipAddress = nil
	} else {
		ipAddress = entry.IPAddress
	}

	_, err = r.db.Exec(ctx, query,
		entry.ID,
		entry.Timestamp,
		string(entry.EventType),
		string(entry.Severity),
		entry.UserID,
		entry.ActorID,
		entry.EntityType,
		entry.EntityID,
		entry.Action,
		entry.Description,
		ipAddress,
		entry.UserAgent,
		metadataJSON,
		changesJSON,
		entry.RequestID,
		entry.SessionID,
		entry.TraceID,
		entry.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log entry: %w", err)
	}

	return nil
}

func (r *AuditLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*audit.LogEntry, error) {
	query := `
		SELECT id, timestamp, event_type, severity, user_id, actor_id, entity_type,
			   entity_id, action, description, ip_address, user_agent, metadata,
			   changes, request_id, session_id, trace_id, created_at
		FROM audit_logs
		WHERE id = $1
	`

	row := r.db.QueryRow(ctx, query, id)
	return r.scanLogEntry(row)
}

func (r *AuditLogRepository) List(ctx context.Context, filter audit.LogFilter) ([]*audit.LogEntry, int64, error) {
	whereClause, args := r.buildWhereClause(filter)

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Data query with pagination
	dataQuery := fmt.Sprintf(`
		SELECT id, timestamp, event_type, severity, user_id, actor_id, entity_type,
			   entity_id, action, description, ip_address, user_agent, metadata,
			   changes, request_id, session_id, trace_id, created_at
		FROM audit_logs %s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*audit.LogEntry
	for rows.Next() {
		entry, err := r.scanLogEntry(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating audit log rows: %w", err)
	}

	return entries, total, nil
}

func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*audit.LogEntry, int64, error) {
	filter := audit.LogFilter{
		UserID: &userID,
		Limit:  limit,
		Offset: offset,
	}
	return r.List(ctx, filter)
}

func (r *AuditLogRepository) GetByEntityID(ctx context.Context, entityType, entityID string, limit, offset int) ([]*audit.LogEntry, int64, error) {
	filter := audit.LogFilter{
		EntityType: entityType,
		EntityID:   entityID,
		Limit:      limit,
		Offset:     offset,
	}
	return r.List(ctx, filter)
}

func (r *AuditLogRepository) GetSummary(ctx context.Context, filter audit.LogFilter) (*audit.LogSummary, error) {
	whereClause, args := r.buildWhereClause(filter)

	// Get total events
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var totalEvents int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to count total events: %w", err)
	}

	// Get events by type
	eventTypeQuery := fmt.Sprintf(`
		SELECT event_type, COUNT(*) 
		FROM audit_logs %s 
		GROUP BY event_type
	`, whereClause)

	rows, err := r.db.Query(ctx, eventTypeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by type: %w", err)
	}
	defer rows.Close()

	eventsByType := make(map[audit.EventType]int64)
	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan event type count: %w", err)
		}
		eventsByType[audit.EventType(eventType)] = count
	}

	// Get events by severity
	severityQuery := fmt.Sprintf(`
		SELECT severity, COUNT(*) 
		FROM audit_logs %s 
		GROUP BY severity
	`, whereClause)

	rows, err = r.db.Query(ctx, severityQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by severity: %w", err)
	}
	defer rows.Close()

	eventsBySeverity := make(map[audit.Severity]int64)
	for rows.Next() {
		var severity string
		var count int64
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity count: %w", err)
		}
		eventsBySeverity[audit.Severity(severity)] = count
	}

	// Get unique users count
	userWhereClause := whereClause
	if whereClause == "" {
		userWhereClause = "WHERE user_id IS NOT NULL"
	} else {
		userWhereClause += " AND user_id IS NOT NULL"
	}

	uniqueUsersQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT user_id) 
		FROM audit_logs %s
	`, userWhereClause)

	var uniqueUsers int64
	err = r.db.QueryRow(ctx, uniqueUsersQuery, args...).Scan(&uniqueUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count unique users: %w", err)
	}

	// Get unique IPs count
	ipWhereClause := whereClause
	if whereClause == "" {
		ipWhereClause = "WHERE ip_address IS NOT NULL"
	} else {
		ipWhereClause += " AND ip_address IS NOT NULL"
	}

	uniqueIPsQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT ip_address) 
		FROM audit_logs %s
	`, ipWhereClause)

	var uniqueIPs int64
	err = r.db.QueryRow(ctx, uniqueIPsQuery, args...).Scan(&uniqueIPs)
	if err != nil {
		return nil, fmt.Errorf("failed to count unique IPs: %w", err)
	}

	// Get time range
	timeRangeQuery := fmt.Sprintf(`
		SELECT MIN(timestamp), MAX(timestamp) 
		FROM audit_logs %s
	`, whereClause)

	var minTime, maxTime sql.NullTime
	err = r.db.QueryRow(ctx, timeRangeQuery, args...).Scan(&minTime, &maxTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get time range: %w", err)
	}

	timeRange := audit.TimeRange{}
	if minTime.Valid {
		timeRange.Start = minTime.Time
	}
	if maxTime.Valid {
		timeRange.End = maxTime.Time
	}

	return &audit.LogSummary{
		TotalEvents:      totalEvents,
		EventsByType:     eventsByType,
		EventsBySeverity: eventsBySeverity,
		UniqueUsers:      uniqueUsers,
		UniqueIPs:        uniqueIPs,
		TimeRange:        timeRange,
	}, nil
}

func (r *AuditLogRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := "DELETE FROM audit_logs WHERE timestamp < $1"
	result, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}
	return result.RowsAffected(), nil
}

func (r *AuditLogRepository) Export(ctx context.Context, filter audit.LogFilter, format string) ([]byte, error) {
	entries, _, err := r.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get entries for export: %w", err)
	}

	switch format {
	case "json":
		return json.MarshalIndent(entries, "", "  ")
	case "csv":
		return r.exportToCSV(entries)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

func (r *AuditLogRepository) exportToCSV(entries []*audit.LogEntry) ([]byte, error) {
	var csv strings.Builder
	csv.WriteString("ID,Timestamp,EventType,Severity,UserID,ActorID,EntityType,EntityID,Action,Description,IPAddress,UserAgent,RequestID,SessionID,TraceID\n")

	for _, entry := range entries {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,\"%s\",%s,%s,%s,%s,%s\n",
			entry.ID.String(),
			entry.Timestamp.Format(time.RFC3339),
			entry.EventType,
			entry.Severity,
			formatUUIDPtr(entry.UserID),
			formatUUIDPtr(entry.ActorID),
			entry.EntityType,
			entry.EntityID,
			entry.Action,
			strings.ReplaceAll(entry.Description, "\"", "\"\""),
			entry.IPAddress,
			entry.UserAgent,
			entry.RequestID,
			entry.SessionID,
			entry.TraceID,
		))
	}

	return []byte(csv.String()), nil
}

func (r *AuditLogRepository) scanLogEntry(scanner interface {
	Scan(dest ...interface{}) error
}) (*audit.LogEntry, error) {
	var entry audit.LogEntry
	var eventType, severity string
	var metadataJSON, changesJSON []byte
	var ipAddress, userAgent, description, requestID, sessionID, traceID sql.NullString

	err := scanner.Scan(
		&entry.ID,
		&entry.Timestamp,
		&eventType,
		&severity,
		&entry.UserID,
		&entry.ActorID,
		&entry.EntityType,
		&entry.EntityID,
		&entry.Action,
		&description,
		&ipAddress,
		&userAgent,
		&metadataJSON,
		&changesJSON,
		&requestID,
		&sessionID,
		&traceID,
		&entry.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, audit.ErrLogNotFound
		}
		return nil, err
	}

	entry.EventType = audit.EventType(eventType)
	entry.Severity = audit.Severity(severity)

	// Handle nullable strings
	if description.Valid {
		entry.Description = description.String
	}
	if ipAddress.Valid {
		entry.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		entry.UserAgent = userAgent.String
	}
	if requestID.Valid {
		entry.RequestID = requestID.String
	}
	if sessionID.Valid {
		entry.SessionID = sessionID.String
	}
	if traceID.Valid {
		entry.TraceID = traceID.String
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &entry.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	if changesJSON != nil {
		if err := json.Unmarshal(changesJSON, &entry.Changes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal changes: %w", err)
		}
	}

	return &entry, nil
}

func (r *AuditLogRepository) buildWhereClause(filter audit.LogFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argCount := 0

	if filter.UserID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argCount))
		args = append(args, *filter.UserID)
	}

	if filter.ActorID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argCount))
		args = append(args, *filter.ActorID)
	}

	if len(filter.EventTypes) > 0 {
		argCount++
		eventTypeStrs := make([]string, len(filter.EventTypes))
		for i, et := range filter.EventTypes {
			eventTypeStrs[i] = string(et)
		}
		conditions = append(conditions, fmt.Sprintf("event_type = ANY($%d)", argCount))
		args = append(args, eventTypeStrs)
	}

	if len(filter.Severities) > 0 {
		argCount++
		severityStrs := make([]string, len(filter.Severities))
		for i, s := range filter.Severities {
			severityStrs[i] = string(s)
		}
		conditions = append(conditions, fmt.Sprintf("severity = ANY($%d)", argCount))
		args = append(args, severityStrs)
	}

	if filter.EntityType != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argCount))
		args = append(args, filter.EntityType)
	}

	if filter.EntityID != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argCount))
		args = append(args, filter.EntityID)
	}

	if filter.IPAddress != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("ip_address = $%d", argCount))
		args = append(args, filter.IPAddress)
	}

	if !filter.StartTime.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argCount))
		args = append(args, filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		argCount++
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argCount))
		args = append(args, filter.EndTime)
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

func formatUUIDPtr(uuid *uuid.UUID) string {
	if uuid == nil {
		return ""
	}
	return uuid.String()
}
