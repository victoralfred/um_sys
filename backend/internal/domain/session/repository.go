package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for session storage operations
type Repository interface {
	// Store saves a session to the repository
	Store(ctx context.Context, session *Session) error

	// GetByID retrieves a session by its ID
	GetByID(ctx context.Context, sessionID string) (*Session, error)

	// GetByUserID retrieves all active sessions for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *Session) error

	// Delete removes a session from the repository
	Delete(ctx context.Context, sessionID string) error

	// DeleteExpired removes all expired sessions
	DeleteExpired(ctx context.Context) error

	// UpdateLastActivity updates the last activity time for a session
	UpdateLastActivity(ctx context.Context, sessionID string, lastActivity time.Time) error
}
