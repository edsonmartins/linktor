package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConversationContextRepository is an inline mock for ConversationContextRepository
type mockConversationContextRepository struct {
	contexts    map[string]*entity.ConversationContext // key by conversation_id
	byID        map[string]*entity.ConversationContext // key by id
	ReturnError error
}

func newMockConversationContextRepository() *mockConversationContextRepository {
	return &mockConversationContextRepository{
		contexts: make(map[string]*entity.ConversationContext),
		byID:     make(map[string]*entity.ConversationContext),
	}
}

func (m *mockConversationContextRepository) Create(ctx context.Context, convContext *entity.ConversationContext) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.contexts[convContext.ConversationID] = convContext
	m.byID[convContext.ID] = convContext
	return nil
}

func (m *mockConversationContextRepository) FindByID(ctx context.Context, id string) (*entity.ConversationContext, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	cc, ok := m.byID[id]
	if !ok {
		return nil, errors.New(errors.ErrCodeNotFound, "not found")
	}
	return cc, nil
}

func (m *mockConversationContextRepository) FindByConversation(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	cc, ok := m.contexts[conversationID]
	if !ok {
		return nil, errors.New(errors.ErrCodeNotFound, "not found")
	}
	return cc, nil
}

func (m *mockConversationContextRepository) Update(ctx context.Context, convContext *entity.ConversationContext) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.contexts[convContext.ConversationID] = convContext
	m.byID[convContext.ID] = convContext
	return nil
}

func (m *mockConversationContextRepository) UpdateIntent(ctx context.Context, id string, intent *entity.Intent) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	cc, ok := m.byID[id]
	if !ok {
		return errors.New(errors.ErrCodeNotFound, "not found")
	}
	cc.Intent = intent
	return nil
}

func (m *mockConversationContextRepository) UpdateSentiment(ctx context.Context, id string, sentiment entity.Sentiment) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	cc, ok := m.byID[id]
	if !ok {
		return errors.New(errors.ErrCodeNotFound, "not found")
	}
	cc.Sentiment = sentiment
	return nil
}

func (m *mockConversationContextRepository) UpdateContextWindow(ctx context.Context, id string, window []entity.ContextMessage) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	cc, ok := m.byID[id]
	if !ok {
		return errors.New(errors.ErrCodeNotFound, "not found")
	}
	cc.ContextWindow = window
	return nil
}

func (m *mockConversationContextRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	cc, ok := m.byID[id]
	if ok {
		delete(m.contexts, cc.ConversationID)
		delete(m.byID, id)
	}
	return nil
}

func TestNewConversationContextService(t *testing.T) {
	repo := newMockConversationContextRepository()

	t.Run("default config", func(t *testing.T) {
		svc := NewConversationContextService(repo, nil)
		require.NotNil(t, svc)
		assert.Equal(t, 20, svc.config.MaxContextWindowSize)
		assert.Equal(t, 10, svc.config.TrimToSize)
	})

	t.Run("custom config", func(t *testing.T) {
		cfg := &ConversationContextConfig{
			MaxContextWindowSize: 50,
			TrimToSize:           25,
		}
		svc := NewConversationContextService(repo, cfg)
		require.NotNil(t, svc)
		assert.Equal(t, 50, svc.config.MaxContextWindowSize)
		assert.Equal(t, 25, svc.config.TrimToSize)
	})
}

func TestDefaultContextConfig(t *testing.T) {
	cfg := DefaultContextConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 20, cfg.MaxContextWindowSize)
	assert.Equal(t, 10, cfg.TrimToSize)
}

func TestConversationContextService_GetOrCreate(t *testing.T) {
	t.Run("create new", func(t *testing.T) {
		repo := newMockConversationContextRepository()
		svc := NewConversationContextService(repo, nil)
		ctx := context.Background()

		cc, err := svc.GetOrCreate(ctx, "conv-1")
		require.NoError(t, err)
		require.NotNil(t, cc)
		assert.Equal(t, "conv-1", cc.ConversationID)
		assert.NotEmpty(t, cc.ID)
	})

	t.Run("get existing", func(t *testing.T) {
		repo := newMockConversationContextRepository()
		svc := NewConversationContextService(repo, nil)
		ctx := context.Background()

		// Create first
		cc1, err := svc.GetOrCreate(ctx, "conv-1")
		require.NoError(t, err)

		// Get same - should return cached
		cc2, err := svc.GetOrCreate(ctx, "conv-1")
		require.NoError(t, err)
		assert.Equal(t, cc1.ID, cc2.ID)
	})

	t.Run("get existing from repo after cache clear", func(t *testing.T) {
		repo := newMockConversationContextRepository()
		svc := NewConversationContextService(repo, nil)
		ctx := context.Background()

		cc1, err := svc.GetOrCreate(ctx, "conv-1")
		require.NoError(t, err)

		// Clear cache
		svc.ClearCache()

		// Should find it in the repo
		cc2, err := svc.GetOrCreate(ctx, "conv-1")
		require.NoError(t, err)
		assert.Equal(t, cc1.ID, cc2.ID)
	})
}

func TestConversationContextService_AddUserMessage(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	err := svc.AddUserMessage(ctx, "conv-1", "Hello!", "msg-1")
	require.NoError(t, err)

	cc, err := svc.Get(ctx, "conv-1")
	require.NoError(t, err)
	require.Len(t, cc.ContextWindow, 1)
	assert.Equal(t, "user", cc.ContextWindow[0].Role)
	assert.Equal(t, "Hello!", cc.ContextWindow[0].Content)
	assert.Equal(t, "msg-1", cc.ContextWindow[0].MessageID)
}

func TestConversationContextService_AddAssistantMessage(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	err := svc.AddAssistantMessage(ctx, "conv-1", "Hi there!", "msg-2")
	require.NoError(t, err)

	cc, err := svc.Get(ctx, "conv-1")
	require.NoError(t, err)
	require.Len(t, cc.ContextWindow, 1)
	assert.Equal(t, "assistant", cc.ContextWindow[0].Role)
	assert.Equal(t, "Hi there!", cc.ContextWindow[0].Content)
}

func TestConversationContextService_AddSystemMessage(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	err := svc.AddSystemMessage(ctx, "conv-1", "You are a helpful assistant")
	require.NoError(t, err)

	cc, err := svc.Get(ctx, "conv-1")
	require.NoError(t, err)
	require.Len(t, cc.ContextWindow, 1)
	assert.Equal(t, "system", cc.ContextWindow[0].Role)
	assert.Equal(t, "You are a helpful assistant", cc.ContextWindow[0].Content)
}

func TestConversationContextService_SetBot(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	err := svc.SetBot(ctx, "conv-1", "bot-1")
	require.NoError(t, err)

	cc, err := svc.Get(ctx, "conv-1")
	require.NoError(t, err)
	require.NotNil(t, cc.BotID)
	assert.Equal(t, "bot-1", *cc.BotID)
}

func TestConversationContextService_ClearBot(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	// Set bot first
	err := svc.SetBot(ctx, "conv-1", "bot-1")
	require.NoError(t, err)

	// Clear bot
	err = svc.ClearBot(ctx, "conv-1")
	require.NoError(t, err)

	cc, err := svc.Get(ctx, "conv-1")
	require.NoError(t, err)
	assert.Nil(t, cc.BotID)
}

func TestConversationContextService_GetContextWindow(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	t.Run("empty context", func(t *testing.T) {
		msgs, err := svc.GetContextWindow(ctx, "conv-empty", 0)
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("with messages", func(t *testing.T) {
		err := svc.AddUserMessage(ctx, "conv-2", "msg1", "id1")
		require.NoError(t, err)
		err = svc.AddAssistantMessage(ctx, "conv-2", "reply1", "id2")
		require.NoError(t, err)
		err = svc.AddUserMessage(ctx, "conv-2", "msg2", "id3")
		require.NoError(t, err)

		msgs, err := svc.GetContextWindow(ctx, "conv-2", 0)
		require.NoError(t, err)
		assert.Len(t, msgs, 3)
	})

	t.Run("with limit", func(t *testing.T) {
		msgs, err := svc.GetContextWindow(ctx, "conv-2", 2)
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
		// Should return the last 2 messages
		assert.Equal(t, "reply1", msgs[0].Content)
		assert.Equal(t, "msg2", msgs[1].Content)
	})
}

func TestConversationContextService_BuildMessagesForAI(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	// Add some context messages
	err := svc.AddUserMessage(ctx, "conv-ai", "previous question", "m1")
	require.NoError(t, err)
	err = svc.AddAssistantMessage(ctx, "conv-ai", "previous answer", "m2")
	require.NoError(t, err)

	t.Run("with system prompt", func(t *testing.T) {
		msgs, err := svc.BuildMessagesForAI(ctx, "conv-ai", "You are helpful", "new question", 10)
		require.NoError(t, err)
		require.Len(t, msgs, 4) // system + 2 context + current
		assert.Equal(t, "system", msgs[0].Role)
		assert.Equal(t, "You are helpful", msgs[0].Content)
		assert.Equal(t, "user", msgs[1].Role)
		assert.Equal(t, "previous question", msgs[1].Content)
		assert.Equal(t, "assistant", msgs[2].Role)
		assert.Equal(t, "user", msgs[3].Role)
		assert.Equal(t, "new question", msgs[3].Content)
	})

	t.Run("without system prompt", func(t *testing.T) {
		msgs, err := svc.BuildMessagesForAI(ctx, "conv-ai", "", "new question", 10)
		require.NoError(t, err)
		require.Len(t, msgs, 3) // 2 context + current
		assert.Equal(t, "user", msgs[0].Role)
		assert.Equal(t, "previous question", msgs[0].Content)
	})
}

func TestConversationContextService_Delete(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	// Create context
	_, err := svc.GetOrCreate(ctx, "conv-del")
	require.NoError(t, err)

	// Delete
	err = svc.Delete(ctx, "conv-del")
	require.NoError(t, err)

	// Should not be found in repo
	_, ok := repo.contexts["conv-del"]
	assert.False(t, ok)
}

func TestConversationContextService_ClearCache(t *testing.T) {
	repo := newMockConversationContextRepository()
	svc := NewConversationContextService(repo, nil)
	ctx := context.Background()

	// Create multiple contexts
	_, err := svc.GetOrCreate(ctx, "conv-a")
	require.NoError(t, err)
	_, err = svc.GetOrCreate(ctx, "conv-b")
	require.NoError(t, err)

	svc.ClearCache()

	// Cache is cleared but repo still has data
	assert.Len(t, svc.cache, 0)
	assert.Len(t, repo.contexts, 2)
}

func TestConversationContextService_TrimContextWindow(t *testing.T) {
	repo := newMockConversationContextRepository()
	cfg := &ConversationContextConfig{
		MaxContextWindowSize: 5,
		TrimToSize:           3,
	}
	svc := NewConversationContextService(repo, cfg)
	ctx := context.Background()

	// Add more messages than the max window size
	for i := 0; i < 7; i++ {
		err := svc.AddUserMessage(ctx, "conv-trim", "message", "id")
		require.NoError(t, err)
	}

	cc, err := svc.Get(ctx, "conv-trim")
	require.NoError(t, err)
	// After exceeding 5, should have been trimmed to 3, then continued adding
	// Messages 1-5: at msg 6 (len=6>5), trim to 3, then add msg 6 => 4
	// At msg 7 (len=5, not >5) no trim => 5
	assert.LessOrEqual(t, len(cc.ContextWindow), cfg.MaxContextWindowSize+1)
}
