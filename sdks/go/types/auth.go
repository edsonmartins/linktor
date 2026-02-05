package types

import "time"

// UserRole type
type UserRole string

const (
	UserRoleAdmin      UserRole = "admin"
	UserRoleAgent      UserRole = "agent"
	UserRoleSupervisor UserRole = "supervisor"
	UserRoleViewer     UserRole = "viewer"
)

// User model
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      UserRole  `json:"role"`
	TenantID  string    `json:"tenantId"`
	AvatarURL string    `json:"avatarUrl,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LoginResponse from login
type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
	User         User   `json:"user"`
}

// RefreshTokenResponse from refresh
type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}
