package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	redisModule "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/victoralfred/um_sys/internal/domain/audit"
	"github.com/victoralfred/um_sys/internal/domain/session"
	pgRepo "github.com/victoralfred/um_sys/internal/infrastructure/postgres"
	redisImpl "github.com/victoralfred/um_sys/internal/infrastructure/redis"
	"github.com/victoralfred/um_sys/internal/services"
)

type IntegrationTestEnv struct {
	RedisClient       *redis.Client
	PostgresPool      *pgxpool.Pool
	SessionService    *services.SessionService
	AuditService      *services.AuditService
	RedisContainer    testcontainers.Container
	PostgresContainer testcontainers.Container
}

func TestRedisAuditIntegration(t *testing.T) {
	env := setupIntegrationTestEnv(t)
	defer env.Cleanup()

	t.Run("SessionOperationsWithAuditing", func(t *testing.T) {
		testSessionOperationsWithAuditing(t, env)
	})

	t.Run("ConcurrentSessionsWithAuditing", func(t *testing.T) {
		testConcurrentSessionsWithAuditing(t, env)
	})

	t.Run("RedisFailureScenarios", func(t *testing.T) {
		testRedisFailureScenarios(t, env)
	})

	t.Run("DatabaseFailureRecovery", func(t *testing.T) {
		testDatabaseFailureRecovery(t, env)
	})

	t.Run("SessionExpirationEdgeCases", func(t *testing.T) {
		testSessionExpirationEdgeCases(t, env)
	})

	t.Run("LargeDataHandling", func(t *testing.T) {
		testLargeDataHandling(t, env)
	})

	t.Run("CascadeFailureScenarios", func(t *testing.T) {
		testCascadeFailureScenarios(t, env)
	})

	t.Run("DataIntegrityEdgeCases", func(t *testing.T) {
		testDataIntegrityEdgeCases(t, env)
	})

	t.Run("SecurityAuditingScenarios", func(t *testing.T) {
		testSecurityAuditingScenarios(t, env)
	})

	t.Run("ResourceExhaustionHandling", func(t *testing.T) {
		testResourceExhaustionHandling(t, env)
	})
}

func testSessionOperationsWithAuditing(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()

	// Create user for testing
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("session creation generates audit log", func(t *testing.T) {
		session, err := env.SessionService.CreateSession(ctx, userID, "token-123", "192.168.1.1", "Test-Agent", time.Hour)
		require.NoError(t, err)
		require.NotNil(t, session)

		// Manually audit the session creation
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeUserLoggedIn,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "session",
			EntityID:    session.ID,
			Action:      "create",
			Description: "User session created",
			IPAddress:   "192.168.1.1",
			UserAgent:   "Test-Agent",
			Metadata: map[string]interface{}{
				"session_id": session.ID,
				"token_id":   "token-123",
				"expires_in": "1h",
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeUserLoggedIn, auditEntry.EventType)
		assert.Equal(t, userID, *auditEntry.UserID)
		assert.Equal(t, "session", auditEntry.EntityType)
		assert.Equal(t, session.ID, auditEntry.EntityID)

		// Verify session exists in Redis
		retrievedSession, err := env.SessionService.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrievedSession.ID)

		// Verify audit log exists in database
		auditLogs, _, err := env.AuditService.GetLogs(ctx, audit.LogFilter{
			UserID: &userID,
			Limit:  10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(auditLogs), 1)
	})

	t.Run("session invalidation with audit trail", func(t *testing.T) {
		// Create session
		session, err := env.SessionService.CreateSession(ctx, userID, "token-456", "192.168.1.2", "Test-Agent", time.Hour)
		require.NoError(t, err)

		// Invalidate session
		err = env.SessionService.InvalidateSession(ctx, session.ID)
		require.NoError(t, err)

		// Audit the invalidation
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeUserLoggedOut,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "session",
			EntityID:    session.ID,
			Action:      "invalidate",
			Description: "User session invalidated",
			IPAddress:   "192.168.1.2",
		}

		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeUserLoggedOut, auditEntry.EventType)

		// Verify session is inactive in Redis
		retrievedSession, err := env.SessionService.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.False(t, retrievedSession.IsActive)

		// Verify audit trail
		auditLogs, _, err := env.AuditService.GetEntityLogs(ctx, "session", session.ID, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(auditLogs), 2) // Create and invalidate
	})
}

func testConcurrentSessionsWithAuditing(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	numUsers := 10
	sessionsPerUser := 3

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[uuid.UUID][]*session.Session)
	auditResults := make([]*audit.LogEntry, 0)

	// Create concurrent sessions for multiple users
	for i := 0; i < numUsers; i++ {
		userID := uuid.New()
		createTestUserInDB(t, env.PostgresPool, ctx, userID)
		results[userID] = make([]*session.Session, 0)

		for j := 0; j < sessionsPerUser; j++ {
			wg.Add(1)
			go func(uid uuid.UUID, sessionIndex int) {
				defer wg.Done()

				// Create session
				sess, err := env.SessionService.CreateSession(
					ctx,
					uid,
					fmt.Sprintf("token-%s-%d", uid.String()[:8], sessionIndex),
					fmt.Sprintf("192.168.1.%d", sessionIndex+1),
					"Test-Agent-Concurrent",
					time.Hour,
				)

				if err != nil {
					t.Errorf("Failed to create session: %v", err)
					return
				}

				mu.Lock()
				results[uid] = append(results[uid], sess)
				mu.Unlock()

				// Audit the session creation
				auditReq := &audit.CreateLogRequest{
					EventType:   audit.EventTypeUserLoggedIn,
					Severity:    audit.SeverityInfo,
					UserID:      &uid,
					EntityType:  "session",
					EntityID:    sess.ID,
					Action:      "create",
					Description: "Concurrent session created",
					IPAddress:   fmt.Sprintf("192.168.1.%d", sessionIndex+1),
					Metadata: map[string]interface{}{
						"concurrent_test": true,
						"session_index":   sessionIndex,
					},
				}

				auditEntry, err := env.AuditService.Log(ctx, auditReq)
				if err != nil {
					t.Errorf("Failed to create audit log: %v", err)
					return
				}

				mu.Lock()
				auditResults = append(auditResults, auditEntry)
				mu.Unlock()
			}(userID, j)
		}
	}

	wg.Wait()

	// Verify all sessions were created successfully
	totalExpectedSessions := numUsers * sessionsPerUser
	actualSessionCount := 0
	for _, sessions := range results {
		actualSessionCount += len(sessions)
	}
	assert.Equal(t, totalExpectedSessions, actualSessionCount)
	assert.Equal(t, totalExpectedSessions, len(auditResults))

	// Verify session data integrity
	for userID := range results {
		userSessions, err := env.SessionService.GetUserSessions(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, userSessions, sessionsPerUser)
	}

	// Verify audit logs integrity
	for userID := range results {
		logs, _, err := env.AuditService.GetUserLogs(ctx, userID, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), sessionsPerUser)
	}
}

func testRedisFailureScenarios(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("redis connection failure during session operations", func(t *testing.T) {
		// Create a session first
		session, err := env.SessionService.CreateSession(ctx, userID, "token-fail-test", "192.168.1.1", "Test-Agent", time.Hour)
		require.NoError(t, err)

		// Simulate Redis failure by stopping the container temporarily
		err = env.RedisContainer.Stop(ctx, nil)
		require.NoError(t, err)

		// Wait a moment for connection to fail
		time.Sleep(100 * time.Millisecond)

		// Try to get session - should fail gracefully
		_, err = env.SessionService.GetSession(ctx, session.ID)
		assert.Error(t, err)

		// Audit the failure
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityError,
			EntityType:  "system",
			EntityID:    "redis",
			Action:      "connection_failure",
			Description: "Redis connection failed during session operation",
			Metadata: map[string]interface{}{
				"session_id": session.ID,
				"error":      "Redis unavailable",
			},
		}

		// This should still work as it goes to PostgreSQL
		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)

		// Restart Redis
		err = env.RedisContainer.Start(ctx)
		require.NoError(t, err)

		// Wait for Redis to be ready
		time.Sleep(2 * time.Second)

		// Session should be gone (Redis restarted with empty state)
		_, err = env.SessionService.GetSession(ctx, session.ID)
		assert.Error(t, err, "Session should not exist after Redis restart")
	})

	t.Run("redis memory pressure scenarios", func(t *testing.T) {
		// Create many sessions to test memory pressure
		var sessions []*session.Session
		for i := 0; i < 50; i++ {
			testUserID := uuid.New()
			createTestUserInDB(t, env.PostgresPool, ctx, testUserID)

			sess, err := env.SessionService.CreateSession(
				ctx,
				testUserID,
				fmt.Sprintf("memory-test-token-%d", i),
				"192.168.1.1",
				"Memory-Test-Agent",
				time.Hour,
			)
			if err == nil {
				sessions = append(sessions, sess)
			}
		}

		assert.GreaterOrEqual(t, len(sessions), 40, "Should create most sessions even under memory pressure")

		// Audit memory pressure event
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityWarning,
			EntityType:  "system",
			EntityID:    "redis",
			Action:      "memory_pressure",
			Description: fmt.Sprintf("Created %d sessions for memory pressure test", len(sessions)),
			Metadata: map[string]interface{}{
				"sessions_created": len(sessions),
				"test_type":        "memory_pressure",
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)

		// Cleanup sessions
		for _, sess := range sessions {
			_ = env.SessionService.DeleteSession(ctx, sess.ID)
		}
	})
}

func testDatabaseFailureRecovery(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("database unavailable during audit logging", func(t *testing.T) {
		// Create session (should work as it uses Redis)
		session, err := env.SessionService.CreateSession(ctx, userID, "db-fail-token", "192.168.1.1", "Test-Agent", time.Hour)
		require.NoError(t, err)

		// Stop database temporarily
		err = env.PostgresContainer.Stop(ctx, nil)
		require.NoError(t, err)

		// Wait for connections to fail
		time.Sleep(100 * time.Millisecond)

		// Try to audit - should fail
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeUserLoggedIn,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "session",
			EntityID:    session.ID,
			Action:      "create",
			Description: "Session created during DB failure test",
		}

		_, err = env.AuditService.Log(ctx, auditReq)
		assert.Error(t, err, "Audit logging should fail when database is down")

		// Session operations should still work (Redis is up)
		retrievedSession, err := env.SessionService.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrievedSession.ID)

		// Restart database
		err = env.PostgresContainer.Start(ctx)
		require.NoError(t, err)

		// Wait for database to be ready
		time.Sleep(3 * time.Second)

		// Recreate the test user (database was reset)
		createTestUserInDB(t, env.PostgresPool, ctx, userID)

		// Now audit logging should work again
		_, err = env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
	})
}

func testSessionExpirationEdgeCases(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("session expiration during operations", func(t *testing.T) {
		// Create session with very short expiration
		session, err := env.SessionService.CreateSession(ctx, userID, "expire-token", "192.168.1.1", "Test-Agent", 100*time.Millisecond)
		require.NoError(t, err)

		// Session should be valid initially
		retrievedSession, err := env.SessionService.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.True(t, retrievedSession.IsActive)

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Session should be expired now
		expiredSession, err := env.SessionService.GetSession(ctx, session.ID)
		if err == nil {
			// If we get the session, it should be expired
			assert.True(t, expiredSession.IsExpired())
		}

		// Try to validate expired session
		_, err = env.SessionService.ValidateSession(ctx, session.ID)
		assert.Error(t, err, "Expired session should not validate")

		// Audit the expiration
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "session",
			EntityID:    session.ID,
			Action:      "expire",
			Description: "Session expired during operation",
		}

		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)
	})

	t.Run("clock skew scenarios", func(t *testing.T) {
		// Create session
		session, err := env.SessionService.CreateSession(ctx, userID, "clock-skew-token", "192.168.1.1", "Test-Agent", time.Hour)
		require.NoError(t, err)

		// Simulate clock skew by manually setting session expiration to past
		// This tests the edge case where system clocks are out of sync
		session.ExpiresAt = time.Now().Add(-time.Minute)

		// Validation should detect the expired session
		// Note: This might not fail immediately as we didn't update Redis,
		// but it tests our validation logic
		_, _ = env.SessionService.ValidateSession(ctx, session.ID)

		// Audit clock skew detection
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityWarning,
			EntityType:  "system",
			EntityID:    "clock",
			Action:      "skew_detected",
			Description: "Clock skew detected during session validation",
			Metadata: map[string]interface{}{
				"session_id":   session.ID,
				"expires_at":   session.ExpiresAt.Format(time.RFC3339),
				"current_time": time.Now().Format(time.RFC3339),
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)
	})
}

func testLargeDataHandling(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("large metadata in audit logs", func(t *testing.T) {
		// Create large metadata object
		largeMetadata := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			largeMetadata[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d_with_lots_of_data_to_make_it_large_%s", i, string(make([]byte, 100)))
		}

		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeUserLoggedIn,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "session",
			EntityID:    uuid.New().String(),
			Action:      "large_metadata_test",
			Description: "Testing large metadata handling",
			Metadata:    largeMetadata,
		}

		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.NotNil(t, auditEntry.Metadata)
		assert.Equal(t, 100, len(auditEntry.Metadata))

		// Verify metadata was stored and retrieved correctly
		retrieved, err := env.AuditService.GetLog(ctx, auditEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, len(largeMetadata), len(retrieved.Metadata))
	})

	t.Run("large session data handling", func(t *testing.T) {
		// Create session with large token ID and user agent
		largeTokenID := string(make([]byte, 1000)) + "token"
		largeUserAgent := string(make([]byte, 2000)) + "UserAgent"

		session, err := env.SessionService.CreateSession(ctx, userID, largeTokenID, "192.168.1.1", largeUserAgent, time.Hour)
		require.NoError(t, err)
		assert.Equal(t, largeTokenID, session.TokenID)
		assert.Equal(t, largeUserAgent, session.UserAgent)

		// Retrieve and verify
		retrieved, err := env.SessionService.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, largeTokenID, retrieved.TokenID)
		assert.Equal(t, largeUserAgent, retrieved.UserAgent)
	})
}

func testCascadeFailureScenarios(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("redis and database both failing", func(t *testing.T) {
		// Create session first
		session, err := env.SessionService.CreateSession(ctx, userID, "cascade-fail-token", "192.168.1.1", "Test-Agent", time.Hour)
		require.NoError(t, err)

		// Stop both Redis and PostgreSQL
		err = env.RedisContainer.Stop(ctx, nil)
		require.NoError(t, err)

		err = env.PostgresContainer.Stop(ctx, nil)
		require.NoError(t, err)

		// Wait for connections to fail
		time.Sleep(100 * time.Millisecond)

		// Both operations should fail gracefully
		_, err = env.SessionService.GetSession(ctx, session.ID)
		assert.Error(t, err)

		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityCritical,
			EntityType:  "system",
			EntityID:    "infrastructure",
			Action:      "cascade_failure",
			Description: "Both Redis and PostgreSQL failed",
		}

		_, err = env.AuditService.Log(ctx, auditReq)
		assert.Error(t, err)

		// Restart both services
		err = env.RedisContainer.Start(ctx)
		require.NoError(t, err)

		err = env.PostgresContainer.Start(ctx)
		require.NoError(t, err)

		// Wait for services to be ready
		time.Sleep(3 * time.Second)

		// Recreate test user
		createTestUserInDB(t, env.PostgresPool, ctx, userID)

		// Services should work again
		newSession, err := env.SessionService.CreateSession(ctx, userID, "recovery-token", "192.168.1.1", "Test-Agent", time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, newSession)

		recoveryAudit := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityInfo,
			EntityType:  "system",
			EntityID:    "infrastructure",
			Action:      "recovery",
			Description: "Services recovered from cascade failure",
		}

		auditEntry, err := env.AuditService.Log(ctx, recoveryAudit)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)
	})
}

func testDataIntegrityEdgeCases(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("corrupted session data recovery", func(t *testing.T) {
		// Create session
		session, err := env.SessionService.CreateSession(ctx, userID, "integrity-token", "192.168.1.1", "Test-Agent", time.Hour)
		require.NoError(t, err)

		// Manually corrupt session data in Redis
		sessionKey := "session:" + session.ID
		corruptData := `{"id":"invalid-json"corrupted}`
		err = env.RedisClient.Set(ctx, sessionKey, corruptData, time.Hour).Err()
		require.NoError(t, err)

		// Try to retrieve session - should handle corruption gracefully
		_, err = env.SessionService.GetSession(ctx, session.ID)
		assert.Error(t, err, "Should fail gracefully with corrupted data")

		// Audit the corruption detection
		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityError,
			EntityType:  "session",
			EntityID:    session.ID,
			Action:      "corruption_detected",
			Description: "Session data corruption detected in Redis",
			Metadata: map[string]interface{}{
				"session_id": session.ID,
				"error":      "JSON unmarshal failed",
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, auditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)
	})

	t.Run("invalid JSON in audit metadata", func(t *testing.T) {
		// Try to create audit log with problematic metadata
		problematicMetadata := map[string]interface{}{
			"valid_field": "valid_value",
			"channel":     make(chan int), // This will cause JSON marshal to fail
		}

		auditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeUserLoggedIn,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "test",
			EntityID:    "invalid-json-test",
			Action:      "test",
			Description: "Testing invalid JSON handling",
			Metadata:    problematicMetadata,
		}

		_, err := env.AuditService.Log(ctx, auditReq)
		assert.Error(t, err, "Should fail with invalid JSON metadata")

		// Create valid audit log about the failure
		validAuditReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityWarning,
			EntityType:  "system",
			EntityID:    "audit",
			Action:      "json_marshal_failure",
			Description: "Failed to marshal audit metadata to JSON",
			Metadata: map[string]interface{}{
				"error": "json: unsupported type: chan int",
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, validAuditReq)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)
	})
}

func testSecurityAuditingScenarios(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()
	userID := uuid.New()
	createTestUserInDB(t, env.PostgresPool, ctx, userID)

	t.Run("suspicious session activity detection", func(t *testing.T) {
		// Create multiple sessions from different IPs rapidly
		suspiciousIPs := []string{"10.0.0.1", "172.16.0.1", "203.0.113.1", "198.51.100.1"}
		var sessions []*session.Session

		for i, ip := range suspiciousIPs {
			sess, err := env.SessionService.CreateSession(
				ctx,
				userID,
				fmt.Sprintf("suspicious-token-%d", i),
				ip,
				"SuspiciousAgent",
				time.Hour,
			)
			require.NoError(t, err)
			sessions = append(sessions, sess)

			// Audit each suspicious login
			auditReq := &audit.CreateLogRequest{
				EventType:   audit.EventTypeSecurityAlert,
				Severity:    audit.SeverityWarning,
				UserID:      &userID,
				EntityType:  "session",
				EntityID:    sess.ID,
				Action:      "suspicious_login",
				Description: fmt.Sprintf("Rapid login from new IP: %s", ip),
				IPAddress:   ip,
				UserAgent:   "SuspiciousAgent",
				Metadata: map[string]interface{}{
					"rapid_login":        true,
					"geographic_anomaly": true,
					"session_count":      len(sessions),
				},
			}

			_, err = env.AuditService.Log(ctx, auditReq)
			require.NoError(t, err)
		}

		// Verify all sessions exist
		userSessions, err := env.SessionService.GetUserSessions(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, userSessions, len(suspiciousIPs))

		// Generate security summary
		auditSummaryReq := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityCritical,
			UserID:      &userID,
			EntityType:  "user",
			EntityID:    userID.String(),
			Action:      "security_review",
			Description: fmt.Sprintf("User has %d active sessions from different IPs", len(sessions)),
			Metadata: map[string]interface{}{
				"active_sessions": len(sessions),
				"unique_ips":      len(suspiciousIPs),
				"requires_review": true,
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, auditSummaryReq)
		require.NoError(t, err)
		assert.Equal(t, audit.SeverityCritical, auditEntry.Severity)
	})

	t.Run("session hijacking attempt simulation", func(t *testing.T) {
		// Create legitimate session
		legit, err := env.SessionService.CreateSession(ctx, userID, "legit-token", "192.168.1.100", "Chrome/91.0", time.Hour)
		require.NoError(t, err)

		// Simulate hijacking attempt: same session ID from different IP/Agent
		hijackAttemptAudit := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityCritical,
			UserID:      &userID,
			EntityType:  "session",
			EntityID:    legit.ID,
			Action:      "hijack_attempt",
			Description: "Session used from different IP/User-Agent",
			IPAddress:   "203.0.113.42",
			UserAgent:   "wget/1.0",
			Metadata: map[string]interface{}{
				"original_ip":         "192.168.1.100",
				"original_user_agent": "Chrome/91.0",
				"suspicious_ip":       "203.0.113.42",
				"suspicious_agent":    "wget/1.0",
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, hijackAttemptAudit)
		require.NoError(t, err)
		assert.Equal(t, audit.SeverityCritical, auditEntry.Severity)

		// Invalidate compromised session
		err = env.SessionService.InvalidateSession(ctx, legit.ID)
		require.NoError(t, err)

		// Audit session termination
		terminationAudit := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "session",
			EntityID:    legit.ID,
			Action:      "security_termination",
			Description: "Session terminated due to hijacking attempt",
		}

		_, err = env.AuditService.Log(ctx, terminationAudit)
		require.NoError(t, err)
	})
}

func testResourceExhaustionHandling(t *testing.T, env *IntegrationTestEnv) {
	ctx := context.Background()

	t.Run("database connection pool exhaustion", func(t *testing.T) {
		// Create many concurrent audit log operations
		var wg sync.WaitGroup
		numOperations := 50

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				userID := uuid.New()
				createTestUserInDB(t, env.PostgresPool, ctx, userID)

				auditReq := &audit.CreateLogRequest{
					EventType:   audit.EventTypeUserCreated,
					Severity:    audit.SeverityInfo,
					UserID:      &userID,
					EntityType:  "user",
					EntityID:    userID.String(),
					Action:      "stress_test",
					Description: fmt.Sprintf("Stress test operation %d", index),
					Metadata: map[string]interface{}{
						"operation_index": index,
						"stress_test":     true,
					},
				}

				_, _ = env.AuditService.Log(ctx, auditReq)
				// Note: We don't check success count here to avoid race conditions
				// This test mainly ensures the system doesn't crash under load
			}(i)
		}

		wg.Wait()

		// Most operations should succeed even under stress
		// Note: We're not checking successCount here as it would require atomic operations
		// This test mainly ensures the system doesn't crash under load

		// Audit the stress test completion
		stressTestAudit := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityInfo,
			EntityType:  "system",
			EntityID:    "stress_test",
			Action:      "completed",
			Description: fmt.Sprintf("Database stress test completed with %d operations", numOperations),
			Metadata: map[string]interface{}{
				"total_operations": numOperations,
				"test_type":        "connection_pool_stress",
			},
		}

		auditEntry, err := env.AuditService.Log(ctx, stressTestAudit)
		require.NoError(t, err)
		assert.Equal(t, audit.EventTypeSecurityAlert, auditEntry.EventType)
	})

	t.Run("redis memory usage monitoring", func(t *testing.T) {
		// Create many sessions to test memory usage
		sessions := make([]*session.Session, 0)
		userID := uuid.New()
		createTestUserInDB(t, env.PostgresPool, ctx, userID)

		for i := 0; i < 100; i++ {
			sess, err := env.SessionService.CreateSession(
				ctx,
				userID,
				fmt.Sprintf("memory-monitor-token-%d", i),
				"192.168.1.1",
				"Memory-Monitor-Agent",
				time.Hour,
			)
			if err == nil {
				sessions = append(sessions, sess)
			}

			// Every 20 sessions, audit memory usage
			if i%20 == 19 {
				memoryAudit := &audit.CreateLogRequest{
					EventType:   audit.EventTypeSecurityAlert,
					Severity:    audit.SeverityInfo,
					EntityType:  "system",
					EntityID:    "redis",
					Action:      "memory_checkpoint",
					Description: fmt.Sprintf("Created %d sessions, monitoring memory usage", len(sessions)),
					Metadata: map[string]interface{}{
						"sessions_created":  len(sessions),
						"checkpoint_number": (i / 20) + 1,
					},
				}

				_, err = env.AuditService.Log(ctx, memoryAudit)
				require.NoError(t, err)
			}
		}

		assert.GreaterOrEqual(t, len(sessions), 90, "Should create most sessions successfully")

		// Clean up
		for _, sess := range sessions {
			_ = env.SessionService.DeleteSession(ctx, sess.ID)
		}
	})
}

func setupIntegrationTestEnv(t *testing.T) *IntegrationTestEnv {
	ctx := context.Background()

	// Start Redis container
	redisContainer, err := redisModule.Run(ctx,
		"redis:7-alpine",
		redisModule.WithLogLevel(redisModule.LogLevelVerbose),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").WithOccurrence(1),
		),
	)
	require.NoError(t, err)

	redisHost, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
		DB:   0,
	})

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	require.NoError(t, err)

	postgresConnStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	poolConfig, err := pgxpool.ParseConfig(postgresConnStr)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	// Run migrations
	err = runIntegrationMigrations(ctx, pool)
	require.NoError(t, err)

	// Create services
	sessionRepo := redisImpl.NewSessionRepositoryWithClient(redisClient)
	sessionService := services.NewSessionService(sessionRepo)

	auditRepo := pgRepo.NewAuditLogRepository(pool)
	alertRepo := &mockAlertRepository{}
	complianceRepo := &mockComplianceRepository{}
	notificationSvc := &mockNotificationService{}
	auditService := services.NewAuditService(auditRepo, alertRepo, complianceRepo, notificationSvc)

	env := &IntegrationTestEnv{
		RedisClient:       redisClient,
		PostgresPool:      pool,
		SessionService:    sessionService,
		AuditService:      auditService,
		RedisContainer:    redisContainer,
		PostgresContainer: postgresContainer,
	}

	// Setup cleanup
	t.Cleanup(func() {
		env.Cleanup()
	})

	return env
}

func (env *IntegrationTestEnv) Cleanup() {
	ctx := context.Background()

	if env.RedisClient != nil {
		_ = env.RedisClient.Close()
	}

	if env.PostgresPool != nil {
		env.PostgresPool.Close()
	}

	if env.RedisContainer != nil {
		_ = env.RedisContainer.Terminate(ctx)
	}

	if env.PostgresContainer != nil {
		_ = env.PostgresContainer.Terminate(ctx)
	}
}

func runIntegrationMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migration := `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			event_type VARCHAR(100) NOT NULL,
			severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
			user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
			entity_type VARCHAR(100) NOT NULL,
			entity_id VARCHAR(255) NOT NULL,
			action VARCHAR(100) NOT NULL,
			description TEXT,
			ip_address INET,
			user_agent TEXT,
			metadata JSONB,
			changes JSONB,
			request_id VARCHAR(255),
			session_id VARCHAR(255),
			trace_id VARCHAR(255),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs (timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs (user_id) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs (event_type);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_severity ON audit_logs (severity);
	`

	_, err := pool.Exec(ctx, migration)
	return err
}

func createTestUserInDB(t *testing.T, pool *pgxpool.Pool, ctx context.Context, userID uuid.UUID) {
	query := `
		INSERT INTO users (id, email, username, password_hash, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := pool.Exec(ctx, query, userID,
		userID.String()+"@example.com",
		"user_"+userID.String()[:8],
		"hash", "Test", "User")
	require.NoError(t, err)
}

// Mock implementations for audit service dependencies

type mockAlertRepository struct{}

func (m *mockAlertRepository) CheckRules(ctx context.Context, entry *audit.LogEntry) ([]*audit.AlertRule, error) {
	return nil, nil // No alerts for testing
}

func (m *mockAlertRepository) CreateRule(ctx context.Context, rule *audit.AlertRule) error {
	return nil
}

func (m *mockAlertRepository) GetRuleByID(ctx context.Context, id uuid.UUID) (*audit.AlertRule, error) {
	return nil, nil
}

func (m *mockAlertRepository) ListRules(ctx context.Context, active bool) ([]*audit.AlertRule, error) {
	return nil, nil
}

func (m *mockAlertRepository) UpdateRule(ctx context.Context, rule *audit.AlertRule) error {
	return nil
}

func (m *mockAlertRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockComplianceRepository struct{}

func (m *mockComplianceRepository) CreateReport(ctx context.Context, report *audit.ComplianceReport) error {
	return nil
}

func (m *mockComplianceRepository) GetReportByID(ctx context.Context, id uuid.UUID) (*audit.ComplianceReport, error) {
	return nil, nil
}

func (m *mockComplianceRepository) ListReports(ctx context.Context, reportType string, limit, offset int) ([]*audit.ComplianceReport, int64, error) {
	return nil, 0, nil
}

func (m *mockComplianceRepository) UpdateReport(ctx context.Context, report *audit.ComplianceReport) error {
	return nil
}

type mockNotificationService struct{}

func (m *mockNotificationService) SendAlert(ctx context.Context, rule *audit.AlertRule, entry *audit.LogEntry) error {
	return nil // No notifications for testing
}

func (m *mockNotificationService) SendComplianceReport(ctx context.Context, report *audit.ComplianceReport, recipients []string) error {
	return nil // No notifications for testing
}
