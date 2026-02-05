"""
Linktor SDK for Python

Official SDK for interacting with the Linktor API.
"""

from linktor.client import LinktorClient, LinktorAsyncClient
from linktor.types import *
from linktor.utils.errors import (
    LinktorError,
    AuthenticationError,
    AuthorizationError,
    NotFoundError,
    ValidationError,
    RateLimitError,
    ConflictError,
    ServerError,
    NetworkError,
    TimeoutError,
    WebSocketError,
)
from linktor.utils.webhook import (
    verify_webhook_signature,
    verify_webhook,
    compute_signature,
    construct_event,
)

__version__ = "1.0.0"
__all__ = [
    # Client
    "LinktorClient",
    "LinktorAsyncClient",
    # Errors
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
    # Webhook
    "verify_webhook_signature",
    "verify_webhook",
    "compute_signature",
    "construct_event",
]
