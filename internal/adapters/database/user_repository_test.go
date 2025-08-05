package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoralfred/um_sys/internal/adapters/database"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

func TestUserRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	td := database.SetupTestDatabase(t)
	defer td.Cleanup()

	repo := database.NewUserRepository(td.Pool)

	tests := []struct {
		name    string
		user    *user.User
		wantErr bool
	}{
		{
			name: "create valid user",
			user: &user.User{
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashed_password",
				FirstName:    "Test",
				LastName:     "User",
				Status:       user.StatusActive,
			},
			wantErr: false,
		},
		{
			name:    "create user with nil",
			user:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.user)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.user.ID)
				assert.NotZero(t, tt.user.CreatedAt)
				assert.NotZero(t, tt.user.UpdatedAt)
			}
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	td := database.SetupTestDatabase(t)
	defer td.Cleanup()

	repo := database.NewUserRepository(td.Pool)

	// Create test user
	testUser := &user.User{
		Email:        "getbyid@example.com",
		Username:     "getbyiduser",
		PasswordHash: "hashed_password",
		Status:       user.StatusActive,
	}
	err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "get existing user",
			id:      testUser.ID,
			wantErr: false,
		},
		{
			name:    "get non-existent user",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.id, got.ID)
			}
		})
	}
}
