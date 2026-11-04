package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// MockConversationOwnerRuleRepository is an in-memory ConversationOwnerRuleRepository
// with the same rules as the PostgreSQL one (00021): same conversa_ref always
// applies; another conversation applies only if its desde is not older.
type MockConversationOwnerRuleRepository struct {
	mu    sync.Mutex
	Rules map[string]*entity.ConversationOwnerRule
}

// NewMockConversationOwnerRuleRepository creates the mock.
func NewMockConversationOwnerRuleRepository() *MockConversationOwnerRuleRepository {
	return &MockConversationOwnerRuleRepository{Rules: map[string]*entity.ConversationOwnerRule{}}
}

func ownerKey(tenantID, channelID, identifier string) string {
	return tenantID + "|" + channelID + "|" + identifier
}

// Upsert implements the repository contract.
func (m *MockConversationOwnerRuleRepository) Upsert(_ context.Context, rule *entity.ConversationOwnerRule) (bool, *entity.ConversationOwnerRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := ownerKey(rule.TenantID, rule.ChannelID, rule.ContactIdentifier)
	if cur, ok := m.Rules[k]; ok && cur.ConversaRef != rule.ConversaRef && cur.Desde.After(rule.Desde) {
		c := *cur
		return false, &c, nil
	}
	r := *rule
	r.UpdatedAt = time.Now()
	m.Rules[k] = &r
	c := r
	return true, &c, nil
}

// Find implements the repository contract.
func (m *MockConversationOwnerRuleRepository) Find(_ context.Context, tenantID, channelID, identifier string) (*entity.ConversationOwnerRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.Rules[ownerKey(tenantID, channelID, identifier)]; ok {
		c := *r
		return &c, nil
	}
	return nil, nil
}

// Release implements the repository contract.
func (m *MockConversationOwnerRuleRepository) Release(_ context.Context, tenantID, channelID, identifier, conversaRef string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := ownerKey(tenantID, channelID, identifier)
	if r, ok := m.Rules[k]; ok && r.ConversaRef == conversaRef {
		delete(m.Rules, k)
		return true, nil
	}
	return false, nil
}
