package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents an application error code
type ErrorCode string

const (
	// General errors
	ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeConflict     ErrorCode = "CONFLICT"
	ErrCodeBadRequest   ErrorCode = "BAD_REQUEST"

	// Auth errors
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid       ErrorCode = "TOKEN_INVALID"

	// Resource errors
	ErrCodeTenantNotFound       ErrorCode = "TENANT_NOT_FOUND"
	ErrCodeUserNotFound         ErrorCode = "USER_NOT_FOUND"
	ErrCodeChannelNotFound      ErrorCode = "CHANNEL_NOT_FOUND"
	ErrCodeContactNotFound      ErrorCode = "CONTACT_NOT_FOUND"
	ErrCodeConversationNotFound ErrorCode = "CONVERSATION_NOT_FOUND"
	ErrCodeMessageNotFound      ErrorCode = "MESSAGE_NOT_FOUND"

	// Channel errors
	ErrCodeChannelDisconnected ErrorCode = "CHANNEL_DISCONNECTED"
	ErrCodeChannelError        ErrorCode = "CHANNEL_ERROR"

	// Rate limiting
	ErrCodeRateLimited ErrorCode = "RATE_LIMITED"

	// Quota errors
	ErrCodeQuotaExceeded ErrorCode = "QUOTA_EXCEEDED"

	// Timeout errors
	ErrCodeTimeout ErrorCode = "TIMEOUT"
)

// AppError represents an application error
type AppError struct {
	Code       ErrorCode         `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	StatusCode int               `json:"-"`
	Err        error             `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details map[string]string) *AppError {
	e.Details = details
	return e
}

// WithError wraps an underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: codeToStatus(code),
	}
}

// Wrap wraps an error with an AppError
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: codeToStatus(code),
		Err:        err,
	}
}

// codeToStatus maps error codes to HTTP status codes
func codeToStatus(code ErrorCode) int {
	switch code {
	case ErrCodeInternal:
		return http.StatusInternalServerError
	case ErrCodeValidation, ErrCodeBadRequest:
		return http.StatusBadRequest
	case ErrCodeNotFound, ErrCodeTenantNotFound, ErrCodeUserNotFound,
		ErrCodeChannelNotFound, ErrCodeContactNotFound,
		ErrCodeConversationNotFound, ErrCodeMessageNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized, ErrCodeInvalidCredentials,
		ErrCodeTokenExpired, ErrCodeTokenInvalid:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeQuotaExceeded:
		return http.StatusPaymentRequired
	default:
		return http.StatusInternalServerError
	}
}

// Common error constructors

// Internal creates an internal server error
func Internal(message string) *AppError {
	return New(ErrCodeInternal, message)
}

// NotFound creates a not found error
func NotFound(resource string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

// Validation creates a validation error
func Validation(message string) *AppError {
	return New(ErrCodeValidation, message)
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *AppError {
	return New(ErrCodeUnauthorized, message)
}

// Forbidden creates a forbidden error
func Forbidden(message string) *AppError {
	return New(ErrCodeForbidden, message)
}

// Conflict creates a conflict error
func Conflict(message string) *AppError {
	return New(ErrCodeConflict, message)
}

// RateLimited creates a rate limited error
func RateLimited(message string) *AppError {
	return New(ErrCodeRateLimited, message)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts AppError from an error
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Code {
		case ErrCodeNotFound, ErrCodeTenantNotFound, ErrCodeUserNotFound,
			ErrCodeChannelNotFound, ErrCodeContactNotFound,
			ErrCodeConversationNotFound, ErrCodeMessageNotFound:
			return true
		}
	}
	return false
}

// IsUnauthorized checks if an error is an unauthorized error
func IsUnauthorized(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Code {
		case ErrCodeUnauthorized, ErrCodeInvalidCredentials,
			ErrCodeTokenExpired, ErrCodeTokenInvalid:
			return true
		}
	}
	return false
}

// IsValidation checks if an error is a validation error
func IsValidation(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrCodeValidation || appErr.Code == ErrCodeBadRequest
	}
	return false
}
