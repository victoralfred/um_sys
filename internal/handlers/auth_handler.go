package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/user"
	"github.com/victoralfred/um_sys/internal/services"
	"github.com/victoralfred/um_sys/pkg/security"
	"go.uber.org/zap"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userService       *services.UserService
	tokenService      *services.TokenService
	passwordHasher    *security.PasswordHasher
	passwordValidator *security.PasswordValidator
	logger            *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	userService *services.UserService,
	tokenService *services.TokenService,
	passwordHasher *security.PasswordHasher,
	passwordValidator *security.PasswordValidator,
	logger *zap.Logger,
) *AuthHandler {
	return &AuthHandler{
		userService:       userService,
		tokenService:      tokenService,
		passwordHasher:    passwordHasher,
		passwordValidator: passwordValidator,
		logger:            logger,
	}
}

// RegisterRequest represents registration request
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=8,max=128"`
	FirstName string `json:"first_name" binding:"max=100"`
	LastName  string `json:"last_name" binding:"max=100"`
}

// RegisterResponse represents registration response
type RegisterResponse struct {
	Success bool                  `json:"success"`
	Data    *RegisterResponseData `json:"data,omitempty"`
	Error   *ErrorResponse        `json:"error,omitempty"`
}

type RegisterResponseData struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RegisterResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate password strength
	validationResult, err := h.passwordValidator.Validate(req.Password, req.Username, req.Email)
	if err != nil || !validationResult.IsValid {
		c.JSON(http.StatusBadRequest, RegisterResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "WEAK_PASSWORD",
				Message: "Password does not meet security requirements",
				Details: strings.Join(validationResult.Errors, "; "),
			},
		})
		return
	}

	// Check if email already exists
	existingUser, _ := h.userService.GetByEmail(c.Request.Context(), req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, RegisterResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "EMAIL_EXISTS",
				Message: "Email address is already registered",
			},
		})
		return
	}

	// Check if username already exists
	existingUser, _ = h.userService.GetByUsername(c.Request.Context(), req.Username)
	if existingUser != nil {
		c.JSON(http.StatusConflict, RegisterResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "USERNAME_EXISTS",
				Message: "Username is already taken",
			},
		})
		return
	}

	// Hash password
	hashedPassword, err := h.passwordHasher.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("Failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, RegisterResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to process registration",
			},
		})
		return
	}

	// Create user
	newUser := &user.User{
		ID:           uuid.New(),
		Email:        strings.ToLower(req.Email),
		Username:     req.Username,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       user.StatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.userService.Create(c.Request.Context(), newUser); err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, RegisterResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create user account",
			},
		})
		return
	}

	// TODO: Send verification email

	c.JSON(http.StatusCreated, RegisterResponse{
		Success: true,
		Data: &RegisterResponseData{
			UserID:  newUser.ID.String(),
			Email:   newUser.Email,
			Message: "Registration successful. Please check your email to verify your account.",
		},
	})
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required_without=Username"`
	Username string `json:"username" binding:"required_without=Email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Success bool               `json:"success"`
	Data    *LoginResponseData `json:"data,omitempty"`
	Error   *ErrorResponse     `json:"error,omitempty"`
}

type LoginResponseData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         *UserInfo `json:"user"`
}

type UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	// Find user by email or username
	var foundUser *user.User
	var err error

	if req.Email != "" {
		foundUser, err = h.userService.GetByEmail(c.Request.Context(), strings.ToLower(req.Email))
	} else {
		foundUser, err = h.userService.GetByUsername(c.Request.Context(), req.Username)
	}

	if err != nil || foundUser == nil {
		c.JSON(http.StatusUnauthorized, LoginResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "INVALID_CREDENTIALS",
				Message: "Invalid email/username or password",
			},
		})
		return
	}

	// Check if account is locked
	if foundUser.LockedUntil != nil && foundUser.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusUnauthorized, LoginResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "ACCOUNT_LOCKED",
				Message: "Account is temporarily locked due to multiple failed login attempts",
			},
		})
		return
	}

	// Verify password
	if !h.passwordHasher.VerifyPassword(req.Password, foundUser.PasswordHash) {
		// Increment failed login attempts
		_ = h.userService.IncrementFailedLoginAttempts(c.Request.Context(), foundUser.ID)

		c.JSON(http.StatusUnauthorized, LoginResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "INVALID_CREDENTIALS",
				Message: "Invalid email/username or password",
			},
		})
		return
	}

	// Check if account is active
	if foundUser.Status != user.StatusActive {
		c.JSON(http.StatusUnauthorized, LoginResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "ACCOUNT_INACTIVE",
				Message: "Account is not active",
			},
		})
		return
	}

	// Generate tokens
	tokenPair, err := h.tokenService.GenerateTokenPair(c.Request.Context(), foundUser)
	if err != nil {
		h.logger.Error("Failed to generate tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, LoginResponse{
			Success: false,
			Error: &ErrorResponse{
				Code:    "TOKEN_GENERATION_FAILED",
				Message: "Failed to generate authentication tokens",
			},
		})
		return
	}

	// Update last login
	_ = h.userService.UpdateLastLogin(c.Request.Context(), foundUser.ID, time.Now())

	c.JSON(http.StatusOK, LoginResponse{
		Success: true,
		Data: &LoginResponseData{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			TokenType:    tokenPair.TokenType,
			ExpiresIn:    tokenPair.ExpiresIn,
			ExpiresAt:    tokenPair.ExpiresAt,
			User: &UserInfo{
				ID:        foundUser.ID.String(),
				Email:     foundUser.Email,
				Username:  foundUser.Username,
				FirstName: foundUser.FirstName,
				LastName:  foundUser.LastName,
			},
		},
	})
}

// GetCurrentUser handles getting current user info
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID format",
			},
		})
		return
	}

	// Get user from database
	currentUser, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": gin.H{
				"id":             currentUser.ID.String(),
				"email":          currentUser.Email,
				"username":       currentUser.Username,
				"first_name":     currentUser.FirstName,
				"last_name":      currentUser.LastName,
				"phone_number":   currentUser.PhoneNumber,
				"email_verified": currentUser.EmailVerified,
				"mfa_enabled":    currentUser.MFAEnabled,
				"status":         currentUser.Status,
				"created_at":     currentUser.CreatedAt,
				"updated_at":     currentUser.UpdatedAt,
			},
		},
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
