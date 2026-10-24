package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedWindowContact puts a whatsapp_official contact with an open conversation
// whose last inbound message happened `ago` before now (ago < 0: no inbound).
func seedWindowContact(d *directSendDeps, channelID string, ago time.Duration) {
	contact := seedContact(d.contacts, "contact-w", "tenant-1", "Cliente", "", "5511988887777")
	identity := &entity.ContactIdentity{ID: "identity-w", ContactID: contact.ID, TenantID: "tenant-1",
		ChannelType: string(entity.ChannelTypeWhatsAppOfficial), Identifier: "5511988887777"}
	contact.Identities = []*entity.ContactIdentity{identity}
	d.contacts.Identities[contact.ID] = []*entity.ContactIdentity{identity}
	d.convos.Conversations["conv-w"] = &entity.Conversation{ID: "conv-w", TenantID: "tenant-1", ContactID: contact.ID,
		ChannelID: channelID, Status: entity.ConversationStatusOpen, CreatedAt: time.Now()}
	if ago >= 0 {
		d.messages.Messages["in-1"] = &entity.Message{ID: "in-1", ConversationID: "conv-w", SenderType: entity.SenderTypeContact,
			Content: "oi", CreatedAt: time.Now().Add(-ago)}
	}
}

func errorCode(resp Response) string {
	if resp.Error == nil {
		return ""
	}
	return resp.Error.Code
}

// ---------------------------------------------------------------------------
// GET /api/v1/channels/:id/window
// ---------------------------------------------------------------------------

func getWindow(d *directSendDeps, scopes []string, channelID, to string) (*httptest.ResponseRecorder, Response) {
	gin.SetMode(gin.TestMode)
	auth := middleware.NewAuthMiddleware(nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Set("user_id", "apikey:key-1")
		c.Set(middleware.ScopesKey, scopes)
		c.Next()
	})
	r.GET("/api/v1/channels/:id/window", auth.RequireScope(middleware.ScopeMessagesSend), d.handler.Window)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+channelID+"/window?to="+to, nil))
	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w, resp
}

func TestWindow_OpenWithClosesAt(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsAppOfficial)
	seedWindowContact(d, "channel-1", 2*time.Hour)

	w, resp := getWindow(d, []string{middleware.ScopeMessagesSend}, "channel-1", "%2B5511988887777")

	require.Equal(t, http.StatusOK, w.Code, "%+v", resp.Error)
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, true, data["applies"])
	assert.Equal(t, true, data["open"])
	assert.Equal(t, "5511988887777", data["to"])
	last, err := time.Parse(time.RFC3339Nano, data["last_inbound_at"].(string))
	require.NoError(t, err)
	closes, err := time.Parse(time.RFC3339Nano, data["closes_at"].(string))
	require.NoError(t, err)
	assert.Equal(t, 24*time.Hour, closes.Sub(last))
}

func TestWindow_ClosedCases(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsAppOfficial)
	seedWindowContact(d, "channel-1", 25*time.Hour)

	_, resp := getWindow(d, []string{middleware.ScopeMessagesSend}, "channel-1", "5511988887777")
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, false, data["open"], "última recebida há 25h")
	assert.NotNil(t, data["closes_at"])

	_, resp = getWindow(d, []string{middleware.ScopeMessagesSend}, "channel-1", "5511900000000")
	data = resp.Data.(map[string]interface{})
	assert.Equal(t, true, data["applies"])
	assert.Equal(t, false, data["open"], "contato desconhecido: nada recebido")
	assert.Nil(t, data["last_inbound_at"])
	assert.Len(t, d.contacts.Contacts, 1, "consulta não cria contato")
}

func TestWindow_ChannelWithoutTheRuleAndErrors(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	seedConnectedChannel(d.channels, "channel-x", "tenant-2", entity.ChannelTypeWhatsAppOfficial)

	_, resp := getWindow(d, []string{middleware.ScopeMessagesSend}, "channel-1", "5511988887777")
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, false, data["applies"])
	assert.Equal(t, true, data["open"])

	w, resp := getWindow(d, []string{middleware.ScopeMessagesSend}, "channel-x", "5511988887777")
	assert.Equal(t, http.StatusNotFound, w.Code, "canal de outro tenant")
	assert.Equal(t, "CHANNEL_NOT_FOUND", errorCode(resp))

	w, _ = getWindow(d, []string{middleware.ScopeMessagesSend}, "channel-1", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w, _ = getWindow(d, []string{"contacts:read"}, "channel-1", "5511988887777")
	assert.Equal(t, http.StatusForbidden, w.Code)
}
