package rbac

import "errors"

var (
	// ErrRoleNotFound is returned when a role is not found
	ErrRoleNotFound = errors.New("role not found")

	// ErrPermissionNotFound is returned when a permission is not found
	ErrPermissionNotFound = errors.New("permission not found")

	// ErrPolicyNotFound is returned when a policy is not found
	ErrPolicyNotFound = errors.New("policy not found")

	// ErrRoleAlreadyExists is returned when trying to create a duplicate role
	ErrRoleAlreadyExists = errors.New("role already exists")

	// ErrPermissionAlreadyExists is returned when trying to create a duplicate permission
	ErrPermissionAlreadyExists = errors.New("permission already exists")

	// ErrRoleAlreadyAssigned is returned when trying to assign an already assigned role
	ErrRoleAlreadyAssigned = errors.New("role already assigned to user")

	// ErrPermissionAlreadyGranted is returned when trying to grant an already granted permission
	ErrPermissionAlreadyGranted = errors.New("permission already granted to role")

	// ErrSystemRoleModification is returned when trying to modify a system role
	ErrSystemRoleModification = errors.New("system roles cannot be modified")

	// ErrSystemRoleDeletion is returned when trying to delete a system role
	ErrSystemRoleDeletion = errors.New("system roles cannot be deleted")

	// ErrInvalidResource is returned when resource is invalid
	ErrInvalidResource = errors.New("invalid resource")

	// ErrInvalidAction is returned when action is invalid
	ErrInvalidAction = errors.New("invalid action")

	// ErrAccessDenied is returned when access is denied
	ErrAccessDenied = errors.New("access denied")

	// ErrInsufficientPermissions is returned when user lacks required permissions
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// ErrRoleExpired is returned when a role assignment has expired
	ErrRoleExpired = errors.New("role assignment has expired")

	// ErrCircularRoleHierarchy is returned when role hierarchy would create a cycle
	ErrCircularRoleHierarchy = errors.New("circular role hierarchy detected")
)
