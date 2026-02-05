/**
 * Error types and handling
 */

export class LinktorError extends Error {
  public readonly code: string;
  public readonly statusCode?: number;
  public readonly requestId?: string;
  public readonly details?: Record<string, unknown>;

  constructor(
    message: string,
    code: string,
    statusCode?: number,
    requestId?: string,
    details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'LinktorError';
    this.code = code;
    this.statusCode = statusCode;
    this.requestId = requestId;
    this.details = details;

    // Maintains proper stack trace for where error was thrown
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, LinktorError);
    }
  }
}

export class AuthenticationError extends LinktorError {
  constructor(message = 'Authentication failed', requestId?: string) {
    super(message, 'AUTHENTICATION_ERROR', 401, requestId);
    this.name = 'AuthenticationError';
  }
}

export class AuthorizationError extends LinktorError {
  constructor(message = 'Access denied', requestId?: string) {
    super(message, 'AUTHORIZATION_ERROR', 403, requestId);
    this.name = 'AuthorizationError';
  }
}

export class NotFoundError extends LinktorError {
  constructor(resource: string, id?: string, requestId?: string) {
    const message = id ? `${resource} with id '${id}' not found` : `${resource} not found`;
    super(message, 'NOT_FOUND', 404, requestId);
    this.name = 'NotFoundError';
  }
}

export class ValidationError extends LinktorError {
  constructor(message: string, details?: Record<string, unknown>, requestId?: string) {
    super(message, 'VALIDATION_ERROR', 400, requestId, details);
    this.name = 'ValidationError';
  }
}

export class RateLimitError extends LinktorError {
  public readonly retryAfter?: number;

  constructor(message = 'Rate limit exceeded', retryAfter?: number, requestId?: string) {
    super(message, 'RATE_LIMIT_ERROR', 429, requestId, { retryAfter });
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
  }
}

export class ConflictError extends LinktorError {
  constructor(message: string, requestId?: string) {
    super(message, 'CONFLICT_ERROR', 409, requestId);
    this.name = 'ConflictError';
  }
}

export class ServerError extends LinktorError {
  constructor(message = 'Internal server error', requestId?: string) {
    super(message, 'SERVER_ERROR', 500, requestId);
    this.name = 'ServerError';
  }
}

export class NetworkError extends LinktorError {
  constructor(message = 'Network error', details?: Record<string, unknown>) {
    super(message, 'NETWORK_ERROR', undefined, undefined, details);
    this.name = 'NetworkError';
  }
}

export class TimeoutError extends LinktorError {
  constructor(message = 'Request timeout', requestId?: string) {
    super(message, 'TIMEOUT_ERROR', 408, requestId);
    this.name = 'TimeoutError';
  }
}

export class WebSocketError extends LinktorError {
  constructor(message: string, details?: Record<string, unknown>) {
    super(message, 'WEBSOCKET_ERROR', undefined, undefined, details);
    this.name = 'WebSocketError';
  }
}

/**
 * Create appropriate error from HTTP response
 */
export function createErrorFromResponse(
  status: number,
  body: { code?: string; message?: string; details?: Record<string, unknown> },
  requestId?: string
): LinktorError {
  const message = body.message || 'Unknown error';

  switch (status) {
    case 400:
      return new ValidationError(message, body.details, requestId);
    case 401:
      return new AuthenticationError(message, requestId);
    case 403:
      return new AuthorizationError(message, requestId);
    case 404:
      return new NotFoundError('Resource', undefined, requestId);
    case 409:
      return new ConflictError(message, requestId);
    case 429:
      return new RateLimitError(message, undefined, requestId);
    case 500:
    case 502:
    case 503:
    case 504:
      return new ServerError(message, requestId);
    default:
      return new LinktorError(message, body.code || 'UNKNOWN_ERROR', status, requestId, body.details);
  }
}

/**
 * Check if error is retryable
 */
export function isRetryableError(error: unknown): boolean {
  if (error instanceof RateLimitError) return true;
  if (error instanceof ServerError) return true;
  if (error instanceof NetworkError) return true;
  if (error instanceof TimeoutError) return true;
  return false;
}
