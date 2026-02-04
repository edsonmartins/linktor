package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// ReceiveMessageOutput represents the result of receiving a message
type ReceiveMessageOutput struct {
	Message      *entity.Message
	Conversation *entity.Conversation
	Contact      *entity.Contact
	IsNew        bool
}

// ReceiveMessageUseCase handles receiving messages from channels
type ReceiveMessageUseCase struct {
	messageRepo      repository.MessageRepository
	conversationRepo repository.ConversationRepository
	channelRepo      repository.ChannelRepository
	contactRepo      repository.ContactRepository
	producer         *nats.Producer
	normalizer       *service.MessageNormalizer
}

// NewReceiveMessageUseCase creates a new receive message use case
func NewReceiveMessageUseCase(
	messageRepo repository.MessageRepository,
	conversationRepo repository.ConversationRepository,
	channelRepo repository.ChannelRepository,
	contactRepo repository.ContactRepository,
	producer *nats.Producer,
	normalizer *service.MessageNormalizer,
) *ReceiveMessageUseCase {
	return &ReceiveMessageUseCase{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		channelRepo:      channelRepo,
		contactRepo:      contactRepo,
		producer:         producer,
		normalizer:       normalizer,
	}
}

// Execute processes an incoming message from a channel
func (uc *ReceiveMessageUseCase) Execute(ctx context.Context, inbound *nats.InboundMessage) (*ReceiveMessageOutput, error) {
	// Check for duplicate message
	if inbound.ExternalID != "" {
		existing, err := uc.messageRepo.FindByExternalID(ctx, inbound.ExternalID)
		if err == nil && existing != nil {
			// Message already processed
			return nil, errors.New(errors.ErrCodeConflict, "message already exists")
		}
	}

	// Normalize the inbound message
	normalized := uc.normalizer.NormalizeInbound(inbound)

	// Get or create contact
	contact, _, err := uc.getOrCreateContact(ctx, inbound)
	if err != nil {
		return nil, err
	}
	normalized.ContactID = contact.ID

	// Get channel
	channel, err := uc.channelRepo.FindByID(ctx, inbound.ChannelID)
	if err != nil {
		return nil, err
	}

	// Get or create conversation
	conversation, isNewConversation, err := uc.getOrCreateConversation(ctx, inbound.TenantID, channel.ID, contact.ID)
	if err != nil {
		return nil, err
	}
	normalized.ConversationID = conversation.ID

	// Create message entity
	message := uc.normalizer.ToEntity(normalized)
	message.ID = uuid.New().String()

	// Set attachment message IDs
	for _, att := range message.Attachments {
		att.MessageID = message.ID
	}

	// Save message to database
	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	// Save attachments
	for _, att := range message.Attachments {
		if err := uc.messageRepo.CreateAttachment(ctx, att); err != nil {
			return nil, err
		}
	}

	// Update conversation
	if err := uc.conversationRepo.IncrementUnreadCount(ctx, conversation.ID); err != nil {
		// Log error but don't fail
	}

	// Reopen conversation if it was resolved
	if conversation.Status == entity.ConversationStatusResolved || conversation.Status == entity.ConversationStatusClosed {
		conversation.Status = entity.ConversationStatusOpen
		conversation.ResolvedAt = nil
		uc.conversationRepo.Update(ctx, conversation)
	}

	// Publish event
	uc.publishMessageReceivedEvent(ctx, inbound.TenantID, message, conversation, contact)

	return &ReceiveMessageOutput{
		Message:      message,
		Conversation: conversation,
		Contact:      contact,
		IsNew:        isNewConversation,
	}, nil
}

// getOrCreateContact finds or creates a contact based on the inbound message
func (uc *ReceiveMessageUseCase) getOrCreateContact(ctx context.Context, inbound *nats.InboundMessage) (*entity.Contact, bool, error) {
	// Extract identifier from metadata or external ID
	identifier := inbound.ExternalID
	if id, ok := inbound.Metadata["sender_id"]; ok {
		identifier = id
	}
	if phone, ok := inbound.Metadata["phone"]; ok {
		identifier = phone
	}

	// Try to find existing contact by identity
	contact, err := uc.contactRepo.FindByIdentity(ctx, inbound.TenantID, inbound.ChannelType, identifier)
	if err == nil && contact != nil {
		return contact, false, nil
	}

	// Try to find by phone if available
	if phone, ok := inbound.Metadata["phone"]; ok && phone != "" {
		contact, err = uc.contactRepo.FindByPhone(ctx, inbound.TenantID, phone)
		if err == nil && contact != nil {
			// Add identity for this channel
			identity := &entity.ContactIdentity{
				ID:          uuid.New().String(),
				ContactID:   contact.ID,
				ChannelType: inbound.ChannelType,
				Identifier:  identifier,
				Metadata:    inbound.Metadata,
				CreatedAt:   time.Now(),
			}
			uc.contactRepo.AddIdentity(ctx, identity)
			return contact, false, nil
		}
	}

	// Create new contact
	now := time.Now()
	name := "Unknown"
	if n, ok := inbound.Metadata["sender_name"]; ok {
		name = n
	}
	if n, ok := inbound.Metadata["name"]; ok {
		name = n
	}

	phone := ""
	if p, ok := inbound.Metadata["phone"]; ok {
		phone = p
	}

	contact = &entity.Contact{
		ID:           uuid.New().String(),
		TenantID:     inbound.TenantID,
		Name:         name,
		Phone:        phone,
		CustomFields: make(map[string]string),
		Tags:         []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.contactRepo.Create(ctx, contact); err != nil {
		return nil, false, err
	}

	// Add identity
	identity := &entity.ContactIdentity{
		ID:          uuid.New().String(),
		ContactID:   contact.ID,
		ChannelType: inbound.ChannelType,
		Identifier:  identifier,
		Metadata:    inbound.Metadata,
		CreatedAt:   now,
	}
	if err := uc.contactRepo.AddIdentity(ctx, identity); err != nil {
		// Log but continue
	}

	// Publish contact created event
	uc.publishContactCreatedEvent(ctx, inbound.TenantID, contact)

	return contact, true, nil
}

// getOrCreateConversation finds or creates a conversation
func (uc *ReceiveMessageUseCase) getOrCreateConversation(ctx context.Context, tenantID, channelID, contactID string) (*entity.Conversation, bool, error) {
	// Try to find open conversation
	conversation, err := uc.conversationRepo.FindOpenByContactAndChannel(ctx, contactID, channelID)
	if err == nil && conversation != nil {
		return conversation, false, nil
	}

	// Create new conversation
	now := time.Now()
	conversation = &entity.Conversation{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ChannelID:   channelID,
		ContactID:   contactID,
		Status:      entity.ConversationStatusOpen,
		Priority:    entity.ConversationPriorityNormal,
		UnreadCount: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, false, err
	}

	// Publish conversation created event
	uc.publishConversationCreatedEvent(ctx, tenantID, conversation)

	return conversation, true, nil
}

func (uc *ReceiveMessageUseCase) publishMessageReceivedEvent(ctx context.Context, tenantID string, message *entity.Message, conversation *entity.Conversation, contact *entity.Contact) {
	event := &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: tenantID,
		Payload: map[string]interface{}{
			"message_id":      message.ID,
			"conversation_id": conversation.ID,
			"contact_id":      contact.ID,
			"content_type":    string(message.ContentType),
			"content":         message.Content,
		},
		Timestamp: time.Now(),
	}
	uc.producer.PublishEvent(ctx, event)
}

func (uc *ReceiveMessageUseCase) publishContactCreatedEvent(ctx context.Context, tenantID string, contact *entity.Contact) {
	event := &nats.Event{
		Type:     nats.EventContactCreated,
		TenantID: tenantID,
		Payload: map[string]interface{}{
			"contact_id": contact.ID,
			"name":       contact.Name,
			"phone":      contact.Phone,
			"email":      contact.Email,
		},
		Timestamp: time.Now(),
	}
	uc.producer.PublishEvent(ctx, event)
}

func (uc *ReceiveMessageUseCase) publishConversationCreatedEvent(ctx context.Context, tenantID string, conversation *entity.Conversation) {
	event := &nats.Event{
		Type:     nats.EventConversationCreated,
		TenantID: tenantID,
		Payload: map[string]interface{}{
			"conversation_id": conversation.ID,
			"channel_id":      conversation.ChannelID,
			"contact_id":      conversation.ContactID,
		},
		Timestamp: time.Now(),
	}
	uc.producer.PublishEvent(ctx, event)
}
