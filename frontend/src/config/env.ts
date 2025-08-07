// Environment configuration utilities

interface EnvConfig {
  // API Configuration
  apiBaseUrl: string;
  apiTimeout: number;

  // App Configuration
  appName: string;
  appVersion: string;
  appEnvironment: string;

  // Feature Flags
  enableDebugMode: boolean;
  enableAnalytics: boolean;
  enableMFA: boolean;
  enableBulkOperations: boolean;

  // UI Configuration
  defaultPageSize: number;
  maxPageSize: number;
  theme: string;

  // Security Configuration
  sessionTimeout: number;
  tokenRefreshThreshold: number;
  maxLoginAttempts: number;

  // Development Configuration
  mockApi: boolean;
  logLevel: string;
}

// Helper function to parse boolean environment variables
const parseBoolean = (value: string | undefined, defaultValue: boolean = false): boolean => {
  if (!value) return defaultValue;
  return value.toLowerCase() === 'true';
};

// Helper function to parse number environment variables
const parseNumber = (value: string | undefined, defaultValue: number): number => {
  if (!value) return defaultValue;
  const parsed = parseInt(value, 10);
  return isNaN(parsed) ? defaultValue : parsed;
};

// Create configuration object from environment variables
const createEnvConfig = (): EnvConfig => ({
  // API Configuration
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  apiTimeout: parseNumber(import.meta.env.VITE_API_TIMEOUT, 30000),

  // App Configuration
  appName: import.meta.env.VITE_APP_NAME || 'UManager',
  appVersion: import.meta.env.VITE_APP_VERSION || '1.0.0',
  appEnvironment: import.meta.env.VITE_APP_ENVIRONMENT || 'development',

  // Feature Flags
  enableDebugMode: parseBoolean(import.meta.env.VITE_ENABLE_DEBUG_MODE, true),
  enableAnalytics: parseBoolean(import.meta.env.VITE_ENABLE_ANALYTICS, false),
  enableMFA: parseBoolean(import.meta.env.VITE_ENABLE_MFA, true),
  enableBulkOperations: parseBoolean(import.meta.env.VITE_ENABLE_BULK_OPERATIONS, true),

  // UI Configuration
  defaultPageSize: parseNumber(import.meta.env.VITE_DEFAULT_PAGE_SIZE, 20),
  maxPageSize: parseNumber(import.meta.env.VITE_MAX_PAGE_SIZE, 100),
  theme: import.meta.env.VITE_THEME || 'light',

  // Security Configuration
  sessionTimeout: parseNumber(import.meta.env.VITE_SESSION_TIMEOUT, 1800000), // 30 minutes
  tokenRefreshThreshold: parseNumber(import.meta.env.VITE_TOKEN_REFRESH_THRESHOLD, 300000), // 5 minutes
  maxLoginAttempts: parseNumber(import.meta.env.VITE_MAX_LOGIN_ATTEMPTS, 5),

  // Development Configuration
  mockApi: parseBoolean(import.meta.env.VITE_MOCK_API, false),
  logLevel: import.meta.env.VITE_LOG_LEVEL || 'info',
});

// Export the configuration
export const env = createEnvConfig();

// Environment helper functions
export const isDevelopment = () => env.appEnvironment === 'development';
export const isProduction = () => env.appEnvironment === 'production';
export const isDebugMode = () => env.enableDebugMode && isDevelopment();

// Feature flag helpers
export const featureFlags = {
  mfa: () => env.enableMFA,
  bulkOperations: () => env.enableBulkOperations,
  analytics: () => env.enableAnalytics,
  debugMode: () => env.enableDebugMode,
} as const;

// Log configuration in development
if (isDevelopment() && isDebugMode()) {
  console.group('🔧 Environment Configuration');
  console.log('API Base URL:', env.apiBaseUrl);
  console.log('Environment:', env.appEnvironment);
  console.log('Debug Mode:', env.enableDebugMode);
  console.log('Mock API:', env.mockApi);
  console.log('Feature Flags:', {
    mfa: featureFlags.mfa(),
    bulkOperations: featureFlags.bulkOperations(),
    analytics: featureFlags.analytics(),
  });
  console.groupEnd();
}