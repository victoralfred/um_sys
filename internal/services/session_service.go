package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/session"
)

// SessionService handles session management operations
type SessionService struct {
	repo session.Repository
}

// NewSessionService creates a new session service
func NewSessionService(repo session.Repository) *SessionService {
	return &SessionService{
		repo: repo,
	}
}

// CreateSession creates a new user session
func (s *SessionService) CreateSession(ctx context.Context, userID uuid.UUID, tokenID, ipAddress, userAgent string, expiresIn time.Duration) (*session.Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	sess := &session.Session{
		ID:           sessionID,
		UserID:       userID,
		TokenID:      tokenID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    now,
		LastActivity: now,
		ExpiresAt:    now.Add(expiresIn),
		IsActive:     true,
	}

	if err := s.repo.Store(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return sess, nil
}

// GetSession retrieves a session by ID and updates last activity
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*session.Session, error) {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Update last activity
	if sess.IsActive && !sess.IsExpired() {
		sess.UpdateActivity()
		if err := s.repo.Update(ctx, sess); err != nil {
			// Log error but don't fail the request
			return sess, nil
		}
	}

	return sess, nil
}

// GetUserSessions retrieves all active sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*session.Session, error) {
	sessions, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Filter out expired sessions
	var activeSessions []*session.Session
	for _, sess := range sessions {
		if !sess.IsExpired() && sess.IsActive {
			activeSessions = append(activeSessions, sess)
		}
	}

	return activeSessions, nil
}

// InvalidateSession deactivates a session
func (s *SessionService) InvalidateSession(ctx context.Context, sessionID string) error {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	sess.Deactivate()
	return s.repo.Update(ctx, sess)
}

// InvalidateUserSessions deactivates all sessions for a user
func (s *SessionService) InvalidateUserSessions(ctx context.Context, userID uuid.UUID) error {
	sessions, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	for _, sess := range sessions {
		if sess.IsActive {
			sess.Deactivate()
			if err := s.repo.Update(ctx, sess); err != nil {
				return fmt.Errorf("failed to invalidate session %s: %w", sess.ID, err)
			}
		}
	}

	return nil
}

// DeleteSession permanently removes a session
func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	return s.repo.Delete(ctx, sessionID)
}

// CleanupExpiredSessions removes expired sessions
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) error {
	return s.repo.DeleteExpired(ctx)
}

// ValidateSession checks if a session is valid and active
func (s *SessionService) ValidateSession(ctx context.Context, sessionID string) (*session.Session, error) {
	sess, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if sess.IsExpired() {
		return nil, fmt.Errorf("session has expired")
	}

	if !sess.IsActive {
		return nil, fmt.Errorf("session is not active")
	}

	return sess, nil
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
