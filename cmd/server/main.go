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
// @tag.description Visual Response Engine - render SVG templates to images for channels that don't support interactive elements

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	redisv9 "github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/msgfy/linktor/docs" // Swagger docs

	"github.com/msgfy/linktor/internal/adapters/ai/anthropic"
	"github.com/msgfy/linktor/internal/adapters/ai/ollama"
	"github.com/msgfy/linktor/internal/adapters/ai/openai"
	"github.com/msgfy/linktor/internal/adapters/direto"
	"github.com/msgfy/linktor/internal/adapters/email"
	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/mattermost"
	"github.com/msgfy/linktor/internal/adapters/rcs"
	"github.com/msgfy/linktor/internal/adapters/slack"
	"github.com/msgfy/linktor/internal/adapters/sms"
	"github.com/msgfy/linktor/internal/adapters/teams"
	"github.com/msgfy/linktor/internal/adapters/telegram"
	"github.com/msgfy/linktor/internal/adapters/webchat"
	"github.com/msgfy/linktor/internal/adapters/whatsapp"
	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/api/handlers"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/config"
	"github.com/msgfy/linktor/internal/infrastructure/database"
	"github.com/msgfy/linktor/internal/infrastructure/metrics"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/outbox"
	storageLib "github.com/msgfy/linktor/internal/infrastructure/storage"
	infrawebhook "github.com/msgfy/linktor/internal/infrastructure/webhook"
	"github.com/msgfy/linktor/internal/integration/vendax"
	"github.com/msgfy/linktor/internal/outbound"
	"github.com/msgfy/linktor/internal/whatsapp/analytics"
	"github.com/msgfy/linktor/internal/whatsapp/calling"
	"github.com/msgfy/linktor/internal/whatsapp/ctwa"
	"github.com/msgfy/linktor/internal/whatsapp/payments"
	"github.com/msgfy/linktor/pkg/crypto"
	apperrors "github.com/msgfy/linktor/pkg/errors"
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

	// Seed demo data only outside release mode, unless explicitly enabled.
	if cfg.Server.Mode != "release" || os.Getenv("LINKTOR_SEED_DEMO") == "true" {
		if err := db.Seed(context.Background()); err != nil {
			logger.Error("Failed to seed database: " + err.Error())
		}
	} else {
		logger.Info("Skipping demo database seed in release mode")
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

	// Initialize the secret-at-rest encryptor for channel credentials/tokens.
	// Previous keys are accepted for decryption during a key rotation.
	encryptor, err := crypto.NewEncryptorWithKeys(cfg.Crypto.EncryptionKey, cfg.Crypto.PreviousKeys...)
	if err != nil {
		logger.Fatal("Failed to initialize crypto encryptor: " + err.Error())
	}

	// Initialize repositories
	logger.Info("Initializing repositories...")
	tenantRepo := database.NewTenantRepository(db)
	userRepo := database.NewUserRepository(db)
	messageRepo := database.NewMessageRepository(db)
	conversationRepo := database.NewConversationRepository(db)
	contactRepo := database.NewContactRepository(db)
	outboxRepo := database.NewOutboxRepository(db)
	channelRepo := database.NewChannelRepository(db, encryptor)
	botRepo := database.NewBotRepository(db)
	contextRepo := database.NewConversationContextRepository(db)
	aiResponseRepo := database.NewAIResponseRepository(db)
	kbRepo := database.NewKnowledgeBaseRepository(db)
	kiRepo := database.NewKnowledgeItemRepository(db)
	flowRepo := database.NewFlowRepository(db)
	analyticsRepo := database.NewAnalyticsRepository(db)
	templateRepo := database.NewTemplateRepository(db)
	historyImportRepo := database.NewHistoryImportRepository(db)
	paymentRepo := database.NewPaymentRepository(db)
	observabilityRepo := database.NewObservabilityRepository(db)
	apiKeyRepo := database.NewAPIKeyRepository(db)
	orderRepo := database.NewOrderRepository(db)
	cartRepo := database.NewCartRepository(db)
	auditLogRepo := database.NewAuditLogRepository(db)
	cannedRepo := database.NewCannedResponseRepository(db)
	roleRepo := database.NewRoleRepository(db)
	tenantSettingsRepo := database.NewTenantSettingsRepository(db)
	campaignRepo := database.NewCampaignRepository(db)
	sandboxAllowlistRepo := database.NewSandboxAllowlistRepository(db)

	// Initialize services
	logger.Info("Initializing services...")
	normalizer := service.NewMessageNormalizer()
	authService := service.NewAuthService(userRepo, &cfg.JWT)
	userService := service.NewUserService(userRepo, tenantRepo)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo)

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
	analyticsService := service.NewAnalyticsService(analyticsRepo, nil)
	observabilityService := service.NewObservabilityService(observabilityRepo, nats.NewMonitor(natsClient))

	// Initialize commerce + WhatsApp Flows services
	commerceService := service.NewCommerceService(channelRepo, orderRepo, cartRepo)
	whatsappFlowsService := service.NewWhatsAppFlowsService(flowRepo, channelRepo)

	// Initialize template service
	templateService := service.NewTemplateService(templateRepo, channelRepo)

	// Initialize coexistence monitor service
	coexistenceMonitor := service.NewCoexistenceMonitorService(channelRepo, producer)

	// Initialize history import service for WhatsApp Coexistence
	_ = service.NewHistoryImportService(channelRepo, conversationRepo, messageRepo, contactRepo, historyImportRepo)

	// Initialize VRE (Visual Response Engine) service
	logger.Info("Initializing VRE service...")
	var vreService *service.VREService
	var redisClient *redis.Client
	var rateLimiter *middleware.RateLimiter

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

	rateRedis := redisv9.NewClient(&redisv9.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rateRedis.Ping(context.Background()).Err(); err != nil {
		logger.Warn("Failed to connect to Redis - HTTP rate limiting disabled: " + err.Error())
		_ = rateRedis.Close()
	} else {
		defer rateRedis.Close()
		rateLimiter = middleware.NewRateLimiter(rateRedis, nil)
		logger.Info("Redis connected for HTTP rate limiting")
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

		// Configure storage for VRE CDN upload if upload dir is set
		if uploadDir := os.Getenv("VRE_UPLOAD_DIR"); uploadDir != "" {
			baseURL := os.Getenv("VRE_UPLOAD_BASE_URL")
			if baseURL == "" {
				baseURL = "/uploads/vre"
			}
			vreService.SetStorageClient(storageLib.NewLocalClient(uploadDir, baseURL))
			logger.Info("VRE storage configured: " + uploadDir)
		}

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

	// whatomate-inspired services: audit, canned, RBAC, assignment, SLA, campaigns.
	assignmentRepo := database.NewAssignmentRepository(db)
	slaRepo := database.NewSLARepository(db)
	auditService := service.NewAuditService(auditLogRepo)
	sandboxAllowlistService := service.NewSandboxAllowlistService(sandboxAllowlistRepo, channelRepo)
	cannedService := service.NewCannedResponseService(cannedRepo)
	roleService := service.NewRoleService(roleRepo, redisClient)
	settingsService := service.NewSettingsService(tenantSettingsRepo)
	assignmentService := service.NewAssignmentService(assignmentRepo, conversationRepo, tenantSettingsRepo)
	slaMonitor := service.NewSLAMonitorService(slaRepo, tenantSettingsRepo)
	campaignService := service.NewCampaignService(campaignRepo, channelRepo, producer)

	// Outbound delivery subsystem: per-channel stateless senders resolved from
	// (decrypted) credentials, driven by a NATS-backed worker. This is what
	// actually delivers send_message and campaign messages over the Cloud API.
	outboundResolver := outbound.NewResolver(channelRepo)
	// Sandbox delivery guard (INV-017): every sandbox channel's sender is
	// wrapped to consult the recipient allowlist at send time. Without this
	// wiring the resolver fails closed for sandbox channels.
	outboundResolver.SetSandboxAllowlist(sandboxAllowlistRepo)
	// WhatsApp 24h customer-service window (INV-015). Two-stage rollout via
	// LINKTOR_WA_WINDOW_ENFORCEMENT: off | dry_run (default: observe-only, no
	// behavior change) | enforce (block before the Meta call). Reverting
	// enforce → dry_run is a config change, no deploy.
	outboundResolver.AddPolicy(outbound.NewSessionWindowPolicy(
		outbound.ParsePolicyMode(os.Getenv("LINKTOR_WA_WINDOW_ENFORCEMENT")), messageRepo))
	outboundResolver.Register(whatsappofficial.NewSenderFactory())
	outboundResolver.Register(telegram.NewSenderFactory())
	outboundResolver.Register(sms.NewSenderFactory())
	outboundResolver.Register(facebook.NewSenderFactory())
	outboundResolver.Register(instagram.NewSenderFactory())
	outboundResolver.Register(rcs.NewSenderFactory())
	outboundResolver.Register(email.NewSenderFactory())
	outboundResolver.Register(teams.NewSenderFactory())
	outboundResolver.Register(slack.NewSenderFactory())
	outboundResolver.Register(mattermost.NewSenderFactory())
	outboundResolver.Register(direto.NewSenderFactory())
	// Stateful channels deliver through their live plugin adapter (whatsmeow
	// session / webchat WebSocket hub) but ride the same unified worker.
	outboundResolver.Register(outbound.NewPluginSenderFactory("webchat", plugin.GetGlobalRegistry()))
	outboundResolver.Register(outbound.NewPluginSenderFactory("whatsapp", plugin.GetGlobalRegistry()))
	outboundResolver.Register(outbound.NewPluginSenderFactory("whatsapp_unofficial", plugin.GetGlobalRegistry()))
	var outboundWorker *outbound.Worker
	if consumer != nil && producer != nil {
		// 80 msg/s/channel ~ WhatsApp Cloud API default throughput tier.
		outboundWorker = outbound.NewWorker(consumer, producer, outboundResolver, campaignRepo, 80)
		// Record delivery activity (sends, transient/permanent failures) to the
		// observability log store so the admin channel-log viewer shows why a
		// message did or did not reach the channel.
		outboundWorker.SetChannelLogger(channelActivityLogger{svc: observabilityService, source: entity.LogSourceChannel})
	}

	// Enable automatic conversation routing on inbound messages.
	receiveMessageUC.SetAssignmentService(assignmentService)

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

	// Media store re-hosts inbound attachments that sit behind authenticated
	// provider URLs (Slack url_private, Mattermost file API) so external
	// consumers can fetch them. Optional: nil falls back to the provider URL.
	mediaStore := buildMediaStore()

	// Create webhook handler
	requireWebhookSecrets := cfg.Server.Mode == "release" || os.Getenv("LINKTOR_REQUIRE_WEBHOOK_SECRETS") == "true"
	webhookHandler := handlers.NewWebhookHandler(channelRepo, producer, templateService, requireWebhookSecrets, mediaStore)

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
	observabilityHandler := handlers.NewObservabilityHandler(observabilityService)

	// Create contact service and handler
	contactService := service.NewContactService(contactRepo)
	contactHandler := handlers.NewContactHandler(contactService)

	// Create conversation service and handler
	conversationService := service.NewConversationService(conversationRepo, contactRepo, channelRepo, producer)
	conversationHandler := handlers.NewConversationHandler(conversationService, escalateConversationUC)

	// Create message service and handler
	messageService := service.NewMessageService(messageRepo, conversationRepo, channelRepo, contactRepo, producer)
	// Synchronous sandbox recipient check (UX): the API rejects immediately;
	// the authoritative guard remains in the outbound delivery funnel.
	messageService.SetSandboxAllowlist(sandboxAllowlistRepo)
	messageHandler := handlers.NewMessageHandler(messageService)
	attachmentHandler := handlers.NewAttachmentHandler(mediaStore)
	mediaHandler := handlers.NewMediaHandler(mediaStore)

	// Create flow handler
	flowHandler := handlers.NewFlowHandler(flowService)

	// Create channel service and handler
	channelService := service.NewChannelService(channelRepo, plugin.GetGlobalRegistry(), producer)
	// Record channel connection lifecycle (connect/disconnect/logout/failures)
	// to the observability log store for the admin channel-log viewer.
	channelService.SetObservability(observabilityService)
	// Optional allowlist of channel types this deployment may create/connect.
	// LINKTOR_ENABLED_CHANNEL_TYPES="whatsapp_official,telegram,..." restricts to
	// the channels that passed homologation; unset means all types are allowed.
	if raw := strings.TrimSpace(os.Getenv("LINKTOR_ENABLED_CHANNEL_TYPES")); raw != "" {
		var enabled []entity.ChannelType
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				enabled = append(enabled, entity.ChannelType(t))
			}
		}
		channelService.SetEnabledChannelTypes(enabled)
		logger.Info(fmt.Sprintf("channel type allowlist enabled: %v", enabled))
	}
	channelHandler := handlers.NewChannelHandler(channelService, producer)

	// Create tenant service and handler
	tenantService := service.NewTenantService(tenantRepo, userRepo, channelRepo, contactRepo)
	tenantHandler := handlers.NewTenantHandler(tenantService)

	// Create analytics handler
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	// Create WhatsApp Analytics handler
	whatsappAnalyticsHandler := handlers.NewWhatsAppAnalyticsHandler(channelRepo)

	// Create Payments handler
	paymentsHandler := handlers.NewPaymentsHandler(channelRepo)

	// Create Calling handler
	callingHandler := handlers.NewCallingHandler(channelRepo)

	// Create CTWA handler
	ctwaHandler := handlers.NewCTWAHandler(channelRepo)

	// whatomate-inspired handlers
	auditHandler := handlers.NewAuditHandler(auditService)
	sandboxAllowlistHandler := handlers.NewSandboxAllowlistHandler(sandboxAllowlistService, auditService)
	cannedHandler := handlers.NewCannedResponseHandler(cannedService, auditService)
	roleHandler := handlers.NewRoleHandler(roleService, auditService)
	settingsHandler := handlers.NewSettingsHandler(settingsService, assignmentService, auditService)
	campaignHandler := handlers.NewCampaignHandler(campaignService, auditService)
	permission := middleware.NewPermissionMiddleware(roleService)
	auditMw := middleware.NewAuditMiddleware(auditService)

	// Create Order handler (commerce)
	orderHandler := handlers.NewOrderHandler(orderRepo)
	whatsappFlowsHandler := handlers.NewWhatsAppFlowsHandler(whatsappFlowsService)
	_ = commerceService

	// Mattermost inbound manager: owns one reconnecting WebSocket listener per
	// connected Mattermost channel, started at boot and updated on connect/
	// disconnect so newly connected channels stream immediately.
	mattermostManager := mattermost.NewManager(channelRepo, producer, mediaStore)

	channelService.SetLifecycleHooks(service.ChannelLifecycleHooks{
		OnConnected: func(ctx context.Context, channel *entity.Channel) {
			registerWhatsAppAdvancedClient(channel, paymentRepo, whatsappAnalyticsHandler, paymentsHandler, callingHandler, ctwaHandler)
			if channel.Type == entity.ChannelTypeMattermost {
				mattermostManager.StartChannel(channel)
			}
		},
		OnUpdated: func(ctx context.Context, channel *entity.Channel) {
			// Credentials may have changed: drop the cached outbound sender.
			outboundResolver.Invalidate(channel.ID)
			if channel.IsConnected() {
				registerWhatsAppAdvancedClient(channel, paymentRepo, whatsappAnalyticsHandler, paymentsHandler, callingHandler, ctwaHandler)
				if channel.Type == entity.ChannelTypeMattermost {
					mattermostManager.StartChannel(channel) // restarts with fresh creds
				}
			} else if channel.Type == entity.ChannelTypeMattermost {
				mattermostManager.StopChannel(channel.ID)
			}
		},
		OnDisconnected: func(ctx context.Context, channel *entity.Channel) {
			outboundResolver.Invalidate(channel.ID)
			unregisterWhatsAppAdvancedClient(channel.ID, whatsappAnalyticsHandler, paymentsHandler, callingHandler, ctwaHandler)
			if channel.Type == entity.ChannelTypeMattermost {
				mattermostManager.StopChannel(channel.ID)
			}
		},
		OnDeleted: func(ctx context.Context, channel *entity.Channel) {
			outboundResolver.Invalidate(channel.ID)
			if channel.Type == entity.ChannelTypeMattermost {
				mattermostManager.StopChannel(channel.ID)
			}
		},
	})

	registerWhatsAppAdvancedClients(
		context.Background(),
		channelRepo,
		paymentRepo,
		whatsappAnalyticsHandler,
		paymentsHandler,
		callingHandler,
		ctwaHandler,
	)

	// Reconnect WhatsApp channels with stored sessions
	logger.Info("Reconnecting WhatsApp channels...")
	if reconnected, err := channelService.ReconnectWhatsAppChannels(context.Background()); err != nil {
		logger.Warn("Failed to reconnect some WhatsApp channels: " + err.Error())
	} else if reconnected > 0 {
		logger.Info(fmt.Sprintf("Reconnected %d WhatsApp channel(s)", reconnected))
	}

	// Create VRE handler (if VRE service is available)
	var vreHandler *handlers.VREHandler
	if vreService != nil {
		vreHandler = handlers.NewVREHandler(vreService, producer)
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
	authMiddleware := middleware.NewAuthMiddleware(authService, apiKeyService)

	// Create auth handler. Tokens are delivered to browsers as HttpOnly
	// cookies; TTLs mirror the JWT config (access in minutes, refresh in hours).
	authHandler := handlers.NewAuthHandler(authService, userService, handlers.AuthCookieConfig{
		Secure:        cfg.JWT.CookieSecure,
		Domain:        cfg.JWT.CookieDomain,
		SameSite:      handlers.ParseSameSite(cfg.JWT.CookieSameSite),
		AccessMaxAge:  cfg.JWT.AccessTokenTTL * 60,
		RefreshMaxAge: cfg.JWT.RefreshTokenTTL * 3600,
	})

	// Create user handler
	userHandler := handlers.NewUserHandler(userService)

	// Create API key handler
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyService)

	// Initialize Agent WebSocket Hub
	logger.Info("Starting Agent WebSocket Hub...")
	agentHub := handlers.GetAgentHub()
	wsHandler := handlers.NewWebSocketHandler(agentHub, cfg.JWT.Secret)

	// Start message consumers (only if NATS is available)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Background monitors are tracked so shutdown can wait for an in-flight
	// iteration (e.g. a sweep mid-write) instead of exiting under it.
	var backgroundJobs sync.WaitGroup

	// Start coexistence monitor background job (runs every hour)
	backgroundJobs.Add(1)
	go func() {
		defer backgroundJobs.Done()
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

	// Start SLA monitor (auto-close idle conversations, flag first-response breaches).
	backgroundJobs.Add(1)
	go func() {
		defer backgroundJobs.Done()
		slaMonitor.Start(ctx, 1*time.Minute)
	}()

	// Start campaign sweeper: reconcile recipients stuck in 'queued' (worker never
	// confirmed them — dead-lettered or down) into 'failed' so they can be retried.
	backgroundJobs.Add(1)
	go func() {
		defer backgroundJobs.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := campaignService.SweepStale(ctx, 15*time.Minute); err != nil {
					logger.Warn("campaign sweep failed: " + err.Error())
				}
			}
		}
	}()
	logger.Info("Campaign sweeper started (every 5m, stale threshold 15m)")

	var aiConsumer *nats.AIConsumer
	var vendaxBridge *vendax.Bridge

	// VendaX bridge (L0): integra o Linktor ao VendaX Core por NATS. Opt-in por env.
	// Ver docs/vendax-integration/PLANO-integracao-linktor-vendax.md.
	if natsClient != nil && os.Getenv("LINKTOR_VENDAX_BRIDGE_ENABLED") == "true" {
		logger.Info("Starting VendaX bridge...")
		vendaxBridge = vendax.NewBridge(natsClient, sendMessageUC, conversationRepo, contactRepo, channelRepo)
		if err := vendaxBridge.Start(ctx); err != nil {
			logger.Warn("Failed to start VendaX bridge: " + err.Error())
			vendaxBridge = nil
		}
	}

	if consumer != nil {
		logger.Info("Starting message consumers...")

		// Observe the dead-letter stream so exhausted messages surface in the
		// logs instead of dying silently.
		dlqConsumer := nats.NewDLQConsumer(natsClient)
		if err := dlqConsumer.Subscribe(ctx, nil); err != nil {
			logger.Warn("Failed to subscribe DLQ observer: " + err.Error())
		}

		// Subscribe to inbound messages. Expected/permanent errors (duplicate
		// delivery, validation, forbidden, not-found) are acked — retrying them is
		// pointless and would pollute the DLQ and fire an alert on every duplicate
		// webhook. Only genuinely transient errors are returned to trigger a NAK.
		inboundLogger := channelActivityLogger{svc: observabilityService, source: entity.LogSourceWebhook}
		if err := consumer.SubscribeAllInbound(ctx, func(ctx context.Context, msg *nats.InboundMessage) error {
			start := time.Now()
			inboundMeta := inboundLogMeta(msg)
			// Re-host any inline (base64) media to the object store so the chat
			// gets a real <img src> instead of an empty URL (broken image).
			rehostInboundMedia(ctx, mediaStore, channelRepo, msg)
			out, err := receiveMessageUC.Execute(ctx, msg)
			if err != nil {
				if !isRetryableInboundError(err) {
					// A conflict means the message was a duplicate delivery, which
					// is an expected outcome rather than a processing failure.
					result := metrics.ResultFailed
					var appErr *apperrors.AppError
					if errors.As(err, &appErr) && appErr.Code == apperrors.ErrCodeConflict {
						result = metrics.ResultDuplicate
					}
					metrics.RecordInbound(msg.ChannelType, result, start)
					if result == metrics.ResultDuplicate {
						inboundLogger.Info(ctx, msg.TenantID, msg.ChannelID, "Webhook duplicado ignorado", inboundMeta)
					} else {
						inboundLogger.Warn(ctx, msg.TenantID, msg.ChannelID, "Webhook recebido descartado (não recuperável): "+err.Error(), inboundMeta)
					}
					logger.Warn("inbound message dropped (non-retryable): " + err.Error())
					return nil // ack: do not retry
				}
				metrics.RecordInbound(msg.ChannelType, metrics.ResultFailed, start)
				inboundLogger.Warn(ctx, msg.TenantID, msg.ChannelID, "Falha temporária ao processar webhook (será reprocessado): "+err.Error(), inboundMeta)
				return err
			}
			metrics.RecordInbound(msg.ChannelType, metrics.ResultProcessed, start)
			// Execute resolves and writes the authoritative tenant onto msg.
			inboundLogger.Info(ctx, msg.TenantID, msg.ChannelID, "Mensagem recebida do canal", inboundMeta)
			// Push the new message to agents on this tenant so an open conversation
			// updates live. Without this the admin WebSocket only carried presence/
			// typing and inbound messages appeared only after a manual refetch.
			if out != nil && out.Message != nil {
				handlers.BroadcastNewMessage(msg.TenantID, out.Message.ConversationID, out.Message)
			}
			maybeTriggerBot(ctx, botRepo, producer, out)
			return nil
		}); err != nil {
			logger.Warn("Failed to subscribe to inbound messages")
		}

		// Reflect NATS connectivity into the Prometheus gauge (linktor_nats_up)
		// and sample queue depth / consumer lag (incl. DLQ backlog) into gauges.
		if natsClient != nil {
			metrics.SetNATSUp(natsClient.IsConnected())
			go func() {
				t := time.NewTicker(10 * time.Second)
				defer t.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						metrics.SetNATSUp(natsClient.IsConnected())
					}
				}
			}()
			go nats.NewMonitor(natsClient).StartMetricsCollector(ctx, 15*time.Second)

			// Drain the transactional outbox: publish durably-queued events
			// (e.g. message.received) that were written in the same tx as their
			// aggregate. The relay republishes with each event's idempotency key,
			// so a retry after a partial failure is deduplicated by JetStream.
			go outbox.NewRelay(outboxRepo, producer).Start(ctx, 2*time.Second)
		}

		// Subscribe to status updates
		if err := consumer.SubscribeStatus(ctx, func(ctx context.Context, status *nats.StatusUpdate) error {
			return handleMessageStatusUpdate(ctx, messageRepo, conversationRepo, campaignRepo, producer, status)
		}); err != nil {
			logger.Warn("Failed to subscribe to status updates")
		}

		// Outbound webhook pipeline (durable). The dispatcher converts internal
		// message/status events into `linktor-channel-v1` envelopes and enqueues
		// them on the durable webhooks stream; the delivery worker consumes that
		// stream, signs with a fresh timestamp and performs the HTTP POST with
		// NATS-driven retry + DLQ. Splitting enqueue from delivery makes the
		// pipeline crash-safe: an event is never lost between receipt and delivery.
		webhookDeliveryWorker := infrawebhook.NewDeliveryWorker(infrawebhook.NewWebhookProducer(), channelRepo)
		if err := consumer.SubscribeWebhooks(ctx, webhookDeliveryWorker.Handle); err != nil {
			logger.Warn("Failed to start webhook delivery worker: " + err.Error())
		} else {
			logger.Info("Webhook delivery worker started")
		}

		webhookDispatcher := infrawebhook.NewDispatcher(producer, channelRepo)
		if err := webhookDispatcher.Start(ctx, consumer); err != nil {
			logger.Warn("Failed to start outbound webhook dispatcher: " + err.Error())
		} else {
			logger.Info("Outbound webhook dispatcher started")
		}

		// Bind the Mattermost manager to the app context and start listeners for
		// already-connected channels. Runtime connect/disconnect is handled via
		// the channel lifecycle hooks above.
		if err := mattermostManager.Start(ctx); err != nil {
			logger.Warn("Failed to start Mattermost listeners: " + err.Error())
		}

		// Start the outbound delivery worker.
		if outboundWorker != nil {
			if err := outboundWorker.Start(ctx); err != nil {
				logger.Warn("Failed to start outbound delivery worker: " + err.Error())
			}
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
	router.Use(metrics.GinMiddleware())
	router.Use(middleware.Logger())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS())

	// Prometheus scrape endpoint. Unauthenticated like /health so scrapers reach
	// it without credentials; restrict it at the ingress/network layer if the
	// deployment exposes the router publicly.
	router.GET("/metrics", gin.WrapH(metrics.Handler()))

	// Health check endpoints. Readiness verifies every hard dependency
	// (Postgres, Redis, and — when enabled — NATS); a disconnected broker fails
	// readiness so the orchestrator stops routing traffic to an instance that
	// cannot process the messaging pipeline.
	var redisPing func(context.Context) error
	if redisClient != nil {
		redisPing = func(ctx context.Context) error { return redisClient.Ping(ctx).Err() }
	}
	healthHandler := handlers.NewHealthHandler(db.Pool, redisPing)
	if natsClient != nil {
		healthHandler.SetNATSChecker(natsClient.IsConnected)
	}
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// WebSocket endpoint for WebChat
	router.GET("/ws/:channelId", webchatHandler.WebSocketHandler)

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := api.Group("/auth")
		if rateLimiter != nil {
			auth.Use(rateLimiter.LimitByKey(func(c *gin.Context) string {
				return "auth:" + c.ClientIP()
			}))
		}
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}

		// WebChat widget config (no auth required)
		api.GET("/webchat/:channelId/config", webchatHandler.GetWidgetConfig)

		// Media proxy (no session): streams stored attachments by opaque key so
		// message URLs stay stable and never expire (vs presigned S3 links).
		api.GET("/media/*key", mediaHandler.Serve)

		// Webhook routes (auth via signature verification)
		webhooks := api.Group("/webhooks")
		if rateLimiter != nil {
			webhooks.Use(rateLimiter.LimitByKey(func(c *gin.Context) string {
				return "webhook:" + c.ClientIP()
			}))
		}
		if rateLimiter != nil {
			webhookDedup := middleware.NewWebhookDedup(rateRedis, 6*time.Hour)
			webhooks.Use(webhookDedup.Middleware("global"))
			logger.Info("Webhook deduplication enabled (Redis-backed, 6h TTL)")
		}
		{
			webhooks.Any("/whatsapp/:channelId", webhookHandler.WhatsAppWebhook)
			webhooks.POST("/direto/:channelId", webhookHandler.DiretoWebhook)
			webhooks.POST("/telegram/:channelId", webhookHandler.TelegramWebhook)
			webhooks.POST("/teams/:channelId", webhookHandler.TeamsWebhook)
			// Multi-tenant Teams: one shared Bot Framework messaging endpoint (no
			// channelId) resolves the channel from the token audience + AAD tenant.
			webhooks.POST("/teams", webhookHandler.TeamsWebhookShared)
			webhooks.POST("/slack/:channelId", webhookHandler.SlackWebhook)
			webhooks.Any("/twilio/:channelId", webhookHandler.TwilioWebhook)
			webhooks.Any("/sms/:channelId", webhookHandler.TwilioWebhook) // Alias for Twilio
			webhooks.Any("/facebook/:channelId", webhookHandler.FacebookWebhook)
			webhooks.Any("/messenger/:channelId", webhookHandler.FacebookWebhook) // Alias for Facebook
			webhooks.Any("/instagram/:channelId", webhookHandler.InstagramWebhook)
			webhooks.Any("/rcs/:channelId", webhookHandler.RCSWebhook)
			webhooks.Any("/voice/:channelId", webhookHandler.VoiceWebhook)
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
		if rateLimiter != nil {
			protected.Use(rateLimiter.Limit())
		}
		{
			// User info
			protected.GET("/me", authHandler.Me)
			protected.PUT("/me", authHandler.UpdateMe)
			protected.PUT("/me/password", authHandler.ChangePassword)

			// WebChat widget token: an authenticated backend mints a short-lived
			// signed token for its browser widget, which then opens the public
			// /ws/:channelId WebSocket. Enforcement is opt-in: the WS handler only
			// requires a token when the channel has a widget_secret configured.
			protected.POST("/webchat/:channelId/widget-token", func(c *gin.Context) {
				tenantID := middleware.MustGetTenantID(c)
				if tenantID == "" {
					return
				}
				ch, err := channelService.GetByTenantAndID(c.Request.Context(), tenantID, c.Param("channelId"))
				if err != nil {
					handlers.RespondError(c, err)
					return
				}
				if ch.Type != entity.ChannelTypeWebChat {
					c.JSON(http.StatusBadRequest, gin.H{"error": "not a webchat channel"})
					return
				}
				secret := ch.Config["widget_secret"]
				if secret == "" {
					secret = ch.Credentials["widget_secret"]
				}
				if secret == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "widget_secret not configured for this channel"})
					return
				}
				tok, err := webchat.IssueWidgetToken(ch.ID, secret, webchat.DefaultWidgetTokenTTL, time.Now())
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue widget token"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"token": tok, "expires_in": int(webchat.DefaultWidgetTokenTTL.Seconds())})
			})

			// Tenant/Organization
			protected.GET("/tenant", tenantHandler.Get)
			protected.PUT("/tenant", authMiddleware.RequireRole("admin", "owner"), auditMw.Record(), tenantHandler.Update)
			protected.GET("/tenant/usage", tenantHandler.GetUsage)

			// Conversations
			conversations := protected.Group("/conversations")
			{
				conversations.GET("", conversationHandler.List)
				conversations.POST("", conversationHandler.Create)
				conversations.GET("/:id", conversationHandler.Get)
				conversations.PUT("/:id", conversationHandler.Update)
				conversations.POST("/:id/assign", conversationHandler.Assign)
				conversations.POST("/:id/resolve", conversationHandler.Resolve)
				conversations.POST("/:id/reopen", conversationHandler.Reopen)
				conversations.GET("/:id/escalation-context", conversationHandler.GetEscalationContext)
				conversations.POST("/:id/escalate", conversationHandler.Escalate)
				// Messages within a conversation
				conversations.GET("/:id/messages", messageHandler.List)
				conversations.POST("/:id/messages", messageHandler.Send)
				conversations.POST("/:id/messages/:messageId/reactions", messageHandler.SendReaction)
				// Upload media to attach to an outgoing message. Kept off the
				// /messages/:messageId path so the static segment doesn't collide
				// with the :messageId wildcard in gin's route tree.
				conversations.POST("/:id/attachments", attachmentHandler.Upload)
			}

			// Messages (direct access by ID)
			protected.GET("/messages/:id", messageHandler.Get)

			// Contacts
			contacts := protected.Group("/contacts")
			contacts.Use(auditMw.Record())
			{
				contacts.GET("", contactHandler.List)
				contacts.POST("", contactHandler.Create)
				contacts.GET("/:id", contactHandler.Get)
				contacts.PUT("/:id", contactHandler.Update)
				contacts.DELETE("/:id", contactHandler.Delete)
				contacts.POST("/:id/identities", contactHandler.AddIdentity)
				contacts.DELETE("/:id/identities/:identityId", contactHandler.RemoveIdentity)
			}

			// Channels
			channels := protected.Group("/channels")
			channels.Use(auditMw.Record())
			{
				channels.GET("", channelHandler.List)
				channels.POST("", channelHandler.Create)
				// Specific routes must come before generic /:id
				channels.POST("/reencrypt", authMiddleware.RequireRole("admin", "owner"), channelHandler.Reencrypt)
				channels.POST("/test", channelHandler.TestConnection)
				channels.POST("/test-whatsapp", channelHandler.TestWhatsAppConnection)
				channels.POST("/test-telegram", channelHandler.TestTelegramConnection)
				channels.POST("/test-twilio", channelHandler.TestTwilioConnection)
				channels.POST("/test-facebook", channelHandler.TestFacebookConnection)
				channels.POST("/test-instagram", channelHandler.TestInstagramConnection)
				channels.POST("/test-teams", channelHandler.TestTeamsConnection)
				channels.POST("/test-slack", channelHandler.TestSlackConnection)
				channels.POST("/test-mattermost", channelHandler.TestMattermostConnection)
				channels.PUT("/:id/status", channelHandler.UpdateStatus)
				channels.PUT("/:id/enabled", channelHandler.UpdateEnabled)
				channels.POST("/:id/connect", channelHandler.Connect)
				channels.POST("/:id/pair", channelHandler.RequestPairCode)
				channels.POST("/:id/passkey/response", channelHandler.SubmitPasskeyResponse)
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

			// Observability
			observability := protected.Group("/observability")
			observability.Use(authMiddleware.RequireRole("admin", "owner"))
			{
				observability.GET("/logs", observabilityHandler.GetLogs)
				observability.POST("/logs/cleanup", observabilityHandler.CleanupLogs)
				observability.GET("/queue", observabilityHandler.GetQueueStats)
				observability.GET("/queue/:stream", observabilityHandler.GetStreamInfo)
				observability.POST("/queue/reset-consumer", observabilityHandler.ResetConsumer)
				observability.GET("/stats", observabilityHandler.GetSystemStats)
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
				templates.DELETE("", templateHandler.BulkDelete)
				templates.GET("/library", templateHandler.ListLibrary)
				templates.POST("/library", templateHandler.CreateFromLibrary)
				templates.GET("/:id", templateHandler.Get)
				templates.PATCH("/:id", templateHandler.Edit)
				templates.DELETE("/:id", templateHandler.Delete)
				templates.POST("/:id/refresh", templateHandler.Refresh)
				templates.POST("/sync/:channelId", templateHandler.Sync)
				templates.GET("/namespace/:channel_id", templateHandler.FetchNamespace)
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

			// WhatsApp Flows (Meta API)
			waFlows := protected.Group("/whatsapp/flows")
			{
				waFlows.POST("", whatsappFlowsHandler.CreateFlow)
				waFlows.PUT("/:id", whatsappFlowsHandler.UpdateFlow)
				waFlows.DELETE("/:id", whatsappFlowsHandler.DeleteFlow)
				waFlows.POST("/:id/publish", whatsappFlowsHandler.PublishFlow)
				waFlows.GET("/:id/preview", whatsappFlowsHandler.GetFlowPreview)
				waFlows.POST("/sync", whatsappFlowsHandler.SyncFlows)
			}

			// Orders (commerce)
			orderRoutes := protected.Group("/orders")
			{
				orderRoutes.GET("", orderHandler.ListOrders)
				orderRoutes.GET("/stats", orderHandler.GetOrderStats)
				orderRoutes.GET("/:orderId", orderHandler.GetOrder)
				orderRoutes.GET("/:orderId/history", orderHandler.GetOrderHistory)
				orderRoutes.PATCH("/:orderId/status", orderHandler.UpdateOrderStatus)
				orderRoutes.PATCH("/:orderId/shipping", orderHandler.UpdateShipping)
				orderRoutes.POST("/:orderId/cancel", orderHandler.CancelOrder)
			}
			protected.GET("/customers/:phone/orders", orderHandler.GetCustomerOrders)

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

			// User management (admin only)
			users := protected.Group("/users")
			users.Use(authMiddleware.RequireRole("admin", "owner"))
			users.Use(auditMw.Record())
			{
				users.GET("", userHandler.List)
				users.POST("", userHandler.Create)
				users.GET("/:id", userHandler.Get)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
			}

			// API keys (admin only)
			apiKeys := protected.Group("/api-keys")
			apiKeys.Use(authMiddleware.RequireRole("admin", "owner"))
			{
				apiKeys.GET("", apiKeyHandler.List)
				apiKeys.POST("", apiKeyHandler.Create)
				apiKeys.DELETE("/:id", apiKeyHandler.Delete)
			}

			// Canned responses (quick replies)
			canned := protected.Group("/canned-responses")
			{
				canned.GET("", cannedHandler.List)
				canned.POST("", permission.Require(entity.ResourceCanned, entity.ActionCreate), cannedHandler.Create)
				canned.GET("/use/:shortcut", cannedHandler.Use)
				canned.GET("/:id", cannedHandler.Get)
				canned.PUT("/:id", permission.Require(entity.ResourceCanned, entity.ActionUpdate), cannedHandler.Update)
				canned.DELETE("/:id", permission.Require(entity.ResourceCanned, entity.ActionDelete), cannedHandler.Delete)
			}

			// RBAC roles (admin only)
			roles := protected.Group("/roles")
			roles.Use(authMiddleware.RequireRole("admin", "owner"))
			{
				roles.GET("/catalog", roleHandler.Catalog)
				roles.GET("", roleHandler.List)
				roles.POST("", roleHandler.Create)
				roles.GET("/:id", roleHandler.Get)
				roles.PUT("/:id", roleHandler.Update)
				roles.DELETE("/:id", roleHandler.Delete)
			}

			// Tenant operational settings (assignment + SLA)
			settings := protected.Group("/settings")
			{
				settings.GET("", settingsHandler.Get)
				settings.PUT("", authMiddleware.RequireRole("admin", "owner"), settingsHandler.Update)
			}
			// Auto-assign a conversation to an available agent now
			protected.POST("/conversations/:id/auto-assign", settingsHandler.AutoAssign)

			// Audit trail (admin only)
			audit := protected.Group("/audit-logs")
			audit.Use(authMiddleware.RequireRole("admin", "owner"))
			{
				audit.GET("", auditHandler.List)
			}

			// Sandbox recipient allowlist (INV-017). Admin only: the allowlist
			// is the security boundary keeping synthetic traffic away from real
			// recipients, so editing it is a privileged operation.
			sandboxAllowlist := protected.Group("/sandbox/allowlist")
			sandboxAllowlist.Use(authMiddleware.RequireRole("admin", "owner"))
			{
				sandboxAllowlist.GET("", sandboxAllowlistHandler.List)
				sandboxAllowlist.POST("", sandboxAllowlistHandler.Add)
				sandboxAllowlist.DELETE("/:id", sandboxAllowlistHandler.Remove)
			}

			// Bulk campaigns
			campaigns := protected.Group("/campaigns")
			{
				campaigns.GET("", campaignHandler.List)
				campaigns.POST("", permission.Require(entity.ResourceCampaigns, entity.ActionCreate), campaignHandler.Create)
				campaigns.GET("/:id", campaignHandler.Get)
				campaigns.GET("/:id/recipients", campaignHandler.ListRecipients)
				campaigns.POST("/:id/start", permission.Require(entity.ResourceCampaigns, entity.ActionUpdate), campaignHandler.Start)
				campaigns.POST("/:id/retry", permission.Require(entity.ResourceCampaigns, entity.ActionUpdate), campaignHandler.Retry)
				campaigns.POST("/:id/cancel", permission.Require(entity.ResourceCampaigns, entity.ActionUpdate), campaignHandler.Cancel)
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

	// Stop VendaX bridge
	if vendaxBridge != nil {
		vendaxBridge.Stop()
	}

	// Wait (bounded) for background monitors to finish their current iteration.
	backgroundDone := make(chan struct{})
	go func() {
		backgroundJobs.Wait()
		close(backgroundDone)
	}()
	select {
	case <-backgroundDone:
	case <-time.After(10 * time.Second):
		logger.Warn("Background monitors did not stop within 10s, continuing shutdown")
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

func registerWhatsAppAdvancedClients(
	ctx context.Context,
	channelRepo *database.ChannelRepository,
	paymentRepo *database.PaymentRepository,
	whatsappAnalyticsHandler *handlers.WhatsAppAnalyticsHandler,
	paymentsHandler *handlers.PaymentsHandler,
	callingHandler *handlers.CallingHandler,
	ctwaHandler *handlers.CTWAHandler,
) {
	channels, err := channelRepo.FindByTypes(ctx, []entity.ChannelType{
		entity.ChannelTypeWhatsApp,
		entity.ChannelTypeWhatsAppOfficial,
	})
	if err != nil {
		logger.Warn("Failed to load WhatsApp channels for advanced clients: " + err.Error())
		return
	}

	registered := 0
	for _, channel := range channels {
		if registerWhatsAppAdvancedClient(channel, paymentRepo, whatsappAnalyticsHandler, paymentsHandler, callingHandler, ctwaHandler) {
			registered++
		}
	}

	if registered > 0 {
		logger.Info(fmt.Sprintf("Registered advanced WhatsApp clients for %d channel(s)", registered))
	}
}

func registerWhatsAppAdvancedClient(
	channel *entity.Channel,
	paymentRepo *database.PaymentRepository,
	whatsappAnalyticsHandler *handlers.WhatsAppAnalyticsHandler,
	paymentsHandler *handlers.PaymentsHandler,
	callingHandler *handlers.CallingHandler,
	ctwaHandler *handlers.CTWAHandler,
) bool {
	if channel == nil || !channel.Enabled {
		return false
	}
	if channel.Type != entity.ChannelTypeWhatsApp && channel.Type != entity.ChannelTypeWhatsAppOfficial {
		return false
	}

	accessToken := channelConfigValue(channel, "access_token")
	phoneNumberID := channelConfigValue(channel, "phone_number_id")
	if accessToken == "" || phoneNumberID == "" {
		unregisterWhatsAppAdvancedClient(channel.ID, whatsappAnalyticsHandler, paymentsHandler, callingHandler, ctwaHandler)
		return false
	}

	apiVersion := channelConfigValue(channel, "api_version")
	businessID := channelConfigValue(channel, "business_id")
	if businessID == "" {
		businessID = channelConfigValue(channel, "waba_id")
	}
	if businessID == "" {
		businessID = channel.WABAID
	}

	whatsappAnalyticsHandler.RegisterClient(channel.ID, analytics.NewClient(&analytics.ClientConfig{
		AccessToken:   accessToken,
		BusinessID:    businessID,
		PhoneNumberID: phoneNumberID,
		APIVersion:    apiVersion,
	}))
	paymentsHandler.RegisterClient(channel.ID, payments.NewClient(&payments.ClientConfig{
		AccessToken:    accessToken,
		PhoneNumberID:  phoneNumberID,
		APIVersion:     apiVersion,
		GatewayConfig:  paymentGatewayConfig(channel.Config),
		OrganizationID: channel.TenantID,
		ChannelID:      channel.ID,
		Store:          paymentRepo,
	}))
	callingHandler.RegisterClient(channel.ID, calling.NewClient(&calling.ClientConfig{
		AccessToken:   accessToken,
		PhoneNumberID: phoneNumberID,
		APIVersion:    apiVersion,
	}))
	ctwaHandler.RegisterClient(channel.ID, ctwa.NewClient(&ctwa.ClientConfig{}))

	return true
}

func unregisterWhatsAppAdvancedClient(
	channelID string,
	whatsappAnalyticsHandler *handlers.WhatsAppAnalyticsHandler,
	paymentsHandler *handlers.PaymentsHandler,
	callingHandler *handlers.CallingHandler,
	ctwaHandler *handlers.CTWAHandler,
) {
	whatsappAnalyticsHandler.UnregisterClient(channelID)
	paymentsHandler.UnregisterClient(channelID)
	callingHandler.UnregisterClient(channelID)
	ctwaHandler.UnregisterClient(channelID)
}

func channelConfigValue(channel *entity.Channel, key string) string {
	if channel.Credentials != nil {
		if value := channel.Credentials[key]; value != "" {
			return value
		}
	}
	if channel.Config != nil {
		return channel.Config[key]
	}
	return ""
}

func paymentGatewayConfig(config map[string]string) *payments.GatewayConfig {
	if config == nil {
		return nil
	}

	gatewayType := config["payment_gateway_type"]
	if gatewayType == "" {
		gatewayType = config["gateway_type"]
	}
	if gatewayType == "" {
		return nil
	}

	return &payments.GatewayConfig{
		Type:          payments.GatewayType(gatewayType),
		APIKey:        config["payment_api_key"],
		APISecret:     config["payment_api_secret"],
		MerchantID:    config["payment_merchant_id"],
		WebhookSecret: config["payment_webhook_secret"],
		WebhookURL:    config["payment_webhook_url"],
		SandboxMode:   config["payment_sandbox_mode"] == "true" || config["sandbox_mode"] == "true",
	}
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

// toRecipientStatus maps a wire status to a campaign recipient status, or ""
// when the status is not one tracked per recipient.
func toRecipientStatus(status string) entity.RecipientStatus {
	switch status {
	case "sent":
		return entity.RecipientSent
	case "delivered":
		return entity.RecipientDelivered
	case "read":
		return entity.RecipientRead
	case "failed":
		return entity.RecipientFailed
	default:
		return ""
	}
}

// isRetryableInboundError reports whether an inbound processing error is worth a
// NATS redelivery. Application-level errors that will never succeed on retry
// (duplicate/conflict, validation/bad-request, forbidden, not-found,
// unauthorized) are treated as non-retryable so they are acked instead of
// cycling to the DLQ. Anything else (DB hiccups, transient infra) is retryable.
func isRetryableInboundError(err error) bool {
	if err == nil {
		return false
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case apperrors.ErrCodeConflict,
			apperrors.ErrCodeValidation,
			apperrors.ErrCodeBadRequest,
			apperrors.ErrCodeForbidden,
			apperrors.ErrCodeUnauthorized,
			apperrors.ErrCodeNotFound:
			return false
		}
	}
	return true
}

// maybeTriggerBot fires the bot auto-reply pipeline for an inbound message when
// it is eligible: it must come from a contact (never echo bot/agent messages),
// the conversation must not already be assigned to a human agent, and the
// channel must have an active bot. It publishes a BotResponseRequest that the AI
// consumer turns into a generated reply. Without this, the bot subscribers were
// wired but never received traffic, so auto-reply never happened.
func maybeTriggerBot(ctx context.Context, botRepo *database.BotRepository, producer *nats.Producer, out *usecase.ReceiveMessageOutput) {
	if producer == nil || botRepo == nil || out == nil || out.Message == nil || out.Conversation == nil {
		return
	}
	if out.Message.SenderType != entity.SenderTypeContact {
		return // only respond to inbound contact messages
	}
	if out.Conversation.AssignedUserID != nil && *out.Conversation.AssignedUserID != "" {
		return // a human agent owns this conversation
	}
	bot, err := botRepo.FindByChannel(ctx, out.Conversation.ChannelID)
	if err != nil || bot == nil {
		return // no active bot on this channel
	}
	contactID := out.Conversation.ContactID
	if out.Contact != nil {
		contactID = out.Contact.ID
	}
	if err := producer.PublishBotResponse(ctx, &nats.BotResponseRequest{
		MessageID:      out.Message.ID,
		ConversationID: out.Conversation.ID,
		TenantID:       out.Conversation.TenantID,
		ChannelID:      out.Conversation.ChannelID,
		ContactID:      contactID,
		BotID:          bot.ID,
		Content:        out.Message.Content,
		Timestamp:      time.Now(),
	}); err != nil {
		logger.Warn("failed to trigger bot response: " + err.Error())
	}
}

func handleMessageStatusUpdate(ctx context.Context, messageRepo *database.MessageRepository, conversationRepo *database.ConversationRepository, campaignRepo *database.CampaignRepository, producer nats.Publisher, status *nats.StatusUpdate) error {
	if status == nil {
		return nil
	}

	// Correlate delivery/read/failed status back to a campaign recipient by the
	// provider message ID (stored on the recipient when the worker sent it).
	if campaignRepo != nil {
		if rs := toRecipientStatus(status.Status); rs != "" {
			if status.ExternalID != "" {
				_ = campaignRepo.UpdateRecipientStatusByMessageID(ctx, status.ExternalID, rs)
			}
			if status.MessageID != "" && status.MessageID != status.ExternalID {
				_ = campaignRepo.UpdateRecipientStatusByMessageID(ctx, status.MessageID, rs)
			}
		}
	}

	messageID := status.MessageID
	if messageID == "" && status.ExternalID != "" {
		message, err := messageRepo.FindByExternalID(ctx, status.ExternalID)
		if err != nil {
			logger.Warn("Ignoring status update for unknown external message id: " + status.ExternalID)
			return nil
		}
		messageID = message.ID
	}
	if messageID == "" {
		logger.Warn("Ignoring status update without message id")
		return nil
	}

	if status.ExternalID != "" {
		if message, err := messageRepo.FindByID(ctx, messageID); err == nil && message.ExternalID == "" {
			message.ExternalID = status.ExternalID
			if updateErr := messageRepo.Update(ctx, message); updateErr != nil {
				return updateErr
			}
		}
	}

	if err := messageRepo.UpdateStatus(ctx, messageID, toMessageStatus(status.Status), status.ErrorMessage); err != nil {
		logger.Warn("Ignoring status update for unknown message id: " + messageID)
		return nil
	}

	// Emit an outbound-webhook status event so external consumers (DeskLenz)
	// receive message.{sent,delivered,read,failed}. Best-effort: never block the
	// status pipeline on event publication.
	publishStatusWebhookEvent(ctx, messageRepo, conversationRepo, producer, messageID, status)

	return nil
}

// publishStatusWebhookEvent publishes a `message.<status>` event carrying the
// channel id so the webhook dispatcher can resolve the channel's endpoint.
func publishStatusWebhookEvent(ctx context.Context, messageRepo *database.MessageRepository, conversationRepo *database.ConversationRepository, producer nats.Publisher, messageID string, status *nats.StatusUpdate) {
	if producer == nil || conversationRepo == nil {
		return
	}
	eventType := statusEventType(status.Status)
	if eventType == "" {
		return
	}

	message, err := messageRepo.FindByID(ctx, messageID)
	if err != nil || message == nil {
		return
	}
	conversation, err := conversationRepo.FindByID(ctx, message.ConversationID)
	if err != nil || conversation == nil {
		return
	}

	_ = producer.PublishEvent(ctx, &nats.Event{
		Type:     eventType,
		TenantID: conversation.TenantID,
		Payload: map[string]interface{}{
			"message_id":  messageID,
			"channel_id":  conversation.ChannelID,
			"external_id": status.ExternalID,
			"status":      status.Status,
			"error":       status.ErrorMessage,
		},
		Timestamp: time.Now(),
	})
}

// buildMediaStore constructs the optional inbound-media store from the
// environment, preferring MinIO (S3-compatible) when configured and falling back
// to a local filesystem store, then to nil. When nil, inbound media re-hosting
// is disabled and provider URLs are passed through unchanged.
func buildMediaStore() storageLib.Client {
	// Preferred: MinIO / S3-compatible object storage.
	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		bucket := os.Getenv("MINIO_BUCKET")
		if bucket == "" {
			bucket = "linktor-media"
		}
		client, err := storageLib.NewMinIOClient(
			endpoint,
			os.Getenv("MINIO_ACCESS_KEY"),
			os.Getenv("MINIO_SECRET_KEY"),
			bucket,
			os.Getenv("MINIO_REGION"),
			os.Getenv("MINIO_USE_SSL") == "true",
			mediaPublicBaseURL(),
		)
		if err != nil {
			logger.Warn("Inbound media store: MinIO init failed, falling back: " + err.Error())
		} else {
			logger.Info("Inbound media store: MinIO bucket '" + bucket + "' @ " + endpoint)
			return client
		}
	}

	// Fallback: local filesystem.
	if dir := os.Getenv("MEDIA_UPLOAD_DIR"); dir != "" {
		baseURL := os.Getenv("MEDIA_UPLOAD_BASE_URL")
		if baseURL == "" {
			baseURL = "/uploads/media"
		}
		logger.Info("Inbound media store: local dir " + dir)
		return storageLib.NewLocalClient(dir, baseURL)
	}

	return nil
}

// statusEventType maps a provider status string to the NATS event type emitted
// on the outbound webhook. Unknown statuses return "" (no event).
func statusEventType(status string) string {
	switch status {
	case "sent":
		return nats.EventMessageSent
	case "delivered":
		return nats.EventMessageDelivered
	case "read":
		return nats.EventMessageRead
	case "failed":
		return nats.EventMessageFailed
	default:
		return ""
	}
}

// ListParams alias for database package
type ListParams = database.ListParams
