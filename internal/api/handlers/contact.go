package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
)

// ContactHandler handles contact endpoints
type ContactHandler struct {
	contactService *service.ContactService
}

// NewContactHandler creates a new contact handler
func NewContactHandler(contactService *service.ContactService) *ContactHandler {
	return &ContactHandler{
		contactService: contactService,
	}
}

// CreateContactRequest represents a create contact request
type CreateContactRequest struct {
	Name         string            `json:"name"`
	Email        string            `json:"email"`
	Phone        string            `json:"phone"`
	AvatarURL    string            `json:"avatar_url"`
	CustomFields map[string]string `json:"custom_fields"`
	Tags         []string          `json:"tags"`
}

// List godoc
// @Summary      List contacts
// @Description  Returns all contacts for the current tenant with pagination
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Param        search query string false "Search by name, email or phone"
// @Success      200 {object} Response{data=[]entity.Contact,meta=MetaResponse}
// @Failure      401 {object} Response
// @Router       /contacts [get]
func (h *ContactHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	contacts, total, err := h.contactService.List(c.Request.Context(), tenantID, nil)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondWithMeta(c, contacts, &MetaResponse{
		Page:       1,
		PageSize:   20,
		TotalItems: total,
	})
}

// Create godoc
// @Summary      Create contact
// @Description  Create a new contact for the current tenant
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateContactRequest true "Contact data"
// @Success      201 {object} Response{data=entity.Contact}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /contacts [post]
func (h *ContactHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateContactInput{
		TenantID:     tenantID,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		AvatarURL:    req.AvatarURL,
		CustomFields: req.CustomFields,
		Tags:         req.Tags,
	}

	contact, err := h.contactService.Create(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, contact)
}

// Get godoc
// @Summary      Get contact
// @Description  Returns a contact by ID
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Contact ID"
// @Success      200 {object} Response{data=entity.Contact}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /contacts/{id} [get]
func (h *ContactHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Contact ID is required", nil)
		return
	}

	contact, err := h.contactService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, contact)
}

// Update godoc
// @Summary      Update contact
// @Description  Update a contact's information
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Contact ID"
// @Param        request body CreateContactRequest true "Contact update data"
// @Success      200 {object} Response{data=entity.Contact}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /contacts/{id} [put]
func (h *ContactHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Contact ID is required", nil)
		return
	}

	var req CreateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateContactInput{
		Name:         &req.Name,
		Email:        &req.Email,
		Phone:        &req.Phone,
		AvatarURL:    &req.AvatarURL,
		CustomFields: req.CustomFields,
		Tags:         req.Tags,
	}

	contact, err := h.contactService.Update(c.Request.Context(), id, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, contact)
}

// Delete godoc
// @Summary      Delete contact
// @Description  Delete a contact by ID
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Contact ID"
// @Success      204 "No Content"
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /contacts/{id} [delete]
func (h *ContactHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Contact ID is required", nil)
		return
	}

	if err := h.contactService.Delete(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

// AddIdentityRequest represents an add identity request
type AddIdentityRequest struct {
	ChannelType string            `json:"channel_type" binding:"required"`
	Identifier  string            `json:"identifier" binding:"required"`
	Metadata    map[string]string `json:"metadata"`
}

// AddIdentity godoc
// @Summary      Add identity
// @Description  Add a channel identity to a contact
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Contact ID"
// @Param        request body AddIdentityRequest true "Identity data"
// @Success      200 {object} Response{data=entity.Contact}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /contacts/{id}/identities [post]
func (h *ContactHandler) AddIdentity(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Contact ID is required", nil)
		return
	}

	var req AddIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	contact, err := h.contactService.AddIdentity(c.Request.Context(), id, req.ChannelType, req.Identifier, req.Metadata)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, contact)
}

// RemoveIdentity godoc
// @Summary      Remove identity
// @Description  Remove a channel identity from a contact
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Contact ID"
// @Param        identityId path string true "Identity ID"
// @Success      200 {object} Response{data=entity.Contact}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /contacts/{id}/identities/{identityId} [delete]
func (h *ContactHandler) RemoveIdentity(c *gin.Context) {
	id := c.Param("id")
	identityID := c.Param("identityId")

	if id == "" || identityID == "" {
		RespondValidationError(c, "Contact ID and Identity ID are required", nil)
		return
	}

	contact, err := h.contactService.RemoveIdentity(c.Request.Context(), id, identityID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, contact)
}
