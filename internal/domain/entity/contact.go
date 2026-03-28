package entity

import (
	"time"
)

// ContactIdentity represents a contact's identity on a specific channel
type ContactIdentity struct {
	ID          string            `json:"id"`
	ContactID   string            `json:"contact_id"`
	ChannelType string            `json:"channel_type"`
	Identifier  string            `json:"identifier"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Contact represents a contact/customer
type Contact struct {
	ID           string             `json:"id"`
	TenantID     string             `json:"tenant_id"`
	Name         string             `json:"name,omitempty"`
	Email        string             `json:"email,omitempty"`
	Phone        string             `json:"phone,omitempty"`
	AvatarURL    string             `json:"avatar_url,omitempty"`
	CustomFields map[string]string  `json:"custom_fields,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	Identities   []*ContactIdentity `json:"identities,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// NewContact creates a new contact
func NewContact(tenantID string) *Contact {
	now := time.Now()
	return &Contact{
		TenantID:     tenantID,
		CustomFields: make(map[string]string),
		Tags:         []string{},
		Identities:   []*ContactIdentity{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// AddIdentity adds an identity to the contact
func (c *Contact) AddIdentity(identity *ContactIdentity) {
	identity.ContactID = c.ID
	c.Identities = append(c.Identities, identity)
}

// GetIdentityByChannel returns the identity for a specific channel type
func (c *Contact) GetIdentityByChannel(channelType string) *ContactIdentity {
	for _, identity := range c.Identities {
		if identity.ChannelType == channelType {
			return identity
		}
	}
	return nil
}

// HasTag checks if the contact has a specific tag
func (c *Contact) HasTag(tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ContactPreference represents opt-in/out preferences per channel
type ContactPreference struct {
	ChannelType string     `json:"channel_type"`
	OptedIn     bool       `json:"opted_in"`
	BlockedAt   *time.Time `json:"blocked_at,omitempty"`
}

// IsBlocked returns true if the contact is blocked
func (c *Contact) IsBlocked() bool {
	if c.CustomFields == nil {
		return false
	}
	return c.CustomFields["_blocked"] == "true"
}

// Block marks the contact as blocked
func (c *Contact) Block() {
	if c.CustomFields == nil {
		c.CustomFields = make(map[string]string)
	}
	c.CustomFields["_blocked"] = "true"
	c.CustomFields["_blocked_at"] = time.Now().UTC().Format(time.RFC3339)
	c.UpdatedAt = time.Now()
}

// Unblock removes the blocked status
func (c *Contact) Unblock() {
	if c.CustomFields == nil {
		return
	}
	delete(c.CustomFields, "_blocked")
	delete(c.CustomFields, "_blocked_at")
	c.UpdatedAt = time.Now()
}

// GetBlockedAt returns when the contact was blocked, or nil
func (c *Contact) GetBlockedAt() *time.Time {
	if c.CustomFields == nil {
		return nil
	}
	blockedAt, ok := c.CustomFields["_blocked_at"]
	if !ok {
		return nil
	}
	t, err := time.Parse(time.RFC3339, blockedAt)
	if err != nil {
		return nil
	}
	return &t
}
