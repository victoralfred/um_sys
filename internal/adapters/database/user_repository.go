package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// UserRepository implements user.Repository interface
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) user.Repository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	if u == nil {
		return user.ErrUserNil
	}

	// Generate ID and timestamps if not set
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	query := `
		INSERT INTO users (
			id, email, username, password_hash, 
			first_name, last_name, phone_number, status,
			email_verified, phone_verified, mfa_enabled, mfa_secret,
			profile_picture_url, bio, locale, timezone,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)`

	_, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.Username, u.PasswordHash,
		u.FirstName, u.LastName, u.PhoneNumber, u.Status,
		u.EmailVerified, u.PhoneVerified, u.MFAEnabled, u.MFASecret,
		u.ProfilePictureURL, u.Bio, u.Locale, u.Timezone,
		u.CreatedAt, u.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "users_email_key") {
			return user.ErrEmailAlreadyExists
		}
		if strings.Contains(err.Error(), "users_username_key") {
			return user.ErrUsernameAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if id == uuid.Nil {
		return nil, user.ErrInvalidUserID
	}

	u := &user.User{}
	query := `
		SELECT 
			id, email, username, password_hash,
			first_name, last_name, phone_number, status,
			email_verified, email_verified_at, phone_verified,
			mfa_enabled, mfa_secret, profile_picture_url,
			bio, locale, timezone, last_login_at,
			password_changed_at, failed_login_attempts, locked_until,
			created_at, updated_at, deleted_at
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash,
		&u.FirstName, &u.LastName, &u.PhoneNumber, &u.Status,
		&u.EmailVerified, &u.EmailVerifiedAt, &u.PhoneVerified,
		&u.MFAEnabled, &u.MFASecret, &u.ProfilePictureURL,
		&u.Bio, &u.Locale, &u.Timezone, &u.LastLoginAt,
		&u.PasswordChangedAt, &u.FailedLoginAttempts, &u.LockedUntil,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return u, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if email == "" {
		return nil, user.ErrEmailRequired
	}

	u := &user.User{}
	query := `
		SELECT 
			id, email, username, password_hash,
			first_name, last_name, phone_number, status,
			email_verified, email_verified_at, phone_verified,
			mfa_enabled, mfa_secret, profile_picture_url,
			bio, locale, timezone, last_login_at,
			password_changed_at, failed_login_attempts, locked_until,
			created_at, updated_at, deleted_at
		FROM users 
		WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`

	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash,
		&u.FirstName, &u.LastName, &u.PhoneNumber, &u.Status,
		&u.EmailVerified, &u.EmailVerifiedAt, &u.PhoneVerified,
		&u.MFAEnabled, &u.MFASecret, &u.ProfilePictureURL,
		&u.Bio, &u.Locale, &u.Timezone, &u.LastLoginAt,
		&u.PasswordChangedAt, &u.FailedLoginAttempts, &u.LockedUntil,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return u, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	if username == "" {
		return nil, user.ErrUsernameRequired
	}

	u := &user.User{}
	query := `
		SELECT 
			id, email, username, password_hash,
			first_name, last_name, phone_number, status,
			email_verified, email_verified_at, phone_verified,
			mfa_enabled, mfa_secret, profile_picture_url,
			bio, locale, timezone, last_login_at,
			password_changed_at, failed_login_attempts, locked_until,
			created_at, updated_at, deleted_at
		FROM users 
		WHERE LOWER(username) = LOWER($1) AND deleted_at IS NULL`

	err := r.db.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash,
		&u.FirstName, &u.LastName, &u.PhoneNumber, &u.Status,
		&u.EmailVerified, &u.EmailVerifiedAt, &u.PhoneVerified,
		&u.MFAEnabled, &u.MFASecret, &u.ProfilePictureURL,
		&u.Bio, &u.Locale, &u.Timezone, &u.LastLoginAt,
		&u.PasswordChangedAt, &u.FailedLoginAttempts, &u.LockedUntil,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return u, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	if u == nil {
		return user.ErrUserNil
	}

	u.UpdatedAt = time.Now()

	query := `
		UPDATE users SET
			email = $2, username = $3, password_hash = $4,
			first_name = $5, last_name = $6, phone_number = $7, status = $8,
			email_verified = $9, email_verified_at = $10, phone_verified = $11,
			mfa_enabled = $12, mfa_secret = $13, profile_picture_url = $14,
			bio = $15, locale = $16, timezone = $17,
			last_login_at = $18, password_changed_at = $19,
			failed_login_attempts = $20, locked_until = $21,
			updated_at = $22
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.Username, u.PasswordHash,
		u.FirstName, u.LastName, u.PhoneNumber, u.Status,
		u.EmailVerified, u.EmailVerifiedAt, u.PhoneVerified,
		u.MFAEnabled, u.MFASecret, u.ProfilePictureURL,
		u.Bio, u.Locale, u.Timezone,
		u.LastLoginAt, u.PasswordChangedAt,
		u.FailedLoginAttempts, u.LockedUntil,
		u.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "users_email_key") {
			return user.ErrEmailAlreadyExists
		}
		if strings.Contains(err.Error(), "users_username_key") {
			return user.ErrUsernameAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return user.ErrInvalidUserID
	}

	now := time.Now()
	query := `
		UPDATE users 
		SET deleted_at = $2, updated_at = $2 
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	// Build query with filters
	var conditions []string
	var args []interface{}
	argCount := 0

	// Always exclude soft deleted users
	conditions = append(conditions, "deleted_at IS NULL")

	// Add search condition
	if filter.Search != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(email) LIKE LOWER($%d) OR LOWER(username) LIKE LOWER($%d) OR LOWER(first_name) LIKE LOWER($%d) OR LOWER(last_name) LIKE LOWER($%d))",
			argCount, argCount, argCount, argCount,
		))
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm)
	}

	// Add email verified filter
	if filter.EmailVerified != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("email_verified = $%d", argCount))
		args = append(args, *filter.EmailVerified)
	}

	// Add MFA enabled filter
	if filter.MFAEnabled != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("mfa_enabled = $%d", argCount))
		args = append(args, *filter.MFAEnabled)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Prepare sort clause
	sortClause := "ORDER BY created_at DESC" // default sort
	if filter.SortBy != "" {
		// Validate sort field to prevent SQL injection
		validSortFields := map[string]bool{
			"email":      true,
			"username":   true,
			"created_at": true,
			"updated_at": true,
		}
		if validSortFields[filter.SortBy] {
			order := "ASC"
			if filter.SortOrder == "desc" {
				order = "DESC"
			}
			sortClause = fmt.Sprintf("ORDER BY %s %s", filter.SortBy, order)
		} else {
			return nil, 0, user.ErrInvalidSortField
		}
	}

	// Add pagination
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		return nil, 0, user.ErrInvalidOffset
	}

	// Build final query
	query := fmt.Sprintf(`
		SELECT 
			id, email, username, password_hash,
			first_name, last_name, phone_number, status,
			email_verified, email_verified_at, phone_verified,
			mfa_enabled, mfa_secret, profile_picture_url,
			bio, locale, timezone, last_login_at,
			password_changed_at, failed_login_attempts, locked_until,
			created_at, updated_at, deleted_at
		FROM users 
		%s 
		%s 
		LIMIT $%d OFFSET $%d`,
		whereClause, sortClause, argCount+1, argCount+2)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u := &user.User{}
		err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.PasswordHash,
			&u.FirstName, &u.LastName, &u.PhoneNumber, &u.Status,
			&u.EmailVerified, &u.EmailVerifiedAt, &u.PhoneVerified,
			&u.MFAEnabled, &u.MFASecret, &u.ProfilePictureURL,
			&u.Bio, &u.Locale, &u.Timezone, &u.LastLoginAt,
			&u.PasswordChangedAt, &u.FailedLoginAttempts, &u.LockedUntil,
			&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, total, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	if id == uuid.Nil {
		return user.ErrInvalidUserID
	}

	query := `
		UPDATE users 
		SET last_login_at = $2, updated_at = $3, failed_login_attempts = 0, locked_until = NULL
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
	if id == uuid.Nil {
		return user.ErrInvalidUserID
	}

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

// UpdateMFA updates MFA settings for a user
func (r *UserRepository) UpdateMFA(ctx context.Context, id uuid.UUID, enabled bool, secret string, backupCodes []string) error {
	if id == uuid.Nil {
		return user.ErrInvalidUserID
	}

	if enabled && secret == "" {
		return user.ErrMFASecretRequired
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Update user MFA settings
	now := time.Now()
	query := `
		UPDATE users 
		SET mfa_enabled = $2, mfa_secret = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := tx.Exec(ctx, query, id, enabled, secret, now)
	if err != nil {
		return fmt.Errorf("failed to update MFA settings: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// Handle backup codes (this would typically be in a separate table)
	// For now, we'll skip this as it's not in our current schema

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExistsByEmail checks if a user exists with the given email
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, user.ErrEmailRequired
	}

	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL
		)`

	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// ExistsByUsername checks if a user exists with the given username
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if username == "" {
		return false, user.ErrUsernameRequired
	}

	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE LOWER(username) = LOWER($1) AND deleted_at IS NULL
		)`

	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return exists, nil
}
