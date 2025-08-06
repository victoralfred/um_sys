package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// UserRepository implements user.Repository with PostgreSQL
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (
			id, email, username, password_hash, 
			first_name, last_name, phone_number,
			is_active, is_verified, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err := r.db.Exec(ctx, query,
		u.ID,
		u.Email,
		u.Username,
		u.PasswordHash,
		u.FirstName,
		u.LastName,
		u.PhoneNumber,
		u.Status == user.StatusActive,
		u.EmailVerified,
		u.CreatedAt,
		u.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	query := `
		SELECT 
			id, email, username, password_hash,
			first_name, last_name, phone_number,
			is_active, is_verified, verified_at,
			last_login_at, failed_login_attempts, locked_until,
			mfa_enabled, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	var u user.User
	var isActive, isVerified, mfaEnabled bool
	var verifiedAt, lastLoginAt, lockedUntil sql.NullTime

	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.PasswordHash,
		&u.FirstName,
		&u.LastName,
		&u.PhoneNumber,
		&isActive,
		&isVerified,
		&verifiedAt,
		&lastLoginAt,
		&u.FailedLoginAttempts,
		&lockedUntil,
		&mfaEnabled,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	// Map database fields to domain model
	if isActive {
		u.Status = user.StatusActive
	} else {
		u.Status = user.StatusInactive
	}

	u.EmailVerified = isVerified
	if verifiedAt.Valid {
		u.EmailVerifiedAt = &verifiedAt.Time
	}
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	u.MFAEnabled = mfaEnabled

	return &u, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT 
			id, email, username, password_hash,
			first_name, last_name, phone_number,
			is_active, is_verified, verified_at,
			last_login_at, failed_login_attempts, locked_until,
			mfa_enabled, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	var u user.User
	var isActive, isVerified, mfaEnabled bool
	var verifiedAt, lastLoginAt, lockedUntil sql.NullTime

	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.PasswordHash,
		&u.FirstName,
		&u.LastName,
		&u.PhoneNumber,
		&isActive,
		&isVerified,
		&verifiedAt,
		&lastLoginAt,
		&u.FailedLoginAttempts,
		&lockedUntil,
		&mfaEnabled,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Map database fields to domain model
	if isActive {
		u.Status = user.StatusActive
	} else {
		u.Status = user.StatusInactive
	}

	u.EmailVerified = isVerified
	if verifiedAt.Valid {
		u.EmailVerifiedAt = &verifiedAt.Time
	}
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	u.MFAEnabled = mfaEnabled

	return &u, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	query := `
		SELECT 
			id, email, username, password_hash,
			first_name, last_name, phone_number,
			is_active, is_verified, verified_at,
			last_login_at, failed_login_attempts, locked_until,
			mfa_enabled, created_at, updated_at
		FROM users
		WHERE username = $1 AND deleted_at IS NULL`

	var u user.User
	var isActive, isVerified, mfaEnabled bool
	var verifiedAt, lastLoginAt, lockedUntil sql.NullTime

	err := r.db.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.PasswordHash,
		&u.FirstName,
		&u.LastName,
		&u.PhoneNumber,
		&isActive,
		&isVerified,
		&verifiedAt,
		&lastLoginAt,
		&u.FailedLoginAttempts,
		&lockedUntil,
		&mfaEnabled,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	// Map database fields to domain model
	if isActive {
		u.Status = user.StatusActive
	} else {
		u.Status = user.StatusInactive
	}

	u.EmailVerified = isVerified
	if verifiedAt.Valid {
		u.EmailVerifiedAt = &verifiedAt.Time
	}
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	u.MFAEnabled = mfaEnabled

	return &u, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users SET
			email = $2,
			username = $3,
			password_hash = $4,
			first_name = $5,
			last_name = $6,
			phone_number = $7,
			is_active = $8,
			is_verified = $9,
			verified_at = $10,
			last_login_at = $11,
			failed_login_attempts = $12,
			locked_until = $13,
			mfa_enabled = $14,
			mfa_secret = $15,
			updated_at = $16
		WHERE id = $1 AND deleted_at IS NULL`

	var verifiedAt, lastLoginAt, lockedUntil *time.Time
	if u.EmailVerifiedAt != nil {
		verifiedAt = u.EmailVerifiedAt
	}
	if u.LastLoginAt != nil {
		lastLoginAt = u.LastLoginAt
	}
	if u.LockedUntil != nil {
		lockedUntil = u.LockedUntil
	}

	result, err := r.db.Exec(ctx, query,
		u.ID,
		u.Email,
		u.Username,
		u.PasswordHash,
		u.FirstName,
		u.LastName,
		u.PhoneNumber,
		u.Status == user.StatusActive,
		u.EmailVerified,
		verifiedAt,
		lastLoginAt,
		u.FailedLoginAttempts,
		lockedUntil,
		u.MFAEnabled,
		u.MFASecret,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users 
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now()
	result, err := r.db.Exec(ctx, query, id, now)

	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// ExistsByEmail checks if a user exists with the given email
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE email = $1 AND deleted_at IS NULL
		)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// ExistsByUsername checks if a user exists with the given username
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE username = $1 AND deleted_at IS NULL
		)`

	var exists bool
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return exists, nil
}

// UpdateLastLogin updates the user's last login time and IP
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	query := `
		UPDATE users 
		SET 
			last_login_at = $2,
			failed_login_attempts = 0,
			locked_until = NULL,
			updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query, id, loginTime, time.Now())

	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// IncrementFailedLoginAttempts increments failed login attempts
func (r *UserRepository) IncrementFailedLoginAttempts(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users 
		SET 
			failed_login_attempts = failed_login_attempts + 1,
			locked_until = CASE 
				WHEN failed_login_attempts >= 4 THEN NOW() + INTERVAL '15 minutes'
				ELSE locked_until
			END,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query, id)

	if err != nil {
		return fmt.Errorf("failed to increment failed login attempts: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}
