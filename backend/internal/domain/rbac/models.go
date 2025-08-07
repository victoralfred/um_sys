package rbac

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a role in the system
type Role struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsSystem    bool       `json:"is_system"` // System roles cannot be deleted
	Priority    int        `json:"priority"`  // Higher priority = more permissions
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// Permission represents a permission in the system
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Resource    string    `json:"resource"` // e.g., "users", "posts", "billing"
	Action      string    `json:"action"`   // e.g., "create", "read", "update", "delete"
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserRole represents the relationship between users and roles
type UserRole struct {
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	GrantedBy uuid.UUID  `json:"granted_by"`
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// RolePermission represents the relationship between roles and permissions
type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id"`
	PermissionID uuid.UUID `json:"permission_id"`
	GrantedAt    time.Time `json:"granted_at"`
}

// PolicyRule represents a fine-grained access control rule
type PolicyRule struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Effect      PolicyEffect           `json:"effect"`     // Allow or Deny
	Conditions  map[string]interface{} `json:"conditions"` // JSON conditions
	Priority    int                    `json:"priority"`   // For conflict resolution
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PolicyEffect represents the effect of a policy rule
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

// AccessRequest represents a request to check access
type AccessRequest struct {
	UserID   uuid.UUID              `json:"user_id"`
	Resource string                 `json:"resource"`
	Action   string                 `json:"action"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// AccessResponse represents the response to an access check
type AccessResponse struct {
	Allowed      bool     `json:"allowed"`
	Reason       string   `json:"reason,omitempty"`
	MatchedRules []string `json:"matched_rules,omitempty"`
}

// Predefined system roles
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleModerator  = "moderator"
	RoleUser       = "user"
	RoleGuest      = "guest"
)

// Common permissions
const (
	// User permissions
	PermissionUsersCreate = "users:create"
	PermissionUsersRead   = "users:read"
	PermissionUsersUpdate = "users:update"
	PermissionUsersDelete = "users:delete"
	PermissionUsersList   = "users:list"

	// Role permissions
	PermissionRolesCreate = "roles:create"
	PermissionRolesRead   = "roles:read"
	PermissionRolesUpdate = "roles:update"
	PermissionRolesDelete = "roles:delete"
	PermissionRolesList   = "roles:list"
	PermissionRolesAssign = "roles:assign"

	// Billing permissions
	PermissionBillingView   = "billing:view"
	PermissionBillingManage = "billing:manage"

	// System permissions
	PermissionSystemManage = "system:manage"
	PermissionSystemAudit  = "system:audit"
)
