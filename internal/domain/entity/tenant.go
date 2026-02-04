package entity

import (
	"time"
)

// Plan represents a subscription plan
type Plan string

const (
	PlanFree         Plan = "free"
	PlanStarter      Plan = "starter"
	PlanProfessional Plan = "professional"
	PlanEnterprise   Plan = "enterprise"
)

// TenantStatus represents a tenant's status
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusCancelled TenantStatus = "cancelled"
)

// TenantLimits represents tenant usage limits
type TenantLimits struct {
	MaxUsers            int   `json:"max_users"`
	MaxChannels         int   `json:"max_channels"`
	MaxContacts         int   `json:"max_contacts"`
	MaxMessagesPerMonth int64 `json:"max_messages_per_month"`
}

// Tenant represents an organization/company
type Tenant struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Plan      Plan              `json:"plan"`
	Status    TenantStatus      `json:"status"`
	Settings  map[string]string `json:"settings"`
	Limits    *TenantLimits     `json:"limits"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// NewTenant creates a new tenant
func NewTenant(name, slug string, plan Plan) *Tenant {
	now := time.Now()
	return &Tenant{
		Name:     name,
		Slug:     slug,
		Plan:     plan,
		Status:   TenantStatusActive,
		Settings: make(map[string]string),
		Limits:   GetPlanLimits(plan),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetPlanLimits returns the limits for a plan
func GetPlanLimits(plan Plan) *TenantLimits {
	switch plan {
	case PlanFree:
		return &TenantLimits{
			MaxUsers:            5,
			MaxChannels:         3,
			MaxContacts:         1000,
			MaxMessagesPerMonth: 10000,
		}
	case PlanStarter:
		return &TenantLimits{
			MaxUsers:            15,
			MaxChannels:         10,
			MaxContacts:         10000,
			MaxMessagesPerMonth: 50000,
		}
	case PlanProfessional:
		return &TenantLimits{
			MaxUsers:            50,
			MaxChannels:         25,
			MaxContacts:         100000,
			MaxMessagesPerMonth: 250000,
		}
	case PlanEnterprise:
		return &TenantLimits{
			MaxUsers:            -1, // Unlimited
			MaxChannels:         -1,
			MaxContacts:         -1,
			MaxMessagesPerMonth: -1,
		}
	default:
		return GetPlanLimits(PlanFree)
	}
}

// IsActive returns true if tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}
