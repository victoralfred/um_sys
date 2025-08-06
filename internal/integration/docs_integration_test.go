package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"github.com/victoralfred/um_sys/internal/config"
	"github.com/victoralfred/um_sys/internal/handlers"
	"github.com/victoralfred/um_sys/internal/middleware"
	"github.com/victoralfred/um_sys/internal/repositories"
	"github.com/victoralfred/um_sys/internal/server"
	"github.com/victoralfred/um_sys/internal/services"
	"github.com/victoralfred/um_sys/pkg/security"
)

func TestAPIDocumentationEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test server
	router := setupTestServerWithDocs(t)

	t.Run("swagger.json endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/docs/swagger.json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var swaggerDoc map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &swaggerDoc)
		require.NoError(t, err)

		// Validate OpenAPI structure
		assert.Equal(t, "3.0.0", swaggerDoc["openapi"])
		assert.Contains(t, swaggerDoc, "info")
		assert.Contains(t, swaggerDoc, "paths")
		assert.Contains(t, swaggerDoc, "components")

		// Check that our auth endpoints are documented
		paths, ok := swaggerDoc["paths"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, paths, "/v1/auth/register")
		assert.Contains(t, paths, "/v1/auth/login")
	})

	t.Run("swagger UI endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/docs", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))

		bodyStr := w.Body.String()
		assert.Contains(t, bodyStr, "<!DOCTYPE html>")
		assert.Contains(t, bodyStr, "UManager API Documentation")
		assert.Contains(t, bodyStr, "swagger-ui-bundle")
	})

	t.Run("redoc UI endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/docs/redoc", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))

		bodyStr := w.Body.String()
		assert.Contains(t, bodyStr, "<!DOCTYPE html>")
		assert.Contains(t, bodyStr, "UManager API Documentation")
		assert.Contains(t, bodyStr, "redoc.standalone.js")
	})

	t.Run("docs index endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/docs/", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))

		bodyStr := w.Body.String()
		assert.Contains(t, bodyStr, "UManager API Documentation")
		assert.Contains(t, bodyStr, "Swagger UI")
		assert.Contains(t, bodyStr, "ReDoc")
		assert.Contains(t, bodyStr, "/docs")
		assert.Contains(t, bodyStr, "/docs/redoc")
	})
}

func setupTestServerWithDocs(t *testing.T) *gin.Engine {
	// Start PostgreSQL container
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Cleanup container after test
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	dbPool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		dbPool.Close()
	})

	// Run basic migration for integration test
	_, err = dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			phone_number VARCHAR(20),
			is_active BOOLEAN NOT NULL DEFAULT true,
			is_verified BOOLEAN NOT NULL DEFAULT false,
			verified_at TIMESTAMP,
			last_login_at TIMESTAMP,
			failed_login_attempts INT NOT NULL DEFAULT 0,
			locked_until TIMESTAMP,
			mfa_enabled BOOLEAN NOT NULL DEFAULT false,
			mfa_secret VARCHAR(255),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP
		)`)
	require.NoError(t, err)

	// Set up services
	logger := zap.NewNop()
	userRepo := repositories.NewUserRepository(dbPool)
	userService := services.NewUserService(userRepo)

	// Set up token service
	tokenService := services.NewTokenService(
		"test-secret-key-at-least-32-bytes-long!!",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
		userRepo,
	)

	// Set up password services
	passwordHasher := security.NewPasswordHasher()
	passwordValidator := security.NewPasswordValidator(&security.PasswordPolicy{
		MinLength:           8,
		RequireUppercase:    1,
		RequireLowercase:    1,
		RequireNumbers:      1,
		RequireSpecialChars: 0,
		MinEntropy:          30,
	})

	// Create handlers
	authHandler := handlers.NewAuthHandler(
		userService,
		tokenService,
		passwordHasher,
		passwordValidator,
		logger,
	)
	docsHandler := handlers.NewDocsHandler()

	// Create middleware adapters
	tokenMiddleware := middleware.NewTokenServiceAdapter(tokenService)
	rbacMiddleware := middleware.NewSimpleRBACService()

	// Set up server
	cfg := &config.Config{
		Port:        8080,
		Environment: "test",
		Version:     "1.0.0",
		StartTime:   time.Now(),
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		RateLimit: config.RateLimitConfig{
			Global: 100,
		},
	}

	services := &server.Services{
		UserService:  userService,
		TokenService: tokenMiddleware,
		RBACService:  rbacMiddleware,
		AuthHandler:  authHandler,
		DocsHandler:  docsHandler,
	}

	httpServer := server.New(cfg, services, logger)
	httpServer.Setup()
	return httpServer.Router()
}
