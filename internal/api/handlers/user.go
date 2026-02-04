package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// UserHandler handles user endpoints
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Role        string     `json:"role"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// toUserResponse converts entity to response
func toUserResponse(user *entity.User) *UserResponse {
	return &UserResponse{
		ID:          user.ID,
		TenantID:    user.TenantID,
		Email:       user.Email,
		Name:        user.Name,
		Role:        string(user.Role),
		AvatarURL:   user.AvatarURL,
		Status:      string(user.Status),
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// toUserResponses converts entities to responses
func toUserResponses(users []*entity.User) []*UserResponse {
	responses := make([]*UserResponse, len(users))
	for i, user := range users {
		responses[i] = toUserResponse(user)
	}
	return responses
}

// CreateUserRequest represents a create user request
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=agent supervisor admin"`
}

// UpdateUserRequest represents an update user request
type UpdateUserRequest struct {
	Name      *string `json:"name"`
	Role      *string `json:"role"`
	AvatarURL *string `json:"avatar_url"`
	Status    *string `json:"status"`
}

// List returns all users for the tenant
func (h *UserHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	params := repository.NewListParams()
	// TODO: Parse query params for pagination

	users, total, err := h.userService.List(c.Request.Context(), tenantID, params)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondWithMeta(c, toUserResponses(users), &MetaResponse{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: total,
		TotalPages: int((total + int64(params.PageSize) - 1) / int64(params.PageSize)),
		HasNext:    int64(params.Page*params.PageSize) < total,
		HasPrev:    params.Page > 1,
	})
}

// Create creates a new user
func (h *UserHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateUserInput{
		TenantID: tenantID,
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     entity.UserRole(req.Role),
	}

	user, err := h.userService.Create(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, toUserResponse(user))
}

// Get returns a user by ID
func (h *UserHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "User ID is required", nil)
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	// Verify tenant access
	tenantID := middleware.MustGetTenantID(c)
	if tenantID != "" && user.TenantID != tenantID {
		RespondNotFound(c, "User")
		return
	}

	RespondSuccess(c, toUserResponse(user))
}

// Update updates a user
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "User ID is required", nil)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateUserInput{
		Name:      req.Name,
		AvatarURL: req.AvatarURL,
	}

	if req.Role != nil {
		role := entity.UserRole(*req.Role)
		input.Role = &role
	}

	if req.Status != nil {
		status := entity.UserStatus(*req.Status)
		input.Status = &status
	}

	user, err := h.userService.Update(c.Request.Context(), id, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, toUserResponse(user))
}

// Delete deletes a user
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "User ID is required", nil)
		return
	}

	// Prevent self-deletion
	currentUserID := middleware.GetUserID(c)
	if id == currentUserID {
		RespondValidationError(c, "Cannot delete your own account", nil)
		return
	}

	if err := h.userService.Delete(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}
