// @title           Linktor API
// @version         1.0
// @description     Linktor is an omnichannel conversation platform API that enables businesses to manage customer communications across multiple channels including WhatsApp, Telegram, SMS, Email, and more.
// @termsOfService  https://linktor.io/terms

// @contact.name   Linktor Support
// @contact.url    https://linktor.io/support
// @contact.email  support@linktor.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      api.linktor.io
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name auth
// @tag.description Authentication endpoints for login and token management

// @tag.name users
// @tag.description User management endpoints

// @tag.name channels
// @tag.description Channel configuration and management

// @tag.name contacts
// @tag.description Contact management and identity linking

// @tag.name conversations
// @tag.description Conversation handling and lifecycle management

// @tag.name messages
// @tag.description Message sending and retrieval

// @tag.name bots
// @tag.description AI bot configuration and management

// @tag.name knowledge
// @tag.description Knowledge base management for AI bots

// @tag.name flows
// @tag.description Conversation flow automation

// @tag.name analytics
// @tag.description Analytics and reporting

// @tag.name observability
// @tag.description System monitoring, logs, and queue management

// @tag.name health
// @tag.description Health check endpoints

// @tag.name webhooks
// @tag.description Webhook endpoints for external integrations

// @tag.name VRE
// @tag.description Visual Response Engine - render HTML templates to images for channels that don't support interactive elements

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
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/msgfy/linktor/docs" // Swagger docs

	"github.com/msgfy/linktor/internal/adapters/ai/anthropic"
	"github.com/msgfy/linktor/internal/adapters/ai/ollama"
	"github.com/msgfy/linktor/internal/adapters/ai/openai"
	"github.com/msgfy/linktor/internal/adapters/email"
	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/rcs"
	"github.com/msgfy/linktor/internal/adapters/sms"
	"github.com/msgfy/linktor/internal/adapters/telegram"
	"github.com/msgfy/linktor/internal/adapters/webchat"
	"github.com/msgfy/linktor/internal/adapters/whatsapp"
	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/api/handlers"
	"github.com/msgfy/linktor/internal/whatsapp/analytics"
	"github.com/msgfy/linktor/internal/whatsapp/calling"
	"github.com/msgfy/linktor/internal/whatsapp/ctwa"
	"github.com/msgfy/linktor/internal/whatsapp/payments"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/config"
	"github.com/msgfy/linktor/internal/infrastructure/database"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
	"github.com/msgfy/linktor/pkg/plugin"

	"github.com/go-redis/redis/v8"
)

func main() {
	// Load .env file (optional - won't fail if not found)
	_ = godotenv.Load()

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
		logger.Fatal("Failed to run migrations: " + err.Error())
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
		logger.Warn(fmt.Sprintf("Failed to connect to NATS (%s): %v - messaging features disabled", cfg.NATS.URL, err))
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
	templateRepo := database.NewTemplateRepository(db)
	historyImportRepo := database.NewHistoryImportRepository(db)

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

	// Initialize template service
	templateService := service.NewTemplateService(templateRepo, channelRepo)

	// Initialize coexistence monitor service
	coexistenceMonitor := service.NewCoexistenceMonitorService(channelRepo)

	// Initialize history import service for WhatsApp Coexistence
	_ = service.NewHistoryImportService(channelRepo, conversationRepo, messageRepo, contactRepo, historyImportRepo)

	// Initialize VRE (Visual Response Engine) service
	logger.Info("Initializing VRE service...")
	var vreService *service.VREService
	var redisClient *redis.Client

	// Connect to Redis if configured (for VRE caching)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			redisClient = redis.NewClient(opt)
			if err := redisClient.Ping(context.Background()).Err(); err != nil {
				logger.Warn("Failed to connect to Redis - VRE caching disabled: " + err.Error())
				redisClient = nil
			} else {
				logger.Info("Redis connected for VRE caching")
			}
		}
	}

	// Create VRE service
	vreConfig := &service.VREServiceConfig{
		TemplatesPath:  "./templates",
		CacheTTL:       5 * time.Minute,
		ChromePoolSize: 3,
		DefaultWidth:   800,
		DefaultQuality: 85,
	}
	if templatesPath := os.Getenv("VRE_TEMPLATES_PATH"); templatesPath != "" {
		vreConfig.TemplatesPath = templatesPath
	}

	vreService, err = service.NewVREService(vreConfig, redisClient)
	if err != nil {
		logger.Warn("Failed to initialize VRE service - visual rendering disabled: " + err.Error())
	} else {
		logger.Info("VRE service initialized")
		// Cleanup VRE on shutdown
		defer vreService.Close()
	}

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
	whatsappOfficialAdapter := whatsappofficial.NewAdapter()
	// Note: WhatsApp adapter requires per-channel initialization with credentials
	// Registration makes it available as a channel type
	plugin.Register(plugin.ChannelTypeWhatsAppOfficial, whatsappOfficialAdapter)

	// Initialize WhatsApp Unofficial adapter (whatsmeow)
	logger.Info("Initializing WhatsApp Unofficial adapter...")
	whatsappAdapter := whatsapp.NewAdapter()
	// Note: WhatsApp unofficial requires QR code authentication
	plugin.Register(plugin.ChannelTypeWhatsApp, whatsappAdapter)

	// Initialize RCS adapter
	logger.Info("Initializing RCS adapter...")
	rcsAdapter := rcs.NewAdapter()
	// Note: RCS adapter requires provider credentials (Zenvia, Infobip, etc)
	plugin.Register(plugin.ChannelTypeRCS, rcsAdapter)

	// Initialize Telegram adapter
	logger.Info("Initializing Telegram adapter...")
	telegramAdapter := telegram.NewAdapter()
	// Note: Telegram adapter requires per-channel initialization with bot token
	plugin.Register(plugin.ChannelTypeTelegram, telegramAdapter)

	// Initialize SMS/Twilio adapter
	logger.Info("Initializing SMS/Twilio adapter...")
	smsAdapter := sms.NewAdapter()
	// Note: SMS adapter requires per-channel initialization with Twilio credentials
	plugin.Register(plugin.ChannelTypeSMS, smsAdapter)

	// Initialize Facebook Messenger adapter
	logger.Info("Initializing Facebook Messenger adapter...")
	facebookAdapter := facebook.NewAdapter()
	// Note: Facebook adapter requires per-channel initialization with OAuth credentials
	plugin.Register(plugin.ChannelTypeFacebook, facebookAdapter)

	// Initialize Instagram DM adapter
	logger.Info("Initializing Instagram DM adapter...")
	instagramAdapter := instagram.NewAdapter()
	// Note: Instagram adapter requires per-channel initialization with OAuth credentials
	plugin.Register(plugin.ChannelTypeInstagram, instagramAdapter)

	// Initialize Email adapter
	logger.Info("Initializing Email adapter...")
	emailAdapter := email.NewAdapter()
	// Note: Email adapter supports multiple providers (SMTP, SendGrid, Mailgun, SES, Postmark)
	plugin.Register(plugin.ChannelTypeEmail, emailAdapter)

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

	// Create channel service and handler
	channelService := service.NewChannelService(channelRepo, plugin.GetGlobalRegistry(), producer)
	channelHandler := handlers.NewChannelHandler(channelService, producer)

	// Reconnect WhatsApp channels with stored sessions
	logger.Info("Reconnecting WhatsApp channels...")
	if reconnected, err := channelService.ReconnectWhatsAppChannels(context.Background()); err != nil {
		logger.Warn("Failed to reconnect some WhatsApp channels: " + err.Error())
	} else if reconnected > 0 {
		logger.Info(fmt.Sprintf("Reconnected %d WhatsApp channel(s)", reconnected))
	}

	// Create analytics handler
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	// Create WhatsApp Analytics handler
	whatsappAnalyticsHandler := handlers.NewWhatsAppAnalyticsHandler()

	// Create Payments handler
	paymentsHandler := handlers.NewPaymentsHandler()

	// Create Calling handler
	callingHandler := handlers.NewCallingHandler()

	// Create CTWA handler
	ctwaHandler := handlers.NewCTWAHandler()

	// Note: WhatsApp Analytics, Payments, Calling, and CTWA clients are registered per-channel
	// when channels are connected. The handlers use a channel-based client registry.
	// Example of registering clients (done when channel connects):
	// whatsappAnalyticsHandler.RegisterClient(channelID, analytics.NewClient(&analytics.ClientConfig{...}))
	// paymentsHandler.RegisterClient(channelID, payments.NewClient(&payments.ClientConfig{...}))
	// callingHandler.RegisterClient(channelID, calling.NewClient(&calling.ClientConfig{...}))
	// ctwaHandler.RegisterClient(channelID, ctwa.NewClient(&ctwa.ClientConfig{...}))
	_ = analytics.ClientConfig{} // Ensure import is used
	_ = payments.ClientConfig{}
	_ = calling.ClientConfig{}
	_ = ctwa.ClientConfig{}

	// Create VRE handler (if VRE service is available)
	var vreHandler *handlers.VREHandler
	if vreService != nil {
		vreHandler = handlers.NewVREHandler(vreService)
	}

	// Create OAuth handler
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
	}
	oauthHandler := handlers.NewOAuthHandler(channelRepo, baseURL)

	// Create WhatsApp Embedded Signup handler for Coexistence
	waEmbeddedSignupHandler := handlers.NewWhatsAppEmbeddedSignupHandler(channelRepo, baseURL)

	// Create template handler
	templateHandler := handlers.NewTemplateHandler(templateService)

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Create auth handler
	authHandler := handlers.NewAuthHandler(authService, userService)

	// Create user handler
	userHandler := handlers.NewUserHandler(userService)

	// Initialize Agent WebSocket Hub
	logger.Info("Starting Agent WebSocket Hub...")
	agentHub := handlers.GetAgentHub()
	wsHandler := handlers.NewWebSocketHandler(agentHub, cfg.JWT.Secret)

	// Start message consumers (only if NATS is available)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start coexistence monitor background job (runs every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run immediately on startup
		if err := coexistenceMonitor.MonitorCoexistenceActivity(ctx); err != nil {
			logger.Warn("Coexistence monitor check failed: " + err.Error())
		}

		for {
			select {
			case <-ctx.Done():
				logger.Info("Coexistence monitor stopped")
				return
			case <-ticker.C:
				if err := coexistenceMonitor.MonitorCoexistenceActivity(ctx); err != nil {
					logger.Warn("Coexistence monitor check failed: " + err.Error())
				}
			}
		}
	}()
	logger.Info("Coexistence monitor started (runs every hour)")

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

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
			webhooks.Any("/twilio/:channelId", webhookHandler.TwilioWebhook)
			webhooks.Any("/sms/:channelId", webhookHandler.TwilioWebhook) // Alias for Twilio
			webhooks.Any("/facebook/:channelId", webhookHandler.FacebookWebhook)
			webhooks.Any("/messenger/:channelId", webhookHandler.FacebookWebhook) // Alias for Facebook
			webhooks.Any("/instagram/:channelId", webhookHandler.InstagramWebhook)
			webhooks.Any("/rcs/:channelId", webhookHandler.RCSWebhook)
			webhooks.Any("/email/:channelId", webhookHandler.EmailWebhook)
			webhooks.Any("/email/:channelId/sendgrid", webhookHandler.EmailWebhook)
			webhooks.Any("/email/:channelId/mailgun", webhookHandler.EmailWebhook)
			webhooks.Any("/email/:channelId/ses", webhookHandler.EmailWebhook)
			webhooks.Any("/email/:channelId/postmark", webhookHandler.EmailWebhook)
			webhooks.POST("/generic/:channelId", webhookHandler.GenericWebhook)
			webhooks.POST("/status/:channelId", webhookHandler.StatusCallback)

			// WhatsApp-specific webhooks
			webhooks.POST("/payments/:channelId", paymentsHandler.HandleWebhook)
			webhooks.POST("/calls/:channelId", callingHandler.HandleWebhook)
			webhooks.POST("/ctwa/:channelId", ctwaHandler.ProcessReferralWebhook)
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
				channels.GET("", channelHandler.List)
				channels.POST("", channelHandler.Create)
				// Specific routes must come before generic /:id
				channels.PUT("/:id/status", channelHandler.UpdateStatus)
				channels.PUT("/:id/enabled", channelHandler.UpdateEnabled)
				channels.POST("/:id/connect", channelHandler.Connect)
				channels.POST("/:id/pair", channelHandler.RequestPairCode)
				channels.POST("/:id/disconnect", channelHandler.Disconnect)
				// WhatsApp Coexistence routes
				channels.GET("/:id/coexistence-status", waEmbeddedSignupHandler.GetCoexistenceStatus)
				channels.POST("/:id/subscribe-echoes", waEmbeddedSignupHandler.SubscribeMessageEchoes)
				// Generic routes last
				channels.GET("/:id", channelHandler.Get)
				channels.PUT("/:id", channelHandler.Update)
				channels.DELETE("/:id", channelHandler.Delete)
			}

			// OAuth routes for Facebook/Instagram
			oauth := protected.Group("/oauth")
			{
				// Facebook OAuth
				oauth.POST("/facebook/login", oauthHandler.FacebookLogin)
				oauth.POST("/facebook/callback", oauthHandler.FacebookCallback)

				// Instagram OAuth
				oauth.POST("/instagram/login", oauthHandler.InstagramLogin)
				oauth.POST("/instagram/callback", oauthHandler.InstagramCallback)

				// WhatsApp Embedded Signup (Coexistence)
				waEmbedded := oauth.Group("/whatsapp/embedded-signup")
				{
					waEmbedded.POST("/start", waEmbeddedSignupHandler.StartEmbeddedSignup)
					waEmbedded.POST("/callback", waEmbeddedSignupHandler.CompleteEmbeddedSignup)
					waEmbedded.POST("/create-channel", waEmbeddedSignupHandler.CreateCoexistenceChannel)
				}

				// Channel creation from OAuth
				oauth.POST("/channels", oauthHandler.CreateChannel)

				// Token refresh
				oauth.POST("/refresh", oauthHandler.RefreshToken)
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

			// WhatsApp Templates
			templates := protected.Group("/templates")
			{
				templates.GET("", templateHandler.List)
				templates.POST("", templateHandler.Create)
				templates.GET("/:id", templateHandler.Get)
				templates.DELETE("/:id", templateHandler.Delete)
				templates.POST("/sync/:channelId", templateHandler.Sync)
			}

			// Analytics
			analyticsRoutes := protected.Group("/analytics")
			{
				analyticsRoutes.GET("/overview", analyticsHandler.GetOverview)
				analyticsRoutes.GET("/conversations", analyticsHandler.GetConversations)
				analyticsRoutes.GET("/flows", analyticsHandler.GetFlows)
				analyticsRoutes.GET("/escalations", analyticsHandler.GetEscalations)
				analyticsRoutes.GET("/channels", analyticsHandler.GetChannels)
			}

			// WhatsApp Analytics (per-channel)
			waAnalytics := protected.Group("/channels/:id/analytics")
			{
				waAnalytics.GET("/conversations", whatsappAnalyticsHandler.GetConversationAnalytics)
				waAnalytics.GET("/phone", whatsappAnalyticsHandler.GetPhoneNumberAnalytics)
				waAnalytics.GET("/templates/:templateId", whatsappAnalyticsHandler.GetTemplateAnalytics)
				waAnalytics.GET("/stats", whatsappAnalyticsHandler.GetAggregatedStats)
				waAnalytics.GET("/export", whatsappAnalyticsHandler.ExportAnalytics)
				waAnalytics.GET("/dashboard", whatsappAnalyticsHandler.GetDashboardData)
			}

			// Payments (per-channel)
			paymentsRoutes := protected.Group("/channels/:id/payments")
			{
				paymentsRoutes.POST("", paymentsHandler.CreatePayment)
				paymentsRoutes.GET("/stats", paymentsHandler.GetPaymentStats)
				paymentsRoutes.GET("/:paymentId", paymentsHandler.GetPayment)
				paymentsRoutes.GET("/reference/:referenceId", paymentsHandler.GetPaymentByReference)
				paymentsRoutes.POST("/:paymentId/refund", paymentsHandler.ProcessRefund)
				paymentsRoutes.GET("/customer/:phone", paymentsHandler.GetCustomerPayments)
			}

			// Calling (per-channel)
			callingRoutes := protected.Group("/channels/:id/calls")
			{
				callingRoutes.GET("", callingHandler.GetRecentCalls)
				callingRoutes.POST("", callingHandler.InitiateCall)
				callingRoutes.GET("/stats", callingHandler.GetCallStats)
				callingRoutes.GET("/phone/:phone", callingHandler.GetCallsByPhone)
				callingRoutes.GET("/:callId", callingHandler.GetCall)
				callingRoutes.POST("/:callId/end", callingHandler.EndCall)
				callingRoutes.GET("/:callId/quality", callingHandler.GetCallQuality)
				callingRoutes.GET("/:callId/recording", callingHandler.GetCallRecording)
			}

			// CTWA (Click-to-WhatsApp Ads)
			ctwaRoutes := protected.Group("/channels/:id/ctwa")
			{
				ctwaRoutes.GET("/stats", ctwaHandler.GetStats)
				ctwaRoutes.GET("/dashboard", ctwaHandler.GetDashboard)
				ctwaRoutes.GET("/report", ctwaHandler.GenerateReport)
				ctwaRoutes.GET("/top-ads", ctwaHandler.GetTopAds)
				ctwaRoutes.GET("/free-window/:phone", ctwaHandler.GetFreeWindow)
				ctwaRoutes.GET("/referrals/:referralId", ctwaHandler.GetReferral)
				ctwaRoutes.GET("/referrals/phone/:phone", ctwaHandler.GetReferralByPhone)
				ctwaRoutes.GET("/referrals/:referralId/conversions", ctwaHandler.GetConversionsByReferral)
				ctwaRoutes.GET("/campaigns/:campaignId/referrals", ctwaHandler.GetReferralsByCampaign)
				ctwaRoutes.POST("/conversions", ctwaHandler.TrackConversion)
				ctwaRoutes.GET("/conversions/:conversionId", ctwaHandler.GetConversion)
			}

			// VRE (Visual Response Engine)
			if vreHandler != nil {
				vreRoutes := protected.Group("/vre")
				{
					vreRoutes.POST("/render", vreHandler.Render)
					vreRoutes.POST("/render-and-send", vreHandler.RenderAndSend)
					vreRoutes.GET("/templates", vreHandler.ListTemplates)
					vreRoutes.GET("/templates/:id/preview", vreHandler.PreviewTemplate)
					vreRoutes.POST("/templates/:id", vreHandler.UploadTemplate)
					vreRoutes.GET("/config", vreHandler.GetBrandConfig)
					vreRoutes.PUT("/config", vreHandler.UpdateBrandConfig)
					vreRoutes.DELETE("/cache", vreHandler.InvalidateCache)
				}
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

			// User management (admin only)
			users := protected.Group("/users")
			users.Use(authMiddleware.RequireRole("admin"))
			{
				users.GET("", userHandler.List)
				users.POST("", userHandler.Create)
				users.GET("/:id", userHandler.Get)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
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
	whatsappOfficialAdapter.Disconnect(context.Background())
	whatsappAdapter.Disconnect(context.Background())
	telegramAdapter.Disconnect(context.Background())
	smsAdapter.Disconnect(context.Background())
	facebookAdapter.Disconnect(context.Background())
	instagramAdapter.Disconnect(context.Background())
	rcsAdapter.Disconnect(context.Background())
	emailAdapter.Disconnect(context.Background())

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
