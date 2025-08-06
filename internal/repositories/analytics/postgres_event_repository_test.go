package analytics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/umanager?sslmode=disable")
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err)

	return db
}

func createTestUser(t *testing.T, db *sql.DB) uuid.UUID {
	userID := uuid.New()
	email := "test-" + userID.String() + "@example.com"
	username := "testuser-" + userID.String()[:8]
	_, err := db.Exec(`
		INSERT INTO users (id, email, username, password_hash, first_name, last_name) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, email, username, "hash", "Test", "User")
	require.NoError(t, err)
	return userID
}

func TestPostgresEventRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("Failed to close database: %v", closeErr)
		}
	}()

	repo := NewPostgresEventRepository(db)

	t.Run("Store and Get Event", func(t *testing.T) {
		// Create test user first
		userID := createTestUser(t, db)

		event := &analytics.Event{
			ID:     uuid.New(),
			Type:   analytics.EventTypeUserLogin,
			UserID: &userID,
			Properties: map[string]interface{}{
				"method": "email",
				"ip":     "192.168.1.1",
			},
			Context: &analytics.EventContext{
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				Path:      "/login",
			},
			Timestamp: time.Now(),
			CreatedAt: time.Now(),
		}

		// Store event
		err := repo.Store(ctx, event)
		require.NoError(t, err)

		// Get event
		retrieved, err := repo.Get(ctx, event.ID)
		require.NoError(t, err)
		assert.Equal(t, event.ID, retrieved.ID)
		assert.Equal(t, event.Type, retrieved.Type)
		assert.Equal(t, event.UserID, retrieved.UserID)
		assert.Equal(t, event.Properties["method"], retrieved.Properties["method"])
		assert.Equal(t, event.Context.IPAddress, retrieved.Context.IPAddress)
	})

	t.Run("List Events with Filter", func(t *testing.T) {
		// Create test user
		userID := createTestUser(t, db)
		sessionID := "session-123"

		// Create test events
		events := []*analytics.Event{
			{
				ID:        uuid.New(),
				Type:      analytics.EventTypeUserLogin,
				UserID:    &userID,
				SessionID: &sessionID,
				Timestamp: time.Now().Add(-2 * time.Hour),
				CreatedAt: time.Now(),
			},
			{
				ID:        uuid.New(),
				Type:      analytics.EventTypePageView,
				UserID:    &userID,
				SessionID: &sessionID,
				Timestamp: time.Now().Add(-1 * time.Hour),
				CreatedAt: time.Now(),
			},
		}

		for _, event := range events {
			err := repo.Store(ctx, event)
			require.NoError(t, err)
		}

		// Test filter by user ID
		filter := analytics.EventFilter{
			UserID: &userID,
			Limit:  10,
		}

		results, total, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)
		assert.GreaterOrEqual(t, total, int64(2))

		// Test filter by session ID
		filter.UserID = nil
		filter.SessionID = &sessionID

		results, total, err = repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)
		assert.GreaterOrEqual(t, total, int64(2))
	})

	t.Run("GetEventCounts", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour)

		counts, err := repo.GetEventCounts(ctx, startTime, now, "type")
		require.NoError(t, err)
		assert.IsType(t, map[string]int64{}, counts)
	})

	t.Run("DeleteOlderThan", func(t *testing.T) {
		// Create an old event
		oldEvent := &analytics.Event{
			ID:        uuid.New(),
			Type:      analytics.EventTypeError,
			Timestamp: time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
			CreatedAt: time.Now(),
		}

		err := repo.Store(ctx, oldEvent)
		require.NoError(t, err)

		// Delete events older than 7 days
		cutoffTime := time.Now().Add(-7 * 24 * time.Hour)
		count, err := repo.DeleteOlderThan(ctx, cutoffTime)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))

		// Verify the event was deleted
		_, err = repo.Get(ctx, oldEvent.ID)
		assert.Equal(t, analytics.ErrEventNotFound, err)
	})
}
