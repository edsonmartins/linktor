/**
 * API Client - Plugin Pattern
 * Centralized HTTP client with interceptors and automatic token refresh
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081/api/v1'
const WEBHOOK_BASE_URL = (
  process.env.NEXT_PUBLIC_WEBHOOK_BASE_URL ||
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/api\/v1\/?$/, '') ||
  'http://localhost:8081'
).replace(/\/$/, '')

// Flag to prevent multiple simultaneous refresh attempts
let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

type RequestConfig = {
  method?: string
  headers?: Record<string, string>
  body?: unknown
  params?: Record<string, string>
}

type ErrorPayload = {
  error?: { message?: string; code?: string }
  message?: string
}

type MetaResponse = {
  page: number
  page_size: number
  total_pages: number
  total_items: number
  has_next: boolean
  has_previous: boolean
}

type ApiEnvelope<T> = {
  success: boolean
  data?: T
  error?: ErrorPayload['error']
  meta?: MetaResponse
}

class ApiError extends Error {
  public code: string

  constructor(
    public status: number,
    public statusText: string,
    public data?: unknown
  ) {
    // Extract message from backend response
    const errorData = data as ErrorPayload | null
    const message = errorData?.error?.message || errorData?.message || `API Error: ${status} ${statusText}`
    super(message)
    this.name = 'ApiError'
    this.code = errorData?.error?.code || statusText
  }
}

/**
 * Session management.
 *
 * Access/refresh tokens live in HttpOnly cookies set by the API and are never
 * readable by JS (XSS cannot exfiltrate them). The browser attaches them
 * automatically because every request uses `credentials: 'include'`.
 *
 * A separate, non-sensitive `linktor_authed` marker cookie (readable by JS)
 * mirrors session presence so middleware.ts can redirect unauthenticated
 * visitors server-side. It is NOT an auth credential.
 */
const AUTH_MARKER_COOKIE = 'linktor_authed'

function setAuthMarkerCookie(present: boolean) {
  if (typeof document === 'undefined') return
  if (present) {
    document.cookie = `${AUTH_MARKER_COOKIE}=1; path=/; max-age=${60 * 60 * 24 * 30}; samesite=lax`
  } else {
    document.cookie = `${AUTH_MARKER_COOKIE}=; path=/; max-age=0; samesite=lax`
  }
}

// One-time cleanup: drop tokens left in localStorage by pre-cookie sessions.
if (typeof window !== 'undefined') {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}

const tokenStorage = {
  /** Marks the session as authenticated for the server-side middleware. */
  setAuthenticated: () => setAuthMarkerCookie(true),
  /** Clears the client-visible session marker (cookies are cleared by the API). */
  clear: () => setAuthMarkerCookie(false),
  /** Whether a session marker is present (HttpOnly auth cookies aren't readable). */
  isAuthenticated: () => {
    if (typeof document === 'undefined') return false
    return document.cookie.split('; ').some((c) => c === `${AUTH_MARKER_COOKIE}=1`)
  },
}

/**
 * Request interceptors - plugin pattern for request modification
 */
type RequestInterceptor = (config: RequestConfig) => RequestConfig | Promise<RequestConfig>
type ResponseInterceptor = (response: Response) => Response | Promise<Response>

const requestInterceptors: RequestInterceptor[] = []
const responseInterceptors: ResponseInterceptor[] = []

/**
 * Refresh the access token. The refresh token travels as an HttpOnly cookie,
 * so there is no token to read or send — the API rotates both cookies.
 */
async function refreshAccessToken(): Promise<boolean> {
  // If already refreshing, wait for the existing promise
  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }

  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        tokenStorage.clear()
        return false
      }

      // Success rotates the auth cookies server-side; keep the marker fresh.
      tokenStorage.setAuthenticated()
      return true
    } catch {
      tokenStorage.clear()
      return false
    } finally {
      isRefreshing = false
      refreshPromise = null
    }
  })()

  return refreshPromise
}

/**
 * Core fetch wrapper with automatic token refresh
 */
async function request<T>(endpoint: string, config: RequestConfig = {}, isRetry = false, unwrapEnvelope = true): Promise<T> {
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

  // Make request. credentials:'include' sends the HttpOnly auth cookies.
  let response = await fetch(url, {
    method: finalConfig.method || 'GET',
    credentials: 'include',
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

  // Handle 401 - try to refresh token
  if (response.status === 401 && !isRetry) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      // Retry the request with new token
      return request<T>(endpoint, config, true, unwrapEnvelope)
    } else {
      // Refresh failed - redirect to login
      if (typeof window !== 'undefined') {
        tokenStorage.clear()
        window.location.href = '/login'
      }
      throw new ApiError(401, 'Unauthorized', { message: 'Session expired' })
    }
  }

  // Handle other errors
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
  if (unwrapEnvelope && json && typeof json === 'object' && 'success' in json && 'data' in json) {
    return json.data as T
  }

  return json as T
}

async function requestEnvelope<T>(endpoint: string, config: RequestConfig = {}, isRetry = false): Promise<ApiEnvelope<T>> {
  return request<ApiEnvelope<T>>(endpoint, config, isRetry, false)
}

/**
 * Multipart upload. The core `request` wrapper always sends JSON, so file
 * uploads get their own path: FormData with no explicit Content-Type (the
 * browser sets the multipart boundary). Mirrors the 401 refresh + envelope
 * unwrapping behaviour of `request`.
 */
async function uploadFile<T>(endpoint: string, formData: FormData, isRetry = false): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method: 'POST',
    credentials: 'include',
    body: formData,
  })

  if (response.status === 401 && !isRetry) {
    const refreshed = await refreshAccessToken()
    if (refreshed) return uploadFile<T>(endpoint, formData, true)
    if (typeof window !== 'undefined') {
      tokenStorage.clear()
      window.location.href = '/login'
    }
    throw new ApiError(401, 'Unauthorized', { message: 'Session expired' })
  }

  if (!response.ok) {
    const data = await response.json().catch(() => null)
    throw new ApiError(response.status, response.statusText, data)
  }

  const text = await response.text()
  if (!text) return null as T
  const json = JSON.parse(text)
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

  upload: <T>(endpoint: string, formData: FormData) => uploadFile<T>(endpoint, formData),

  post: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: 'POST', body }),

  put: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: 'PUT', body }),

  patch: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: 'PATCH', body }),

  delete: <T>(endpoint: string) =>
    request<T>(endpoint, { method: 'DELETE' }),

  getEnvelope: <T>(endpoint: string, params?: Record<string, string>) =>
    requestEnvelope<T>(endpoint, { method: 'GET', params }),
}

export { tokenStorage, ApiError, API_BASE_URL, WEBHOOK_BASE_URL }
export type { ApiEnvelope, MetaResponse, RequestConfig, RequestInterceptor, ResponseInterceptor }
