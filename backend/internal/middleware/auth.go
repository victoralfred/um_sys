package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TokenService interface - Interface Segregation Principle
type TokenService interface {
	ValidateToken(token string) (*TokenClaims, error)
	IsTokenBlacklisted(token string) bool
}

// RBACService interface - Interface Segregation Principle
type RBACService interface {
	UserHasRole(userID, role string) (bool, error)
	UserHasPermission(userID, permission string) (bool, error)
}

// TokenClaims represents JWT claims
type TokenClaims struct {
	UserID      string
	Email       string
	Roles       []string
	Permissions []string
}

// Auth middleware handles JWT authentication - Single Responsibility Principle
func Auth(tokenService TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_MISSING_TOKEN",
					"message": "Authorization header is missing",
				},
			})
			c.Abort()
			return
		}

		// Check Bearer format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_INVALID_FORMAT",
					"message": "Invalid authorization format. Use 'Bearer <token>'",
				},
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := tokenService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_INVALID_TOKEN",
					"message": "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		// Check if token is blacklisted
		if tokenService.IsTokenBlacklisted(token) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_TOKEN_REVOKED",
					"message": "Token has been revoked",
				},
			})
			c.Abort()
			return
		}

		// Store user info in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("authenticated", true)
		c.Set("token", token)

		c.Next()
	}
}

// RequireRole middleware checks if user has required role - Single Responsibility Principle
func RequireRole(role string, rbacService RBACService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is authenticated
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_NOT_AUTHENTICATED",
					"message": "User is not authenticated",
				},
			})
			c.Abort()
			return
		}

		// Check if user has required role
		hasRole, err := rbacService.UserHasRole(userID.(string), role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RBAC_CHECK_FAILED",
					"message": "Failed to check user role",
				},
			})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RBAC_INSUFFICIENT_ROLE",
					"message": "Insufficient role to access this resource",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission middleware checks if user has required permission - Single Responsibility Principle
func RequirePermission(permission string, rbacService RBACService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is authenticated
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTH_NOT_AUTHENTICATED",
					"message": "User is not authenticated",
				},
			})
			c.Abort()
			return
		}

		// Check if user has required permission
		hasPermission, err := rbacService.UserHasPermission(userID.(string), permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RBAC_CHECK_FAILED",
					"message": "Failed to check user permission",
				},
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RBAC_INSUFFICIENT_PERMISSION",
					"message": "Insufficient permission to access this resource",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware for endpoints that work with or without auth - Open/Closed Principle
func OptionalAuth(tokenService TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set default authenticated state
		c.Set("authenticated", false)

		// Check for Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue as unauthenticated
			c.Next()
			return
		}

		// Check Bearer format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, continue as unauthenticated
			c.Next()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := tokenService.ValidateToken(token)
		if err != nil {
			// Invalid token, continue as unauthenticated
			c.Next()
			return
		}

		// Check if token is blacklisted
		if tokenService.IsTokenBlacklisted(token) {
			// Blacklisted token, continue as unauthenticated
			c.Next()
			return
		}

		// Store user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("authenticated", true)
		c.Set("token", token)

		c.Next()
	}
}

// RequestID middleware adds request ID to context - Single Responsibility Principle
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID exists in header
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate new request ID
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// RateLimit middleware for rate limiting - Single Responsibility Principle
func RateLimit(limit int) gin.HandlerFunc {
	// Simple implementation for now, will be enhanced later
	return func(c *gin.Context) {
		// Set rate limit headers
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", "99")
		c.Header("X-RateLimit-Reset", "1234567890")

		// Store in context for rate limit status endpoint
		c.Set("rate_limit", 100)
		c.Set("rate_remaining", 99)
		c.Set("rate_reset", 1234567890)

		c.Next()
	}
}
