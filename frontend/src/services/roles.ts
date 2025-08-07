import { httpClient } from './api';
import type { Role, Permission, PolicyRule } from '../types/rbac';

export const rolesService = {
  // Get all roles
  async getRoles(filters?: {
    search?: string;
    isSystem?: boolean;
    limit?: number;
    offset?: number;
  }): Promise<{
    success: boolean;
    data?: {
      roles: Role[];
      total: number;
      page: number;
      pageSize: number;
      totalPages: number;
    };
    error?: any;
  }> {
    return httpClient.get<{
      success: boolean;
      data?: {
        roles: Role[];
        total: number;
        page: number;
        pageSize: number;
        totalPages: number;
      };
      error?: any;
    }>('/api/roles', { params: filters });
  },

  // Get a single role
  async getRole(roleId: string): Promise<{
    success: boolean;
    data?: Role;
    error?: any;
  }> {
    return httpClient.get<{
      success: boolean;
      data?: Role;
      error?: any;
    }>(`/api/roles/${roleId}`);
  },

  // Create a new role
  async createRole(roleData: {
    name: string;
    description: string;
    priority?: number;
  }): Promise<{
    success: boolean;
    data?: Role;
    error?: any;
  }> {
    return httpClient.post<{
      success: boolean;
      data?: Role;
      error?: any;
    }>('/api/roles', roleData);
  },

  // Update a role
  async updateRole(roleId: string, updates: {
    name?: string;
    description?: string;
    priority?: number;
  }): Promise<{
    success: boolean;
    data?: Role;
    error?: any;
  }> {
    return httpClient.put<{
      success: boolean;
      data?: Role;
      error?: any;
    }>(`/api/roles/${roleId}`, updates);
  },

  // Delete a role
  async deleteRole(roleId: string): Promise<{
    success: boolean;
    error?: any;
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: any;
    }>(`/api/roles/${roleId}`);
  },

  // Get role permissions
  async getRolePermissions(roleId: string): Promise<{
    success: boolean;
    data?: Permission[];
    error?: any;
  }> {
    return httpClient.get<{
      success: boolean;
      data?: Permission[];
      error?: any;
    }>(`/api/roles/${roleId}/permissions`);
  },

  // Assign permission to role
  async assignPermission(roleId: string, permissionId: string): Promise<{
    success: boolean;
    error?: any;
  }> {
    return httpClient.post<{
      success: boolean;
      error?: any;
    }>(`/api/roles/${roleId}/permissions`, { permissionId });
  },

  // Remove permission from role
  async removePermission(roleId: string, permissionId: string): Promise<{
    success: boolean;
    error?: any;
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: any;
    }>(`/api/roles/${roleId}/permissions/${permissionId}`);
  },

  // Get all permissions
  async getPermissions(filters?: {
    resource?: string;
    action?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{
    success: boolean;
    data?: {
      permissions: Permission[];
      total: number;
      page: number;
      pageSize: number;
      totalPages: number;
    };
    error?: any;
  }> {
    return httpClient.get<{
      success: boolean;
      data?: {
        permissions: Permission[];
        total: number;
        page: number;
        pageSize: number;
        totalPages: number;
      };
      error?: any;
    }>('/api/permissions', { params: filters });
  },

  // Create a new permission
  async createPermission(permissionData: {
    resource: string;
    action: string;
    description: string;
  }): Promise<{
    success: boolean;
    data?: Permission;
    error?: any;
  }> {
    return httpClient.post<{
      success: boolean;
      data?: Permission;
      error?: any;
    }>('/api/permissions', permissionData);
  },

  // Update a permission
  async updatePermission(permissionId: string, updates: {
    resource?: string;
    action?: string;
    description?: string;
  }): Promise<{
    success: boolean;
    data?: Permission;
    error?: any;
  }> {
    return httpClient.put<{
      success: boolean;
      data?: Permission;
      error?: any;
    }>(`/api/permissions/${permissionId}`, updates);
  },

  // Delete a permission
  async deletePermission(permissionId: string): Promise<{
    success: boolean;
    error?: any;
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: any;
    }>(`/api/permissions/${permissionId}`);
  },

  // Check user access
  async checkAccess(data: {
    userId: string;
    resource: string;
    action: string;
    context?: Record<string, any>;
  }): Promise<{
    success: boolean;
    data?: {
      allowed: boolean;
      reason?: string;
      matchedRules?: string[];
    };
    error?: any;
  }> {
    return httpClient.post<{
      success: boolean;
      data?: {
        allowed: boolean;
        reason?: string;
        matchedRules?: string[];
      };
      error?: any;
    }>('/api/rbac/check-access', data);
  },

  // Get policy rules
  async getPolicyRules(filters?: {
    resource?: string;
    action?: string;
    effect?: 'allow' | 'deny';
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{
    success: boolean;
    data?: {
      rules: PolicyRule[];
      total: number;
      page: number;
      pageSize: number;
      totalPages: number;
    };
    error?: any;
  }> {
    return httpClient.get<{
      success: boolean;
      data?: {
        rules: PolicyRule[];
        total: number;
        page: number;
        pageSize: number;
        totalPages: number;
      };
      error?: any;
    }>('/api/policy-rules', { params: filters });
  },

  // Create a policy rule
  async createPolicyRule(ruleData: {
    name: string;
    description: string;
    resource: string;
    action: string;
    effect: 'allow' | 'deny';
    conditions?: Record<string, any>;
    priority?: number;
  }): Promise<{
    success: boolean;
    data?: PolicyRule;
    error?: any;
  }> {
    return httpClient.post<{
      success: boolean;
      data?: PolicyRule;
      error?: any;
    }>('/api/policy-rules', ruleData);
  },

  // Update a policy rule
  async updatePolicyRule(ruleId: string, updates: {
    name?: string;
    description?: string;
    resource?: string;
    action?: string;
    effect?: 'allow' | 'deny';
    conditions?: Record<string, any>;
    priority?: number;
  }): Promise<{
    success: boolean;
    data?: PolicyRule;
    error?: any;
  }> {
    return httpClient.put<{
      success: boolean;
      data?: PolicyRule;
      error?: any;
    }>(`/api/policy-rules/${ruleId}`, updates);
  },

  // Delete a policy rule
  async deletePolicyRule(ruleId: string): Promise<{
    success: boolean;
    error?: any;
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: any;
    }>(`/api/policy-rules/${ruleId}`);
  },
};