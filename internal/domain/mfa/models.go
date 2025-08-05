package mfa

import (
	"time"

	"github.com/google/uuid"
)

// Method represents the type of MFA method
type Method string

const (
	MethodTOTP       Method = "totp"        // Time-based One-Time Password (Google Authenticator)
	MethodSMS        Method = "sms"         // SMS verification
	MethodEmail      Method = "email"       // Email verification
	MethodBackupCode Method = "backup_code" // Backup codes
)

// Status represents the MFA status
type Status string

const (
	StatusPending  Status = "pending"  // MFA setup initiated but not verified
	StatusActive   Status = "active"   // MFA is active and verified
	StatusDisabled Status = "disabled" // MFA is disabled
)

// Settings represents MFA settings for a user
type Settings struct {
	UserID        uuid.UUID  `json:"user_id"`
	Enabled       bool       `json:"enabled"`
	Methods       []Method   `json:"methods"`
	PrimaryMethod Method     `json:"primary_method"`
	TOTPSecret    string     `json:"-"` // Hidden in JSON
	BackupCodes   []string   `json:"-"` // Hidden in JSON
	RecoveryCodes []string   `json:"-"` // Hidden in JSON
	PhoneNumber   string     `json:"phone_number,omitempty"`
	Email         string     `json:"email,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SetupRequest represents a request to set up MFA
type SetupRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	Method      Method    `json:"method" binding:"required,oneof=totp sms email"`
	PhoneNumber string    `json:"phone_number,omitempty" binding:"required_if=Method sms"`
	Email       string    `json:"email,omitempty" binding:"required_if=Method email"`
}

// SetupResponse represents the response for MFA setup
type SetupResponse struct {
	Method      Method    `json:"method"`
	Secret      string    `json:"secret,omitempty"`       // For TOTP
	QRCode      string    `json:"qr_code,omitempty"`      // For TOTP (base64 encoded)
	BackupCodes []string  `json:"backup_codes,omitempty"` // Backup codes
	SetupID     string    `json:"setup_id"`               // Temporary setup ID
	ExpiresAt   time.Time `json:"expires_at"`
}

// VerifySetupRequest represents a request to verify MFA setup
type VerifySetupRequest struct {
	UserID  uuid.UUID `json:"user_id"`
	SetupID string    `json:"setup_id" binding:"required"`
	Code    string    `json:"code" binding:"required,len=6"`
}

// VerifyRequest represents a request to verify MFA code
type VerifyRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Method Method    `json:"method" binding:"required"`
	Code   string    `json:"code" binding:"required"`
}

// VerifyResponse represents the response for MFA verification
type VerifyResponse struct {
	Valid        bool      `json:"valid"`
	Method       Method    `json:"method"`
	VerifiedAt   time.Time `json:"verified_at"`
	RateLimited  bool      `json:"rate_limited,omitempty"`
	AttemptsLeft int       `json:"attempts_left,omitempty"`
}

// DisableRequest represents a request to disable MFA
type DisableRequest struct {
	UserID   uuid.UUID `json:"user_id"`
	Password string    `json:"password" binding:"required"` // Require password confirmation
	Code     string    `json:"code,omitempty"`              // Current MFA code if enabled
}

// BackupCode represents a backup code
type BackupCode struct {
	Code      string     `json:"code"`
	Used      bool       `json:"used"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// RecoveryRequest represents a request to recover MFA access
type RecoveryRequest struct {
	UserID       uuid.UUID `json:"user_id"`
	RecoveryCode string    `json:"recovery_code" binding:"required"`
}

// Challenge represents an MFA challenge
type Challenge struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Method    Method    `json:"method"`
	Code      string    `json:"-"` // Hidden
	ExpiresAt time.Time `json:"expires_at"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLog represents an MFA audit log entry
type AuditLog struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Action    string    `json:"action"` // setup, verify, disable, recover
	Method    Method    `json:"method"`
	Success   bool      `json:"success"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Details   string    `json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Constants for configuration
const (
	TOTPIssuer       = "UManager"
	TOTPPeriod       = 30 // seconds
	TOTPDigits       = 6
	BackupCodeCount  = 10
	BackupCodeLength = 8
	MaxAttempts      = 3
	RateLimitWindow  = 15 * time.Minute
	CodeExpiry       = 5 * time.Minute
	SetupExpiry      = 10 * time.Minute
)
