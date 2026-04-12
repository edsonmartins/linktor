package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// APIKeyHandler handles tenant API key endpoints.
type APIKeyHandler struct {
	apiKeyService *service.APIKeyService
}

// NewAPIKeyHandler creates a new API key handler.
func NewAPIKeyHandler(apiKeyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{apiKeyService: apiKeyService}
}

// APIKeyResponse represents API key metadata. It never includes the key hash.
type APIKeyResponse struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	UserID     *string    `json:"user_id,omitempty"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateAPIKeyResponse includes the one-time raw key.
type CreateAPIKeyResponse struct {
	*APIKeyResponse
	Key string `json:"key"`
}

// CreateAPIKeyRequest represents a create API key request.
type CreateAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// List returns API keys for the current tenant.
func (h *APIKeyHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	apiKeys, err := h.apiKeyService.List(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, toAPIKeyResponses(apiKeys))
}

// Create generates a new API key and returns the raw key once.
func (h *APIKeyHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	result, err := h.apiKeyService.Create(c.Request.Context(), &service.CreateAPIKeyInput{
		TenantID:  tenantID,
		UserID:    c.GetString(middleware.UserIDKey),
		Name:      req.Name,
		Scopes:    req.Scopes,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, &CreateAPIKeyResponse{
		APIKeyResponse: toAPIKeyResponse(result.APIKey),
		Key:            result.Key,
	})
}

// Delete removes an API key for the current tenant.
func (h *APIKeyHandler) Delete(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "API key ID is required", nil)
		return
	}

	if err := h.apiKeyService.Delete(c.Request.Context(), tenantID, id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

func toAPIKeyResponse(apiKey *entity.APIKey) *APIKeyResponse {
	return &APIKeyResponse{
		ID:         apiKey.ID,
		TenantID:   apiKey.TenantID,
		UserID:     apiKey.UserID,
		Name:       apiKey.Name,
		KeyPrefix:  apiKey.KeyPrefix,
		Scopes:     apiKey.Scopes,
		LastUsedAt: apiKey.LastUsedAt,
		ExpiresAt:  apiKey.ExpiresAt,
		CreatedAt:  apiKey.CreatedAt,
	}
}

func toAPIKeyResponses(apiKeys []*entity.APIKey) []*APIKeyResponse {
	responses := make([]*APIKeyResponse, len(apiKeys))
	for i, apiKey := range apiKeys {
		responses[i] = toAPIKeyResponse(apiKey)
	}
	return responses
}
