package types

import "time"

// FlowStatus type
type FlowStatus string

const (
	FlowStatusActive   FlowStatus = "active"
	FlowStatusInactive FlowStatus = "inactive"
	FlowStatusDraft    FlowStatus = "draft"
)

// FlowExecutionStatus type
type FlowExecutionStatus string

const (
	FlowExecutionStatusRunning   FlowExecutionStatus = "running"
	FlowExecutionStatusWaiting   FlowExecutionStatus = "waiting"
	FlowExecutionStatusCompleted FlowExecutionStatus = "completed"
	FlowExecutionStatusFailed    FlowExecutionStatus = "failed"
	FlowExecutionStatusCancelled FlowExecutionStatus = "cancelled"
)

// FlowNode in a flow
type FlowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position map[string]float64     `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

// FlowEdge connecting nodes
type FlowEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
	Label        string `json:"label,omitempty"`
	Condition    string `json:"condition,omitempty"`
}

// FlowVariable in a flow
type FlowVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Description  string      `json:"description,omitempty"`
}

// Flow model
type Flow struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenantId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Status      FlowStatus             `json:"status"`
	Version     int                    `json:"version"`
	Nodes       []FlowNode             `json:"nodes"`
	Edges       []FlowEdge             `json:"edges"`
	Variables   []FlowVariable         `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamps
}

// FlowExecutionStep in execution history
type FlowExecutionStep struct {
	NodeID      string                 `json:"nodeId"`
	NodeType    string                 `json:"nodeType"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// FlowExecution model
type FlowExecution struct {
	ID             string                 `json:"id"`
	FlowID         string                 `json:"flowId"`
	ConversationID string                 `json:"conversationId"`
	Status         FlowExecutionStatus    `json:"status"`
	CurrentNodeID  string                 `json:"currentNodeId,omitempty"`
	Variables      map[string]interface{} `json:"variables"`
	History        []FlowExecutionStep    `json:"history"`
	StartedAt      time.Time              `json:"startedAt"`
	CompletedAt    *time.Time             `json:"completedAt,omitempty"`
	Error          string                 `json:"error,omitempty"`
}

// CreateFlowInput for creating flows
type CreateFlowInput struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Nodes       []FlowNode             `json:"nodes,omitempty"`
	Edges       []FlowEdge             `json:"edges,omitempty"`
	Variables   []FlowVariable         `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateFlowInput for updating flows
type UpdateFlowInput struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Status      FlowStatus             `json:"status,omitempty"`
	Nodes       []FlowNode             `json:"nodes,omitempty"`
	Edges       []FlowEdge             `json:"edges,omitempty"`
	Variables   []FlowVariable         `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
