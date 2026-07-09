package whatsapp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
)

func TestIsLID(t *testing.T) {
	assert.True(t, isLID(types.NewJID("123", types.HiddenUserServer)))
	assert.True(t, isLID(types.NewJID("123", types.HostedLIDServer)))
	assert.False(t, isLID(types.NewJID("5511999999999", types.DefaultUserServer)))
	assert.False(t, isLID(types.NewJID("g", types.GroupServer)))
}

func TestResolvePN_NonLIDPassthrough(t *testing.T) {
	c := &Client{}
	pn := types.NewJID("5511999999999", types.DefaultUserServer)
	assert.Equal(t, pn, c.ResolvePN(context.Background(), pn))
}

func TestResolvePN_NilClientPassthrough(t *testing.T) {
	c := &Client{}
	lid := types.NewJID("456", types.HiddenUserServer)
	// No underlying whatsmeow client -> must return the LID unchanged, not panic.
	assert.Equal(t, lid, c.ResolvePN(context.Background(), lid))
}

func TestResolveLID_NonPNPassthrough(t *testing.T) {
	c := &Client{}
	lid := types.NewJID("456", types.HiddenUserServer)
	assert.Equal(t, lid, c.ResolveLID(context.Background(), lid))
}

func TestLIDPNCache_GetPutExpiry(t *testing.T) {
	c := &Client{}
	key := types.NewJID("789", types.HiddenUserServer)
	val := types.NewJID("5511888887777", types.DefaultUserServer)

	_, ok := c.lidPNGet(key)
	assert.False(t, ok, "empty cache misses")

	c.lidPNPut(key, val)
	got, ok := c.lidPNGet(key)
	assert.True(t, ok)
	assert.Equal(t, val, got)

	// Force expiry and confirm it is treated as a miss.
	c.lidPNMu.Lock()
	c.lidPNCache[key] = lidCacheEntry{jid: val, expiresAt: time.Now().Add(-time.Minute)}
	c.lidPNMu.Unlock()
	_, ok = c.lidPNGet(key)
	assert.False(t, ok, "expired entries miss")
}

func TestGetAvatar_NotReady(t *testing.T) {
	c := &Client{}
	_, err := c.GetAvatar(context.Background(), types.NewJID("5511999999999", types.DefaultUserServer), "")
	assert.ErrorIs(t, err, ErrClientNotReady)
}
