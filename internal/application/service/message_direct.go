package service

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
)

// MetadataIdempotencyKey is the metadata field carrying the caller's logical
// send key. Uniqueness is (tenant_id, idempotency_key) — see
// repository.MessageIdempotencyRepository.
const MetadataIdempotencyKey = "idempotency_key"

// MaxIdempotencyKeyLength mirrors the column width of
// message_idempotency_keys.idempotency_key. Validated up front so an oversized
// key is a 400, not a driver error surfacing as 500.
const MaxIdempotencyKeyLength = 255

// abandonedReservationAge is how long a reservation whose message never
// appeared is treated as merely in flight. Past it, the reservation is assumed
// orphaned by a crashed process and can be reclaimed. Comfortably longer than
// any single send, and short enough that a retry is not blocked for long.
const abandonedReservationAge = 5 * time.Minute

// directMediaContentTypes are the content types the direct route can deliver
// beyond text. Each is carried entirely by its attachments, so the route needs
// no payload model of its own for them — which is exactly why location,
// contact, template, interactive, poll and reaction are absent.
var directMediaContentTypes = map[string]bool{
	string(entity.ContentTypeImage):    true,
	string(entity.ContentTypeVideo):    true,
	string(entity.ContentTypeAudio):    true,
	string(entity.ContentTypeDocument): true,
	string(entity.ContentTypeSticker):  true,
}

// DirectSendInput is a channel+recipient send that does not require the caller
// to know a conversation: the identity, contact and conversation are resolved
// (or created) inside the tenant before the message is persisted and published.
type DirectSendInput struct {
	TenantID    string
	ChannelID   string
	To          string
	ContentType string
	Content     string
	// SenderID identifies the acting principal (the API key's user id). Recorded
	// on the message so a direct send is attributable like any other.
	SenderID string
	Metadata map[string]string
	// Attachments carries already-uploaded media, exactly as the conversation
	// route does. Without it the direct route could only send text, which forced
	// an integrator with no conversation in hand to keep a second messaging stack
	// alive just for images and audio.
	Attachments []MessageAttachmentInput
}

// DirectSendResult is the canonical Linktor answer to a direct send.
// Idempotent is true when the call matched an existing idempotency key and no
// new message was created.
type DirectSendResult struct {
	MessageID      string
	ConversationID string
	ChannelID      string
	ContactID      string
	Status         entity.MessageStatus
	Message        *entity.Message
	Idempotent     bool
}

// SetIdempotencyStore wires the store backing metadata.idempotency_key on the
// direct-send route. Optional at composition time: without it the key is still
// persisted on the message, but repeated calls are not collapsed.
func (s *MessageService) SetIdempotencyStore(repo repository.MessageIdempotencyRepository) {
	s.idempotency = repo
}

// SendDirect sends a message to a recipient on a channel, creating the identity,
// contact and conversation as needed.
//
// It deliberately funnels into Send rather than publishing to the adapter
// directly: retry, DLQ, the sandbox guard, status tracking and metadata
// propagation all live on the outbound path, and a second publisher would have
// none of them.
func (s *MessageService) SendDirect(ctx context.Context, input *DirectSendInput) (*DirectSendResult, error) {
	if input == nil {
		return nil, errors.Validation("request body is required")
	}
	if input.TenantID == "" {
		return nil, errors.Validation("tenant is required")
	}
	if input.ChannelID == "" {
		return nil, errors.Validation("channel_id is required")
	}
	recipient := strings.TrimSpace(input.To)
	if recipient == "" {
		return nil, errors.Validation("to is required")
	}

	contentType := strings.TrimSpace(input.ContentType)
	hasAttachments := len(input.Attachments) > 0
	if contentType == "" {
		// An attachment with no declared type is a document: it is the one kind
		// every transport can carry, so guessing it degrades presentation at
		// worst, never delivery.
		if hasAttachments {
			contentType = string(entity.ContentTypeDocument)
		} else {
			contentType = string(entity.ContentTypeText)
		}
	}
	// Media types are allowed, but only with something to deliver. The types left
	// out (location, contact, template, interactive, poll, reaction) need
	// structured payloads this route does not model, and silently degrading them
	// to text would deliver the wrong thing.
	if contentType != string(entity.ContentTypeText) {
		if !directMediaContentTypes[contentType] {
			return nil, errors.Validation(
				"content_type must be \"text\" or one of: image, video, audio, document, sticker")
		}
		if !hasAttachments {
			// Without this the message goes out declared as an image carrying
			// nothing — the recipient gets an empty bubble and the send still
			// reports success.
			return nil, errors.Validation("attachments are required for content_type \"" + contentType + "\"")
		}
	}
	if strings.TrimSpace(input.Content) == "" && !hasAttachments {
		return nil, errors.Validation("text is required")
	}
	for i, a := range input.Attachments {
		if strings.TrimSpace(a.URL) == "" {
			return nil, errors.Validation("attachments[" + strconv.Itoa(i) + "].url is required")
		}
	}

	// Channel must belong to the tenant. A channel from another tenant is
	// reported as missing, never as forbidden — existence itself is tenant data.
	channel, err := s.channelRepo.FindByID(ctx, input.ChannelID)
	if err != nil || channel == nil || channel.TenantID != input.TenantID {
		return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}
	if !channel.IsEnabled() {
		return nil, errors.New(errors.ErrCodeChannelDisconnected, "channel is not enabled")
	}
	if !channel.IsConnected() {
		return nil, errors.New(errors.ErrCodeChannelDisconnected, "channel is not connected")
	}

	metadata := copyMetadata(input.Metadata)
	idempotencyKey := strings.TrimSpace(metadata[MetadataIdempotencyKey])
	if len(idempotencyKey) > MaxIdempotencyKeyLength {
		// The column is VARCHAR(255); without this the driver's "value too long"
		// surfaces as a 500 for what is plainly a bad request.
		return nil, errors.Validation("metadata.idempotency_key must be at most " +
			strconv.Itoa(MaxIdempotencyKeyLength) + " characters")
	}

	// Sandbox is checked here, before anything is written: Send re-checks it, but
	// by then the contact and conversation exist and their "created" events are
	// queued. A blocked send would leave that debris behind on every attempt.
	if channel.IsSandbox() {
		if err := s.checkSandboxRecipient(ctx, channel, CanonicalRecipient(string(channel.Type), recipient)); err != nil {
			return nil, err
		}
	}

	// Pre-allocate the message id so the idempotency reservation names the
	// message it protects before that message exists.
	messageID := uuid.New().String()
	if idempotencyKey != "" && s.idempotency != nil {
		existing, err := s.idempotency.Reserve(ctx, input.TenantID, idempotencyKey, messageID)
		if err != nil {
			return nil, err
		}
		if existing != "" {
			result, err := s.resolveExistingClaim(ctx, input.TenantID, idempotencyKey, existing, messageID, channel.ID)
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}
			// Reclaimed an abandoned reservation: we now own the key and fall
			// through to send under our own message id.
		}
	}

	result, err := s.sendDirectReserved(ctx, input, channel, recipient, contentType, metadata, messageID)
	if err != nil {
		// The send failed, so the key protects nothing. Releasing it lets the
		// integrator retry with the same key instead of being answered forever
		// with a message that was never created.
		if idempotencyKey != "" && s.idempotency != nil {
			if relErr := s.idempotency.Release(ctx, input.TenantID, idempotencyKey, messageID); relErr != nil {
				logger.Warn("envio direto: falha ao liberar idempotency_key após erro: " + relErr.Error())
			}
		}
		return nil, err
	}
	return result, nil
}

// sendDirectReserved performs the resolution + send once the idempotency key (if
// any) is already reserved for messageID.
func (s *MessageService) sendDirectReserved(
	ctx context.Context,
	input *DirectSendInput,
	channel *entity.Channel,
	recipient, contentType string,
	metadata map[string]string,
	messageID string,
) (*DirectSendResult, error) {
	contact, err := s.resolveDirectContact(ctx, channel, recipient)
	if err != nil {
		return nil, err
	}

	conversation, err := s.resolveDirectConversation(ctx, channel, contact.ID)
	if err != nil {
		return nil, err
	}

	message, err := s.Send(ctx, &SendMessageInput{
		TenantID:       input.TenantID,
		MessageID:      messageID,
		ConversationID: conversation.ID,
		// Pin the address the caller named. The contact may carry more than one
		// identity for this channel type, and the identity query has no ORDER BY,
		// so re-deriving could deliver to a different number than requested.
		RecipientID: CanonicalRecipient(string(channel.Type), recipient),
		SenderID:    input.SenderID,
		SenderType:  string(entity.SenderTypeSystem),
		ContentType: contentType,
		Content:     input.Content,
		Metadata:    metadata,
		Attachments: input.Attachments,
	})
	if err != nil {
		return nil, err
	}

	return &DirectSendResult{
		MessageID:      message.ID,
		ConversationID: conversation.ID,
		ChannelID:      channel.ID,
		ContactID:      contact.ID,
		Status:         message.Status,
		Message:        message,
	}, nil
}

// resolveExistingClaim decides what a repeated idempotency key means.
//
// The happy case is a claim whose message exists: that is the original send, and
// returning it is the idempotent answer. The awkward case is a claim naming a
// message that does not exist, which is either a concurrent first request that
// has not committed yet, or a reservation abandoned by a process that died
// between reserving and writing. Answering "success" for the latter would be a
// permanent lie — the row has no TTL, so every future retry of that key would
// get 202 for a message that was never sent.
//
// So: an old orphan is reclaimed (returns nil, nil — the caller sends under its
// own id), and a young one is reported as in flight so the caller retries. Both
// are honest; neither invents a delivery.
func (s *MessageService) resolveExistingClaim(
	ctx context.Context, tenantID, key, claimedMessageID, ourMessageID, channelID string,
) (*DirectSendResult, error) {
	message, err := s.messageRepo.FindByID(ctx, claimedMessageID)
	if err == nil && message != nil {
		return &DirectSendResult{
			MessageID:      message.ID,
			ConversationID: message.ConversationID,
			ChannelID:      channelID,
			Status:         message.Status,
			Message:        message,
			Idempotent:     true,
		}, nil
	}

	reclaimed, err := s.idempotency.ReclaimIfStale(ctx, tenantID, key, ourMessageID,
		time.Now().Add(-abandonedReservationAge))
	if err != nil {
		return nil, err
	}
	if reclaimed {
		return nil, nil
	}
	return nil, errors.New(errors.ErrCodeConflict,
		"a send with this idempotency key is in flight; retry shortly")
}

// resolveDirectContact finds the contact that owns the recipient identity on
// this channel type inside the tenant, creating contact+identity when it is the
// first time we address them.
func (s *MessageService) resolveDirectContact(ctx context.Context, channel *entity.Channel, recipient string) (*entity.Contact, error) {
	channelType := string(channel.Type)
	canonical := CanonicalRecipient(channelType, recipient)

	for _, candidate := range recipientCandidates(channelType, recipient) {
		if contact, err := s.contactRepo.FindByIdentity(ctx, channel.TenantID, channelType, candidate); err == nil && contact != nil {
			return contact, nil
		}
	}

	// No identity on this channel type yet: the person may still exist as a
	// contact reached through another channel, in which case we attach the new
	// identity instead of duplicating them.
	if contact := s.findContactByAddress(ctx, channel.TenantID, channelType, recipient); contact != nil {
		identity := &entity.ContactIdentity{
			ID:          uuid.New().String(),
			ContactID:   contact.ID,
			TenantID:    channel.TenantID,
			ChannelType: channelType,
			Identifier:  canonical,
			CreatedAt:   time.Now(),
		}
		if _, err := s.contactRepo.CreateIdentityIfAbsent(ctx, identity); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to attach contact identity")
		}
		contact.Identities = append(contact.Identities, identity)
		return contact, nil
	}

	now := time.Now()
	contact := &entity.Contact{
		ID:           uuid.New().String(),
		TenantID:     channel.TenantID,
		CustomFields: make(map[string]string),
		Tags:         []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if isEmailChannel(channelType) {
		contact.Email = canonical
	} else if _, ok := entity.NormalizeE164(recipient); ok {
		contact.Phone = canonical
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create contact")
	}

	identity := &entity.ContactIdentity{
		ID:          uuid.New().String(),
		ContactID:   contact.ID,
		TenantID:    channel.TenantID,
		ChannelType: channelType,
		Identifier:  canonical,
		CreatedAt:   now,
	}
	// The tenant-scoped unique index on (tenant_id, channel_type, identifier)
	// arbitrates a race with a concurrent inbound message from the same person:
	// the loser discards the contact it just created and reuses the winner, so
	// one person never ends up with two contact rows.
	event, err := directContactCreatedEvent(channel, contact)
	if err != nil {
		return nil, err
	}
	inserted, err := s.contactRepo.CreateIdentityIfAbsentWithOutboxEvent(ctx, identity, event)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create contact identity")
	}
	if !inserted {
		winner, findErr := s.contactRepo.FindByIdentity(ctx, channel.TenantID, channelType, canonical)
		if findErr == nil && winner != nil {
			_ = s.contactRepo.Delete(ctx, contact.ID)
			return winner, nil
		}
		// Could not resolve the winner: keep our contact so the send is not
		// lost. It simply carries no identity, and Send falls back to the
		// contact's phone/email for the recipient.
		return contact, nil
	}

	contact.Identities = append(contact.Identities, identity)
	return contact, nil
}

// findContactByAddress looks the recipient up as a plain address (email or
// phone) on contacts of the tenant, so a person already known through another
// channel is reused rather than duplicated.
func (s *MessageService) findContactByAddress(ctx context.Context, tenantID, channelType, recipient string) *entity.Contact {
	if isEmailChannel(channelType) || IsEmailRecipient(recipient) {
		if contact, err := s.contactRepo.FindByEmail(ctx, tenantID, strings.ToLower(strings.TrimSpace(recipient))); err == nil && contact != nil {
			return contact
		}
		return nil
	}
	for _, candidate := range phoneCandidates(recipient) {
		if contact, err := s.contactRepo.FindByPhone(ctx, tenantID, candidate); err == nil && contact != nil {
			return contact
		}
	}
	return nil
}

// resolveDirectConversation reuses the contact's open conversation on the
// channel, or opens one (emitting conversation.created through the
// transactional outbox, exactly like inbound intake does).
func (s *MessageService) resolveDirectConversation(ctx context.Context, channel *entity.Channel, contactID string) (*entity.Conversation, error) {
	existing, err := s.conversationRepo.FindOpenByContactAndChannel(ctx, contactID, channel.ID)
	if err == nil && existing != nil {
		if existing.TenantID != channel.TenantID {
			return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
		}
		return existing, nil
	}

	now := time.Now()
	conversation := &entity.Conversation{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ContactID:   contactID,
		Environment: channel.Environment, // denormalized at birth (INV-018)
		Status:      entity.ConversationStatusOpen,
		Priority:    entity.ConversationPriorityNormal,
		Metadata:    make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	event, err := directConversationCreatedEvent(conversation)
	if err != nil {
		return nil, err
	}
	if err := s.conversationRepo.CreateWithOutboxEvent(ctx, conversation, event); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create conversation")
	}
	return conversation, nil
}

// CanonicalRecipient is the stored form of a recipient for a channel type. It
// matches what the inbound adapters persist, so a direct send and a later
// inbound message from the same person resolve to the same identity:
//   - email channels: lowercased address;
//   - phone numbers: digits only, no "+" (whatsmeow builds a JID from this and
//     the Cloud API accepts it verbatim);
//   - anything else (JIDs, Slack/Teams user ids): unchanged.
func CanonicalRecipient(channelType, recipient string) string {
	r := strings.TrimSpace(recipient)
	if isEmailChannel(channelType) {
		return strings.ToLower(r)
	}
	if e164, ok := entity.NormalizeE164(r); ok {
		return strings.TrimPrefix(e164, "+")
	}
	return r
}

// recipientCandidates lists the identity identifiers that may already represent
// this recipient, most canonical first. A phone can legitimately be stored with
// or without "+" depending on which adapter first saw it.
func recipientCandidates(channelType, recipient string) []string {
	canonical := CanonicalRecipient(channelType, recipient)
	out := []string{canonical}
	for _, c := range append([]string{strings.TrimSpace(recipient)}, phoneCandidates(recipient)...) {
		if c != "" && c != canonical && !contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

// phoneCandidates returns the "+55..." / "55..." pair for a phone-like value,
// or nil when the value is not a phone number at all.
func phoneCandidates(recipient string) []string {
	e164, ok := entity.NormalizeE164(recipient)
	if !ok {
		return nil
	}
	return []string{strings.TrimPrefix(e164, "+"), e164}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func isEmailChannel(channelType string) bool {
	return channelType == string(entity.ChannelTypeEmail)
}

// emailAddress matches a plain address well enough to tell an email recipient
// from a phone or a channel-specific id.
var emailAddress = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// IsEmailRecipient reports whether a recipient looks like an email address.
func IsEmailRecipient(recipient string) bool {
	return emailAddress.MatchString(strings.TrimSpace(recipient))
}

func copyMetadata(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func directContactCreatedEvent(channel *entity.Channel, contact *entity.Contact) (*entity.OutboxEvent, error) {
	return newDirectOutboxEvent(nats.EventContactCreated, channel.TenantID, "contact", contact.ID,
		"evt-contact-created-"+contact.ID, map[string]interface{}{
			"contact_id":   contact.ID,
			"name":         contact.Name,
			"phone":        contact.Phone,
			"email":        contact.Email,
			"channel_id":   channel.ID,
			"channel_type": string(channel.Type),
		})
}

func directConversationCreatedEvent(conversation *entity.Conversation) (*entity.OutboxEvent, error) {
	return newDirectOutboxEvent(nats.EventConversationCreated, conversation.TenantID, "conversation", conversation.ID,
		"evt-conversation-created-"+conversation.ID, map[string]interface{}{
			"conversation_id": conversation.ID,
			"channel_id":      conversation.ChannelID,
			"contact_id":      conversation.ContactID,
		})
}

func newDirectOutboxEvent(eventType, tenantID, aggregateType, aggregateID, idempotencyKey string, payload map[string]interface{}) (*entity.OutboxEvent, error) {
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
