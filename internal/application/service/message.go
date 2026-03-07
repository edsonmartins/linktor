package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// SendMessageInput represents input for sending a message
type SendMessageInput struct {
	ConversationID string
	SenderID       string
	SenderType     string
	ContentType    string
	Content        string
	Metadata       map[string]string
}

// MessageService handles message operations
type MessageService struct {
	messageRepo      repository.MessageRepository
	conversationRepo repository.ConversationRepository
	channelRepo      repository.ChannelRepository
	contactRepo      repository.ContactRepository
	producer         *nats.Producer
}

// NewMessageService creates a new message service
func NewMessageService(
	messageRepo repository.MessageRepository,
	conversationRepo repository.ConversationRepository,
	channelRepo repository.ChannelRepository,
	contactRepo repository.ContactRepository,
	producer *nats.Producer,
) *MessageService {
	return &MessageService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		channelRepo:      channelRepo,
		contactRepo:      contactRepo,
		producer:         producer,
	}
}

// ListByConversation returns all messages for a conversation
func (s *MessageService) ListByConversation(ctx context.Context, conversationID string, params *repository.ListParams) ([]*entity.Message, int64, error) {
	if params == nil {
		params = repository.NewListParams()
		params.PageSize = 50
		params.SortBy = "created_at"
		params.SortDir = "desc"
	}
	return s.messageRepo.FindByConversation(ctx, conversationID, params)
}

// Send sends a new message
func (s *MessageService) Send(ctx context.Context, input *SendMessageInput) (*entity.Message, error) {
	if input.ConversationID == "" {
		return nil, errors.Validation("conversation_id is required")
	}
	if input.Content == "" {
		return nil, errors.Validation("content is required")
	}

	// Get conversation
	conversation, err := s.conversationRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	// Get channel
	channel, err := s.channelRepo.FindByID(ctx, conversation.ChannelID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}

	// Get contact for recipient ID
	contact, err := s.contactRepo.FindByID(ctx, conversation.ContactID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	// Load identities for recipient lookup
	identities, idErr := s.contactRepo.FindIdentitiesByContact(ctx, contact.ID)
	if idErr == nil {
		contact.Identities = identities
	}

	now := time.Now()
	message := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: input.ConversationID,
		SenderType:     entity.SenderType(input.SenderType),
		SenderID:       input.SenderID,
		ContentType:    entity.ContentType(input.ContentType),
		Content:        input.Content,
		Metadata:       input.Metadata,
		Status:         entity.MessageStatusPending,
		Attachments:    make([]*entity.MessageAttachment, 0),
		CreatedAt:      now,
	}

	if message.Metadata == nil {
		message.Metadata = make(map[string]string)
	}

	// Save message to database
	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create message")
	}

	// Publish to NATS for channel delivery (if producer is available)
	if s.producer != nil {
		recipientID := findRecipientForChannel(contact, string(channel.Type))
		outbound := &nats.OutboundMessage{
			ID:             message.ID,
			TenantID:       conversation.TenantID,
			ChannelID:      conversation.ChannelID,
			ChannelType:    string(channel.Type),
			ConversationID: conversation.ID,
			ContactID:      contact.ID,
			RecipientID:    recipientID,
			ContentType:    string(message.ContentType),
			Content:        message.Content,
			Metadata:       message.Metadata,
			Timestamp:      now,
		}

		if err := s.producer.PublishOutbound(ctx, outbound); err != nil {
			s.messageRepo.UpdateStatus(ctx, message.ID, entity.MessageStatusFailed, err.Error())
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to publish message")
		}
	}

	// Track first reply if this is from a user/agent
	if entity.SenderType(input.SenderType) == entity.SenderTypeUser && conversation.FirstReplyAt == nil {
		conversation.FirstReplyAt = &now
		s.conversationRepo.Update(ctx, conversation)
	}

	return message, nil
}

// GetByID returns a message by ID
func (s *MessageService) GetByID(ctx context.Context, id string) (*entity.Message, error) {
	message, err := s.messageRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeMessageNotFound, "message not found")
	}
	return message, nil
}

// UpdateStatus updates a message status
func (s *MessageService) UpdateStatus(ctx context.Context, id string, status entity.MessageStatus, errorMessage string) (*entity.Message, error) {
	if err := s.messageRepo.UpdateStatus(ctx, id, status, errorMessage); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update message status")
	}

	message, err := s.messageRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeMessageNotFound, "message not found")
	}

	return message, nil
}

// SendReaction sends a reaction to a message
// If emoji is empty, the reaction is removed
func (s *MessageService) SendReaction(ctx context.Context, conversationID, messageID, emoji, senderID string) error {
	// Get the original message to find external_id
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return errors.New(errors.ErrCodeMessageNotFound, "message not found")
	}

	// Get the conversation to find the channel
	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	// Publish reaction event to NATS for the adapter to send
	if s.producer != nil {
		event := &nats.Event{
			Type:     "message.reaction",
			TenantID: conversation.TenantID,
			Payload: map[string]interface{}{
				"channel_id":      conversation.ChannelID,
				"message_id":      messageID,
				"external_id":     message.ExternalID,
				"conversation_id": conversationID,
				"emoji":           emoji,
				"sender_id":       senderID,
			},
			Timestamp: time.Now(),
		}

		if err := s.producer.PublishEvent(ctx, event); err != nil {
			return errors.Wrap(err, errors.ErrCodeInternal, "failed to publish reaction")
		}
	}

	return nil
}

// findRecipientForChannel finds the recipient identifier for a given channel type
func findRecipientForChannel(contact *entity.Contact, channelType string) string {
	identity := contact.GetIdentityByChannel(channelType)
	if identity != nil {
		return identity.Identifier
	}
	if contact.Phone != "" {
		return contact.Phone
	}
	if contact.Email != "" {
		return contact.Email
	}
	return ""
}
