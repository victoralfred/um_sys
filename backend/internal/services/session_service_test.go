package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/infrastructure/redis"
	"github.com/victoralfred/um_sys/internal/services"
)

func TestSessionService_CreateSession(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	session, err := service.CreateSession(ctx, userID, "token-123", "127.0.0.1", "Test-Agent", time.Hour)
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.NotEmpty(t, session.ID)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, "token-123", session.TokenID)
	assert.Equal(t, "127.0.0.1", session.IPAddress)
	assert.Equal(t, "Test-Agent", session.UserAgent)
	assert.True(t, session.IsActive)
	assert.False(t, session.IsExpired())
}

func TestSessionService_GetSession(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	// Create session
	session, err := service.CreateSession(ctx, userID, "token-123", "127.0.0.1", "Test-Agent", time.Hour)
	require.NoError(t, err)

	// Get session
	retrieved, err := service.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)

	// Test non-existent session
	_, err = service.GetSession(ctx, "non-existent")
	assert.Error(t, err)
}

func TestSessionService_GetUserSessions(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	// No sessions initially
	sessions, err := service.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Create multiple sessions
	session1, err := service.CreateSession(ctx, userID, "token-1", "127.0.0.1", "Agent-1", time.Hour)
	require.NoError(t, err)

	session2, err := service.CreateSession(ctx, userID, "token-2", "127.0.0.2", "Agent-2", time.Hour)
	require.NoError(t, err)

	// Get user sessions
	sessions, err = service.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	sessionIDs := []string{sessions[0].ID, sessions[1].ID}
	assert.Contains(t, sessionIDs, session1.ID)
	assert.Contains(t, sessionIDs, session2.ID)
}

func TestSessionService_InvalidateSession(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	// Create session
	session, err := service.CreateSession(ctx, userID, "token-123", "127.0.0.1", "Test-Agent", time.Hour)
	require.NoError(t, err)
	assert.True(t, session.IsActive)

	// Invalidate session
	err = service.InvalidateSession(ctx, session.ID)
	require.NoError(t, err)

	// Verify session is inactive
	retrieved, err := service.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.IsActive)
}

func TestSessionService_InvalidateUserSessions(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	// Create multiple sessions
	session1, err := service.CreateSession(ctx, userID, "token-1", "127.0.0.1", "Agent-1", time.Hour)
	require.NoError(t, err)

	session2, err := service.CreateSession(ctx, userID, "token-2", "127.0.0.2", "Agent-2", time.Hour)
	require.NoError(t, err)

	// Invalidate all user sessions
	err = service.InvalidateUserSessions(ctx, userID)
	require.NoError(t, err)

	// Verify all sessions are inactive
	retrieved1, err := service.GetSession(ctx, session1.ID)
	require.NoError(t, err)
	assert.False(t, retrieved1.IsActive)

	retrieved2, err := service.GetSession(ctx, session2.ID)
	require.NoError(t, err)
	assert.False(t, retrieved2.IsActive)
}

func TestSessionService_DeleteSession(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	// Create session
	session, err := service.CreateSession(ctx, userID, "token-123", "127.0.0.1", "Test-Agent", time.Hour)
	require.NoError(t, err)

	// Delete session
	err = service.DeleteSession(ctx, session.ID)
	require.NoError(t, err)

	// Verify session is gone
	_, err = service.GetSession(ctx, session.ID)
	assert.Error(t, err)
}

func TestSessionService_ValidateSession(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	// Create valid session
	session, err := service.CreateSession(ctx, userID, "token-123", "127.0.0.1", "Test-Agent", time.Hour)
	require.NoError(t, err)

	// Validate valid session
	validated, err := service.ValidateSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, validated.ID)

	// Create expired session
	expiredSession, err := service.CreateSession(ctx, userID, "token-expired", "127.0.0.1", "Test-Agent", -time.Hour)
	require.NoError(t, err)

	// Validate expired session should fail
	_, err = service.ValidateSession(ctx, expiredSession.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")

	// Invalidate session
	err = service.InvalidateSession(ctx, session.ID)
	require.NoError(t, err)

	// Validate inactive session should fail
	_, err = service.ValidateSession(ctx, session.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestSessionService_CleanupExpiredSessions(t *testing.T) {
	service := setupSessionService(t)
	ctx := context.Background()
	userID := uuid.New()

	// Create expired session
	expiredSession, err := service.CreateSession(ctx, userID, "token-expired", "127.0.0.1", "Test-Agent", -time.Hour)
	require.NoError(t, err)

	// Create active session
	activeSession, err := service.CreateSession(ctx, userID, "token-active", "127.0.0.2", "Test-Agent", time.Hour)
	require.NoError(t, err)

	// Cleanup expired sessions
	err = service.CleanupExpiredSessions(ctx)
	require.NoError(t, err)

	// Verify expired session is gone
	_, err = service.GetSession(ctx, expiredSession.ID)
	assert.Error(t, err)

	// Verify active session still exists
	_, err = service.GetSession(ctx, activeSession.ID)
	require.NoError(t, err)
}

func setupSessionService(t *testing.T) *services.SessionService {
	repo := redis.NewSessionRepository("localhost:6380", 0, "")
	require.NotNil(t, repo)
	return services.NewSessionService(repo)
}
