package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// FlowEngineService handles conversational flow execution
type FlowEngineService struct {
	flowRepo    repository.FlowRepository
	contextRepo repository.ConversationContextRepository
}

// NewFlowEngineService creates a new flow engine service
func NewFlowEngineService(
	flowRepo repository.FlowRepository,
	contextRepo repository.ConversationContextRepository,
) *FlowEngineService {
	return &FlowEngineService{
		flowRepo:    flowRepo,
		contextRepo: contextRepo,
	}
}

// CheckTrigger checks if any flow should be triggered by the message
func (s *FlowEngineService) CheckTrigger(ctx context.Context, tenantID string, message string, convContext *entity.ConversationContext) (*entity.Flow, bool) {
	// Check if there's already an active flow
	if s.HasActiveFlow(convContext) {
		return nil, false
	}

	// Get all active flows for the tenant
	flows, err := s.flowRepo.FindActiveByTenant(ctx, tenantID)
	if err != nil || len(flows) == 0 {
		return nil, false
	}

	lowerMessage := strings.ToLower(message)

	// Check each flow's trigger
	for _, flow := range flows {
		switch flow.Trigger {
		case entity.FlowTriggerKeyword:
			// Check if message contains the keyword
			if strings.Contains(lowerMessage, strings.ToLower(flow.TriggerValue)) {
				return flow, true
			}

		case entity.FlowTriggerIntent:
			// Check if detected intent matches
			if convContext != nil && convContext.Intent != nil {
				if strings.EqualFold(convContext.Intent.Name, flow.TriggerValue) {
					return flow, true
				}
			}

		case entity.FlowTriggerWelcome:
			// Check if this is a new conversation (no state)
			if convContext == nil || len(convContext.State) == 0 {
				return flow, true
			}
		}
	}

	return nil, false
}

// StartFlow starts a new flow execution
func (s *FlowEngineService) StartFlow(ctx context.Context, flow *entity.Flow, convContext *entity.ConversationContext) (*entity.FlowExecutionResult, error) {
	// Get start node
	startNode := flow.GetStartNode()
	if startNode == nil {
		return nil, errors.New(errors.ErrCodeBadRequest, "flow has no start node")
	}

	// Create execution state
	state := entity.NewFlowExecutionState(flow.ID, startNode.ID)

	// Update conversation context with flow state
	if convContext.State == nil {
		convContext.State = make(map[string]interface{})
	}
	convContext.State["active_flow_id"] = flow.ID
	convContext.State["current_node_id"] = startNode.ID
	convContext.State["flow_started_at"] = state.StartedAt
	convContext.State["collected_data"] = state.CollectedData

	// Execute the start node
	return s.ExecuteNode(ctx, flow, startNode, convContext, "")
}

// ContinueFlow continues an existing flow with user input
func (s *FlowEngineService) ContinueFlow(ctx context.Context, tenantID string, userInput string, convContext *entity.ConversationContext) (*entity.FlowExecutionResult, error) {
	// Get active flow
	flowID, ok := convContext.State["active_flow_id"].(string)
	if !ok || flowID == "" {
		return nil, errors.New(errors.ErrCodeBadRequest, "no active flow")
	}

	flow, err := s.flowRepo.FindByID(ctx, flowID)
	if err != nil {
		// Flow may have been deleted, clear state
		s.ClearFlowState(convContext)
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "flow not found")
	}

	// Get current node
	currentNodeID, ok := convContext.State["current_node_id"].(string)
	if !ok || currentNodeID == "" {
		s.ClearFlowState(convContext)
		return nil, errors.New(errors.ErrCodeBadRequest, "no current node")
	}

	currentNode := flow.GetNode(currentNodeID)
	if currentNode == nil {
		s.ClearFlowState(convContext)
		return nil, errors.New(errors.ErrCodeBadRequest, "current node not found")
	}

	// Process transition based on user input
	nextNodeID := s.ProcessTransition(currentNode, userInput)
	if nextNodeID == "" {
		// No valid transition, repeat current node or end flow
		if currentNode.Type == entity.FlowNodeEnd {
			s.ClearFlowState(convContext)
			return &entity.FlowExecutionResult{FlowEnded: true}, nil
		}
		// Re-execute current node
		return s.ExecuteNode(ctx, flow, currentNode, convContext, userInput)
	}

	// Get next node
	nextNode := flow.GetNode(nextNodeID)
	if nextNode == nil {
		s.ClearFlowState(convContext)
		return nil, errors.New(errors.ErrCodeBadRequest, "next node not found: "+nextNodeID)
	}

	// Update current node in state
	convContext.State["current_node_id"] = nextNodeID

	// Execute the next node
	return s.ExecuteNode(ctx, flow, nextNode, convContext, userInput)
}

// ExecuteNode executes a flow node and returns the result
func (s *FlowEngineService) ExecuteNode(ctx context.Context, flow *entity.Flow, node *entity.FlowNode, convContext *entity.ConversationContext, userInput string) (*entity.FlowExecutionResult, error) {
	result := &entity.FlowExecutionResult{}

	// Store any collected data from user input
	if userInput != "" && node.Type == entity.FlowNodeQuestion {
		s.StoreCollectedData(convContext, node.ID, userInput)
	}

	switch node.Type {
	case entity.FlowNodeMessage:
		// Send message and continue to next node
		result.Message = s.ProcessTemplate(node.Content, convContext)
		result.QuickReplies = node.QuickReplies
		result.Actions = node.Actions

		// Auto-advance to next node
		if len(node.Transitions) > 0 {
			result.NextNodeID = node.Transitions[0].ToNodeID
			convContext.State["current_node_id"] = result.NextNodeID
		}

	case entity.FlowNodeQuestion:
		// Send question and wait for user response
		result.Message = s.ProcessTemplate(node.Content, convContext)
		result.QuickReplies = node.QuickReplies
		result.ShouldWait = true

	case entity.FlowNodeCondition:
		// Evaluate condition and transition
		nextNodeID := s.ProcessTransition(node, userInput)
		if nextNodeID != "" {
			result.NextNodeID = nextNodeID
			convContext.State["current_node_id"] = nextNodeID
			// Execute the next node immediately
			nextNode := flow.GetNode(nextNodeID)
			if nextNode != nil {
				return s.ExecuteNode(ctx, flow, nextNode, convContext, "")
			}
		}

	case entity.FlowNodeAction:
		// Execute actions
		result.Message = s.ProcessTemplate(node.Content, convContext)
		result.Actions = node.Actions

		// Continue to next node
		if len(node.Transitions) > 0 {
			result.NextNodeID = node.Transitions[0].ToNodeID
			convContext.State["current_node_id"] = result.NextNodeID
		}

	case entity.FlowNodeVRE:
		// VRE (Visual Response Engine) node - renders a visual template
		result.IsVREResponse = true

		if node.VREConfig != nil {
			// Store VRE config in result for the bot orchestrator to render
			result.Message = node.VREConfig.TemplateID // Template ID for rendering
			result.VRECaption = s.ProcessTemplate(node.VREConfig.Caption, convContext)
			result.VREFollowUp = s.ProcessTemplate(node.VREConfig.FollowUpText, convContext)

			// Build template data from flow collected data using the mapping
			templateData := make(map[string]interface{})
			collected := s.GetCollectedData(convContext)
			for templateKey, flowKey := range node.VREConfig.DataMapping {
				// Process template syntax in the flow key (e.g., {{user_name}})
				value := s.ProcessTemplate(flowKey, convContext)
				if collected != nil && collected[flowKey] != "" {
					templateData[templateKey] = collected[flowKey]
				} else {
					templateData[templateKey] = value
				}
			}

			// Store template data in context for the VRE renderer
			if convContext.State == nil {
				convContext.State = make(map[string]interface{})
			}
			convContext.State["vre_template_data"] = templateData
		}

		// Continue to next node
		if len(node.Transitions) > 0 {
			result.NextNodeID = node.Transitions[0].ToNodeID
			convContext.State["current_node_id"] = result.NextNodeID
		}

	case entity.FlowNodeEnd:
		// End the flow
		result.Message = s.ProcessTemplate(node.Content, convContext)
		result.FlowEnded = true
		s.ClearFlowState(convContext)
	}

	return result, nil
}

// ProcessTransition determines the next node based on user input
func (s *FlowEngineService) ProcessTransition(node *entity.FlowNode, userInput string) string {
	if len(node.Transitions) == 0 {
		return ""
	}

	lowerInput := strings.ToLower(strings.TrimSpace(userInput))
	var defaultTransition *entity.FlowTransition

	// Sort by priority (higher first) would be ideal, but for now just iterate
	for i := range node.Transitions {
		transition := &node.Transitions[i]

		switch transition.Condition {
		case entity.TransitionConditionDefault:
			defaultTransition = transition

		case entity.TransitionConditionReplyEquals:
			// Check if user input matches exactly (case-insensitive)
			if strings.EqualFold(lowerInput, strings.ToLower(transition.Value)) {
				return transition.ToNodeID
			}
			// Also check quick reply IDs
			for _, qr := range node.QuickReplies {
				if strings.EqualFold(lowerInput, strings.ToLower(qr.ID)) ||
					strings.EqualFold(lowerInput, strings.ToLower(qr.Title)) {
					if strings.EqualFold(qr.ID, transition.Value) || strings.EqualFold(qr.Title, transition.Value) {
						return transition.ToNodeID
					}
				}
			}

		case entity.TransitionConditionContains:
			if strings.Contains(lowerInput, strings.ToLower(transition.Value)) {
				return transition.ToNodeID
			}

		case entity.TransitionConditionRegex:
			if matched, _ := regexp.MatchString(transition.Value, userInput); matched {
				return transition.ToNodeID
			}
		}
	}

	// Return default transition if exists
	if defaultTransition != nil {
		return defaultTransition.ToNodeID
	}

	return ""
}

// HasActiveFlow checks if there's an active flow in the context
func (s *FlowEngineService) HasActiveFlow(convContext *entity.ConversationContext) bool {
	if convContext == nil || convContext.State == nil {
		return false
	}
	flowID, ok := convContext.State["active_flow_id"].(string)
	return ok && flowID != ""
}

// GetActiveFlowID returns the active flow ID if any
func (s *FlowEngineService) GetActiveFlowID(convContext *entity.ConversationContext) string {
	if convContext == nil || convContext.State == nil {
		return ""
	}
	flowID, _ := convContext.State["active_flow_id"].(string)
	return flowID
}

// ClearFlowState clears the flow execution state from context
func (s *FlowEngineService) ClearFlowState(convContext *entity.ConversationContext) {
	if convContext == nil || convContext.State == nil {
		return
	}
	delete(convContext.State, "active_flow_id")
	delete(convContext.State, "current_node_id")
	delete(convContext.State, "flow_started_at")
	// Keep collected_data for reference
}

// StoreCollectedData stores user input as collected data
func (s *FlowEngineService) StoreCollectedData(convContext *entity.ConversationContext, key, value string) {
	if convContext == nil || convContext.State == nil {
		return
	}

	collected, ok := convContext.State["collected_data"].(map[string]string)
	if !ok {
		collected = make(map[string]string)
	}
	collected[key] = value
	convContext.State["collected_data"] = collected
}

// GetCollectedData returns the collected data from the flow
func (s *FlowEngineService) GetCollectedData(convContext *entity.ConversationContext) map[string]string {
	if convContext == nil || convContext.State == nil {
		return nil
	}
	collected, ok := convContext.State["collected_data"].(map[string]string)
	if !ok {
		return nil
	}
	return collected
}

// ProcessTemplate processes a message template with collected data
func (s *FlowEngineService) ProcessTemplate(template string, convContext *entity.ConversationContext) string {
	if template == "" || convContext == nil {
		return template
	}

	result := template

	// Replace {{key}} with collected data
	collected := s.GetCollectedData(convContext)
	if collected != nil {
		for key, value := range collected {
			placeholder := "{{" + key + "}}"
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	// Replace entity values
	if convContext.Entities != nil {
		for key, value := range convContext.Entities {
			placeholder := "{{entity." + key + "}}"
			if strVal, ok := value.(string); ok {
				result = strings.ReplaceAll(result, placeholder, strVal)
			}
		}
	}

	return result
}

// FlowService handles flow CRUD operations
type FlowService struct {
	flowRepo repository.FlowRepository
}

// NewFlowService creates a new flow service
func NewFlowService(flowRepo repository.FlowRepository) *FlowService {
	return &FlowService{
		flowRepo: flowRepo,
	}
}

// Create creates a new flow
func (s *FlowService) Create(ctx context.Context, tenantID string, input *entity.CreateFlowInput) (*entity.Flow, error) {
	// Validate nodes
	if len(input.Nodes) == 0 {
		return nil, errors.New(errors.ErrCodeBadRequest, "flow must have at least one node")
	}

	// Validate start node exists
	startNodeExists := false
	for _, node := range input.Nodes {
		if node.ID == input.StartNodeID {
			startNodeExists = true
			break
		}
	}
	if !startNodeExists {
		return nil, errors.New(errors.ErrCodeBadRequest, "start node not found in nodes")
	}

	flow := entity.NewFlow(tenantID, input.Name, input.Trigger, input.TriggerValue)
	flow.ID = uuid.New().String()
	flow.BotID = input.BotID
	flow.Description = input.Description
	flow.StartNodeID = input.StartNodeID
	flow.Nodes = input.Nodes
	flow.Priority = input.Priority

	if err := s.flowRepo.Create(ctx, flow); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create flow")
	}

	return flow, nil
}

// GetByID gets a flow by ID
func (s *FlowService) GetByID(ctx context.Context, id string) (*entity.Flow, error) {
	return s.flowRepo.FindByID(ctx, id)
}

// List lists flows for a tenant
func (s *FlowService) List(ctx context.Context, tenantID string, filter *entity.FlowFilter, params *repository.ListParams) ([]*entity.Flow, int64, error) {
	return s.flowRepo.FindByTenant(ctx, tenantID, filter, params)
}

// Update updates a flow
func (s *FlowService) Update(ctx context.Context, id string, input *entity.UpdateFlowInput) (*entity.Flow, error) {
	flow, err := s.flowRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		flow.Name = *input.Name
	}
	if input.Description != nil {
		flow.Description = *input.Description
	}
	if input.TriggerValue != nil {
		flow.TriggerValue = *input.TriggerValue
	}
	if input.StartNodeID != nil {
		flow.StartNodeID = *input.StartNodeID
	}
	if input.Nodes != nil {
		flow.Nodes = *input.Nodes
	}
	if input.Priority != nil {
		flow.Priority = *input.Priority
	}

	if err := s.flowRepo.Update(ctx, flow); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update flow")
	}

	return flow, nil
}

// Delete deletes a flow
func (s *FlowService) Delete(ctx context.Context, id string) error {
	return s.flowRepo.Delete(ctx, id)
}

// Activate activates a flow
func (s *FlowService) Activate(ctx context.Context, id string) error {
	return s.flowRepo.UpdateStatus(ctx, id, true)
}

// Deactivate deactivates a flow
func (s *FlowService) Deactivate(ctx context.Context, id string) error {
	return s.flowRepo.UpdateStatus(ctx, id, false)
}

// TestFlow tests a flow with simulated input
func (s *FlowService) TestFlow(ctx context.Context, flowID string, inputs []string) ([]*entity.FlowExecutionResult, error) {
	flow, err := s.flowRepo.FindByID(ctx, flowID)
	if err != nil {
		return nil, err
	}

	// Create a temporary context for testing
	tempContext := &entity.ConversationContext{
		State:    make(map[string]interface{}),
		Entities: make(map[string]interface{}),
	}

	// Create a temporary engine (without repos since we're testing)
	engine := &FlowEngineService{}

	var results []*entity.FlowExecutionResult

	// Start the flow
	startNode := flow.GetStartNode()
	if startNode == nil {
		return nil, errors.New(errors.ErrCodeBadRequest, "flow has no start node")
	}

	result, err := engine.ExecuteNode(ctx, flow, startNode, tempContext, "")
	if err != nil {
		return nil, err
	}
	results = append(results, result)

	// Process each input
	for _, input := range inputs {
		if result.FlowEnded {
			break
		}

		// Get current node
		currentNodeID, _ := tempContext.State["current_node_id"].(string)
		if currentNodeID == "" {
			break
		}

		currentNode := flow.GetNode(currentNodeID)
		if currentNode == nil {
			break
		}

		// Process transition
		nextNodeID := engine.ProcessTransition(currentNode, input)
		if nextNodeID == "" {
			continue
		}

		nextNode := flow.GetNode(nextNodeID)
		if nextNode == nil {
			continue
		}

		tempContext.State["current_node_id"] = nextNodeID

		result, err = engine.ExecuteNode(ctx, flow, nextNode, tempContext, input)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}
