package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/auth"
	"github.com/victoralfred/um_sys/internal/domain/user"
	"github.com/victoralfred/um_sys/internal/services"
)

func TestTokenService_GenerateTokenPair(t *testing.T) {
	ctx := context.Background()

	t.Run("successful token generation", func(t *testing.T) {
		// Arrange
		tokenService := services.NewTokenService("test-secret-key-at-least-32-bytes-long!", "test-issuer", 15*time.Minute, 7*24*time.Hour, nil)

		testUser := &user.User{
			ID:       uuid.New(),
			Email:    "test@example.com",
			Username: "testuser",
			Status:   user.StatusActive,
		}

		// Act
		tokenPair, err := tokenService.GenerateTokenPair(ctx, testUser)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.Equal(t, "Bearer", tokenPair.TokenType)
		assert.Equal(t, 900, tokenPair.ExpiresIn) // 15 minutes
		assert.True(t, tokenPair.ExpiresAt.After(time.Now()))
	})
}

func TestTokenService_ValidateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("valid access token", func(t *testing.T) {
		// Arrange
		tokenService := services.NewTokenService("test-secret-key-at-least-32-bytes-long!", "test-issuer", 15*time.Minute, 7*24*time.Hour, nil)

		testUser := &user.User{
			ID:       uuid.New(),
			Email:    "test@example.com",
			Username: "testuser",
			Status:   user.StatusActive,
		}

		tokenPair, err := tokenService.GenerateTokenPair(ctx, testUser)
		require.NoError(t, err)

		// Act
		claims, err := tokenService.ValidateToken(ctx, tokenPair.AccessToken, auth.AccessToken)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, claims.UserID)
		assert.Equal(t, testUser.Email, claims.Email)
		assert.Equal(t, testUser.Username, claims.Username)
		assert.Equal(t, auth.AccessToken, claims.TokenType)
	})

	t.Run("invalid token", func(t *testing.T) {
		// Arrange
		tokenService := services.NewTokenService("test-secret-key-at-least-32-bytes-long!", "test-issuer", 15*time.Minute, 7*24*time.Hour, nil)

		// Act
		claims, err := tokenService.ValidateToken(ctx, "invalid.token.here", auth.AccessToken)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("wrong token type", func(t *testing.T) {
		// Arrange
		tokenService := services.NewTokenService("test-secret-key-at-least-32-bytes-long!", "test-issuer", 15*time.Minute, 7*24*time.Hour, nil)

		testUser := &user.User{
			ID:       uuid.New(),
			Email:    "test@example.com",
			Username: "testuser",
			Status:   user.StatusActive,
		}

		tokenPair, err := tokenService.GenerateTokenPair(ctx, testUser)
		require.NoError(t, err)

		// Act - try to validate refresh token as access token
		claims, err := tokenService.ValidateToken(ctx, tokenPair.RefreshToken, auth.AccessToken)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestTokenService_RefreshTokens(t *testing.T) {
	ctx := context.Background()

	t.Run("successful token refresh", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockUserRepository)
		tokenService := services.NewTokenService("test-secret-key-at-least-32-bytes-long!", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockRepo)

		testUser := &user.User{
			ID:       uuid.New(),
			Email:    "test@example.com",
			Username: "testuser",
			Status:   user.StatusActive,
		}

		// Generate initial token pair
		initialTokenPair, err := tokenService.GenerateTokenPair(ctx, testUser)
		require.NoError(t, err)

		// Mock repository call
		mockRepo.On("GetByID", ctx, testUser.ID).Return(testUser, nil)

		// Act
		newTokenPair, err := tokenService.RefreshTokens(ctx, initialTokenPair.RefreshToken)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, newTokenPair.AccessToken)
		assert.NotEmpty(t, newTokenPair.RefreshToken)
		assert.NotEqual(t, initialTokenPair.AccessToken, newTokenPair.AccessToken)

		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		// Arrange
		tokenService := services.NewTokenService("test-secret-key-at-least-32-bytes-long!", "test-issuer", 15*time.Minute, 7*24*time.Hour, nil)

		// Act
		newTokenPair, err := tokenService.RefreshTokens(ctx, "invalid.refresh.token")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, newTokenPair)
	})
}
