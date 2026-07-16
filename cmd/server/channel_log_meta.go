package main

import "github.com/msgfy/linktor/internal/infrastructure/nats"

// inboundLogMeta builds the metadata for a channel observability log entry from
// an inbound message. It deliberately carries only routing facts — channel,
// direction, and the contact's origin/destination identifiers — and never the
// message body, which is not appropriate to surface in the channel log.
func inboundLogMeta(msg *nats.InboundMessage) map[string]string {
	meta := map[string]string{"direction": "inbound"}
	if msg == nil {
		return meta
	}
	meta["channel_type"] = msg.ChannelType

	md := msg.Metadata
	// Origin: who sent it (the contact).
	if from := firstNonEmpty(md["sender_id"], md["from"], md["sender_pn"], md["phone"]); from != "" {
		meta["from"] = from
	}
	if name := md["sender_name"]; name != "" {
		meta["sender_name"] = name
	}
	// Destination: our channel address, when the provider supplies it.
	if to := firstNonEmpty(md["to"], md["recipient"], md["business_phone"], md["phone_number"]); to != "" {
		meta["to"] = to
	}
	return meta
}

// firstNonEmpty returns the first non-empty string, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
