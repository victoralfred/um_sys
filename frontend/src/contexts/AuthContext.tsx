import { createContext, useContext, ParentComponent } from 'solid-js';
import { authState, authActions } from '../stores/auth';
import type { AuthState, LoginRequest, RegisterRequest } from '../types/auth';

// Create the auth context
interface AuthContextValue {
  // State
  state: AuthState;
  
  // Actions
  login: (credentials: LoginRequest) => Promise<any>;
  register: (userData: RegisterRequest) => Promise<any>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<boolean>;
  updateProfile: (updates: any) => void;
}

const AuthContext = createContext<AuthContextValue>();

// Auth provider component
export const AuthProvider: ParentComponent = (props) => {
  const contextValue: AuthContextValue = {
    state: authState,
    login: authActions.login,
    register: authActions.register,
    logout: authActions.logout,
    refreshToken: authActions.refreshToken,
    updateProfile: authActions.updateProfile,
  };

  return (
    <AuthContext.Provider value={contextValue}>
      {props.children}
    </AuthContext.Provider>
  );
};

// Hook to use auth context
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

// Auth guard component
interface AuthGuardProps {
  children: any;
  fallback?: any;
  requireAuth?: boolean;
}

export const AuthGuard: ParentComponent<AuthGuardProps> = (props) => {
  const { state } = useAuth();
  const requireAuth = props.requireAuth !== false; // Default to true

  if (requireAuth && !state.isAuthenticated) {
    return props.fallback || <div>Please log in to access this page.</div>;
  }

  if (!requireAuth && state.isAuthenticated) {
    return props.fallback || <div>You are already logged in.</div>;
  }

  return <>{props.children}</>;
};