package vendax

import "sync"

// dedupe é um conjunto FIFO de chaves já vistas, com tamanho máximo. Protege a entrega outbound de
// duplicatas: o Core publica o outbound via NATS core, e um retry do outbox pode reemitir a mesma
// idempotencyKey — sem esta guarda o SendMessageUseCase entregaria a mensagem duas vezes ao cliente.
type dedupe struct {
	mu    sync.Mutex
	seen  map[string]struct{}
	order []string
	max   int
}

func newDedupe(max int) *dedupe {
	return &dedupe{seen: make(map[string]struct{}), max: max}
}

// seenBefore registra a chave e devolve true se ela já havia sido vista. Chave vazia nunca dedup
// (sem idempotencyKey não há como distinguir reenvio de mensagem nova).
func (d *dedupe) seenBefore(key string) bool {
	if key == "" {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[key]; ok {
		return true
	}
	d.seen[key] = struct{}{}
	d.order = append(d.order, key)
	if len(d.order) > d.max {
		oldest := d.order[0]
		d.order = d.order[1:]
		delete(d.seen, oldest)
	}
	return false
}
