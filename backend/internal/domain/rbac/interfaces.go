package rbac

import (
	"context"

	"github.com/google/uuid"
)

// RoleRepository defines the interface for role persistence
type RoleRepository interface {
	// Create creates a new role
	Create(ctx context.Context, role *Role) error

	// GetByID retrieves a role by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)

	// GetByName retrieves a role by name
	GetByName(ctx context.Context, name string) (*Role, error)

	// Update updates a role
	Update(ctx context.Context, role *Role) error

	// Delete soft deletes a role
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all roles with pagination
	List(ctx context.Context, limit, offset int) ([]*Role, int64, error)

	// GetUserRoles retrieves all roles for a user
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)

	// AssignRole assigns a role to a user
	AssignRole(ctx context.Context, userRole *UserRole) error

	// RemoveRole removes a role from a user
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error

	// HasRole checks if a user has a specific role
	HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
}

// PermissionRepository defines the interface for permission persistence
type PermissionRepository interface {
	// Create creates a new permission
	Create(ctx context.Context, permission *Permission) error

	// GetByID retrieves a permission by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Permission, error)

	// GetByResourceAction retrieves a permission by resource and action
	GetByResourceAction(ctx context.Context, resource, action string) (*Permission, error)

	// Update updates a permission
	Update(ctx context.Context, permission *Permission) error

	// Delete deletes a permission
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all permissions with pagination
	List(ctx context.Context, limit, offset int) ([]*Permission, int64, error)

	// GetRolePermissions retrieves all permissions for a role
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*Permission, error)

	// GetUserPermissions retrieves all permissions for a user (through their roles)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*Permission, error)

	// GrantPermission grants a permission to a role
	GrantPermission(ctx context.Context, roleID, permissionID uuid.UUID) error

	// RevokePermission revokes a permission from a role
	RevokePermission(ctx context.Context, roleID, permissionID uuid.UUID) error

	// HasPermission checks if a user has a specific permission
	HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
}

// PolicyRepository defines the interface for policy persistence
type PolicyRepository interface {
	// Create creates a new policy rule
	Create(ctx context.Context, policy *PolicyRule) error

	// GetByID retrieves a policy rule by ID
	GetByID(ctx context.Context, id uuid.UUID) (*PolicyRule, error)

	// Update updates a policy rule
	Update(ctx context.Context, policy *PolicyRule) error

	// Delete deletes a policy rule
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all policy rules with pagination
	List(ctx context.Context, limit, offset int) ([]*PolicyRule, int64, error)

	// GetApplicablePolicies retrieves all policies applicable to a resource/action
	GetApplicablePolicies(ctx context.Context, resource, action string) ([]*PolicyRule, error)

	// EvaluatePolicies evaluates policies for an access request
	EvaluatePolicies(ctx context.Context, req *AccessRequest) (*AccessResponse, error)
}

// RBACService defines the main interface for RBAC operations
type RBACService interface {
	// Role management
	CreateRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context, limit, offset int) ([]*Role, int64, error)

	// User-Role assignment
	AssignRoleToUser(ctx context.Context, userID, roleID, grantedBy uuid.UUID) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)

	// Permission management
	CreatePermission(ctx context.Context, permission *Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*Permission, error)
	UpdatePermission(ctx context.Context, permission *Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context, limit, offset int) ([]*Permission, int64, error)

	// Role-Permission assignment
	GrantPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RevokePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*Permission, error)

	// Access control
	CheckAccess(ctx context.Context, req *AccessRequest) (*AccessResponse, error)
	HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*Permission, error)

	// Policy management
	CreatePolicy(ctx context.Context, policy *PolicyRule) error
	UpdatePolicy(ctx context.Context, policy *PolicyRule) error
	DeletePolicy(ctx context.Context, id uuid.UUID) error
	EvaluatePolicies(ctx context.Context, req *AccessRequest) (*AccessResponse, error)

	// Initialization
	InitializeSystemRoles(ctx context.Context) error
	InitializeDefaultPermissions(ctx context.Context) error
}

// CacheService defines caching interface for RBAC
type CacheService interface {
	// GetUserRoles gets cached user roles
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)

	// SetUserRoles caches user roles
	SetUserRoles(ctx context.Context, userID uuid.UUID, roles []*Role) error

	// GetUserPermissions gets cached user permissions
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*Permission, error)

	// SetUserPermissions caches user permissions
	SetUserPermissions(ctx context.Context, userID uuid.UUID, permissions []*Permission) error

	// InvalidateUser invalidates all cache for a user
	InvalidateUser(ctx context.Context, userID uuid.UUID) error

	// InvalidateRole invalidates cache for all users with a role
	InvalidateRole(ctx context.Context, roleID uuid.UUID) error
}
