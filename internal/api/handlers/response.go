package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/pkg/errors"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
	Meta    *MetaResponse  `json:"meta,omitempty"`
}

// ErrorResponse represents an error in the response
type ErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// MetaResponse represents pagination metadata
type MetaResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	TotalItems int64 `json:"total_items"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_previous"`
}

// RespondSuccess sends a success response
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// RespondCreated sends a created response
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// RespondWithMeta sends a success response with pagination metadata
func RespondWithMeta(c *gin.Context, data interface{}, meta *MetaResponse) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// RespondError sends an error response
func RespondError(c *gin.Context, err error) {
	if appErr := errors.GetAppError(err); appErr != nil {
		c.JSON(appErr.StatusCode, Response{
			Success: false,
			Error: &ErrorResponse{
				Code:    string(appErr.Code),
				Message: appErr.Message,
				Details: appErr.Details,
			},
		})
		return
	}

	// Generic internal error
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: &ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
		},
	})
}

// RespondValidationError sends a validation error response
func RespondValidationError(c *gin.Context, message string, details map[string]string) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: &ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: message,
			Details: details,
		},
	})
}

// RespondNotFound sends a not found response
func RespondNotFound(c *gin.Context, resource string) {
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error: &ErrorResponse{
			Code:    "NOT_FOUND",
			Message: resource + " not found",
		},
	})
}

// RespondUnauthorized sends an unauthorized response
func RespondUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error: &ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	})
}

// RespondForbidden sends a forbidden response
func RespondForbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Success: false,
		Error: &ErrorResponse{
			Code:    "FORBIDDEN",
			Message: message,
		},
	})
}

// RespondNoContent sends a no content response
func RespondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
