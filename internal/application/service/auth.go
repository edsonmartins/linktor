package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/config"
	"github.com/msgfy/linktor/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// Token types distinguish access tokens (used as API credentials) from refresh
// tokens (only accepted at the refresh endpoint). Without this, the two are
// interchangeable: a refresh token could be replayed as an API credential and an
// access token could be replayed at /auth/refresh.
const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

// TokenClaims represents JWT token claims
type TokenClaims struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	// Type distinguishes "access" from "refresh". Validated wherever a token is
	// consumed so the two cannot be swapped (JWT type confusion).
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

// LoginResult represents the result of a login operation
type LoginResult struct {
	User         *entity.User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// RefreshResult represents the result of a token refresh operation
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// AuthService handles authentication operations
type AuthService struct {
	userRepo repository.UserRepository
	config   *config.JWTConfig
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo repository.UserRepository, config *config.JWTConfig) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		config:   config,
	}
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, errors.New(errors.ErrCodeInvalidCredentials, "Invalid email or password")
	}

	// Check if user is active
	if user.Status != entity.UserStatusActive {
		return nil, errors.New(errors.ErrCodeInvalidCredentials, "Account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New(errors.ErrCodeInvalidCredentials, "Invalid email or password")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to generate access token")
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to generate refresh token")
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		// Log error but don't fail login
	}

	return &LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenTTL * 60),
	}, nil
}

// RefreshToken refreshes access token using refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	// Parse refresh token — must be a refresh token, never an access token.
	claims, err := s.parseToken(refreshToken, tokenTypeRefresh)
	if err != nil {
		return nil, errors.New(errors.ErrCodeTokenInvalid, "Invalid refresh token")
	}

	// Get user
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeTokenInvalid, "User not found")
	}

	// Check if user is still active
	if user.Status != entity.UserStatusActive {
		return nil, errors.New(errors.ErrCodeTokenInvalid, "Account is not active")
	}

	// Generate new tokens
	newAccessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to generate access token")
	}

	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to generate refresh token")
	}

	return &RefreshResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.config.AccessTokenTTL * 60),
	}, nil
}

// ValidateAccessToken validates an access token and returns claims. A refresh
// token presented here is rejected (JWT type confusion).
func (s *AuthService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	return s.parseToken(tokenString, tokenTypeAccess)
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New(errors.ErrCodeUserNotFound, "User not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New(errors.ErrCodeInvalidCredentials, "Current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "Failed to hash password")
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "Failed to update password")
	}

	return nil
}

// generateAccessToken generates a new access token
func (s *AuthService) generateAccessToken(user *entity.User) (string, error) {
	expiresAt := time.Now().Add(time.Duration(s.config.AccessTokenTTL) * time.Minute)

	claims := &TokenClaims{
		TenantID: user.TenantID,
		UserID:   user.ID,
		Email:    user.Email,
		Role:     string(user.Role),
		Type:     tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.Issuer,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Secret))
}

// generateRefreshToken generates a new refresh token
func (s *AuthService) generateRefreshToken(user *entity.User) (string, error) {
	expiresAt := time.Now().Add(time.Duration(s.config.RefreshTokenTTL) * time.Hour)

	claims := &TokenClaims{
		TenantID: user.TenantID,
		UserID:   user.ID,
		Email:    user.Email,
		Role:     string(user.Role),
		Type:     tokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.Issuer,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Secret))
}

// parseToken parses and validates a JWT token, asserting HMAC signing (to reject
// `alg` confusion such as "none" or an RS256 forgery) and that the token's `typ`
// claim matches expectedType (to reject access/refresh type confusion).
func (s *AuthService) parseToken(tokenString, expectedType string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(errors.ErrCodeTokenInvalid, "Invalid signing method")
		}
		return []byte(s.config.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New(errors.ErrCodeTokenInvalid, "Invalid token")
	}

	if claims.Type != expectedType {
		return nil, errors.New(errors.ErrCodeTokenInvalid, "Wrong token type")
	}

	return claims, nil
}

// HashPassword hashes a password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
