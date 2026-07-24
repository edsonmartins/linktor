"""
Channel types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Optional, Union
from pydantic import BaseModel, Field

from linktor.types.common import ChannelType, PaginationParams


class ConnectionStatus(str, Enum):
    """Live connection state (wire field ``connection_status``)."""

    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    ERROR = "error"


class CoexistenceStatus(str, Enum):
    """WhatsApp Business App + Cloud API coexistence state."""

    INACTIVE = "inactive"
    PENDING = "pending"
    ACTIVE = "active"
    WARNING = "warning"
    DISCONNECTED = "disconnected"


# Deprecated alias kept for backwards-compatible imports.
ChannelStatus = ConnectionStatus


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
    """Channel model — mirrors the backend wire shape (snake_case).

    Credentials are write-only and never present on a response.
    """

    id: str
    tenant_id: str
    type: ChannelType
    name: str
    identifier: Optional[str] = None
    enabled: bool = False
    connection_status: ConnectionStatus = ConnectionStatus.DISCONNECTED
    config: Optional[dict[str, str]] = None
    webhook_url: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    # WhatsApp coexistence
    is_coexistence: Optional[bool] = None
    waba_id: Optional[str] = None
    last_echo_at: Optional[datetime] = None
    coexistence_status: Optional[CoexistenceStatus] = None
    message_template_namespace: Optional[str] = None

    class Config:
        populate_by_name = True
        extra = "ignore"


class CreateChannelInput(BaseModel):
    """Create channel input.

    ``config`` holds non-secret settings (phone_number_id, waba_id, ...);
    ``credentials`` holds secrets (access_token, bot_token, ...) — stored
    encrypted and never returned.
    """

    name: str
    type: ChannelType
    identifier: Optional[str] = None
    config: Optional[dict[str, str]] = None
    credentials: Optional[dict[str, str]] = None
    webhook_url: Optional[str] = None


class UpdateChannelInput(BaseModel):
    """Update channel input. Omit ``credentials`` to keep the stored secrets."""

    name: Optional[str] = None
    identifier: Optional[str] = None
    config: Optional[dict[str, str]] = None
    credentials: Optional[dict[str, str]] = None
    webhook_url: Optional[str] = None


class ConnectResult(BaseModel):
    """Result of connecting a channel.

    For WhatsApp Web-style linking, ``qr_code`` carries the payload to render and
    ``expires_in`` its lifetime in seconds — call ``connect`` again to refresh an
    expired code. ``pair_code`` is the phone-linking code. When
    ``passkey_required`` is true the account is passkey-locked and must be linked
    by signing ``passkey_challenge`` (submit via the passkey endpoint), not by QR.
    """

    channel: Optional[Channel] = None
    qr_code: Optional[str] = None
    expires_in: Optional[int] = None
    pair_code: Optional[str] = None
    passkey_required: Optional[bool] = None
    passkey_challenge: Optional[Any] = None

    class Config:
        extra = "ignore"


class PairCodeInput(BaseModel):
    """Body for requesting a WhatsApp pairing code."""

    phone_number: str


class ListChannelsParams(PaginationParams):
    """List channels parameters"""

    type: Optional[ChannelType] = None
    status: Optional[ConnectionStatus] = None
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
    "ConnectionStatus",
    "CoexistenceStatus",
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
    "ConnectResult",
    "PairCodeInput",
    "ListChannelsParams",
    "ChannelCapabilities",
]
