package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// SendMessageInput represents input for sending a message
type SendMessageInput struct {
	TenantID       string
	ConversationID string
	SenderID       string
	SenderType     entity.SenderType
	ContentType    entity.ContentType
	Content        string
	Metadata       map[string]string
	Attachments    []*AttachmentInput
	QuickReplies   []entity.QuickReply // Interactive buttons/options
}

// AttachmentInput represents an attachment to be sent
type AttachmentInput struct {
	Type         string
	URL          string
	Filename     string
	MimeType     string
	SizeBytes    int64
	ThumbnailURL string
}

// SendMessageOutput represents the result of sending a message
type SendMessageOutput struct {
	Message      *entity.Message
	Conversation *entity.Conversation
}

// SendMessageUseCase handles sending messages
type SendMessageUseCase struct {
	messageRepo      repository.MessageRepository
	conversationRepo repository.ConversationRepository
	channelRepo      repository.ChannelRepository
	contactRepo      repository.ContactRepository
	producer         *nats.Producer
}

// NewSendMessageUseCase creates a new send message use case
func NewSendMessageUseCase(
	messageRepo repository.MessageRepository,
	conversationRepo repository.ConversationRepository,
	channelRepo repository.ChannelRepository,
	contactRepo repository.ContactRepository,
	producer *nats.Producer,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		channelRepo:      channelRepo,
		contactRepo:      contactRepo,
		producer:         producer,
	}
}

// Execute sends a message
func (uc *SendMessageUseCase) Execute(ctx context.Context, input *SendMessageInput) (*SendMessageOutput, error) {
	// Validate input
	if input.ConversationID == "" {
		return nil, errors.Validation("conversation_id is required")
	}
	if input.Content == "" && len(input.Attachments) == 0 {
		return nil, errors.Validation("content or attachments required")
	}

	// Get conversation
	conversation, err := uc.conversationRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, err
	}

	// Verify tenant ownership
	if conversation.TenantID != input.TenantID {
		return nil, errors.Forbidden("conversation does not belong to tenant")
	}

	// Get channel
	channel, err := uc.channelRepo.FindByID(ctx, conversation.ChannelID)
	if err != nil {
		return nil, err
	}

	// Check channel is active (enabled + connected)
	if !channel.IsActive() {
		return nil, errors.New(errors.ErrCodeChannelDisconnected, "channel is not active")
	}

	// Get contact
	contact, err := uc.contactRepo.FindByID(ctx, conversation.ContactID)
	if err != nil {
		return nil, err
	}

	// Find recipient identifier for the channel
	recipientID := uc.findRecipientID(ctx, contact, string(channel.Type))

	// Create message entity
	now := time.Now()
	message := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: input.ConversationID,
		SenderType:     input.SenderType,
		SenderID:       input.SenderID,
		ContentType:    input.ContentType,
		Content:        input.Content,
		Metadata:       input.Metadata,
		Status:         entity.MessageStatusPending,
		Attachments:    make([]*entity.MessageAttachment, 0),
		CreatedAt:      now,
	}

	if message.Metadata == nil {
		message.Metadata = make(map[string]string)
	}

	// Handle quick replies - convert to interactive message for supported channels
	if len(input.QuickReplies) > 0 && channelSupportsInteractive(channel.Type) {
		message.ContentType = entity.ContentTypeInteractive
		interactiveJSON := buildInteractiveFromQuickReplies(input.Content, input.QuickReplies)
		message.Metadata["interactive"] = interactiveJSON
		message.Metadata["interactive_type"] = getInteractiveType(len(input.QuickReplies))
	}

	// Create attachments
	for _, att := range input.Attachments {
		attachment := &entity.MessageAttachment{
			ID:           uuid.New().String(),
			MessageID:    message.ID,
			Type:         att.Type,
			URL:          att.URL,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
			SizeBytes:    att.SizeBytes,
			ThumbnailURL: att.ThumbnailURL,
			Metadata:     make(map[string]string),
			CreatedAt:    now,
		}
		message.Attachments = append(message.Attachments, attachment)
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

	// Publish to NATS for channel delivery
	outbound := &nats.OutboundMessage{
		ID:             message.ID,
		TenantID:       input.TenantID,
		ChannelID:      conversation.ChannelID,
		ChannelType:    string(channel.Type),
		ConversationID: conversation.ID,
		ContactID:      contact.ID,
		RecipientID:    recipientID,
		ContentType:    string(message.ContentType),
		Content:        message.Content,
		Metadata:       message.Metadata,
		Attachments:    uc.toNATSAttachments(message.Attachments),
		Timestamp:      now,
	}

	if err := uc.producer.PublishOutbound(ctx, outbound); err != nil {
		// Update message status to failed
		uc.messageRepo.UpdateStatus(ctx, message.ID, entity.MessageStatusFailed, err.Error())
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to publish message")
	}

	// Track first reply if this is from an agent
	if input.SenderType == entity.SenderTypeUser && conversation.FirstReplyAt == nil {
		now := time.Now()
		conversation.FirstReplyAt = &now
		uc.conversationRepo.Update(ctx, conversation)
	}

	return &SendMessageOutput{
		Message:      message,
		Conversation: conversation,
	}, nil
}

func (uc *SendMessageUseCase) findRecipientID(ctx context.Context, contact *entity.Contact, channelType string) string {
	// Try to find identity for this channel type
	identities, err := uc.contactRepo.FindIdentitiesByContact(ctx, contact.ID)
	if err == nil {
		for _, identity := range identities {
			if identity.ChannelType == channelType {
				return identity.Identifier
			}
		}
	}

	// Fallback to phone or email
	if contact.Phone != "" {
		return contact.Phone
	}
	if contact.Email != "" {
		return contact.Email
	}

	return ""
}

func (uc *SendMessageUseCase) toNATSAttachments(attachments []*entity.MessageAttachment) []nats.AttachmentData {
	result := make([]nats.AttachmentData, 0, len(attachments))
	for _, att := range attachments {
		result = append(result, nats.AttachmentData{
			Type:         att.Type,
			URL:          att.URL,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
			SizeBytes:    att.SizeBytes,
			ThumbnailURL: att.ThumbnailURL,
			Metadata:     att.Metadata,
		})
	}
	return result
}

// channelSupportsInteractive checks if a channel type supports interactive messages
func channelSupportsInteractive(channelType entity.ChannelType) bool {
	switch channelType {
	case entity.ChannelTypeWhatsApp, entity.ChannelTypeWhatsAppOfficial:
		return true
	case entity.ChannelTypeTelegram:
		return true // Telegram supports inline keyboards
	default:
		return false
	}
}

// getInteractiveType returns the interactive type based on number of options
func getInteractiveType(optionCount int) string {
	if optionCount <= 3 {
		return "button"
	}
	return "list"
}

// buildInteractiveFromQuickReplies creates interactive message JSON from quick replies
func buildInteractiveFromQuickReplies(bodyText string, quickReplies []entity.QuickReply) string {
	if len(quickReplies) == 0 {
		return ""
	}

	// Build based on count - buttons for ≤3, list for >3
	if len(quickReplies) <= 3 {
		// Button format
		buttons := make([]map[string]interface{}, 0, len(quickReplies))
		for _, qr := range quickReplies {
			title := qr.Title
			if len(title) > 20 {
				title = title[:20] // WhatsApp limit
			}
			buttons = append(buttons, map[string]interface{}{
				"type": "reply",
				"reply": map[string]string{
					"id":    qr.ID,
					"title": title,
				},
			})
		}

		interactive := map[string]interface{}{
			"type": "button",
			"body": map[string]string{
				"text": bodyText,
			},
			"action": map[string]interface{}{
				"buttons": buttons,
			},
		}

		data, _ := json.Marshal(interactive)
		return string(data)
	}

	// List format for more than 3 options
	rows := make([]map[string]string, 0, len(quickReplies))
	for _, qr := range quickReplies {
		title := qr.Title
		if len(title) > 24 {
			title = title[:24] // WhatsApp limit for list items
		}
		id := qr.ID
		if len(id) > 200 {
			id = id[:200]
		}
		rows = append(rows, map[string]string{
			"id":    id,
			"title": title,
		})
	}

	interactive := map[string]interface{}{
		"type": "list",
		"body": map[string]string{
			"text": bodyText,
		},
		"action": map[string]interface{}{
			"button":   "Options",
			"sections": []map[string]interface{}{
				{
					"rows": rows,
				},
			},
		},
	}

	data, _ := json.Marshal(interactive)
	return string(data)
}
