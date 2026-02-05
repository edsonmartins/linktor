"""
Contact types
"""

from datetime import datetime
from typing import Any, Optional
from pydantic import BaseModel, Field

from linktor.types.common import ChannelType, PaginationParams


class ContactIdentifier(BaseModel):
    """Contact identifier for a channel"""

    channel_type: ChannelType = Field(alias="channelType")
    channel_id: str = Field(alias="channelId")
    identifier: str
    display_name: Optional[str] = Field(None, alias="displayName")
    profile_picture: Optional[str] = Field(None, alias="profilePicture")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class Contact(BaseModel):
    """Contact model"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    external_id: Optional[str] = Field(None, alias="externalId")
    name: str
    email: Optional[str] = None
    phone: Optional[str] = None
    avatar_url: Optional[str] = Field(None, alias="avatarUrl")
    identifiers: list[ContactIdentifier]
    custom_fields: Optional[dict[str, Any]] = Field(None, alias="customFields")
    tags: Optional[list[str]] = None
    metadata: Optional[dict[str, Any]] = None
    last_seen_at: Optional[datetime] = Field(None, alias="lastSeenAt")
    conversation_count: int = Field(alias="conversationCount")
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class CreateContactInput(BaseModel):
    """Create contact input"""

    name: str
    email: Optional[str] = None
    phone: Optional[str] = None
    external_id: Optional[str] = Field(None, alias="externalId")
    avatar_url: Optional[str] = Field(None, alias="avatarUrl")
    custom_fields: Optional[dict[str, Any]] = Field(None, alias="customFields")
    tags: Optional[list[str]] = None
    identifiers: Optional[list[dict[str, Any]]] = None

    class Config:
        populate_by_name = True


class UpdateContactInput(BaseModel):
    """Update contact input"""

    name: Optional[str] = None
    email: Optional[str] = None
    phone: Optional[str] = None
    external_id: Optional[str] = Field(None, alias="externalId")
    avatar_url: Optional[str] = Field(None, alias="avatarUrl")
    custom_fields: Optional[dict[str, Any]] = Field(None, alias="customFields")
    tags: Optional[list[str]] = None

    class Config:
        populate_by_name = True


class ListContactsParams(PaginationParams):
    """List contacts parameters"""

    search: Optional[str] = None
    email: Optional[str] = None
    phone: Optional[str] = None
    tags: Optional[list[str]] = None
    channel_type: Optional[ChannelType] = Field(None, alias="channelType")
    sort_by: Optional[str] = Field(None, alias="sortBy")
    sort_order: Optional[str] = Field(None, alias="sortOrder")

    class Config:
        populate_by_name = True


class MergeContactsInput(BaseModel):
    """Merge contacts input"""

    primary_contact_id: str = Field(alias="primaryContactId")
    secondary_contact_ids: list[str] = Field(alias="secondaryContactIds")

    class Config:
        populate_by_name = True


__all__ = [
    "ContactIdentifier",
    "Contact",
    "CreateContactInput",
    "UpdateContactInput",
    "ListContactsParams",
    "MergeContactsInput",
]
