package entity

import "time"

// AssignmentStrategy controls how new conversations are routed to agents.
type AssignmentStrategy string

const (
	// AssignmentManual leaves conversations unassigned for agents to pick up.
	AssignmentManual AssignmentStrategy = "manual"
	// AssignmentRoundRobin assigns to the agent who was assigned least recently.
	AssignmentRoundRobin AssignmentStrategy = "round_robin"
	// AssignmentLoadBalanced assigns to the agent with the fewest open conversations.
	AssignmentLoadBalanced AssignmentStrategy = "load_balanced"
)

// TenantSettings holds per-tenant operational configuration for assignment and
// SLA behavior.
type TenantSettings struct {
	TenantID                string                 `json:"tenant_id"`
	AssignmentStrategy      AssignmentStrategy     `json:"assignment_strategy"`
	AutoAssignEnabled       bool                   `json:"auto_assign_enabled"`
	SLAFirstResponseMinutes int                    `json:"sla_first_response_minutes"`
	SLAResolutionMinutes    int                    `json:"sla_resolution_minutes"`
	AutoCloseAfterMinutes   int                    `json:"auto_close_after_minutes"`
	BusinessHours           map[string]interface{} `json:"business_hours,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
}

// DefaultTenantSettings returns settings with safe defaults (manual assignment,
// no SLA, no auto-close).
func DefaultTenantSettings(tenantID string) *TenantSettings {
	now := time.Now()
	return &TenantSettings{
		TenantID:           tenantID,
		AssignmentStrategy: AssignmentManual,
		BusinessHours:      map[string]interface{}{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
