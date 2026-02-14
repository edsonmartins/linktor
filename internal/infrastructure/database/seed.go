package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// Seed populates the database with initial data if empty
func (db *PostgresDB) Seed(ctx context.Context) error {
	// Check if users exist (more reliable than tenants for detecting complete seed)
	var userCount int64
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return err
	}

	if userCount > 0 {
		logger.Info("Database already seeded, skipping...")
		return nil
	}

	// Clean up any partial seed (tenant without users)
	db.Pool.Exec(ctx, "DELETE FROM messages")
	db.Pool.Exec(ctx, "DELETE FROM conversations")
	db.Pool.Exec(ctx, "DELETE FROM contact_identities")
	db.Pool.Exec(ctx, "DELETE FROM contacts")
	db.Pool.Exec(ctx, "DELETE FROM channels")
	db.Pool.Exec(ctx, "DELETE FROM users")
	db.Pool.Exec(ctx, "DELETE FROM tenants")

	logger.Info("Seeding database with initial data...")

	// Create default tenant
	tenantID := uuid.New().String()
	tenant := entity.NewTenant("Demo Company", "demo", entity.PlanProfessional)
	tenant.ID = tenantID

	tenantRepo := NewTenantRepository(db)
	if err := tenantRepo.Create(ctx, tenant); err != nil {
		return err
	}
	logger.Info("Created tenant: Demo Company")

	// Create admin user
	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminUser := entity.NewUser(tenantID, "admin@demo.com", string(hashedPassword), "Admin User", entity.UserRoleAdmin)
	adminUser.ID = uuid.New().String()

	userRepo := NewUserRepository(db)
	if err := userRepo.Create(ctx, adminUser); err != nil {
		return err
	}
	logger.Info("Created admin user: admin@demo.com / admin123")

	// Create agent user
	agentUser := entity.NewUser(tenantID, "agent@demo.com", string(hashedPassword), "Agent User", entity.UserRoleAgent)
	agentUser.ID = uuid.New().String()

	if err := userRepo.Create(ctx, agentUser); err != nil {
		return err
	}
	logger.Info("Created agent user: agent@demo.com / admin123")

	// Create WebChat channel
	channelID := uuid.New().String()
	now := time.Now()
	channel := &entity.Channel{
		ID:               channelID,
		TenantID:         tenantID,
		Name:             "Website Chat",
		Type:             entity.ChannelTypeWebChat,
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config: map[string]string{
			"welcome_message": "Olá! Como posso ajudar?",
			"primary_color":   "#007bff",
			"position":        "bottom-right",
		},
		Credentials: map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	channelRepo := NewChannelRepository(db)
	if err := channelRepo.Create(ctx, channel); err != nil {
		return err
	}
	logger.Info("Created WebChat channel: Website Chat")

	// Create sample contact
	contactID := uuid.New().String()
	contact := &entity.Contact{
		ID:           contactID,
		TenantID:     tenantID,
		Name:         "João Silva",
		Email:        "joao@example.com",
		Phone:        "+5511999999999",
		CustomFields: map[string]string{"company": "Acme Corp"},
		Tags:         []string{"vip", "enterprise"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	contactRepo := NewContactRepository(db)
	if err := contactRepo.Create(ctx, contact); err != nil {
		return err
	}
	logger.Info("Created sample contact: João Silva")

	// Create sample conversation
	conversationID := uuid.New().String()
	conversation := &entity.Conversation{
		ID:             conversationID,
		TenantID:       tenantID,
		ChannelID:      channelID,
		ContactID:      contactID,
		AssignedUserID: &agentUser.ID,
		Status:         entity.ConversationStatusOpen,
		Priority:       entity.ConversationPriorityNormal,
		Subject:        "Dúvida sobre produto",
		UnreadCount:    1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	conversationRepo := NewConversationRepository(db)
	if err := conversationRepo.Create(ctx, conversation); err != nil {
		return err
	}
	logger.Info("Created sample conversation")

	// Create sample messages
	messageRepo := NewMessageRepository(db)

	msg1 := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderType:     entity.SenderTypeContact,
		SenderID:       contactID,
		ContentType:    entity.ContentTypeText,
		Content:        "Olá, gostaria de saber mais sobre o produto X",
		Status:         entity.MessageStatusDelivered,
		Metadata:       map[string]string{},
		CreatedAt:      now.Add(-5 * time.Minute),
	}
	if err := messageRepo.Create(ctx, msg1); err != nil {
		return err
	}

	msg2 := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderType:     entity.SenderTypeUser,
		SenderID:       agentUser.ID,
		ContentType:    entity.ContentTypeText,
		Content:        "Olá João! Claro, posso ajudar. O produto X é nossa solução mais completa.",
		Status:         entity.MessageStatusDelivered,
		Metadata:       map[string]string{},
		CreatedAt:      now.Add(-4 * time.Minute),
	}
	if err := messageRepo.Create(ctx, msg2); err != nil {
		return err
	}

	msg3 := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderType:     entity.SenderTypeContact,
		SenderID:       contactID,
		ContentType:    entity.ContentTypeText,
		Content:        "Qual o preço?",
		Status:         entity.MessageStatusDelivered,
		Metadata:       map[string]string{},
		CreatedAt:      now.Add(-1 * time.Minute),
	}
	if err := messageRepo.Create(ctx, msg3); err != nil {
		return err
	}

	logger.Info("Created 3 sample messages")

	logger.Info("Database seeding completed!")
	logger.Info("=========================================")
	logger.Info("Login credentials:")
	logger.Info("  Admin: admin@demo.com / admin123")
	logger.Info("  Agent: agent@demo.com / admin123")
	logger.Info("=========================================")

	return nil
}
