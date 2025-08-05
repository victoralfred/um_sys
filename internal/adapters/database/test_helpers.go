package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDatabase holds test database resources
type TestDatabase struct {
	Container testcontainers.Container
	Pool      *pgxpool.Pool
	ConnStr   string
}

// SetupTestDatabase creates a PostgreSQL container for testing
func SetupTestDatabase(t *testing.T) *TestDatabase {
	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("Failed to parse connection string: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	// Run migrations
	if err := RunTestMigrations(ctx, pool); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &TestDatabase{
		Container: pgContainer,
		Pool:      pool,
		ConnStr:   connStr,
	}
}

// Cleanup cleans up test database resources
func (td *TestDatabase) Cleanup() {
	ctx := context.Background()

	if td.Pool != nil {
		td.Pool.Close()
	}

	if td.Container != nil {
		_ = td.Container.Terminate(ctx)
	}
}

// RunTestMigrations runs database migrations for tests
func RunTestMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Run migration to create users table
	migration := `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			phone_number VARCHAR(20),
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			email_verified BOOLEAN NOT NULL DEFAULT false,
			email_verified_at TIMESTAMP,
			phone_verified BOOLEAN NOT NULL DEFAULT false,
			mfa_enabled BOOLEAN NOT NULL DEFAULT false,
			mfa_secret VARCHAR(255),
			profile_picture_url VARCHAR(500),
			bio TEXT,
			locale VARCHAR(10) DEFAULT 'en',
			timezone VARCHAR(50) DEFAULT 'UTC',
			last_login_at TIMESTAMP,
			password_changed_at TIMESTAMP,
			failed_login_attempts INT DEFAULT 0,
			locked_until TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP
		);

		CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
		CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
		CREATE INDEX idx_users_deleted_at ON users(deleted_at);
	`

	_, err := pool.Exec(ctx, migration)
	return err
}
