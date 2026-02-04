package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/adapters/ai/anthropic"
	"github.com/msgfy/linktor/internal/adapters/ai/ollama"
	"github.com/msgfy/linktor/internal/adapters/ai/openai"
	"github.com/msgfy/linktor/internal/adapters/webchat"
	whatsapp "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/api/handlers"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/config"
	"github.com/msgfy/linktor/internal/infrastructure/database"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
	"github.com/msgfy/linktor/pkg/plugin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Log.Level, cfg.Log.Format); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Starting Linktor server...")

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	logger.Info("Connecting to PostgreSQL...")
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Run migrations
	logger.Info("Running database migrations...")
	if err := db.RunMigrations(context.Background()); err != nil {
		logger.Fatal("Failed to run migrations")
	}

	// Seed database with initial data
	if err := db.Seed(context.Background()); err != nil {
		logger.Error("Failed to seed database: " + err.Error())
	}

	// Initialize NATS (optional - will work without it but messaging features disabled)
	logger.Info("Connecting to NATS JetStream...")
	natsClient, err := nats.NewClient(&cfg.NATS)
	var producer *nats.Producer
	var consumer *nats.Consumer
	if err != nil {
		logger.Warn("Failed to connect to NATS - messaging features disabled")
	} else {
		defer natsClient.Close()
		producer = nats.NewProducer(natsClient)
		consumer = nats.NewConsumer(natsClient)
	}

	// Initialize repositories
	logger.Info("Initializing repositories...")
	tenantRepo := database.NewTenantRepository(db)
	userRepo := database.NewUserRepository(db)
	messageRepo := database.NewMessageRepository(db)
	conversationRepo := database.NewConversationRepository(db)
	contactRepo := database.NewContactRepository(db)
	channelRepo := database.NewChannelRepository(db)
	botRepo := database.NewBotRepository(db)
	contextRepo := database.NewConversationContextRepository(db)
	aiResponseRepo := database.NewAIResponseRepository(db)
	kbRepo := database.NewKnowledgeBaseRepository(db)
	kiRepo := database.NewKnowledgeItemRepository(db)
	flowRepo := database.NewFlowRepository(db)
	analyticsRepo := database.NewAnalyticsRepository(db)

	// Initialize services
	logger.Info("Initializing services...")
	normalizer := service.NewMessageNormalizer()
	authService := service.NewAuthService(userRepo, &cfg.JWT)
	userService := service.NewUserService(userRepo, tenantRepo)

	// Initialize AI services
	logger.Info("Initializing AI services...")
	aiFactory := service.GetAIProviderFactory()

	// Register OpenAI provider if configured
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey != "" {
		openAIProvider := openai.NewProvider(&openai.ProviderConfig{
			APIKey:       openAIKey,
			OrgID:        os.Getenv("OPENAI_ORG_ID"),
			DefaultModel: os.Getenv("OPENAI_DEFAULT_MODEL"),
		})
		aiFactory.Register(openAIProvider)
		logger.Info("OpenAI provider registered")
	} else {
		logger.Warn("OpenAI API key not configured - AI features limited")
	}

	// Register Anthropic provider if configured
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey != "" {
		anthropicProvider := anthropic.NewProvider(&anthropic.ProviderConfig{
			APIKey:       anthropicKey,
			DefaultModel: os.Getenv("ANTHROPIC_DEFAULT_MODEL"),
		})
		aiFactory.Register(anthropicProvider)
		logger.Info("Anthropic provider registered")
	}

	// Register Ollama provider (always available locally)
	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	ollamaProvider := ollama.NewProvider(&ollama.ProviderConfig{
		BaseURL:      ollamaURL,
		DefaultModel: os.Getenv("OLLAMA_DEFAULT_MODEL"),
	})
	aiFactory.Register(ollamaProvider)
	if ollamaProvider.IsAvailable() {
		logger.Info("Ollama provider registered and available")
	} else {
		logger.Info("Ollama provider registered but not available")
	}

	// Initialize conversation context service
	contextService := service.NewConversationContextService(contextRepo, nil)

	// Initialize intent service
	intentService := service.NewIntentService(aiFactory, nil)

	// Initialize flow services
	flowEngine := service.NewFlowEngineService(flowRepo, contextRepo)
	flowService := service.NewFlowService(flowRepo)

	// Initialize analytics service
	analyticsService := service.NewAnalyticsService(analyticsRepo)

	// Initialize use cases
	sendMessageUC := usecase.NewSendMessageUseCase(
		messageRepo,
		conversationRepo,
		channelRepo,
		contactRepo,
		producer,
	)
	receiveMessageUC := usecase.NewReceiveMessageUseCase(
		messageRepo,
		conversationRepo,
		channelRepo,
		contactRepo,
		producer,
		normalizer,
	)

	// Initialize embedding service
	embeddingService := service.NewEmbeddingService(aiFactory, nil)

	// Initialize vector store (pgvector)
	vectorStore := database.NewPgVectorStore(db)

	// Initialize knowledge service
	knowledgeService := service.NewKnowledgeService(kbRepo, kiRepo, embeddingService, vectorStore)

	// Initialize AI use cases
	analyzeMessageUC := usecase.NewAnalyzeMessageUseCase(
		botRepo,
		contextService,
		intentService,
		producer,
	)
	generateAIResponseUC := usecase.NewGenerateAIResponseUseCase(
		aiFactory,
		botRepo,
		aiResponseRepo,
		contextService,
		knowledgeService,
		producer,
	)

	// Initialize bot service
	botService := service.NewBotService(
		botRepo,
		channelRepo,
		contextRepo,
		contextService,
		aiFactory,
		flowEngine,
	)

	// Initialize escalation use case
	escalateConversationUC := usecase.NewEscalateConversationUseCase(
		conversationRepo,
		messageRepo,
		contactRepo,
		channelRepo,
		botRepo,
		userRepo,
		contextRepo,
		aiFactory,
		producer,
	)

	// Initialize WebChat adapter
	logger.Info("Initializing WebChat adapter...")
	webchatAdapter := webchat.NewAdapter()
	if err := webchatAdapter.Connect(context.Background()); err != nil {
		logger.Fatal("Failed to start WebChat adapter")
	}

	// Register adapter in global registry
	plugin.Register(plugin.ChannelTypeWebChat, webchatAdapter)

	// Initialize WhatsApp Official adapter
	logger.Info("Initializing WhatsApp Official adapter...")
	whatsappAdapter := whatsapp.NewAdapter()
	// Note: WhatsApp adapter requires per-channel initialization with credentials
	// Registration makes it available as a channel type
	plugin.Register(plugin.ChannelTypeWhatsAppOfficial, whatsappAdapter)

	// Create WebChat handler
	webchatHandler := webchat.NewHandler(
		webchatAdapter,
		channelRepo,
		conversationRepo,
		contactRepo,
		producer,
	)

	// Create webhook handler
	webhookHandler := handlers.NewWebhookHandler(channelRepo, producer)

	// Create bot handler
	botHandler := handlers.NewBotHandler(botService)

	// Create AI handler
	aiHandler := handlers.NewAIHandler(
		aiFactory,
		intentService,
		generateAIResponseUC,
		analyzeMessageUC,
		escalateConversationUC,
	)

	// Create knowledge handler
	knowledgeHandler := handlers.NewKnowledgeHandler(knowledgeService)

	// Create conversation service and handler
	conversationService := service.NewConversationService()
	conversationHandler := handlers.NewConversationHandler(conversationService, escalateConversationUC)

	// Create flow handler
	flowHandler := handlers.NewFlowHandler(flowService)

	// Create analytics handler
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Create auth handler
	authHandler := handlers.NewAuthHandler(authService, userService)

	// Initialize Agent WebSocket Hub
	logger.Info("Starting Agent WebSocket Hub...")
	agentHub := handlers.GetAgentHub()
	wsHandler := handlers.NewWebSocketHandler(agentHub, cfg.JWT.Secret)

	// Start message consumers (only if NATS is available)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var aiConsumer *nats.AIConsumer

	if consumer != nil {
		logger.Info("Starting message consumers...")
		// Subscribe to inbound messages
		if err := consumer.SubscribeAllInbound(ctx, func(ctx context.Context, msg *nats.InboundMessage) error {
			_, err := receiveMessageUC.Execute(ctx, msg)
			return err
		}); err != nil {
			logger.Warn("Failed to subscribe to inbound messages")
		}

		// Subscribe to status updates
		if err := consumer.SubscribeStatus(ctx, func(ctx context.Context, status *nats.StatusUpdate) error {
			return messageRepo.UpdateStatus(ctx, status.MessageID, toMessageStatus(status.Status), status.ErrorMessage)
		}); err != nil {
			logger.Warn("Failed to subscribe to status updates")
		}

		// Initialize AI consumer
		logger.Info("Starting AI consumers...")
		aiConsumer = nats.NewAIConsumer(natsClient)
		if err := aiConsumer.EnsureStream(ctx); err != nil {
			logger.Warn("Failed to create AI stream: " + err.Error())
		} else {
			// Subscribe to bot analysis requests
			if err := aiConsumer.SubscribeBotAnalysis(ctx, func(ctx context.Context, req *nats.BotAnalysisRequest) error {
				_, err := analyzeMessageUC.Execute(ctx, &usecase.AnalyzeMessageInput{
					MessageID:      req.MessageID,
					ConversationID: req.ConversationID,
					TenantID:       req.TenantID,
					Content:        req.Content,
					ChannelID:      req.ChannelID,
				})
				return err
			}); err != nil {
				logger.Warn("Failed to subscribe to bot analysis: " + err.Error())
			}

			// Subscribe to bot response requests
			if err := aiConsumer.SubscribeBotResponse(ctx, func(ctx context.Context, req *nats.BotResponseRequest) error {
				result, err := generateAIResponseUC.Execute(ctx, &usecase.GenerateAIResponseInput{
					MessageID:      req.MessageID,
					ConversationID: req.ConversationID,
					TenantID:       req.TenantID,
					ChannelID:      req.ChannelID,
					Content:        req.Content,
				})
				if err != nil {
					return err
				}

				// If response was generated, send it via the send message use case
				if result != nil && result.Response != "" && !result.ShouldEscalate {
					_, err = sendMessageUC.Execute(ctx, &usecase.SendMessageInput{
						TenantID:       req.TenantID,
						ConversationID: req.ConversationID,
						SenderID:       req.BotID,
						SenderType:     entity.SenderTypeBot,
						ContentType:    entity.ContentTypeText,
						Content:        result.Response,
						Metadata: map[string]string{
							"ai_model":      result.Model,
							"ai_confidence": fmt.Sprintf("%.2f", result.Confidence),
						},
					})
				}
				return err
			}); err != nil {
				logger.Warn("Failed to subscribe to bot response: " + err.Error())
			}

			logger.Info("AI consumers started")
		}
	}

	// Initialize Gin router
	logger.Info("Initializing HTTP router...")
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	router.GET("/ready", func(c *gin.Context) {
		// Check database
		if err := db.Ping(context.Background()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "database unavailable"})
			return
		}
		// Check NATS (optional)
		natsStatus := "disabled"
		if natsClient != nil {
			if natsClient.IsConnected() {
				natsStatus = "connected"
			} else {
				natsStatus = "disconnected"
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "nats": natsStatus})
	})

	// WebSocket endpoint for WebChat
	router.GET("/ws/:channelId", webchatHandler.WebSocketHandler)

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// WebChat widget config (no auth required)
		api.GET("/webchat/:channelId/config", webchatHandler.GetWidgetConfig)

		// Webhook routes (auth via signature verification)
		webhooks := api.Group("/webhooks")
		{
			webhooks.Any("/whatsapp/:channelId", webhookHandler.WhatsAppWebhook)
			webhooks.POST("/telegram/:channelId", webhookHandler.TelegramWebhook)
			webhooks.POST("/generic/:channelId", webhookHandler.GenericWebhook)
			webhooks.POST("/status/:channelId", webhookHandler.StatusCallback)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(authMiddleware.Authenticate())
		{
			// User info
			protected.GET("/me", authHandler.Me)

			// Conversations
			conversations := protected.Group("/conversations")
			{
				conversations.GET("", createListConversationsHandler(conversationRepo))
				conversations.GET("/:id", createGetConversationHandler(conversationRepo))
				// Messages within a conversation
				conversations.GET("/:id/messages", createListMessagesHandler(messageRepo))
				conversations.POST("/:id/messages", createSendMessageHandler(sendMessageUC))
			}

			// Contacts
			contacts := protected.Group("/contacts")
			{
				contacts.GET("", createListContactsHandler(contactRepo))
				contacts.GET("/:id", createGetContactHandler(contactRepo))
			}

			// Channels
			channels := protected.Group("/channels")
			{
				channels.GET("", createListChannelsHandler(channelRepo))
				channels.GET("/:id", createGetChannelHandler(channelRepo))
			}

			// Bots
			bots := protected.Group("/bots")
			{
				bots.GET("", botHandler.List)
				bots.POST("", botHandler.Create)
				bots.GET("/:id", botHandler.Get)
				bots.PUT("/:id", botHandler.Update)
				bots.DELETE("/:id", botHandler.Delete)
				bots.POST("/:id/activate", botHandler.Activate)
				bots.POST("/:id/deactivate", botHandler.Deactivate)
				bots.POST("/:id/channels", botHandler.AssignChannel)
				bots.DELETE("/:id/channels/:channelId", botHandler.UnassignChannel)
				bots.PUT("/:id/config", botHandler.UpdateConfig)
				bots.POST("/:id/escalation-rules", botHandler.AddEscalationRule)
				bots.POST("/:id/test", botHandler.Test)
			}

			// AI
			ai := protected.Group("/ai")
			{
				ai.GET("/providers", aiHandler.ListProviders)
				ai.GET("/providers/:provider/models", aiHandler.GetModels)
				ai.POST("/complete", aiHandler.Complete)
				ai.POST("/classify-intent", aiHandler.ClassifyIntent)
				ai.POST("/analyze-sentiment", aiHandler.AnalyzeSentiment)
				ai.POST("/generate-response", aiHandler.GenerateResponse)
				ai.POST("/analyze-message", aiHandler.AnalyzeMessage)
				ai.POST("/escalate", aiHandler.Escalate)
			}

			// Knowledge Bases
			knowledge := protected.Group("/knowledge-bases")
			{
				knowledge.GET("", knowledgeHandler.ListKnowledgeBases)
				knowledge.POST("", knowledgeHandler.CreateKnowledgeBase)
				knowledge.GET("/:id", knowledgeHandler.GetKnowledgeBase)
				knowledge.PUT("/:id", knowledgeHandler.UpdateKnowledgeBase)
				knowledge.DELETE("/:id", knowledgeHandler.DeleteKnowledgeBase)
				knowledge.GET("/:id/items", knowledgeHandler.ListItems)
				knowledge.POST("/:id/items", knowledgeHandler.AddItem)
				knowledge.POST("/:id/items/bulk", knowledgeHandler.BulkAddItems)
				knowledge.GET("/:id/items/:itemId", knowledgeHandler.GetItem)
				knowledge.PUT("/:id/items/:itemId", knowledgeHandler.UpdateItem)
				knowledge.DELETE("/:id/items/:itemId", knowledgeHandler.DeleteItem)
				knowledge.POST("/:id/search", knowledgeHandler.Search)
				knowledge.POST("/:id/regenerate-embeddings", knowledgeHandler.RegenerateEmbeddings)
			}

			// Flows (Conversational Decision Trees)
			flows := protected.Group("/flows")
			{
				flows.GET("", flowHandler.List)
				flows.POST("", flowHandler.Create)
				flows.GET("/:id", flowHandler.Get)
				flows.PUT("/:id", flowHandler.Update)
				flows.DELETE("/:id", flowHandler.Delete)
				flows.POST("/:id/activate", flowHandler.Activate)
				flows.POST("/:id/deactivate", flowHandler.Deactivate)
				flows.POST("/:id/test", flowHandler.Test)
			}

			// Analytics
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/overview", analyticsHandler.GetOverview)
				analytics.GET("/conversations", analyticsHandler.GetConversations)
				analytics.GET("/flows", analyticsHandler.GetFlows)
				analytics.GET("/escalations", analyticsHandler.GetEscalations)
				analytics.GET("/channels", analyticsHandler.GetChannels)
			}

			// Conversation Management (with escalation support)
			convMgmt := protected.Group("/conversations-v2")
			{
				convMgmt.GET("", conversationHandler.List)
				convMgmt.POST("", conversationHandler.Create)
				convMgmt.GET("/:id", conversationHandler.Get)
				convMgmt.PUT("/:id", conversationHandler.Update)
				convMgmt.POST("/:id/assign", conversationHandler.Assign)
				convMgmt.POST("/:id/resolve", conversationHandler.Resolve)
				convMgmt.POST("/:id/reopen", conversationHandler.Reopen)
				convMgmt.POST("/:id/escalate", conversationHandler.Escalate)
				convMgmt.GET("/:id/escalation-context", conversationHandler.GetEscalationContext)
			}
		}

		// Agent WebSocket (JWT via query param)
		protected.GET("/ws", wsHandler.HandleConnection)
	}

	// Serve static widget files
	router.Static("/widget", "./web/embed")

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Cancel context to stop consumers
	cancel()

	// Stop AI consumers
	if aiConsumer != nil {
		aiConsumer.Stop()
	}

	// Disconnect adapters
	webchatAdapter.Disconnect(context.Background())
	whatsappAdapter.Disconnect(context.Background())

	// Create context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited properly")
}

// Helper function to convert status string to entity.MessageStatus
func toMessageStatus(status string) entity.MessageStatus {
	switch status {
	case "sent":
		return entity.MessageStatusSent
	case "delivered":
		return entity.MessageStatusDelivered
	case "read":
		return entity.MessageStatusRead
	case "failed":
		return entity.MessageStatusFailed
	default:
		return entity.MessageStatusPending
	}
}

// Simplified handlers (in production, these would be in separate handler files)

func createListMessagesHandler(repo *database.MessageRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		conversationID := c.Param("id")
		params := &database.ListParams{
			Page:     1,
			PageSize: 50,
			SortBy:   "created_at",
			SortDir:  "desc",
		}

		messages, total, err := repo.FindByConversation(c.Request.Context(), conversationID, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  messages,
			"total": total,
		})
	}
}

func createSendMessageHandler(uc *usecase.SendMessageUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		conversationID := c.Param("id")
		tenantID := c.GetString(middleware.TenantIDKey)
		userID := c.GetString(middleware.UserIDKey)

		var input struct {
			ContentType string            `json:"content_type"`
			Content     string            `json:"content"`
			Metadata    map[string]string `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		result, err := uc.Execute(c.Request.Context(), &usecase.SendMessageInput{
			TenantID:       tenantID,
			ConversationID: conversationID,
			SenderID:       userID,
			SenderType:     entity.SenderTypeUser,
			ContentType:    entity.ContentType(input.ContentType),
			Content:        input.Content,
			Metadata:       input.Metadata,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Broadcast new message to WebSocket clients
		handlers.BroadcastNewMessage(tenantID, conversationID, result.Message)

		c.JSON(http.StatusOK, gin.H{"data": result.Message})
	}
}

func createListConversationsHandler(repo *database.ConversationRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString(middleware.TenantIDKey)
		params := &database.ListParams{
			Page:     1,
			PageSize: 20,
			SortBy:   "updated_at",
			SortDir:  "desc",
		}

		conversations, total, err := repo.FindByTenant(c.Request.Context(), tenantID, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  conversations,
			"total": total,
		})
	}
}

func createGetConversationHandler(repo *database.ConversationRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		conversation, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": conversation})
	}
}

func createListContactsHandler(repo *database.ContactRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString(middleware.TenantIDKey)
		params := &database.ListParams{
			Page:     1,
			PageSize: 20,
			SortBy:   "created_at",
			SortDir:  "desc",
		}

		contacts, total, err := repo.FindByTenant(c.Request.Context(), tenantID, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  contacts,
			"total": total,
		})
	}
}

func createGetContactHandler(repo *database.ContactRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		contact, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": contact})
	}
}

func createListChannelsHandler(repo *database.ChannelRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString(middleware.TenantIDKey)
		params := &database.ListParams{
			Page:     1,
			PageSize: 20,
			SortBy:   "created_at",
			SortDir:  "desc",
		}

		channels, total, err := repo.FindByTenant(c.Request.Context(), tenantID, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  channels,
			"total": total,
		})
	}
}

func createGetChannelHandler(repo *database.ChannelRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		channel, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": channel})
	}
}

// ListParams alias for database package
type ListParams = database.ListParams
