package middleware

import (
	"context"
	"time"

	"github.com/victoralfred/um_sys/internal/domain/auth"
	"github.com/victoralfred/um_sys/internal/services"
)

// TokenServiceAdapter adapts services.TokenService to middleware.TokenService interface
// This follows the Adapter Pattern and Dependency Inversion Principle
type TokenServiceAdapter struct {
	service *services.TokenService
}

// NewTokenServiceAdapter creates a new adapter
func NewTokenServiceAdapter(service *services.TokenService) *TokenServiceAdapter {
	return &TokenServiceAdapter{
		service: service,
	}
}

// ValidateToken adapts the ValidateToken method
func (a *TokenServiceAdapter) ValidateToken(token string) (*TokenClaims, error) {
	ctx := context.Background()

	// Validate as access token
	claims, err := a.service.ValidateToken(ctx, token, auth.AccessToken)
	if err != nil {
		return nil, err
	}

	// Convert to middleware TokenClaims
	return &TokenClaims{
		UserID:      claims.UserID.String(),
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: extractPermissionsFromRoles(claims.Roles),
	}, nil
}

// IsTokenBlacklisted checks if token is blacklisted
func (a *TokenServiceAdapter) IsTokenBlacklisted(token string) bool {
	ctx := context.Background()

	// Parse token to get JTI
	claims, err := a.service.ValidateToken(ctx, token, auth.AccessToken)
	if err != nil {
		// If we can't validate, consider it invalid (not blacklisted)
		return false
	}

	// Check if revoked
	revoked, err := a.service.IsTokenRevoked(ctx, claims.JTI)
	if err != nil {
		// If we can't check, consider it not blacklisted
		return false
	}

	return revoked
}

// RBACServiceAdapter adapts services.RBACService to middleware.RBACService interface
type RBACServiceAdapter struct {
	service *services.RBACService
}

// NewRBACServiceAdapter creates a new adapter
func NewRBACServiceAdapter(service *services.RBACService) *RBACServiceAdapter {
	return &RBACServiceAdapter{
		service: service,
	}
}

// UserHasRole checks if user has role
func (a *RBACServiceAdapter) UserHasRole(userID, role string) (bool, error) {
	// TODO: Implement when RBACService has the method
	// For now, return a simple implementation
	if role == "admin" && userID == "admin-user-id" {
		return true, nil
	}
	return false, nil
}

// UserHasPermission checks if user has permission
func (a *RBACServiceAdapter) UserHasPermission(userID, permission string) (bool, error) {
	// TODO: Implement when RBACService has the method
	// For now, return a simple implementation
	return false, nil
}

// extractPermissionsFromRoles extracts permissions based on roles
// This is a simplified implementation - in production, permissions would come from database
func extractPermissionsFromRoles(roles []string) []string {
	permissions := []string{}

	for _, role := range roles {
		switch role {
		case "admin":
			permissions = append(permissions,
				"users:read", "users:write", "users:delete",
				"roles:read", "roles:write", "roles:delete",
				"billing:read", "billing:write",
				"audit:read", "audit:write",
			)
		case "manager":
			permissions = append(permissions,
				"users:read", "users:write",
				"roles:read",
				"billing:read",
				"audit:read",
			)
		case "user":
			permissions = append(permissions,
				"profile:read", "profile:write",
				"billing:read:own",
			)
		}
	}

	return unique(permissions)
}

// unique returns unique strings from slice
func unique(strings []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, s := range strings {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

// SimpleTokenService is a minimal implementation for testing
type SimpleTokenService struct {
	validTokens map[string]*TokenClaims
	blacklist   map[string]bool
}

// NewSimpleTokenService creates a simple token service for testing
func NewSimpleTokenService() *SimpleTokenService {
	return &SimpleTokenService{
		validTokens: make(map[string]*TokenClaims),
		blacklist:   make(map[string]bool),
	}
}

// AddValidToken adds a valid token for testing
func (s *SimpleTokenService) AddValidToken(token string, claims *TokenClaims) {
	s.validTokens[token] = claims
}

// BlacklistToken blacklists a token
func (s *SimpleTokenService) BlacklistToken(token string) {
	s.blacklist[token] = true
}

// ValidateToken validates a token
func (s *SimpleTokenService) ValidateToken(token string) (*TokenClaims, error) {
	claims, ok := s.validTokens[token]
	if !ok {
		return nil, auth.ErrInvalidToken
	}
	return claims, nil
}

// IsTokenBlacklisted checks if token is blacklisted
func (s *SimpleTokenService) IsTokenBlacklisted(token string) bool {
	return s.blacklist[token]
}

// SimpleRBACService is a minimal implementation for testing
type SimpleRBACService struct {
	userRoles       map[string][]string
	userPermissions map[string][]string
}

// NewSimpleRBACService creates a simple RBAC service for testing
func NewSimpleRBACService() *SimpleRBACService {
	return &SimpleRBACService{
		userRoles:       make(map[string][]string),
		userPermissions: make(map[string][]string),
	}
}

// SetUserRoles sets roles for a user
func (s *SimpleRBACService) SetUserRoles(userID string, roles []string) {
	s.userRoles[userID] = roles
}

// SetUserPermissions sets permissions for a user
func (s *SimpleRBACService) SetUserPermissions(userID string, permissions []string) {
	s.userPermissions[userID] = permissions
}

// UserHasRole checks if user has role
func (s *SimpleRBACService) UserHasRole(userID, role string) (bool, error) {
	roles, ok := s.userRoles[userID]
	if !ok {
		return false, nil
	}

	for _, r := range roles {
		if r == role {
			return true, nil
		}
	}

	return false, nil
}

// UserHasPermission checks if user has permission
func (s *SimpleRBACService) UserHasPermission(userID, permission string) (bool, error) {
	permissions, ok := s.userPermissions[userID]
	if !ok {
		return false, nil
	}

	for _, p := range permissions {
		if p == permission {
			return true, nil
		}
	}

	return false, nil
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if request is allowed
func (r *RateLimiter) Allow(key string) bool {
	now := time.Now()
	windowStart := now.Add(-r.window)

	// Get existing requests
	requests, ok := r.requests[key]
	if !ok {
		r.requests[key] = []time.Time{now}
		return true
	}

	// Filter out old requests
	validRequests := []time.Time{}
	for _, req := range requests {
		if req.After(windowStart) {
			validRequests = append(validRequests, req)
		}
	}

	// Check if under limit
	if len(validRequests) < r.limit {
		validRequests = append(validRequests, now)
		r.requests[key] = validRequests
		return true
	}

	r.requests[key] = validRequests
	return false
}

// Remaining returns remaining requests
func (r *RateLimiter) Remaining(key string) int {
	now := time.Now()
	windowStart := now.Add(-r.window)

	requests, ok := r.requests[key]
	if !ok {
		return r.limit
	}

	// Count valid requests
	count := 0
	for _, req := range requests {
		if req.After(windowStart) {
			count++
		}
	}

	return r.limit - count
}

// Reset returns when the rate limit resets
func (r *RateLimiter) Reset(key string) time.Time {
	requests, ok := r.requests[key]
	if !ok || len(requests) == 0 {
		return time.Now().Add(r.window)
	}

	// Find oldest request in window
	return requests[0].Add(r.window)
}
