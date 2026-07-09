package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// WhatsApp is migrating identities from phone-number JIDs (@s.whatsapp.net) to
// opaque "LID" JIDs (@lid). Inbound messages from privacy-enabled contacts and
// groups increasingly arrive addressed by LID, which no longer contains the
// phone number. To keep conversations keyed on a stable phone identity we
// resolve LID→PN via the device store, memoized with a TTL.

const lidPNCacheTTL = 30 * time.Minute

type lidCacheEntry struct {
	jid       types.JID
	expiresAt time.Time
}

// isLID reports whether a JID is a hidden-user (LID) address.
func isLID(jid types.JID) bool {
	return jid.Server == types.HiddenUserServer || jid.Server == types.HostedLIDServer
}

// ResolvePN maps a hidden-user (@lid) JID to its phone-number JID so inbound
// senders identified only by LID still match contacts keyed by phone. Non-LID
// JIDs and unresolvable LIDs are returned unchanged.
func (c *Client) ResolvePN(ctx context.Context, jid types.JID) types.JID {
	if !isLID(jid) {
		return jid
	}

	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()
	if client == nil {
		return jid
	}

	key := jid.ToNonAD()
	if pn, ok := c.lidPNGet(key); ok {
		return pn
	}

	pn, err := client.Store.LIDs.GetPNForLID(ctx, key)
	if err != nil || pn.User == "" {
		return jid
	}
	c.lidPNPut(key, pn)
	return pn
}

// ResolveLID maps a phone-number JID to its LID when the mapping is known,
// returning the input unchanged otherwise. Useful when addressing a recipient
// that expects LID-based delivery.
func (c *Client) ResolveLID(ctx context.Context, jid types.JID) types.JID {
	if jid.Server != types.DefaultUserServer {
		return jid
	}

	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()
	if client == nil {
		return jid
	}

	key := jid.ToNonAD()
	if lid, ok := c.lidPNGet(key); ok {
		return lid
	}

	lid, err := client.Store.LIDs.GetLIDForPN(ctx, key)
	if err != nil || lid.User == "" {
		return jid
	}
	c.lidPNPut(key, lid)
	return lid
}

func (c *Client) lidPNGet(key types.JID) (types.JID, bool) {
	c.lidPNMu.Lock()
	defer c.lidPNMu.Unlock()

	e, ok := c.lidPNCache[key]
	if !ok || time.Now().After(e.expiresAt) {
		return types.JID{}, false
	}
	return e.jid, true
}

func (c *Client) lidPNPut(key, val types.JID) {
	c.lidPNMu.Lock()
	defer c.lidPNMu.Unlock()

	if c.lidPNCache == nil {
		c.lidPNCache = make(map[types.JID]lidCacheEntry)
	}
	c.lidPNCache[key] = lidCacheEntry{jid: val, expiresAt: time.Now().Add(lidPNCacheTTL)}
}

// AvatarInfo is the result of a profile-picture lookup that supports skipping
// the download when the picture has not changed (via ExistingID).
type AvatarInfo struct {
	URL       string
	PictureID string
	// Unchanged is true when the caller passed an ExistingID that still matches,
	// so no new URL was fetched and the cached avatar remains valid.
	Unchanged bool
}

// GetAvatar fetches a contact's profile-picture URL, passing existingID so
// WhatsApp can answer "not modified" cheaply. When the picture is unchanged the
// returned AvatarInfo has Unchanged=true and an empty URL; callers keep their
// cached copy. A contact with no picture yields a zero AvatarInfo and nil error.
func (c *Client) GetAvatar(ctx context.Context, jid types.JID, existingID string) (*AvatarInfo, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	info, err := client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{
		ExistingID: existingID,
		Preview:    false,
	})
	if err != nil {
		return nil, err
	}
	// A nil info with existingID set means the picture is unchanged.
	if info == nil {
		if existingID != "" {
			return &AvatarInfo{PictureID: existingID, Unchanged: true}, nil
		}
		return &AvatarInfo{}, nil
	}
	return &AvatarInfo{URL: info.URL, PictureID: info.ID}, nil
}
