package service

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
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
	// TODO: Add repositories and NATS publisher
}

// NewMessageService creates a new message service
func NewMessageService() *MessageService {
	return &MessageService{}
}

// ListByConversation returns all messages for a conversation
func (s *MessageService) ListByConversation(ctx context.Context, conversationID string, params *repository.ListParams) ([]*entity.Message, int64, error) {
	// TODO: Implement
	return []*entity.Message{}, 0, nil
}

// Send sends a new message
func (s *MessageService) Send(ctx context.Context, input *SendMessageInput) (*entity.Message, error) {
	// TODO: Implement
	return &entity.Message{}, nil
}

// GetByID returns a message by ID
func (s *MessageService) GetByID(ctx context.Context, id string) (*entity.Message, error) {
	// TODO: Implement
	return &entity.Message{}, nil
}

// UpdateStatus updates a message status
func (s *MessageService) UpdateStatus(ctx context.Context, id string, status entity.MessageStatus, errorMessage string) (*entity.Message, error) {
	// TODO: Implement
	return &entity.Message{}, nil
}

// SendReaction sends a reaction to a message
// If emoji is empty, the reaction is removed
func (s *MessageService) SendReaction(ctx context.Context, conversationID, messageID, emoji, senderID string) error {
	// TODO: Implement
	// 1. Get the original message to find external_id
	// 2. Get the conversation to find the channel
	// 3. Publish reaction to NATS for the adapter to send
	return nil
}
