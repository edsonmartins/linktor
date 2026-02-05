"""
AI/Agent types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Optional
from pydantic import BaseModel, Field

from linktor.types.common import PaginationParams


class AgentStatus(str, Enum):
    """Agent status"""

    ACTIVE = "active"
    INACTIVE = "inactive"
    DRAFT = "draft"


class ToolParameter(BaseModel):
    """Tool parameter definition"""

    name: str
    type: str  # string, number, boolean, array, object
    description: str
    required: bool
    default: Optional[Any] = None
    enum: Optional[list[str]] = None


class AgentTool(BaseModel):
    """Agent tool definition"""

    name: str
    description: str
    parameters: list[ToolParameter]
    handler: Optional[str] = None


class Agent(BaseModel):
    """Agent model"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    name: str
    description: Optional[str] = None
    status: AgentStatus
    model: str
    system_prompt: Optional[str] = Field(None, alias="systemPrompt")
    temperature: float
    max_tokens: int = Field(alias="maxTokens")
    tools: Optional[list[AgentTool]] = None
    knowledge_base_ids: Optional[list[str]] = Field(None, alias="knowledgeBaseIds")
    metadata: Optional[dict[str, Any]] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class ChatMessage(BaseModel):
    """Chat message for completions"""

    role: str  # system, user, assistant, tool
    content: str
    name: Optional[str] = None
    tool_call_id: Optional[str] = Field(None, alias="toolCallId")
    tool_calls: Optional[list[dict[str, Any]]] = Field(None, alias="toolCalls")

    class Config:
        populate_by_name = True


class TokenUsage(BaseModel):
    """Token usage statistics"""

    prompt_tokens: int = Field(alias="promptTokens")
    completion_tokens: int = Field(alias="completionTokens")
    total_tokens: int = Field(alias="totalTokens")

    class Config:
        populate_by_name = True


class CompletionRequest(BaseModel):
    """Completion request"""

    agent_id: Optional[str] = Field(None, alias="agentId")
    model: Optional[str] = None
    messages: list[ChatMessage]
    temperature: Optional[float] = None
    max_tokens: Optional[int] = Field(None, alias="maxTokens")
    stream: Optional[bool] = None
    tools: Optional[list[AgentTool]] = None
    knowledge_base_ids: Optional[list[str]] = Field(None, alias="knowledgeBaseIds")
    conversation_id: Optional[str] = Field(None, alias="conversationId")

    class Config:
        populate_by_name = True


class CompletionResponse(BaseModel):
    """Completion response"""

    id: str
    model: str
    message: ChatMessage
    usage: TokenUsage
    finish_reason: str = Field(alias="finishReason")

    class Config:
        populate_by_name = True


class CompletionChunk(BaseModel):
    """Streaming completion chunk"""

    id: str
    delta: dict[str, Any]
    finish_reason: Optional[str] = Field(None, alias="finishReason")

    class Config:
        populate_by_name = True


class EmbeddingRequest(BaseModel):
    """Embedding request"""

    input: str | list[str]
    model: Optional[str] = None


class Embedding(BaseModel):
    """Single embedding"""

    index: int
    embedding: list[float]


class EmbeddingResponse(BaseModel):
    """Embedding response"""

    data: list[Embedding]
    model: str
    usage: dict[str, int]


class CreateAgentInput(BaseModel):
    """Create agent input"""

    name: str
    description: Optional[str] = None
    model: str
    system_prompt: Optional[str] = Field(None, alias="systemPrompt")
    temperature: Optional[float] = None
    max_tokens: Optional[int] = Field(None, alias="maxTokens")
    tools: Optional[list[AgentTool]] = None
    knowledge_base_ids: Optional[list[str]] = Field(None, alias="knowledgeBaseIds")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class UpdateAgentInput(BaseModel):
    """Update agent input"""

    name: Optional[str] = None
    description: Optional[str] = None
    status: Optional[AgentStatus] = None
    model: Optional[str] = None
    system_prompt: Optional[str] = Field(None, alias="systemPrompt")
    temperature: Optional[float] = None
    max_tokens: Optional[int] = Field(None, alias="maxTokens")
    tools: Optional[list[AgentTool]] = None
    knowledge_base_ids: Optional[list[str]] = Field(None, alias="knowledgeBaseIds")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class ListAgentsParams(PaginationParams):
    """List agents parameters"""

    status: Optional[AgentStatus] = None
    search: Optional[str] = None


class InvokeAgentInput(BaseModel):
    """Invoke agent input"""

    message: str
    conversation_id: Optional[str] = Field(None, alias="conversationId")
    context: Optional[dict[str, Any]] = None
    stream: Optional[bool] = None

    class Config:
        populate_by_name = True


class ToolResult(BaseModel):
    """Tool execution result"""

    tool_call_id: str = Field(alias="toolCallId")
    name: str
    result: Any

    class Config:
        populate_by_name = True


class InvokeAgentResponse(BaseModel):
    """Invoke agent response"""

    response: str
    conversation_id: str = Field(alias="conversationId")
    tool_results: Optional[list[ToolResult]] = Field(None, alias="toolResults")
    usage: TokenUsage

    class Config:
        populate_by_name = True


__all__ = [
    "AgentStatus",
    "ToolParameter",
    "AgentTool",
    "Agent",
    "ChatMessage",
    "TokenUsage",
    "CompletionRequest",
    "CompletionResponse",
    "CompletionChunk",
    "EmbeddingRequest",
    "Embedding",
    "EmbeddingResponse",
    "CreateAgentInput",
    "UpdateAgentInput",
    "ListAgentsParams",
    "InvokeAgentInput",
    "ToolResult",
    "InvokeAgentResponse",
]
