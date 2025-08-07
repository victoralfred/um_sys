import { createStore } from 'solid-js/store';
import { createSignal } from 'solid-js';
import { authService } from '../services/auth';
import type {
  AuthState,
  LoginRequest,
  RegisterRequest,
  LoginResponse,
  UserInfo,
} from '../types/auth';

const STORAGE_KEYS = {
  ACCESS_TOKEN: 'umanager_access_token',
  REFRESH_TOKEN: 'umanager_refresh_token',
  EXPIRES_AT: 'umanager_expires_at',
  USER: 'umanager_user',
} as const;

// Initialize state from localStorage if available
const getInitialState = (): AuthState => {
  if (typeof window === 'undefined') {
    return {
      user: null,
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      isAuthenticated: false,
      isLoading: false,
    };
  }

  const accessToken = window.localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
  const refreshToken = window.localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
  const expiresAtStr = window.localStorage.getItem(STORAGE_KEYS.EXPIRES_AT);
  const userStr = window.localStorage.getItem(STORAGE_KEYS.USER);

  const expiresAt = expiresAtStr ? new Date(expiresAtStr) : null;
  const user = userStr ? JSON.parse(userStr) : null;

  // Check if token is expired
  const isTokenExpired = expiresAt ? new Date() >= expiresAt : true;

  if (isTokenExpired || !accessToken || !user) {
    // Clear invalid tokens
    clearStoredAuth();
    return {
      user: null,
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      isAuthenticated: false,
      isLoading: false,
    };
  }

  return {
    user,
    accessToken,
    refreshToken,
    expiresAt,
    isAuthenticated: true,
    isLoading: false,
  };
};

const clearStoredAuth = () => {
  if (typeof window === 'undefined') return;
  
  window.localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
  window.localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
  window.localStorage.removeItem(STORAGE_KEYS.EXPIRES_AT);
  window.localStorage.removeItem(STORAGE_KEYS.USER);
};

const storeAuth = (accessToken: string, refreshToken: string, expiresAt: Date, user: UserInfo) => {
  if (typeof window === 'undefined') return;
  
  window.localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, accessToken);
  window.localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, refreshToken);
  window.localStorage.setItem(STORAGE_KEYS.EXPIRES_AT, expiresAt.toISOString());
  window.localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(user));
};

// Create the auth store
const [authState, setAuthState] = createStore<AuthState>(getInitialState());

// Loading signal for async operations
const [authLoading, setAuthLoading] = createSignal(false);

// Auth actions
const authActions = {
  // Set loading state
  setLoading: (loading: boolean) => {
    setAuthLoading(loading);
    setAuthState('isLoading', loading);
  },

  // Login action
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    authActions.setLoading(true);
    
    try {
      const data = await authService.login(credentials);

      if (data.success && data.data) {
        const { accessToken, refreshToken, expiresAt, user } = data.data;
        const expiresAtDate = new Date(expiresAt);

        // Store auth data
        storeAuth(accessToken, refreshToken, expiresAtDate, user);

        // Update state
        setAuthState({
          user,
          accessToken,
          refreshToken,
          expiresAt: expiresAtDate,
          isAuthenticated: true,
          isLoading: false,
        });
      }

      return data;
    } catch (error) {
      console.error('Login error:', error);
      return {
        success: false,
        error: {
          code: 'NETWORK_ERROR',
          message: error instanceof Error ? error.message : 'Failed to connect to server',
        },
      };
    } finally {
      authActions.setLoading(false);
    }
  },

  // Register action
  register: async (userData: RegisterRequest): Promise<LoginResponse> => {
    authActions.setLoading(true);
    
    try {
      const data = await authService.register(userData);

      if (data.success && data.data) {
        const { accessToken, refreshToken, expiresAt, user } = data.data;
        const expiresAtDate = new Date(expiresAt);

        // Store auth data
        storeAuth(accessToken, refreshToken, expiresAtDate, user);

        // Update state
        setAuthState({
          user,
          accessToken,
          refreshToken,
          expiresAt: expiresAtDate,
          isAuthenticated: true,
          isLoading: false,
        });
      }

      return data;
    } catch (error) {
      console.error('Registration error:', error);
      return {
        success: false,
        error: {
          code: 'NETWORK_ERROR',
          message: error instanceof Error ? error.message : 'Failed to connect to server',
        },
      };
    } finally {
      authActions.setLoading(false);
    }
  },

  // Logout action
  logout: async () => {
    authActions.setLoading(true);
    
    try {
      if (authState.accessToken) {
        await authService.logout();
      }
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      // Clear local state regardless of server response
      clearStoredAuth();
      setAuthState({
        user: null,
        accessToken: null,
        refreshToken: null,
        expiresAt: null,
        isAuthenticated: false,
        isLoading: false,
      });
      authActions.setLoading(false);
    }
  },

  // Refresh token action
  refreshToken: async (): Promise<boolean> => {
    if (!authState.refreshToken) {
      authActions.logout();
      return false;
    }

    try {
      const data = await authService.refreshToken(authState.refreshToken);

      if (data.success && data.data) {
        const { accessToken, refreshToken, expiresAt, user } = data.data;
        const expiresAtDate = new Date(expiresAt);

        // Store new auth data
        storeAuth(accessToken, refreshToken, expiresAtDate, user);

        // Update state
        setAuthState({
          user,
          accessToken,
          refreshToken,
          expiresAt: expiresAtDate,
          isAuthenticated: true,
        });

        return true;
      }
      
      // Refresh failed, logout user
      authActions.logout();
      return false;
    } catch (error) {
      console.error('Token refresh error:', error);
      authActions.logout();
      return false;
    }
  },

  // Check if token needs refresh (within 5 minutes of expiry)
  needsTokenRefresh: (): boolean => {
    if (!authState.expiresAt) return false;
    const now = new Date();
    const fiveMinutesFromNow = new Date(now.getTime() + 5 * 60 * 1000);
    return authState.expiresAt <= fiveMinutesFromNow;
  },

  // Update user profile
  updateProfile: (updates: Partial<UserInfo>) => {
    if (!authState.user) return;

    const updatedUser = { ...authState.user, ...updates };
    setAuthState('user', updatedUser);
    
    // Update localStorage
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(updatedUser));
    }
  },
};

// Auto-refresh token when needed
if (typeof window !== 'undefined') {
  window.setInterval(() => {
    if (authState.isAuthenticated && authActions.needsTokenRefresh()) {
      authActions.refreshToken();
    }
  }, 60000); // Check every minute
}

export {
  authState,
  authLoading,
  authActions,
};