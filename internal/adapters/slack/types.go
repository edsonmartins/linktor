// Package slack implements the Slack channel connector on top of the Slack
// Events API (inbound) and Web API (outbound).
//
// Inbound: Slack POSTs events to a public webhook (handled by the API webhook
// handler), authenticated with an X-Slack-Signature HMAC over the raw body. The
// initial url_verification handshake echoes the challenge. Outbound: the
// connector calls chat.postMessage with the bot token.
//
// Credentials (channel.credentials):
//   - bot_token       Slack bot token (xoxb-...)
//   - signing_secret  Slack app signing secret (HMAC key for inbound)
//   - app_id          Slack app id (optional, for reference)
//   - bot_user_id     the bot's own user id (optional; used to filter echoes)
//
// Socket Mode (WebSocket, no public webhook) is a possible follow-up; the v1
// connector uses the public Events API webhook.
package slack

// Credential keys stored in channel.Credentials.
const (
	CredBotToken      = "bot_token"
	CredSigningSecret = "signing_secret"
	CredAppID         = "app_id"
	CredBotUserID     = "bot_user_id"
)

// ChannelType is the canonical channel type string for Slack.
const ChannelType = "slack"

// EventEnvelope is the outer Slack Events API payload.
type EventEnvelope struct {
	Type      string      `json:"type"`      // "url_verification" | "event_callback"
	Challenge string      `json:"challenge"` // present on url_verification
	TeamID    string      `json:"team_id"`
	APIAppID  string      `json:"api_app_id"`
	Event     *InnerEvent `json:"event,omitempty"`
	EventID   string      `json:"event_id"`
}

// InnerEvent is the actual event (we consume "message").
type InnerEvent struct {
	Type     string `json:"type"`    // "message"
	Subtype  string `json:"subtype"` // "bot_message", "message_changed", ...
	User     string `json:"user"`
	BotID    string `json:"bot_id"`
	Text     string `json:"text"`
	Channel  string `json:"channel"`
	TS       string `json:"ts"`
	ThreadTS string `json:"thread_ts"`
	Files    []File `json:"files,omitempty"`
}

// File is a Slack file attachment reference.
type File struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Mimetype   string `json:"mimetype"`
	URLPrivate string `json:"url_private"`
	Size       int64  `json:"size"`
}

// IsBotEcho reports whether the event originated from a bot (the app's own
// outbound messages echo back as events and must be ignored to avoid loops).
func (e *InnerEvent) IsBotEcho(botUserID string) bool {
	if e == nil {
		return true
	}
	if e.BotID != "" || e.Subtype == "bot_message" {
		return true
	}
	if botUserID != "" && e.User == botUserID {
		return true
	}
	return false
}
