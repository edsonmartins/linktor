package entity

import (
	"time"
)

// ConversationStatus represents the status of a conversation
type ConversationStatus string

const (
	ConversationStatusOpen     ConversationStatus = "open"
	ConversationStatusPending  ConversationStatus = "pending"
	ConversationStatusResolved ConversationStatus = "resolved"
	ConversationStatusClosed   ConversationStatus = "closed"
)

// ConversationPriority represents the priority of a conversation
type ConversationPriority string

const (
	ConversationPriorityLow    ConversationPriority = "low"
	ConversationPriorityNormal ConversationPriority = "normal"
	ConversationPriorityHigh   ConversationPriority = "high"
	ConversationPriorityUrgent ConversationPriority = "urgent"
)

// Conversation represents a conversation thread
type Conversation struct {
	ID             string               `json:"id"`
	TenantID       string               `json:"tenant_id"`
	ContactID      string               `json:"contact_id"`
	ChannelID      string               `json:"channel_id"`
	AssignedUserID *string              `json:"assigned_user_id,omitempty"`
	Status         ConversationStatus   `json:"status"`
	Priority       ConversationPriority `json:"priority"`
	Subject        string               `json:"subject,omitempty"`
	Tags           []string             `json:"tags,omitempty"`
	Metadata       map[string]string    `json:"metadata,omitempty"`
	UnreadCount    int                  `json:"unread_count"`
	LastMessageAt  *time.Time           `json:"last_message_at,omitempty"`
	FirstReplyAt   *time.Time           `json:"first_reply_at,omitempty"`
	ResolvedAt     *time.Time           `json:"resolved_at,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// NewConversation creates a new conversation
func NewConversation(tenantID, contactID, channelID string) *Conversation {
	now := time.Now()
	return &Conversation{
		TenantID:  tenantID,
		ContactID: contactID,
		ChannelID: channelID,
		Status:    ConversationStatusOpen,
		Priority:  ConversationPriorityNormal,
		Tags:      []string{},
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsOpen returns true if the conversation is open
func (c *Conversation) IsOpen() bool {
	return c.Status == ConversationStatusOpen || c.Status == ConversationStatusPending
}

// Resolve marks the conversation as resolved
func (c *Conversation) Resolve() {
	now := time.Now()
	c.Status = ConversationStatusResolved
	c.ResolvedAt = &now
	c.UpdatedAt = now
}

// Reopen reopens a resolved conversation
func (c *Conversation) Reopen() {
	c.Status = ConversationStatusOpen
	c.ResolvedAt = nil
	c.UpdatedAt = time.Now()
}

// Assign assigns the conversation to a user
func (c *Conversation) Assign(userID string) {
	c.AssignedUserID = &userID
	c.UpdatedAt = time.Now()
}

// Unassign removes the assignment from the conversation
func (c *Conversation) Unassign() {
	c.AssignedUserID = nil
	c.UpdatedAt = time.Now()
}
