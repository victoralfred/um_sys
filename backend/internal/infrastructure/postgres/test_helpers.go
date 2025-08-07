package postgres

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

// setupTestDB creates a PostgreSQL container for testing
func setupTestDB(t *testing.T) *TestDatabase {
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
				WithStartupTimeout(10*time.Second),
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
	if err := runTestMigrations(ctx, pool); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Setup cleanup
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		if pgContainer != nil {
			_ = pgContainer.Terminate(ctx)
		}
	})

	return &TestDatabase{
		Container: pgContainer,
		Pool:      pool,
		ConnStr:   connStr,
	}
}

// runTestMigrations runs database migrations for tests
func runTestMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Create users table (needed for foreign keys)
	usersMigration := `
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

		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
	`

	// Create audit_logs table
	auditMigration := `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			event_type VARCHAR(100) NOT NULL,
			severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
			user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
			entity_type VARCHAR(100) NOT NULL,
			entity_id VARCHAR(255) NOT NULL,
			action VARCHAR(100) NOT NULL,
			description TEXT,
			ip_address INET,
			user_agent TEXT,
			metadata JSONB,
			changes JSONB,
			request_id VARCHAR(255),
			session_id VARCHAR(255),
			trace_id VARCHAR(255),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Create indexes for efficient querying
		CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs (timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs (user_id) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs (actor_id) WHERE actor_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs (event_type);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_severity ON audit_logs (severity);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs (ip_address) WHERE ip_address IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs (request_id) WHERE request_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_logs_session_id ON audit_logs (session_id) WHERE session_id IS NOT NULL;

		-- Create composite indexes for common queries
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user_timestamp ON audit_logs (user_id, timestamp DESC) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_timestamp ON audit_logs (entity_type, entity_id, timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_event_severity ON audit_logs (event_type, severity);

		-- Create GIN index for metadata searches
		CREATE INDEX IF NOT EXISTS idx_audit_logs_metadata ON audit_logs USING GIN (metadata) WHERE metadata IS NOT NULL;
	`

	_, err := pool.Exec(ctx, usersMigration)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, auditMigration)
	return err
}
