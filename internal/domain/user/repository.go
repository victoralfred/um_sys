package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for user persistence
type Repository interface {
	// Create creates a new user
	Create(ctx context.Context, user *User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*User, error)

	// GetByUsername retrieves a user by username
	GetByUsername(ctx context.Context, username string) (*User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *User) error

	// Delete soft deletes a user
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves a paginated list of users
	List(ctx context.Context, filter ListFilter) ([]*User, int64, error)

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error

	// UpdateMFA updates MFA settings for a user
	UpdateMFA(ctx context.Context, id uuid.UUID, enabled bool, secret string, backupCodes []string) error

	// ExistsByEmail checks if a user exists with the given email
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByUsername checks if a user exists with the given username
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}
