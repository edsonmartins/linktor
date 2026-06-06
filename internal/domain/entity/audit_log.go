package entity

import "time"

// AuditLog records a security- or compliance-relevant mutation performed by a
// user (or the system) within a tenant. Actor identity is denormalized so the
// trail stays meaningful even after the user is deleted.
type AuditLog struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	ActorID      string                 `json:"actor_id,omitempty"`
	ActorEmail   string                 `json:"actor_email,omitempty"`
	ActorName    string                 `json:"actor_name,omitempty"`
	Action       string                 `json:"action"`                  // e.g. "channel.create", "role.update"
	ResourceType string                 `json:"resource_type,omitempty"` // e.g. "channel", "campaign"
	ResourceID   string                 `json:"resource_id,omitempty"`
	Changes      map[string]interface{} `json:"changes,omitempty"` // arbitrary before/after or detail payload
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// NewAuditLog creates an audit entry stamped with the current time.
func NewAuditLog(tenantID, action string) *AuditLog {
	return &AuditLog{
		TenantID:  tenantID,
		Action:    action,
		Changes:   make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}
