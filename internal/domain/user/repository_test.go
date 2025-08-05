package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// MockRepository is a mock implementation of user.Repository for unit tests
type MockRepository struct {
	CreateFunc           func(ctx context.Context, u *user.User) error
	GetByIDFunc          func(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmailFunc       func(ctx context.Context, email string) (*user.User, error)
	GetByUsernameFunc    func(ctx context.Context, username string) (*user.User, error)
	UpdateFunc           func(ctx context.Context, u *user.User) error
	DeleteFunc           func(ctx context.Context, id uuid.UUID) error
	ListFunc             func(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error)
	UpdateLastLoginFunc  func(ctx context.Context, id uuid.UUID) error
	UpdateMFAFunc        func(ctx context.Context, id uuid.UUID, enabled bool, secret string, backupCodes []string) error
	ExistsByEmailFunc    func(ctx context.Context, email string) (bool, error)
	ExistsByUsernameFunc func(ctx context.Context, username string) (bool, error)
}

func (m *MockRepository) Create(ctx context.Context, u *user.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, u)
	}
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, user.ErrUserNotFound
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, user.ErrUserNotFound
}

func (m *MockRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	if m.GetByUsernameFunc != nil {
		return m.GetByUsernameFunc(ctx, username)
	}
	return nil, user.ErrUserNotFound
}

func (m *MockRepository) Update(ctx context.Context, u *user.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, u)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	return []*user.User{}, 0, nil
}

func (m *MockRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	if m.UpdateLastLoginFunc != nil {
		return m.UpdateLastLoginFunc(ctx, id)
	}
	return nil
}

func (m *MockRepository) UpdateMFA(ctx context.Context, id uuid.UUID, enabled bool, secret string, backupCodes []string) error {
	if m.UpdateMFAFunc != nil {
		return m.UpdateMFAFunc(ctx, id, enabled, secret, backupCodes)
	}
	return nil
}

func (m *MockRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsByEmailFunc != nil {
		return m.ExistsByEmailFunc(ctx, email)
	}
	return false, nil
}

func (m *MockRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.ExistsByUsernameFunc != nil {
		return m.ExistsByUsernameFunc(ctx, username)
	}
	return false, nil
}

// TestUserDomainModel tests the user domain model
func TestUserDomainModel(t *testing.T) {
	t.Run("NewUser creates valid user", func(t *testing.T) {
		u, err := user.NewUser("test@example.com", "testuser", "hashed")
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, "test@example.com", u.Email)
		assert.Equal(t, "testuser", u.Username)
		assert.Equal(t, "hashed", u.PasswordHash)
		assert.NotEqual(t, uuid.Nil, u.ID)
	})

	t.Run("NewUser validates required fields", func(t *testing.T) {
		_, err := user.NewUser("", "testuser", "hashed")
		assert.ErrorIs(t, err, user.ErrEmailRequired)

		_, err = user.NewUser("test@example.com", "", "hashed")
		assert.ErrorIs(t, err, user.ErrUsernameRequired)

		_, err = user.NewUser("test@example.com", "testuser", "")
		assert.ErrorIs(t, err, user.ErrPasswordRequired)
	})
}
