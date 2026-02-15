package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
	"go.uber.org/zap"
)

// HistoryImportService handles chat history import operations for WhatsApp Coexistence
type HistoryImportService struct {
	channelRepo      repository.ChannelRepository
	conversationRepo repository.ConversationRepository
	messageRepo      repository.MessageRepository
	contactRepo      repository.ContactRepository
	importRepo       repository.HistoryImportRepository
	waClient         *whatsapp_official.Client
	// Track running imports for cancellation
	runningImports map[string]context.CancelFunc
}

// NewHistoryImportService creates a new history import service
func NewHistoryImportService(
	channelRepo repository.ChannelRepository,
	conversationRepo repository.ConversationRepository,
	messageRepo repository.MessageRepository,
	contactRepo repository.ContactRepository,
	importRepo repository.HistoryImportRepository,
) *HistoryImportService {
	return &HistoryImportService{
		channelRepo:      channelRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		contactRepo:      contactRepo,
		importRepo:       importRepo,
		runningImports:   make(map[string]context.CancelFunc),
	}
}

// StartImportInput represents input for starting a history import
type StartImportInput struct {
	ChannelID   string
	TenantID    string
	ImportSince *time.Time // How far back to import (max 6 months)
}

// ImportProgress represents the progress of an import
type ImportProgress struct {
	ID                    string `json:"id"`
	ChannelID             string `json:"channel_id"`
	Status                string `json:"status"`
	TotalConversations    int    `json:"total_conversations"`
	ImportedConversations int    `json:"imported_conversations"`
	TotalMessages         int    `json:"total_messages"`
	ImportedMessages      int    `json:"imported_messages"`
	TotalContacts         int    `json:"total_contacts"`
	ImportedContacts      int    `json:"imported_contacts"`
	Progress              int    `json:"progress"` // 0-100
	ErrorMessage          string `json:"error_message,omitempty"`
	StartedAt             string `json:"started_at,omitempty"`
	CompletedAt           string `json:"completed_at,omitempty"`
}

// StartImport initiates a chat history import job
func (s *HistoryImportService) StartImport(ctx context.Context, input *StartImportInput) (*entity.HistoryImport, error) {
	// Validate channel exists and is coexistence enabled
	channel, err := s.channelRepo.FindByID(ctx, input.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "channel not found")
	}

	if !channel.IsCoexistenceChannel() {
		return nil, errors.New(errors.ErrCodeValidation, "channel is not coexistence enabled")
	}

	// Create new import job
	importJob := entity.NewHistoryImport(input.ChannelID, input.TenantID)
	importJob.ID = uuid.New().String()

	// Set custom import date if provided
	if input.ImportSince != nil {
		// Ensure it's within 6 months
		sixMonthsAgo := time.Now().AddDate(0, -6, 0)
		if input.ImportSince.Before(sixMonthsAgo) {
			importJob.ImportSince = &sixMonthsAgo
		} else {
			importJob.ImportSince = input.ImportSince
		}
	}

	// Save import job to repository
	if err := s.importRepo.Create(ctx, importJob); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to save import job")
	}

	logger.Info("Starting history import",
		zap.String("import_id", importJob.ID),
		zap.String("channel_id", input.ChannelID),
		zap.Time("import_since", *importJob.ImportSince),
	)

	// Create cancellable context for the import
	importCtx, cancel := context.WithCancel(context.Background())
	s.runningImports[importJob.ID] = cancel

	// Start import in background with cancellable context
	go func() {
		defer func() {
			delete(s.runningImports, importJob.ID)
		}()
		s.runImport(importCtx, importJob, channel)
	}()

	return importJob, nil
}

// runImport executes the import job in the background
func (s *HistoryImportService) runImport(ctx context.Context, importJob *entity.HistoryImport, channel *entity.Channel) {
	importJob.Start()
	s.importRepo.Update(context.Background(), importJob)

	defer func() {
		if r := recover(); r != nil {
			logger.Error("History import panicked",
				zap.String("import_id", importJob.ID),
				zap.Any("panic", r),
			)
			importJob.Fail(fmt.Sprintf("panic: %v", r), nil)
			s.importRepo.Update(context.Background(), importJob)
		}
	}()

	// Check for cancellation at start
	select {
	case <-ctx.Done():
		importJob.Cancel()
		s.importRepo.Update(context.Background(), importJob)
		logger.Info("Import cancelled before start", zap.String("import_id", importJob.ID))
		return
	default:
	}

	// Initialize WhatsApp client for this channel
	accessToken, ok := channel.Credentials["access_token"]
	if !ok || accessToken == "" {
		importJob.Fail("missing access token", nil)
		return
	}

	phoneNumberID, ok := channel.Config["phone_number_id"]
	if !ok || phoneNumberID == "" {
		importJob.Fail("missing phone number ID", nil)
		return
	}

	// Create WhatsApp client with default API version
	apiVersion := channel.Config["api_version"]
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	waConfig := &whatsapp_official.Config{
		AccessToken:   accessToken,
		PhoneNumberID: phoneNumberID,
		APIVersion:    apiVersion,
	}
	s.waClient = whatsapp_official.NewClient(waConfig)

	// Fetch conversations from WhatsApp Cloud API
	err := s.importConversations(ctx, importJob, channel)
	if err != nil {
		logger.Error("History import failed",
			zap.String("import_id", importJob.ID),
			zap.Error(err),
		)
		importJob.Fail(err.Error(), nil)
		s.importRepo.Update(context.Background(), importJob)
		return
	}

	importJob.Complete()
	s.importRepo.Update(context.Background(), importJob)
	logger.Info("History import completed",
		zap.String("import_id", importJob.ID),
		zap.Int("conversations", importJob.ImportedConversations),
		zap.Int("messages", importJob.ImportedMessages),
		zap.Int("contacts", importJob.ImportedContacts),
	)
}

// importConversations imports conversations from WhatsApp Cloud API
func (s *HistoryImportService) importConversations(ctx context.Context, importJob *entity.HistoryImport, channel *entity.Channel) error {
	// Check for cancellation
	select {
	case <-ctx.Done():
		importJob.Cancel()
		return ctx.Err()
	default:
	}

	// Note: WhatsApp Cloud API does not provide a direct endpoint to fetch chat history.
	// This is a placeholder implementation. In practice, you would need to:
	// 1. Use the WhatsApp Business API to fetch conversations
	// 2. Process each conversation and its messages
	// 3. Create/update contacts, conversations, and messages in the database

	// For now, we simulate the import process
	logger.Info("Importing conversations from WhatsApp",
		zap.String("import_id", importJob.ID),
		zap.String("channel_id", channel.ID),
	)

	// TODO: Implement actual WhatsApp conversation fetching
	// The Cloud API typically provides:
	// - GET /{phone-number-id}/conversations - List conversations
	// - GET /{phone-number-id}/messages - List messages

	// Set totals (placeholder)
	importJob.SetTotals(0, 0, 0)

	// Import would happen here with periodic context checks...

	return nil
}

// GetImportProgress returns the progress of an import job
func (s *HistoryImportService) GetImportProgress(ctx context.Context, importID string) (*ImportProgress, error) {
	importJob, err := s.importRepo.FindByID(ctx, importID)
	if err != nil {
		return nil, err
	}

	progress := &ImportProgress{
		ID:                    importJob.ID,
		ChannelID:             importJob.ChannelID,
		Status:                string(importJob.Status),
		TotalConversations:    importJob.TotalConversations,
		ImportedConversations: importJob.ImportedConversations,
		TotalMessages:         importJob.TotalMessages,
		ImportedMessages:      importJob.ImportedMessages,
		TotalContacts:         importJob.TotalContacts,
		ImportedContacts:      importJob.ImportedContacts,
		Progress:              int(importJob.Progress()),
		ErrorMessage:          importJob.ErrorMessage,
	}

	if importJob.StartedAt != nil {
		progress.StartedAt = importJob.StartedAt.Format(time.RFC3339)
	}
	if importJob.CompletedAt != nil {
		progress.CompletedAt = importJob.CompletedAt.Format(time.RFC3339)
	}

	return progress, nil
}

// ListImports returns all imports for a channel
func (s *HistoryImportService) ListImports(ctx context.Context, channelID string) ([]*entity.HistoryImport, error) {
	return s.importRepo.FindByChannelID(ctx, channelID)
}

// CancelImport cancels a running import job
func (s *HistoryImportService) CancelImport(ctx context.Context, importID string) error {
	cancel, ok := s.runningImports[importID]
	if !ok {
		return errors.New(errors.ErrCodeNotFound, "import not found or not running")
	}
	cancel()
	delete(s.runningImports, importID)
	logger.Info("Import cancelled", zap.String("import_id", importID))
	return nil
}

// importContact creates or updates a contact from WhatsApp data
func (s *HistoryImportService) importContact(ctx context.Context, tenantID, channelID, phone, name string) (*entity.Contact, error) {
	// Check if contact exists
	existing, err := s.contactRepo.FindByPhone(ctx, phone, tenantID)
	if err == nil && existing != nil {
		// Update name if needed
		if name != "" && existing.Name != name {
			existing.Name = name
			existing.UpdatedAt = time.Now()
			err = s.contactRepo.Update(ctx, existing)
			if err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	// Create new contact
	contact := &entity.Contact{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Name:      name,
		Phone:     phone,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.contactRepo.Create(ctx, contact)
	if err != nil {
		return nil, err
	}

	// Create ContactIdentity for this channel
	identity := &entity.ContactIdentity{
		ID:          uuid.New().String(),
		ContactID:   contact.ID,
		ChannelType: string(entity.ChannelTypeWhatsAppOfficial),
		Identifier:  phone,
		Metadata: map[string]string{
			"channel_id": channelID,
		},
		CreatedAt: time.Now(),
	}

	// Add identity to contact (ignore error - identity is optional)
	s.contactRepo.AddIdentity(ctx, identity)

	return contact, nil
}

// importMessage imports a single message from WhatsApp
func (s *HistoryImportService) importMessage(
	ctx context.Context,
	conversationID string,
	senderType entity.SenderType,
	senderID string,
	contentType entity.ContentType,
	content string,
	originalTimestamp time.Time,
) (*entity.Message, error) {
	msg := entity.NewImportedMessage(
		conversationID,
		senderType,
		senderID,
		contentType,
		content,
		originalTimestamp,
	)
	msg.ID = uuid.New().String()

	err := s.messageRepo.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
