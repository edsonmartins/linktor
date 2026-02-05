"""
Flow types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Optional
from pydantic import BaseModel, Field

from linktor.types.common import PaginationParams


class FlowStatus(str, Enum):
    """Flow status"""

    ACTIVE = "active"
    INACTIVE = "inactive"
    DRAFT = "draft"


class FlowNodeType(str, Enum):
    """Flow node type"""

    START = "start"
    MESSAGE = "message"
    CONDITION = "condition"
    ACTION = "action"
    INPUT = "input"
    AI = "ai"
    API = "api"
    DELAY = "delay"
    ASSIGN = "assign"
    END = "end"


class FlowExecutionStatus(str, Enum):
    """Flow execution status"""

    RUNNING = "running"
    WAITING = "waiting"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class Position(BaseModel):
    """Node position"""

    x: float
    y: float


class FlowNodeData(BaseModel):
    """Flow node data"""

    label: Optional[str] = None
    message_content: Optional[str] = Field(None, alias="messageContent")
    message_type: Optional[str] = Field(None, alias="messageType")
    buttons: Optional[list[dict[str, Any]]] = None
    conditions: Optional[list[dict[str, Any]]] = None
    action_type: Optional[str] = Field(None, alias="actionType")
    action_params: Optional[dict[str, Any]] = Field(None, alias="actionParams")
    input_variable: Optional[str] = Field(None, alias="inputVariable")
    input_validation: Optional[dict[str, Any]] = Field(None, alias="inputValidation")
    agent_id: Optional[str] = Field(None, alias="agentId")
    prompt: Optional[str] = None
    output_variable: Optional[str] = Field(None, alias="outputVariable")
    api_url: Optional[str] = Field(None, alias="apiUrl")
    api_method: Optional[str] = Field(None, alias="apiMethod")
    api_headers: Optional[dict[str, str]] = Field(None, alias="apiHeaders")
    api_body: Optional[str] = Field(None, alias="apiBody")
    delay_seconds: Optional[int] = Field(None, alias="delaySeconds")
    assignments: Optional[list[dict[str, Any]]] = None

    class Config:
        populate_by_name = True


class FlowNode(BaseModel):
    """Flow node"""

    id: str
    type: FlowNodeType
    position: Position
    data: FlowNodeData


class FlowEdge(BaseModel):
    """Flow edge"""

    id: str
    source: str
    target: str
    source_handle: Optional[str] = Field(None, alias="sourceHandle")
    target_handle: Optional[str] = Field(None, alias="targetHandle")
    label: Optional[str] = None
    condition: Optional[str] = None

    class Config:
        populate_by_name = True


class FlowVariable(BaseModel):
    """Flow variable"""

    name: str
    type: str  # string, number, boolean, array, object
    default_value: Optional[Any] = Field(None, alias="defaultValue")
    description: Optional[str] = None

    class Config:
        populate_by_name = True


class Flow(BaseModel):
    """Flow model"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    name: str
    description: Optional[str] = None
    status: FlowStatus
    version: int
    nodes: list[FlowNode]
    edges: list[FlowEdge]
    variables: Optional[list[FlowVariable]] = None
    metadata: Optional[dict[str, Any]] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class FlowExecutionStep(BaseModel):
    """Flow execution step"""

    node_id: str = Field(alias="nodeId")
    node_type: FlowNodeType = Field(alias="nodeType")
    started_at: datetime = Field(alias="startedAt")
    completed_at: Optional[datetime] = Field(None, alias="completedAt")
    input: Optional[dict[str, Any]] = None
    output: Optional[dict[str, Any]] = None
    error: Optional[str] = None

    class Config:
        populate_by_name = True


class FlowExecution(BaseModel):
    """Flow execution"""

    id: str
    flow_id: str = Field(alias="flowId")
    conversation_id: str = Field(alias="conversationId")
    status: FlowExecutionStatus
    current_node_id: Optional[str] = Field(None, alias="currentNodeId")
    variables: dict[str, Any]
    history: list[FlowExecutionStep]
    started_at: datetime = Field(alias="startedAt")
    completed_at: Optional[datetime] = Field(None, alias="completedAt")
    error: Optional[str] = None

    class Config:
        populate_by_name = True


class CreateFlowInput(BaseModel):
    """Create flow input"""

    name: str
    description: Optional[str] = None
    nodes: Optional[list[FlowNode]] = None
    edges: Optional[list[FlowEdge]] = None
    variables: Optional[list[FlowVariable]] = None
    metadata: Optional[dict[str, Any]] = None


class UpdateFlowInput(BaseModel):
    """Update flow input"""

    name: Optional[str] = None
    description: Optional[str] = None
    status: Optional[FlowStatus] = None
    nodes: Optional[list[FlowNode]] = None
    edges: Optional[list[FlowEdge]] = None
    variables: Optional[list[FlowVariable]] = None
    metadata: Optional[dict[str, Any]] = None


class ListFlowsParams(PaginationParams):
    """List flows parameters"""

    status: Optional[FlowStatus] = None
    search: Optional[str] = None


class ExecuteFlowInput(BaseModel):
    """Execute flow input"""

    conversation_id: str = Field(alias="conversationId")
    variables: Optional[dict[str, Any]] = None
    start_node_id: Optional[str] = Field(None, alias="startNodeId")

    class Config:
        populate_by_name = True


__all__ = [
    "FlowStatus",
    "FlowNodeType",
    "FlowExecutionStatus",
    "Position",
    "FlowNodeData",
    "FlowNode",
    "FlowEdge",
    "FlowVariable",
    "Flow",
    "FlowExecutionStep",
    "FlowExecution",
    "CreateFlowInput",
    "UpdateFlowInput",
    "ListFlowsParams",
    "ExecuteFlowInput",
]
