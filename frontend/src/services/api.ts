import { env } from '../config/env';
import type { ApiResponse, RequestConfig, HttpClient } from '../types/api';

// Custom errors
export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string,
    public details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export class NetworkError extends Error {
  constructor(message: string = 'Network error occurred') {
    super(message);
    this.name = 'NetworkError';
  }
}

export class TimeoutError extends Error {
  constructor(message: string = 'Request timeout') {
    super(message);
    this.name = 'TimeoutError';
  }
}

// HTTP Client implementation
class HttpClientImpl implements HttpClient {
  private baseURL: string;
  private defaultHeaders: Record<string, string>;

  constructor(baseURL: string = env.apiBaseUrl) {
    this.baseURL = baseURL.replace(/\/+$/, ''); // Remove trailing slashes
    this.defaultHeaders = {
      'Content-Type': 'application/json',
    };
  }

  private getAuthToken(): string | null {
    if (typeof window === 'undefined') return null;
    return window.localStorage.getItem('umanager_access_token');
  }

  private async request<T>(
    endpoint: string,
    config: RequestConfig = {}
  ): Promise<T> {
    const {
      method = 'GET',
      headers = {},
      body,
      params,
    } = config;

    // Build URL
    let url = `${this.baseURL}${endpoint}`;
    
    // Add query parameters
    if (params && typeof window !== 'undefined' && window.URLSearchParams) {
      const searchParams = new window.URLSearchParams();
      Object.entries(params).forEach(([key, value]) => {
        searchParams.append(key, String(value));
      });
      url += `?${searchParams.toString()}`;
    }

    // Build headers  
    const requestHeaders = new (typeof window !== 'undefined' && window.Headers ? window.Headers : Headers)({
      ...this.defaultHeaders,
      ...headers,
    });

    // Add auth token if available
    const token = this.getAuthToken();
    if (token) {
      requestHeaders.set('Authorization', `Bearer ${token}`);
    }

    // Build request options
    const requestInit: RequestInit = {
      method,
      headers: requestHeaders,
    };

    // Add body for non-GET requests
    if (body && method !== 'GET') {
      if (typeof window !== 'undefined' && window.FormData && body instanceof window.FormData) {
        // Remove Content-Type header for FormData (browser will set it with boundary)
        requestHeaders.delete('Content-Type');
        requestInit.body = body;
      } else if (typeof body === 'string') {
        requestInit.body = body;
      } else {
        requestInit.body = JSON.stringify(body);
      }
    }

    try {
      // Create timeout promise
      const timeoutPromise = new Promise<never>((_, reject) => {
        if (typeof window !== 'undefined') {
          window.setTimeout(() => reject(new TimeoutError()), env.apiTimeout);
        } else {
          reject(new TimeoutError());
        }
      });

      // Make the request with timeout
      const fetchFn = typeof window !== 'undefined' ? window.fetch : fetch;
      const response = await Promise.race([
        fetchFn(url, requestInit),
        timeoutPromise,
      ]);

      // Handle response
      return await this.handleResponse<T>(response);
    } catch (error) {
      if (error instanceof TimeoutError) {
        throw error;
      }

      if (error instanceof TypeError && error.message.includes('fetch')) {
        throw new NetworkError('Failed to connect to server');
      }

      throw error;
    }
  }

  private async handleResponse<T>(response: Response): Promise<T> {
    let data: unknown;

    // Parse response body
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      data = await response.json();
    } else {
      data = await response.text();
    }

    // Handle successful responses
    if (response.ok) {
      return data;
    }

    // Handle error responses
    let errorMessage = 'An error occurred';
    let errorCode = 'UNKNOWN_ERROR';
    let errorDetails: Record<string, unknown> = {};

    if (typeof data === 'object' && data !== null) {
      // API error response format
      const errorData = data as Record<string, unknown>;
      if (errorData.error && typeof errorData.error === 'object' && errorData.error !== null) {
        const error = errorData.error as Record<string, unknown>;
        errorMessage = (error.message as string) || errorMessage;
        errorCode = (error.code as string) || errorCode;
        errorDetails = (error.details as Record<string, unknown>) || {};
      } else if (errorData.message && typeof errorData.message === 'string') {
        errorMessage = errorData.message;
      }
    } else if (typeof data === 'string') {
      errorMessage = data;
    }

    // Handle specific HTTP status codes
    switch (response.status) {
      case 400:
        throw new ApiError(errorMessage || 'Bad Request', 400, errorCode, errorDetails);
      case 401:
        // Handle token expiry
        this.handleUnauthorized();
        throw new ApiError(errorMessage || 'Unauthorized', 401, errorCode, errorDetails);
      case 403:
        throw new ApiError(errorMessage || 'Forbidden', 403, errorCode, errorDetails);
      case 404:
        throw new ApiError(errorMessage || 'Not Found', 404, errorCode, errorDetails);
      case 409:
        throw new ApiError(errorMessage || 'Conflict', 409, errorCode, errorDetails);
      case 422:
        throw new ApiError(errorMessage || 'Validation Error', 422, errorCode, errorDetails);
      case 429:
        throw new ApiError(errorMessage || 'Too Many Requests', 429, errorCode, errorDetails);
      case 500:
        throw new ApiError(errorMessage || 'Internal Server Error', 500, errorCode, errorDetails);
      case 502:
        throw new ApiError(errorMessage || 'Bad Gateway', 502, errorCode, errorDetails);
      case 503:
        throw new ApiError(errorMessage || 'Service Unavailable', 503, errorCode, errorDetails);
      default:
        throw new ApiError(errorMessage, response.status, errorCode, errorDetails);
    }
  }

  private handleUnauthorized() {
    // Clear stored auth data
    if (typeof window !== 'undefined') {
      window.localStorage.removeItem('umanager_access_token');
      window.localStorage.removeItem('umanager_refresh_token');
      window.localStorage.removeItem('umanager_expires_at');
      window.localStorage.removeItem('umanager_user');
    }

    // Redirect to login page or emit auth error event
    // This will be handled by the auth store
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new window.CustomEvent<void>('auth:unauthorized'));
    }
  }

  // HTTP methods
  async get<T>(endpoint: string, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, { ...config, method: 'GET' });
  }

  async post<T>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, { ...config, method: 'POST', body: data });
  }

  async put<T>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, { ...config, method: 'PUT', body: data });
  }

  async patch<T>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, { ...config, method: 'PATCH', body: data });
  }

  async delete<T>(endpoint: string, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, { ...config, method: 'DELETE' });
  }

  // File upload helper
  async upload<T>(endpoint: string, file: File, additionalData?: Record<string, unknown>): Promise<T> {
    if (typeof window === 'undefined' || !window.FormData) {
      throw new Error('FormData not available in this environment');
    }
    const formData = new window.FormData();
    formData.append('file', file);

    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, String(value));
      });
    }

    return this.request<T>(endpoint, {
      method: 'POST',
      body: formData,
    });
  }

  // Download helper
  async download(endpoint: string, filename?: string, config?: RequestConfig): Promise<void> {
    try {
      const fetchFn = typeof window !== 'undefined' ? window.fetch : fetch;
      const response = await fetchFn(`${this.baseURL}${endpoint}`, {
        method: config?.method || 'GET',
        headers: {
          'Authorization': `Bearer ${this.getAuthToken()}`,
          ...config?.headers,
        },
      });

      if (!response.ok) {
        throw new ApiError('Download failed', response.status);
      }

      if (typeof window !== 'undefined') {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        
        const link = document.createElement('a');
        link.href = url;
        link.download = filename || 'download';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        window.URL.revokeObjectURL(url);
      }
    } catch {
      throw new ApiError('Download failed', 500);
    }
  }

  // Request retry helper
  async retryRequest<T>(
    requestFn: () => Promise<T>,
    maxRetries: number = 3,
    delay: number = 1000
  ): Promise<T> {
    let lastError: Error;

    for (let i = 0; i <= maxRetries; i++) {
      try {
        return await requestFn();
      } catch (error) {
        lastError = error as Error;

        // Don't retry on certain errors
        if (error instanceof ApiError && [400, 401, 403, 404, 422].includes(error.status!)) {
          throw error;
        }

        // Wait before retrying (exponential backoff)
        if (i < maxRetries) {
          await new Promise(resolve => {
            if (typeof window !== 'undefined') {
              window.setTimeout(resolve, delay * Math.pow(2, i));
            } else {
              setTimeout(resolve, delay * Math.pow(2, i));
            }
          });
        }
      }
    }

    throw lastError!;
  }
}

// Create and export the HTTP client instance
export const httpClient = new HttpClientImpl();

// Export types
export type { HttpClient, RequestConfig, ApiResponse };