package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock FlowRepository
// ---------------------------------------------------------------------------

type mockFlowRepo struct {
	flows       map[string]*entity.Flow
	returnError error
}

func newMockFlowRepo() *mockFlowRepo {
	return &mockFlowRepo{flows: make(map[string]*entity.Flow)}
}

func (m *mockFlowRepo) Create(ctx context.Context, flow *entity.Flow) error {
	if m.returnError != nil {
		return m.returnError
	}
	m.flows[flow.ID] = flow
	return nil
}

func (m *mockFlowRepo) FindByID(ctx context.Context, id string) (*entity.Flow, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	f, ok := m.flows[id]
	if !ok {
		return nil, fmt.Errorf("flow not found: %s", id)
	}
	return f, nil
}

func (m *mockFlowRepo) FindByTenant(ctx context.Context, tenantID string, filter *entity.FlowFilter, params *repository.ListParams) ([]*entity.Flow, int64, error) {
	if m.returnError != nil {
		return nil, 0, m.returnError
	}
	var result []*entity.Flow
	for _, f := range m.flows {
		if f.TenantID == tenantID {
			result = append(result, f)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockFlowRepo) FindByBot(ctx context.Context, botID string) ([]*entity.Flow, error) {
	return nil, nil
}

func (m *mockFlowRepo) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Flow, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	var result []*entity.Flow
	for _, f := range m.flows {
		if f.TenantID == tenantID && f.IsActive {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockFlowRepo) FindByTrigger(ctx context.Context, tenantID string, trigger entity.FlowTriggerType, triggerValue string) ([]*entity.Flow, error) {
	return nil, nil
}

func (m *mockFlowRepo) Update(ctx context.Context, flow *entity.Flow) error {
	if m.returnError != nil {
		return m.returnError
	}
	m.flows[flow.ID] = flow
	return nil
}

func (m *mockFlowRepo) UpdateStatus(ctx context.Context, id string, isActive bool) error {
	if m.returnError != nil {
		return m.returnError
	}
	f, ok := m.flows[id]
	if !ok {
		return fmt.Errorf("flow not found: %s", id)
	}
	f.IsActive = isActive
	return nil
}

func (m *mockFlowRepo) Delete(ctx context.Context, id string) error {
	if m.returnError != nil {
		return m.returnError
	}
	delete(m.flows, id)
	return nil
}

func (m *mockFlowRepo) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}

func (m *mockFlowRepo) CountByBot(ctx context.Context, botID string) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock ConversationContextRepository
// ---------------------------------------------------------------------------

type mockContextRepo struct {
	contexts    map[string]*entity.ConversationContext
	returnError error
}

func newMockContextRepo() *mockContextRepo {
	return &mockContextRepo{contexts: make(map[string]*entity.ConversationContext)}
}

func (m *mockContextRepo) Create(ctx context.Context, c *entity.ConversationContext) error {
	if m.returnError != nil {
		return m.returnError
	}
	m.contexts[c.ID] = c
	return nil
}

func (m *mockContextRepo) FindByID(ctx context.Context, id string) (*entity.ConversationContext, error) {
	c, ok := m.contexts[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (m *mockContextRepo) FindByConversation(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	for _, c := range m.contexts {
		if c.ConversationID == conversationID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockContextRepo) Update(ctx context.Context, c *entity.ConversationContext) error {
	if m.returnError != nil {
		return m.returnError
	}
	m.contexts[c.ID] = c
	return nil
}

func (m *mockContextRepo) UpdateIntent(ctx context.Context, id string, intent *entity.Intent) error {
	return nil
}

func (m *mockContextRepo) UpdateSentiment(ctx context.Context, id string, sentiment entity.Sentiment) error {
	return nil
}

func (m *mockContextRepo) UpdateContextWindow(ctx context.Context, id string, window []entity.ContextMessage) error {
	return nil
}

func (m *mockContextRepo) Delete(ctx context.Context, id string) error {
	delete(m.contexts, id)
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newFlowEngine() (*FlowEngineService, *mockFlowRepo, *mockContextRepo) {
	flowRepo := newMockFlowRepo()
	contextRepo := newMockContextRepo()
	svc := NewFlowEngineService(flowRepo, contextRepo)
	return svc, flowRepo, contextRepo
}

// makeSimpleFlow creates a flow with: message -> question -> (yes -> end, no -> end)
func makeSimpleFlow(tenantID string) *entity.Flow {
	flow := entity.NewFlow(tenantID, "Test Flow", entity.FlowTriggerKeyword, "help")
	flow.ID = "flow-1"
	flow.IsActive = true
	flow.StartNodeID = "msg-1"
	flow.Nodes = []entity.FlowNode{
		{
			ID:      "msg-1",
			Type:    entity.FlowNodeMessage,
			Content: "Welcome! Do you need assistance?",
			Transitions: []entity.FlowTransition{
				{ID: "t1", ToNodeID: "q-1", Condition: entity.TransitionConditionDefault},
			},
		},
		{
			ID:      "q-1",
			Type:    entity.FlowNodeQuestion,
			Content: "Would you like to continue?",
			QuickReplies: []entity.QuickReply{
				{ID: "yes", Title: "Yes"},
				{ID: "no", Title: "No"},
			},
			Transitions: []entity.FlowTransition{
				{ID: "t2", ToNodeID: "end-yes", Condition: entity.TransitionConditionReplyEquals, Value: "yes"},
				{ID: "t3", ToNodeID: "end-no", Condition: entity.TransitionConditionReplyEquals, Value: "no"},
				{ID: "t4", ToNodeID: "q-1", Condition: entity.TransitionConditionDefault},
			},
		},
		{
			ID:      "end-yes",
			Type:    entity.FlowNodeEnd,
			Content: "Great! Thank you.",
		},
		{
			ID:      "end-no",
			Type:    entity.FlowNodeEnd,
			Content: "No problem. Goodbye!",
		},
	}
	return flow
}

// ---------------------------------------------------------------------------
// CheckTrigger tests
// ---------------------------------------------------------------------------

func TestFlowEngine_CheckTrigger_Keyword(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	matched, triggered := svc.CheckTrigger(context.Background(), "t-1", "I need help please", convCtx)

	assert.True(t, triggered)
	assert.Equal(t, flow.ID, matched.ID)
}

func TestFlowEngine_CheckTrigger_KeywordCaseInsensitive(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	matched, triggered := svc.CheckTrigger(context.Background(), "t-1", "HELP ME", convCtx)

	assert.True(t, triggered)
	assert.NotNil(t, matched)
}

func TestFlowEngine_CheckTrigger_NoMatch(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	_, triggered := svc.CheckTrigger(context.Background(), "t-1", "hello there", convCtx)

	assert.False(t, triggered)
}

func TestFlowEngine_CheckTrigger_ActiveFlowBlocks(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	// Already has an active flow
	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{
			"active_flow_id": "some-flow",
		},
	}

	_, triggered := svc.CheckTrigger(context.Background(), "t-1", "help", convCtx)

	assert.False(t, triggered)
}

func TestFlowEngine_CheckTrigger_Intent(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := entity.NewFlow("t-1", "Intent Flow", entity.FlowTriggerIntent, "support")
	flow.ID = "flow-intent"
	flow.IsActive = true
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{
		State:  make(map[string]interface{}),
		Intent: &entity.Intent{Name: "support"},
	}

	matched, triggered := svc.CheckTrigger(context.Background(), "t-1", "anything", convCtx)

	assert.True(t, triggered)
	assert.Equal(t, flow.ID, matched.ID)
}

func TestFlowEngine_CheckTrigger_Welcome(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := entity.NewFlow("t-1", "Welcome Flow", entity.FlowTriggerWelcome, "")
	flow.ID = "flow-welcome"
	flow.IsActive = true
	flowRepo.flows[flow.ID] = flow

	// New conversation - no state
	convCtx := &entity.ConversationContext{}

	matched, triggered := svc.CheckTrigger(context.Background(), "t-1", "hi", convCtx)

	assert.True(t, triggered)
	assert.Equal(t, flow.ID, matched.ID)
}

func TestFlowEngine_CheckTrigger_WelcomeSkipsExistingState(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := entity.NewFlow("t-1", "Welcome Flow", entity.FlowTriggerWelcome, "")
	flow.ID = "flow-welcome"
	flow.IsActive = true
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{"some_key": "some_value"},
	}

	_, triggered := svc.CheckTrigger(context.Background(), "t-1", "hi", convCtx)

	assert.False(t, triggered)
}

func TestFlowEngine_CheckTrigger_NoFlows(t *testing.T) {
	svc, _, _ := newFlowEngine()
	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	_, triggered := svc.CheckTrigger(context.Background(), "t-1", "help", convCtx)

	assert.False(t, triggered)
}

// ---------------------------------------------------------------------------
// StartFlow tests
// ---------------------------------------------------------------------------

func TestFlowEngine_StartFlow(t *testing.T) {
	svc, _, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	result, err := svc.StartFlow(context.Background(), flow, convCtx)

	require.NoError(t, err)
	assert.Equal(t, "Welcome! Do you need assistance?", result.Message)
	// Message node auto-advances to question node
	assert.Equal(t, "q-1", convCtx.State["current_node_id"])
}

func TestFlowEngine_StartFlow_NoStartNode(t *testing.T) {
	svc, _, _ := newFlowEngine()
	flow := entity.NewFlow("t-1", "Empty", entity.FlowTriggerKeyword, "test")
	flow.ID = "flow-empty"
	flow.StartNodeID = "nonexistent"
	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	_, err := svc.StartFlow(context.Background(), flow, convCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no start node")
}

// ---------------------------------------------------------------------------
// ContinueFlow tests
// ---------------------------------------------------------------------------

func TestFlowEngine_ContinueFlow_ValidTransition(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{
			"active_flow_id":  flow.ID,
			"current_node_id": "q-1",
		},
	}

	result, err := svc.ContinueFlow(context.Background(), "t-1", "yes", convCtx)

	require.NoError(t, err)
	assert.True(t, result.FlowEnded)
	assert.Equal(t, "Great! Thank you.", result.Message)
}

func TestFlowEngine_ContinueFlow_NoReply(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{
			"active_flow_id":  flow.ID,
			"current_node_id": "q-1",
		},
	}

	// "maybe" doesn't match "yes" or "no", but default transition re-asks
	result, err := svc.ContinueFlow(context.Background(), "t-1", "maybe", convCtx)

	require.NoError(t, err)
	assert.False(t, result.FlowEnded)
	assert.Equal(t, "Would you like to continue?", result.Message)
}

func TestFlowEngine_ContinueFlow_NoActiveFlow(t *testing.T) {
	svc, _, _ := newFlowEngine()
	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{},
	}

	_, err := svc.ContinueFlow(context.Background(), "t-1", "yes", convCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active flow")
}

func TestFlowEngine_ContinueFlow_FlowDeleted(t *testing.T) {
	svc, _, _ := newFlowEngine()
	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{
			"active_flow_id":  "deleted-flow",
			"current_node_id": "node-1",
		},
	}

	_, err := svc.ContinueFlow(context.Background(), "t-1", "yes", convCtx)

	assert.Error(t, err)
	// State should be cleared
	_, hasFlow := convCtx.State["active_flow_id"]
	assert.False(t, hasFlow)
}

// ---------------------------------------------------------------------------
// ProcessTransition tests
// ---------------------------------------------------------------------------

func TestFlowEngine_ProcessTransition_ReplyEquals(t *testing.T) {
	svc, _, _ := newFlowEngine()
	node := &entity.FlowNode{
		ID:   "q-1",
		Type: entity.FlowNodeQuestion,
		Transitions: []entity.FlowTransition{
			{ToNodeID: "n-yes", Condition: entity.TransitionConditionReplyEquals, Value: "yes"},
			{ToNodeID: "n-no", Condition: entity.TransitionConditionReplyEquals, Value: "no"},
		},
	}

	assert.Equal(t, "n-yes", svc.ProcessTransition(node, "yes"))
	assert.Equal(t, "n-yes", svc.ProcessTransition(node, "YES"))
	assert.Equal(t, "n-no", svc.ProcessTransition(node, "No"))
	assert.Equal(t, "", svc.ProcessTransition(node, "maybe"))
}

func TestFlowEngine_ProcessTransition_Contains(t *testing.T) {
	svc, _, _ := newFlowEngine()
	node := &entity.FlowNode{
		ID: "q-1",
		Transitions: []entity.FlowTransition{
			{ToNodeID: "n-support", Condition: entity.TransitionConditionContains, Value: "support"},
		},
	}

	assert.Equal(t, "n-support", svc.ProcessTransition(node, "I need support please"))
	assert.Equal(t, "", svc.ProcessTransition(node, "hello"))
}

func TestFlowEngine_ProcessTransition_Regex(t *testing.T) {
	svc, _, _ := newFlowEngine()
	node := &entity.FlowNode{
		ID: "q-1",
		Transitions: []entity.FlowTransition{
			{ToNodeID: "n-email", Condition: entity.TransitionConditionRegex, Value: `\S+@\S+\.\S+`},
			{ToNodeID: "n-default", Condition: entity.TransitionConditionDefault},
		},
	}

	assert.Equal(t, "n-email", svc.ProcessTransition(node, "user@example.com"))
	assert.Equal(t, "n-default", svc.ProcessTransition(node, "not an email"))
}

func TestFlowEngine_ProcessTransition_Default(t *testing.T) {
	svc, _, _ := newFlowEngine()
	node := &entity.FlowNode{
		ID: "q-1",
		Transitions: []entity.FlowTransition{
			{ToNodeID: "n-yes", Condition: entity.TransitionConditionReplyEquals, Value: "yes"},
			{ToNodeID: "n-fallback", Condition: entity.TransitionConditionDefault},
		},
	}

	assert.Equal(t, "n-fallback", svc.ProcessTransition(node, "anything"))
}

func TestFlowEngine_ProcessTransition_QuickReplyID(t *testing.T) {
	svc, _, _ := newFlowEngine()
	node := &entity.FlowNode{
		ID: "q-1",
		QuickReplies: []entity.QuickReply{
			{ID: "opt_1", Title: "Option One"},
			{ID: "opt_2", Title: "Option Two"},
		},
		Transitions: []entity.FlowTransition{
			{ToNodeID: "n-1", Condition: entity.TransitionConditionReplyEquals, Value: "opt_1"},
			{ToNodeID: "n-2", Condition: entity.TransitionConditionReplyEquals, Value: "opt_2"},
		},
	}

	// User sends the button title, transition matches by quick reply mapping
	assert.Equal(t, "n-1", svc.ProcessTransition(node, "Option One"))
}

func TestFlowEngine_ProcessTransition_NoTransitions(t *testing.T) {
	svc, _, _ := newFlowEngine()
	node := &entity.FlowNode{ID: "q-1"}

	assert.Equal(t, "", svc.ProcessTransition(node, "anything"))
}

// ---------------------------------------------------------------------------
// HasActiveFlow / GetActiveFlowID / ClearFlowState
// ---------------------------------------------------------------------------

func TestFlowEngine_HasActiveFlow(t *testing.T) {
	svc, _, _ := newFlowEngine()

	assert.False(t, svc.HasActiveFlow(nil))
	assert.False(t, svc.HasActiveFlow(&entity.ConversationContext{}))
	assert.False(t, svc.HasActiveFlow(&entity.ConversationContext{State: map[string]interface{}{}}))
	assert.True(t, svc.HasActiveFlow(&entity.ConversationContext{
		State: map[string]interface{}{"active_flow_id": "flow-1"},
	}))
}

func TestFlowEngine_GetActiveFlowID(t *testing.T) {
	svc, _, _ := newFlowEngine()

	assert.Equal(t, "", svc.GetActiveFlowID(nil))
	assert.Equal(t, "flow-1", svc.GetActiveFlowID(&entity.ConversationContext{
		State: map[string]interface{}{"active_flow_id": "flow-1"},
	}))
}

func TestFlowEngine_ClearFlowState(t *testing.T) {
	svc, _, _ := newFlowEngine()
	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{
			"active_flow_id":  "flow-1",
			"current_node_id": "node-1",
			"flow_started_at": "2024-01-01",
			"collected_data":  map[string]string{"name": "John"},
		},
	}

	svc.ClearFlowState(convCtx)

	assert.Empty(t, convCtx.State["active_flow_id"])
	assert.Empty(t, convCtx.State["current_node_id"])
	assert.Empty(t, convCtx.State["flow_started_at"])
	// collected_data should be preserved
	assert.NotNil(t, convCtx.State["collected_data"])
}

// ---------------------------------------------------------------------------
// StoreCollectedData / GetCollectedData
// ---------------------------------------------------------------------------

func TestFlowEngine_CollectedData(t *testing.T) {
	svc, _, _ := newFlowEngine()
	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{},
	}

	svc.StoreCollectedData(convCtx, "name", "John")
	svc.StoreCollectedData(convCtx, "email", "john@example.com")

	data := svc.GetCollectedData(convCtx)
	assert.Equal(t, "John", data["name"])
	assert.Equal(t, "john@example.com", data["email"])
}

func TestFlowEngine_GetCollectedData_NilContext(t *testing.T) {
	svc, _, _ := newFlowEngine()

	assert.Nil(t, svc.GetCollectedData(nil))
	assert.Nil(t, svc.GetCollectedData(&entity.ConversationContext{}))
}

// ---------------------------------------------------------------------------
// ProcessTemplate
// ---------------------------------------------------------------------------

func TestFlowEngine_ProcessTemplate(t *testing.T) {
	svc, _, _ := newFlowEngine()
	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{
			"collected_data": map[string]string{
				"name":  "Maria",
				"email": "maria@example.com",
			},
		},
		Entities: map[string]interface{}{
			"city": "São Paulo",
		},
	}

	tests := []struct {
		template string
		expected string
	}{
		{"Hello {{name}}, welcome!", "Hello Maria, welcome!"},
		{"Your email is {{email}}", "Your email is maria@example.com"},
		{"You are from {{entity.city}}", "You are from São Paulo"},
		{"No placeholders", "No placeholders"},
		{"", ""},
		{"Unknown {{missing}}", "Unknown {{missing}}"},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			assert.Equal(t, tt.expected, svc.ProcessTemplate(tt.template, convCtx))
		})
	}
}

func TestFlowEngine_ProcessTemplate_NilContext(t *testing.T) {
	svc, _, _ := newFlowEngine()
	assert.Equal(t, "Hello {{name}}", svc.ProcessTemplate("Hello {{name}}", nil))
}

// ---------------------------------------------------------------------------
// ExecuteNode tests
// ---------------------------------------------------------------------------

func TestFlowEngine_ExecuteNode_Message(t *testing.T) {
	svc, _, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}
	node := flow.GetNode("msg-1")

	result, err := svc.ExecuteNode(context.Background(), flow, node, convCtx, "")

	require.NoError(t, err)
	assert.Equal(t, "Welcome! Do you need assistance?", result.Message)
	assert.Equal(t, "q-1", result.NextNodeID)
}

func TestFlowEngine_ExecuteNode_Question(t *testing.T) {
	svc, _, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}
	node := flow.GetNode("q-1")

	result, err := svc.ExecuteNode(context.Background(), flow, node, convCtx, "")

	require.NoError(t, err)
	assert.Equal(t, "Would you like to continue?", result.Message)
	assert.True(t, result.ShouldWait)
	assert.Len(t, result.QuickReplies, 2)
}

func TestFlowEngine_ExecuteNode_End(t *testing.T) {
	svc, _, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	convCtx := &entity.ConversationContext{
		State: map[string]interface{}{
			"active_flow_id":  flow.ID,
			"current_node_id": "end-yes",
		},
	}
	node := flow.GetNode("end-yes")

	result, err := svc.ExecuteNode(context.Background(), flow, node, convCtx, "")

	require.NoError(t, err)
	assert.True(t, result.FlowEnded)
	assert.Equal(t, "Great! Thank you.", result.Message)
	// Flow state should be cleared
	assert.Empty(t, convCtx.State["active_flow_id"])
}

func TestFlowEngine_ExecuteNode_Action(t *testing.T) {
	svc, _, _ := newFlowEngine()
	flow := &entity.Flow{
		ID:          "flow-1",
		StartNodeID: "act-1",
		Nodes: []entity.FlowNode{
			{
				ID:      "act-1",
				Type:    entity.FlowNodeAction,
				Content: "Processing your request...",
				Actions: []entity.FlowAction{
					{Type: entity.FlowActionTag, Config: map[string]interface{}{"tag": "vip"}},
				},
				Transitions: []entity.FlowTransition{
					{ToNodeID: "end-1", Condition: entity.TransitionConditionDefault},
				},
			},
			{ID: "end-1", Type: entity.FlowNodeEnd, Content: "Done."},
		},
	}
	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}
	node := flow.GetNode("act-1")

	result, err := svc.ExecuteNode(context.Background(), flow, node, convCtx, "")

	require.NoError(t, err)
	assert.Equal(t, "Processing your request...", result.Message)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, entity.FlowActionTag, result.Actions[0].Type)
}

func TestFlowEngine_ExecuteNode_QuestionCollectsData(t *testing.T) {
	svc, _, _ := newFlowEngine()
	flow := &entity.Flow{
		ID:          "flow-1",
		StartNodeID: "q-name",
		Nodes: []entity.FlowNode{
			{
				ID:      "q-name",
				Type:    entity.FlowNodeQuestion,
				Content: "What is your name?",
			},
		},
	}
	convCtx := &entity.ConversationContext{State: map[string]interface{}{}}
	node := flow.GetNode("q-name")

	result, err := svc.ExecuteNode(context.Background(), flow, node, convCtx, "John")

	require.NoError(t, err)
	assert.True(t, result.ShouldWait)
	// User input should be stored as collected data
	data := svc.GetCollectedData(convCtx)
	assert.Equal(t, "John", data["q-name"])
}

// ---------------------------------------------------------------------------
// FlowService CRUD tests
// ---------------------------------------------------------------------------

func TestFlowService_Create(t *testing.T) {
	repo := newMockFlowRepo()
	svc := NewFlowService(repo)

	input := &entity.CreateFlowInput{
		Name:         "Order Flow",
		Trigger:      entity.FlowTriggerKeyword,
		TriggerValue: "order",
		StartNodeID:  "start",
		Nodes: []entity.FlowNode{
			{ID: "start", Type: entity.FlowNodeMessage, Content: "Welcome to ordering"},
		},
	}

	flow, err := svc.Create(context.Background(), "t-1", input)

	require.NoError(t, err)
	assert.NotEmpty(t, flow.ID)
	assert.Equal(t, "Order Flow", flow.Name)
	assert.Equal(t, "t-1", flow.TenantID)
}

func TestFlowService_Create_NoNodes(t *testing.T) {
	repo := newMockFlowRepo()
	svc := NewFlowService(repo)

	input := &entity.CreateFlowInput{
		Name:        "Empty Flow",
		StartNodeID: "start",
		Nodes:       []entity.FlowNode{},
	}

	_, err := svc.Create(context.Background(), "t-1", input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one node")
}

func TestFlowService_Create_StartNodeNotFound(t *testing.T) {
	repo := newMockFlowRepo()
	svc := NewFlowService(repo)

	input := &entity.CreateFlowInput{
		Name:        "Bad Flow",
		StartNodeID: "nonexistent",
		Nodes: []entity.FlowNode{
			{ID: "node-1", Type: entity.FlowNodeMessage},
		},
	}

	_, err := svc.Create(context.Background(), "t-1", input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start node not found")
}

func TestFlowService_Update(t *testing.T) {
	repo := newMockFlowRepo()
	svc := NewFlowService(repo)

	flow := entity.NewFlow("t-1", "Original", entity.FlowTriggerKeyword, "test")
	flow.ID = "flow-1"
	repo.flows[flow.ID] = flow

	newName := "Updated"
	_, err := svc.Update(context.Background(), "flow-1", &entity.UpdateFlowInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, "Updated", repo.flows["flow-1"].Name)
}

func TestFlowService_Delete(t *testing.T) {
	repo := newMockFlowRepo()
	svc := NewFlowService(repo)

	flow := entity.NewFlow("t-1", "To Delete", entity.FlowTriggerKeyword, "test")
	flow.ID = "flow-1"
	repo.flows[flow.ID] = flow

	err := svc.Delete(context.Background(), "flow-1")

	require.NoError(t, err)
	_, exists := repo.flows["flow-1"]
	assert.False(t, exists)
}

func TestFlowService_ActivateDeactivate(t *testing.T) {
	repo := newMockFlowRepo()
	svc := NewFlowService(repo)

	flow := entity.NewFlow("t-1", "Toggle", entity.FlowTriggerKeyword, "test")
	flow.ID = "flow-1"
	repo.flows[flow.ID] = flow

	err := svc.Activate(context.Background(), "flow-1")
	require.NoError(t, err)
	assert.True(t, repo.flows["flow-1"].IsActive)

	err = svc.Deactivate(context.Background(), "flow-1")
	require.NoError(t, err)
	assert.False(t, repo.flows["flow-1"].IsActive)
}

// ---------------------------------------------------------------------------
// Full flow execution (integration)
// ---------------------------------------------------------------------------

func TestFlowEngine_FullExecution(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	// 1. Start flow
	result, err := svc.StartFlow(context.Background(), flow, convCtx)
	require.NoError(t, err)
	assert.Equal(t, "Welcome! Do you need assistance?", result.Message)

	// 2. Continue with "yes"
	result, err = svc.ContinueFlow(context.Background(), "t-1", "yes", convCtx)
	require.NoError(t, err)
	assert.True(t, result.FlowEnded)
	assert.Equal(t, "Great! Thank you.", result.Message)

	// 3. Flow state should be cleared
	assert.False(t, svc.HasActiveFlow(convCtx))
}

func TestFlowEngine_FullExecution_NoBranch(t *testing.T) {
	svc, flowRepo, _ := newFlowEngine()
	flow := makeSimpleFlow("t-1")
	flowRepo.flows[flow.ID] = flow

	convCtx := &entity.ConversationContext{State: make(map[string]interface{})}

	// Start flow
	_, err := svc.StartFlow(context.Background(), flow, convCtx)
	require.NoError(t, err)

	// Continue with "no"
	result, err := svc.ContinueFlow(context.Background(), "t-1", "no", convCtx)
	require.NoError(t, err)
	assert.True(t, result.FlowEnded)
	assert.Equal(t, "No problem. Goodbye!", result.Message)
}
