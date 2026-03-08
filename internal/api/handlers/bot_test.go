package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBotRepository is an inline mock for BotRepository
type mockBotRepository struct {
	Bots        map[string]*entity.Bot
	ReturnError error
}

func newMockBotRepository() *mockBotRepository {
	return &mockBotRepository{
		Bots: make(map[string]*entity.Bot),
	}
}

func (m *mockBotRepository) Create(ctx context.Context, bot *entity.Bot) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Bots[bot.ID] = bot
	return nil
}

func (m *mockBotRepository) FindByID(ctx context.Context, id string) (*entity.Bot, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	bot, ok := m.Bots[id]
	if !ok {
		return nil, fmt.Errorf("bot not found: %s", id)
	}
	return bot, nil
}

func (m *mockBotRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Bot, int64, error) {
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

func (m *mockBotRepository) FindByChannel(ctx context.Context, channelID string) (*entity.Bot, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, b := range m.Bots {
		for _, ch := range b.Channels {
			if ch == channelID {
				return b, nil
			}
		}
	}
	return nil, fmt.Errorf("bot not found for channel: %s", channelID)
}

func (m *mockBotRepository) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Bot, error) {
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

func (m *mockBotRepository) Update(ctx context.Context, bot *entity.Bot) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Bots[bot.ID] = bot
	return nil
}

func (m *mockBotRepository) UpdateStatus(ctx context.Context, id string, status entity.BotStatus) error {
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

func (m *mockBotRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Bots, id)
	return nil
}

func (m *mockBotRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
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

func (m *mockBotRepository) AssignChannel(ctx context.Context, botID, channelID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	bot, ok := m.Bots[botID]
	if !ok {
		return fmt.Errorf("bot not found: %s", botID)
	}
	bot.Channels = append(bot.Channels, channelID)
	return nil
}

func (m *mockBotRepository) UnassignChannel(ctx context.Context, botID, channelID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	bot, ok := m.Bots[botID]
	if !ok {
		return fmt.Errorf("bot not found: %s", botID)
	}
	for i, ch := range bot.Channels {
		if ch == channelID {
			bot.Channels = append(bot.Channels[:i], bot.Channels[i+1:]...)
			return nil
		}
	}
	return nil
}

// mockConversationContextRepository is an inline mock for ConversationContextRepository
type mockConversationContextRepository struct{}

func (m *mockConversationContextRepository) Create(ctx context.Context, convContext *entity.ConversationContext) error {
	return nil
}
func (m *mockConversationContextRepository) FindByID(ctx context.Context, id string) (*entity.ConversationContext, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockConversationContextRepository) FindByConversation(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockConversationContextRepository) Update(ctx context.Context, convContext *entity.ConversationContext) error {
	return nil
}
func (m *mockConversationContextRepository) UpdateIntent(ctx context.Context, id string, intent *entity.Intent) error {
	return nil
}
func (m *mockConversationContextRepository) UpdateSentiment(ctx context.Context, id string, sentiment entity.Sentiment) error {
	return nil
}
func (m *mockConversationContextRepository) UpdateContextWindow(ctx context.Context, id string, window []entity.ContextMessage) error {
	return nil
}
func (m *mockConversationContextRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func setupBotTest(t *testing.T) (*BotHandler, *mockBotRepository, *testutil.MockChannelRepository) {
	t.Helper()

	botRepo := newMockBotRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contextRepo := &mockConversationContextRepository{}
	contextService := service.NewConversationContextService(contextRepo, nil)
	aiFactory := service.NewAIProviderFactory()
	flowEngine := service.NewFlowEngineService(newMockFlowRepository(), contextRepo)

	botService := service.NewBotService(botRepo, channelRepo, contextRepo, contextService, aiFactory, flowEngine)
	handler := NewBotHandler(botService)

	return handler, botRepo, channelRepo
}

func TestBotHandler_List(t *testing.T) {
	handler, botRepo, _ := setupBotTest(t)

	botRepo.Bots["bot-1"] = &entity.Bot{
		ID:       "bot-1",
		TenantID: "tenant-1",
		Name:     "Support Bot",
		Type:     entity.BotTypeAI,
		Provider: entity.AIProviderOpenAI,
		Model:    "gpt-4",
		Status:   entity.BotStatusActive,
	}
	botRepo.Bots["bot-2"] = &entity.Bot{
		ID:       "bot-2",
		TenantID: "tenant-1",
		Name:     "Sales Bot",
		Type:     entity.BotTypeAI,
		Provider: entity.AIProviderAnthropic,
		Model:    "claude-3",
		Status:   entity.BotStatusInactive,
	}

	w, c := newTestContext(http.MethodGet, "/bots", nil)
	c.Set("tenant_id", "tenant-1")
	c.Request.URL.RawQuery = "page=1&page_size=20"

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.NotNil(t, rawResp["data"])
	assert.NotNil(t, rawResp["pagination"])

	dataList, ok := rawResp["data"].([]interface{})
	require.True(t, ok)
	assert.Len(t, dataList, 2)
}

func TestBotHandler_List_NoTenantID(t *testing.T) {
	handler, _, _ := setupBotTest(t)

	w, c := newTestContext(http.MethodGet, "/bots", nil)

	handler.List(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBotHandler_Get(t *testing.T) {
	handler, botRepo, _ := setupBotTest(t)

	botRepo.Bots["bot-1"] = &entity.Bot{
		ID:       "bot-1",
		TenantID: "tenant-1",
		Name:     "Support Bot",
		Type:     entity.BotTypeAI,
		Provider: entity.AIProviderOpenAI,
		Model:    "gpt-4",
		Status:   entity.BotStatusActive,
	}

	w, c := newTestContext(http.MethodGet, "/bots/bot-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "bot-1", dataMap["id"])
	assert.Equal(t, "Support Bot", dataMap["name"])
}

func TestBotHandler_Get_NotFound(t *testing.T) {
	handler, _, _ := setupBotTest(t)

	w, c := newTestContext(http.MethodGet, "/bots/nonexistent", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Get(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestBotHandler_Get_EmptyID(t *testing.T) {
	handler, _, _ := setupBotTest(t)

	w, c := newTestContext(http.MethodGet, "/bots/", nil)
	c.Set("tenant_id", "tenant-1")
	// No params set - empty ID

	handler.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestBotHandler_Update(t *testing.T) {
	handler, botRepo, _ := setupBotTest(t)

	botRepo.Bots["bot-1"] = &entity.Bot{
		ID:       "bot-1",
		TenantID: "tenant-1",
		Name:     "Support Bot",
		Type:     entity.BotTypeAI,
		Provider: entity.AIProviderOpenAI,
		Model:    "gpt-4",
		Status:   entity.BotStatusActive,
	}

	newName := "Updated Bot"
	body := UpdateBotRequest{
		Name: &newName,
	}

	w, c := newTestContext(http.MethodPut, "/bots/bot-1", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}

	handler.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "Updated Bot", dataMap["name"])
}

func TestBotHandler_Update_InvalidBody(t *testing.T) {
	handler, _, _ := setupBotTest(t)

	w, c := newTestContext(http.MethodPut, "/bots/bot-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}
	c.Request.Body = http.NoBody

	handler.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestBotHandler_Delete(t *testing.T) {
	handler, botRepo, _ := setupBotTest(t)

	botRepo.Bots["bot-1"] = &entity.Bot{
		ID:       "bot-1",
		TenantID: "tenant-1",
		Name:     "Support Bot",
	}

	w, c := newTestContext(http.MethodDelete, "/bots/bot-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}

	handler.Delete(c)

	gotCode := w.Code
	if gotCode == http.StatusOK {
		gotCode = c.Writer.Status()
	}
	assert.Equal(t, http.StatusNoContent, gotCode)
	assert.Empty(t, botRepo.Bots)
}

func TestBotHandler_Activate(t *testing.T) {
	handler, botRepo, _ := setupBotTest(t)

	botRepo.Bots["bot-1"] = &entity.Bot{
		ID:       "bot-1",
		TenantID: "tenant-1",
		Name:     "Support Bot",
		Status:   entity.BotStatusInactive,
	}

	w, c := newTestContext(http.MethodPost, "/bots/bot-1/activate", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}

	handler.Activate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Verify status changed
	assert.Equal(t, entity.BotStatusActive, botRepo.Bots["bot-1"].Status)
}

func TestBotHandler_Deactivate(t *testing.T) {
	handler, botRepo, _ := setupBotTest(t)

	botRepo.Bots["bot-1"] = &entity.Bot{
		ID:       "bot-1",
		TenantID: "tenant-1",
		Name:     "Support Bot",
		Status:   entity.BotStatusActive,
	}

	w, c := newTestContext(http.MethodPost, "/bots/bot-1/deactivate", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}

	handler.Deactivate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	assert.Equal(t, entity.BotStatusInactive, botRepo.Bots["bot-1"].Status)
}

func TestBotHandler_AssignChannel(t *testing.T) {
	handler, botRepo, channelRepo := setupBotTest(t)

	botRepo.Bots["bot-1"] = &entity.Bot{
		ID:       "bot-1",
		TenantID: "tenant-1",
		Name:     "Support Bot",
		Status:   entity.BotStatusActive,
	}

	channelRepo.Channels["channel-1"] = &entity.Channel{
		ID:       "channel-1",
		TenantID: "tenant-1",
		Name:     "WhatsApp Main",
		Type:     entity.ChannelTypeWhatsApp,
		Enabled:  true,
	}

	body := AssignChannelRequest{
		ChannelID: "channel-1",
	}

	w, c := newTestContext(http.MethodPost, "/bots/bot-1/channels", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}

	handler.AssignChannel(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestBotHandler_AssignChannel_InvalidBody(t *testing.T) {
	handler, _, _ := setupBotTest(t)

	w, c := newTestContext(http.MethodPost, "/bots/bot-1/channels", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "bot-1"}}
	c.Request.Body = http.NoBody

	handler.AssignChannel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}
