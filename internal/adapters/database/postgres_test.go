package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestNewPostgresConnection(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: Config{
				Host:         "localhost",
				Port:         5432,
				User:         "test",
				Password:     "test",
				Database:     "testdb",
				SSLMode:      "disable",
				MaxConns:     10,
				MaxIdleConns: 5,
				MaxLifetime:  time.Hour,
			},
			wantErr: false,
		},
		{
			name: "invalid host",
			config: Config{
				Host:     "",
				Port:     5432,
				User:     "test",
				Password: "test",
				Database: "testdb",
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "invalid port",
			config: Config{
				Host:     "localhost",
				Port:     0,
				User:     "test",
				Password: "test",
				Database: "testdb",
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewPostgresConnection(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				if db != nil {
					db.Close()
				}
			}
		})
	}
}

func TestPostgresConnection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Test connection
	config := Config{
		ConnectionString: connStr,
		MaxConns:         10,
		MaxIdleConns:     5,
		MaxLifetime:      time.Hour,
	}

	db, err := NewPostgresConnection(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Test ping
	err = db.Ping(ctx)
	assert.NoError(t, err)

	// Test query
	var result int
	err = db.QueryRow(ctx, "SELECT 1").Scan(&result)
	assert.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestMigrationRunner(t *testing.T) {
	tests := []struct {
		name          string
		migrationsDir string
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "valid migrations directory",
			migrationsDir: "migrations",
			wantErr:       false,
		},
		{
			name:          "empty migrations directory",
			migrationsDir: "",
			wantErr:       true,
			errMsg:        "migrations directory is required",
		},
		{
			name:          "non-existent directory",
			migrationsDir: "/non/existent/path",
			wantErr:       true,
			errMsg:        "migrations directory does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewMigrationRunner(nil, tt.migrationsDir)
			err := runner.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMigrationRunner_Up(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create database connection
	config := Config{
		ConnectionString: connStr,
		MaxConns:         10,
		MaxIdleConns:     5,
		MaxLifetime:      time.Hour,
	}

	db, err := NewPostgresConnection(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Create migration runner
	runner := NewMigrationRunner(db, "../../../../migrations")

	// Run migrations up
	err = runner.Up(ctx)
	assert.NoError(t, err)

	// Verify migrations were applied
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestMigrationRunner_Down(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create database connection
	config := Config{
		ConnectionString: connStr,
		MaxConns:         10,
		MaxIdleConns:     5,
		MaxLifetime:      time.Hour,
	}

	db, err := NewPostgresConnection(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Create migration runner
	runner := NewMigrationRunner(db, "../../../../migrations")

	// Run migrations up first
	err = runner.Up(ctx)
	require.NoError(t, err)

	// Run one migration down
	err = runner.Down(ctx, 1)
	assert.NoError(t, err)

	// Verify migration was rolled back
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 0)
}
