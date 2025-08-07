// User types based on backend domain models

export type UserStatus = 'active' | 'inactive' | 'suspended' | 'locked' | 'deleted';

export interface User {
  id: string;
  email: string;
  username: string;
  firstName: string;
  lastName: string;
  phoneNumber?: string;
  status: UserStatus;
  emailVerified: boolean;
  emailVerifiedAt?: string; // ISO date string
  phoneVerified: boolean;
  mfaEnabled: boolean;
  profilePictureUrl?: string;
  bio?: string;
  locale?: string;
  timezone?: string;
  passwordChangedAt?: string; // ISO date string
  lastLoginAt?: string; // ISO date string
  lockedUntil?: string; // ISO date string
  deletedAt?: string; // ISO date string
  createdAt: string; // ISO date string
  updatedAt: string; // ISO date string
}

export interface UserListFilter {
  search?: string;
  emailVerified?: boolean;
  mfaEnabled?: boolean;
  status?: UserStatus;
  limit?: number;
  offset?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface UserListResponse {
  success: boolean;
  data?: {
    users: User[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  };
  error?: {
    code: string;
    message: string;
  };
}

export interface CreateUserRequest {
  email: string;
  username: string;
  password: string;
  firstName?: string;
  lastName?: string;
  phoneNumber?: string;
}

export interface UpdateUserRequest {
  email?: string;
  username?: string;
  firstName?: string;
  lastName?: string;
  phoneNumber?: string;
  bio?: string;
  locale?: string;
  timezone?: string;
}