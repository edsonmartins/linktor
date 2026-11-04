package webhook

import (
	"encoding/json"
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// Consumers of a channel (P5 of the CRM ↔ Sales Copilot agreement, VCR-36).
// The channel's WebhookURL + webhook_secret are the DEFAULT consumer, as
// before. A channel may declare more, by name, with no new table:
//
//	Config["consumers.<name>.webhook_url"]
//	Config["consumers.<name>.webhook_events"]   (same filter as webhook_events)
//	Credentials["consumers.<name>.webhook_secret"]   (encrypted at rest)
//
// Config["webhook_consumer_name"] optionally names the default consumer (for
// example "crm"): an owner rule naming it goes to the default, with no fallback.
const cfgDefaultConsumerName = "webhook_consumer_name"

// Motivos de fallback ao consumidor padrão (routing.motivo).
const (
	MotivoConsumidorNaoConfigurado = "consumidor_nao_configurado"
	MotivoTentativasEsgotadas      = "tentativas_esgotadas"
	MotivoRegraIndisponivel        = "regra_indisponivel"
)

// Routing is the optional `routing` block of the envelope: which consumer the
// event went to, the owner it was meant for when that differs, whether it was a
// fallback to the default, and why.
type Routing struct {
	Consumer string `json:"consumer"`
	Dono     string `json:"dono,omitempty"`
	Fallback bool   `json:"fallback"`
	Motivo   string `json:"motivo,omitempty"`
}

func consumerKey(name, field string) string {
	return "consumers." + name + "." + field
}

// consumerURL is the delivery URL of a named consumer ("" = not configured).
func consumerURL(ch *entity.Channel, name string) string {
	return strings.TrimSpace(ch.Config[consumerKey(name, "webhook_url")])
}

// consumerSecret is the HMAC secret of a named consumer ("" = not configured).
func consumerSecret(ch *entity.Channel, name string) string {
	return ch.Credentials[consumerKey(name, "webhook_secret")]
}

// defaultConsumerName is how the default consumer is called in routing ("default" when unnamed).
func defaultConsumerName(ch *entity.Channel) string {
	if n := strings.TrimSpace(ch.Config[cfgDefaultConsumerName]); n != "" {
		return n
	}
	return "default"
}

// consumerSubscribedTo is subscribedTo for a named consumer's own filter.
func consumerSubscribedTo(ch *entity.Channel, name, eventType string) bool {
	return eventsAllow(ch.Config[consumerKey(name, "webhook_events")], eventType)
}

// withRouting returns body with the envelope's `routing` set. Used by the
// worker when it switches a delivery to the default consumer; the body is then
// signed as sent (sign-what-you-send).
func withRouting(body []byte, r Routing) ([]byte, error) {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	env["routing"] = raw
	return json.Marshal(env)
}
