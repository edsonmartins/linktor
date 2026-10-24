package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTemplates is the local template registry the precheck reads.
type fakeTemplates map[string]entity.TemplateStatus

func (f fakeTemplates) FindByName(_ context.Context, _, _, name, language string) (*entity.Template, error) {
	status, ok := f[name+"/"+language]
	if !ok {
		return nil, nil
	}
	return &entity.Template{Name: name, Language: language, Status: status}, nil
}

// ---------------------------------------------------------------------------
// content_type "template" on the direct route
// ---------------------------------------------------------------------------

func TestDirectSend_TemplateGoesOutAsTemplateMetadata(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsAppOfficial)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id":   "channel-1",
		"to":           "+5511999999999",
		"content_type": "template",
		"template": map[string]interface{}{
			"name": "boas_vindas", "language": "pt_BR",
			"components": []interface{}{map[string]interface{}{"type": "body", "parameters": []interface{}{map[string]string{"type": "text", "text": "Ana"}}}},
		},
	})

	require.Equal(t, http.StatusAccepted, w.Code, "%+v", resp.Error)
	data := directSendData(t, resp)
	message := d.messages.Messages[data["id"].(string)]
	require.NotNil(t, message)
	assert.Equal(t, entity.ContentTypeTemplate, message.ContentType)
	assert.Equal(t, "template:boas_vindas", message.Content)
	assert.Equal(t, "boas_vindas", message.Metadata["template_name"])
	assert.Equal(t, "pt_BR", message.Metadata["template_language"])
	var components []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(message.Metadata["template_components"]), &components))
	assert.Equal(t, "body", components[0]["type"])

	require.Len(t, d.producer.OutboundMessages, 1)
	out := d.producer.OutboundMessages[0]
	assert.Equal(t, "template", out.ContentType)
	assert.Equal(t, "boas_vindas", out.Metadata["template_name"])
}

func TestDirectSend_TemplateRequiresNameAndLanguage(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsAppOfficial)

	for _, tpl := range []interface{}{nil, map[string]string{"name": "x"}, map[string]string{"language": "pt_BR"}} {
		body := map[string]interface{}{"channel_id": "channel-1", "to": "+5511999999999", "content_type": "template"}
		if tpl != nil {
			body["template"] = tpl
		}
		w, resp := postDirectSend(d, body)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "VALIDATION_ERROR", errorCode(resp))
	}
	assert.Empty(t, d.producer.OutboundMessages)
}

// ---------------------------------------------------------------------------
// precheck: distinct synchronous refusals on whatsapp_official
// ---------------------------------------------------------------------------

func TestDirectSend_PrecheckRefusalsAreDistinctAndWriteNothing(t *testing.T) {
	cases := []struct {
		name   string
		body   map[string]interface{}
		inbox  time.Duration
		status int
		code   string
	}{
		{"número inválido", map[string]interface{}{"to": "não-é-telefone", "text": "oi"}, time.Hour, http.StatusBadRequest, "INVALID_RECIPIENT"},
		{"fora da janela", map[string]interface{}{"to": "+5511988887777", "text": "oi"}, 25 * time.Hour, http.StatusUnprocessableEntity, "OUTSIDE_WINDOW"},
		{"sem mensagem recebida", map[string]interface{}{"to": "+5511988887777", "text": "oi"}, -1, http.StatusUnprocessableEntity, "OUTSIDE_WINDOW"},
		{"template não aprovado", map[string]interface{}{"to": "+5511988887777", "content_type": "template",
			"template": map[string]string{"name": "pausado", "language": "pt_BR"}}, 25 * time.Hour, http.StatusUnprocessableEntity, "TEMPLATE_NOT_APPROVED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := setupDirectSend()
			d.handler.messageService.SetTemplateStatusLookup(fakeTemplates{"pausado/pt_BR": entity.TemplateStatusPaused})
			seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsAppOfficial)
			seedWindowContact(d, "channel-1", tc.inbox)
			tc.body["channel_id"] = "channel-1"
			tc.body["precheck"] = true

			w, resp := postDirectSend(d, tc.body)

			assert.Equal(t, tc.status, w.Code)
			assert.Equal(t, tc.code, errorCode(resp))
			assert.Empty(t, d.producer.OutboundMessages)
			assert.Len(t, d.convos.Conversations, 1, "nada criado")
		})
	}
}

func TestDirectSend_PrecheckLetsThroughWhatIsAllowed(t *testing.T) {
	cases := []struct {
		name  string
		body  map[string]interface{}
		inbox time.Duration
	}{
		{"texto na janela", map[string]interface{}{"text": "oi"}, time.Hour},
		{"template aprovado fora da janela", map[string]interface{}{"content_type": "template",
			"template": map[string]string{"name": "aprovado", "language": "pt_BR"}}, 25 * time.Hour},
		{"template sem registro local (Meta decide)", map[string]interface{}{"content_type": "template",
			"template": map[string]string{"name": "desconhecido", "language": "pt_BR"}}, 25 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := setupDirectSend()
			d.handler.messageService.SetTemplateStatusLookup(fakeTemplates{"aprovado/pt_BR": entity.TemplateStatusApproved})
			seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsAppOfficial)
			seedWindowContact(d, "channel-1", tc.inbox)
			tc.body["channel_id"] = "channel-1"
			tc.body["to"] = "+5511988887777"
			tc.body["precheck"] = true

			w, resp := postDirectSend(d, tc.body)
			assert.Equal(t, http.StatusAccepted, w.Code, "%+v", resp.Error)
		})
	}
}

func TestDirectSend_WithoutPrecheckNothingChanges(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsAppOfficial)
	seedWindowContact(d, "channel-1", 25*time.Hour)

	w, _ := postDirectSend(d, map[string]interface{}{"channel_id": "channel-1", "to": "+5511988887777", "text": "oi"})
	assert.Equal(t, http.StatusAccepted, w.Code, "sem precheck, o guard de saída decide depois, como antes")

	d2 := setupDirectSend()
	seedConnectedChannel(d2.channels, "channel-2", "tenant-1", entity.ChannelTypeWhatsApp)
	w2, _ := postDirectSend(d2, map[string]interface{}{"channel_id": "channel-2", "to": "+5511988887777", "text": "oi", "precheck": true})
	assert.Equal(t, http.StatusAccepted, w2.Code, "canal sem a regra da janela não é afetado pelo precheck")
	w3, resp3 := postDirectSend(d2, map[string]interface{}{"channel_id": "channel-2", "to": "120363000000000000@g.us", "text": "oi", "precheck": true})
	assert.Equal(t, http.StatusAccepted, w3.Code, "nem a checagem de número: grupo no canal não oficial segue (%+v)", resp3.Error)
}
