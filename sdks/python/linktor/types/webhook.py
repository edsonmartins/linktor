"""
Webhook types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Generic, Optional, TypeVar
from pydantic import BaseModel, Field

from linktor.types.common import ChannelType, MessageDirection, MessageStatus


class WebhookEventType(str, Enum):
    """Webhook event types"""

    MESSAGE_RECEIVED = "message.received"
    MESSAGE_SENT = "message.sent"
    MESSAGE_DELIVERED = "message.delivered"
    MESSAGE_READ = "message.read"
    MESSAGE_FAILED = "message.failed"
    CONVERSATION_CREATED = "conversation.created"
    CONVERSATION_UPDATED = "conversation.updated"
    CONVERSATION_ASSIGNED = "conversation.assigned"
    CONVERSATION_RESOLVED = "conversation.resolved"
    CONTACT_CREATED = "contact.created"
    CONTACT_UPDATED = "contact.updated"
    CHANNEL_CONNECTED = "channel.connected"
    CHANNEL_DISCONNECTED = "channel.disconnected"
    CHANNEL_ERROR = "channel.error"


T = TypeVar("T")


class WebhookEvent(BaseModel, Generic[T]):
    """Webhook event"""

    id: str
    type: WebhookEventType
    timestamp: datetime
    tenant_id: str = Field(alias="tenantId")
    data: T

    class Config:
        populate_by_name = True


class MessageReceivedEventData(BaseModel):
    """Message received event data"""

    message: dict[str, Any]
    conversation_id: str = Field(alias="conversationId")
    contact_id: str = Field(alias="contactId")
    channel_id: str = Field(alias="channelId")
    channel_type: ChannelType = Field(alias="channelType")

    class Config:
        populate_by_name = True


class MessageStatusEventData(BaseModel):
    """Message status event data"""

    message_id: str = Field(alias="messageId")
    conversation_id: str = Field(alias="conversationId")
    status: MessageStatus
    direction: MessageDirection
    timestamp: datetime
    error: Optional[str] = None

    class Config:
        populate_by_name = True


class ConversationEventData(BaseModel):
    """Conversation event data"""

    conversation_id: str = Field(alias="conversationId")
    contact_id: str = Field(alias="contactId")
    channel_id: str = Field(alias="channelId")
    status: str
    assigned_to: Optional[str] = Field(None, alias="assignedTo")
    previous_status: Optional[str] = Field(None, alias="previousStatus")
    previous_assigned_to: Optional[str] = Field(None, alias="previousAssignedTo")

    class Config:
        populate_by_name = True


class ContactEventData(BaseModel):
    """Contact event data"""

    contact_id: str = Field(alias="contactId")
    name: str
    email: Optional[str] = None
    phone: Optional[str] = None
    previous_data: Optional[dict[str, Any]] = Field(None, alias="previousData")

    class Config:
        populate_by_name = True


class ChannelEventData(BaseModel):
    """Channel event data"""

    channel_id: str = Field(alias="channelId")
    channel_type: ChannelType = Field(alias="channelType")
    status: str
    error: Optional[str] = None

    class Config:
        populate_by_name = True


class RetryPolicy(BaseModel):
    """Webhook retry policy"""

    max_retries: int = Field(alias="maxRetries")
    retry_interval: int = Field(alias="retryInterval")  # seconds
    exponential_backoff: bool = Field(alias="exponentialBackoff")

    class Config:
        populate_by_name = True


class WebhookConfig(BaseModel):
    """Webhook configuration"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    url: str
    secret: str
    events: list[WebhookEventType]
    status: str  # active, inactive
    headers: Optional[dict[str, str]] = None
    retry_policy: Optional[RetryPolicy] = Field(None, alias="retryPolicy")
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class CreateWebhookInput(BaseModel):
    """Create webhook input"""

    url: str
    events: list[WebhookEventType]
    headers: Optional[dict[str, str]] = None
    retry_policy: Optional[RetryPolicy] = Field(None, alias="retryPolicy")

    class Config:
        populate_by_name = True


class UpdateWebhookInput(BaseModel):
    """Update webhook input"""

    url: Optional[str] = None
    events: Optional[list[WebhookEventType]] = None
    status: Optional[str] = None
    headers: Optional[dict[str, str]] = None
    retry_policy: Optional[RetryPolicy] = Field(None, alias="retryPolicy")

    class Config:
        populate_by_name = True


class WebhookDelivery(BaseModel):
    """Webhook delivery record"""

    id: str
    webhook_id: str = Field(alias="webhookId")
    event_id: str = Field(alias="eventId")
    event_type: WebhookEventType = Field(alias="eventType")
    status: str  # pending, success, failed
    attempts: int
    last_attempt_at: Optional[datetime] = Field(None, alias="lastAttemptAt")
    response_status: Optional[int] = Field(None, alias="responseStatus")
    response_body: Optional[str] = Field(None, alias="responseBody")
    error: Optional[str] = None
    created_at: datetime = Field(alias="createdAt")

    class Config:
        populate_by_name = True


__all__ = [
    "WebhookEventType",
    "WebhookEvent",
    "MessageReceivedEventData",
    "MessageStatusEventData",
    "ConversationEventData",
    "ContactEventData",
    "ChannelEventData",
    "RetryPolicy",
    "WebhookConfig",
    "CreateWebhookInput",
    "UpdateWebhookInput",
    "WebhookDelivery",
]
