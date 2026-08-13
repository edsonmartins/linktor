package types

// DirectSendInput is the body of POST /messages/send: a channel + recipient
// send that does not require the caller to know a conversation. Linktor
// resolves (or creates) the recipient's identity, contact and conversation
// inside the tenant before persisting and queueing the message.
//
// Metadata is carried end to end — it is persisted on the message and reaches
// the channel adapter. Two keys change behaviour rather than merely riding
// along:
//   - "idempotency_key": unique per tenant. Repeating a call with the same key
//     returns the original message instead of sending a second one.
//   - "subject": used as the subject line by the email channel.
//
// Metadata may not carry Linktor-internal fields (reply threading, reaction
// targets, campaign bookkeeping, routing ids); such a request is rejected.
type DirectSendInput struct {
	ChannelID string `json:"channel_id"`
	To        string `json:"to"`
	// ContentType is currently restricted to "text" (the default when empty).
	ContentType string            `json:"content_type,omitempty"`
	Text        string            `json:"text,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DirectSendResult is the canonical answer to a direct send. The message is
// queued, not yet delivered: Status is "queued" and the delivery outcome
// arrives later on the channel's webhook as message.sent / message.failed.
type DirectSendResult struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	ChannelID      string `json:"channel_id"`
	Status         string `json:"status"`
}
