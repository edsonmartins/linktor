package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// KnowledgeHandler handles knowledge base endpoints
type KnowledgeHandler struct {
	knowledgeService *service.KnowledgeService
}

// NewKnowledgeHandler creates a new knowledge handler
func NewKnowledgeHandler(knowledgeService *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{
		knowledgeService: knowledgeService,
	}
}

// CreateKnowledgeBaseRequest represents a create knowledge base request
type CreateKnowledgeBaseRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Type        string                 `json:"type" binding:"required"` // faq, documents, website
	Config      *entity.KnowledgeConfig `json:"config"`
}

// UpdateKnowledgeBaseRequest represents an update knowledge base request
type UpdateKnowledgeBaseRequest struct {
	Name        *string                 `json:"name"`
	Description *string                 `json:"description"`
	Config      *entity.KnowledgeConfig `json:"config"`
}

// AddItemRequest represents an add item request
type AddItemRequest struct {
	Question string            `json:"question" binding:"required"`
	Answer   string            `json:"answer" binding:"required"`
	Keywords []string          `json:"keywords"`
	Source   string            `json:"source"`
	Metadata map[string]string `json:"metadata"`
}

// UpdateItemRequest represents an update item request
type UpdateItemRequest struct {
	Question *string           `json:"question"`
	Answer   *string           `json:"answer"`
	Keywords []string          `json:"keywords"`
	Source   *string           `json:"source"`
	Metadata map[string]string `json:"metadata"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

// ListKnowledgeBases returns all knowledge bases for the tenant
func (h *KnowledgeHandler) ListKnowledgeBases(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := &repository.ListParams{
		Page:     page,
		PageSize: pageSize,
		SortBy:   c.DefaultQuery("sort_by", "created_at"),
		SortDir:  c.DefaultQuery("sort_dir", "desc"),
	}

	kbs, total, err := h.knowledgeService.ListKnowledgeBases(c.Request.Context(), tenantID, params)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondPaginated(c, kbs, total, params.Page, params.PageSize)
}

// CreateKnowledgeBase creates a new knowledge base
func (h *KnowledgeHandler) CreateKnowledgeBase(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateKnowledgeBaseInput{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Type:        entity.KnowledgeType(req.Type),
		Config:      req.Config,
	}

	kb, err := h.knowledgeService.CreateKnowledgeBase(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, kb)
}

// GetKnowledgeBase returns a knowledge base by ID
func (h *KnowledgeHandler) GetKnowledgeBase(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	kb, err := h.knowledgeService.GetKnowledgeBase(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, kb)
}

// UpdateKnowledgeBase updates a knowledge base
func (h *KnowledgeHandler) UpdateKnowledgeBase(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	var req UpdateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateKnowledgeBaseInput{
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
	}

	kb, err := h.knowledgeService.UpdateKnowledgeBase(c.Request.Context(), id, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, kb)
}

// DeleteKnowledgeBase deletes a knowledge base
func (h *KnowledgeHandler) DeleteKnowledgeBase(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	if err := h.knowledgeService.DeleteKnowledgeBase(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

// ListItems returns all items in a knowledge base
func (h *KnowledgeHandler) ListItems(c *gin.Context) {
	kbID := c.Param("id")
	if kbID == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := &repository.ListParams{
		Page:     page,
		PageSize: pageSize,
		SortBy:   c.DefaultQuery("sort_by", "created_at"),
		SortDir:  c.DefaultQuery("sort_dir", "desc"),
	}

	items, total, err := h.knowledgeService.ListItems(c.Request.Context(), kbID, params)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondPaginated(c, items, total, params.Page, params.PageSize)
}

// AddItem adds an item to a knowledge base
func (h *KnowledgeHandler) AddItem(c *gin.Context) {
	kbID := c.Param("id")
	if kbID == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.AddItemInput{
		KnowledgeBaseID: kbID,
		Question:        req.Question,
		Answer:          req.Answer,
		Keywords:        req.Keywords,
		Source:          req.Source,
		Metadata:        req.Metadata,
	}

	item, err := h.knowledgeService.AddItem(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, item)
}

// GetItem returns an item by ID
func (h *KnowledgeHandler) GetItem(c *gin.Context) {
	itemID := c.Param("itemId")
	if itemID == "" {
		RespondValidationError(c, "Item ID is required", nil)
		return
	}

	item, err := h.knowledgeService.GetItem(c.Request.Context(), itemID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, item)
}

// UpdateItem updates an item
func (h *KnowledgeHandler) UpdateItem(c *gin.Context) {
	itemID := c.Param("itemId")
	if itemID == "" {
		RespondValidationError(c, "Item ID is required", nil)
		return
	}

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateItemInput{
		Question: req.Question,
		Answer:   req.Answer,
		Keywords: req.Keywords,
		Source:   req.Source,
		Metadata: req.Metadata,
	}

	item, err := h.knowledgeService.UpdateItem(c.Request.Context(), itemID, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, item)
}

// DeleteItem deletes an item
func (h *KnowledgeHandler) DeleteItem(c *gin.Context) {
	itemID := c.Param("itemId")
	if itemID == "" {
		RespondValidationError(c, "Item ID is required", nil)
		return
	}

	if err := h.knowledgeService.DeleteItem(c.Request.Context(), itemID); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

// Search performs semantic search on a knowledge base
func (h *KnowledgeHandler) Search(c *gin.Context) {
	kbID := c.Param("id")
	if kbID == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	results, err := h.knowledgeService.Search(c.Request.Context(), kbID, req.Query, limit)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"query":   req.Query,
		"count":   len(results),
	})
}

// RegenerateEmbeddings regenerates embeddings for all items in a knowledge base
func (h *KnowledgeHandler) RegenerateEmbeddings(c *gin.Context) {
	kbID := c.Param("id")
	if kbID == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	if err := h.knowledgeService.RegenerateEmbeddings(c.Request.Context(), kbID); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Embeddings regeneration started"})
}

// BulkAddItems adds multiple items to a knowledge base
func (h *KnowledgeHandler) BulkAddItems(c *gin.Context) {
	kbID := c.Param("id")
	if kbID == "" {
		RespondValidationError(c, "Knowledge base ID is required", nil)
		return
	}

	var req struct {
		Items []AddItemRequest `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	if len(req.Items) > 100 {
		RespondValidationError(c, "Maximum 100 items per request", nil)
		return
	}

	results := make([]*entity.KnowledgeItem, 0, len(req.Items))
	errors := make([]string, 0)

	for i, itemReq := range req.Items {
		input := &service.AddItemInput{
			KnowledgeBaseID: kbID,
			Question:        itemReq.Question,
			Answer:          itemReq.Answer,
			Keywords:        itemReq.Keywords,
			Source:          itemReq.Source,
			Metadata:        itemReq.Metadata,
		}

		item, err := h.knowledgeService.AddItem(c.Request.Context(), input)
		if err != nil {
			errors = append(errors, "Item "+strconv.Itoa(i)+": "+err.Error())
		} else {
			results = append(results, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"created": len(results),
		"items":   results,
		"errors":  errors,
	})
}
