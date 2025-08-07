package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/domain/session"
	"github.com/victoralfred/um_sys/internal/infrastructure/redis"
)

func TestSessionRepository_Store(t *testing.T) {
	repo := setupRedisRepository(t)
	ctx := context.Background()

	sess := &session.Session{
		ID:           "test-session-1",
		UserID:       uuid.New(),
		TokenID:      "test-token-1",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	err := repo.Store(ctx, sess)
	require.NoError(t, err)

	// Verify session was stored
	retrieved, err := repo.GetByID(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, retrieved.ID)
	assert.Equal(t, sess.UserID, retrieved.UserID)
	assert.Equal(t, sess.TokenID, retrieved.TokenID)
}

func TestSessionRepository_GetByID(t *testing.T) {
	repo := setupRedisRepository(t)
	ctx := context.Background()

	// Test non-existent session
	_, err := repo.GetByID(ctx, "non-existent")
	assert.Error(t, err)

	// Test existing session
	sess := &session.Session{
		ID:           "test-session-2",
		UserID:       uuid.New(),
		TokenID:      "test-token-2",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	err = repo.Store(ctx, sess)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, retrieved.ID)
}

func TestSessionRepository_GetByUserID(t *testing.T) {
	repo := setupRedisRepository(t)
	ctx := context.Background()
	userID := uuid.New()

	// Test no sessions for user
	sessions, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Create multiple sessions for user
	sess1 := &session.Session{
		ID:           "user-session-1",
		UserID:       userID,
		TokenID:      "token-1",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	sess2 := &session.Session{
		ID:           "user-session-2",
		UserID:       userID,
		TokenID:      "token-2",
		IPAddress:    "127.0.0.2",
		UserAgent:    "Test-Agent/2.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	err = repo.Store(ctx, sess1)
	require.NoError(t, err)
	err = repo.Store(ctx, sess2)
	require.NoError(t, err)

	sessions, err = repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestSessionRepository_Update(t *testing.T) {
	repo := setupRedisRepository(t)
	ctx := context.Background()

	sess := &session.Session{
		ID:           "update-session",
		UserID:       uuid.New(),
		TokenID:      "update-token",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	err := repo.Store(ctx, sess)
	require.NoError(t, err)

	// Update session
	sess.IsActive = false
	sess.LastActivity = time.Now().Add(time.Minute)

	err = repo.Update(ctx, sess)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, sess.ID)
	require.NoError(t, err)
	assert.False(t, updated.IsActive)
}

func TestSessionRepository_Delete(t *testing.T) {
	repo := setupRedisRepository(t)
	ctx := context.Background()

	sess := &session.Session{
		ID:           "delete-session",
		UserID:       uuid.New(),
		TokenID:      "delete-token",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	err := repo.Store(ctx, sess)
	require.NoError(t, err)

	// Delete session
	err = repo.Delete(ctx, sess.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, sess.ID)
	assert.Error(t, err)
}

func TestSessionRepository_UpdateLastActivity(t *testing.T) {
	repo := setupRedisRepository(t)
	ctx := context.Background()

	sess := &session.Session{
		ID:           "activity-session",
		UserID:       uuid.New(),
		TokenID:      "activity-token",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	err := repo.Store(ctx, sess)
	require.NoError(t, err)

	// Update last activity
	newActivity := time.Now().Add(10 * time.Minute)
	err = repo.UpdateLastActivity(ctx, sess.ID, newActivity)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, sess.ID)
	require.NoError(t, err)
	assert.WithinDuration(t, newActivity, updated.LastActivity, time.Second)
}

func TestSessionRepository_DeleteExpired(t *testing.T) {
	repo := setupRedisRepository(t)
	ctx := context.Background()

	// Create expired session
	expiredSess := &session.Session{
		ID:           "expired-session",
		UserID:       uuid.New(),
		TokenID:      "expired-token",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now().Add(-2 * time.Hour),
		LastActivity: time.Now().Add(-2 * time.Hour),
		ExpiresAt:    time.Now().Add(-time.Hour),
		IsActive:     true,
	}

	// Create active session
	activeSess := &session.Session{
		ID:           "active-session",
		UserID:       uuid.New(),
		TokenID:      "active-token",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent/1.0",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
	}

	err := repo.Store(ctx, expiredSess)
	require.NoError(t, err)
	err = repo.Store(ctx, activeSess)
	require.NoError(t, err)

	// Delete expired sessions
	err = repo.DeleteExpired(ctx)
	require.NoError(t, err)

	// Verify expired session is gone
	_, err = repo.GetByID(ctx, expiredSess.ID)
	assert.Error(t, err)

	// Verify active session still exists
	_, err = repo.GetByID(ctx, activeSess.ID)
	require.NoError(t, err)
}

func setupRedisRepository(t *testing.T) session.Repository {
	// This will fail until we implement the Redis repository
	repo := redis.NewSessionRepository("localhost:6380", 0, "")
	require.NotNil(t, repo)
	return repo
}
