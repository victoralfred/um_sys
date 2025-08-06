package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/domain/audit"
)

func TestAuditLogRepository_Create(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewAuditLogRepository(testDB.Pool)
	ctx := context.Background()

	t.Run("create basic audit log entry", func(t *testing.T) {
		// Create a user first to satisfy foreign key constraint
		userID := createTestUser(t, testDB, ctx)

		entry := &audit.LogEntry{
			ID:          uuid.New(),
			Timestamp:   time.Now(),
			EventType:   audit.EventTypeUserCreated,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "user",
			EntityID:    userID.String(),
			Action:      "create",
			Description: "User account created",
			IPAddress:   "192.168.1.1",
			UserAgent:   "Mozilla/5.0",
			RequestID:   "req-123",
			SessionID:   "sess-456",
			TraceID:     "trace-789",
			CreatedAt:   time.Now(),
		}

		err := repo.Create(ctx, entry)
		require.NoError(t, err)

		// Verify entry was created
		retrieved, err := repo.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entry.ID, retrieved.ID)
		assert.Equal(t, entry.EventType, retrieved.EventType)
		assert.Equal(t, entry.Severity, retrieved.Severity)
		assert.Equal(t, entry.EntityType, retrieved.EntityType)
		assert.Equal(t, entry.Action, retrieved.Action)
	})

	t.Run("create audit log with metadata and changes", func(t *testing.T) {
		entry := &audit.LogEntry{
			ID:          uuid.New(),
			Timestamp:   time.Now(),
			EventType:   audit.EventTypeUserUpdated,
			Severity:    audit.SeverityInfo,
			EntityType:  "user",
			EntityID:    uuid.New().String(),
			Action:      "update",
			Description: "User profile updated",
			Metadata: map[string]interface{}{
				"field_updated": "email",
				"update_reason": "user_request",
			},
			Changes: &audit.Changes{
				Before: []byte(`{"email": "old@example.com"}`),
				After:  []byte(`{"email": "new@example.com"}`),
				Fields: []string{"email"},
			},
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, entry)
		require.NoError(t, err)

		// Verify metadata and changes were stored correctly
		retrieved, err := repo.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata)
		assert.Equal(t, "email", retrieved.Metadata["field_updated"])
		assert.NotNil(t, retrieved.Changes)
		assert.Equal(t, []string{"email"}, retrieved.Changes.Fields)
	})
}

func TestAuditLogRepository_GetByID(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewAuditLogRepository(testDB.Pool)
	ctx := context.Background()

	t.Run("get existing audit log", func(t *testing.T) {
		entry := createTestAuditLog(t, repo, ctx)

		retrieved, err := repo.GetByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entry.ID, retrieved.ID)
		assert.Equal(t, entry.EventType, retrieved.EventType)
	})

	t.Run("get non-existent audit log", func(t *testing.T) {
		nonExistentID := uuid.New()

		_, err := repo.GetByID(ctx, nonExistentID)
		assert.ErrorIs(t, err, audit.ErrLogNotFound)
	})
}

func TestAuditLogRepository_List(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewAuditLogRepository(testDB.Pool)
	ctx := context.Background()

	// Create test data with real users
	userID1 := createTestUser(t, testDB, ctx)
	userID2 := createTestUser(t, testDB, ctx)
	now := time.Now()

	entries := []*audit.LogEntry{
		{
			ID:         uuid.New(),
			Timestamp:  now.Add(-3 * time.Hour),
			EventType:  audit.EventTypeUserCreated,
			Severity:   audit.SeverityInfo,
			UserID:     &userID1,
			EntityType: "user",
			EntityID:   userID1.String(),
			Action:     "create",
			IPAddress:  "192.168.1.1",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			Timestamp:  now.Add(-2 * time.Hour),
			EventType:  audit.EventTypeUserLoggedIn,
			Severity:   audit.SeverityInfo,
			UserID:     &userID1,
			EntityType: "user",
			EntityID:   userID1.String(),
			Action:     "login",
			IPAddress:  "192.168.1.1",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			Timestamp:  now.Add(-1 * time.Hour),
			EventType:  audit.EventTypeUserCreated,
			Severity:   audit.SeverityInfo,
			UserID:     &userID2,
			EntityType: "user",
			EntityID:   userID2.String(),
			Action:     "create",
			IPAddress:  "192.168.1.2",
			CreatedAt:  now,
		},
	}

	for _, entry := range entries {
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	t.Run("list all entries", func(t *testing.T) {
		filter := audit.LogFilter{
			Limit: 10,
		}

		results, total, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, int(total), len(entries))
		assert.GreaterOrEqual(t, len(results), len(entries))

		// Should be ordered by timestamp DESC
		if len(results) >= 2 {
			assert.True(t, results[0].Timestamp.After(results[1].Timestamp) ||
				results[0].Timestamp.Equal(results[1].Timestamp))
		}
	})

	t.Run("filter by user ID", func(t *testing.T) {
		filter := audit.LogFilter{
			UserID: &userID1,
			Limit:  10,
		}

		results, total, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total) // userID1 has 2 entries
		assert.Len(t, results, 2)

		for _, result := range results {
			assert.Equal(t, userID1, *result.UserID)
		}
	})

	t.Run("filter by event type", func(t *testing.T) {
		filter := audit.LogFilter{
			EventTypes: []audit.EventType{audit.EventTypeUserCreated},
			Limit:      10,
		}

		results, total, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(2)) // At least 2 user creation events

		for _, result := range results {
			assert.Equal(t, audit.EventTypeUserCreated, result.EventType)
		}
	})

	t.Run("filter by time range", func(t *testing.T) {
		filter := audit.LogFilter{
			StartTime: now.Add(-2*time.Hour + 30*time.Minute),
			EndTime:   now,
			Limit:     10,
		}

		results, total, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1)) // Should include at least 1 entry in the time range

		for _, result := range results {
			assert.True(t, result.Timestamp.After(filter.StartTime) || result.Timestamp.Equal(filter.StartTime))
			assert.True(t, result.Timestamp.Before(filter.EndTime) || result.Timestamp.Equal(filter.EndTime))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		filter := audit.LogFilter{
			Limit:  1,
			Offset: 0,
		}

		page1, total, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, page1, 1)
		assert.GreaterOrEqual(t, total, int64(3))

		filter.Offset = 1
		page2, _, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, page2, 1)

		// Pages should be different
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
	})
}

func TestAuditLogRepository_GetSummary(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewAuditLogRepository(testDB.Pool)
	ctx := context.Background()

	// Create test data with different types and severities
	userID1 := createTestUser(t, testDB, ctx)
	userID2 := createTestUser(t, testDB, ctx)
	now := time.Now()

	entries := []*audit.LogEntry{
		{
			ID:         uuid.New(),
			Timestamp:  now.Add(-2 * time.Hour),
			EventType:  audit.EventTypeUserCreated,
			Severity:   audit.SeverityInfo,
			UserID:     &userID1,
			EntityType: "user",
			EntityID:   userID1.String(),
			Action:     "create",
			IPAddress:  "192.168.1.1",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			Timestamp:  now.Add(-1 * time.Hour),
			EventType:  audit.EventTypeUserLoggedIn,
			Severity:   audit.SeverityInfo,
			UserID:     &userID2,
			EntityType: "user",
			EntityID:   userID2.String(),
			Action:     "login",
			IPAddress:  "192.168.1.2",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			Timestamp:  now.Add(-30 * time.Minute),
			EventType:  audit.EventTypeSecurityAlert,
			Severity:   audit.SeverityError,
			EntityType: "system",
			EntityID:   "security",
			Action:     "alert",
			IPAddress:  "192.168.1.3",
			CreatedAt:  now,
		},
	}

	for _, entry := range entries {
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	t.Run("get summary with all data", func(t *testing.T) {
		filter := audit.LogFilter{}

		summary, err := repo.GetSummary(ctx, filter)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, summary.TotalEvents, int64(3))
		assert.GreaterOrEqual(t, summary.EventsByType[audit.EventTypeUserCreated], int64(1))
		assert.GreaterOrEqual(t, summary.EventsByType[audit.EventTypeUserLoggedIn], int64(1))
		assert.GreaterOrEqual(t, summary.EventsByType[audit.EventTypeSecurityAlert], int64(1))
		assert.GreaterOrEqual(t, summary.EventsBySeverity[audit.SeverityInfo], int64(2))
		assert.GreaterOrEqual(t, summary.EventsBySeverity[audit.SeverityError], int64(1))
		assert.GreaterOrEqual(t, summary.UniqueUsers, int64(2))
		assert.GreaterOrEqual(t, summary.UniqueIPs, int64(3))
		assert.False(t, summary.TimeRange.Start.IsZero())
		assert.False(t, summary.TimeRange.End.IsZero())
	})

	t.Run("get summary with user filter", func(t *testing.T) {
		filter := audit.LogFilter{
			UserID: &userID1,
		}

		summary, err := repo.GetSummary(ctx, filter)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, summary.TotalEvents, int64(1))
		assert.Equal(t, int64(1), summary.UniqueUsers)
	})
}

func TestAuditLogRepository_DeleteOlderThan(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewAuditLogRepository(testDB.Pool)
	ctx := context.Background()

	now := time.Now()

	// Create old and recent entries
	oldEntry := &audit.LogEntry{
		ID:         uuid.New(),
		Timestamp:  now.Add(-100 * 24 * time.Hour), // 100 days ago
		EventType:  audit.EventTypeUserCreated,
		Severity:   audit.SeverityInfo,
		EntityType: "user",
		EntityID:   uuid.New().String(),
		Action:     "create",
		CreatedAt:  now,
	}

	recentEntry := &audit.LogEntry{
		ID:         uuid.New(),
		Timestamp:  now.Add(-1 * time.Hour), // 1 hour ago
		EventType:  audit.EventTypeUserCreated,
		Severity:   audit.SeverityInfo,
		EntityType: "user",
		EntityID:   uuid.New().String(),
		Action:     "create",
		CreatedAt:  now,
	}

	err := repo.Create(ctx, oldEntry)
	require.NoError(t, err)
	err = repo.Create(ctx, recentEntry)
	require.NoError(t, err)

	t.Run("delete old entries", func(t *testing.T) {
		cutoff := now.Add(-30 * 24 * time.Hour) // 30 days ago
		deletedCount, err := repo.DeleteOlderThan(ctx, cutoff)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, deletedCount, int64(1))

		// Old entry should be gone
		_, err = repo.GetByID(ctx, oldEntry.ID)
		assert.ErrorIs(t, err, audit.ErrLogNotFound)

		// Recent entry should still exist
		_, err = repo.GetByID(ctx, recentEntry.ID)
		require.NoError(t, err)
	})
}

func TestAuditLogRepository_Export(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewAuditLogRepository(testDB.Pool)
	ctx := context.Background()

	// Create test entry
	entry := createTestAuditLog(t, repo, ctx)

	t.Run("export to JSON", func(t *testing.T) {
		filter := audit.LogFilter{Limit: 10}
		data, err := repo.Export(ctx, filter, "json")
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), entry.ID.String())
	})

	t.Run("export to CSV", func(t *testing.T) {
		filter := audit.LogFilter{Limit: 10}
		data, err := repo.Export(ctx, filter, "csv")
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), "ID,Timestamp,EventType") // CSV header
		assert.Contains(t, string(data), entry.ID.String())
	})

	t.Run("unsupported export format", func(t *testing.T) {
		filter := audit.LogFilter{Limit: 10}
		_, err := repo.Export(ctx, filter, "xml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export format")
	})
}

func createTestUser(t *testing.T, testDB *TestDatabase, ctx context.Context) uuid.UUID {
	userID := uuid.New()
	query := `
		INSERT INTO users (id, email, username, password_hash, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := testDB.Pool.Exec(ctx, query, userID,
		userID.String()+"@example.com", // Unique email
		"user_"+userID.String()[:8],    // Unique username
		"hash", "Test", "User")
	require.NoError(t, err)
	return userID
}

func createTestAuditLog(t *testing.T, repo *AuditLogRepository, ctx context.Context) *audit.LogEntry {
	// Create audit log without user reference to avoid foreign key issues
	entry := &audit.LogEntry{
		ID:          uuid.New(),
		Timestamp:   time.Now(),
		EventType:   audit.EventTypeUserCreated,
		Severity:    audit.SeverityInfo,
		EntityType:  "user",
		EntityID:    uuid.New().String(),
		Action:      "create",
		Description: "Test user creation",
		IPAddress:   "127.0.0.1",
		UserAgent:   "Test-Agent",
		RequestID:   "test-req-123",
		CreatedAt:   time.Now(),
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)
	return entry
}
