package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
)

// helper to build the common test fixtures
type receiveMessageFixture struct {
	messageRepo      *testutil.MockMessageRepository
	conversationRepo *testutil.MockConversationRepository
	channelRepo      *testutil.MockChannelRepository
	contactRepo      *testutil.MockContactRepository
	producer         *testutil.MockProducer
	normalizer       *service.MessageNormalizer
	uc               *ReceiveMessageUseCase
}

func newReceiveMessageFixture() *receiveMessageFixture {
	messageRepo := testutil.NewMockMessageRepository()
	conversationRepo := testutil.NewMockConversationRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()
	producer := testutil.NewMockProducer()
	normalizer := service.NewMessageNormalizer()

	uc := NewReceiveMessageUseCase(messageRepo, conversationRepo, channelRepo, contactRepo, producer, normalizer)

	return &receiveMessageFixture{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		channelRepo:      channelRepo,
		contactRepo:      contactRepo,
		producer:         producer,
		normalizer:       normalizer,
		uc:               uc,
	}
}

func makeChannel(id, tenantID string) *entity.Channel {
	now := time.Now()
	return &entity.Channel{
		ID:               id,
		TenantID:         tenantID,
		Type:             entity.ChannelTypeWhatsApp,
		Name:             "Test Channel",
		Identifier:       "+5511999999999",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config:           make(map[string]string),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func makeInbound(channelID, tenantID string) *nats.InboundMessage {
	return &nats.InboundMessage{
		ID:          "inbound-1",
		TenantID:    tenantID,
		ChannelID:   channelID,
		ChannelType: "whatsapp",
		ExternalID:  "ext-123",
		ContentType: "text",
		Content:     "Hello, world!",
		Metadata: map[string]string{
			"phone":       "+5511888888888",
			"sender_name": "John Doe",
		},
		Timestamp: time.Now(),
	}
}

func TestReceiveMessageUseCase(t *testing.T) {
	ctx := context.Background()

	t.Run("Happy Path - New Contact + New Conversation", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// Contact created
		assert.NotEmpty(t, output.Contact.ID)
		assert.Equal(t, "tenant-1", output.Contact.TenantID)
		assert.Equal(t, "John Doe", output.Contact.Name)
		assert.Equal(t, "+5511888888888", output.Contact.Phone)

		// Conversation created
		assert.NotEmpty(t, output.Conversation.ID)
		assert.Equal(t, "tenant-1", output.Conversation.TenantID)
		assert.Equal(t, "ch-1", output.Conversation.ChannelID)
		assert.Equal(t, output.Contact.ID, output.Conversation.ContactID)
		assert.Equal(t, entity.ConversationStatusOpen, output.Conversation.Status)

		// Message saved
		assert.NotEmpty(t, output.Message.ID)
		assert.Equal(t, output.Conversation.ID, output.Message.ConversationID)
		assert.Equal(t, entity.ContentTypeText, output.Message.ContentType)
		assert.Equal(t, "Hello, world!", output.Message.Content)
		assert.Equal(t, entity.SenderTypeContact, output.Message.SenderType)
		assert.Equal(t, entity.MessageStatusPending, output.Message.Status)
		assert.Equal(t, "ext-123", output.Message.ExternalID)

		// IsNew should be true (new conversation)
		assert.True(t, output.IsNew)

		// ContactCreated + ConversationCreated publish directly; MessageReceived is
		// written to the transactional outbox (same tx as the message) and later
		// published by the relay — so it is NOT in producer.Events here.
		require.Len(t, f.producer.Events, 2)
		assert.Equal(t, nats.EventContactCreated, f.producer.Events[0].Type)
		assert.Equal(t, nats.EventConversationCreated, f.producer.Events[1].Type)

		require.Len(t, f.messageRepo.OutboxEvents, 1)
		assert.Equal(t, nats.EventMessageReceived, f.messageRepo.OutboxEvents[0].EventType)
		assert.Equal(t, "evt-message-received-"+output.Message.ID, f.messageRepo.OutboxEvents[0].IdempotencyKey)

		// Message persisted in repo
		assert.Len(t, f.messageRepo.Messages, 1)
	})

	t.Run("Existing Contact by Identity", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		// Pre-populate contact with matching identity
		existingContact := &entity.Contact{
			ID:           "contact-existing",
			TenantID:     "tenant-1",
			Name:         "Existing User",
			Phone:        "+5511888888888",
			CustomFields: make(map[string]string),
			Tags:         []string{},
			Identities: []*entity.ContactIdentity{
				{
					ID:          "identity-1",
					ContactID:   "contact-existing",
					ChannelType: "whatsapp",
					Identifier:  "+5511888888888",
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		f.contactRepo.Contacts[existingContact.ID] = existingContact

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// Existing contact used
		assert.Equal(t, "contact-existing", output.Contact.ID)
		assert.Equal(t, "Existing User", output.Contact.Name)

		// No new contact created (still just 1 in repo)
		assert.Len(t, f.contactRepo.Contacts, 1)

		// Only ConversationCreated publishes directly (no ContactCreated for an
		// existing contact); MessageReceived goes to the outbox.
		require.Len(t, f.producer.Events, 1)
		assert.Equal(t, nats.EventConversationCreated, f.producer.Events[0].Type)
		require.Len(t, f.messageRepo.OutboxEvents, 1)
		assert.Equal(t, nats.EventMessageReceived, f.messageRepo.OutboxEvents[0].EventType)
	})

	t.Run("Existing Contact by Phone - Adds Identity", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		// Pre-populate contact matching by phone, but no matching identity
		existingContact := &entity.Contact{
			ID:           "contact-phone",
			TenantID:     "tenant-1",
			Name:         "Phone User",
			Phone:        "+5511888888888",
			CustomFields: make(map[string]string),
			Tags:         []string{},
			Identities:   []*entity.ContactIdentity{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		f.contactRepo.Contacts[existingContact.ID] = existingContact

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// Existing contact used
		assert.Equal(t, "contact-phone", output.Contact.ID)

		// New identity added
		identities := f.contactRepo.Identities["contact-phone"]
		require.Len(t, identities, 1)
		assert.Equal(t, "whatsapp", identities[0].ChannelType)
		assert.Equal(t, "+5511888888888", identities[0].Identifier)

		// No ContactCreated event
		for _, evt := range f.producer.Events {
			assert.NotEqual(t, nats.EventContactCreated, evt.Type)
		}
	})

	// WS10-GETORCREATE: two inbound webhooks from the SAME brand-new contact
	// arriving concurrently must not create two contact rows. This drives the
	// identity race deterministically: the identity insert reports inserted=false
	// because a winner committed first, so the loser reuses the winner and rolls
	// back its orphan contact.
	t.Run("Concurrent New Contact - Identity Race Reuses Winner", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		// Simulate the winning request committing its contact + identity in the
		// window between the loser's find-miss and its own identity insert.
		winner := &entity.Contact{
			ID:       "contact-winner",
			TenantID: "tenant-1",
			Name:     "Winner",
			Phone:    "+5511888888888",
			Identities: []*entity.ContactIdentity{{
				ID:          "identity-winner",
				ContactID:   "contact-winner",
				TenantID:    "tenant-1",
				ChannelType: "whatsapp",
				Identifier:  "+5511888888888",
			}},
			CustomFields: make(map[string]string),
			Tags:         []string{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		var injected bool
		f.contactRepo.CreateIdentityHook = func() {
			if injected {
				return
			}
			injected = true
			f.contactRepo.Contacts[winner.ID] = winner
		}

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// The loser reused the winning contact, not its own freshly-created one.
		assert.Equal(t, "contact-winner", output.Contact.ID)

		// Exactly one contact remains: the orphan was rolled back.
		assert.Len(t, f.contactRepo.Contacts, 1)

		// No duplicate ContactCreated event for the loser's orphan.
		for _, evt := range f.producer.Events {
			assert.NotEqual(t, nats.EventContactCreated, evt.Type)
		}
	})

	// Same identity arriving twice sequentially reuses the existing contact via
	// the find fast path and never leaves a duplicate.
	t.Run("Repeated New Contact - No Duplicate", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		first, err := f.uc.Execute(ctx, makeInbound("ch-1", "tenant-1"))
		require.NoError(t, err)
		require.NotNil(t, first)
		assert.True(t, first.IsNew)

		second := makeInbound("ch-1", "tenant-1")
		second.ExternalID = "ext-456" // distinct message, same contact identity
		out, err := f.uc.Execute(ctx, second)
		require.NoError(t, err)
		require.NotNil(t, out)

		// Same contact reused, only one contact row.
		assert.Equal(t, first.Contact.ID, out.Contact.ID)
		assert.Len(t, f.contactRepo.Contacts, 1)
	})

	t.Run("Existing Open Conversation", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		existingContact := &entity.Contact{
			ID:       "contact-1",
			TenantID: "tenant-1",
			Name:     "Test User",
			Phone:    "+5511888888888",
			Identities: []*entity.ContactIdentity{
				{
					ID:          "id-1",
					ContactID:   "contact-1",
					ChannelType: "whatsapp",
					Identifier:  "+5511888888888",
				},
			},
			CustomFields: make(map[string]string),
			Tags:         []string{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		f.contactRepo.Contacts[existingContact.ID] = existingContact

		existingConv := &entity.Conversation{
			ID:          "conv-existing",
			TenantID:    "tenant-1",
			ChannelID:   "ch-1",
			ContactID:   "contact-1",
			Status:      entity.ConversationStatusOpen,
			Priority:    entity.ConversationPriorityNormal,
			UnreadCount: 0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		f.conversationRepo.Conversations[existingConv.ID] = existingConv

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// Existing conversation reused
		assert.Equal(t, "conv-existing", output.Conversation.ID)
		assert.False(t, output.IsNew)

		// Existing contact + existing conversation → nothing published directly;
		// MessageReceived is queued in the outbox.
		require.Len(t, f.producer.Events, 0)
		require.Len(t, f.messageRepo.OutboxEvents, 1)
		assert.Equal(t, nats.EventMessageReceived, f.messageRepo.OutboxEvents[0].EventType)
	})

	t.Run("Resolved Conversation - Creates New Conversation", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		existingContact := &entity.Contact{
			ID:       "contact-1",
			TenantID: "tenant-1",
			Name:     "Test User",
			Phone:    "+5511888888888",
			Identities: []*entity.ContactIdentity{
				{
					ID:          "id-1",
					ContactID:   "contact-1",
					ChannelType: "whatsapp",
					Identifier:  "+5511888888888",
				},
			},
			CustomFields: make(map[string]string),
			Tags:         []string{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		f.contactRepo.Contacts[existingContact.ID] = existingContact

		resolvedAt := time.Now().Add(-1 * time.Hour)
		resolvedConv := &entity.Conversation{
			ID:          "conv-resolved",
			TenantID:    "tenant-1",
			ChannelID:   "ch-1",
			ContactID:   "contact-1",
			Status:      entity.ConversationStatusResolved,
			Priority:    entity.ConversationPriorityNormal,
			UnreadCount: 0,
			ResolvedAt:  &resolvedAt,
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		}
		f.conversationRepo.Conversations[resolvedConv.ID] = resolvedConv

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// A new conversation is created (resolved is not "open")
		assert.NotEqual(t, "conv-resolved", output.Conversation.ID)
		assert.Equal(t, entity.ConversationStatusOpen, output.Conversation.Status)
		assert.True(t, output.IsNew)

		// ConversationCreated event published
		var hasConvCreated bool
		for _, evt := range f.producer.Events {
			if evt.Type == nats.EventConversationCreated {
				hasConvCreated = true
			}
		}
		assert.True(t, hasConvCreated)
	})

	t.Run("Deduplication - ExternalID Already Exists", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		// Pre-populate a message with the same ExternalID
		existingMsg := &entity.Message{
			ID:         "msg-existing",
			ExternalID: "ext-123",
			Content:    "old message",
		}
		f.messageRepo.Messages[existingMsg.ID] = existingMsg

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ExternalID = "ext-123"

		output, err := f.uc.Execute(ctx, inbound)
		assert.Nil(t, output)
		require.Error(t, err)

		// Should be a conflict error
		assert.Contains(t, err.Error(), "CONFLICT")
	})

	t.Run("Deduplication - Same ExternalID Different Channel Is Allowed", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel1 := makeChannel("ch-1", "tenant-1")
		channel2 := makeChannel("ch-2", "tenant-1")
		f.channelRepo.Channels[channel1.ID] = channel1
		f.channelRepo.Channels[channel2.ID] = channel2

		existingConv := &entity.Conversation{
			ID:        "conv-existing",
			TenantID:  "tenant-1",
			ChannelID: "ch-1",
			ContactID: "contact-existing",
			Status:    entity.ConversationStatusOpen,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		f.conversationRepo.Conversations[existingConv.ID] = existingConv
		f.messageRepo.Messages["msg-existing"] = &entity.Message{
			ID:             "msg-existing",
			ConversationID: "conv-existing",
			ExternalID:     "ext-123",
			Content:        "old message",
		}

		inbound := makeInbound("ch-2", "tenant-1")
		inbound.ExternalID = "ext-123"

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, "ch-2", output.Conversation.ChannelID)
	})

	t.Run("Rejects Tenant Mismatch For Channel", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-2")

		output, err := f.uc.Execute(ctx, inbound)
		assert.Nil(t, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "FORBIDDEN")
	})

	t.Run("Rejects Channel Type Mismatch", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ChannelType = "telegram"

		output, err := f.uc.Execute(ctx, inbound)
		assert.Nil(t, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "VALIDATION_ERROR")
	})

	t.Run("Rejects Inactive Channel", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		channel.ConnectionStatus = entity.ConnectionStatusDisconnected
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		assert.Nil(t, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CHANNEL_DISCONNECTED")
	})

	t.Run("Deduplication - Empty ExternalID Skips Check", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ExternalID = ""

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// Message processed normally
		assert.NotEmpty(t, output.Message.ID)
		assert.Equal(t, "Hello, world!", output.Message.Content)
	})

	t.Run("Message with Attachments", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ExternalID = "ext-attach"
		inbound.ContentType = "image"
		inbound.Content = "Photo caption"
		inbound.Attachments = []nats.AttachmentData{
			{
				Type:     "image",
				URL:      "https://example.com/photo.jpg",
				Filename: "photo.jpg",
				MimeType: "image/jpeg",
			},
		}

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// Attachments saved with correct MessageID
		attachments := f.messageRepo.Attachments[output.Message.ID]
		require.Len(t, attachments, 1)
		assert.Equal(t, output.Message.ID, attachments[0].MessageID)
		assert.Equal(t, "image", attachments[0].Type)
		assert.Equal(t, "https://example.com/photo.jpg", attachments[0].URL)
		assert.Equal(t, "photo.jpg", attachments[0].Filename)
		assert.Equal(t, "image/jpeg", attachments[0].MimeType)

		// Message entity also has attachments
		require.Len(t, output.Message.Attachments, 1)
		assert.Equal(t, output.Message.ID, output.Message.Attachments[0].MessageID)
	})

	t.Run("Error - Channel Not Found", func(t *testing.T) {
		f := newReceiveMessageFixture()
		// No channel added

		inbound := makeInbound("ch-nonexistent", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		assert.Nil(t, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel not found")
	})

	t.Run("Error - Contact Creation Failure", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		f.contactRepo.ReturnError = fmt.Errorf("database connection failed")

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		assert.Nil(t, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection failed")
	})

	t.Run("Error - Message Save Failure", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")

		// We need the contact and conversation to be created first, then fail on message save.
		// Since ReturnError affects all operations, we use a workaround:
		// set the error after initial setup won't work with a simple flag.
		// Instead, we pre-populate a contact so contactRepo succeeds,
		// then set messageRepo.ReturnError.
		existingContact := &entity.Contact{
			ID:       "contact-1",
			TenantID: "tenant-1",
			Name:     "Test User",
			Phone:    "+5511888888888",
			Identities: []*entity.ContactIdentity{
				{
					ID:          "id-1",
					ContactID:   "contact-1",
					ChannelType: "whatsapp",
					Identifier:  "+5511888888888",
				},
			},
			CustomFields: make(map[string]string),
			Tags:         []string{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		f.contactRepo.Contacts[existingContact.ID] = existingContact

		f.messageRepo.ReturnError = fmt.Errorf("disk full")

		output, err := f.uc.Execute(ctx, inbound)
		assert.Nil(t, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "disk full")
	})

	t.Run("Contact Name from Metadata - sender_name", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ExternalID = "ext-name-1"
		inbound.Metadata = map[string]string{
			"phone":       "+5511777777777",
			"sender_name": "Alice Sender",
		}

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		assert.Equal(t, "Alice Sender", output.Contact.Name)
	})

	t.Run("Contact Name from Metadata - name overrides sender_name", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ExternalID = "ext-name-2"
		inbound.Metadata = map[string]string{
			"phone":       "+5511666666666",
			"sender_name": "Sender Name",
			"name":        "Full Name",
		}

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		assert.Equal(t, "Full Name", output.Contact.Name)
	})

	t.Run("Contact Name Fallback to Unknown", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ExternalID = "ext-name-3"
		inbound.Metadata = map[string]string{
			"phone": "+5511555555555",
		}

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		assert.Equal(t, "Unknown", output.Contact.Name)
	})

	t.Run("Unread Count Increment", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		existingContact := &entity.Contact{
			ID:       "contact-1",
			TenantID: "tenant-1",
			Name:     "Test User",
			Phone:    "+5511888888888",
			Identities: []*entity.ContactIdentity{
				{
					ID:          "id-1",
					ContactID:   "contact-1",
					ChannelType: "whatsapp",
					Identifier:  "+5511888888888",
				},
			},
			CustomFields: make(map[string]string),
			Tags:         []string{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		f.contactRepo.Contacts[existingContact.ID] = existingContact

		existingConv := &entity.Conversation{
			ID:          "conv-1",
			TenantID:    "tenant-1",
			ChannelID:   "ch-1",
			ContactID:   "contact-1",
			Status:      entity.ConversationStatusOpen,
			Priority:    entity.ConversationPriorityNormal,
			UnreadCount: 5,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		f.conversationRepo.Conversations[existingConv.ID] = existingConv

		inbound := makeInbound("ch-1", "tenant-1")

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// UnreadCount should have been incremented from 5 to 6
		conv := f.conversationRepo.Conversations["conv-1"]
		assert.Equal(t, 6, conv.UnreadCount)
	})

	t.Run("Event Payload Verification - MessageReceived", func(t *testing.T) {
		f := newReceiveMessageFixture()
		channel := makeChannel("ch-1", "tenant-1")
		f.channelRepo.Channels[channel.ID] = channel

		inbound := makeInbound("ch-1", "tenant-1")
		inbound.ExternalID = "ext-payload"

		output, err := f.uc.Execute(ctx, inbound)
		require.NoError(t, err)
		require.NotNil(t, output)

		// The MessageReceived event is queued in the outbox; its payload is the
		// exact JSON the relay will publish.
		require.Len(t, f.messageRepo.OutboxEvents, 1)
		outEvt := f.messageRepo.OutboxEvents[0]
		assert.Equal(t, nats.EventMessageReceived, outEvt.EventType)
		assert.Equal(t, "tenant-1", outEvt.TenantID)
		assert.Equal(t, "message", outEvt.AggregateType)
		assert.Equal(t, output.Message.ID, outEvt.AggregateID)

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(outEvt.Payload, &payload))
		assert.Equal(t, output.Message.ID, payload["message_id"])
		assert.Equal(t, output.Conversation.ID, payload["conversation_id"])
		assert.Equal(t, output.Contact.ID, payload["contact_id"])
		assert.Equal(t, "text", payload["content_type"])
		assert.Equal(t, "Hello, world!", payload["content"])
	})
}

// TestReceiveMessage_OutboxDecouplesFromNATSOutage verifies that the "message
// received" event is durably enqueued in the transactional outbox as part of the
// message write, so a NATS outage during intake never loses the event: the
// message commits, the outbox row is written, and the relay publishes it later.
// Intake no longer depends on broker availability.
func TestReceiveMessage_OutboxDecouplesFromNATSOutage(t *testing.T) {
	ctx := context.Background()
	f := newReceiveMessageFixture()
	channel := makeChannel("ch-1", "tenant-1")
	f.channelRepo.Channels[channel.ID] = channel

	// The producer is irrelevant to the critical event now — it is written to the
	// outbox in the same tx as the message, not published inline.
	f.producer.ReturnError = fmt.Errorf("nats unavailable")

	out, err := f.uc.Execute(ctx, makeInbound("ch-1", "tenant-1"))
	require.NoError(t, err, "intake must succeed even while NATS is down")
	require.NotNil(t, out)

	assert.Len(t, f.messageRepo.Messages, 1, "message persisted")
	require.Len(t, f.messageRepo.OutboxEvents, 1, "received event durably queued for the relay")
	assert.Equal(t, nats.EventMessageReceived, f.messageRepo.OutboxEvents[0].EventType)
	assert.Equal(t, "evt-message-received-"+out.Message.ID, f.messageRepo.OutboxEvents[0].IdempotencyKey)
}

// TestReceiveMessage_RedeliveryDoesNotDuplicateOutbox verifies exactly-once
// enqueue: a redelivery of the same inbound is the ON CONFLICT no-op, returns a
// conflict (so the consumer acks), and does NOT enqueue a second outbox event —
// the original delivery's event is already durably queued for the relay.
func TestReceiveMessage_RedeliveryDoesNotDuplicateOutbox(t *testing.T) {
	ctx := context.Background()
	f := newReceiveMessageFixture()
	channel := makeChannel("ch-1", "tenant-1")
	f.channelRepo.Channels[channel.ID] = channel
	inbound := makeInbound("ch-1", "tenant-1")

	// First delivery: message persisted, one outbox event queued.
	_, err := f.uc.Execute(ctx, inbound)
	require.NoError(t, err)
	require.Len(t, f.messageRepo.Messages, 1)
	require.Len(t, f.messageRepo.OutboxEvents, 1)

	// Redelivery of the same inbound → conflict, no double-persist, no second
	// outbox row.
	_, err = f.uc.Execute(ctx, inbound)
	require.Error(t, err, "duplicate delivery returns conflict so the consumer acks")

	assert.Len(t, f.messageRepo.Messages, 1, "no duplicate message row")
	assert.Len(t, f.messageRepo.OutboxEvents, 1, "no duplicate outbox event")
}
