"""
Conversation and Message types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Optional
from pydantic import BaseModel, Field

from linktor.types.common import (
    ChannelType,
    ContentType,
    MessageDirection,
    MessageStatus,
    PaginationParams,
)


class ConversationStatus(str, Enum):
    """Conversation status"""

    OPEN = "open"
    PENDING = "pending"
    RESOLVED = "resolved"
    SNOOZED = "snoozed"


class SenderType(str, Enum):
    """Message sender type"""

    CONTACT = "contact"
    AGENT = "agent"
    BOT = "bot"
    SYSTEM = "system"


class MediaContent(BaseModel):
    """Media content in messages"""

    url: str
    mime_type: str = Field(alias="mimeType")
    filename: Optional[str] = None
    size: Optional[int] = None
    caption: Optional[str] = None
    thumbnail_url: Optional[str] = Field(None, alias="thumbnailUrl")

    class Config:
        populate_by_name = True


class LocationContent(BaseModel):
    """Location content"""

    latitude: float
    longitude: float
    name: Optional[str] = None
    address: Optional[str] = None


class ContactContent(BaseModel):
    """Contact content"""

    name: str
    phones: Optional[list[str]] = None
    emails: Optional[list[str]] = None


class ButtonContent(BaseModel):
    """Button content"""

    type: str  # 'reply', 'url', 'call'
    text: str
    payload: Optional[str] = None
    url: Optional[str] = None
    phone: Optional[str] = None


class ListRow(BaseModel):
    """List row"""

    id: str
    title: str
    description: Optional[str] = None


class ListSection(BaseModel):
    """List section"""

    title: str
    rows: list[ListRow]


class TemplateParameter(BaseModel):
    """Template parameter"""

    type: str  # 'text', 'image', 'document', 'video'
    text: Optional[str] = None
    media: Optional[MediaContent] = None


class TemplateComponent(BaseModel):
    """Template component"""

    type: str  # 'header', 'body', 'button'
    parameters: Optional[list[TemplateParameter]] = None


class TemplateContent(BaseModel):
    """Template content"""

    name: str
    language: str
    components: Optional[list[TemplateComponent]] = None


class MessageContent(BaseModel):
    """Message content"""

    text: Optional[str] = None
    media: Optional[MediaContent] = None
    location: Optional[LocationContent] = None
    contact: Optional[ContactContent] = None
    buttons: Optional[list[ButtonContent]] = None
    list_sections: Optional[list[ListSection]] = Field(None, alias="listSections")
    template: Optional[TemplateContent] = None

    class Config:
        populate_by_name = True


class Message(BaseModel):
    """Message model"""

    id: str
    conversation_id: str = Field(alias="conversationId")
    direction: MessageDirection
    content_type: ContentType = Field(alias="contentType")
    content: MessageContent
    status: MessageStatus
    external_id: Optional[str] = Field(None, alias="externalId")
    sender_id: Optional[str] = Field(None, alias="senderId")
    sender_type: SenderType = Field(alias="senderType")
    delivered_at: Optional[datetime] = Field(None, alias="deliveredAt")
    read_at: Optional[datetime] = Field(None, alias="readAt")
    failed_reason: Optional[str] = Field(None, alias="failedReason")
    metadata: Optional[dict[str, Any]] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class Conversation(BaseModel):
    """Conversation model"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    channel_id: str = Field(alias="channelId")
    channel_type: ChannelType = Field(alias="channelType")
    contact_id: str = Field(alias="contactId")
    status: ConversationStatus
    assigned_to: Optional[str] = Field(None, alias="assignedTo")
    assigned_at: Optional[datetime] = Field(None, alias="assignedAt")
    last_message: Optional[Message] = Field(None, alias="lastMessage")
    last_message_at: Optional[datetime] = Field(None, alias="lastMessageAt")
    unread_count: int = Field(alias="unreadCount")
    metadata: Optional[dict[str, Any]] = None
    tags: Optional[list[str]] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class ListConversationsParams(PaginationParams):
    """List conversations parameters"""

    status: Optional[ConversationStatus] = None
    channel_id: Optional[str] = Field(None, alias="channelId")
    channel_type: Optional[ChannelType] = Field(None, alias="channelType")
    assigned_to: Optional[str] = Field(None, alias="assignedTo")
    contact_id: Optional[str] = Field(None, alias="contactId")
    tags: Optional[list[str]] = None
    search: Optional[str] = None
    sort_by: Optional[str] = Field(None, alias="sortBy")
    sort_order: Optional[str] = Field(None, alias="sortOrder")

    class Config:
        populate_by_name = True


class SendMessageInput(BaseModel):
    """Send message input"""

    text: Optional[str] = None
    media: Optional[dict[str, Any]] = None
    location: Optional[LocationContent] = None
    buttons: Optional[list[ButtonContent]] = None
    template: Optional[TemplateContent] = None
    metadata: Optional[dict[str, Any]] = None


class UpdateConversationInput(BaseModel):
    """Update conversation input"""

    status: Optional[ConversationStatus] = None
    assigned_to: Optional[str] = Field(None, alias="assignedTo")
    tags: Optional[list[str]] = None
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class ListMessagesParams(PaginationParams):
    """List messages parameters"""

    before: Optional[str] = None
    after: Optional[str] = None


__all__ = [
    "ConversationStatus",
    "SenderType",
    "MediaContent",
    "LocationContent",
    "ContactContent",
    "ButtonContent",
    "ListRow",
    "ListSection",
    "TemplateParameter",
    "TemplateComponent",
    "TemplateContent",
    "MessageContent",
    "Message",
    "Conversation",
    "ListConversationsParams",
    "SendMessageInput",
    "UpdateConversationInput",
    "ListMessagesParams",
]
