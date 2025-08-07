package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// UserService handles user business logic
type UserService struct {
	userRepo user.Repository
}

// NewUserService creates a new user service
func NewUserService(userRepo user.Repository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetByEmail retrieves a user by email
func (s *UserService) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, u *user.User) error {
	return s.userRepo.Create(ctx, u)
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, u *user.User) error {
	return s.userRepo.Update(ctx, u)
}

// Delete deletes a user
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
}

// GetByUsername retrieves a user by username
func (s *UserService) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}

// UpdateLastLogin updates user's last login time
func (s *UserService) UpdateLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	return s.userRepo.UpdateLastLogin(ctx, id, loginTime)
}

// IncrementFailedLoginAttempts increments failed login attempts
func (s *UserService) IncrementFailedLoginAttempts(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.IncrementFailedLoginAttempts(ctx, id)
}
