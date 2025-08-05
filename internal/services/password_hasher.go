package services

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/victoralfred/um_sys/internal/domain/auth"
)

// BCryptPasswordHasher implements password hashing using bcrypt
type BCryptPasswordHasher struct {
	cost int
}

// NewBCryptPasswordHasher creates a new bcrypt password hasher
func NewBCryptPasswordHasher(cost int) *BCryptPasswordHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &BCryptPasswordHasher{
		cost: cost,
	}
}

// HashPassword hashes a plain text password
func (h *BCryptPasswordHasher) HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	// Validate password strength
	if err := h.validatePasswordStrength(password); err != nil {
		return "", err
	}

	// Generate hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against a hash
func (h *BCryptPasswordHasher) VerifyPassword(password, hash string) error {
	if password == "" || hash == "" {
		return auth.ErrInvalidCredentials
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return auth.ErrInvalidCredentials
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}

	return nil
}

// validatePasswordStrength checks if password meets security requirements
func (h *BCryptPasswordHasher) validatePasswordStrength(password string) error {
	// Check minimum length
	if len(password) < 8 {
		return auth.ErrPasswordTooWeak
	}

	// In production, add more checks:
	// - At least one uppercase letter
	// - At least one lowercase letter
	// - At least one digit
	// - At least one special character
	// - No common passwords
	// - No dictionary words

	return nil
}
