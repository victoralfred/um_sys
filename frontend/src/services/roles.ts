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
    error?: { code: string; message: string; };
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
      error?: { code: string; message: string; };
    }>('/api/roles', { params: filters });
  },

  // Get a single role
  async getRole(roleId: string): Promise<{
    success: boolean;
    data?: Role;
    error?: { code: string; message: string; };
  }> {
    return httpClient.get<{
      success: boolean;
      data?: Role;
      error?: { code: string; message: string; };
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
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      data?: Role;
      error?: { code: string; message: string; };
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
    error?: { code: string; message: string; };
  }> {
    return httpClient.put<{
      success: boolean;
      data?: Role;
      error?: { code: string; message: string; };
    }>(`/api/roles/${roleId}`, updates);
  },

  // Delete a role
  async deleteRole(roleId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/roles/${roleId}`);
  },

  // Get role permissions
  async getRolePermissions(roleId: string): Promise<{
    success: boolean;
    data?: Permission[];
    error?: { code: string; message: string; };
  }> {
    return httpClient.get<{
      success: boolean;
      data?: Permission[];
      error?: { code: string; message: string; };
    }>(`/api/roles/${roleId}/permissions`);
  },

  // Assign permission to role
  async assignPermission(roleId: string, permissionId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/roles/${roleId}/permissions`, { permissionId });
  },

  // Remove permission from role
  async removePermission(roleId: string, permissionId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: { code: string; message: string; };
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
    error?: { code: string; message: string; };
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
      error?: { code: string; message: string; };
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
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      data?: Permission;
      error?: { code: string; message: string; };
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
    error?: { code: string; message: string; };
  }> {
    return httpClient.put<{
      success: boolean;
      data?: Permission;
      error?: { code: string; message: string; };
    }>(`/api/permissions/${permissionId}`, updates);
  },

  // Delete a permission
  async deletePermission(permissionId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/permissions/${permissionId}`);
  },

  // Check user access
  async checkAccess(data: {
    userId: string;
    resource: string;
    action: string;
    context?: Record<string, unknown>;
  }): Promise<{
    success: boolean;
    data?: {
      allowed: boolean;
      reason?: string;
      matchedRules?: string[];
    };
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      data?: {
        allowed: boolean;
        reason?: string;
        matchedRules?: string[];
      };
      error?: { code: string; message: string; };
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
    error?: { code: string; message: string; };
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
      error?: { code: string; message: string; };
    }>('/api/policy-rules', { params: filters });
  },

  // Create a policy rule
  async createPolicyRule(ruleData: {
    name: string;
    description: string;
    resource: string;
    action: string;
    effect: 'allow' | 'deny';
    conditions?: Record<string, unknown>;
    priority?: number;
  }): Promise<{
    success: boolean;
    data?: PolicyRule;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      data?: PolicyRule;
      error?: { code: string; message: string; };
    }>('/api/policy-rules', ruleData);
  },

  // Update a policy rule
  async updatePolicyRule(ruleId: string, updates: {
    name?: string;
    description?: string;
    resource?: string;
    action?: string;
    effect?: 'allow' | 'deny';
    conditions?: Record<string, unknown>;
    priority?: number;
  }): Promise<{
    success: boolean;
    data?: PolicyRule;
    error?: { code: string; message: string; };
  }> {
    return httpClient.put<{
      success: boolean;
      data?: PolicyRule;
      error?: { code: string; message: string; };
    }>(`/api/policy-rules/${ruleId}`, updates);
  },

  // Delete a policy rule
  async deletePolicyRule(ruleId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/policy-rules/${ruleId}`);
  },
};