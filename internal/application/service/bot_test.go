package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/testutil"
)

// ============================================================================
// MockBotRepository
// ============================================================================

type MockBotRepository struct {
	Bots            map[string]*entity.Bot
	ChannelBotMap   map[string]string // channelID -> botID
	StatusUpdates   map[string]entity.BotStatus
	DeletedIDs      []string
	ReturnError     error
	FindByChannelFn func(ctx context.Context, channelID string) (*entity.Bot, error)
}

func NewMockBotRepository() *MockBotRepository {
	return &MockBotRepository{
		Bots:          make(map[string]*entity.Bot),
		ChannelBotMap: make(map[string]string),
		StatusUpdates: make(map[string]entity.BotStatus),
	}
}

func (m *MockBotRepository) Create(ctx context.Context, bot *entity.Bot) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Bots[bot.ID] = bot
	return nil
}

func (m *MockBotRepository) FindByID(ctx context.Context, id string) (*entity.Bot, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	bot, ok := m.Bots[id]
	if !ok {
		return nil, fmt.Errorf("bot not found: %s", id)
	}
	return bot, nil
}

func (m *MockBotRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Bot, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Bot
	for _, b := range m.Bots {
		if b.TenantID == tenantID {
			result = append(result, b)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockBotRepository) FindByChannel(ctx context.Context, channelID string) (*entity.Bot, error) {
	if m.FindByChannelFn != nil {
		return m.FindByChannelFn(ctx, channelID)
	}
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	botID, ok := m.ChannelBotMap[channelID]
	if !ok {
		return nil, fmt.Errorf("no bot for channel: %s", channelID)
	}
	bot, ok := m.Bots[botID]
	if !ok {
		return nil, fmt.Errorf("bot not found: %s", botID)
	}
	return bot, nil
}

func (m *MockBotRepository) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Bot, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Bot
	for _, b := range m.Bots {
		if b.TenantID == tenantID && b.Status == entity.BotStatusActive {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *MockBotRepository) Update(ctx context.Context, bot *entity.Bot) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Bots[bot.ID] = bot
	return nil
}

func (m *MockBotRepository) UpdateStatus(ctx context.Context, id string, status entity.BotStatus) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	bot, ok := m.Bots[id]
	if !ok {
		return fmt.Errorf("bot not found: %s", id)
	}
	bot.Status = status
	m.StatusUpdates[id] = status
	return nil
}

func (m *MockBotRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Bots, id)
	m.DeletedIDs = append(m.DeletedIDs, id)
	return nil
}

func (m *MockBotRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, b := range m.Bots {
		if b.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockBotRepository) AssignChannel(ctx context.Context, botID, channelID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.ChannelBotMap[channelID] = botID
	if bot, ok := m.Bots[botID]; ok {
		bot.AssignChannel(channelID)
	}
	return nil
}

func (m *MockBotRepository) UnassignChannel(ctx context.Context, botID, channelID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.ChannelBotMap, channelID)
	if bot, ok := m.Bots[botID]; ok {
		bot.UnassignChannel(channelID)
	}
	return nil
}

// ============================================================================
// MockConversationContextRepository
// ============================================================================

type MockConversationContextRepository struct {
	Contexts    map[string]*entity.ConversationContext
	ReturnError error
}

func NewMockConversationContextRepository() *MockConversationContextRepository {
	return &MockConversationContextRepository{
		Contexts: make(map[string]*entity.ConversationContext),
	}
}

func (m *MockConversationContextRepository) Create(ctx context.Context, convContext *entity.ConversationContext) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Contexts[convContext.ID] = convContext
	return nil
}

func (m *MockConversationContextRepository) FindByID(ctx context.Context, id string) (*entity.ConversationContext, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	c, ok := m.Contexts[id]
	if !ok {
		return nil, fmt.Errorf("context not found: %s", id)
	}
	return c, nil
}

func (m *MockConversationContextRepository) FindByConversation(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, c := range m.Contexts {
		if c.ConversationID == conversationID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("context not found for conversation: %s", conversationID)
}

func (m *MockConversationContextRepository) Update(ctx context.Context, convContext *entity.ConversationContext) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Contexts[convContext.ID] = convContext
	return nil
}

func (m *MockConversationContextRepository) UpdateIntent(ctx context.Context, id string, intent *entity.Intent) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	c, ok := m.Contexts[id]
	if !ok {
		return fmt.Errorf("context not found: %s", id)
	}
	c.Intent = intent
	return nil
}

func (m *MockConversationContextRepository) UpdateSentiment(ctx context.Context, id string, sentiment entity.Sentiment) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	c, ok := m.Contexts[id]
	if !ok {
		return fmt.Errorf("context not found: %s", id)
	}
	c.Sentiment = sentiment
	return nil
}

func (m *MockConversationContextRepository) UpdateContextWindow(ctx context.Context, id string, window []entity.ContextMessage) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	c, ok := m.Contexts[id]
	if !ok {
		return fmt.Errorf("context not found: %s", id)
	}
	c.ContextWindow = window
	return nil
}

func (m *MockConversationContextRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Contexts, id)
	return nil
}

// ============================================================================
// Mock AIProvider for the factory
// ============================================================================

type mockAIProvider struct {
	name      entity.AIProviderType
	available bool
}

func (m *mockAIProvider) Name() entity.AIProviderType                                              { return m.name }
func (m *mockAIProvider) Models() []string                                                         { return []string{"test-model"} }
func (m *mockAIProvider) DefaultModel() string                                                     { return "test-model" }
func (m *mockAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{Content: "test response"}, nil
}
func (m *mockAIProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, nil
}
func (m *mockAIProvider) ClassifyIntent(ctx context.Context, req *IntentClassificationRequest) (*entity.IntentResult, error) {
	return nil, nil
}
func (m *mockAIProvider) AnalyzeSentiment(ctx context.Context, req *SentimentAnalysisRequest) (*entity.SentimentResult, error) {
	return nil, nil
}
func (m *mockAIProvider) IsAvailable() bool { return m.available }

// ============================================================================
// Helper to build the service under test
// ============================================================================

func newTestBotService(botRepo *MockBotRepository, channelRepo *testutil.MockChannelRepository, aiFactory *AIProviderFactory) *BotServiceImpl {
	if channelRepo == nil {
		channelRepo = testutil.NewMockChannelRepository()
	}
	contextRepo := NewMockConversationContextRepository()
	return NewBotService(botRepo, channelRepo, contextRepo, nil, aiFactory, nil)
}

// ============================================================================
// Tests: Create
// ============================================================================

func TestBotCreate_Success(t *testing.T) {
	botRepo := NewMockBotRepository()
	factory := NewAIProviderFactory()
	factory.Register(&mockAIProvider{name: entity.AIProviderOpenAI, available: true})

	svc := newTestBotService(botRepo, nil, factory)

	input := &CreateBotInput{
		TenantID:     "tenant-1",
		Name:         "My Bot",
		Type:         entity.BotTypeAI,
		Provider:     entity.AIProviderOpenAI,
		Model:        "gpt-4",
		SystemPrompt: "You are helpful.",
		Temperature:  0.5,
		MaxTokens:    512,
	}

	bot, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bot == nil {
		t.Fatal("expected bot, got nil")
	}
	if bot.Name != "My Bot" {
		t.Errorf("expected name 'My Bot', got %q", bot.Name)
	}
	if bot.TenantID != "tenant-1" {
		t.Errorf("expected tenant 'tenant-1', got %q", bot.TenantID)
	}
	if bot.Config.SystemPrompt != "You are helpful." {
		t.Errorf("expected system prompt 'You are helpful.', got %q", bot.Config.SystemPrompt)
	}
	if bot.Config.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %f", bot.Config.Temperature)
	}
	if bot.Config.MaxTokens != 512 {
		t.Errorf("expected max tokens 512, got %d", bot.Config.MaxTokens)
	}
	if bot.ID == "" {
		t.Error("expected bot ID to be set")
	}

	// Verify stored in repo
	if _, ok := botRepo.Bots[bot.ID]; !ok {
		t.Error("bot not stored in repository")
	}
}

func TestBotCreate_AllowsUnavailableProvider(t *testing.T) {
	botRepo := NewMockBotRepository()
	factory := NewAIProviderFactory() // no providers registered

	svc := newTestBotService(botRepo, nil, factory)

	input := &CreateBotInput{
		TenantID: "tenant-1",
		Name:     "My Bot",
		Type:     entity.BotTypeAI,
		Provider: entity.AIProviderOpenAI,
		Model:    "gpt-4",
	}

	bot, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error for unavailable provider, got %v", err)
	}
	if bot == nil {
		t.Fatal("expected bot to be created, got nil")
	}
	if bot.Provider != entity.AIProviderOpenAI {
		t.Fatalf("expected provider openai, got %q", bot.Provider)
	}
}

// ============================================================================
// Tests: GetByID
// ============================================================================

func TestBotGetByID_Found(t *testing.T) {
	botRepo := NewMockBotRepository()
	bot := entity.NewBot("tenant-1", "Test Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot

	svc := newTestBotService(botRepo, nil, nil)

	result, err := svc.GetByID(context.Background(), "bot-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != "bot-1" {
		t.Errorf("expected ID 'bot-1', got %q", result.ID)
	}
	if result.Name != "Test Bot" {
		t.Errorf("expected name 'Test Bot', got %q", result.Name)
	}
}

func TestBotGetByID_NotFound(t *testing.T) {
	botRepo := NewMockBotRepository()
	svc := newTestBotService(botRepo, nil, nil)

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found, got nil")
	}
}

// ============================================================================
// Tests: List
// ============================================================================

func TestBotList_ReturnsBots(t *testing.T) {
	botRepo := NewMockBotRepository()

	bot1 := entity.NewBot("tenant-1", "Bot A", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot1.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot1

	bot2 := entity.NewBot("tenant-1", "Bot B", entity.BotTypeAI, entity.AIProviderAnthropic, "claude-3")
	bot2.ID = "bot-2"
	botRepo.Bots["bot-2"] = bot2

	bot3 := entity.NewBot("tenant-2", "Bot C", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot3.ID = "bot-3"
	botRepo.Bots["bot-3"] = bot3

	svc := newTestBotService(botRepo, nil, nil)

	bots, total, err := svc.List(context.Background(), "tenant-1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(bots) != 2 {
		t.Errorf("expected 2 bots, got %d", len(bots))
	}
}

// ============================================================================
// Tests: Update
// ============================================================================

func TestBotUpdate_Success(t *testing.T) {
	botRepo := NewMockBotRepository()
	bot := entity.NewBot("tenant-1", "Old Name", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-3.5")
	bot.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot

	svc := newTestBotService(botRepo, nil, nil)

	newName := "New Name"
	newModel := "gpt-4"
	newPrompt := "Be concise."
	newTemp := 0.3
	newMaxTokens := 2048
	welcomeMsg := "Hello!"
	fallbackMsg := "Sorry, try again."

	updated, err := svc.Update(context.Background(), "bot-1", &UpdateBotInput{
		Name:            &newName,
		Model:           &newModel,
		SystemPrompt:    &newPrompt,
		Temperature:     &newTemp,
		MaxTokens:       &newMaxTokens,
		WelcomeMessage:  &welcomeMsg,
		FallbackMessage: &fallbackMsg,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.Name)
	}
	if updated.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", updated.Model)
	}
	if updated.Config.SystemPrompt != "Be concise." {
		t.Errorf("expected prompt 'Be concise.', got %q", updated.Config.SystemPrompt)
	}
	if updated.Config.Temperature != 0.3 {
		t.Errorf("expected temperature 0.3, got %f", updated.Config.Temperature)
	}
	if updated.Config.MaxTokens != 2048 {
		t.Errorf("expected max tokens 2048, got %d", updated.Config.MaxTokens)
	}
	if updated.Config.WelcomeMessage == nil || *updated.Config.WelcomeMessage != "Hello!" {
		t.Errorf("expected welcome message 'Hello!', got %v", updated.Config.WelcomeMessage)
	}
	if updated.Config.FallbackMessage != "Sorry, try again." {
		t.Errorf("expected fallback 'Sorry, try again.', got %q", updated.Config.FallbackMessage)
	}
}

func TestBotUpdate_NotFound(t *testing.T) {
	botRepo := NewMockBotRepository()
	svc := newTestBotService(botRepo, nil, nil)

	name := "X"
	_, err := svc.Update(context.Background(), "nonexistent", &UpdateBotInput{Name: &name})
	if err == nil {
		t.Fatal("expected error for not found, got nil")
	}
}

// ============================================================================
// Tests: Delete
// ============================================================================

func TestBotDelete_Success(t *testing.T) {
	botRepo := NewMockBotRepository()
	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot

	svc := newTestBotService(botRepo, nil, nil)

	err := svc.Delete(context.Background(), "bot-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := botRepo.Bots["bot-1"]; ok {
		t.Error("bot should have been deleted from repository")
	}
}

// ============================================================================
// Tests: Activate / Deactivate
// ============================================================================

func TestBotActivate_Success(t *testing.T) {
	botRepo := NewMockBotRepository()
	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot

	svc := newTestBotService(botRepo, nil, nil)

	err := svc.Activate(context.Background(), "bot-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if botRepo.Bots["bot-1"].Status != entity.BotStatusActive {
		t.Errorf("expected status active, got %s", botRepo.Bots["bot-1"].Status)
	}
}

func TestBotDeactivate_Success(t *testing.T) {
	botRepo := NewMockBotRepository()
	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	bot.Status = entity.BotStatusActive
	botRepo.Bots["bot-1"] = bot

	svc := newTestBotService(botRepo, nil, nil)

	err := svc.Deactivate(context.Background(), "bot-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if botRepo.Bots["bot-1"].Status != entity.BotStatusInactive {
		t.Errorf("expected status inactive, got %s", botRepo.Bots["bot-1"].Status)
	}
}

// ============================================================================
// Tests: AssignChannel
// ============================================================================

func TestBotAssignChannel_Success(t *testing.T) {
	botRepo := NewMockBotRepository()
	channelRepo := testutil.NewMockChannelRepository()

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot

	channel := &entity.Channel{ID: "ch-1", TenantID: "tenant-1"}
	channelRepo.Channels["ch-1"] = channel

	svc := newTestBotService(botRepo, channelRepo, nil)

	err := svc.AssignChannel(context.Background(), "bot-1", "ch-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if botRepo.ChannelBotMap["ch-1"] != "bot-1" {
		t.Error("channel should be mapped to bot-1")
	}
}

func TestBotAssignChannel_BotNotFound(t *testing.T) {
	botRepo := NewMockBotRepository()
	channelRepo := testutil.NewMockChannelRepository()
	svc := newTestBotService(botRepo, channelRepo, nil)

	err := svc.AssignChannel(context.Background(), "nonexistent", "ch-1")
	if err == nil {
		t.Fatal("expected error for bot not found, got nil")
	}
}

func TestBotAssignChannel_ChannelNotFound(t *testing.T) {
	botRepo := NewMockBotRepository()
	channelRepo := testutil.NewMockChannelRepository()

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot

	svc := newTestBotService(botRepo, channelRepo, nil)

	err := svc.AssignChannel(context.Background(), "bot-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for channel not found, got nil")
	}
}

func TestBotAssignChannel_DifferentTenant(t *testing.T) {
	botRepo := NewMockBotRepository()
	channelRepo := testutil.NewMockChannelRepository()

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	botRepo.Bots["bot-1"] = bot

	channel := &entity.Channel{ID: "ch-1", TenantID: "tenant-2"} // different tenant
	channelRepo.Channels["ch-1"] = channel

	svc := newTestBotService(botRepo, channelRepo, nil)

	err := svc.AssignChannel(context.Background(), "bot-1", "ch-1")
	if err == nil {
		t.Fatal("expected forbidden error for different tenant, got nil")
	}
}

func TestBotAssignChannel_ReassignsFromAnotherBot(t *testing.T) {
	botRepo := NewMockBotRepository()
	channelRepo := testutil.NewMockChannelRepository()

	oldBot := entity.NewBot("tenant-1", "Old Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	oldBot.ID = "bot-old"
	oldBot.Channels = []string{"ch-1"}
	botRepo.Bots["bot-old"] = oldBot
	botRepo.ChannelBotMap["ch-1"] = "bot-old"

	newBot := entity.NewBot("tenant-1", "New Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	newBot.ID = "bot-new"
	botRepo.Bots["bot-new"] = newBot

	channel := &entity.Channel{ID: "ch-1", TenantID: "tenant-1"}
	channelRepo.Channels["ch-1"] = channel

	svc := newTestBotService(botRepo, channelRepo, nil)

	err := svc.AssignChannel(context.Background(), "bot-new", "ch-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if botRepo.ChannelBotMap["ch-1"] != "bot-new" {
		t.Errorf("expected channel mapped to bot-new, got %s", botRepo.ChannelBotMap["ch-1"])
	}
	// Old bot should have the channel removed
	if oldBot.HasChannel("ch-1") {
		t.Error("old bot should no longer have channel ch-1")
	}
}

// ============================================================================
// Tests: UnassignChannel
// ============================================================================

func TestBotUnassignChannel_Success(t *testing.T) {
	botRepo := NewMockBotRepository()

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	bot.Channels = []string{"ch-1"}
	botRepo.Bots["bot-1"] = bot
	botRepo.ChannelBotMap["ch-1"] = "bot-1"

	svc := newTestBotService(botRepo, nil, nil)

	err := svc.UnassignChannel(context.Background(), "bot-1", "ch-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := botRepo.ChannelBotMap["ch-1"]; ok {
		t.Error("channel should have been removed from map")
	}
	if bot.HasChannel("ch-1") {
		t.Error("bot should no longer have channel ch-1")
	}
}

// ============================================================================
// Tests: ShouldBotHandle
// ============================================================================

func TestBotShouldBotHandle_ActiveBotWithChannel(t *testing.T) {
	botRepo := NewMockBotRepository()
	svc := newTestBotService(botRepo, nil, nil)

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	bot.Status = entity.BotStatusActive
	bot.Channels = []string{"ch-1"}

	conv := &entity.Conversation{
		ID:        "conv-1",
		ChannelID: "ch-1",
	}

	should, err := svc.ShouldBotHandle(context.Background(), conv, bot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !should {
		t.Error("expected true, bot is active and assigned to channel")
	}
}

func TestBotShouldBotHandle_InactiveBot(t *testing.T) {
	botRepo := NewMockBotRepository()
	svc := newTestBotService(botRepo, nil, nil)

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	bot.Status = entity.BotStatusInactive
	bot.Channels = []string{"ch-1"}

	conv := &entity.Conversation{
		ID:        "conv-1",
		ChannelID: "ch-1",
	}

	should, err := svc.ShouldBotHandle(context.Background(), conv, bot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if should {
		t.Error("expected false, bot is inactive")
	}
}

func TestBotShouldBotHandle_BotWithoutChannel(t *testing.T) {
	botRepo := NewMockBotRepository()
	svc := newTestBotService(botRepo, nil, nil)

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	bot.Status = entity.BotStatusActive
	bot.Channels = []string{} // no channels

	conv := &entity.Conversation{
		ID:        "conv-1",
		ChannelID: "ch-1",
	}

	should, err := svc.ShouldBotHandle(context.Background(), conv, bot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if should {
		t.Error("expected false, bot has no channel assigned")
	}
}

func TestBotShouldBotHandle_ConversationAssignedToHuman(t *testing.T) {
	botRepo := NewMockBotRepository()
	svc := newTestBotService(botRepo, nil, nil)

	bot := entity.NewBot("tenant-1", "Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = "bot-1"
	bot.Status = entity.BotStatusActive
	bot.Channels = []string{"ch-1"}

	userID := "user-42"
	conv := &entity.Conversation{
		ID:             "conv-1",
		ChannelID:      "ch-1",
		AssignedUserID: &userID,
	}

	should, err := svc.ShouldBotHandle(context.Background(), conv, bot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if should {
		t.Error("expected false, conversation is assigned to a human")
	}
}
