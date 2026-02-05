"""
Webhook signature verification utilities
"""

import hashlib
import hmac
import json
import time
from typing import Any, Callable, Optional, TypeVar

from linktor.types.webhook import WebhookEvent, WebhookEventType

SIGNATURE_HEADER = "x-linktor-signature"
TIMESTAMP_HEADER = "x-linktor-timestamp"
DEFAULT_TOLERANCE = 300  # 5 minutes

T = TypeVar("T")


def compute_signature(payload: str | bytes, secret: str) -> str:
    """
    Compute HMAC-SHA256 signature

    Args:
        payload: Raw request body
        secret: Webhook secret

    Returns:
        Hex-encoded signature
    """
    if isinstance(payload, str):
        payload = payload.encode("utf-8")

    return hmac.new(
        secret.encode("utf-8"),
        payload,
        hashlib.sha256,
    ).hexdigest()


def verify_webhook_signature(
    payload: str | bytes,
    signature: str,
    secret: str,
) -> bool:
    """
    Verify webhook signature

    Args:
        payload: Raw request body
        signature: Signature from x-linktor-signature header
        secret: Webhook secret

    Returns:
        True if signature is valid
    """
    if not signature or not secret:
        return False

    expected_signature = compute_signature(payload, secret)

    # Timing-safe comparison
    return hmac.compare_digest(signature, expected_signature)


def verify_webhook(
    payload: str | bytes,
    headers: dict[str, str],
    secret: str,
    tolerance: int = DEFAULT_TOLERANCE,
) -> bool:
    """
    Verify webhook with timestamp validation

    Args:
        payload: Raw request body
        headers: Request headers (case-insensitive)
        secret: Webhook secret
        tolerance: Maximum age of the webhook in seconds

    Returns:
        True if signature and timestamp are valid
    """
    # Normalize header keys to lowercase
    headers_lower = {k.lower(): v for k, v in headers.items()}

    signature = headers_lower.get(SIGNATURE_HEADER)
    timestamp = headers_lower.get(TIMESTAMP_HEADER)

    if not signature:
        return False

    # Verify timestamp if present
    if timestamp:
        try:
            webhook_time = int(timestamp)
            now = int(time.time())
            if abs(now - webhook_time) > tolerance:
                return False
        except ValueError:
            return False

    return verify_webhook_signature(payload, signature, secret)


def construct_event(
    payload: str | bytes,
    headers: dict[str, str],
    secret: str,
    tolerance: int = DEFAULT_TOLERANCE,
) -> WebhookEvent[Any]:
    """
    Parse and validate webhook event

    Args:
        payload: Raw request body
        headers: Request headers
        secret: Webhook secret
        tolerance: Maximum age in seconds

    Returns:
        Parsed webhook event

    Raises:
        ValueError: If verification fails or payload is invalid
    """
    if not verify_webhook(payload, headers, secret, tolerance):
        raise ValueError("Webhook signature verification failed")

    if isinstance(payload, bytes):
        payload = payload.decode("utf-8")

    try:
        data = json.loads(payload)
    except json.JSONDecodeError as e:
        raise ValueError(f"Invalid JSON payload: {e}")

    if not data.get("id") or not data.get("type") or not data.get("timestamp"):
        raise ValueError("Invalid webhook event structure")

    return WebhookEvent(**data)


def is_event_type(event: WebhookEvent[Any], event_type: WebhookEventType) -> bool:
    """
    Check if event is of a specific type

    Args:
        event: Webhook event
        event_type: Expected event type

    Returns:
        True if event matches the type
    """
    return event.type == event_type


WebhookHandler = Callable[[WebhookEvent[Any]], None]


def create_webhook_handler(
    secret: str,
    handlers: dict[WebhookEventType, WebhookHandler],
    tolerance: int = DEFAULT_TOLERANCE,
) -> Callable[[str | bytes, dict[str, str]], tuple[int, Optional[str]]]:
    """
    Create a webhook handler factory

    Args:
        secret: Webhook secret
        handlers: Dict mapping event types to handler functions
        tolerance: Maximum age in seconds

    Returns:
        Handler function that returns (status_code, body)
    """

    def handle_webhook(
        payload: str | bytes,
        headers: dict[str, str],
    ) -> tuple[int, Optional[str]]:
        try:
            event = construct_event(payload, headers, secret, tolerance)

            handler = handlers.get(event.type)
            if handler:
                handler(event)

            return 200, None
        except ValueError as e:
            return 400, str(e)
        except Exception as e:
            return 500, str(e)

    return handle_webhook


__all__ = [
    "compute_signature",
    "verify_webhook_signature",
    "verify_webhook",
    "construct_event",
    "is_event_type",
    "create_webhook_handler",
    "SIGNATURE_HEADER",
    "TIMESTAMP_HEADER",
    "DEFAULT_TOLERANCE",
]
