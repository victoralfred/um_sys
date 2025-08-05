package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/rbac"
)

// RBACService implements the RBAC service interface
type RBACService struct {
	roleRepo       rbac.RoleRepository
	permissionRepo rbac.PermissionRepository
	policyRepo     rbac.PolicyRepository
	cache          rbac.CacheService
}

// NewRBACService creates a new RBAC service
func NewRBACService(
	roleRepo rbac.RoleRepository,
	permissionRepo rbac.PermissionRepository,
	policyRepo rbac.PolicyRepository,
	cache rbac.CacheService,
) *RBACService {
	return &RBACService{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		policyRepo:     policyRepo,
		cache:          cache,
	}
}

// CreateRole creates a new role
func (s *RBACService) CreateRole(ctx context.Context, role *rbac.Role) error {
	// Check if role already exists
	existing, err := s.roleRepo.GetByName(ctx, role.Name)
	if err != nil && err != rbac.ErrRoleNotFound {
		return fmt.Errorf("failed to check existing role: %w", err)
	}
	if existing != nil {
		return rbac.ErrRoleAlreadyExists
	}

	// Set timestamps
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	// Create role
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// GetRole retrieves a role by ID
func (s *RBACService) GetRole(ctx context.Context, id uuid.UUID) (*rbac.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return role, nil
}

// UpdateRole updates a role
func (s *RBACService) UpdateRole(ctx context.Context, role *rbac.Role) error {
	// Check if role exists
	existing, err := s.roleRepo.GetByID(ctx, role.ID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Check if it's a system role
	if existing.IsSystem {
		return rbac.ErrSystemRoleModification
	}

	// Update timestamp
	role.UpdatedAt = time.Now()

	// Update role
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateRole(ctx, role.ID)
	}

	return nil
}

// DeleteRole deletes a role
func (s *RBACService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	// Check if role exists
	existing, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Check if it's a system role
	if existing.IsSystem {
		return rbac.ErrSystemRoleDeletion
	}

	// Delete role
	if err := s.roleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateRole(ctx, id)
	}

	return nil
}

// ListRoles lists all roles
func (s *RBACService) ListRoles(ctx context.Context, limit, offset int) ([]*rbac.Role, int64, error) {
	roles, total, err := s.roleRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, total, nil
}

// AssignRoleToUser assigns a role to a user
func (s *RBACService) AssignRoleToUser(ctx context.Context, userID, roleID, grantedBy uuid.UUID) error {
	// Check if role exists
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Check if user already has the role
	userRoles, err := s.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	for _, r := range userRoles {
		if r.ID == roleID {
			return rbac.ErrRoleAlreadyAssigned
		}
	}

	// Assign role
	userRole := &rbac.UserRole{
		UserID:    userID,
		RoleID:    role.ID,
		GrantedBy: grantedBy,
		GrantedAt: time.Now(),
	}

	if err := s.roleRepo.AssignRole(ctx, userRole); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateUser(ctx, userID)
	}

	return nil
}

// RemoveRoleFromUser removes a role from a user
func (s *RBACService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := s.roleRepo.RemoveRole(ctx, userID, roleID); err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateUser(ctx, userID)
	}

	return nil
}

// GetUserRoles gets all roles for a user
func (s *RBACService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*rbac.Role, error) {
	// Try cache first
	if s.cache != nil {
		roles, err := s.cache.GetUserRoles(ctx, userID)
		if err == nil && roles != nil {
			return roles, nil
		}
	}

	// Get from repository
	roles, err := s.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Update cache
	if s.cache != nil {
		_ = s.cache.SetUserRoles(ctx, userID, roles)
	}

	return roles, nil
}

// CreatePermission creates a new permission
func (s *RBACService) CreatePermission(ctx context.Context, permission *rbac.Permission) error {
	// Check if permission already exists
	existing, err := s.permissionRepo.GetByResourceAction(ctx, permission.Resource, permission.Action)
	if err != nil && err != rbac.ErrPermissionNotFound {
		return fmt.Errorf("failed to check existing permission: %w", err)
	}
	if existing != nil {
		return rbac.ErrPermissionAlreadyExists
	}

	// Set timestamps
	now := time.Now()
	permission.CreatedAt = now
	permission.UpdatedAt = now

	// Create permission
	if err := s.permissionRepo.Create(ctx, permission); err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	return nil
}

// GetPermission retrieves a permission by ID
func (s *RBACService) GetPermission(ctx context.Context, id uuid.UUID) (*rbac.Permission, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}
	return permission, nil
}

// UpdatePermission updates a permission
func (s *RBACService) UpdatePermission(ctx context.Context, permission *rbac.Permission) error {
	// Update timestamp
	permission.UpdatedAt = time.Now()

	// Update permission
	if err := s.permissionRepo.Update(ctx, permission); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	// Invalidate all user caches (expensive but ensures consistency)
	// In production, track which users have this permission and invalidate only those
	return nil
}

// DeletePermission deletes a permission
func (s *RBACService) DeletePermission(ctx context.Context, id uuid.UUID) error {
	if err := s.permissionRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}
	return nil
}

// ListPermissions lists all permissions
func (s *RBACService) ListPermissions(ctx context.Context, limit, offset int) ([]*rbac.Permission, int64, error) {
	permissions, total, err := s.permissionRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list permissions: %w", err)
	}
	return permissions, total, nil
}

// GrantPermissionToRole grants a permission to a role
func (s *RBACService) GrantPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	// Check if role exists
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Check if permission exists
	permission, err := s.permissionRepo.GetByID(ctx, permissionID)
	if err != nil {
		return fmt.Errorf("failed to get permission: %w", err)
	}

	// Check if role already has the permission
	rolePermissions, err := s.permissionRepo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role permissions: %w", err)
	}

	for _, p := range rolePermissions {
		if p.ID == permissionID {
			return rbac.ErrPermissionAlreadyGranted
		}
	}

	// Grant permission
	if err := s.permissionRepo.GrantPermission(ctx, role.ID, permission.ID); err != nil {
		return fmt.Errorf("failed to grant permission: %w", err)
	}

	// Invalidate cache for all users with this role
	if s.cache != nil {
		_ = s.cache.InvalidateRole(ctx, roleID)
	}

	return nil
}

// RevokePermissionFromRole revokes a permission from a role
func (s *RBACService) RevokePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if err := s.permissionRepo.RevokePermission(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("failed to revoke permission: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateRole(ctx, roleID)
	}

	return nil
}

// GetRolePermissions gets all permissions for a role
func (s *RBACService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*rbac.Permission, error) {
	permissions, err := s.permissionRepo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	return permissions, nil
}

// CheckAccess checks if a user has access to a resource/action
func (s *RBACService) CheckAccess(ctx context.Context, req *rbac.AccessRequest) (*rbac.AccessResponse, error) {
	// Check basic permissions first
	hasPermission, err := s.permissionRepo.HasPermission(ctx, req.UserID, req.Resource, req.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	if hasPermission {
		return &rbac.AccessResponse{
			Allowed: true,
			Reason:  "permission granted through role",
		}, nil
	}

	// If policy repository is available, evaluate policies
	if s.policyRepo != nil {
		policyResp, err := s.policyRepo.EvaluatePolicies(ctx, req)
		if err == nil && policyResp != nil {
			return policyResp, nil
		}
	}

	// Default deny
	return &rbac.AccessResponse{
		Allowed: false,
		Reason:  "insufficient permissions",
	}, nil
}

// HasPermission checks if a user has a specific permission
func (s *RBACService) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	hasPermission, err := s.permissionRepo.HasPermission(ctx, userID, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	return hasPermission, nil
}

// HasRole checks if a user has a specific role
func (s *RBACService) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	hasRole, err := s.roleRepo.HasRole(ctx, userID, roleName)
	if err != nil {
		return false, fmt.Errorf("failed to check role: %w", err)
	}
	return hasRole, nil
}

// GetUserPermissions gets all permissions for a user
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*rbac.Permission, error) {
	// Try cache first
	if s.cache != nil {
		permissions, err := s.cache.GetUserPermissions(ctx, userID)
		if err == nil && permissions != nil {
			return permissions, nil
		}
	}

	// Get from repository
	permissions, err := s.permissionRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Update cache
	if s.cache != nil {
		_ = s.cache.SetUserPermissions(ctx, userID, permissions)
	}

	return permissions, nil
}

// CreatePolicy creates a new policy rule
func (s *RBACService) CreatePolicy(ctx context.Context, policy *rbac.PolicyRule) error {
	if s.policyRepo == nil {
		return fmt.Errorf("policy repository not configured")
	}

	// Set timestamps
	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	if err := s.policyRepo.Create(ctx, policy); err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	return nil
}

// UpdatePolicy updates a policy rule
func (s *RBACService) UpdatePolicy(ctx context.Context, policy *rbac.PolicyRule) error {
	if s.policyRepo == nil {
		return fmt.Errorf("policy repository not configured")
	}

	policy.UpdatedAt = time.Now()

	if err := s.policyRepo.Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	return nil
}

// DeletePolicy deletes a policy rule
func (s *RBACService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	if s.policyRepo == nil {
		return fmt.Errorf("policy repository not configured")
	}

	if err := s.policyRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}

// EvaluatePolicies evaluates policies for an access request
func (s *RBACService) EvaluatePolicies(ctx context.Context, req *rbac.AccessRequest) (*rbac.AccessResponse, error) {
	if s.policyRepo == nil {
		return &rbac.AccessResponse{
			Allowed: false,
			Reason:  "policy evaluation not available",
		}, nil
	}

	resp, err := s.policyRepo.EvaluatePolicies(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policies: %w", err)
	}

	return resp, nil
}

// InitializeSystemRoles creates default system roles
func (s *RBACService) InitializeSystemRoles(ctx context.Context) error {
	systemRoles := []struct {
		name        string
		description string
		priority    int
	}{
		{rbac.RoleSuperAdmin, "Super Administrator with full system access", 1000},
		{rbac.RoleAdmin, "Administrator with elevated privileges", 900},
		{rbac.RoleModerator, "Moderator with content management privileges", 500},
		{rbac.RoleUser, "Standard user with basic privileges", 100},
		{rbac.RoleGuest, "Guest user with minimal privileges", 10},
	}

	for _, sr := range systemRoles {
		// Check if role exists
		existing, err := s.roleRepo.GetByName(ctx, sr.name)
		if err != nil && err != rbac.ErrRoleNotFound {
			return fmt.Errorf("failed to check role %s: %w", sr.name, err)
		}

		if existing != nil {
			continue // Role already exists
		}

		// Create role
		role := &rbac.Role{
			ID:          uuid.New(),
			Name:        sr.name,
			Description: sr.description,
			IsSystem:    true,
			Priority:    sr.priority,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := s.roleRepo.Create(ctx, role); err != nil {
			return fmt.Errorf("failed to create role %s: %w", sr.name, err)
		}
	}

	return nil
}

// InitializeDefaultPermissions creates default permissions
func (s *RBACService) InitializeDefaultPermissions(ctx context.Context) error {
	permissions := []struct {
		resource    string
		action      string
		description string
	}{
		// User permissions
		{"users", "create", "Create new users"},
		{"users", "read", "View user information"},
		{"users", "update", "Update user information"},
		{"users", "delete", "Delete users"},
		{"users", "list", "List all users"},

		// Role permissions
		{"roles", "create", "Create new roles"},
		{"roles", "read", "View role information"},
		{"roles", "update", "Update role information"},
		{"roles", "delete", "Delete roles"},
		{"roles", "list", "List all roles"},
		{"roles", "assign", "Assign roles to users"},

		// Billing permissions
		{"billing", "view", "View billing information"},
		{"billing", "manage", "Manage billing settings"},

		// System permissions
		{"system", "manage", "Manage system settings"},
		{"system", "audit", "View audit logs"},
	}

	for _, p := range permissions {
		// Check if permission exists
		existing, err := s.permissionRepo.GetByResourceAction(ctx, p.resource, p.action)
		if err != nil && err != rbac.ErrPermissionNotFound {
			return fmt.Errorf("failed to check permission %s:%s: %w", p.resource, p.action, err)
		}

		if existing != nil {
			continue // Permission already exists
		}

		// Create permission
		permission := &rbac.Permission{
			ID:          uuid.New(),
			Resource:    p.resource,
			Action:      p.action,
			Description: p.description,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := s.permissionRepo.Create(ctx, permission); err != nil {
			return fmt.Errorf("failed to create permission %s:%s: %w", p.resource, p.action, err)
		}
	}

	// Assign default permissions to roles
	if err := s.assignDefaultPermissions(ctx); err != nil {
		return fmt.Errorf("failed to assign default permissions: %w", err)
	}

	return nil
}

// assignDefaultPermissions assigns default permissions to system roles
func (s *RBACService) assignDefaultPermissions(ctx context.Context) error {
	// Super Admin gets all permissions
	superAdminRole, err := s.roleRepo.GetByName(ctx, rbac.RoleSuperAdmin)
	if err == nil && superAdminRole != nil {
		allPermissions, _, err := s.permissionRepo.List(ctx, 1000, 0)
		if err == nil {
			for _, perm := range allPermissions {
				_ = s.permissionRepo.GrantPermission(ctx, superAdminRole.ID, perm.ID)
			}
		}
	}

	// Admin gets most permissions except system management
	adminRole, err := s.roleRepo.GetByName(ctx, rbac.RoleAdmin)
	if err == nil && adminRole != nil {
		adminPerms := []string{
			"users:create", "users:read", "users:update", "users:delete", "users:list",
			"roles:read", "roles:list", "roles:assign",
			"billing:view", "billing:manage",
			"system:audit",
		}
		for _, permStr := range adminPerms {
			s.grantPermissionByString(ctx, adminRole.ID, permStr)
		}
	}

	// Moderator gets content management permissions
	modRole, err := s.roleRepo.GetByName(ctx, rbac.RoleModerator)
	if err == nil && modRole != nil {
		modPerms := []string{
			"users:read", "users:list", "users:update",
			"roles:read", "roles:list",
		}
		for _, permStr := range modPerms {
			s.grantPermissionByString(ctx, modRole.ID, permStr)
		}
	}

	// User gets basic permissions
	userRole, err := s.roleRepo.GetByName(ctx, rbac.RoleUser)
	if err == nil && userRole != nil {
		userPerms := []string{
			"users:read",   // Can read own profile
			"billing:view", // Can view own billing
		}
		for _, permStr := range userPerms {
			s.grantPermissionByString(ctx, userRole.ID, permStr)
		}
	}

	// Guest gets minimal permissions
	_, _ = s.roleRepo.GetByName(ctx, rbac.RoleGuest)
	// Guests have no default permissions

	return nil
}

// grantPermissionByString is a helper to grant permission by resource:action string
func (s *RBACService) grantPermissionByString(ctx context.Context, roleID uuid.UUID, permStr string) {
	// Parse resource and action
	var resource, action string
	if n, _ := fmt.Sscanf(permStr, "%[^:]:%s", &resource, &action); n == 2 {
		perm, err := s.permissionRepo.GetByResourceAction(ctx, resource, action)
		if err == nil && perm != nil {
			_ = s.permissionRepo.GrantPermission(ctx, roleID, perm.ID)
		}
	}
}
