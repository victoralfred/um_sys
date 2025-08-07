package auth

import "errors"

var (
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrAccountInactive is returned when trying to login with inactive account
	ErrAccountInactive = errors.New("account is inactive")

	// ErrAccountLocked is returned when account is locked
	ErrAccountLocked = errors.New("account is locked")

	// ErrTokenExpired is returned when token has expired
	ErrTokenExpired = errors.New("token has expired")

	// ErrInvalidToken is returned when token is invalid
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenRevoked is returned when token has been revoked
	ErrTokenRevoked = errors.New("token has been revoked")

	// ErrEmailAlreadyExists is returned when email already exists
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrUsernameAlreadyExists is returned when username already exists
	ErrUsernameAlreadyExists = errors.New("username already exists")

	// ErrPasswordTooWeak is returned when password doesn't meet requirements
	ErrPasswordTooWeak = errors.New("password does not meet security requirements")

	// ErrMFARequired is returned when MFA is required but not provided
	ErrMFARequired = errors.New("multi-factor authentication required")

	// ErrInvalidMFACode is returned when MFA code is invalid
	ErrInvalidMFACode = errors.New("invalid MFA code")
)
