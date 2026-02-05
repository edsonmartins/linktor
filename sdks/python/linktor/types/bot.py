"""
Bot types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Optional
from pydantic import BaseModel, Field

from linktor.types.common import PaginationParams


class BotStatus(str, Enum):
    """Bot status"""

    ACTIVE = "active"
    INACTIVE = "inactive"
    DRAFT = "draft"


class BotType(str, Enum):
    """Bot type"""

    FLOW = "flow"
    AI = "ai"
    HYBRID = "hybrid"


class DaySchedule(BaseModel):
    """Day schedule for operating hours"""

    day: int  # 0-6 (Sunday-Saturday)
    enabled: bool
    start_time: str = Field(alias="startTime")
    end_time: str = Field(alias="endTime")

    class Config:
        populate_by_name = True


class OperatingHours(BaseModel):
    """Operating hours configuration"""

    enabled: bool
    timezone: str
    schedule: list[DaySchedule]
    outside_hours_message: Optional[str] = Field(None, alias="outsideHoursMessage")

    class Config:
        populate_by_name = True


class AIBotConfig(BaseModel):
    """AI bot configuration"""

    model: str
    temperature: Optional[float] = None
    max_tokens: Optional[int] = Field(None, alias="maxTokens")
    system_prompt: Optional[str] = Field(None, alias="systemPrompt")
    use_knowledge_base: bool = Field(alias="useKnowledgeBase")
    enable_streaming: Optional[bool] = Field(None, alias="enableStreaming")

    class Config:
        populate_by_name = True


class BotConfig(BaseModel):
    """Bot configuration"""

    welcome_message: Optional[str] = Field(None, alias="welcomeMessage")
    fallback_message: Optional[str] = Field(None, alias="fallbackMessage")
    handoff_message: Optional[str] = Field(None, alias="handoffMessage")
    handoff_triggers: Optional[list[str]] = Field(None, alias="handoffTriggers")
    operating_hours: Optional[OperatingHours] = Field(None, alias="operatingHours")
    ai_config: Optional[AIBotConfig] = Field(None, alias="aiConfig")

    class Config:
        populate_by_name = True


class Bot(BaseModel):
    """Bot model"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    name: str
    description: Optional[str] = None
    status: BotStatus
    type: BotType
    config: BotConfig
    channel_ids: list[str] = Field(alias="channelIds")
    flow_id: Optional[str] = Field(None, alias="flowId")
    agent_id: Optional[str] = Field(None, alias="agentId")
    knowledge_base_ids: Optional[list[str]] = Field(None, alias="knowledgeBaseIds")
    metadata: Optional[dict[str, Any]] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class CreateBotInput(BaseModel):
    """Create bot input"""

    name: str
    description: Optional[str] = None
    type: BotType
    config: Optional[dict[str, Any]] = None
    channel_ids: Optional[list[str]] = Field(None, alias="channelIds")
    flow_id: Optional[str] = Field(None, alias="flowId")
    agent_id: Optional[str] = Field(None, alias="agentId")
    knowledge_base_ids: Optional[list[str]] = Field(None, alias="knowledgeBaseIds")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class UpdateBotInput(BaseModel):
    """Update bot input"""

    name: Optional[str] = None
    description: Optional[str] = None
    status: Optional[BotStatus] = None
    config: Optional[dict[str, Any]] = None
    channel_ids: Optional[list[str]] = Field(None, alias="channelIds")
    flow_id: Optional[str] = Field(None, alias="flowId")
    agent_id: Optional[str] = Field(None, alias="agentId")
    knowledge_base_ids: Optional[list[str]] = Field(None, alias="knowledgeBaseIds")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class ListBotsParams(PaginationParams):
    """List bots parameters"""

    status: Optional[BotStatus] = None
    type: Optional[BotType] = None
    channel_id: Optional[str] = Field(None, alias="channelId")
    search: Optional[str] = None

    class Config:
        populate_by_name = True


__all__ = [
    "BotStatus",
    "BotType",
    "DaySchedule",
    "OperatingHours",
    "AIBotConfig",
    "BotConfig",
    "Bot",
    "CreateBotInput",
    "UpdateBotInput",
    "ListBotsParams",
]
