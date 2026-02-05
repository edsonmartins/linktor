"""
Channel types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Optional, Union
from pydantic import BaseModel, Field

from linktor.types.common import ChannelType, PaginationParams


class ChannelStatus(str, Enum):
    """Channel status"""

    ACTIVE = "active"
    INACTIVE = "inactive"
    CONNECTING = "connecting"
    ERROR = "error"


class WhatsAppConfig(BaseModel):
    """WhatsApp channel config"""

    type: str = "whatsapp"
    phone_number_id: str = Field(alias="phoneNumberId")
    business_account_id: str = Field(alias="businessAccountId")
    access_token: str = Field(alias="accessToken")
    verify_token: str = Field(alias="verifyToken")
    app_secret: Optional[str] = Field(None, alias="appSecret")

    class Config:
        populate_by_name = True


class TelegramConfig(BaseModel):
    """Telegram channel config"""

    type: str = "telegram"
    bot_token: str = Field(alias="botToken")
    bot_username: Optional[str] = Field(None, alias="botUsername")
    webhook_secret: Optional[str] = Field(None, alias="webhookSecret")

    class Config:
        populate_by_name = True


class FacebookConfig(BaseModel):
    """Facebook channel config"""

    type: str = "facebook"
    page_id: str = Field(alias="pageId")
    page_access_token: str = Field(alias="pageAccessToken")
    app_secret: Optional[str] = Field(None, alias="appSecret")
    page_name: Optional[str] = Field(None, alias="pageName")

    class Config:
        populate_by_name = True


class InstagramConfig(BaseModel):
    """Instagram channel config"""

    type: str = "instagram"
    instagram_id: str = Field(alias="instagramId")
    page_access_token: str = Field(alias="pageAccessToken")
    app_secret: Optional[str] = Field(None, alias="appSecret")
    username: Optional[str] = None

    class Config:
        populate_by_name = True


class WebchatConfig(BaseModel):
    """Webchat channel config"""

    type: str = "webchat"
    widget_id: str = Field(alias="widgetId")
    allowed_origins: Optional[list[str]] = Field(None, alias="allowedOrigins")
    theme: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class SMSConfig(BaseModel):
    """SMS channel config"""

    type: str = "sms"
    provider: str = "twilio"
    account_sid: str = Field(alias="accountSid")
    auth_token: str = Field(alias="authToken")
    phone_number: str = Field(alias="phoneNumber")

    class Config:
        populate_by_name = True


class EmailConfig(BaseModel):
    """Email channel config"""

    type: str = "email"
    provider: str  # smtp, sendgrid, mailgun, ses, postmark
    from_email: str = Field(alias="fromEmail")
    from_name: Optional[str] = Field(None, alias="fromName")
    api_key: Optional[str] = Field(None, alias="apiKey")

    class Config:
        populate_by_name = True


class RCSConfig(BaseModel):
    """RCS channel config"""

    type: str = "rcs"
    provider: str  # zenvia, infobip, pontaltech
    agent_id: str = Field(alias="agentId")
    api_key: str = Field(alias="apiKey")
    brand_name: Optional[str] = Field(None, alias="brandName")

    class Config:
        populate_by_name = True


ChannelConfig = Union[
    WhatsAppConfig,
    TelegramConfig,
    FacebookConfig,
    InstagramConfig,
    WebchatConfig,
    SMSConfig,
    EmailConfig,
    RCSConfig,
]


class Channel(BaseModel):
    """Channel model"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    name: str
    type: ChannelType
    status: ChannelStatus
    config: dict[str, Any]
    webhook_url: Optional[str] = Field(None, alias="webhookUrl")
    metadata: Optional[dict[str, Any]] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class CreateChannelInput(BaseModel):
    """Create channel input"""

    name: str
    type: ChannelType
    config: dict[str, Any]
    metadata: Optional[dict[str, Any]] = None


class UpdateChannelInput(BaseModel):
    """Update channel input"""

    name: Optional[str] = None
    config: Optional[dict[str, Any]] = None
    metadata: Optional[dict[str, Any]] = None


class ListChannelsParams(PaginationParams):
    """List channels parameters"""

    type: Optional[ChannelType] = None
    status: Optional[ChannelStatus] = None
    search: Optional[str] = None


class ChannelCapabilities(BaseModel):
    """Channel capabilities"""

    supported_content_types: list[str] = Field(alias="supportedContentTypes")
    supports_media: bool = Field(alias="supportsMedia")
    supports_buttons: bool = Field(alias="supportsButtons")
    supports_lists: bool = Field(alias="supportsLists")
    supports_templates: bool = Field(alias="supportsTemplates")
    supports_location: bool = Field(alias="supportsLocation")
    max_message_length: int = Field(alias="maxMessageLength")
    max_media_size: int = Field(alias="maxMediaSize")

    class Config:
        populate_by_name = True


__all__ = [
    "ChannelStatus",
    "WhatsAppConfig",
    "TelegramConfig",
    "FacebookConfig",
    "InstagramConfig",
    "WebchatConfig",
    "SMSConfig",
    "EmailConfig",
    "RCSConfig",
    "ChannelConfig",
    "Channel",
    "CreateChannelInput",
    "UpdateChannelInput",
    "ListChannelsParams",
    "ChannelCapabilities",
]
