package officialcalls

import "encoding/json"

// webhookEnvelope is the standard Cloud API webhook shape, narrowed to the
// "calls" field we care about here.
type webhookEnvelope struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				MessagingProduct string             `json:"messaging_product"`
				Metadata         WebhookMetadata    `json:"metadata"`
				Calls            []WebhookCallEvent `json:"calls"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// WebhookMetadata identifies the receiving phone number.
type WebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// CallWebhook is a flattened call event paired with the receiving number.
type CallWebhook struct {
	PhoneNumberID string
	Event         WebhookCallEvent
}

// ParseWebhookCalls extracts call events from a Cloud API webhook body. It
// ignores non-call changes and never errors on unrelated payloads (returns an
// empty slice), so it is safe to call on every inbound webhook.
func ParseWebhookCalls(body []byte) ([]CallWebhook, error) {
	var env webhookEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	var out []CallWebhook
	for _, entry := range env.Entry {
		for _, ch := range entry.Changes {
			// Only the "calls" change field carries call events; ignore anything
			// else even if it happens to include a calls array.
			if ch.Field != "calls" || len(ch.Value.Calls) == 0 {
				continue
			}
			pnID := ch.Value.Metadata.PhoneNumberID
			for _, ev := range ch.Value.Calls {
				out = append(out, CallWebhook{PhoneNumberID: pnID, Event: ev})
			}
		}
	}
	return out, nil
}
