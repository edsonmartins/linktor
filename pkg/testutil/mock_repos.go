package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// ============================================================================
// MockContactRepository
// ============================================================================

// MockContactRepository is a mock implementation of repository.ContactRepository
type MockContactRepository struct {
	Contacts   map[string]*entity.Contact
	Identities map[string][]*entity.ContactIdentity
	// CreateIdentityHook, when set, runs at the start of CreateIdentityIfAbsent
	// before the uniqueness check. Tests use it to deterministically simulate a
	// concurrent request winning the identity race.
	CreateIdentityHook func()
	OutboxEvents       []*entity.OutboxEvent
	ReturnError        error
}

// NewMockContactRepository creates a new MockContactRepository
func NewMockContactRepository() *MockContactRepository {
	return &MockContactRepository{
		Contacts:   make(map[string]*entity.Contact),
		Identities: make(map[string][]*entity.ContactIdentity),
	}
}

func (m *MockContactRepository) Create(ctx context.Context, contact *entity.Contact) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Contacts[contact.ID] = contact
	return nil
}

func (m *MockContactRepository) FindByID(ctx context.Context, id string) (*entity.Contact, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	contact, ok := m.Contacts[id]
	if !ok {
		return nil, fmt.Errorf("contact not found: %s", id)
	}
	return contact, nil
}

func (m *MockContactRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Contact, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Contact
	for _, c := range m.Contacts {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockContactRepository) FindByEmail(ctx context.Context, tenantID, email string) (*entity.Contact, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, c := range m.Contacts {
		if c.TenantID == tenantID && c.Email == email {
			return c, nil
		}
	}
	return nil, fmt.Errorf("contact not found by email: %s", email)
}

func (m *MockContactRepository) FindByPhone(ctx context.Context, tenantID, phone string) (*entity.Contact, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, c := range m.Contacts {
		if c.TenantID == tenantID && c.Phone == phone {
			return c, nil
		}
	}
	return nil, fmt.Errorf("contact not found by phone: %s", phone)
}

func (m *MockContactRepository) FindByIdentity(ctx context.Context, tenantID, channelType, identifier string) (*entity.Contact, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, c := range m.Contacts {
		if c.TenantID != tenantID {
			continue
		}
		for _, id := range c.Identities {
			if id.ChannelType == channelType && id.Identifier == identifier {
				return c, nil
			}
		}
	}
	return nil, fmt.Errorf("contact not found by identity: %s/%s", channelType, identifier)
}

func (m *MockContactRepository) Update(ctx context.Context, contact *entity.Contact) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Contacts[contact.ID] = contact
	return nil
}

func (m *MockContactRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Contacts, id)
	return nil
}

func (m *MockContactRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, c := range m.Contacts {
		if c.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockContactRepository) AddIdentity(ctx context.Context, identity *entity.ContactIdentity) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Identities[identity.ContactID] = append(m.Identities[identity.ContactID], identity)
	if c, ok := m.Contacts[identity.ContactID]; ok {
		c.Identities = append(c.Identities, identity)
	}
	return nil
}

// CreateIdentityIfAbsent models the tenant-scoped partial UNIQUE index on
// (tenant_id, channel_type, identifier): if any contact in the same tenant
// already owns a matching identity, the insert is a no-op and reports
// inserted=false, mirroring the ON CONFLICT DO NOTHING behavior of Postgres.
func (m *MockContactRepository) CreateIdentityIfAbsent(ctx context.Context, identity *entity.ContactIdentity) (bool, error) {
	if m.ReturnError != nil {
		return false, m.ReturnError
	}
	if m.CreateIdentityHook != nil {
		m.CreateIdentityHook()
	}

	// Determine the tenant scope. Fall back to the owning contact's tenant when
	// the identity does not carry one explicitly.
	tenantID := identity.TenantID
	if tenantID == "" {
		if c, ok := m.Contacts[identity.ContactID]; ok {
			tenantID = c.TenantID
		}
	}

	// Enforce (tenant_id, channel_type, identifier) uniqueness across contacts.
	for _, c := range m.Contacts {
		if c.TenantID != tenantID {
			continue
		}
		for _, existing := range c.Identities {
			if existing.ChannelType == identity.ChannelType && existing.Identifier == identity.Identifier {
				return false, nil
			}
		}
	}

	identity.TenantID = tenantID
	m.Identities[identity.ContactID] = append(m.Identities[identity.ContactID], identity)
	if c, ok := m.Contacts[identity.ContactID]; ok {
		c.Identities = append(c.Identities, identity)
	}
	return true, nil
}

func (m *MockContactRepository) CreateIdentityIfAbsentWithOutboxEvent(ctx context.Context, identity *entity.ContactIdentity, event *entity.OutboxEvent) (bool, error) {
	inserted, err := m.CreateIdentityIfAbsent(ctx, identity)
	if err != nil {
		return false, err
	}
	if inserted && event != nil {
		m.OutboxEvents = append(m.OutboxEvents, event)
	}
	return inserted, nil
}

func (m *MockContactRepository) RemoveIdentity(ctx context.Context, contactID, identityID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	identities := m.Identities[contactID]
	for i, id := range identities {
		if id.ID == identityID {
			m.Identities[contactID] = append(identities[:i], identities[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("identity not found: %s", identityID)
}

func (m *MockContactRepository) FindIdentitiesByContact(ctx context.Context, contactID string) ([]*entity.ContactIdentity, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	identities, ok := m.Identities[contactID]
	if !ok {
		return []*entity.ContactIdentity{}, nil
	}
	return identities, nil
}

// ============================================================================
// MockConversationRepository
// ============================================================================

// MockConversationRepository is a mock implementation of repository.ConversationRepository
type MockConversationRepository struct {
	Conversations map[string]*entity.Conversation
	OutboxEvents  []*entity.OutboxEvent
	ReturnError   error
}

// NewMockConversationRepository creates a new MockConversationRepository
func NewMockConversationRepository() *MockConversationRepository {
	return &MockConversationRepository{
		Conversations: make(map[string]*entity.Conversation),
	}
}

func (m *MockConversationRepository) Create(ctx context.Context, conversation *entity.Conversation) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Conversations[conversation.ID] = conversation
	return nil
}

func (m *MockConversationRepository) CreateWithOutboxEvent(ctx context.Context, conversation *entity.Conversation, event *entity.OutboxEvent) error {
	if err := m.Create(ctx, conversation); err != nil {
		return err
	}
	if event != nil {
		m.OutboxEvents = append(m.OutboxEvents, event)
	}
	return nil
}

func (m *MockConversationRepository) FindByID(ctx context.Context, id string) (*entity.Conversation, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	conv, ok := m.Conversations[id]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}
	return conv, nil
}

func (m *MockConversationRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Conversation
	for _, c := range m.Conversations {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockConversationRepository) FindByChannel(ctx context.Context, channelID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Conversation
	for _, c := range m.Conversations {
		if c.ChannelID == channelID {
			result = append(result, c)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockConversationRepository) FindByContact(ctx context.Context, contactID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Conversation
	for _, c := range m.Conversations {
		if c.ContactID == contactID {
			result = append(result, c)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockConversationRepository) FindByAssignee(ctx context.Context, assigneeID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Conversation
	for _, c := range m.Conversations {
		if c.AssignedUserID != nil && *c.AssignedUserID == assigneeID {
			result = append(result, c)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockConversationRepository) FindOpenByContactAndChannel(ctx context.Context, contactID, channelID string) (*entity.Conversation, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, c := range m.Conversations {
		if c.ContactID == contactID && c.ChannelID == channelID && c.IsOpen() {
			return c, nil
		}
	}
	return nil, fmt.Errorf("no open conversation found for contact %s on channel %s", contactID, channelID)
}

func (m *MockConversationRepository) Update(ctx context.Context, conversation *entity.Conversation) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Conversations[conversation.ID] = conversation
	return nil
}

func (m *MockConversationRepository) UpdateStatus(ctx context.Context, id string, status entity.ConversationStatus) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	conv, ok := m.Conversations[id]
	if !ok {
		return fmt.Errorf("conversation not found: %s", id)
	}
	conv.Status = status
	return nil
}

func (m *MockConversationRepository) UpdateAssignee(ctx context.Context, id string, assigneeID *string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	conv, ok := m.Conversations[id]
	if !ok {
		return fmt.Errorf("conversation not found: %s", id)
	}
	conv.AssignedUserID = assigneeID
	return nil
}

func (m *MockConversationRepository) IncrementUnreadCount(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	conv, ok := m.Conversations[id]
	if !ok {
		return fmt.Errorf("conversation not found: %s", id)
	}
	conv.UnreadCount++
	return nil
}

func (m *MockConversationRepository) ResetUnreadCount(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	conv, ok := m.Conversations[id]
	if !ok {
		return fmt.Errorf("conversation not found: %s", id)
	}
	conv.UnreadCount = 0
	return nil
}

func (m *MockConversationRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Conversations, id)
	return nil
}

func (m *MockConversationRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, c := range m.Conversations {
		if c.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockConversationRepository) CountByStatus(ctx context.Context, tenantID string, status entity.ConversationStatus) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, c := range m.Conversations {
		if c.TenantID == tenantID && c.Status == status {
			count++
		}
	}
	return count, nil
}

func (m *MockConversationRepository) CountActiveByUser(ctx context.Context, userID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, c := range m.Conversations {
		if c.AssignedUserID != nil && *c.AssignedUserID == userID && c.IsOpen() {
			count++
		}
	}
	return count, nil
}

func (m *MockConversationRepository) CountWaiting(ctx context.Context, tenantID string, minPriority entity.ConversationPriority) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, c := range m.Conversations {
		if c.TenantID == tenantID && c.AssignedUserID == nil && c.IsOpen() {
			count++
		}
	}
	return count, nil
}

// ============================================================================
// MockMessageRepository
// ============================================================================

// MockMessageRepository is a mock implementation of repository.MessageRepository
type MockMessageRepository struct {
	Messages     map[string]*entity.Message
	Attachments  map[string][]*entity.MessageAttachment
	OutboxEvents []*entity.OutboxEvent
	ReturnError  error
}

// NewMockMessageRepository creates a new MockMessageRepository
func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		Messages:    make(map[string]*entity.Message),
		Attachments: make(map[string][]*entity.MessageAttachment),
	}
}

func (m *MockMessageRepository) Create(ctx context.Context, message *entity.Message) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Messages[message.ID] = message
	return nil
}

// CreateWithOutboxEvent mirrors the transactional-outbox repo: it dedupes by
// (conversation_id, external_id) and, on a genuine insert, captures the outbox
// event. A duplicate returns inserted=false and enqueues nothing.
func (m *MockMessageRepository) CreateWithOutboxEvent(ctx context.Context, message *entity.Message, event *entity.OutboxEvent) (bool, error) {
	if m.ReturnError != nil {
		return false, m.ReturnError
	}
	if message.ExternalID != "" {
		for _, existing := range m.Messages {
			if existing.ExternalID == message.ExternalID && existing.ConversationID == message.ConversationID {
				return false, nil // duplicate delivery
			}
		}
	}
	m.Messages[message.ID] = message
	if event != nil {
		m.OutboxEvents = append(m.OutboxEvents, event)
	}
	return true, nil
}

func (m *MockMessageRepository) FindByID(ctx context.Context, id string) (*entity.Message, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	msg, ok := m.Messages[id]
	if !ok {
		return nil, fmt.Errorf("message not found: %s", id)
	}
	return msg, nil
}

func (m *MockMessageRepository) FindByExternalID(ctx context.Context, externalID string) (*entity.Message, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, msg := range m.Messages {
		if msg.ExternalID == externalID {
			return msg, nil
		}
	}
	return nil, fmt.Errorf("message not found by external ID: %s", externalID)
}

// FindByExternalIDInConversation mirrors the real repository: a miss is (nil, nil), not an error.
func (m *MockMessageRepository) FindByExternalIDInConversation(
	ctx context.Context, externalID, conversationID string,
) (*entity.Message, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, msg := range m.Messages {
		if msg.ExternalID == externalID && msg.ConversationID == conversationID {
			return msg, nil
		}
	}
	return nil, nil
}

func (m *MockMessageRepository) LastInboundAt(ctx context.Context, conversationID string) (*time.Time, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var last *time.Time
	for _, msg := range m.Messages {
		if msg.ConversationID == conversationID && msg.SenderType == entity.SenderTypeContact {
			t := msg.CreatedAt
			if last == nil || t.After(*last) {
				last = &t
			}
		}
	}
	return last, nil
}

func (m *MockMessageRepository) FindByConversation(ctx context.Context, conversationID string, params *repository.ListParams) ([]*entity.Message, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Message
	for _, msg := range m.Messages {
		if msg.ConversationID == conversationID {
			result = append(result, msg)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockMessageRepository) Update(ctx context.Context, message *entity.Message) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Messages[message.ID] = message
	return nil
}

func (m *MockMessageRepository) UpdateStatus(ctx context.Context, id string, status entity.MessageStatus, errorMessage string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	msg, ok := m.Messages[id]
	if !ok {
		return fmt.Errorf("message not found: %s", id)
	}
	msg.Status = status
	msg.ErrorMessage = errorMessage
	return nil
}

func (m *MockMessageRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Messages, id)
	return nil
}

func (m *MockMessageRepository) CountByConversation(ctx context.Context, conversationID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, msg := range m.Messages {
		if msg.ConversationID == conversationID {
			count++
		}
	}
	return count, nil
}

func (m *MockMessageRepository) CountUnreadByConversation(ctx context.Context, conversationID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, msg := range m.Messages {
		if msg.ConversationID == conversationID && msg.Status != entity.MessageStatusRead {
			count++
		}
	}
	return count, nil
}

func (m *MockMessageRepository) MarkAsRead(ctx context.Context, conversationID string, upToMessageID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	return nil
}

func (m *MockMessageRepository) CreateAttachment(ctx context.Context, attachment *entity.MessageAttachment) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Attachments[attachment.MessageID] = append(m.Attachments[attachment.MessageID], attachment)
	return nil
}

func (m *MockMessageRepository) FindAttachmentsByMessage(ctx context.Context, messageID string) ([]*entity.MessageAttachment, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	attachments, ok := m.Attachments[messageID]
	if !ok {
		return []*entity.MessageAttachment{}, nil
	}
	return attachments, nil
}

// ============================================================================
// MockChannelRepository
// ============================================================================

// MockChannelRepository is a mock implementation of repository.ChannelRepository
type MockChannelRepository struct {
	Channels    map[string]*entity.Channel
	ReturnError error
	// UpdateCalls counts Update invocations so tests can assert a rejected
	// operation never reached the repository.
	UpdateCalls int
}

// NewMockChannelRepository creates a new MockChannelRepository
func NewMockChannelRepository() *MockChannelRepository {
	return &MockChannelRepository{
		Channels: make(map[string]*entity.Channel),
	}
}

func (m *MockChannelRepository) Create(ctx context.Context, channel *entity.Channel) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Channels[channel.ID] = channel
	return nil
}

func (m *MockChannelRepository) FindByID(ctx context.Context, id string) (*entity.Channel, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	ch, ok := m.Channels[id]
	if !ok {
		return nil, fmt.Errorf("channel not found: %s", id)
	}
	return ch, nil
}

func (m *MockChannelRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Channel, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Channel
	for _, ch := range m.Channels {
		if ch.TenantID == tenantID {
			result = append(result, ch)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockChannelRepository) FindByType(ctx context.Context, tenantID string, channelType entity.ChannelType) ([]*entity.Channel, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Channel
	for _, ch := range m.Channels {
		if ch.TenantID == tenantID && ch.Type == channelType {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (m *MockChannelRepository) FindAllByType(ctx context.Context, channelType entity.ChannelType) ([]*entity.Channel, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Channel
	for _, ch := range m.Channels {
		if ch.Type == channelType {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (m *MockChannelRepository) FindEnabledByTenant(ctx context.Context, tenantID string) ([]*entity.Channel, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Channel
	for _, ch := range m.Channels {
		if ch.TenantID == tenantID && ch.Enabled {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (m *MockChannelRepository) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Channel, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Channel
	for _, ch := range m.Channels {
		if ch.TenantID == tenantID && ch.IsActive() {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (m *MockChannelRepository) Update(ctx context.Context, channel *entity.Channel) error {
	m.UpdateCalls++
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Channels[channel.ID] = channel
	return nil
}

func (m *MockChannelRepository) UpdateEnabled(ctx context.Context, id string, enabled bool) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	ch, ok := m.Channels[id]
	if !ok {
		return fmt.Errorf("channel not found: %s", id)
	}
	ch.Enabled = enabled
	return nil
}

func (m *MockChannelRepository) UpdateConnectionStatus(ctx context.Context, id string, status entity.ConnectionStatus) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	ch, ok := m.Channels[id]
	if !ok {
		return fmt.Errorf("channel not found: %s", id)
	}
	ch.ConnectionStatus = status
	return nil
}

func (m *MockChannelRepository) UpdateStatus(ctx context.Context, id string, status entity.ChannelStatus) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	ch, ok := m.Channels[id]
	if !ok {
		return fmt.Errorf("channel not found: %s", id)
	}
	ch.ConnectionStatus = status
	return nil
}

func (m *MockChannelRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Channels, id)
	return nil
}

func (m *MockChannelRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, ch := range m.Channels {
		if ch.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockChannelRepository) FindByTypes(ctx context.Context, types []entity.ChannelType) ([]*entity.Channel, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Channel
	for _, ch := range m.Channels {
		for _, t := range types {
			if ch.Type == t {
				result = append(result, ch)
				break
			}
		}
	}
	return result, nil
}

func (m *MockChannelRepository) FindCoexistenceChannels(ctx context.Context) ([]*entity.Channel, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Channel
	for _, ch := range m.Channels {
		if ch.IsCoexistence {
			result = append(result, ch)
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// MockSandboxAllowlistRepository
// ---------------------------------------------------------------------------

// MockSandboxAllowlistRepository is a mock implementation of
// repository.SandboxAllowlistRepository. IsAllowedCalls counts allowlist
// consultations so tests can assert the delivery guard queries at send time
// instead of capturing/caching a result.
type MockSandboxAllowlistRepository struct {
	Entries        map[string]*entity.SandboxAllowlistEntry
	ReturnError    error
	IsAllowedCalls int
}

// NewMockSandboxAllowlistRepository creates a new MockSandboxAllowlistRepository.
func NewMockSandboxAllowlistRepository() *MockSandboxAllowlistRepository {
	return &MockSandboxAllowlistRepository{Entries: make(map[string]*entity.SandboxAllowlistEntry)}
}

func (m *MockSandboxAllowlistRepository) Create(ctx context.Context, entry *entity.SandboxAllowlistEntry) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Entries[entry.ID] = entry
	return nil
}

func (m *MockSandboxAllowlistRepository) FindByID(ctx context.Context, tenantID, id string) (*entity.SandboxAllowlistEntry, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	entry, ok := m.Entries[id]
	if !ok || entry.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "sandbox allowlist entry not found")
	}
	return entry, nil
}

func (m *MockSandboxAllowlistRepository) FindByTenant(ctx context.Context, tenantID string) ([]*entity.SandboxAllowlistEntry, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.SandboxAllowlistEntry
	for _, e := range m.Entries {
		if e.TenantID == tenantID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *MockSandboxAllowlistRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	entry, ok := m.Entries[id]
	if !ok || entry.TenantID != tenantID {
		return errors.New(errors.ErrCodeNotFound, "sandbox allowlist entry not found")
	}
	delete(m.Entries, id)
	return nil
}

func (m *MockSandboxAllowlistRepository) IsAllowed(ctx context.Context, tenantID, channelID, recipient string) (bool, error) {
	m.IsAllowedCalls++
	if m.ReturnError != nil {
		return false, m.ReturnError
	}
	for _, e := range m.Entries {
		if e.TenantID == tenantID && e.Recipient == recipient &&
			(e.ChannelID == "" || e.ChannelID == channelID) {
			return true, nil
		}
	}
	return false, nil
}
