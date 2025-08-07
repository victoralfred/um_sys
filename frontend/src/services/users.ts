import { httpClient } from './api';
import type {
  User,
  UserListFilter,
  UserListResponse,
  CreateUserRequest,
  UpdateUserRequest,
} from '../types/user';

export const usersService = {
  // Get users list with filtering and pagination
  async getUsers(filters: UserListFilter = {}): Promise<UserListResponse> {
    return httpClient.get<UserListResponse>('/api/users', { 
      params: filters as Record<string, string | number | boolean> 
    });
  },

  // Get a single user by ID
  async getUser(userId: string): Promise<{
    success: boolean;
    data?: User;
    error?: { code: string; message: string; };
  }> {
    return httpClient.get<{
      success: boolean;
      data?: User;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}`);
  },

  // Create a new user
  async createUser(userData: CreateUserRequest): Promise<{
    success: boolean;
    data?: User;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      data?: User;
      error?: { code: string; message: string; };
    }>('/api/users', userData);
  },

  // Update a user
  async updateUser(userId: string, updates: UpdateUserRequest): Promise<{
    success: boolean;
    data?: User;
    error?: { code: string; message: string; };
  }> {
    return httpClient.put<{
      success: boolean;
      data?: User;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}`, updates);
  },

  // Delete a user
  async deleteUser(userId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}`);
  },

  // Bulk delete users
  async bulkDeleteUsers(userIds: string[]): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      error?: { code: string; message: string; };
    }>('/api/users/bulk-delete', { userIds });
  },

  // Update user status
  async updateUserStatus(userId: string, status: 'active' | 'inactive' | 'suspended' | 'locked'): Promise<{
    success: boolean;
    data?: User;
    error?: { code: string; message: string; };
  }> {
    return httpClient.patch<{
      success: boolean;
      data?: User;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/status`, { status });
  },

  // Reset user password (admin action)
  async resetUserPassword(userId: string): Promise<{
    success: boolean;
    data?: { temporaryPassword: string };
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      data?: { temporaryPassword: string };
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/reset-password`);
  },

  // Unlock user account
  async unlockUser(userId: string): Promise<{
    success: boolean;
    data?: User;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      data?: User;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/unlock`);
  },

  // Send email verification to user
  async sendEmailVerification(userId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/send-email-verification`);
  },

  // Get user roles
  async getUserRoles(userId: string): Promise<{
    success: boolean;
    data?: Array<{
      id: string;
      name: string;
      description: string;
      grantedAt: string;
      expiresAt?: string;
    }>;
    error?: { code: string; message: string; };
  }> {
    return httpClient.get<{
      success: boolean;
      data?: Array<{
        id: string;
        name: string;
        description: string;
        grantedAt: string;
        expiresAt?: string;
      }>;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/roles`);
  },

  // Assign role to user
  async assignRole(userId: string, roleId: string, expiresAt?: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/roles`, { roleId, expiresAt });
  },

  // Remove role from user
  async removeRole(userId: string, roleId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/roles/${roleId}`);
  },

  // Get user permissions
  async getUserPermissions(userId: string): Promise<{
    success: boolean;
    data?: Array<{
      resource: string;
      action: string;
      description: string;
    }>;
    error?: { code: string; message: string; };
  }> {
    return httpClient.get<{
      success: boolean;
      data?: Array<{
        resource: string;
        action: string;
        description: string;
      }>;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/permissions`);
  },

  // Export users
  async exportUsers(
    format: 'csv' | 'excel' = 'csv',
    filters: UserListFilter = {}
  ): Promise<void> {
    return httpClient.download(
      `/api/users/export?format=${format}`,
      `users.${format}`,
      { params: filters as Record<string, string | number | boolean> }
    );
  },

  // Import users from file
  async importUsers(file: File): Promise<{
    success: boolean;
    data?: {
      imported: number;
      failed: number;
      errors?: Array<{
        row: number;
        error: string;
      }>;
    };
    error?: { code: string; message: string; };
  }> {
    return httpClient.upload<{
      success: boolean;
      data?: {
        imported: number;
        failed: number;
        errors?: Array<{
          row: number;
          error: string;
        }>;
      };
      error?: { code: string; message: string; };
    }>('/api/users/import', file);
  },

  // Get user activity log
  async getUserActivity(userId: string, filters?: {
    startDate?: string;
    endDate?: string;
    action?: string;
    limit?: number;
    offset?: number;
  }): Promise<{
    success: boolean;
    data?: {
      activities: Array<{
        id: string;
        action: string;
        resource: string;
        details: Record<string, unknown>;
        ipAddress: string;
        userAgent: string;
        createdAt: string;
      }>;
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
        activities: Array<{
          id: string;
          action: string;
          resource: string;
          details: Record<string, unknown>;
          ipAddress: string;
          userAgent: string;
          createdAt: string;
        }>;
        total: number;
        page: number;
        pageSize: number;
        totalPages: number;
      };
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/activity`, { params: filters });
  },

  // Get user sessions
  async getUserSessions(userId: string): Promise<{
    success: boolean;
    data?: Array<{
      id: string;
      deviceName: string;
      ipAddress: string;
      location?: string;
      isCurrentSession: boolean;
      lastActivity: string;
      createdAt: string;
    }>;
    error?: { code: string; message: string; };
  }> {
    return httpClient.get<{
      success: boolean;
      data?: Array<{
        id: string;
        deviceName: string;
        ipAddress: string;
        location?: string;
        isCurrentSession: boolean;
        lastActivity: string;
        createdAt: string;
      }>;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/sessions`);
  },

  // Revoke user session
  async revokeUserSession(userId: string, sessionId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.delete<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/sessions/${sessionId}`);
  },

  // Revoke all user sessions except current
  async revokeAllUserSessions(userId: string): Promise<{
    success: boolean;
    error?: { code: string; message: string; };
  }> {
    return httpClient.post<{
      success: boolean;
      error?: { code: string; message: string; };
    }>(`/api/users/${userId}/sessions/revoke-all`);
  },
};