package user

import "errors"

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailAlreadyExists is returned when email already exists
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrUsernameAlreadyExists is returned when username already exists
	ErrUsernameAlreadyExists = errors.New("username already exists")

	// ErrInvalidUserID is returned when user ID is invalid
	ErrInvalidUserID = errors.New("invalid user ID")

	// ErrUserNil is returned when user is nil
	ErrUserNil = errors.New("user cannot be nil")

	// ErrEmailRequired is returned when email is empty
	ErrEmailRequired = errors.New("email is required")

	// ErrUsernameRequired is returned when username is empty
	ErrUsernameRequired = errors.New("username is required")

	// ErrPasswordRequired is returned when password is empty
	ErrPasswordRequired = errors.New("password is required")

	// ErrInvalidSortField is returned when sort field is invalid
	ErrInvalidSortField = errors.New("invalid sort field")

	// ErrInvalidOffset is returned when offset is negative
	ErrInvalidOffset = errors.New("invalid offset")

	// ErrMFASecretRequired is returned when MFA secret is required but not provided
	ErrMFASecretRequired = errors.New("MFA secret is required when enabling")

	// ErrNotFound is an alias for ErrUserNotFound
	ErrNotFound = ErrUserNotFound

	// ErrDuplicateEmail is an alias for ErrEmailAlreadyExists
	ErrDuplicateEmail = ErrEmailAlreadyExists

	// ErrDuplicateUsername is an alias for ErrUsernameAlreadyExists
	ErrDuplicateUsername = ErrUsernameAlreadyExists

	// ErrInvalidEmail is returned when email format is invalid
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidID is an alias for ErrInvalidUserID
	ErrInvalidID = ErrInvalidUserID
)
