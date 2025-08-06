package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTokenService for testing
type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) ValidateToken(token string) (*TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenClaims), args.Error(1)
}

func (m *MockTokenService) IsTokenBlacklisted(token string) bool {
	args := m.Called(token)
	return args.Bool(0)
}

// MockRBACService for testing
type MockRBACService struct {
	mock.Mock
}

func (m *MockRBACService) UserHasRole(userID, role string) (bool, error) {
	args := m.Called(userID, role)
	return args.Bool(0), args.Error(1)
}

func (m *MockRBACService) UserHasPermission(userID, permission string) (bool, error) {
	args := m.Called(userID, permission)
	return args.Bool(0), args.Error(1)
}

func TestAuth_MissingAuthHeader(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	tokenService := NewSimpleTokenService()

	router := gin.New()
	router.Use(Auth(tokenService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_MISSING_TOKEN")
}

func TestAuth_InvalidAuthFormat(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	tokenService := NewSimpleTokenService()

	router := gin.New()
	router.Use(Auth(tokenService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_INVALID_FORMAT")
}

func TestAuth_InvalidToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockTokenService := new(MockTokenService)
	mockTokenService.On("ValidateToken", "invalid-token").Return(nil, assert.AnError)

	router := gin.New()
	router.Use(Auth(mockTokenService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_INVALID_TOKEN")
	mockTokenService.AssertExpectations(t)
}

func TestAuth_BlacklistedToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockTokenService := new(MockTokenService)
	claims := &TokenClaims{
		UserID: "user123",
		Email:  "test@example.com",
	}
	mockTokenService.On("ValidateToken", "blacklisted-token").Return(claims, nil)
	mockTokenService.On("IsTokenBlacklisted", "blacklisted-token").Return(true)

	router := gin.New()
	router.Use(Auth(mockTokenService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer blacklisted-token")
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_TOKEN_REVOKED")
	mockTokenService.AssertExpectations(t)
}

func TestAuth_ValidToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockTokenService := new(MockTokenService)
	claims := &TokenClaims{
		UserID:      "user123",
		Email:       "test@example.com",
		Roles:       []string{"user"},
		Permissions: []string{"read:profile"},
	}
	mockTokenService.On("ValidateToken", "valid-token").Return(claims, nil)
	mockTokenService.On("IsTokenBlacklisted", "valid-token").Return(false)

	router := gin.New()
	router.Use(Auth(mockTokenService))
	router.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		email, _ := c.Get("email")
		c.JSON(200, gin.H{
			"user_id": userID,
			"email":   email,
		})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user123")
	assert.Contains(t, w.Body.String(), "test@example.com")
	mockTokenService.AssertExpectations(t)
}

func TestRequireRole_NoAuthentication(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	rbacService := NewSimpleRBACService()

	router := gin.New()
	router.Use(RequireRole("admin", rbacService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_NOT_AUTHENTICATED")
}

func TestRequireRole_InsufficientRole(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockRBACService := new(MockRBACService)
	mockRBACService.On("UserHasRole", "user123", "admin").Return(false, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user123")
		c.Next()
	})
	router.Use(RequireRole("admin", mockRBACService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "RBAC_INSUFFICIENT_ROLE")
	mockRBACService.AssertExpectations(t)
}

func TestRequireRole_HasRole(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockRBACService := new(MockRBACService)
	mockRBACService.On("UserHasRole", "user123", "admin").Return(true, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user123")
		c.Next()
	})
	router.Use(RequireRole("admin", mockRBACService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	mockRBACService.AssertExpectations(t)
}

func TestRequirePermission_NoPermission(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockRBACService := new(MockRBACService)
	mockRBACService.On("UserHasPermission", "user123", "users:delete").Return(false, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user123")
		c.Next()
	})
	router.Use(RequirePermission("users:delete", mockRBACService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "RBAC_INSUFFICIENT_PERMISSION")
	mockRBACService.AssertExpectations(t)
}

func TestOptionalAuth_NoToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	tokenService := NewSimpleTokenService()

	router := gin.New()
	router.Use(OptionalAuth(tokenService))
	router.GET("/test", func(c *gin.Context) {
		authenticated, _ := c.Get("authenticated")
		c.JSON(200, gin.H{"authenticated": authenticated})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"authenticated":false`)
}

func TestOptionalAuth_ValidToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockTokenService := new(MockTokenService)
	claims := &TokenClaims{
		UserID: "user123",
		Email:  "test@example.com",
	}
	mockTokenService.On("ValidateToken", "valid-token").Return(claims, nil)
	mockTokenService.On("IsTokenBlacklisted", "valid-token").Return(false)

	router := gin.New()
	router.Use(OptionalAuth(mockTokenService))
	router.GET("/test", func(c *gin.Context) {
		authenticated, _ := c.Get("authenticated")
		userID, _ := c.Get("user_id")
		c.JSON(200, gin.H{
			"authenticated": authenticated,
			"user_id":       userID,
		})
	})

	// Act
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"authenticated":true`)
	assert.Contains(t, w.Body.String(), "user123")
	mockTokenService.AssertExpectations(t)
}
