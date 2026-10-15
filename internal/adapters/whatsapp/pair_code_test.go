package whatsapp

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// O WhatsApp recusa (400) o pedido de código com nome fora de "Navegador (Sistema)".
// Com o padrão "Linktor" todo /pair respondia 500, qualquer que fosse o telefone.
func TestPairDisplayName(t *testing.T) {
	assert.Equal(t, "Chrome (Linux)", pairDisplayName("Linktor"), "o padrão do Linktor não é aceito pelo WhatsApp")
	assert.Equal(t, "Chrome (Linux)", pairDisplayName(""))
	assert.Equal(t, "Chrome (Linux)", pairDisplayName("Cobrança RQ"))
	assert.Equal(t, "Firefox (Mac OS)", pairDisplayName("Firefox (Mac OS)"), "nome já no formato fica como está")
	assert.Equal(t, "Chrome (Windows)", pairDisplayName("Chrome (Windows)"))
}

func TestWaitForLoginSocket(t *testing.T) {
	inicio := time.Now()
	var conectado atomic.Bool
	go func() {
		time.Sleep(200 * time.Millisecond)
		conectado.Store(true)
	}()
	waitForLoginSocket(context.Background(), conectado.Load)
	decorrido := time.Since(inicio)
	assert.GreaterOrEqual(t, decorrido, 1100*time.Millisecond, "espera conectar e mais um segundo")
	assert.Less(t, decorrido, 3*time.Second)

	// Quem pediu desistiu: não segura a requisição.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	inicio = time.Now()
	waitForLoginSocket(ctx, func() bool { return false })
	assert.Less(t, time.Since(inicio), 500*time.Millisecond)
}
