package mfa

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for MFA persistence
type Repository interface {
	// GetSettings retrieves MFA settings for a user
	GetSettings(ctx context.Context, userID uuid.UUID) (*Settings, error)

	// SaveSettings saves or updates MFA settings
	SaveSettings(ctx context.Context, settings *Settings) error

	// DeleteSettings deletes MFA settings for a user
	DeleteSettings(ctx context.Context, userID uuid.UUID) error

	// SaveChallenge saves an MFA challenge
	SaveChallenge(ctx context.Context, challenge *Challenge) error

	// GetChallenge retrieves an MFA challenge
	GetChallenge(ctx context.Context, id uuid.UUID) (*Challenge, error)

	// DeleteChallenge deletes an MFA challenge
	DeleteChallenge(ctx context.Context, id uuid.UUID) error

	// IncrementChallengeAttempts increments the attempt counter for a challenge
	IncrementChallengeAttempts(ctx context.Context, id uuid.UUID) error

	// SaveBackupCode saves a backup code
	SaveBackupCode(ctx context.Context, userID uuid.UUID, code *BackupCode) error

	// GetBackupCodes retrieves all backup codes for a user
	GetBackupCodes(ctx context.Context, userID uuid.UUID) ([]*BackupCode, error)

	// MarkBackupCodeUsed marks a backup code as used
	MarkBackupCodeUsed(ctx context.Context, userID uuid.UUID, code string) error

	// DeleteBackupCodes deletes all backup codes for a user
	DeleteBackupCodes(ctx context.Context, userID uuid.UUID) error

	// LogAudit logs an MFA audit event
	LogAudit(ctx context.Context, log *AuditLog) error

	// GetAuditLogs retrieves audit logs for a user
	GetAuditLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*AuditLog, error)
}

// Service defines the interface for MFA operations
type Service interface {
	// SetupMFA initiates MFA setup for a user
	SetupMFA(ctx context.Context, req *SetupRequest) (*SetupResponse, error)

	// VerifySetup verifies and completes MFA setup
	VerifySetup(ctx context.Context, req *VerifySetupRequest) error

	// VerifyCode verifies an MFA code
	VerifyCode(ctx context.Context, req *VerifyRequest) (*VerifyResponse, error)

	// DisableMFA disables MFA for a user
	DisableMFA(ctx context.Context, req *DisableRequest) error

	// GenerateBackupCodes generates new backup codes
	GenerateBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error)

	// RecoverAccess recovers access using a recovery code
	RecoverAccess(ctx context.Context, req *RecoveryRequest) error

	// GetSettings retrieves MFA settings for a user
	GetSettings(ctx context.Context, userID uuid.UUID) (*Settings, error)

	// IsEnabled checks if MFA is enabled for a user
	IsEnabled(ctx context.Context, userID uuid.UUID) (bool, error)

	// GetAvailableMethods gets available MFA methods for a user
	GetAvailableMethods(ctx context.Context, userID uuid.UUID) ([]Method, error)

	// CreateChallenge creates an MFA challenge
	CreateChallenge(ctx context.Context, userID uuid.UUID, method Method) (*Challenge, error)

	// ValidateChallenge validates an MFA challenge
	ValidateChallenge(ctx context.Context, challengeID uuid.UUID, code string) (bool, error)
}

// TOTPProvider defines the interface for TOTP operations
type TOTPProvider interface {
	// GenerateSecret generates a new TOTP secret
	GenerateSecret() (string, error)

	// GenerateQRCode generates a QR code for TOTP setup
	GenerateQRCode(secret, email string) (string, error)

	// ValidateCode validates a TOTP code
	ValidateCode(secret, code string) (bool, error)

	// GenerateCode generates a TOTP code (for testing)
	GenerateCode(secret string) (string, error)
}

// SMSProvider defines the interface for SMS operations
type SMSProvider interface {
	// SendCode sends an SMS code to a phone number
	SendCode(ctx context.Context, phoneNumber, code string) error

	// VerifyPhoneNumber verifies a phone number is valid
	VerifyPhoneNumber(phoneNumber string) error
}

// EmailProvider defines the interface for email operations
type EmailProvider interface {
	// SendCode sends an email code
	SendCode(ctx context.Context, email, code string) error

	// SendBackupCodes sends backup codes via email
	SendBackupCodes(ctx context.Context, email string, codes []string) error
}

// CodeGenerator defines the interface for generating codes
type CodeGenerator interface {
	// GenerateNumericCode generates a numeric code
	GenerateNumericCode(length int) string

	// GenerateAlphanumericCode generates an alphanumeric code
	GenerateAlphanumericCode(length int) string

	// GenerateBackupCodes generates backup codes
	GenerateBackupCodes(count, length int) []string
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// CheckLimit checks if the rate limit is exceeded
	CheckLimit(ctx context.Context, key string) (bool, int, error)

	// RecordAttempt records an attempt
	RecordAttempt(ctx context.Context, key string) error

	// Reset resets the rate limit
	Reset(ctx context.Context, key string) error

	// GetRemaining gets remaining attempts
	GetRemaining(ctx context.Context, key string) (int, time.Duration, error)
}

// Cache defines the interface for caching MFA data
type Cache interface {
	// StoreSetup stores temporary setup data
	StoreSetup(ctx context.Context, setupID string, data interface{}, expiry time.Duration) error

	// GetSetup retrieves temporary setup data
	GetSetup(ctx context.Context, setupID string) (interface{}, error)

	// DeleteSetup deletes temporary setup data
	DeleteSetup(ctx context.Context, setupID string) error

	// StoreChallenge stores a challenge
	StoreChallenge(ctx context.Context, challenge *Challenge) error

	// GetChallenge retrieves a challenge
	GetChallenge(ctx context.Context, id uuid.UUID) (*Challenge, error)

	// DeleteChallenge deletes a challenge
	DeleteChallenge(ctx context.Context, id uuid.UUID) error
}
