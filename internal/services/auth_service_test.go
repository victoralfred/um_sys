package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/auth"
	"github.com/victoralfred/um_sys/internal/domain/user"
	"github.com/victoralfred/um_sys/internal/services"
)

// Mock implementations
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*user.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdateMFA(ctx context.Context, id uuid.UUID, enabled bool, secret string, backupCodes []string) error {
	args := m.Called(ctx, id, enabled, secret, backupCodes)
	return args.Error(0)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateTokenPair(ctx context.Context, u *user.User) (*auth.TokenPair, error) {
	args := m.Called(ctx, u)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockTokenService) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType) (*auth.Claims, error) {
	args := m.Called(ctx, token, tokenType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Claims), args.Error(1)
}

func (m *MockTokenService) RefreshTokens(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockTokenService) RevokeToken(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockTokenService) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	args := m.Called(ctx, tokenID)
	return args.Bool(0), args.Error(1)
}

type MockPasswordHasher struct {
	mock.Mock
}

func (m *MockPasswordHasher) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordHasher) VerifyPassword(password, hash string) error {
	args := m.Called(password, hash)
	return args.Error(0)
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.RegisterRequest{
			Email:     "test@example.com",
			Username:  "testuser",
			Password:  "SecurePass123!",
			FirstName: "Test",
			LastName:  "User",
		}

		hashedPassword := "$2a$10$hashedpassword"

		mockRepo.On("ExistsByEmail", ctx, req.Email).Return(false, nil)
		mockRepo.On("ExistsByUsername", ctx, req.Username).Return(false, nil)
		mockHasher.On("HashPassword", req.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*user.User")).Return(nil)

		// Act
		createdUser, err := authService.Register(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, createdUser)
		assert.Equal(t, req.Email, createdUser.Email)
		assert.Equal(t, req.Username, createdUser.Username)
		assert.Equal(t, req.FirstName, createdUser.FirstName)
		assert.Equal(t, req.LastName, createdUser.LastName)
		assert.Equal(t, user.StatusActive, createdUser.Status)

		mockRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.RegisterRequest{
			Email:     "existing@example.com",
			Username:  "newuser",
			Password:  "SecurePass123!",
			FirstName: "Test",
			LastName:  "User",
		}

		mockRepo.On("ExistsByEmail", ctx, req.Email).Return(true, nil)

		// Act
		createdUser, err := authService.Register(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Contains(t, err.Error(), "email already exists")

		mockRepo.AssertExpectations(t)
	})

	t.Run("username already exists", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.RegisterRequest{
			Email:     "new@example.com",
			Username:  "existinguser",
			Password:  "SecurePass123!",
			FirstName: "Test",
			LastName:  "User",
		}

		mockRepo.On("ExistsByEmail", ctx, req.Email).Return(false, nil)
		mockRepo.On("ExistsByUsername", ctx, req.Username).Return(true, nil)

		// Act
		createdUser, err := authService.Register(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Contains(t, err.Error(), "username already exists")

		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("successful login with email", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.LoginRequest{
			Email:    "test@example.com",
			Password: "SecurePass123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		testUser := &user.User{
			ID:           uuid.New(),
			Email:        req.Email,
			Username:     "testuser",
			PasswordHash: hashedPassword,
			Status:       user.StatusActive,
			FirstName:    "Test",
			LastName:     "User",
		}

		tokenPair := &auth.TokenPair{
			AccessToken:  "access.token.here",
			RefreshToken: "refresh.token.here",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		mockRepo.On("GetByEmail", ctx, req.Email).Return(testUser, nil)
		mockHasher.On("VerifyPassword", req.Password, hashedPassword).Return(nil)
		mockTokenService.On("GenerateTokenPair", ctx, testUser).Return(tokenPair, nil)
		mockRepo.On("UpdateLastLogin", ctx, testUser.ID).Return(nil)

		// Act
		tokens, loggedInUser, err := authService.Login(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.NotNil(t, loggedInUser)
		assert.Equal(t, tokenPair.AccessToken, tokens.AccessToken)
		assert.Equal(t, testUser.ID, loggedInUser.ID)

		mockRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
	})

	t.Run("successful login with username", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.LoginRequest{
			Username: "testuser",
			Password: "SecurePass123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		testUser := &user.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Username:     req.Username,
			PasswordHash: hashedPassword,
			Status:       user.StatusActive,
			FirstName:    "Test",
			LastName:     "User",
		}

		tokenPair := &auth.TokenPair{
			AccessToken:  "access.token.here",
			RefreshToken: "refresh.token.here",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		mockRepo.On("GetByUsername", ctx, req.Username).Return(testUser, nil)
		mockHasher.On("VerifyPassword", req.Password, hashedPassword).Return(nil)
		mockTokenService.On("GenerateTokenPair", ctx, testUser).Return(tokenPair, nil)
		mockRepo.On("UpdateLastLogin", ctx, testUser.ID).Return(nil)

		// Act
		tokens, loggedInUser, err := authService.Login(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.NotNil(t, loggedInUser)
		assert.Equal(t, tokenPair.AccessToken, tokens.AccessToken)
		assert.Equal(t, testUser.ID, loggedInUser.ID)

		mockRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.LoginRequest{
			Email:    "test@example.com",
			Password: "WrongPassword",
		}

		hashedPassword := "$2a$10$hashedpassword"
		testUser := &user.User{
			ID:           uuid.New(),
			Email:        req.Email,
			Username:     "testuser",
			PasswordHash: hashedPassword,
			Status:       user.StatusActive,
		}

		mockRepo.On("GetByEmail", ctx, req.Email).Return(testUser, nil)
		mockHasher.On("VerifyPassword", req.Password, hashedPassword).Return(auth.ErrInvalidCredentials)

		// Act
		tokens, loggedInUser, err := authService.Login(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Nil(t, loggedInUser)
		assert.Equal(t, auth.ErrInvalidCredentials, err)

		mockRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "Password123!",
		}

		mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, user.ErrNotFound)

		// Act
		tokens, loggedInUser, err := authService.Login(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Nil(t, loggedInUser)
		assert.Equal(t, auth.ErrInvalidCredentials, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("inactive user", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		req := &auth.LoginRequest{
			Email:    "test@example.com",
			Password: "SecurePass123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		testUser := &user.User{
			ID:           uuid.New(),
			Email:        req.Email,
			Username:     "testuser",
			PasswordHash: hashedPassword,
			Status:       user.StatusInactive,
		}

		mockRepo.On("GetByEmail", ctx, req.Email).Return(testUser, nil)
		mockHasher.On("VerifyPassword", req.Password, hashedPassword).Return(nil)

		// Act
		tokens, loggedInUser, err := authService.Login(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Nil(t, loggedInUser)
		assert.Equal(t, auth.ErrAccountInactive, err)

		mockRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})
}

func TestAuthService_Logout(t *testing.T) {
	ctx := context.Background()

	t.Run("successful logout", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		userID := uuid.New()
		tokenID := "jti-123456"

		mockTokenService.On("RevokeToken", ctx, tokenID).Return(nil)

		// Act
		err := authService.Logout(ctx, userID, tokenID)

		// Assert
		assert.NoError(t, err)
		mockTokenService.AssertExpectations(t)
	})
}

func TestAuthService_RefreshTokens(t *testing.T) {
	ctx := context.Background()

	t.Run("successful token refresh", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		refreshToken := "refresh.token.here"
		newTokenPair := &auth.TokenPair{
			AccessToken:  "new.access.token",
			RefreshToken: "new.refresh.token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		mockTokenService.On("RefreshTokens", ctx, refreshToken).Return(newTokenPair, nil)

		// Act
		tokens, err := authService.RefreshTokens(ctx, refreshToken)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.Equal(t, newTokenPair.AccessToken, tokens.AccessToken)
		mockTokenService.AssertExpectations(t)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		token := "valid.access.token"
		claims := &auth.Claims{
			UserID:   uuid.New(),
			Email:    "test@example.com",
			Username: "testuser",
			Roles:    []string{"user"},
		}

		mockTokenService.On("ValidateToken", ctx, token, auth.AccessToken).Return(claims, nil)

		// Act
		validatedClaims, err := authService.ValidateToken(ctx, token)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, validatedClaims)
		assert.Equal(t, claims.UserID, validatedClaims.UserID)
		mockTokenService.AssertExpectations(t)
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("successful password change", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		userID := uuid.New()
		oldPassword := "OldPass123!"
		newPassword := "NewPass456!"
		oldHash := "$2a$10$oldhash"
		newHash := "$2a$10$newhash"

		testUser := &user.User{
			ID:           userID,
			Email:        "test@example.com",
			Username:     "testuser",
			PasswordHash: oldHash,
			Status:       user.StatusActive,
		}

		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil)
		mockHasher.On("VerifyPassword", oldPassword, oldHash).Return(nil)
		mockHasher.On("HashPassword", newPassword).Return(newHash, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*user.User")).Return(nil)

		// Act
		err := authService.ChangePassword(ctx, userID, oldPassword, newPassword)

		// Assert
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})

	t.Run("incorrect old password", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		mockTokenService := new(MockTokenService)
		mockHasher := new(MockPasswordHasher)

		authService := services.NewAuthService(mockRepo, mockTokenService, mockHasher)

		userID := uuid.New()
		oldPassword := "WrongOldPass!"
		newPassword := "NewPass456!"
		oldHash := "$2a$10$oldhash"

		testUser := &user.User{
			ID:           userID,
			Email:        "test@example.com",
			Username:     "testuser",
			PasswordHash: oldHash,
			Status:       user.StatusActive,
		}

		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil)
		mockHasher.On("VerifyPassword", oldPassword, oldHash).Return(auth.ErrInvalidCredentials)

		// Act
		err := authService.ChangePassword(ctx, userID, oldPassword, newPassword)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, auth.ErrInvalidCredentials, err)
		mockRepo.AssertExpectations(t)
		mockHasher.AssertExpectations(t)
	})
}
