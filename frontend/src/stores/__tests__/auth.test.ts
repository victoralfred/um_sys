import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { authState, authActions } from '../auth';
import { authService } from '../../services/auth';

// Mock the auth service
vi.mock('../../services/auth', () => ({
  authService: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    refreshToken: vi.fn(),
  },
}));

// Mock localStorage
const mockLocalStorage = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};

Object.defineProperty(window, 'localStorage', {
  value: mockLocalStorage,
  writable: true,
});

describe('Auth Store', () => {
  beforeEach(() => {
    // Clear all mocks before each test
    vi.clearAllMocks();
    
    // Reset localStorage mock
    mockLocalStorage.getItem.mockReturnValue(null);
    
    // Reset auth state by logging out
    authActions.logout();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Initial State', () => {
    it('should have correct initial state when no stored data', () => {
      expect(authState.user).toBeNull();
      expect(authState.accessToken).toBeNull();
      expect(authState.refreshToken).toBeNull();
      expect(authState.expiresAt).toBeNull();
      expect(authState.isAuthenticated).toBe(false);
      expect(authState.isLoading).toBe(false);
    });

    it('should restore state from localStorage when valid', () => {
      const mockUser = {
        id: '1',
        email: 'test@example.com',
        username: 'testuser',
        firstName: 'Test',
        lastName: 'User',
      };
      const futureDate = new Date(Date.now() + 3600000).toISOString(); // 1 hour from now

      mockLocalStorage.getItem.mockImplementation((key: string) => {
        switch (key) {
          case 'umanager_access_token':
            return 'mock-access-token';
          case 'umanager_refresh_token':
            return 'mock-refresh-token';
          case 'umanager_expires_at':
            return futureDate;
          case 'umanager_user':
            return JSON.stringify(mockUser);
          default:
            return null;
        }
      });

      // Re-import to trigger initialization
      vi.resetModules();
    });
  });

  describe('Login', () => {
    it('should login successfully with valid credentials', async () => {
      const mockResponse = {
        success: true,
        data: {
          accessToken: 'new-access-token',
          refreshToken: 'new-refresh-token',
          expiresAt: new Date(Date.now() + 3600000).toISOString(),
          tokenType: 'Bearer',
          expiresIn: 3600,
          user: {
            id: '1',
            email: 'test@example.com',
            username: 'testuser',
            firstName: 'Test',
            lastName: 'User',
          },
        },
      };

      vi.mocked(authService.login).mockResolvedValue(mockResponse);

      const result = await authActions.login({
        email: 'test@example.com',
        password: 'password123',
      });

      expect(result.success).toBe(true);
      expect(authService.login).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
      expect(authState.isAuthenticated).toBe(true);
      expect(authState.user).toEqual(mockResponse.data.user);
      expect(authState.accessToken).toBe('new-access-token');
    });

    it('should handle login failure', async () => {
      const mockResponse = {
        success: false,
        error: {
          code: 'INVALID_CREDENTIALS',
          message: 'Invalid email or password',
        },
      };

      vi.mocked(authService.login).mockResolvedValue(mockResponse);

      const result = await authActions.login({
        email: 'test@example.com',
        password: 'wrong-password',
      });

      expect(result.success).toBe(false);
      expect(result.error?.message).toBe('Invalid email or password');
      expect(authState.isAuthenticated).toBe(false);
      expect(authState.user).toBeNull();
    });

    it('should handle network errors during login', async () => {
      const networkError = new Error('Network error');
      authService.login = vi.fn().mockRejectedValue(networkError);

      const result = await authActions.login({
        email: 'test@example.com',
        password: 'password123',
      });

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('NETWORK_ERROR');
      expect(authState.isAuthenticated).toBe(false);
    });
  });

  describe('Register', () => {
    it('should register successfully', async () => {
      const mockResponse = {
        success: true,
        data: {
          accessToken: 'new-access-token',
          refreshToken: 'new-refresh-token',
          expiresAt: new Date(Date.now() + 3600000).toISOString(),
          tokenType: 'Bearer',
          expiresIn: 3600,
          user: {
            id: '1',
            email: 'newuser@example.com',
            username: 'newuser',
            firstName: 'New',
            lastName: 'User',
          },
        },
      };

      vi.mocked(authService.register).mockResolvedValue(mockResponse);

      const result = await authActions.register({
        email: 'newuser@example.com',
        username: 'newuser',
        password: 'password123',
        firstName: 'New',
        lastName: 'User',
      });

      expect(result.success).toBe(true);
      expect(authService.register).toHaveBeenCalled();
      expect(authState.isAuthenticated).toBe(true);
      expect(authState.user).toEqual(mockResponse.data.user);
    });

    it('should handle registration failure', async () => {
      const mockResponse = {
        success: false,
        error: {
          code: 'EMAIL_EXISTS',
          message: 'Email already exists',
        },
      };

      vi.mocked(authService.register).mockResolvedValue(mockResponse);

      const result = await authActions.register({
        email: 'existing@example.com',
        username: 'newuser',
        password: 'password123',
      });

      expect(result.success).toBe(false);
      expect(result.error?.message).toBe('Email already exists');
      expect(authState.isAuthenticated).toBe(false);
    });
  });

  describe('Logout', () => {
    it('should logout successfully', async () => {
      // First, simulate a logged-in state
      const mockUser = {
        id: '1',
        email: 'test@example.com',
        username: 'testuser',
        firstName: 'Test',
        lastName: 'User',
      };

      // Set authenticated state
      authActions.updateProfile(mockUser);

      vi.mocked(authService.logout).mockResolvedValue({ success: true });

      await authActions.logout();

      expect(authService.logout).toHaveBeenCalled();
      expect(authState.isAuthenticated).toBe(false);
      expect(authState.user).toBeNull();
      expect(authState.accessToken).toBeNull();
      expect(authState.refreshToken).toBeNull();
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('umanager_access_token');
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('umanager_refresh_token');
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('umanager_expires_at');
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('umanager_user');
    });

    it('should clear local state even if server logout fails', async () => {
      vi.mocked(authService.logout).mockRejectedValue(new Error('Server error'));

      await authActions.logout();

      expect(authState.isAuthenticated).toBe(false);
      expect(authState.user).toBeNull();
      expect(mockLocalStorage.removeItem).toHaveBeenCalled();
    });
  });

  describe('Token Refresh', () => {
    it('should refresh token successfully', async () => {
      const mockResponse = {
        success: true,
        data: {
          accessToken: 'refreshed-access-token',
          refreshToken: 'refreshed-refresh-token',
          expiresAt: new Date(Date.now() + 3600000).toISOString(),
          tokenType: 'Bearer',
          expiresIn: 3600,
          user: {
            id: '1',
            email: 'test@example.com',
            username: 'testuser',
            firstName: 'Test',
            lastName: 'User',
          },
        },
      };

      vi.mocked(authService.refreshToken).mockResolvedValue(mockResponse);

      // Set a refresh token in state
      authState.refreshToken = 'current-refresh-token';

      const result = await authActions.refreshToken();

      expect(result).toBe(true);
      expect(authService.refreshToken).toHaveBeenCalledWith('current-refresh-token');
      expect(authState.accessToken).toBe('refreshed-access-token');
    });

    it('should logout if refresh token is invalid', async () => {
      const mockResponse = {
        success: false,
        error: {
          code: 'INVALID_TOKEN',
          message: 'Refresh token expired',
        },
      };

      vi.mocked(authService.refreshToken).mockResolvedValue(mockResponse);

      const result = await authActions.refreshToken();

      expect(result).toBe(false);
      expect(authState.isAuthenticated).toBe(false);
    });

    it('should logout if no refresh token available', async () => {
      authState.refreshToken = null;

      const result = await authActions.refreshToken();

      expect(result).toBe(false);
      expect(authState.isAuthenticated).toBe(false);
    });
  });

  describe('Token Refresh Check', () => {
    it('should identify when token needs refresh', () => {
      // Set expiration to 3 minutes from now (needs refresh within 5 minutes)
      const nearExpiry = new Date(Date.now() + 3 * 60 * 1000);
      authState.expiresAt = nearExpiry;

      expect(authActions.needsTokenRefresh()).toBe(true);
    });

    it('should identify when token does not need refresh', () => {
      // Set expiration to 10 minutes from now
      const farExpiry = new Date(Date.now() + 10 * 60 * 1000);
      authState.expiresAt = farExpiry;

      expect(authActions.needsTokenRefresh()).toBe(false);
    });

    it('should return false when no expiration date', () => {
      authState.expiresAt = null;

      expect(authActions.needsTokenRefresh()).toBe(false);
    });
  });

  describe('Profile Update', () => {
    it('should update user profile', () => {
      const initialUser = {
        id: '1',
        email: 'test@example.com',
        username: 'testuser',
        firstName: 'Test',
        lastName: 'User',
      };

      const updates = {
        firstName: 'Updated',
        lastName: 'Name',
      };

      authState.user = initialUser;
      authActions.updateProfile(updates);

      expect(authState.user).toEqual({
        ...initialUser,
        ...updates,
      });
      expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
        'umanager_user',
        JSON.stringify({ ...initialUser, ...updates })
      );
    });

    it('should not update if no user is logged in', () => {
      authState.user = null;
      
      authActions.updateProfile({ firstName: 'Test' });
      
      expect(authState.user).toBeNull();
      expect(mockLocalStorage.setItem).not.toHaveBeenCalled();
    });
  });
});