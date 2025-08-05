package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// TokenService defines the interface for JWT token operations
type TokenService interface {
	// GenerateTokenPair generates access and refresh tokens for a user
	GenerateTokenPair(ctx context.Context, user *user.User) (*TokenPair, error)

	// ValidateToken validates a token and returns the claims
	ValidateToken(ctx context.Context, token string, tokenType TokenType) (*Claims, error)

	// RefreshTokens generates new token pair from refresh token
	RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error)

	// RevokeToken revokes a token (adds to blacklist)
	RevokeToken(ctx context.Context, tokenID string) error

	// IsTokenRevoked checks if a token is revoked
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Register creates a new user account
	Register(ctx context.Context, req *RegisterRequest) (*user.User, error)

	// Login authenticates a user and returns tokens
	Login(ctx context.Context, req *LoginRequest) (*TokenPair, *user.User, error)

	// Logout revokes the user's tokens
	Logout(ctx context.Context, userID uuid.UUID, tokenID string) error

	// RefreshTokens refreshes the token pair
	RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error)

	// ValidateToken validates an access token
	ValidateToken(ctx context.Context, token string) (*Claims, error)

	// RequestPasswordReset initiates password reset process
	RequestPasswordReset(ctx context.Context, email string) error

	// ResetPassword resets user password with token
	ResetPassword(ctx context.Context, token, newPassword string) error

	// ChangePassword changes user password (requires old password)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
}

// PasswordHasher defines the interface for password hashing
type PasswordHasher interface {
	// HashPassword hashes a plain text password
	HashPassword(password string) (string, error)

	// VerifyPassword verifies a password against a hash
	VerifyPassword(password, hash string) error
}

// TokenStore defines the interface for token storage (blacklist/whitelist)
type TokenStore interface {
	// Store stores a token with expiration
	Store(ctx context.Context, tokenID string, userID uuid.UUID, expiresAt time.Time) error

	// Exists checks if a token exists
	Exists(ctx context.Context, tokenID string) (bool, error)

	// Delete removes a token
	Delete(ctx context.Context, tokenID string) error

	// DeleteAllForUser removes all tokens for a user
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
}
