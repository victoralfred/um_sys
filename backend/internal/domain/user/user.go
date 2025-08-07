package user

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the user account status
type Status string

const (
	StatusActive    Status = "active"
	StatusInactive  Status = "inactive"
	StatusSuspended Status = "suspended"
	StatusLocked    Status = "locked"
	StatusDeleted   Status = "deleted"
)

// User represents a user in the system
type User struct {
	ID                  uuid.UUID  `json:"id"`
	Email               string     `json:"email"`
	Username            string     `json:"username"`
	PasswordHash        string     `json:"-"`
	FirstName           string     `json:"first_name"`
	LastName            string     `json:"last_name"`
	PhoneNumber         string     `json:"phone_number,omitempty"`
	Status              Status     `json:"status"`
	EmailVerified       bool       `json:"email_verified"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	PhoneVerified       bool       `json:"phone_verified"`
	MFAEnabled          bool       `json:"mfa_enabled"`
	MFASecret           string     `json:"-"`
	MFABackupCodes      []string   `json:"-"`
	ProfilePictureURL   string     `json:"profile_picture_url,omitempty"`
	Bio                 string     `json:"bio,omitempty"`
	Locale              string     `json:"locale,omitempty"`
	Timezone            string     `json:"timezone,omitempty"`
	PasswordResetToken  string     `json:"-"`
	PasswordResetExpiry *time.Time `json:"-"`
	PasswordChangedAt   *time.Time `json:"password_changed_at,omitempty"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// ListFilter contains filters for listing users
type ListFilter struct {
	Search        string
	EmailVerified *bool
	MFAEnabled    *bool
	Limit         int
	Offset        int
	SortBy        string
	SortOrder     string
}

// NewUser creates a new user with basic validation
func NewUser(email, username, passwordHash string) (*User, error) {
	if email == "" {
		return nil, ErrEmailRequired
	}
	if username == "" {
		return nil, ErrUsernameRequired
	}
	if passwordHash == "" {
		return nil, ErrPasswordRequired
	}

	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// IsLocked checks if the user account is locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// IsDeleted checks if the user is soft deleted
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// CanResetPassword checks if password reset token is valid
func (u *User) CanResetPassword() bool {
	if u.PasswordResetToken == "" || u.PasswordResetExpiry == nil {
		return false
	}
	return time.Now().Before(*u.PasswordResetExpiry)
}
