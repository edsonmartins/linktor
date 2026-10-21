package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewContact(t *testing.T) {
	contact := NewContact("tenant1")
	assert.Equal(t, "tenant1", contact.TenantID)
	assert.NotNil(t, contact.CustomFields)
	assert.NotNil(t, contact.Tags)
	assert.NotNil(t, contact.Identities)
	assert.NotZero(t, contact.CreatedAt)
}

func TestContact_AddIdentity(t *testing.T) {
	contact := NewContact("tenant1")
	contact.ID = "contact1"
	identity := &ContactIdentity{ID: "id1", ChannelType: "whatsapp", Identifier: "5511999999999"}
	contact.AddIdentity(identity)
	assert.Len(t, contact.Identities, 1)
	assert.Equal(t, "contact1", identity.ContactID)
}

func TestContact_GetIdentityByChannel(t *testing.T) {
	contact := NewContact("tenant1")
	contact.AddIdentity(&ContactIdentity{ChannelType: "whatsapp", Identifier: "5511999999999"})
	contact.AddIdentity(&ContactIdentity{ChannelType: "telegram", Identifier: "@user"})

	wa := contact.GetIdentityByChannel("whatsapp")
	assert.NotNil(t, wa)
	assert.Equal(t, "5511999999999", wa.Identifier)

	missing := contact.GetIdentityByChannel("email")
	assert.Nil(t, missing)
}

// Um contato acumula uma identidade por sessão do webchat; a resposta tem de ir
// para a mais recente, independentemente da ordem em que foram carregadas.
func TestContact_GetIdentityByChannel_NewestWins(t *testing.T) {
	contact := NewContact("tenant1")
	base := time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC)
	contact.AddIdentity(&ContactIdentity{ChannelType: "webchat", Identifier: "session-new", CreatedAt: base.Add(2 * time.Hour)})
	contact.AddIdentity(&ContactIdentity{ChannelType: "webchat", Identifier: "session-old", CreatedAt: base})
	contact.AddIdentity(&ContactIdentity{ChannelType: "whatsapp", Identifier: "5511999999999", CreatedAt: base.Add(3 * time.Hour)})

	assert.Equal(t, "session-new", contact.GetIdentityByChannel("webchat").Identifier)
	assert.Equal(t, "5511999999999", contact.GetIdentityByChannel("whatsapp").Identifier)
}

func TestContact_HasTag(t *testing.T) {
	contact := NewContact("tenant1")
	contact.Tags = []string{"vip", "premium"}

	assert.True(t, contact.HasTag("vip"))
	assert.True(t, contact.HasTag("premium"))
	assert.False(t, contact.HasTag("basic"))
}

func TestContact_Block(t *testing.T) {
	contact := NewContact("tenant1")
	contact.Block()

	assert.Equal(t, "true", contact.CustomFields["_blocked"])
	assert.NotEmpty(t, contact.CustomFields["_blocked_at"])
	assert.True(t, contact.IsBlocked())
}

func TestContact_Unblock(t *testing.T) {
	contact := NewContact("tenant1")
	contact.Block()
	assert.True(t, contact.IsBlocked())

	contact.Unblock()
	assert.False(t, contact.IsBlocked())
	_, hasBlocked := contact.CustomFields["_blocked"]
	assert.False(t, hasBlocked)
	_, hasBlockedAt := contact.CustomFields["_blocked_at"]
	assert.False(t, hasBlockedAt)
}

func TestContact_IsBlocked_Default(t *testing.T) {
	contact := NewContact("tenant1")
	assert.False(t, contact.IsBlocked())

	// Also test with nil CustomFields
	contact2 := &Contact{}
	assert.False(t, contact2.IsBlocked())
}

func TestContact_GetBlockedAt(t *testing.T) {
	contact := NewContact("tenant1")

	// Not blocked - should return nil
	assert.Nil(t, contact.GetBlockedAt())

	// Block and check
	contact.Block()
	blockedAt := contact.GetBlockedAt()
	assert.NotNil(t, blockedAt)

	// Nil CustomFields
	contact2 := &Contact{}
	assert.Nil(t, contact2.GetBlockedAt())
}
