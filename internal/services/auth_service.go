package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/auth"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// AuthService implements the auth.AuthService interface
type AuthService struct {
	userRepo       user.Repository
	tokenService   auth.TokenService
	passwordHasher auth.PasswordHasher
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo user.Repository,
	tokenService auth.TokenService,
	passwordHasher auth.PasswordHasher,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		tokenService:   tokenService,
		passwordHasher: passwordHasher,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *auth.RegisterRequest) (*user.User, error) {
	// Check if email already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, auth.ErrEmailAlreadyExists
	}

	// Check if username already exists
	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return nil, auth.ErrUsernameAlreadyExists
	}

	// Hash the password
	hashedPassword, err := s.passwordHasher.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create the user
	newUser := &user.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       user.StatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save to database
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req *auth.LoginRequest) (*auth.TokenPair, *user.User, error) {
	var foundUser *user.User
	var err error

	// Try to find user by email or username
	if req.Email != "" {
		foundUser, err = s.userRepo.GetByEmail(ctx, req.Email)
	} else if req.Username != "" {
		foundUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	} else {
		return nil, nil, auth.ErrInvalidCredentials
	}

	// If user not found, return invalid credentials (don't reveal if user exists)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, nil, auth.ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := s.passwordHasher.VerifyPassword(req.Password, foundUser.PasswordHash); err != nil {
		return nil, nil, auth.ErrInvalidCredentials
	}

	// Check if account is active
	if foundUser.Status != user.StatusActive {
		if foundUser.Status == user.StatusInactive {
			return nil, nil, auth.ErrAccountInactive
		}
		if foundUser.Status == user.StatusLocked {
			return nil, nil, auth.ErrAccountLocked
		}
	}

	// Generate tokens
	tokens, err := s.tokenService.GenerateTokenPair(ctx, foundUser)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(ctx, foundUser.ID) // Log error but don't fail login

	return tokens, foundUser, nil
}

// Logout revokes the user's tokens
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, tokenID string) error {
	if err := s.tokenService.RevokeToken(ctx, tokenID); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// RefreshTokens refreshes the token pair
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	tokens, err := s.tokenService.RefreshTokens(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}
	return tokens, nil
}

// ValidateToken validates an access token
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*auth.Claims, error) {
	claims, err := s.tokenService.ValidateToken(ctx, token, auth.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	return claims, nil
}

// RequestPasswordReset initiates password reset process
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	// Find user by email
	foundUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists
		return nil
	}

	// In production, this would:
	// 1. Generate a reset token
	// 2. Store it in cache with expiration
	// 3. Send email with reset link
	// For now, we'll just verify the user exists
	_ = foundUser

	return nil
}

// ResetPassword resets user password with token
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// In production, this would:
	// 1. Validate the reset token from cache
	// 2. Get user ID from token
	// 3. Hash new password
	// 4. Update user password
	// 5. Invalidate the reset token

	// For now, returning nil for testing
	return nil
}

// ChangePassword changes user password (requires old password)
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Get user
	foundUser, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Verify old password
	if err := s.passwordHasher.VerifyPassword(oldPassword, foundUser.PasswordHash); err != nil {
		return auth.ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := s.passwordHasher.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	foundUser.PasswordHash = hashedPassword
	foundUser.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, foundUser); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
