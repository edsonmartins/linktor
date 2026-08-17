package whatsapp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
)

// The address book import exists so an agent's hand-saved numbers can be
// matched against customer records. These tests protect the two rules that
// decide whether that matching is trustworthy: what counts as a person, and
// when a phone number is actually known.

func TestContactEntry_PhoneJIDCarriesTheNumber(t *testing.T) {
	jid := types.NewJID("5521999998888", types.DefaultUserServer)

	entry, keep := contactEntry(jid, types.ContactInfo{FullName: "Rosa"}, jid)

	assert.True(t, keep)
	assert.Equal(t, "5521999998888", entry.Phone)
	assert.Equal(t, "Rosa", entry.FullName)
}

func TestContactEntry_UnresolvedLIDHasNoPhone(t *testing.T) {
	// WhatsApp is moving to opaque LID identities. When the LID→PN mapping is
	// not in the device store, ResolvePN hands the LID back unchanged.
	lid := types.NewJID("184719283746", types.HiddenUserServer)

	entry, keep := contactEntry(lid, types.ContactInfo{PushName: "Alguém"}, lid)

	// The contact is still returned — the caller must be able to see how much
	// of the address book is LID-only. Dropping it would make an incomplete
	// import look complete.
	assert.True(t, keep)
	// And Phone stays empty rather than carrying the opaque id, which would be
	// published as if it were a phone number and match nothing forever.
	assert.Empty(t, entry.Phone)
	assert.Equal(t, "Alguém", entry.PushName)
}

func TestContactEntry_ResolvedLIDCarriesTheResolvedNumber(t *testing.T) {
	lid := types.NewJID("184719283746", types.HiddenUserServer)
	resolved := types.NewJID("5521999998888", types.DefaultUserServer)

	entry, keep := contactEntry(lid, types.ContactInfo{FullName: "Marcello"}, resolved)

	assert.True(t, keep)
	assert.Equal(t, "5521999998888", entry.Phone)
	// The JID stays the original one: it is how the channel addresses this
	// contact, and rewriting it here would break sending.
	assert.Equal(t, lid, entry.JID)
}

func TestContactEntry_GroupsAndBroadcastsAreNotPeople(t *testing.T) {
	// A group JID would never match a customer record. Letting these through
	// would pad the unmatched pile with entries nobody can act on.
	group := types.NewJID("120363000000000000", types.GroupServer)
	broadcast := types.NewJID("status", types.BroadcastServer)

	_, keepGroup := contactEntry(group, types.ContactInfo{FullName: "Equipe"}, group)
	_, keepBroadcast := contactEntry(broadcast, types.ContactInfo{}, broadcast)

	assert.False(t, keepGroup)
	assert.False(t, keepBroadcast)
}

func TestGetAllContacts_RequiresConnectedClient(t *testing.T) {
	client, err := NewClient(&Config{ChannelID: "test-channel"})
	assert.NoError(t, err)

	contacts, err := client.GetAllContacts(context.Background())

	// Never an empty list on a disconnected client: "no contacts" and "not
	// connected" would be indistinguishable, and the import would silently
	// report zero.
	assert.ErrorIs(t, err, ErrClientNotReady)
	assert.Nil(t, contacts)
}
