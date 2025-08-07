package session

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an active user session
type Session struct {
	ID           string    `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	TokenID      string    `json:"token_id"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsActive     bool      `json:"is_active"`
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// UpdateActivity updates the last activity timestamp
func (s *Session) UpdateActivity() {
	s.LastActivity = time.Now()
}

// Deactivate marks the session as inactive
func (s *Session) Deactivate() {
	s.IsActive = false
}
