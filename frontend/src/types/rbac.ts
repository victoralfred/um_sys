// RBAC types based on backend domain models

export interface Role {
  id: string;
  name: string;
  description: string;
  isSystem: boolean;
  priority: number;
  createdAt: string; // ISO date string
  updatedAt: string; // ISO date string
  deletedAt?: string; // ISO date string
}

export interface Permission {
  id: string;
  resource: string; // e.g., "users", "posts", "billing"
  action: string; // e.g., "create", "read", "update", "delete"
  description: string;
  createdAt: string; // ISO date string
  updatedAt: string; // ISO date string
}

export interface UserRole {
  userId: string;
  roleId: string;
  grantedBy: string;
  grantedAt: string; // ISO date string
  expiresAt?: string; // ISO date string
}

export type PolicyEffect = 'allow' | 'deny';

export interface PolicyRule {
  id: string;
  name: string;
  description: string;
  resource: string;
  action: string;
  effect: PolicyEffect;
  conditions: Record<string, any>;
  priority: number;
  createdAt: string; // ISO date string
  updatedAt: string; // ISO date string
}

export interface AccessRequest {
  userId: string;
  resource: string;
  action: string;
  context?: Record<string, any>;
}

export interface AccessResponse {
  allowed: boolean;
  reason?: string;
  matchedRules?: string[];
}

// Predefined system roles
export const SystemRoles = {
  SUPER_ADMIN: 'super_admin',
  ADMIN: 'admin',
  MODERATOR: 'moderator',
  USER: 'user',
  GUEST: 'guest',
} as const;

// Common permissions
export const Permissions = {
  // User permissions
  USERS_CREATE: 'users:create',
  USERS_READ: 'users:read',
  USERS_UPDATE: 'users:update',
  USERS_DELETE: 'users:delete',
  USERS_LIST: 'users:list',
  
  // Role permissions
  ROLES_CREATE: 'roles:create',
  ROLES_READ: 'roles:read',
  ROLES_UPDATE: 'roles:update',
  ROLES_DELETE: 'roles:delete',
  ROLES_LIST: 'roles:list',
  ROLES_ASSIGN: 'roles:assign',
  
  // Billing permissions
  BILLING_VIEW: 'billing:view',
  BILLING_MANAGE: 'billing:manage',
  
  // System permissions
  SYSTEM_MANAGE: 'system:manage',
  SYSTEM_AUDIT: 'system:audit',
} as const;