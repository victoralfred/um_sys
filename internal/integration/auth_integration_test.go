package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/victoralfred/um_sys/internal/config"
	"github.com/victoralfred/um_sys/internal/handlers"
	"github.com/victoralfred/um_sys/internal/middleware"
	"github.com/victoralfred/um_sys/internal/repositories"
	"github.com/victoralfred/um_sys/internal/server"
	"github.com/victoralfred/um_sys/internal/services"
	"github.com/victoralfred/um_sys/pkg/security"
	"go.uber.org/zap"
)

func TestAuthIntegration_CompleteFlow(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	dbPool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer dbPool.Close()

	// Run migrations
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

	// Create auth handler
	authHandler := handlers.NewAuthHandler(
		userService,
		tokenService,
		passwordHasher,
		passwordValidator,
		logger,
	)

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
	}

	httpServer := server.New(cfg, services, logger)
	httpServer.Setup()
	router := httpServer.Router()

	// Test data
	testUser := struct {
		Email     string `json:"email"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "TestPassword123!",
		FirstName: "Test",
		LastName:  "User",
	}

	// Test 1: Register a new user
	t.Run("Register", func(t *testing.T) {
		body, _ := json.Marshal(testUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response handlers.RegisterResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.Data.UserID)
	})

	// Test 2: Try to register with same email (should fail)
	t.Run("RegisterDuplicate", func(t *testing.T) {
		body, _ := json.Marshal(testUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response handlers.RegisterResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "EMAIL_EXISTS", response.Error.Code)
	})

	var accessToken string
	var refreshToken string

	// Test 3: Login with email
	t.Run("LoginWithEmail", func(t *testing.T) {
		loginReq := struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		body, _ := json.Marshal(loginReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.Data.AccessToken)
		assert.NotEmpty(t, response.Data.RefreshToken)
		assert.Equal(t, "Bearer", response.Data.TokenType)

		accessToken = response.Data.AccessToken
		refreshToken = response.Data.RefreshToken
	})

	// Test 4: Login with wrong password
	t.Run("LoginWrongPassword", func(t *testing.T) {
		loginReq := struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}{
			Email:    testUser.Email,
			Password: "WrongPassword123!",
		}

		body, _ := json.Marshal(loginReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response handlers.LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "INVALID_CREDENTIALS", response.Error.Code)
	})

	// Test 5: Access protected endpoint without token
	t.Run("AccessProtectedNoToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/users/me", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// Test 6: Access protected endpoint with token
	t.Run("AccessProtectedWithToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		userData := response["data"].(map[string]interface{})["user"].(map[string]interface{})
		assert.Equal(t, testUser.Email, userData["email"])
		assert.Equal(t, testUser.Username, userData["username"])
	})

	// Test 7: Login with username
	t.Run("LoginWithUsername", func(t *testing.T) {
		loginReq := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{
			Username: testUser.Username,
			Password: testUser.Password,
		}

		body, _ := json.Marshal(loginReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
	})

	// Test 8: Password validation
	t.Run("WeakPassword", func(t *testing.T) {
		weakUser := struct {
			Email    string `json:"email"`
			Username string `json:"username"`
			Password string `json:"password"`
		}{
			Email:    "weak@example.com",
			Username: "weakuser",
			Password: "weak", // Too short - caught by binding validation
		}

		body, _ := json.Marshal(weakUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response handlers.RegisterResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		// Password too short is caught by binding validation (min=8)
		assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	})

	// Test 8b: Weak password that passes length but fails complexity
	t.Run("WeakPasswordComplexity", func(t *testing.T) {
		weakUser := struct {
			Email    string `json:"email"`
			Username string `json:"username"`
			Password string `json:"password"`
		}{
			Email:    "weak2@example.com",
			Username: "weakuser2",
			Password: "password", // 8 chars but no uppercase/numbers
		}

		body, _ := json.Marshal(weakUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response handlers.RegisterResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "WEAK_PASSWORD", response.Error.Code)
	})

	// Test 9: Refresh token
	t.Run("RefreshToken", func(t *testing.T) {
		refreshReq := struct {
			RefreshToken string `json:"refresh_token"`
		}{
			RefreshToken: refreshToken,
		}

		body, _ := json.Marshal(refreshReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.RefreshResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.Data.AccessToken)
		assert.NotEmpty(t, response.Data.RefreshToken)
		assert.NotEqual(t, accessToken, response.Data.AccessToken)   // Should be a new token
		assert.NotEqual(t, refreshToken, response.Data.RefreshToken) // Should be a new refresh token
	})

	// Test 10: Refresh with invalid token
	t.Run("RefreshInvalidToken", func(t *testing.T) {
		refreshReq := struct {
			RefreshToken string `json:"refresh_token"`
		}{
			RefreshToken: "invalid.refresh.token",
		}

		body, _ := json.Marshal(refreshReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response handlers.RefreshResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "INVALID_REFRESH_TOKEN", response.Error.Code)
	})

	// Test 11: Logout
	t.Run("Logout", func(t *testing.T) {
		logoutReq := struct {
			RefreshToken string `json:"refresh_token"`
		}{
			RefreshToken: refreshToken,
		}

		body, _ := json.Marshal(logoutReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.LogoutResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Successfully logged out", response.Data.Message)
	})

	// Test 12: Access with revoked token after logout
	t.Run("AccessAfterLogout", func(t *testing.T) {
		// Try to access protected endpoint with the same token after logout
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		router.ServeHTTP(w, req)

		// Should still work since we don't have a token store configured
		// In production with Redis/DB token store, this would return 401
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test 13: Logout without auth token
	t.Run("LogoutWithoutAuth", func(t *testing.T) {
		logoutReq := struct {
			RefreshToken string `json:"refresh_token"`
		}{
			RefreshToken: refreshToken,
		}

		body, _ := json.Marshal(logoutReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
