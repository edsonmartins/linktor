package entity

import (
	"testing"

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

func TestContact_HasTag(t *testing.T) {
	contact := NewContact("tenant1")
	contact.Tags = []string{"vip", "premium"}

	assert.True(t, contact.HasTag("vip"))
	assert.True(t, contact.HasTag("premium"))
	assert.False(t, contact.HasTag("basic"))
}
