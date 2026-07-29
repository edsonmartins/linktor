package usecase

import (
	"context"
	"encoding/json"
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
	producer         nats.Publisher
	normalizer       *service.MessageNormalizer
	assignmentSvc    *service.AssignmentService // optional; auto-routes new conversations
}

// SetAssignmentService enables automatic conversation assignment. Optional so
// existing constructors and tests are unaffected.
func (uc *ReceiveMessageUseCase) SetAssignmentService(s *service.AssignmentService) {
	uc.assignmentSvc = s
}

// NewReceiveMessageUseCase creates a new receive message use case
func NewReceiveMessageUseCase(
	messageRepo repository.MessageRepository,
	conversationRepo repository.ConversationRepository,
	channelRepo repository.ChannelRepository,
	contactRepo repository.ContactRepository,
	producer nats.Publisher,
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
	if inbound == nil {
		return nil, errors.Validation("inbound message is required")
	}
	if inbound.ChannelID == "" {
		return nil, errors.Validation("channel_id is required")
	}
	if inbound.Metadata == nil {
		inbound.Metadata = make(map[string]string)
	}

	// Get channel first and treat it as the source of truth for tenant/type.
	channel, err := uc.channelRepo.FindByID(ctx, inbound.ChannelID)
	if err != nil {
		return nil, err
	}
	if inbound.TenantID != "" && inbound.TenantID != channel.TenantID {
		return nil, errors.Forbidden("inbound message tenant does not match channel")
	}
	if inbound.ChannelType != "" && inbound.ChannelType != string(channel.Type) {
		return nil, errors.Validation("inbound message channel_type does not match channel")
	}
	if !channel.IsActive() {
		return nil, errors.New(errors.ErrCodeChannelDisconnected, "channel is not active")
	}
	inbound.TenantID = channel.TenantID
	inbound.ChannelType = string(channel.Type)

	// Fast-path duplicate check to skip the contact/conversation work for an
	// obvious redelivery. This is an optimization only — CreateWithOutboxEvent
	// below is the authoritative dedup (ON CONFLICT), and the original delivery's
	// outbox event is durably queued, so a duplicate needs no re-publish here.
	if inbound.ExternalID != "" {
		existing, err := uc.messageRepo.FindByExternalID(ctx, inbound.ExternalID)
		if err == nil && existing != nil {
			existingConversation, convErr := uc.conversationRepo.FindByID(ctx, existing.ConversationID)
			if convErr != nil || existingConversation.ChannelID == channel.ID {
				return nil, errors.New(errors.ErrCodeConflict, "message already exists")
			}
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

	// Get or create conversation
	conversation, isNewConversation, err := uc.getOrCreateConversation(ctx, channel, contact.ID)
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

	// Persist the message and enqueue its "received" event in ONE transaction
	// (transactional outbox): the event is durably queued iff the message
	// committed, and the relay publishes it — no dual-write window, no silent
	// loss. A duplicate delivery is the ON CONFLICT no-op (inserted=false), whose
	// original outbox event is already queued, so we just ack it as a conflict.
	outboxEvent, err := uc.buildMessageReceivedOutboxEvent(inbound.TenantID, channel, message, conversation, contact)
	if err != nil {
		return nil, err
	}
	inserted, err := uc.messageRepo.CreateWithOutboxEvent(ctx, message, outboxEvent)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return nil, errors.New(errors.ErrCodeConflict, "message already exists")
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
		// The channel may now supply a profile name the contact was created
		// without (e.g. the first message arrived before WhatsApp had synced the
		// pushName). Backfill it so the contact stops showing as "Unknown".
		uc.backfillContactName(ctx, contact, inbound)
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
				TenantID:    inbound.TenantID,
				ChannelType: inbound.ChannelType,
				Identifier:  identifier,
				Metadata:    inbound.Metadata,
				CreatedAt:   time.Now(),
			}
			uc.contactRepo.AddIdentity(ctx, identity)
			uc.backfillContactName(ctx, contact, inbound)
			return contact, false, nil
		}
	}

	// Create new contact. Leave the name empty when the channel supplied none —
	// the UI localizes an empty name to "Unknown Contact"; a hardcoded "Unknown"
	// string would both be un-localized and block the backfill above forever.
	now := time.Now()
	name := inboundName(inbound)

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

	// Add identity atomically. The tenant-scoped unique index on
	// (tenant_id, channel_type, identifier) is the arbiter of the race: if a
	// concurrent inbound webhook for the same brand-new contact already inserted
	// this identity, our insert is a no-op (inserted=false). In that case we
	// discard the contact we just created and reuse the winner, so one person
	// never ends up with two contact rows.
	identity := &entity.ContactIdentity{
		ID:          uuid.New().String(),
		ContactID:   contact.ID,
		TenantID:    inbound.TenantID,
		ChannelType: inbound.ChannelType,
		Identifier:  identifier,
		Metadata:    inbound.Metadata,
		CreatedAt:   now,
	}
	// Enqueue the "contact created" event in the SAME transaction as the winning
	// identity insert: the event is durably queued iff we actually secured this
	// contact's identity. A caller that loses the race enqueues nothing.
	contactEvent, err := buildContactCreatedOutboxEvent(inbound.TenantID, inbound.ChannelID, string(inbound.ChannelType), contact)
	if err != nil {
		return nil, false, err
	}
	inserted, err := uc.contactRepo.CreateIdentityIfAbsentWithOutboxEvent(ctx, identity, contactEvent)
	if err != nil {
		return nil, false, err
	}
	if !inserted {
		// Lost the race: another request already owns this identity. Fetch the
		// winning contact and roll back the orphan we just created.
		winner, findErr := uc.contactRepo.FindByIdentity(ctx, inbound.TenantID, inbound.ChannelType, identifier)
		if findErr == nil && winner != nil {
			if delErr := uc.contactRepo.Delete(ctx, contact.ID); delErr != nil {
				// Best effort: the identity insert already failed, so the
				// orphan carries no identity and will not be found again.
			}
			return winner, false, nil
		}
		// Could not resolve the winner; fall through and keep our contact so the
		// message is not lost. The identity simply was not attached, and no
		// ContactCreated event was enqueued for this degenerate path.
	}

	return contact, true, nil
}

// inboundName returns the display name the channel supplied for the sender
// (e.g. WhatsApp profile.name / pushName), or "" when none was provided.
func inboundName(inbound *nats.InboundMessage) string {
	// An explicit "name" takes precedence over the channel-derived "sender_name".
	if n, ok := inbound.Metadata["name"]; ok && n != "" {
		return n
	}
	if n, ok := inbound.Metadata["sender_name"]; ok && n != "" {
		return n
	}
	return ""
}

// nameIsPlaceholder reports whether a stored contact name is missing or the
// legacy hardcoded "Unknown" placeholder, and so should be backfilled from the
// channel-provided name when one becomes available.
func nameIsPlaceholder(name string) bool {
	return name == "" || name == "Unknown"
}

// backfillContactName upgrades an existing contact's name from the channel when
// the contact currently has no real name and the inbound message carries one.
// Best-effort: a failed persist must not block message ingestion, and the
// in-memory contact still carries the resolved name for this message.
func (uc *ReceiveMessageUseCase) backfillContactName(ctx context.Context, contact *entity.Contact, inbound *nats.InboundMessage) {
	if !nameIsPlaceholder(contact.Name) {
		return
	}
	name := inboundName(inbound)
	if name == "" {
		return
	}
	contact.Name = name
	contact.UpdatedAt = time.Now()
	_ = uc.contactRepo.Update(ctx, contact)
}

// getOrCreateConversation finds or creates a conversation. It takes the
// resolved channel so a new conversation is born carrying the channel's
// environment (INV-018) — the denormalization point for sandbox marking.
func (uc *ReceiveMessageUseCase) getOrCreateConversation(ctx context.Context, channel *entity.Channel, contactID string) (*entity.Conversation, bool, error) {
	tenantID := channel.TenantID
	// Try to find open conversation
	conversation, err := uc.conversationRepo.FindOpenByContactAndChannel(ctx, contactID, channel.ID)
	if err == nil && conversation != nil {
		if conversation.TenantID != tenantID {
			return nil, false, errors.Forbidden("conversation does not belong to tenant")
		}
		return conversation, false, nil
	}

	// Create new conversation
	now := time.Now()
	conversation = &entity.Conversation{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ChannelID:   channel.ID,
		ContactID:   contactID,
		Environment: channel.Environment,
		Status:      entity.ConversationStatusOpen,
		Priority:    entity.ConversationPriorityNormal,
		UnreadCount: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Persist the conversation and enqueue its "created" event atomically
	// (transactional outbox), so the event is published iff the conversation
	// committed and never depends on NATS being up at intake time.
	conversationEvent, err := buildConversationCreatedOutboxEvent(tenantID, conversation)
	if err != nil {
		return nil, false, err
	}
	if err := uc.conversationRepo.CreateWithOutboxEvent(ctx, conversation, conversationEvent); err != nil {
		return nil, false, err
	}

	// Auto-assign to an agent if the tenant enabled it (best-effort).
	if uc.assignmentSvc != nil {
		if _, err := uc.assignmentSvc.AutoAssign(ctx, conversation); err != nil {
			// Routing must never block message intake.
			_ = err
		}
	}

	return conversation, true, nil
}

// buildMessageReceivedOutboxEvent builds the durable outbox entry for the
// "message received" event. Its payload carries everything the outbound-webhook
// dispatcher needs to build the `linktor-channel-v1` envelope without extra DB
// reads. The idempotency key is stable per message so a relay re-publish is
// deduplicated by JetStream.
func (uc *ReceiveMessageUseCase) buildMessageReceivedOutboxEvent(tenantID string, channel *entity.Channel, message *entity.Message, conversation *entity.Conversation, contact *entity.Contact) (*entity.OutboxEvent, error) {
	payload := map[string]interface{}{
		"message_id":      message.ID,
		"conversation_id": conversation.ID,
		"contact_id":      contact.ID,
		"channel_id":      channel.ID,
		"channel_type":    string(channel.Type),
		"environment":     string(channel.Environment),
		"content_type":    string(message.ContentType),
		"content":         message.Content,
		"external_id":     message.ExternalID,
		"sender_id":       message.Metadata["sender_id"],
		"sender_name":     message.Metadata["sender_name"],
		// Grupo (quando o adaptador marca): o consumidor thread-eia pela conversa de
		// grupo; sender_id acima continua sendo o indivíduo que falou. Vazio em 1:1.
		"is_group": message.Metadata["is_group"],
		"chat_jid": message.Metadata["chat_jid"],
	}
	if atts := attachmentsPayload(message.Attachments); len(atts) > 0 {
		payload["attachments"] = atts
	}
	// Reação de canal (RFC-010): propaga o alvo/emoji ao consumidor (o bridge VendaX monta a
	// reação inbound). Sem isto a reação chega como texto solto (emoji) sem alvo.
	if message.Metadata["is_reaction"] == "true" {
		payload["is_reaction"] = "true"
		payload["reaction_message_id"] = message.Metadata["reaction_message_id"]
	}

	return newOutboxEvent(nats.EventMessageReceived, tenantID, "message", message.ID,
		"evt-message-received-"+message.ID, payload)
}

// attachmentsPayload flattens message attachments into the plain maps carried
// on the event payload (and later round-tripped through NATS as JSON).
func attachmentsPayload(attachments []*entity.MessageAttachment) []map[string]interface{} {
	if len(attachments) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(attachments))
	for _, att := range attachments {
		out = append(out, map[string]interface{}{
			"url":        att.URL,
			"mime_type":  att.MimeType,
			"filename":   att.Filename,
			"size_bytes": att.SizeBytes,
		})
	}
	return out
}

// newOutboxEvent builds an outbox entry with a JSON-marshaled payload. All three
// inbound events (message received, contact created, conversation created) route
// through the transactional outbox so they are published iff their aggregate
// committed, and re-published safely (JetStream dedups on the idempotency key).
func newOutboxEvent(eventType, tenantID, aggregateType, aggregateID, idempotencyKey string, payload map[string]interface{}) (*entity.OutboxEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal event payload")
	}
	return &entity.OutboxEvent{
		ID:             uuid.New().String(),
		EventType:      eventType,
		TenantID:       tenantID,
		AggregateType:  aggregateType,
		AggregateID:    aggregateID,
		Payload:        data,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
	}, nil
}

func buildContactCreatedOutboxEvent(tenantID, channelID, channelType string, contact *entity.Contact) (*entity.OutboxEvent, error) {
	return newOutboxEvent(nats.EventContactCreated, tenantID, "contact", contact.ID,
		"evt-contact-created-"+contact.ID, map[string]interface{}{
			"contact_id":   contact.ID,
			"name":         contact.Name,
			"phone":        contact.Phone,
			"email":        contact.Email,
			"channel_id":   channelID,
			"channel_type": channelType,
		})
}

func buildConversationCreatedOutboxEvent(tenantID string, conversation *entity.Conversation) (*entity.OutboxEvent, error) {
	return newOutboxEvent(nats.EventConversationCreated, tenantID, "conversation", conversation.ID,
		"evt-conversation-created-"+conversation.ID, map[string]interface{}{
			"conversation_id": conversation.ID,
			"channel_id":      conversation.ChannelID,
			"contact_id":      conversation.ContactID,
		})
}
