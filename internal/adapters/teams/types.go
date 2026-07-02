// Package teams implements the Microsoft Teams channel connector on top of the
// Bot Framework / Bot Connector REST API.
//
// Inbound: Teams delivers Activity payloads to a public webhook (handled by the
// API webhook handler), authenticated with a Bot Connector JWT. Outbound: the
// connector posts Activities back to {serviceUrl}/v3/conversations/{id}/activities
// using a short-lived AAD app token (client-credentials), cached and refreshed.
//
// Credentials (channel.credentials):
//   - app_id        Azure AD application (client) id of the bot
//   - app_password  Azure AD client secret
//   - tenant_id     Azure AD tenant; "common"/"botframework.com" for multi-tenant
//   - service_url    Bot Connector base URL; discovered from inbound, may be seeded
//
// Two ownership models are supported without a fork:
//  1. Single-tenant: the app is registered in the customer's AAD tenant.
//  2. Multi-tenant (Integrall.tech): one app serves many customers; the org is
//     distinguished by the Activity's channelData/tenant id. Channels share
//     app_id/app_password and differ by tenant_id.
package teams

import "time"

// Credential / config keys stored in channel.Credentials.
const (
	CredAppID       = "app_id"
	CredAppPassword = "app_password"
	CredTenantID    = "tenant_id"
	CredServiceURL  = "service_url"
)

// ChannelType is the canonical channel type string for Teams.
const ChannelType = "teams"

// Config is the resolved Teams channel configuration.
type Config struct {
	AppID       string
	AppPassword string
	TenantID    string
	ServiceURL  string
}

// configFromCreds extracts Teams config from a channel credentials map.
func configFromCreds(creds map[string]string) Config {
	return Config{
		AppID:       creds[CredAppID],
		AppPassword: creds[CredAppPassword],
		TenantID:    creds[CredTenantID],
		ServiceURL:  creds[CredServiceURL],
	}
}

// Activity is the subset of the Bot Framework Activity schema we consume/produce.
// See https://learn.microsoft.com/azure/bot-service/rest-api/bot-framework-rest-connector-api-reference
type Activity struct {
	Type         string              `json:"type"`
	ID           string              `json:"id,omitempty"`
	Timestamp    time.Time           `json:"timestamp,omitempty"`
	ServiceURL   string              `json:"serviceUrl,omitempty"`
	ChannelID    string              `json:"channelId,omitempty"` // "msteams"
	From         ChannelAccount      `json:"from,omitempty"`
	Conversation ConversationAccount `json:"conversation,omitempty"`
	Recipient    ChannelAccount      `json:"recipient,omitempty"`
	Text         string              `json:"text,omitempty"`
	Locale       string              `json:"locale,omitempty"`
	Attachments  []Attachment        `json:"attachments,omitempty"`
	ChannelData  ChannelData         `json:"channelData,omitempty"`
	ReplyToID    string              `json:"replyToId,omitempty"`
}

// ChannelAccount identifies a user or bot on the channel.
type ChannelAccount struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	AadObjectID string `json:"aadObjectId,omitempty"`
}

// ConversationAccount identifies a conversation.
type ConversationAccount struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name,omitempty"`
	IsGroup          bool   `json:"isGroup,omitempty"`
	ConversationType string `json:"conversationType,omitempty"`
	TenantID         string `json:"tenantId,omitempty"`
}

// ChannelData carries Teams-specific routing data (notably the AAD tenant).
type ChannelData struct {
	Tenant struct {
		ID string `json:"id,omitempty"`
	} `json:"tenant,omitempty"`
}

// Attachment is a content attachment on an Activity.
type Attachment struct {
	ContentType string      `json:"contentType,omitempty"`
	ContentURL  string      `json:"contentUrl,omitempty"`
	Name        string      `json:"name,omitempty"`
	Content     interface{} `json:"content,omitempty"`
}

// ConversationReference is the durable address needed to reply proactively
// (outside an active turn). It is captured on the first inbound Activity.
type ConversationReference struct {
	ServiceURL     string `json:"serviceUrl"`
	ConversationID string `json:"conversationId"`
	TenantID       string `json:"tenantId,omitempty"`
	BotID          string `json:"botId,omitempty"`
}

// ReferenceFromActivity extracts the conversation reference from an inbound Activity.
func ReferenceFromActivity(a *Activity) ConversationReference {
	return ConversationReference{
		ServiceURL:     a.ServiceURL,
		ConversationID: a.Conversation.ID,
		TenantID:       a.tenantID(),
		BotID:          a.Recipient.ID,
	}
}

// tenantID returns the AAD tenant for the activity, preferring channelData and
// falling back to the conversation tenant — the discriminator for the
// multi-tenant (Integrall.tech) bot ownership model.
func (a *Activity) tenantID() string {
	if a.ChannelData.Tenant.ID != "" {
		return a.ChannelData.Tenant.ID
	}
	return a.Conversation.TenantID
}

// AADTenant exposes the activity's Azure AD tenant id for channel routing on a
// shared inbound endpoint.
func (a *Activity) AADTenant() string {
	return a.tenantID()
}
