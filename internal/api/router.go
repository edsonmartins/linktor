package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/handlers"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/infrastructure/config"
)

// Router holds all dependencies for the HTTP router
type Router struct {
	config         *config.Config
	authMiddleware *middleware.AuthMiddleware
	rateLimiter    *middleware.RateLimiter
	// Handlers
	healthHandler        *handlers.HealthHandler
	authHandler          *handlers.AuthHandler
	tenantHandler        *handlers.TenantHandler
	userHandler          *handlers.UserHandler
	channelHandler       *handlers.ChannelHandler
	contactHandler       *handlers.ContactHandler
	conversationHandler  *handlers.ConversationHandler
	messageHandler       *handlers.MessageHandler
	observabilityHandler *handlers.ObservabilityHandler
}

// NewRouter creates a new router with all dependencies
func NewRouter(
	cfg *config.Config,
	authMiddleware *middleware.AuthMiddleware,
	rateLimiter *middleware.RateLimiter,
	healthHandler *handlers.HealthHandler,
	authHandler *handlers.AuthHandler,
	tenantHandler *handlers.TenantHandler,
	userHandler *handlers.UserHandler,
	channelHandler *handlers.ChannelHandler,
	contactHandler *handlers.ContactHandler,
	conversationHandler *handlers.ConversationHandler,
	messageHandler *handlers.MessageHandler,
	observabilityHandler *handlers.ObservabilityHandler,
) *Router {
	return &Router{
		config:               cfg,
		authMiddleware:       authMiddleware,
		rateLimiter:          rateLimiter,
		healthHandler:        healthHandler,
		authHandler:          authHandler,
		tenantHandler:        tenantHandler,
		userHandler:          userHandler,
		channelHandler:       channelHandler,
		contactHandler:       contactHandler,
		conversationHandler:  conversationHandler,
		messageHandler:       messageHandler,
		observabilityHandler: observabilityHandler,
	}
}

// Setup configures all routes and returns the gin engine
func (r *Router) Setup() *gin.Engine {
	// Set gin mode
	if r.config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Global middleware
	engine.Use(gin.Recovery())
	engine.Use(middleware.Logger())
	engine.Use(middleware.CORS())
	engine.Use(middleware.RequestID())

	// Health check (no auth required)
	engine.GET("/health", r.healthHandler.Health)
	engine.GET("/ready", r.healthHandler.Ready)

	// API v1 routes
	v1 := engine.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/refresh", r.authHandler.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(r.authMiddleware.Authenticate())
		protected.Use(r.rateLimiter.Limit())
		{
			// Current user
			protected.GET("/me", r.authHandler.Me)
			protected.PUT("/me", r.authHandler.UpdateMe)
			protected.PUT("/me/password", r.authHandler.ChangePassword)

			// Tenant management (admin only)
			tenant := protected.Group("/tenant")
			tenant.Use(r.authMiddleware.RequireRole("admin", "owner"))
			{
				tenant.GET("", r.tenantHandler.Get)
				tenant.PUT("", r.tenantHandler.Update)
				tenant.GET("/usage", r.tenantHandler.GetUsage)
			}

			// User management (admin only)
			users := protected.Group("/users")
			users.Use(r.authMiddleware.RequireRole("admin", "owner"))
			{
				users.GET("", r.userHandler.List)
				users.POST("", r.userHandler.Create)
				users.GET("/:id", r.userHandler.Get)
				users.PUT("/:id", r.userHandler.Update)
				users.DELETE("/:id", r.userHandler.Delete)
			}

			// Channel management
			channels := protected.Group("/channels")
			{
				channels.GET("", r.channelHandler.List)
				channels.POST("", r.channelHandler.Create)
				channels.GET("/:id", r.channelHandler.Get)
				channels.PUT("/:id", r.channelHandler.Update)
				channels.DELETE("/:id", r.channelHandler.Delete)
				channels.POST("/:id/connect", r.channelHandler.Connect)
				channels.POST("/:id/pair", r.channelHandler.RequestPairCode)
				channels.POST("/:id/disconnect", r.channelHandler.Disconnect)
				channels.PUT("/:id/status", r.channelHandler.UpdateStatus)
			}

			// Contact management
			contacts := protected.Group("/contacts")
			{
				contacts.GET("", r.contactHandler.List)
				contacts.POST("", r.contactHandler.Create)
				contacts.GET("/:id", r.contactHandler.Get)
				contacts.PUT("/:id", r.contactHandler.Update)
				contacts.DELETE("/:id", r.contactHandler.Delete)
				contacts.POST("/:id/identities", r.contactHandler.AddIdentity)
				contacts.DELETE("/:id/identities/:identityId", r.contactHandler.RemoveIdentity)
			}

			// Conversation management
			conversations := protected.Group("/conversations")
			{
				conversations.GET("", r.conversationHandler.List)
				conversations.POST("", r.conversationHandler.Create)
				conversations.GET("/:id", r.conversationHandler.Get)
				conversations.PUT("/:id", r.conversationHandler.Update)
				conversations.POST("/:id/assign", r.conversationHandler.Assign)
				conversations.POST("/:id/resolve", r.conversationHandler.Resolve)
				conversations.POST("/:id/reopen", r.conversationHandler.Reopen)

				// Messages within conversation
				conversations.GET("/:id/messages", r.messageHandler.List)
				conversations.POST("/:id/messages", r.messageHandler.Send)
				conversations.POST("/:id/messages/:messageId/reactions", r.messageHandler.SendReaction)
			}

			// Message management
			messages := protected.Group("/messages")
			{
				messages.GET("/:id", r.messageHandler.Get)
			}

			// Observability management (admin only)
			observability := protected.Group("/observability")
			observability.Use(r.authMiddleware.RequireRole("admin", "owner"))
			{
				observability.GET("/logs", r.observabilityHandler.GetLogs)
				observability.POST("/logs/cleanup", r.observabilityHandler.CleanupLogs)
				observability.GET("/queue", r.observabilityHandler.GetQueueStats)
				observability.GET("/queue/:stream", r.observabilityHandler.GetStreamInfo)
				observability.POST("/queue/reset-consumer", r.observabilityHandler.ResetConsumer)
				observability.GET("/stats", r.observabilityHandler.GetSystemStats)
			}
		}

		// Webhook endpoints (for external channels)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/whatsapp/:channelId", r.channelHandler.WhatsAppWebhook)
			webhooks.GET("/whatsapp/:channelId", r.channelHandler.WhatsAppVerify)
			webhooks.POST("/telegram/:channelId", r.channelHandler.TelegramWebhook)
			webhooks.POST("/twilio/:channelId", r.channelHandler.TwilioWebhook)
		}
	}

	// 404 handler
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "NOT_FOUND",
			"message": "Resource not found",
		})
	})

	return engine
}
