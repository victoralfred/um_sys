package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/victoralfred/um_sys/internal/domain/session"
)

const (
	sessionKeyPrefix     = "session:"
	userSessionKeyPrefix = "user_sessions:"
)

// SessionRepository implements session.Repository using Redis
type SessionRepository struct {
	client *redis.Client
}

// NewSessionRepository creates a new Redis session repository
func NewSessionRepository(addr string, db int, password string) *SessionRepository {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &SessionRepository{
		client: rdb,
	}
}

// NewSessionRepositoryWithClient creates a new Redis session repository with existing client (for testing)
func NewSessionRepositoryWithClient(client *redis.Client) *SessionRepository {
	return &SessionRepository{
		client: client,
	}
}

// Store saves a session to Redis
func (r *SessionRepository) Store(ctx context.Context, sess *session.Session) error {
	// Serialize session to JSON
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	sessionKey := sessionKeyPrefix + sess.ID
	userSessionsKey := userSessionKeyPrefix + sess.UserID.String()

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Store session data with expiration
	ttl := time.Until(sess.ExpiresAt)
	if ttl > 0 {
		pipe.Set(ctx, sessionKey, data, ttl)
	} else {
		pipe.Set(ctx, sessionKey, data, 0)
	}

	// Add session ID to user's session set
	pipe.SAdd(ctx, userSessionsKey, sess.ID)
	pipe.Expire(ctx, userSessionsKey, 24*time.Hour) // Keep user sessions index for 24h

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}

// GetByID retrieves a session by its ID
func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*session.Session, error) {
	sessionKey := sessionKeyPrefix + sessionID

	data, err := r.client.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var sess session.Session
	if err := json.Unmarshal([]byte(data), &sess); err != nil {
		return nil, fmt.Errorf("failed to deserialize session: %w", err)
	}

	return &sess, nil
}

// GetByUserID retrieves all active sessions for a user
func (r *SessionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*session.Session, error) {
	userSessionsKey := userSessionKeyPrefix + userID.String()

	// Get all session IDs for the user
	sessionIDs, err := r.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		if err == redis.Nil {
			return []*session.Session{}, nil
		}
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return []*session.Session{}, nil
	}

	// Get all sessions in parallel
	var sessions []*session.Session
	for _, sessionID := range sessionIDs {
		sess, err := r.GetByID(ctx, sessionID)
		if err != nil {
			// If session doesn't exist, remove it from user's set
			r.client.SRem(ctx, userSessionsKey, sessionID)
			continue
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// Update updates an existing session
func (r *SessionRepository) Update(ctx context.Context, sess *session.Session) error {
	// Check if session exists
	sessionKey := sessionKeyPrefix + sess.ID
	exists, err := r.client.Exists(ctx, sessionKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}

	if exists == 0 {
		return fmt.Errorf("session not found: %s", sess.ID)
	}

	// Store updated session
	return r.Store(ctx, sess)
}

// Delete removes a session from Redis
func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	sessionKey := sessionKeyPrefix + sessionID

	// Get session to find user ID
	sess, err := r.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	userSessionsKey := userSessionKeyPrefix + sess.UserID.String()

	// Use pipeline for atomic deletion
	pipe := r.client.Pipeline()
	pipe.Del(ctx, sessionKey)
	pipe.SRem(ctx, userSessionsKey, sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteExpired removes all expired sessions
func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	// Use SCAN to iterate through all session keys
	iter := r.client.Scan(ctx, 0, sessionKeyPrefix+"*", 0).Iterator()
	var expiredKeys []string

	for iter.Next(ctx) {
		key := iter.Val()
		sessionID := strings.TrimPrefix(key, sessionKeyPrefix)

		// Check if session exists and get its data
		sess, err := r.GetByID(ctx, sessionID)
		if err != nil {
			// Session doesn't exist or can't be read, mark for cleanup
			expiredKeys = append(expiredKeys, key)
			continue
		}

		// Check if session is expired
		if sess.IsExpired() {
			expiredKeys = append(expiredKeys, key)

			// Also remove from user's session set
			userSessionsKey := userSessionKeyPrefix + sess.UserID.String()
			r.client.SRem(ctx, userSessionsKey, sessionID)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan sessions: %w", err)
	}

	// Delete all expired sessions
	if len(expiredKeys) > 0 {
		if err := r.client.Del(ctx, expiredKeys...).Err(); err != nil {
			return fmt.Errorf("failed to delete expired sessions: %w", err)
		}
	}

	return nil
}

// UpdateLastActivity updates the last activity time for a session
func (r *SessionRepository) UpdateLastActivity(ctx context.Context, sessionID string, lastActivity time.Time) error {
	sess, err := r.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	sess.LastActivity = lastActivity
	return r.Update(ctx, sess)
}
