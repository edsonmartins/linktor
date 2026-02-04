package entity

import (
	"time"
)

// FlowTriggerType represents how a flow is triggered
type FlowTriggerType string

const (
	FlowTriggerIntent  FlowTriggerType = "intent"  // Triggered by detected intent
	FlowTriggerKeyword FlowTriggerType = "keyword" // Triggered by keyword match
	FlowTriggerManual  FlowTriggerType = "manual"  // Manually started
	FlowTriggerWelcome FlowTriggerType = "welcome" // First message in conversation
)

// FlowNodeType represents the type of flow node
type FlowNodeType string

const (
	FlowNodeMessage   FlowNodeType = "message"   // Send a message
	FlowNodeQuestion  FlowNodeType = "question"  // Ask a question with options
	FlowNodeCondition FlowNodeType = "condition" // Conditional branch
	FlowNodeAction    FlowNodeType = "action"    // Execute an action
	FlowNodeEnd       FlowNodeType = "end"       // End the flow
)

// TransitionCondition represents the condition for a transition
type TransitionCondition string

const (
	TransitionConditionDefault     TransitionCondition = "default"      // Always matches (fallback)
	TransitionConditionReplyEquals TransitionCondition = "reply_equals" // Exact match
	TransitionConditionContains    TransitionCondition = "contains"     // Contains substring
	TransitionConditionRegex       TransitionCondition = "regex"        // Regex match
	TransitionConditionEntity      TransitionCondition = "entity"       // Entity was extracted
)

// FlowActionType represents the type of action to execute
type FlowActionType string

const (
	FlowActionTag       FlowActionType = "tag"        // Add tag to conversation
	FlowActionAssign    FlowActionType = "assign"     // Assign to user/team
	FlowActionEscalate  FlowActionType = "escalate"   // Escalate to human
	FlowActionSetEntity FlowActionType = "set_entity" // Store entity value
	FlowActionHTTP      FlowActionType = "http_call"  // Make HTTP request
	FlowActionNotify    FlowActionType = "notify"     // Send notification
)

// QuickReply represents an interactive button/option
type QuickReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Value string `json:"value,omitempty"` // Alternative value (if different from title)
}

// FlowAction represents an action to be executed
type FlowAction struct {
	Type   FlowActionType         `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// FlowTransition represents a transition between nodes
type FlowTransition struct {
	ID        string              `json:"id"`
	ToNodeID  string              `json:"to_node_id"`
	Condition TransitionCondition `json:"condition"`
	Value     string              `json:"value,omitempty"` // Value to compare against
	Priority  int                 `json:"priority"`        // Higher priority checked first
}

// FlowNode represents a single node in the flow
type FlowNode struct {
	ID           string                 `json:"id"`
	Type         FlowNodeType           `json:"type"`
	Content      string                 `json:"content,omitempty"`       // Message text or template
	QuickReplies []QuickReply           `json:"quick_replies,omitempty"` // Buttons for questions
	Transitions  []FlowTransition       `json:"transitions"`
	Actions      []FlowAction           `json:"actions,omitempty"` // Actions to execute
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Flow represents a conversational flow (decision tree)
type Flow struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	BotID        *string         `json:"bot_id,omitempty"` // Optional - can be global
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Trigger      FlowTriggerType `json:"trigger"`
	TriggerValue string          `json:"trigger_value,omitempty"` // Intent name, keyword, etc.
	StartNodeID  string          `json:"start_node_id"`           // Entry point
	Nodes        []FlowNode      `json:"nodes"`
	IsActive     bool            `json:"is_active"`
	Priority     int             `json:"priority"` // Higher priority flows checked first
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// NewFlow creates a new flow
func NewFlow(tenantID, name string, trigger FlowTriggerType, triggerValue string) *Flow {
	now := time.Now()
	return &Flow{
		TenantID:     tenantID,
		Name:         name,
		Trigger:      trigger,
		TriggerValue: triggerValue,
		Nodes:        []FlowNode{},
		IsActive:     false,
		Priority:     0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// AddNode adds a node to the flow
func (f *Flow) AddNode(node FlowNode) {
	f.Nodes = append(f.Nodes, node)
	// Set as start node if first node
	if f.StartNodeID == "" {
		f.StartNodeID = node.ID
	}
	f.UpdatedAt = time.Now()
}

// GetNode returns a node by ID
func (f *Flow) GetNode(nodeID string) *FlowNode {
	for i := range f.Nodes {
		if f.Nodes[i].ID == nodeID {
			return &f.Nodes[i]
		}
	}
	return nil
}

// GetStartNode returns the start node
func (f *Flow) GetStartNode() *FlowNode {
	return f.GetNode(f.StartNodeID)
}

// Activate activates the flow
func (f *Flow) Activate() {
	f.IsActive = true
	f.UpdatedAt = time.Now()
}

// Deactivate deactivates the flow
func (f *Flow) Deactivate() {
	f.IsActive = false
	f.UpdatedAt = time.Now()
}

// SetBotID associates the flow with a specific bot
func (f *Flow) SetBotID(botID string) {
	f.BotID = &botID
	f.UpdatedAt = time.Now()
}

// IsGlobal returns true if the flow is not tied to a specific bot
func (f *Flow) IsGlobal() bool {
	return f.BotID == nil
}

// FlowExecutionState represents the current state of flow execution
type FlowExecutionState struct {
	FlowID        string                 `json:"flow_id"`
	CurrentNodeID string                 `json:"current_node_id"`
	StartedAt     time.Time              `json:"started_at"`
	CollectedData map[string]string      `json:"collected_data"` // Entities collected during flow
	Variables     map[string]interface{} `json:"variables"`      // Runtime variables
}

// NewFlowExecutionState creates a new execution state
func NewFlowExecutionState(flowID, startNodeID string) *FlowExecutionState {
	return &FlowExecutionState{
		FlowID:        flowID,
		CurrentNodeID: startNodeID,
		StartedAt:     time.Now(),
		CollectedData: make(map[string]string),
		Variables:     make(map[string]interface{}),
	}
}

// SetCollectedData stores collected entity data
func (s *FlowExecutionState) SetCollectedData(key, value string) {
	s.CollectedData[key] = value
}

// GetCollectedData retrieves collected entity data
func (s *FlowExecutionState) GetCollectedData(key string) (string, bool) {
	val, ok := s.CollectedData[key]
	return val, ok
}

// FlowExecutionResult represents the result of executing a flow node
type FlowExecutionResult struct {
	Message      string       `json:"message,omitempty"`       // Message to send
	QuickReplies []QuickReply `json:"quick_replies,omitempty"` // Quick replies for the message
	NextNodeID   string       `json:"next_node_id,omitempty"`  // Next node to execute
	Actions      []FlowAction `json:"actions,omitempty"`       // Actions to execute
	FlowEnded    bool         `json:"flow_ended"`              // True if flow has ended
	ShouldWait   bool         `json:"should_wait"`             // True if waiting for user input
}

// CreateFlowInput represents input for creating a flow
type CreateFlowInput struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	BotID        *string         `json:"bot_id,omitempty"`
	Trigger      FlowTriggerType `json:"trigger"`
	TriggerValue string          `json:"trigger_value,omitempty"`
	StartNodeID  string          `json:"start_node_id"`
	Nodes        []FlowNode      `json:"nodes"`
	Priority     int             `json:"priority"`
}

// UpdateFlowInput represents input for updating a flow
type UpdateFlowInput struct {
	Name         *string     `json:"name,omitempty"`
	Description  *string     `json:"description,omitempty"`
	TriggerValue *string     `json:"trigger_value,omitempty"`
	StartNodeID  *string     `json:"start_node_id,omitempty"`
	Nodes        *[]FlowNode `json:"nodes,omitempty"`
	Priority     *int        `json:"priority,omitempty"`
}

// FlowFilter represents filters for listing flows
type FlowFilter struct {
	BotID    *string `json:"bot_id,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
	Trigger  *string `json:"trigger,omitempty"`
}
