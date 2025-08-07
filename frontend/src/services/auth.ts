import { httpClient } from './api';
import type {
  LoginRequest,
  RegisterRequest,
  LoginResponse,
  UserInfo,
} from '../types/auth';

export const authService = {
  // Login user
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    return httpClient.post<LoginResponse>('/api/auth/login', credentials);
  },

  // Register user
  async register(userData: RegisterRequest): Promise<LoginResponse> {
    return httpClient.post<LoginResponse>('/api/auth/register', userData);
  },

  // Refresh access token
  async refreshToken(refreshToken: string): Promise<LoginResponse> {
    return httpClient.post<LoginResponse>('/api/auth/refresh', { refreshToken });
  },

  // Logout user
  async logout(): Promise<{ success: boolean }> {
    return httpClient.post<{ success: boolean }>('/api/auth/logout');
  },

  // Get current user profile
  async getProfile(): Promise<{ success: boolean; data?: UserInfo; error?: unknown }> {
    return httpClient.get<{ success: boolean; data?: UserInfo; error?: unknown }>('/api/auth/profile');
  },

  // Update user profile
  async updateProfile(updates: Partial<UserInfo>): Promise<{ success: boolean; data?: UserInfo; error?: unknown }> {
    return httpClient.put<{ success: boolean; data?: UserInfo; error?: unknown }>('/api/auth/profile', updates);
  },

  // Change password
  async changePassword(data: {
    currentPassword: string;
    newPassword: string;
  }): Promise<{ success: boolean; error?: unknown }> {
    return httpClient.post<{ success: boolean; error?: unknown }>('/api/auth/change-password', data);
  },

  // Request password reset
  async requestPasswordReset(email: string): Promise<{ success: boolean; error?: unknown }> {
    return httpClient.post<{ success: boolean; error?: unknown }>('/api/auth/request-password-reset', { email });
  },

  // Reset password
  async resetPassword(data: {
    token: string;
    newPassword: string;
  }): Promise<{ success: boolean; error?: any }> {
    return httpClient.post<{ success: boolean; error?: any }>('/api/auth/reset-password', data);
  },

  // Verify email
  async verifyEmail(token: string): Promise<{ success: boolean; error?: any }> {
    return httpClient.post<{ success: boolean; error?: any }>('/api/auth/verify-email', { token });
  },

  // Resend email verification
  async resendEmailVerification(): Promise<{ success: boolean; error?: any }> {
    return httpClient.post<{ success: boolean; error?: any }>('/api/auth/resend-email-verification');
  },

  // Enable MFA
  async enableMFA(): Promise<{
    success: boolean;
    data?: {
      secret: string;
      qrCode: string;
      backupCodes: string[];
    };
    error?: any;
  }> {
    return httpClient.post<{
      success: boolean;
      data?: {
        secret: string;
        qrCode: string;
        backupCodes: string[];
      };
      error?: any;
    }>('/api/auth/mfa/enable');
  },

  // Confirm MFA setup
  async confirmMFA(data: {
    token: string;
    secret: string;
  }): Promise<{ success: boolean; error?: any }> {
    return httpClient.post<{ success: boolean; error?: any }>('/api/auth/mfa/confirm', data);
  },

  // Disable MFA
  async disableMFA(data: {
    password: string;
    token?: string;
  }): Promise<{ success: boolean; error?: any }> {
    return httpClient.post<{ success: boolean; error?: any }>('/api/auth/mfa/disable', data);
  },

  // Generate new MFA backup codes
  async generateBackupCodes(password: string): Promise<{
    success: boolean;
    data?: { backupCodes: string[] };
    error?: any;
  }> {
    return httpClient.post<{
      success: boolean;
      data?: { backupCodes: string[] };
      error?: any;
    }>('/api/auth/mfa/backup-codes', { password });
  },

  // Check auth status
  async checkAuthStatus(): Promise<{
    success: boolean;
    data?: {
      isAuthenticated: boolean;
      user?: UserInfo;
      expiresAt?: string;
    };
    error?: any;
  }> {
    return httpClient.get<{
      success: boolean;
      data?: {
        isAuthenticated: boolean;
        user?: UserInfo;
        expiresAt?: string;
      };
      error?: any;
    }>('/api/auth/status');
  },
};