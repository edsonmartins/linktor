package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/pkg/errors"
)

// AuthCookieConfig controls how auth tokens are written as browser cookies.
type AuthCookieConfig struct {
	Secure        bool
	Domain        string
	SameSite      http.SameSite
	AccessMaxAge  int // seconds
	RefreshMaxAge int // seconds
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *service.AuthService
	userService *service.UserService
	cookie      AuthCookieConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, userService *service.UserService, cookie AuthCookieConfig) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		cookie:      cookie,
	}
}

// ParseSameSite maps a config string to http.SameSite (defaults to Lax).
func ParseSameSite(v string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
)

// setAuthCookies writes the HttpOnly access/refresh cookies. The tokens are
// still returned in the JSON body for non-browser clients (CLI/mobile).
func (h *AuthHandler) setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	h.writeCookie(c, accessTokenCookie, accessToken, h.cookie.AccessMaxAge)
	h.writeCookie(c, refreshTokenCookie, refreshToken, h.cookie.RefreshMaxAge)
}

// clearAuthCookies expires the auth cookies.
func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	h.writeCookie(c, accessTokenCookie, "", -1)
	h.writeCookie(c, refreshTokenCookie, "", -1)
}

func (h *AuthHandler) writeCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   h.cookie.Domain,
		MaxAge:   maxAge,
		Secure:   h.cookie.Secure,
		HttpOnly: true,
		SameSite: h.cookie.SameSite,
	})
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int64         `json:"expires_in"`
}

// RefreshRequest represents a token refresh request. The refresh token is
// optional in the body: browsers send it as an HttpOnly cookie instead.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents a token refresh response
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// ChangePasswordRequest represents a change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user with email and password and receive JWT tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} Response{data=LoginResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		RespondError(c, err)
		return
	}

	h.setAuthCookies(c, result.AccessToken, result.RefreshToken)

	RespondSuccess(c, LoginResponse{
		User:         toUserResponse(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Get a new access token using a valid refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest true "Refresh token"
// @Success      200 {object} Response{data=RefreshResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	// Body is optional; a missing/blank JSON body is fine when the refresh
	// token arrives as a cookie.
	_ = c.ShouldBindJSON(&req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookie, err := c.Cookie(refreshTokenCookie); err == nil {
			refreshToken = cookie
		}
	}
	if refreshToken == "" {
		RespondError(c, errors.Unauthorized("missing refresh token"))
		return
	}

	result, err := h.authService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		// A bad/expired refresh token should also drop the stale cookies.
		h.clearAuthCookies(c)
		RespondError(c, err)
		return
	}

	h.setAuthCookies(c, result.AccessToken, result.RefreshToken)

	RespondSuccess(c, RefreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

// Logout godoc
// @Summary      Logout user
// @Description  Clears the auth cookies for the current browser session
// @Tags         auth
// @Produce      json
// @Success      200 {object} Response{data=object{message=string}}
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearAuthCookies(c)
	RespondSuccess(c, gin.H{"message": "Logged out"})
}

// Me godoc
// @Summary      Get current user
// @Description  Returns the currently authenticated user's profile
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} Response{data=UserResponse}
// @Failure      401 {object} Response
// @Router       /me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		RespondUnauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, toUserResponse(user))
}

// UpdateMeRequest represents an update current user request
type UpdateMeRequest struct {
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

// UpdateMe godoc
// @Summary      Update current user
// @Description  Update the currently authenticated user's profile
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateMeRequest true "User update data"
// @Success      200 {object} Response{data=UserResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /me [put]
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		RespondUnauthorized(c, "User not authenticated")
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateUserInput{
		Name:      req.Name,
		AvatarURL: req.AvatarURL,
	}

	user, err := h.userService.Update(c.Request.Context(), userID, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, toUserResponse(user))
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Change the currently authenticated user's password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body ChangePasswordRequest true "Password change data"
// @Success      200 {object} Response{data=object{message=string}}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /me/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		RespondUnauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.IsAppError(err) {
			RespondError(c, err)
			return
		}
		RespondError(c, errors.Internal("Failed to change password"))
		return
	}

	RespondSuccess(c, gin.H{"message": "Password changed successfully"})
}
