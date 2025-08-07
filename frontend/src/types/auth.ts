// Authentication types based on backend API responses

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  username: string;
  password: string;
  firstName?: string;
  lastName?: string;
}

export interface UserInfo {
  id: string;
  email: string;
  username: string;
  firstName: string;
  lastName: string;
}

export interface LoginResponseData {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  expiresAt: string; // ISO date string
  user: UserInfo;
}

export interface LoginResponse {
  success: boolean;
  data?: LoginResponseData;
  error?: {
    code: string;
    message: string;
  };
}

export interface RegisterResponse {
  success: boolean;
  data?: {
    user: UserInfo;
    message: string;
  };
  error?: {
    code: string;
    message: string;
  };
}

export interface ProfileUpdateRequest {
  firstName?: string;
  lastName?: string;
  email?: string;
  username?: string;
}

export interface AuthError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface AuthServiceResponse {
  success: boolean;
  error?: AuthError;
}

export interface MFASetupResponse {
  success: boolean;
  data?: {
    secret: string;
    qrCode: string;
    backupCodes: string[];
  };
  error?: AuthError;
}

export interface MFAVerificationResponse {
  success: boolean;
  data?: {
    backupCodes: string[];
  };
  error?: AuthError;
}

export interface AuthStatusResponse {
  success: boolean;
  data?: {
    isAuthenticated: boolean;
    user?: UserInfo;
    expiresAt?: string;
  };
  error?: AuthError;
}

export interface AuthState {
  user: UserInfo | null;
  accessToken: string | null;
  refreshToken: string | null;
  expiresAt: Date | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}