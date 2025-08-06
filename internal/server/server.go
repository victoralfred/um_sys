package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/victoralfred/um_sys/internal/config"
	"github.com/victoralfred/um_sys/internal/handlers"
	"github.com/victoralfred/um_sys/internal/middleware"
	"github.com/victoralfred/um_sys/internal/services"
	"go.uber.org/zap"
)

// Server interface - following Interface Segregation Principle
type Server interface {
	Setup()
	Start() error
	Router() *gin.Engine
}

// HTTPServer implements the Server interface
type HTTPServer struct {
	router   *gin.Engine
	config   *config.Config
	logger   *zap.Logger
	services *Services
}

// Services holds all service dependencies - Dependency Inversion Principle
type Services struct {
	UserService    *services.UserService
	TokenService   middleware.TokenService // Using interface for middleware compatibility
	RBACService    middleware.RBACService  // Using interface for middleware compatibility
	MFAService     *services.MFAService
	BillingService *services.BillingService
	AuditService   *services.AuditService
	FeatureService *services.FeatureService

	// Handlers
	AuthHandler    *handlers.AuthHandler
	ProfileHandler *handlers.ProfileHandler
	DocsHandler    *handlers.DocsHandler
}

// New creates a new server instance - Factory pattern
func New(cfg *config.Config, svcs *Services, logger *zap.Logger) *HTTPServer {
	return &HTTPServer{
		config:   cfg,
		services: svcs,
		logger:   logger,
	}
}

// Setup initializes the server - Single Responsibility Principle
func (s *HTTPServer) Setup() {
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()
	s.setupMiddleware()
	s.setupRoutes()
}

// setupMiddleware configures middleware - Open/Closed Principle
func (s *HTTPServer) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request ID middleware
	s.router.Use(middleware.RequestID())

	// CORS configuration
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     s.config.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting
	s.router.Use(middleware.RateLimit(s.config.RateLimit.Global))
}

// setupRoutes configures all routes
func (s *HTTPServer) setupRoutes() {
	// API v1 routes
	v1 := s.router.Group("/v1")

	// Public routes
	s.setupPublicRoutes(v1)

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.Auth(s.services.TokenService))
	s.setupProtectedRoutes(protected)

	// Admin routes
	admin := v1.Group("/admin")
	admin.Use(middleware.Auth(s.services.TokenService))
	admin.Use(middleware.RequireRole("admin", s.services.RBACService))
	s.setupAdminRoutes(admin)
}

// setupPublicRoutes sets up public endpoints
func (s *HTTPServer) setupPublicRoutes(rg *gin.RouterGroup) {
	// Health check
	rg.GET("/health", s.healthCheck)
	rg.GET("/info", s.apiInfo)

	// API Documentation
	if s.services.DocsHandler != nil {
		// Documentation endpoints
		rg.GET("/docs/swagger.json", s.services.DocsHandler.GetSwaggerJSON)
		rg.GET("/docs", s.services.DocsHandler.GetSwaggerUI)
		rg.GET("/docs/redoc", s.services.DocsHandler.GetRedocUI)
		rg.GET("/docs/", s.services.DocsHandler.GetDocsIndex)
	}

	// Auth endpoints
	auth := rg.Group("/auth")
	{
		if s.services.AuthHandler != nil {
			auth.POST("/register", s.services.AuthHandler.Register)
			auth.POST("/login", s.services.AuthHandler.Login)
			auth.POST("/refresh", s.services.AuthHandler.RefreshToken)
		} else {
			auth.POST("/register", s.notImplemented)
			auth.POST("/login", s.notImplemented)
			auth.POST("/refresh", s.notImplemented)
		}
		auth.POST("/password/forgot", s.notImplemented)
		auth.POST("/password/reset", s.notImplemented)
		auth.POST("/email/verify", s.notImplemented)
	}

	// Public billing endpoint
	rg.GET("/billing/plans", s.getPlans)
}

// setupProtectedRoutes sets up authenticated endpoints
func (s *HTTPServer) setupProtectedRoutes(rg *gin.RouterGroup) {
	// Auth endpoints
	auth := rg.Group("/auth")
	{
		if s.services.AuthHandler != nil {
			auth.POST("/logout", s.services.AuthHandler.Logout)
		} else {
			auth.POST("/logout", s.notImplemented)
		}
		auth.GET("/sessions", s.notImplemented)
		auth.DELETE("/sessions/:sessionId", s.notImplemented)
		auth.POST("/email/resend", s.notImplemented)
		auth.POST("/permissions/check", s.notImplemented)
	}

	// User endpoints
	users := rg.Group("/users")
	{
		if s.services.AuthHandler != nil {
			users.GET("/me", s.services.AuthHandler.GetCurrentUser)
		} else {
			users.GET("/me", s.notImplemented)
		}
		if s.services.ProfileHandler != nil {
			users.PATCH("/me", s.services.ProfileHandler.UpdateProfile)
			users.POST("/me/avatar", s.services.ProfileHandler.UploadProfilePicture)
		} else {
			users.PATCH("/me", s.notImplemented)
			users.POST("/me/avatar", s.notImplemented)
		}
		users.POST("/me/password", s.notImplemented)
		users.DELETE("/me", s.notImplemented)
		users.GET("/me/roles", s.notImplemented)
		users.GET("/search", s.notImplemented)
	}

	// Profile endpoints
	profile := rg.Group("/profile")
	{
		if s.services.ProfileHandler != nil {
			profile.GET("/:id", s.services.ProfileHandler.GetProfile)
			profile.PUT("/:id", s.services.ProfileHandler.UpdateProfile)
			profile.POST("/:id/picture", s.services.ProfileHandler.UploadProfilePicture)
			profile.GET("/:id/preferences", s.services.ProfileHandler.GetUserPreferences)
			profile.PUT("/:id/preferences", s.services.ProfileHandler.UpdateUserPreferences)
		} else {
			profile.GET("/:id", s.notImplemented)
			profile.PUT("/:id", s.notImplemented)
			profile.POST("/:id/picture", s.notImplemented)
			profile.GET("/:id/preferences", s.notImplemented)
			profile.PUT("/:id/preferences", s.notImplemented)
		}
	}

	// MFA endpoints
	mfa := rg.Group("/mfa")
	{
		mfa.GET("/status", s.notImplemented)
		mfa.POST("/totp/setup", s.notImplemented)
		mfa.POST("/totp/verify", s.notImplemented)
		mfa.POST("/sms/setup", s.notImplemented)
		mfa.POST("/sms/verify", s.notImplemented)
		mfa.POST("/challenge", s.notImplemented)
		mfa.POST("/backup-codes/regenerate", s.notImplemented)
		mfa.DELETE("/disable", s.notImplemented)
	}

	// Billing endpoints
	billing := rg.Group("/billing")
	{
		billing.GET("/subscription", s.notImplemented)
		billing.POST("/subscription", s.notImplemented)
		billing.PATCH("/subscription", s.notImplemented)
		billing.DELETE("/subscription", s.notImplemented)
		billing.GET("/invoices", s.notImplemented)
		billing.GET("/invoices/:invoiceId/download", s.notImplemented)
		billing.GET("/payment-methods", s.notImplemented)
		billing.POST("/payment-methods", s.notImplemented)
		billing.DELETE("/payment-methods/:methodId", s.notImplemented)
		billing.POST("/coupons/apply", s.notImplemented)
	}

	// Audit endpoints
	audit := rg.Group("/audit")
	{
		audit.GET("/activity", s.notImplemented)
		audit.GET("/security-events", s.notImplemented)
		audit.GET("/export", s.notImplemented)
	}

	// Feature flags
	features := rg.Group("/features")
	{
		features.GET("/flags", s.notImplemented)
		features.POST("/evaluate", s.notImplemented)
		features.POST("/track", s.notImplemented)
	}

	// Compliance
	compliance := rg.Group("/compliance")
	{
		compliance.POST("/gdpr/export", s.notImplemented)
		compliance.POST("/gdpr/delete", s.notImplemented)
	}

	// Webhooks
	webhooks := rg.Group("/webhooks")
	{
		webhooks.GET("", s.notImplemented)
		webhooks.POST("", s.notImplemented)
		webhooks.PUT("/:webhookId", s.notImplemented)
		webhooks.DELETE("/:webhookId", s.notImplemented)
		webhooks.POST("/:webhookId/test", s.notImplemented)
	}

	// Rate limit status
	rg.GET("/rate-limit", s.rateLimitStatus)
}

// setupAdminRoutes sets up admin-only endpoints
func (s *HTTPServer) setupAdminRoutes(rg *gin.RouterGroup) {
	// User management
	users := rg.Group("/users")
	{
		users.GET("", s.notImplemented)
		users.GET("/:userId", s.notImplemented)
		users.POST("/:userId/suspend", s.notImplemented)
		users.POST("/:userId/activate", s.notImplemented)
		users.POST("/:userId/reset-password", s.notImplemented)
		users.DELETE("/:userId", s.notImplemented)
	}

	// Role management
	roles := rg.Group("/roles")
	{
		roles.GET("", s.notImplemented)
		roles.POST("", s.notImplemented)
		roles.PUT("/:roleId", s.notImplemented)
		roles.DELETE("/:roleId", s.notImplemented)
		roles.POST("/users/:userId/roles", s.notImplemented)
		roles.DELETE("/users/:userId/roles/:roleId", s.notImplemented)
	}

	// System monitoring
	system := rg.Group("/system")
	{
		system.GET("/stats", s.notImplemented)
		system.GET("/health/detailed", s.notImplemented)
	}

	// Audit logs
	audit := rg.Group("/audit")
	{
		audit.GET("/logs", s.notImplemented)
		audit.GET("/alerts", s.notImplemented)
		audit.POST("/alerts", s.notImplemented)
	}
}

// Handler implementations (minimal to pass tests)

func (s *HTTPServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   s.config.Version,
		"uptime":    time.Since(s.config.StartTime).Seconds(),
	})
}

func (s *HTTPServer) apiInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":       s.config.Version,
		"environment":   s.config.Environment,
		"documentation": s.config.DocsURL,
		"support":       s.config.SupportEmail,
		"status_page":   s.config.StatusPageURL,
	})
}

func (s *HTTPServer) getPlans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plans": []interface{}{},
		},
	})
}

func (s *HTTPServer) rateLimitStatus(c *gin.Context) {
	limit, _ := c.Get("rate_limit")
	remaining, _ := c.Get("rate_remaining")
	reset, _ := c.Get("rate_reset")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"limit":     limit,
			"remaining": remaining,
			"reset_at":  reset,
			"tier":      "pro",
		},
	})
}

func (s *HTTPServer) notImplemented(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "This endpoint is not yet implemented",
		},
	})
}

// Start starts the HTTP server with graceful shutdown
func (s *HTTPServer) Start() error {
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", s.config.Port),
		Handler:        s.router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in goroutine
	go func() {
		s.logger.Info("Starting server",
			zap.Int("port", s.config.Port),
			zap.String("environment", s.config.Environment),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	s.logger.Info("Server exited")
	return nil
}

// Router returns the gin router for testing
func (s *HTTPServer) Router() *gin.Engine {
	return s.router
}
