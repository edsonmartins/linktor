package vendax

import "fmt"

// Subjects do VendaX Core (espelham CoreSubjects.java: tenant.{id}.<dominio>.<evento>).
// O Core usa NATS core pub/sub (plain), não JetStream.

// coreOutboundWildcard: o Core publica a saída por tenant; o bridge assina todos.
const coreOutboundWildcard = "tenant.*.core.outbound"

// coreChannelConfigWildcard: o Admin VendaX (CA-06) publica a config de canais por tenant.
const coreChannelConfigWildcard = "tenant.*.core.channel.config"

// coreInboundSubject: onde o bridge publica o envelope normalizado que o Core consome.
func coreInboundSubject(tenantID string) string {
	return fmt.Sprintf("tenant.%s.linktor.inbound", tenantID)
}
