"""
Error types and handling
"""

from typing import Any, Optional


class LinktorError(Exception):
    """Base Linktor SDK error"""

    def __init__(
        self,
        message: str,
        code: str = "UNKNOWN_ERROR",
        status_code: Optional[int] = None,
        request_id: Optional[str] = None,
        details: Optional[dict[str, Any]] = None,
    ):
        super().__init__(message)
        self.message = message
        self.code = code
        self.status_code = status_code
        self.request_id = request_id
        self.details = details or {}

    def __str__(self) -> str:
        if self.request_id:
            return f"[{self.code}] {self.message} (request_id: {self.request_id})"
        return f"[{self.code}] {self.message}"

    def __repr__(self) -> str:
        return f"LinktorError(code={self.code!r}, message={self.message!r}, status_code={self.status_code})"


class AuthenticationError(LinktorError):
    """Authentication failed"""

    def __init__(self, message: str = "Authentication failed", request_id: Optional[str] = None):
        super().__init__(message, "AUTHENTICATION_ERROR", 401, request_id)


class AuthorizationError(LinktorError):
    """Access denied"""

    def __init__(self, message: str = "Access denied", request_id: Optional[str] = None):
        super().__init__(message, "AUTHORIZATION_ERROR", 403, request_id)


class NotFoundError(LinktorError):
    """Resource not found"""

    def __init__(
        self,
        resource: str,
        resource_id: Optional[str] = None,
        request_id: Optional[str] = None,
    ):
        if resource_id:
            message = f"{resource} with id '{resource_id}' not found"
        else:
            message = f"{resource} not found"
        super().__init__(message, "NOT_FOUND", 404, request_id)
        self.resource = resource
        self.resource_id = resource_id


class ValidationError(LinktorError):
    """Validation error"""

    def __init__(
        self,
        message: str,
        details: Optional[dict[str, Any]] = None,
        request_id: Optional[str] = None,
    ):
        super().__init__(message, "VALIDATION_ERROR", 400, request_id, details)


class RateLimitError(LinktorError):
    """Rate limit exceeded"""

    def __init__(
        self,
        message: str = "Rate limit exceeded",
        retry_after: Optional[int] = None,
        request_id: Optional[str] = None,
    ):
        super().__init__(message, "RATE_LIMIT_ERROR", 429, request_id, {"retry_after": retry_after})
        self.retry_after = retry_after


class ConflictError(LinktorError):
    """Conflict error"""

    def __init__(self, message: str, request_id: Optional[str] = None):
        super().__init__(message, "CONFLICT_ERROR", 409, request_id)


class ServerError(LinktorError):
    """Internal server error"""

    def __init__(
        self, message: str = "Internal server error", request_id: Optional[str] = None
    ):
        super().__init__(message, "SERVER_ERROR", 500, request_id)


class NetworkError(LinktorError):
    """Network error"""

    def __init__(self, message: str = "Network error", details: Optional[dict[str, Any]] = None):
        super().__init__(message, "NETWORK_ERROR", None, None, details)


class TimeoutError(LinktorError):
    """Request timeout"""

    def __init__(self, message: str = "Request timeout", request_id: Optional[str] = None):
        super().__init__(message, "TIMEOUT_ERROR", 408, request_id)


class WebSocketError(LinktorError):
    """WebSocket error"""

    def __init__(self, message: str, details: Optional[dict[str, Any]] = None):
        super().__init__(message, "WEBSOCKET_ERROR", None, None, details)


def create_error_from_response(
    status: int,
    body: dict[str, Any],
    request_id: Optional[str] = None,
) -> LinktorError:
    """Create appropriate error from HTTP response"""
    message = body.get("message", "Unknown error")
    code = body.get("code")
    details = body.get("details")

    if status == 400:
        return ValidationError(message, details, request_id)
    elif status == 401:
        return AuthenticationError(message, request_id)
    elif status == 403:
        return AuthorizationError(message, request_id)
    elif status == 404:
        return NotFoundError("Resource", None, request_id)
    elif status == 409:
        return ConflictError(message, request_id)
    elif status == 429:
        retry_after = body.get("retry_after")
        return RateLimitError(message, retry_after, request_id)
    elif status >= 500:
        return ServerError(message, request_id)
    else:
        return LinktorError(message, code or "UNKNOWN_ERROR", status, request_id, details)


def is_retryable_error(error: Exception) -> bool:
    """Check if error is retryable"""
    if isinstance(error, (RateLimitError, ServerError, NetworkError, TimeoutError)):
        return True
    return False


__all__ = [
    "LinktorError",
    "AuthenticationError",
    "AuthorizationError",
    "NotFoundError",
    "ValidationError",
    "RateLimitError",
    "ConflictError",
    "ServerError",
    "NetworkError",
    "TimeoutError",
    "WebSocketError",
    "create_error_from_response",
    "is_retryable_error",
]
