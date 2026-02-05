/**
 * HTTP client utility
 */

import axios, {
  AxiosError,
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from 'axios';
import {
  createErrorFromResponse,
  isRetryableError,
  LinktorError,
  NetworkError,
  TimeoutError,
} from './errors';

export interface HttpClientConfig {
  baseUrl: string;
  apiKey?: string;
  accessToken?: string;
  timeout?: number;
  maxRetries?: number;
  retryDelay?: number;
  headers?: Record<string, string>;
  onTokenRefresh?: () => Promise<string>;
}

export interface RequestConfig extends Omit<AxiosRequestConfig, 'baseURL'> {
  skipRetry?: boolean;
}

export class HttpClient {
  private client: AxiosInstance;
  private config: HttpClientConfig;
  private retryCount: Map<string, number> = new Map();

  constructor(config: HttpClientConfig) {
    this.config = {
      timeout: 30000,
      maxRetries: 3,
      retryDelay: 1000,
      ...config,
    };

    this.client = axios.create({
      baseURL: this.config.baseUrl,
      timeout: this.config.timeout,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        ...this.config.headers,
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors(): void {
    // Request interceptor - add auth headers
    this.client.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        if (this.config.apiKey) {
          config.headers['X-API-Key'] = this.config.apiKey;
        } else if (this.config.accessToken) {
          config.headers['Authorization'] = `Bearer ${this.config.accessToken}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor - handle errors
    this.client.interceptors.response.use(
      (response: AxiosResponse) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as InternalAxiosRequestConfig & {
          _retry?: boolean;
          _retryCount?: number;
        };

        // Handle token refresh on 401
        if (
          error.response?.status === 401 &&
          !originalRequest._retry &&
          this.config.onTokenRefresh
        ) {
          originalRequest._retry = true;
          try {
            const newToken = await this.config.onTokenRefresh();
            this.config.accessToken = newToken;
            originalRequest.headers['Authorization'] = `Bearer ${newToken}`;
            return this.client(originalRequest);
          } catch {
            throw this.transformError(error);
          }
        }

        throw this.transformError(error);
      }
    );
  }

  private transformError(error: AxiosError): LinktorError {
    if (error.code === 'ECONNABORTED') {
      return new TimeoutError('Request timeout');
    }

    if (!error.response) {
      return new NetworkError('Network error: ' + error.message);
    }

    const requestId = error.response.headers['x-request-id'] as string | undefined;
    const body = error.response.data as {
      code?: string;
      message?: string;
      details?: Record<string, unknown>;
    };

    return createErrorFromResponse(error.response.status, body || {}, requestId);
  }

  private async retryRequest<T>(
    requestFn: () => Promise<AxiosResponse<T>>,
    requestId: string,
    skipRetry?: boolean
  ): Promise<AxiosResponse<T>> {
    const maxRetries = this.config.maxRetries || 3;
    const retryDelay = this.config.retryDelay || 1000;

    let lastError: unknown;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        const response = await requestFn();
        this.retryCount.delete(requestId);
        return response;
      } catch (error) {
        lastError = error;

        if (skipRetry || attempt === maxRetries || !isRetryableError(error)) {
          this.retryCount.delete(requestId);
          throw error;
        }

        // Exponential backoff
        const delay = retryDelay * Math.pow(2, attempt);
        await this.sleep(delay);

        this.retryCount.set(requestId, attempt + 1);
      }
    }

    throw lastError;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  private generateRequestId(): string {
    return `req_${Date.now()}_${Math.random().toString(36).substring(7)}`;
  }

  async get<T>(path: string, config?: RequestConfig): Promise<T> {
    const requestId = this.generateRequestId();
    const response = await this.retryRequest(
      () => this.client.get<T>(path, config),
      requestId,
      config?.skipRetry
    );
    return response.data;
  }

  async post<T>(path: string, data?: unknown, config?: RequestConfig): Promise<T> {
    const requestId = this.generateRequestId();
    const response = await this.retryRequest(
      () => this.client.post<T>(path, data, config),
      requestId,
      config?.skipRetry
    );
    return response.data;
  }

  async put<T>(path: string, data?: unknown, config?: RequestConfig): Promise<T> {
    const requestId = this.generateRequestId();
    const response = await this.retryRequest(
      () => this.client.put<T>(path, data, config),
      requestId,
      config?.skipRetry
    );
    return response.data;
  }

  async patch<T>(path: string, data?: unknown, config?: RequestConfig): Promise<T> {
    const requestId = this.generateRequestId();
    const response = await this.retryRequest(
      () => this.client.patch<T>(path, data, config),
      requestId,
      config?.skipRetry
    );
    return response.data;
  }

  async delete<T>(path: string, config?: RequestConfig): Promise<T> {
    const requestId = this.generateRequestId();
    const response = await this.retryRequest(
      () => this.client.delete<T>(path, config),
      requestId,
      config?.skipRetry
    );
    return response.data;
  }

  async upload<T>(
    path: string,
    file: File | Blob,
    fieldName = 'file',
    additionalData?: Record<string, string>,
    config?: RequestConfig
  ): Promise<T> {
    const formData = new FormData();
    formData.append(fieldName, file);

    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, value);
      });
    }

    const requestId = this.generateRequestId();
    const response = await this.retryRequest(
      () =>
        this.client.post<T>(path, formData, {
          ...config,
          headers: {
            ...config?.headers,
            'Content-Type': 'multipart/form-data',
          },
        }),
      requestId,
      config?.skipRetry
    );
    return response.data;
  }

  /**
   * Stream response for SSE/chunked transfer
   */
  async *stream<T>(
    path: string,
    data?: unknown,
    config?: RequestConfig
  ): AsyncGenerator<T, void, unknown> {
    const response = await this.client.post(path, data, {
      ...config,
      responseType: 'stream',
    });

    const reader = response.data;
    const decoder = new TextDecoder();
    let buffer = '';

    for await (const chunk of reader) {
      buffer += decoder.decode(chunk, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const jsonStr = line.slice(6).trim();
          if (jsonStr && jsonStr !== '[DONE]') {
            try {
              yield JSON.parse(jsonStr) as T;
            } catch {
              // Skip invalid JSON
            }
          }
        }
      }
    }
  }

  /**
   * Update auth configuration
   */
  setAccessToken(token: string): void {
    this.config.accessToken = token;
  }

  setApiKey(apiKey: string): void {
    this.config.apiKey = apiKey;
  }
}
