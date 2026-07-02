// Package mattermost implements the Mattermost channel connector for
// self-hosted Mattermost servers.
//
// Inbound: a persistent WebSocket to {base_url}/api/v4/websocket streams events;
// the connector consumes "posted" events, filters the bot's own posts
// (anti-echo) and reconnects with backoff. Outbound: REST POST
// {base_url}/api/v4/posts authenticated with the bot's Personal Access Token.
//
// Credentials (channel.credentials):
//   - base_url      Mattermost server base URL (per channel — self-hosted!)
//   - bot_token     bot Personal Access Token (bearer)
//   - bot_user_id   the bot's user id (used to filter its own posts)
//
// Network reachability from Linktor to the customer's Mattermost server is a
// deployment prerequisite.
//
// Unlike the webhook-based connectors (Teams/Slack), inbound here requires
// Linktor to hold an outbound WebSocket. Outbound still flows through the
// unified delivery worker via the registered Sender; only inbound needs the
// persistent connection, which a dedicated Manager owns (rather than the heavy
// plugin.ChannelAdapter machinery, since the worker already handles outbound).
package mattermost

import "encoding/json"

// Credential keys stored in channel.Credentials.
const (
	CredBaseURL   = "base_url"
	CredBotToken  = "bot_token"
	CredBotUserID = "bot_user_id"
)

// ChannelType is the canonical channel type string for Mattermost.
const ChannelType = "mattermost"

// Config is the resolved Mattermost channel configuration.
type Config struct {
	BaseURL   string
	BotToken  string
	BotUserID string
}

func configFromCreds(creds map[string]string) Config {
	return Config{
		BaseURL:   creds[CredBaseURL],
		BotToken:  creds[CredBotToken],
		BotUserID: creds[CredBotUserID],
	}
}

// wsEvent is a Mattermost WebSocket event frame. It also carries the fields the
// server uses to reply to an authentication_challenge (status/seq_reply/error).
type wsEvent struct {
	Event    string      `json:"event"`
	Data     wsEventData `json:"data"`
	Seq      int64       `json:"seq"`
	Status   string      `json:"status,omitempty"`
	SeqReply int64       `json:"seq_reply,omitempty"`
	Error    *wsError    `json:"error,omitempty"`
}

// wsError is the error object on a failed WebSocket reply.
type wsError struct {
	Message string `json:"message"`
}

// wsEventData carries event payload; for "posted" the post is a JSON-encoded
// string (Mattermost double-encodes it).
type wsEventData struct {
	Post        string `json:"post"`
	ChannelType string `json:"channel_type"`
	SenderName  string `json:"sender_name"`
}

// Post is the subset of a Mattermost post we consume.
type Post struct {
	ID        string   `json:"id"`
	ChannelID string   `json:"channel_id"`
	UserID    string   `json:"user_id"`
	Message   string   `json:"message"`
	CreateAt  int64    `json:"create_at"`
	FileIDs   []string `json:"file_ids,omitempty"`
}

// FileInfo is the metadata returned by GET /api/v4/files/{id}/info.
type FileInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Extension string `json:"extension"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
}

// parsePost decodes the double-encoded post string from a "posted" event.
func parsePost(raw string) (*Post, error) {
	var p Post
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, err
	}
	return &p, nil
}
