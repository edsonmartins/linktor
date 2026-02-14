package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// MessageRepository defines the interface for message persistence
type MessageRepository interface {
	// Create creates a new message
	Create(ctx context.Context, message *entity.Message) error

	// FindByID finds a message by ID
	FindByID(ctx context.Context, id string) (*entity.Message, error)

	// FindByExternalID finds a message by external ID (from channel provider)
	FindByExternalID(ctx context.Context, externalID string) (*entity.Message, error)

	// FindByConversation finds messages for a conversation with pagination
	FindByConversation(ctx context.Context, conversationID string, params *ListParams) ([]*entity.Message, int64, error)

	// Update updates a message
	Update(ctx context.Context, message *entity.Message) error

	// UpdateStatus updates only the message status
	UpdateStatus(ctx context.Context, id string, status entity.MessageStatus, errorMessage string) error

	// Delete deletes a message
	Delete(ctx context.Context, id string) error

	// CountByConversation counts messages in a conversation
	CountByConversation(ctx context.Context, conversationID string) (int64, error)

	// CountUnreadByConversation counts unread messages in a conversation
	CountUnreadByConversation(ctx context.Context, conversationID string) (int64, error)

	// MarkAsRead marks messages as read up to a given message ID
	MarkAsRead(ctx context.Context, conversationID string, upToMessageID string) error

	// CreateAttachment creates a message attachment
	CreateAttachment(ctx context.Context, attachment *entity.MessageAttachment) error

	// FindAttachmentsByMessage finds attachments for a message
	FindAttachmentsByMessage(ctx context.Context, messageID string) ([]*entity.MessageAttachment, error)
}

// ConversationRepository defines the interface for conversation persistence
type ConversationRepository interface {
	// Create creates a new conversation
	Create(ctx context.Context, conversation *entity.Conversation) error

	// FindByID finds a conversation by ID
	FindByID(ctx context.Context, id string) (*entity.Conversation, error)

	// FindByTenant finds conversations for a tenant with pagination
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.Conversation, int64, error)

	// FindByChannel finds conversations for a channel
	FindByChannel(ctx context.Context, channelID string, params *ListParams) ([]*entity.Conversation, int64, error)

	// FindByContact finds conversations for a contact
	FindByContact(ctx context.Context, contactID string, params *ListParams) ([]*entity.Conversation, int64, error)

	// FindByAssignee finds conversations assigned to a user
	FindByAssignee(ctx context.Context, assigneeID string, params *ListParams) ([]*entity.Conversation, int64, error)

	// FindOpenByContactAndChannel finds open conversation for a contact on a channel
	FindOpenByContactAndChannel(ctx context.Context, contactID, channelID string) (*entity.Conversation, error)

	// Update updates a conversation
	Update(ctx context.Context, conversation *entity.Conversation) error

	// UpdateStatus updates only the conversation status
	UpdateStatus(ctx context.Context, id string, status entity.ConversationStatus) error

	// UpdateAssignee updates the conversation assignee
	UpdateAssignee(ctx context.Context, id string, assigneeID *string) error

	// IncrementUnreadCount increments the unread message count
	IncrementUnreadCount(ctx context.Context, id string) error

	// ResetUnreadCount resets the unread message count to zero
	ResetUnreadCount(ctx context.Context, id string) error

	// Delete deletes a conversation
	Delete(ctx context.Context, id string) error

	// CountByTenant counts conversations for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)

	// CountByStatus counts conversations by status for a tenant
	CountByStatus(ctx context.Context, tenantID string, status entity.ConversationStatus) (int64, error)

	// CountActiveByUser counts active conversations assigned to a user
	CountActiveByUser(ctx context.Context, userID string) (int64, error)

	// CountWaiting counts waiting conversations with given or higher priority
	CountWaiting(ctx context.Context, tenantID string, minPriority entity.ConversationPriority) (int64, error)
}

// ContactRepository defines the interface for contact persistence
type ContactRepository interface {
	// Create creates a new contact
	Create(ctx context.Context, contact *entity.Contact) error

	// FindByID finds a contact by ID
	FindByID(ctx context.Context, id string) (*entity.Contact, error)

	// FindByTenant finds contacts for a tenant with pagination
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.Contact, int64, error)

	// FindByEmail finds a contact by email within a tenant
	FindByEmail(ctx context.Context, tenantID, email string) (*entity.Contact, error)

	// FindByPhone finds a contact by phone within a tenant
	FindByPhone(ctx context.Context, tenantID, phone string) (*entity.Contact, error)

	// FindByIdentity finds a contact by channel identity
	FindByIdentity(ctx context.Context, tenantID, channelType, identifier string) (*entity.Contact, error)

	// Update updates a contact
	Update(ctx context.Context, contact *entity.Contact) error

	// Delete deletes a contact
	Delete(ctx context.Context, id string) error

	// CountByTenant counts contacts for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)

	// AddIdentity adds a channel identity to a contact
	AddIdentity(ctx context.Context, identity *entity.ContactIdentity) error

	// RemoveIdentity removes a channel identity from a contact
	RemoveIdentity(ctx context.Context, contactID, identityID string) error

	// FindIdentitiesByContact finds all identities for a contact
	FindIdentitiesByContact(ctx context.Context, contactID string) ([]*entity.ContactIdentity, error)
}

// ChannelRepository defines the interface for channel persistence
type ChannelRepository interface {
	// Create creates a new channel
	Create(ctx context.Context, channel *entity.Channel) error

	// FindByID finds a channel by ID
	FindByID(ctx context.Context, id string) (*entity.Channel, error)

	// FindByTenant finds channels for a tenant with pagination
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.Channel, int64, error)

	// FindByType finds channels of a specific type for a tenant
	FindByType(ctx context.Context, tenantID string, channelType entity.ChannelType) ([]*entity.Channel, error)

	// FindEnabledByTenant finds enabled channels for a tenant
	FindEnabledByTenant(ctx context.Context, tenantID string) ([]*entity.Channel, error)

	// FindActiveByTenant finds channels that are both enabled AND connected
	FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Channel, error)

	// Update updates a channel
	Update(ctx context.Context, channel *entity.Channel) error

	// UpdateEnabled updates only the channel enabled state
	UpdateEnabled(ctx context.Context, id string, enabled bool) error

	// UpdateConnectionStatus updates only the channel connection status
	UpdateConnectionStatus(ctx context.Context, id string, status entity.ConnectionStatus) error

	// UpdateStatus updates the channel status (deprecated, use UpdateEnabled or UpdateConnectionStatus)
	UpdateStatus(ctx context.Context, id string, status entity.ChannelStatus) error

	// Delete deletes a channel
	Delete(ctx context.Context, id string) error

	// CountByTenant counts channels for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)

	// FindByTypes finds all channels of specific types across all tenants
	FindByTypes(ctx context.Context, types []entity.ChannelType) ([]*entity.Channel, error)
}

