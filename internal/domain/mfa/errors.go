package mfa

import "errors"

var (
	// ErrMFANotEnabled is returned when MFA is not enabled for the user
	ErrMFANotEnabled = errors.New("MFA is not enabled")

	// ErrMFAAlreadyEnabled is returned when trying to enable MFA when it's already enabled
	ErrMFAAlreadyEnabled = errors.New("MFA is already enabled")

	// ErrInvalidMethod is returned when an invalid MFA method is specified
	ErrInvalidMethod = errors.New("invalid MFA method")

	// ErrMethodNotConfigured is returned when a method is not configured for the user
	ErrMethodNotConfigured = errors.New("MFA method not configured")

	// ErrInvalidCode is returned when the MFA code is invalid
	ErrInvalidCode = errors.New("invalid MFA code")

	// ErrCodeExpired is returned when the MFA code has expired
	ErrCodeExpired = errors.New("MFA code has expired")

	// ErrTooManyAttempts is returned when too many failed attempts have been made
	ErrTooManyAttempts = errors.New("too many failed attempts")

	// ErrRateLimited is returned when rate limit is exceeded
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrInvalidSetupID is returned when the setup ID is invalid
	ErrInvalidSetupID = errors.New("invalid setup ID")

	// ErrSetupExpired is returned when the setup has expired
	ErrSetupExpired = errors.New("MFA setup has expired")

	// ErrInvalidBackupCode is returned when the backup code is invalid
	ErrInvalidBackupCode = errors.New("invalid backup code")

	// ErrBackupCodeAlreadyUsed is returned when the backup code has already been used
	ErrBackupCodeAlreadyUsed = errors.New("backup code has already been used")

	// ErrNoBackupCodes is returned when there are no backup codes available
	ErrNoBackupCodes = errors.New("no backup codes available")

	// ErrInvalidRecoveryCode is returned when the recovery code is invalid
	ErrInvalidRecoveryCode = errors.New("invalid recovery code")

	// ErrInvalidPhoneNumber is returned when the phone number is invalid
	ErrInvalidPhoneNumber = errors.New("invalid phone number")

	// ErrInvalidEmail is returned when the email is invalid
	ErrInvalidEmail = errors.New("invalid email address")

	// ErrSMSFailed is returned when SMS sending fails
	ErrSMSFailed = errors.New("failed to send SMS")

	// ErrEmailFailed is returned when email sending fails
	ErrEmailFailed = errors.New("failed to send email")

	// ErrSecretGenerationFailed is returned when secret generation fails
	ErrSecretGenerationFailed = errors.New("failed to generate secret")

	// ErrQRCodeGenerationFailed is returned when QR code generation fails
	ErrQRCodeGenerationFailed = errors.New("failed to generate QR code")

	// ErrChallengeNotFound is returned when a challenge is not found
	ErrChallengeNotFound = errors.New("challenge not found")

	// ErrChallengeExpired is returned when a challenge has expired
	ErrChallengeExpired = errors.New("challenge has expired")

	// ErrSettingsNotFound is returned when MFA settings are not found
	ErrSettingsNotFound = errors.New("MFA settings not found")
)
