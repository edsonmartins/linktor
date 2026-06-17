package entity

import "time"

// Permission is a (resource, action) pair. A role grants a set of these.
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// Resource identifiers used across the RBAC system. These name coarse feature
// areas rather than individual endpoints.
const (
	ResourceUsers         = "users"
	ResourceRoles         = "roles"
	ResourceChannels      = "channels"
	ResourceContacts      = "contacts"
	ResourceConversations = "conversations"
	ResourceMessages      = "messages"
	ResourceBots          = "bots"
	ResourceFlows         = "flows"
	ResourceTemplates     = "templates"
	ResourceCampaigns     = "campaigns"
	ResourceCanned        = "canned_responses"
	ResourceAnalytics     = "analytics"
	ResourceKnowledge     = "knowledge"
	ResourceAudit         = "audit"
	ResourceSettings      = "settings"
)

// Action identifiers used across the RBAC system.
const (
	ActionRead   = "read"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionManage = "manage" // implies all actions on the resource
)

// Built-in system role names. They mirror the legacy UserRole strings so that a
// user's role string resolves to a seeded role of the same name.
const (
	SystemRoleOwner      = "owner"
	SystemRoleAdmin      = "admin"
	SystemRoleSupervisor = "supervisor"
	SystemRoleAgent      = "agent"
)

// Role is a named set of permissions scoped to a tenant.
type Role struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	IsSystem    bool         `json:"is_system"`
	IsDefault   bool         `json:"is_default"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// NewRole creates a custom (non-system) role.
func NewRole(tenantID, name, description string) *Role {
	now := time.Now()
	return &Role{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Permissions: []Permission{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Has reports whether the role grants (resource, action). A "manage" grant on
// the resource satisfies any action; a "manage" grant on resource "*" (owner)
// satisfies everything.
func (r *Role) Has(resource, action string) bool {
	for _, p := range r.Permissions {
		if p.Resource == "*" && p.Action == ActionManage {
			return true
		}
		if p.Resource != resource {
			continue
		}
		if p.Action == action || p.Action == ActionManage {
			return true
		}
	}
	return false
}

// SystemRolePermissions returns the default permission set for a built-in role
// name. Used both to seed roles per tenant and as a fallback when a tenant has
// not been seeded yet. An unknown name yields no permissions.
func SystemRolePermissions(name string) []Permission {
	switch name {
	case SystemRoleOwner:
		return []Permission{{Resource: "*", Action: ActionManage}}
	case SystemRoleAdmin:
		return manageAll(
			ResourceUsers, ResourceRoles, ResourceChannels, ResourceContacts,
			ResourceConversations, ResourceMessages, ResourceBots, ResourceFlows,
			ResourceTemplates, ResourceCampaigns, ResourceCanned, ResourceAnalytics,
			ResourceKnowledge, ResourceAudit, ResourceSettings,
		)
	case SystemRoleSupervisor:
		perms := manageAll(
			ResourceChannels, ResourceContacts, ResourceConversations, ResourceMessages,
			ResourceTemplates, ResourceCampaigns, ResourceCanned, ResourceBots, ResourceFlows,
		)
		perms = append(perms,
			Permission{Resource: ResourceAnalytics, Action: ActionRead},
			Permission{Resource: ResourceUsers, Action: ActionRead},
		)
		return perms
	case SystemRoleAgent:
		return []Permission{
			{Resource: ResourceConversations, Action: ActionRead},
			{Resource: ResourceConversations, Action: ActionUpdate},
			{Resource: ResourceMessages, Action: ActionRead},
			{Resource: ResourceMessages, Action: ActionCreate},
			{Resource: ResourceContacts, Action: ActionRead},
			{Resource: ResourceContacts, Action: ActionUpdate},
			{Resource: ResourceCanned, Action: ActionRead},
		}
	default:
		return nil
	}
}

// SystemRoleNames lists the built-in roles in descending privilege order.
func SystemRoleNames() []string {
	return []string{SystemRoleOwner, SystemRoleAdmin, SystemRoleSupervisor, SystemRoleAgent}
}

func manageAll(resources ...string) []Permission {
	perms := make([]Permission, 0, len(resources))
	for _, res := range resources {
		perms = append(perms, Permission{Resource: res, Action: ActionManage})
	}
	return perms
}
