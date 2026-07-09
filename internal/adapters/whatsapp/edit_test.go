package whatsapp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// When the underlying client is not connected, all mutation helpers must fail
// fast with ErrClientNotReady instead of panicking on a nil client.
func TestEditRevokeForward_NotReady(t *testing.T) {
	c := &Client{}
	ctx := context.Background()

	_, err := c.EditMessage(ctx, "5511999999999", "MID", "new")
	assert.ErrorIs(t, err, ErrClientNotReady)

	_, err = c.RevokeMessage(ctx, "5511999999999", "", "MID")
	assert.ErrorIs(t, err, ErrClientNotReady)

	_, err = c.ForwardText(ctx, "5511999999999", "hi")
	assert.ErrorIs(t, err, ErrClientNotReady)
}

func TestAdapterEditRevoke_NotConnected(t *testing.T) {
	a := NewAdapter()
	ctx := context.Background()

	_, err := a.EditMessage(ctx, "5511999999999", "MID", "new")
	assert.ErrorIs(t, err, ErrClientNotReady)

	_, err = a.RevokeMessage(ctx, "5511999999999", "", "MID")
	assert.ErrorIs(t, err, ErrClientNotReady)
}
