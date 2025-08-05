package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/rbac"
	"github.com/victoralfred/um_sys/internal/services"
)

// Mock implementations
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *rbac.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*rbac.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rbac.Role), args.Error(1)
}

func (m *MockRoleRepository) GetByName(ctx context.Context, name string) (*rbac.Role, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rbac.Role), args.Error(1)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *rbac.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) List(ctx context.Context, limit, offset int) ([]*rbac.Role, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*rbac.Role), args.Get(1).(int64), args.Error(2)
}

func (m *MockRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*rbac.Role, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*rbac.Role), args.Error(1)
}

func (m *MockRoleRepository) AssignRole(ctx context.Context, userRole *rbac.UserRole) error {
	args := m.Called(ctx, userRole)
	return args.Error(0)
}

func (m *MockRoleRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	args := m.Called(ctx, userID, roleName)
	return args.Bool(0), args.Error(1)
}

type MockPermissionRepository struct {
	mock.Mock
}

func (m *MockPermissionRepository) Create(ctx context.Context, permission *rbac.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*rbac.Permission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rbac.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GetByResourceAction(ctx context.Context, resource, action string) (*rbac.Permission, error) {
	args := m.Called(ctx, resource, action)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rbac.Permission), args.Error(1)
}

func (m *MockPermissionRepository) Update(ctx context.Context, permission *rbac.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPermissionRepository) List(ctx context.Context, limit, offset int) ([]*rbac.Permission, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*rbac.Permission), args.Get(1).(int64), args.Error(2)
}

func (m *MockPermissionRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*rbac.Permission, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*rbac.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*rbac.Permission, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*rbac.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GrantPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockPermissionRepository) RevokePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockPermissionRepository) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func TestRBACService_CreateRole(t *testing.T) {
	ctx := context.Background()

	t.Run("successful role creation", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		newRole := &rbac.Role{
			ID:          uuid.New(),
			Name:        "test-role",
			Description: "Test role description",
			IsSystem:    false,
			Priority:    100,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRoleRepo.On("GetByName", ctx, newRole.Name).Return(nil, rbac.ErrRoleNotFound)
		mockRoleRepo.On("Create", ctx, newRole).Return(nil)

		// Act
		err := rbacService.CreateRole(ctx, newRole)

		// Assert
		assert.NoError(t, err)
		mockRoleRepo.AssertExpectations(t)
	})

	t.Run("role already exists", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		existingRole := &rbac.Role{
			ID:   uuid.New(),
			Name: "existing-role",
		}

		newRole := &rbac.Role{
			Name: "existing-role",
		}

		mockRoleRepo.On("GetByName", ctx, newRole.Name).Return(existingRole, nil)

		// Act
		err := rbacService.CreateRole(ctx, newRole)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, rbac.ErrRoleAlreadyExists, err)
		mockRoleRepo.AssertExpectations(t)
	})
}

func TestRBACService_AssignRoleToUser(t *testing.T) {
	ctx := context.Background()

	t.Run("successful role assignment", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		userID := uuid.New()
		roleID := uuid.New()
		grantedBy := uuid.New()

		role := &rbac.Role{
			ID:   roleID,
			Name: "test-role",
		}

		mockRoleRepo.On("GetByID", ctx, roleID).Return(role, nil)
		mockRoleRepo.On("GetUserRoles", ctx, userID).Return([]*rbac.Role{}, nil)
		mockRoleRepo.On("AssignRole", ctx, mock.AnythingOfType("*rbac.UserRole")).Return(nil)

		// Act
		err := rbacService.AssignRoleToUser(ctx, userID, roleID, grantedBy)

		// Assert
		assert.NoError(t, err)
		mockRoleRepo.AssertExpectations(t)
	})

	t.Run("role already assigned", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		userID := uuid.New()
		roleID := uuid.New()
		grantedBy := uuid.New()

		role := &rbac.Role{
			ID:   roleID,
			Name: "test-role",
		}

		mockRoleRepo.On("GetByID", ctx, roleID).Return(role, nil)
		mockRoleRepo.On("GetUserRoles", ctx, userID).Return([]*rbac.Role{role}, nil)

		// Act
		err := rbacService.AssignRoleToUser(ctx, userID, roleID, grantedBy)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, rbac.ErrRoleAlreadyAssigned, err)
		mockRoleRepo.AssertExpectations(t)
	})
}

func TestRBACService_CheckAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("access granted with permission", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		req := &rbac.AccessRequest{
			UserID:   uuid.New(),
			Resource: "users",
			Action:   "read",
		}

		mockPermRepo.On("HasPermission", ctx, req.UserID, req.Resource, req.Action).Return(true, nil)

		// Act
		resp, err := rbacService.CheckAccess(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.True(t, resp.Allowed)
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("access denied without permission", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		req := &rbac.AccessRequest{
			UserID:   uuid.New(),
			Resource: "billing",
			Action:   "manage",
		}

		mockPermRepo.On("HasPermission", ctx, req.UserID, req.Resource, req.Action).Return(false, nil)

		// Act
		resp, err := rbacService.CheckAccess(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.False(t, resp.Allowed)
		assert.Contains(t, resp.Reason, "insufficient permissions")
		mockPermRepo.AssertExpectations(t)
	})
}

func TestRBACService_InitializeSystemRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("successful initialization", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		// Expect checks for existing roles
		mockRoleRepo.On("GetByName", ctx, rbac.RoleSuperAdmin).Return(nil, rbac.ErrRoleNotFound)
		mockRoleRepo.On("GetByName", ctx, rbac.RoleAdmin).Return(nil, rbac.ErrRoleNotFound)
		mockRoleRepo.On("GetByName", ctx, rbac.RoleModerator).Return(nil, rbac.ErrRoleNotFound)
		mockRoleRepo.On("GetByName", ctx, rbac.RoleUser).Return(nil, rbac.ErrRoleNotFound)
		mockRoleRepo.On("GetByName", ctx, rbac.RoleGuest).Return(nil, rbac.ErrRoleNotFound)

		// Expect creation of system roles
		mockRoleRepo.On("Create", ctx, mock.AnythingOfType("*rbac.Role")).Return(nil).Times(5)

		// Act
		err := rbacService.InitializeSystemRoles(ctx)

		// Assert
		assert.NoError(t, err)
		mockRoleRepo.AssertExpectations(t)
	})
}

func TestRBACService_GrantPermissionToRole(t *testing.T) {
	ctx := context.Background()

	t.Run("successful permission grant", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		roleID := uuid.New()
		permissionID := uuid.New()

		role := &rbac.Role{
			ID:       roleID,
			Name:     "test-role",
			IsSystem: false,
		}

		permission := &rbac.Permission{
			ID:       permissionID,
			Resource: "users",
			Action:   "read",
		}

		mockRoleRepo.On("GetByID", ctx, roleID).Return(role, nil)
		mockPermRepo.On("GetByID", ctx, permissionID).Return(permission, nil)
		mockPermRepo.On("GetRolePermissions", ctx, roleID).Return([]*rbac.Permission{}, nil)
		mockPermRepo.On("GrantPermission", ctx, roleID, permissionID).Return(nil)

		// Act
		err := rbacService.GrantPermissionToRole(ctx, roleID, permissionID)

		// Assert
		assert.NoError(t, err)
		mockRoleRepo.AssertExpectations(t)
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("permission already granted", func(t *testing.T) {
		// Arrange
		mockRoleRepo := new(MockRoleRepository)
		mockPermRepo := new(MockPermissionRepository)

		rbacService := services.NewRBACService(mockRoleRepo, mockPermRepo, nil, nil)

		roleID := uuid.New()
		permissionID := uuid.New()

		role := &rbac.Role{
			ID:   roleID,
			Name: "test-role",
		}

		permission := &rbac.Permission{
			ID:       permissionID,
			Resource: "users",
			Action:   "read",
		}

		mockRoleRepo.On("GetByID", ctx, roleID).Return(role, nil)
		mockPermRepo.On("GetByID", ctx, permissionID).Return(permission, nil)
		mockPermRepo.On("GetRolePermissions", ctx, roleID).Return([]*rbac.Permission{permission}, nil)

		// Act
		err := rbacService.GrantPermissionToRole(ctx, roleID, permissionID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, rbac.ErrPermissionAlreadyGranted, err)
		mockRoleRepo.AssertExpectations(t)
		mockPermRepo.AssertExpectations(t)
	})
}
