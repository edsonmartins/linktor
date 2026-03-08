package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
)

// ============================================================================
// Mock BotRepository
// ============================================================================

type mockBotRepository struct {
	Bots        map[string]*entity.Bot
	ReturnError error
}

func newMockBotRepository() *mockBotRepository {
	return &mockBotRepository{Bots: make(map[string]*entity.Bot)}
}

func (m *mockBotRepository) Create(_ context.Context, bot *entity.Bot) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Bots[bot.ID] = bot
	return nil
}

func (m *mockBotRepository) FindByID(_ context.Context, id string) (*entity.Bot, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	bot, ok := m.Bots[id]
	if !ok {
		return nil, fmt.Errorf("bot not found: %s", id)
	}
	return bot, nil
}

func (m *mockBotRepository) FindByTenant(_ context.Context, _ string, _ *repository.ListParams) ([]*entity.Bot, int64, error) {
	return nil, 0, m.ReturnError
}

func (m *mockBotRepository) FindByChannel(_ context.Context, _ string) (*entity.Bot, error) {
	return nil, m.ReturnError
}

func (m *mockBotRepository) FindActiveByTenant(_ context.Context, _ string) ([]*entity.Bot, error) {
	return nil, m.ReturnError
}

func (m *mockBotRepository) Update(_ context.Context, bot *entity.Bot) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Bots[bot.ID] = bot
	return nil
}

func (m *mockBotRepository) UpdateStatus(_ context.Context, id string, status entity.BotStatus) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	bot, ok := m.Bots[id]
	if !ok {
		return fmt.Errorf("bot not found: %s", id)
	}
	bot.Status = status
	return nil
}

func (m *mockBotRepository) Delete(_ context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Bots, id)
	return nil
}

func (m *mockBotRepository) CountByTenant(_ context.Context, _ string) (int64, error) {
	return 0, m.ReturnError
}

func (m *mockBotRepository) AssignChannel(_ context.Context, _, _ string) error {
	return m.ReturnError
}

func (m *mockBotRepository) UnassignChannel(_ context.Context, _, _ string) error {
	return m.ReturnError
}

// ============================================================================
// Mock ConversationContextRepository
// ============================================================================

type mockConversationContextRepository struct {
	Contexts    map[string]*entity.ConversationContext
	ReturnError error
}

func newMockConversationContextRepository() *mockConversationContextRepository {
	return &mockConversationContextRepository{
		Contexts: make(map[string]*entity.ConversationContext),
	}
}

func (m *mockConversationContextRepository) Create(_ context.Context, c *entity.ConversationContext) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Contexts[c.ConversationID] = c
	return nil
}

func (m *mockConversationContextRepository) FindByID(_ context.Context, id string) (*entity.ConversationContext, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, c := range m.Contexts {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, fmt.Errorf("context not found: %s", id)
}

func (m *mockConversationContextRepository) FindByConversation(_ context.Context, conversationID string) (*entity.ConversationContext, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	c, ok := m.Contexts[conversationID]
	if !ok {
		return nil, fmt.Errorf("context not found for conversation: %s", conversationID)
	}
	return c, nil
}

func (m *mockConversationContextRepository) Update(_ context.Context, c *entity.ConversationContext) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Contexts[c.ConversationID] = c
	return nil
}

func (m *mockConversationContextRepository) UpdateIntent(_ context.Context, _ string, _ *entity.Intent) error {
	return m.ReturnError
}

func (m *mockConversationContextRepository) UpdateSentiment(_ context.Context, _ string, _ entity.Sentiment) error {
	return m.ReturnError
}

func (m *mockConversationContextRepository) UpdateContextWindow(_ context.Context, _ string, _ []entity.ContextMessage) error {
	return m.ReturnError
}

func (m *mockConversationContextRepository) Delete(_ context.Context, _ string) error {
	return m.ReturnError
}

// ============================================================================
// Helper to build the use case with all mocks
// ============================================================================

type escalateTestDeps struct {
	conversationRepo *testutil.MockConversationRepository
	messageRepo      *testutil.MockMessageRepository
	contactRepo      *testutil.MockContactRepository
	channelRepo      *testutil.MockChannelRepository
	botRepo          *mockBotRepository
	userRepo         *testutil.MockUserRepository
	contextRepo      *mockConversationContextRepository
	producer         *testutil.MockProducer
	uc               *EscalateConversationUseCase
}

func setupEscalateTest() *escalateTestDeps {
	d := &escalateTestDeps{
		conversationRepo: testutil.NewMockConversationRepository(),
		messageRepo:      testutil.NewMockMessageRepository(),
		contactRepo:      testutil.NewMockContactRepository(),
		channelRepo:      testutil.NewMockChannelRepository(),
		botRepo:          newMockBotRepository(),
		userRepo:         testutil.NewMockUserRepository(),
		contextRepo:      newMockConversationContextRepository(),
		producer:         testutil.NewMockProducer(),
	}
	d.uc = NewEscalateConversationUseCase(
		d.conversationRepo,
		d.messageRepo,
		d.contactRepo,
		d.channelRepo,
		d.botRepo,
		d.userRepo,
		d.contextRepo,
		nil, // aiFactory not needed
		d.producer,
	)
	return d
}

func makeConversation(id, tenantID, channelID string) *entity.Conversation {
	now := time.Now()
	return &entity.Conversation{
		ID:        id,
		TenantID:  tenantID,
		ContactID: "contact-1",
		ChannelID: channelID,
		Status:    entity.ConversationStatusOpen,
		Priority:  entity.ConversationPriorityNormal,
		Metadata:  make(map[string]string),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func makeAgent(id, tenantID string) *entity.User {
	return &entity.User{
		ID:       id,
		TenantID: tenantID,
		Email:    id + "@test.com",
		Name:     "Agent " + id,
		Role:     entity.UserRoleAgent,
		Status:   "active",
	}
}

// ============================================================================
// Execute tests
// ============================================================================

func TestEscalateConversation_HappyPath_AutoAssign(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	agent1 := makeAgent("agent-1", "tenant-1")
	agent2 := makeAgent("agent-2", "tenant-1")
	d.userRepo.Users["agent-1"] = agent1
	d.userRepo.Users["agent-2"] = agent2

	// Give agent-1 two active conversations so agent-2 has lower workload
	assigned1 := "agent-1"
	d.conversationRepo.Conversations["active-1"] = &entity.Conversation{
		ID: "active-1", TenantID: "tenant-1", ChannelID: "channel-1",
		Status: entity.ConversationStatusOpen, AssignedUserID: &assigned1,
	}
	d.conversationRepo.Conversations["active-2"] = &entity.Conversation{
		ID: "active-2", TenantID: "tenant-1", ChannelID: "channel-1",
		Status: entity.ConversationStatusOpen, AssignedUserID: &assigned1,
	}

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		BotID:          "bot-1",
		Reason:         "user needs help",
		Priority:       "high",
		RequestedBy:    "bot",
	}

	output, err := d.uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if output.Status != "assigned" {
		t.Errorf("expected status 'assigned', got '%s'", output.Status)
	}
	if output.AssignedUserID != "agent-2" {
		t.Errorf("expected assignment to agent-2 (lowest workload), got '%s'", output.AssignedUserID)
	}
	if output.QueuePosition != 0 {
		t.Errorf("expected queue position 0 when assigned, got %d", output.QueuePosition)
	}
}

func TestEscalateConversation_AlreadyAssigned(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	assigned := "agent-1"
	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	conv.AssignedUserID = &assigned
	d.conversationRepo.Conversations["conv-1"] = conv

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		Reason:         "help",
		Priority:       "normal",
		RequestedBy:    "bot",
	}

	output, err := d.uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.Status != "already_assigned" {
		t.Errorf("expected status 'already_assigned', got '%s'", output.Status)
	}
	if output.AssignedUserID != "agent-1" {
		t.Errorf("expected assigned user 'agent-1', got '%s'", output.AssignedUserID)
	}
}

func TestEscalateConversation_TenantMismatch(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-other",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		Reason:         "help",
		Priority:       "normal",
		RequestedBy:    "bot",
	}

	_, err := d.uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error for tenant mismatch, got nil")
	}
}

func TestEscalateConversation_ConversationNotFound(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	input := &EscalateConversationInput{
		ConversationID: "nonexistent",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		Reason:         "help",
		Priority:       "normal",
		RequestedBy:    "bot",
	}

	_, err := d.uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error for conversation not found, got nil")
	}
}

func TestEscalateConversation_NoAvailableAgents_Queued(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv
	// No agents added to userRepo

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		Reason:         "help",
		Priority:       "normal",
		RequestedBy:    "bot",
	}

	output, err := d.uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.Status != "queued" {
		t.Errorf("expected status 'queued', got '%s'", output.Status)
	}
	if output.QueuePosition < 1 {
		t.Errorf("expected queue position >= 1, got %d", output.QueuePosition)
	}
	// EstimatedWait = queuePosition * 120
	if output.EstimatedWait != output.QueuePosition*120 {
		t.Errorf("expected estimated wait = %d, got %d", output.QueuePosition*120, output.EstimatedWait)
	}
}

func TestEscalateConversation_PriorityMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected entity.ConversationPriority
	}{
		{"urgent", entity.ConversationPriorityUrgent},
		{"high", entity.ConversationPriorityHigh},
		{"low", entity.ConversationPriorityLow},
		{"normal", entity.ConversationPriorityNormal},
		{"", entity.ConversationPriorityNormal},
		{"unknown", entity.ConversationPriorityNormal},
	}

	for _, tt := range tests {
		t.Run("priority_"+tt.input, func(t *testing.T) {
			got := mapPriority(tt.input)
			if got != tt.expected {
				t.Errorf("mapPriority(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscalateConversation_MetadataSetCorrectly(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		BotID:          "bot-42",
		Reason:         "customer is upset",
		Priority:       "high",
		RequestedBy:    "bot",
	}

	_, err := d.uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	updated := d.conversationRepo.Conversations["conv-1"]
	if updated.Metadata["escalation_reason"] != "customer is upset" {
		t.Errorf("expected escalation_reason = 'customer is upset', got '%s'", updated.Metadata["escalation_reason"])
	}
	if updated.Metadata["escalated_by"] != "bot" {
		t.Errorf("expected escalated_by = 'bot', got '%s'", updated.Metadata["escalated_by"])
	}
	if updated.Metadata["escalated_at"] == "" {
		t.Error("expected escalated_at to be set")
	}
	if updated.Metadata["escalated_from_bot"] != "bot-42" {
		t.Errorf("expected escalated_from_bot = 'bot-42', got '%s'", updated.Metadata["escalated_from_bot"])
	}
}

func TestEscalateConversation_MetadataNoBot(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		BotID:          "", // no bot
		Reason:         "manual escalation",
		Priority:       "normal",
		RequestedBy:    "user",
	}

	_, err := d.uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	updated := d.conversationRepo.Conversations["conv-1"]
	if _, exists := updated.Metadata["escalated_from_bot"]; exists {
		t.Error("expected escalated_from_bot to not be set when BotID is empty")
	}
}

func TestEscalateConversation_EventPublished(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	agent := makeAgent("agent-1", "tenant-1")
	d.userRepo.Users["agent-1"] = agent

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		BotID:          "bot-1",
		Reason:         "needs help",
		Priority:       "normal",
		RequestedBy:    "bot",
	}

	_, err := d.uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(d.producer.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(d.producer.Events))
	}

	event := d.producer.Events[0]
	if event.Type != nats.EventConversationEscalated {
		t.Errorf("expected event type '%s', got '%s'", nats.EventConversationEscalated, event.Type)
	}
	if event.TenantID != "tenant-1" {
		t.Errorf("expected tenant_id 'tenant-1', got '%s'", event.TenantID)
	}

	payload := event.Payload
	if payload["conversation_id"] != "conv-1" {
		t.Errorf("expected conversation_id 'conv-1' in payload, got '%v'", payload["conversation_id"])
	}
	if payload["reason"] != "needs help" {
		t.Errorf("expected reason 'needs help' in payload, got '%v'", payload["reason"])
	}
	if payload["requested_by"] != "bot" {
		t.Errorf("expected requested_by 'bot' in payload, got '%v'", payload["requested_by"])
	}
	if payload["bot_id"] != "bot-1" {
		t.Errorf("expected bot_id 'bot-1' in payload, got '%v'", payload["bot_id"])
	}
	// Agent was assigned, so assigned_user_id should be present
	if payload["assigned_user_id"] == nil || payload["assigned_user_id"] == "" {
		t.Error("expected assigned_user_id in payload when agent is assigned")
	}
}

func TestEscalateConversation_EventPublished_NoBotID(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	input := &EscalateConversationInput{
		ConversationID: "conv-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ContactID:      "contact-1",
		BotID:          "",
		Reason:         "manual",
		Priority:       "normal",
		RequestedBy:    "user",
	}

	_, err := d.uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(d.producer.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(d.producer.Events))
	}

	payload := d.producer.Events[0].Payload
	if _, ok := payload["bot_id"]; ok {
		t.Error("expected bot_id to not be present in payload when BotID is empty")
	}
}

// ============================================================================
// containsUrgentKeywords tests
// ============================================================================

func TestEscalate_ContainsUrgentKeywords(t *testing.T) {
	positives := []string{
		"this is urgent",
		"urgente por favor",
		"emergency situation",
		"emergência total",
		"I have a complaint",
		"reclamação grave",
		"customer is angry",
		"muita raiva",
		"critical issue",
		"problema crítico",
	}

	for _, s := range positives {
		t.Run("positive_"+s, func(t *testing.T) {
			if !containsUrgentKeywords(s) {
				t.Errorf("containsUrgentKeywords(%q) = false, want true", s)
			}
		})
	}

	negatives := []string{
		"hello there",
		"normal question",
		"I need help with my order",
		"what is the status",
		"",
	}

	for _, s := range negatives {
		t.Run("negative_"+s, func(t *testing.T) {
			if containsUrgentKeywords(s) {
				t.Errorf("containsUrgentKeywords(%q) = true, want false", s)
			}
		})
	}
}

// ============================================================================
// EscalateFromBot tests
// ============================================================================

func TestEscalateFromBot_UrgentKeyword_SetsHighPriority(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	output, err := d.uc.EscalateFromBot(ctx, "conv-1", "tenant-1", "channel-1", "contact-1", "bot-1", "customer is angry and upset")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// The conversation should have been updated with high priority
	updated := d.conversationRepo.Conversations["conv-1"]
	if updated.Priority != entity.ConversationPriorityHigh {
		t.Errorf("expected priority 'high' for urgent reason, got '%s'", updated.Priority)
	}
	_ = output
}

func TestEscalateFromBot_NormalReason_SetsNormalPriority(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	_, err := d.uc.EscalateFromBot(ctx, "conv-1", "tenant-1", "channel-1", "contact-1", "bot-1", "user needs help with order")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	updated := d.conversationRepo.Conversations["conv-1"]
	if updated.Priority != entity.ConversationPriorityNormal {
		t.Errorf("expected priority 'normal' for normal reason, got '%s'", updated.Priority)
	}
}

func TestEscalateFromBot_SetsRequestedByBot(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	_, err := d.uc.EscalateFromBot(ctx, "conv-1", "tenant-1", "channel-1", "contact-1", "bot-1", "some reason")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	updated := d.conversationRepo.Conversations["conv-1"]
	if updated.Metadata["escalated_by"] != "bot" {
		t.Errorf("expected escalated_by = 'bot', got '%s'", updated.Metadata["escalated_by"])
	}
	if updated.Metadata["escalated_from_bot"] != "bot-1" {
		t.Errorf("expected escalated_from_bot = 'bot-1', got '%s'", updated.Metadata["escalated_from_bot"])
	}
}

// ============================================================================
// EscalateFromUser tests
// ============================================================================

func TestEscalateFromUser_DefaultPriority(t *testing.T) {
	d := setupEscalateTest()
	ctx := context.Background()

	conv := makeConversation("conv-1", "tenant-1", "channel-1")
	d.conversationRepo.Conversations["conv-1"] = conv

	_, err := d.uc.EscalateFromUser(ctx, "conv-1", "tenant-1", "channel-1", "contact-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	updated := d.conversationRepo.Conversations["conv-1"]
	if updated.Priority != entity.ConversationPriorityNormal {
		t.Errorf("expected priority 'normal', got '%s'", updated.Priority)
	}
	if updated.Metadata["escalated_by"] != "user" {
		t.Errorf("expected escalated_by = 'user', got '%s'", updated.Metadata["escalated_by"])
	}
	if updated.Metadata["escalation_reason"] != "User requested human assistance" {
		t.Errorf("expected escalation_reason = 'User requested human assistance', got '%s'", updated.Metadata["escalation_reason"])
	}
}

// ============================================================================
// mapPriority exhaustive tests
// ============================================================================

func TestEscalate_MapPriority(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected entity.ConversationPriority
	}{
		{"urgent", "urgent", entity.ConversationPriorityUrgent},
		{"high", "high", entity.ConversationPriorityHigh},
		{"low", "low", entity.ConversationPriorityLow},
		{"normal_explicit", "normal", entity.ConversationPriorityNormal},
		{"empty_string", "", entity.ConversationPriorityNormal},
		{"garbage", "xyz", entity.ConversationPriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapPriority(tt.input)
			if got != tt.expected {
				t.Errorf("mapPriority(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
