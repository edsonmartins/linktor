"""
Common types used across the SDK
"""

from datetime import datetime
from enum import Enum
from typing import Any, Generic, TypeVar, Optional
from pydantic import BaseModel, Field

T = TypeVar("T")


class PaginationParams(BaseModel):
    """Pagination parameters for list requests"""

    limit: Optional[int] = None
    offset: Optional[int] = None
    cursor: Optional[str] = None


class Pagination(BaseModel):
    """Pagination info in responses"""

    total: int
    limit: int
    offset: int
    has_more: bool = Field(alias="hasMore")
    next_cursor: Optional[str] = Field(None, alias="nextCursor")

    class Config:
        populate_by_name = True


class PaginatedResponse(BaseModel, Generic[T]):
    """Paginated response wrapper"""

    data: list[T]
    pagination: Pagination


class ApiError(BaseModel):
    """API error details"""

    code: str
    message: str
    details: Optional[dict[str, Any]] = None


class MessageDirection(str, Enum):
    """Message direction"""

    INBOUND = "inbound"
    OUTBOUND = "outbound"


class MessageStatus(str, Enum):
    """Message status"""

    PENDING = "pending"
    SENT = "sent"
    DELIVERED = "delivered"
    READ = "read"
    FAILED = "failed"


class ChannelType(str, Enum):
    """Channel types"""

    WHATSAPP = "whatsapp"
    TELEGRAM = "telegram"
    FACEBOOK = "facebook"
    INSTAGRAM = "instagram"
    WEBCHAT = "webchat"
    SMS = "sms"
    EMAIL = "email"
    RCS = "rcs"


class ContentType(str, Enum):
    """Content types"""

    TEXT = "text"
    IMAGE = "image"
    VIDEO = "video"
    AUDIO = "audio"
    DOCUMENT = "document"
    LOCATION = "location"
    CONTACT = "contact"
    STICKER = "sticker"


class TimestampsMixin(BaseModel):
    """Mixin for created_at/updated_at timestamps"""

    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


__all__ = [
    "PaginationParams",
    "Pagination",
    "PaginatedResponse",
    "ApiError",
    "MessageDirection",
    "MessageStatus",
    "ChannelType",
    "ContentType",
    "TimestampsMixin",
]
