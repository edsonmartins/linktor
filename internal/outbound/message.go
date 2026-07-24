// Package outbound defines the channel-agnostic outbound delivery subsystem:
// a typed message model, a per-channel Sender abstraction, a Resolver that
// builds senders from channel credentials, and a worker that drives delivery
// off the NATS outbound stream.
//
// It is deliberately independent of the inbound plugin adapters: adapters own
// session/transport lifecycle, while Senders are stateless per-channel and own
// only "given these credentials, deliver this message".
package outbound

// Kind enumerates the supported outbound content variants.
type Kind string

const (
	KindText        Kind = "text"
	KindTemplate    Kind = "template"
	KindMedia       Kind = "media"
	KindInteractive Kind = "interactive"
)

// Content is a typed outbound payload. Implementations are value types that
// each Sender translates into its provider's wire format. This replaces the
// previous untyped (ContentType string + Content + Metadata grab-bag) model.
type Content interface {
	Kind() Kind
}

// Text is a plain text message.
type Text struct {
	Body       string
	PreviewURL bool
}

func (Text) Kind() Kind { return KindText }

// Template is an approved template send with positional body parameters.
type Template struct {
	Name     string
	Language string
	// BodyParams are the positional {{1..n}} body substitutions.
	BodyParams []string
	// ComponentsJSON, when non-empty, is a provider-native components payload
	// that takes precedence over BodyParams (header/buttons/carousel support).
	ComponentsJSON string
}

func (Template) Kind() Kind { return KindTemplate }

// MediaType enumerates media sub-kinds.
type MediaType string

const (
	MediaImage    MediaType = "image"
	MediaDocument MediaType = "document"
	MediaVideo    MediaType = "video"
	MediaAudio    MediaType = "audio"
)

// Media is an image/document/video/audio message addressed by URL or media ID.
type Media struct {
	Type     MediaType
	URL      string
	MediaID  string
	Caption  string
	Filename string
}

func (Media) Kind() Kind { return KindMedia }

// InteractiveButton is a channel-agnostic quick-reply option. Each Sender maps
// it to its provider's native control (WhatsApp reply button / list row,
// Telegram inline keyboard button, …).
type InteractiveButton struct {
	ID    string
	Title string
}

// Interactive is a message offering quick-reply buttons/options plus body text.
// Senders that cannot render interactive controls fall back to sending Body as
// plain text, so no message is ever dropped.
type Interactive struct {
	Body    string
	Buttons []InteractiveButton
}

func (Interactive) Kind() Kind { return KindInteractive }

// Message is a normalized outbound message ready for delivery on a specific
// channel. It is produced by translating a transport-level nats.OutboundMessage.
type Message struct {
	ID        string // internal message ID (for status correlation)
	TenantID  string
	ChannelID string
	// ConversationID links the send to its conversation ("" for sends without
	// one, e.g. campaigns). The 24h-window policy uses it to look up the last
	// inbound message.
	ConversationID string
	To             string // recipient address (phone, etc.)
	Content        Content

	// CampaignRecipientID, when set, links this delivery to a campaign recipient
	// so the worker can update its status after a real send.
	CampaignRecipientID string
	// CampaignID, when set, lets the worker honor a campaign cancellation.
	CampaignID string

	// Metadata carries the original transport-level metadata for channel-specific
	// fields that don't fit the typed content (e.g. an email subject).
	Metadata map[string]string
}

// Meta returns the metadata value for key, or "" when absent.
func (m *Message) Meta(key string) string {
	if m.Metadata == nil {
		return ""
	}
	return m.Metadata[key]
}
