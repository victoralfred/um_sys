package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victoralfred/um_sys/internal/config"
	"github.com/victoralfred/um_sys/internal/handlers"
	"github.com/victoralfred/um_sys/internal/middleware"
	"github.com/victoralfred/um_sys/internal/repositories"
	"github.com/victoralfred/um_sys/internal/server"
	"github.com/victoralfred/um_sys/internal/services"
	"github.com/victoralfred/um_sys/pkg/security"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("Starting User Management System Server...")

	// Database configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "umanager")

	// Construct database URL
	dbURL := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Connect to database
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	logger.Info("Connected to database successfully")

	// Run migrations
	logger.Info("Running database migrations...")
	if err := runMigrations(ctx, dbPool); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	// Initialize repositories
	userRepo := repositories.NewUserRepository(dbPool)

	// Initialize services
	userService := services.NewUserService(userRepo)

	tokenService := services.NewTokenService(
		getEnv("JWT_SECRET", "your-secret-key-change-this-in-production-min-32-chars!!"),
		"umanager",
		15*time.Minute, // Access token expiry
		7*24*time.Hour, // Refresh token expiry
		userRepo,
	)

	// Initialize security components
	passwordHasher := security.NewPasswordHasher()
	passwordValidator := security.NewPasswordValidator(&security.PasswordPolicy{
		MinLength:           8,
		RequireUppercase:    1,
		RequireLowercase:    1,
		RequireNumbers:      1,
		RequireSpecialChars: 0,
		MinEntropy:          30,
	})

	// Initialize handlers
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

	// Server configuration
	cfg := &config.Config{
		Port:        8080,
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     "1.0.0",
		StartTime:   time.Now(),
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		RateLimit: config.RateLimitConfig{
			Global: 100,
		},
		DocsURL:       "http://localhost:8080/docs",
		SupportEmail:  "support@umanager.local",
		StatusPageURL: "http://localhost:8080/status",
	}

	// Initialize server with services
	serverServices := &server.Services{
		UserService:  userService,
		TokenService: tokenMiddleware,
		RBACService:  rbacMiddleware,
		AuthHandler:  authHandler,
		DocsHandler:  docsHandler,
	}

	// Create and setup server
	httpServer := server.New(cfg, serverServices, logger)
	httpServer.Setup()

	// Print API information
	fmt.Println("\n===========================================")
	fmt.Println("🚀 User Management System API Server")
	fmt.Println("===========================================")
	fmt.Printf("Server running at: http://localhost:%d\n", cfg.Port)
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("\nPublic endpoints:")
	fmt.Println("  POST   /v1/auth/register    - Register new user")
	fmt.Println("  POST   /v1/auth/login       - Login user")
	fmt.Println("  POST   /v1/auth/refresh     - Refresh tokens")
	fmt.Println("  GET    /v1/health           - Health check")
	fmt.Println("  GET    /v1/info             - API info")
	fmt.Println("\nAPI Documentation:")
	fmt.Println("  GET    /v1/docs             - Interactive Swagger UI")
	fmt.Println("  GET    /v1/docs/            - Documentation index")
	fmt.Println("  GET    /v1/docs/redoc       - ReDoc documentation")
	fmt.Println("  GET    /v1/docs/swagger.json- OpenAPI specification")
	fmt.Println("\nProtected endpoints (require authentication):")
	fmt.Println("  GET    /v1/users/me         - Get current user")
	fmt.Println("  POST   /v1/auth/logout      - Logout user")
	fmt.Println("\n===========================================")
	fmt.Println("\nExample requests:")
	fmt.Println("\n1. Register a new user:")
	fmt.Println(`curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "testuser",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe"
  }'`)
	fmt.Println("\n2. Login:")
	fmt.Println(`curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'`)
	fmt.Println("\n3. Get current user (use token from login):")
	fmt.Println(`curl -X GET http://localhost:8080/v1/users/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"`)
	fmt.Println("\n===========================================")

	// Start server
	if err := httpServer.Start(); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	// Create users table if not exists
	query := `
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
	)`

	_, err := db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at)",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
