/**
 * API Client - Plugin Pattern
 * Centralized HTTP client with interceptors
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081/api/v1'

type RequestConfig = {
  method?: string
  headers?: Record<string, string>
  body?: unknown
  params?: Record<string, string>
}

class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public data?: unknown
  ) {
    super(`API Error: ${status} ${statusText}`)
    this.name = 'ApiError'
  }
}

/**
 * Token management - uses localStorage for persistence
 */
const tokenStorage = {
  getAccessToken: () => {
    if (typeof window === 'undefined') return null
    return localStorage.getItem('access_token')
  },
  getRefreshToken: () => {
    if (typeof window === 'undefined') return null
    return localStorage.getItem('refresh_token')
  },
  setTokens: (accessToken: string, refreshToken: string) => {
    localStorage.setItem('access_token', accessToken)
    localStorage.setItem('refresh_token', refreshToken)
  },
  clearTokens: () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  },
}

/**
 * Request interceptors - plugin pattern for request modification
 */
type RequestInterceptor = (config: RequestConfig) => RequestConfig | Promise<RequestConfig>
type ResponseInterceptor = (response: Response) => Response | Promise<Response>

const requestInterceptors: RequestInterceptor[] = []
const responseInterceptors: ResponseInterceptor[] = []

// Add auth token to requests
requestInterceptors.push((config) => {
  const token = tokenStorage.getAccessToken()
  if (token) {
    config.headers = {
      ...config.headers,
      Authorization: `Bearer ${token}`,
    }
  }
  return config
})

/**
 * Core fetch wrapper
 */
async function request<T>(endpoint: string, config: RequestConfig = {}): Promise<T> {
  let finalConfig = { ...config }

  // Run request interceptors
  for (const interceptor of requestInterceptors) {
    finalConfig = await interceptor(finalConfig)
  }

  // Build URL with params
  let url = `${API_BASE_URL}${endpoint}`
  if (finalConfig.params) {
    const searchParams = new URLSearchParams(finalConfig.params)
    url += `?${searchParams.toString()}`
  }

  // Make request
  let response = await fetch(url, {
    method: finalConfig.method || 'GET',
    headers: {
      'Content-Type': 'application/json',
      ...finalConfig.headers,
    },
    body: finalConfig.body ? JSON.stringify(finalConfig.body) : undefined,
  })

  // Run response interceptors
  for (const interceptor of responseInterceptors) {
    response = await interceptor(response)
  }

  // Handle errors
  if (!response.ok) {
    const data = await response.json().catch(() => null)
    throw new ApiError(response.status, response.statusText, data)
  }

  // Parse response
  const text = await response.text()
  if (!text) return null as T

  const json = JSON.parse(text)

  // Backend wraps responses in { success: true, data: ... }
  // Unwrap the data if present
  if (json && typeof json === 'object' && 'success' in json && 'data' in json) {
    return json.data as T
  }

  return json as T
}

/**
 * HTTP methods
 */
export const api = {
  get: <T>(endpoint: string, params?: Record<string, string>) =>
    request<T>(endpoint, { method: 'GET', params }),

  post: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: 'POST', body }),

  put: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: 'PUT', body }),

  patch: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: 'PATCH', body }),

  delete: <T>(endpoint: string) =>
    request<T>(endpoint, { method: 'DELETE' }),
}

export { tokenStorage, ApiError }
export type { RequestConfig, RequestInterceptor, ResponseInterceptor }
