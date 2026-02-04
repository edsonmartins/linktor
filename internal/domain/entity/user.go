package entity

import (
	"time"
)

// UserRole represents a user's role
type UserRole string

const (
	UserRoleAgent      UserRole = "agent"
	UserRoleSupervisor UserRole = "supervisor"
	UserRoleAdmin      UserRole = "admin"
	UserRoleOwner      UserRole = "owner"
)

// UserStatus represents a user's status
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

// User represents a system user
type User struct {
	ID           string      `json:"id"`
	TenantID     string      `json:"tenant_id"`
	Email        string      `json:"email"`
	PasswordHash string      `json:"-"`
	Name         string      `json:"name"`
	Role         UserRole    `json:"role"`
	AvatarURL    *string     `json:"avatar_url,omitempty"`
	Status       UserStatus  `json:"status"`
	LastLoginAt  *time.Time  `json:"last_login_at,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// NewUser creates a new user
func NewUser(tenantID, email, passwordHash, name string, role UserRole) *User {
	now := time.Now()
	return &User{
		TenantID:     tenantID,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		Role:         role,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsAdmin returns true if user is admin or owner
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleOwner
}

// CanManageUsers returns true if user can manage other users
func (u *User) CanManageUsers() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleOwner
}

// CanManageChannels returns true if user can manage channels
func (u *User) CanManageChannels() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleOwner || u.Role == UserRoleSupervisor
}
