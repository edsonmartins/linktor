package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
)

func newMediaHandler(t *testing.T) (*Handler, *testutil.MockProducer) {
	t.Helper()
	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-1"] = &entity.Channel{ID: "ch-1", TenantID: "t-1", Type: entity.ChannelType(plugin.ChannelTypeWhatsAppRemote)}
	producer := testutil.NewMockProducer()
	return NewHandler(NewHub(), channels, producer, nil, nil), producer
}

func inboundRaw(t *testing.T, evt InboundEvent) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(evt)
	require.NoError(t, err)
	return raw
}

// Uma foto sem legenda vira mensagem de imagem com o anexo inline no contrato
// que rehostInboundMedia entende; o rehost registrado roda antes do publish e,
// tendo movido os bytes para o store, nada inline chega ao NATS.
func TestHandleInbound_MediaRehostedBeforePublish(t *testing.T) {
	h, producer := newMediaHandler(t)
	data := base64.StdEncoding.EncodeToString([]byte("jpegbytes"))

	var sawInline bool
	h.SetRehost(func(_ context.Context, msg *nats.InboundMessage) {
		att := &msg.Attachments[0]
		sawInline = att.Metadata[nats.AttachmentDataKey] == data
		att.URL = "http://media.test/api/v1/media/inbound/whatsapp_remote/ch-1/x.jpg"
		delete(att.Metadata, nats.AttachmentDataKey)
		delete(att.Metadata, nats.AttachmentDataEncodingKey)
	})

	require.NoError(t, h.handleInbound("ch-1", inboundRaw(t, InboundEvent{
		Type: "message", ID: "M1", From: "5511999999999@s.whatsapp.net", Chat: "5511999999999@s.whatsapp.net",
		ContentType: "image", Timestamp: 1,
		Attachments: []InboundAttachment{{Type: "image", MimeType: "image/jpeg", Filename: "foto.jpg", Data: data}},
	})))

	require.Len(t, producer.InboundMessages, 1)
	in := producer.InboundMessages[0]
	assert.True(t, sawInline, "rehost tem de receber o base64")
	assert.Equal(t, "image", in.ContentType)
	assert.Equal(t, "", in.Content)
	require.Len(t, in.Attachments, 1)
	assert.Equal(t, "http://media.test/api/v1/media/inbound/whatsapp_remote/ch-1/x.jpg", in.Attachments[0].URL)
	assert.Equal(t, int64(len("jpegbytes")), in.Attachments[0].SizeBytes)
	assert.Empty(t, in.Attachments[0].Metadata[nats.AttachmentDataKey])
}

// Sem object store o inline só segue se couber no orçamento do NATS; áudio de
// voz mantém msg_type ptt e figurinha vira imagem marcada.
func TestHandleInbound_InlineBudgetAndVariants(t *testing.T) {
	h, producer := newMediaHandler(t)
	small := base64.StdEncoding.EncodeToString([]byte("ogg"))
	big := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("v", maxInlineBytes+1)))

	require.NoError(t, h.handleInbound("ch-1", inboundRaw(t, InboundEvent{
		ID: "M2", From: "5511999999999@s.whatsapp.net", ContentType: "audio", MsgType: "ptt", Timestamp: 1,
		Attachments: []InboundAttachment{{Type: "audio", MimeType: "audio/ogg; codecs=opus", Data: small}},
	})))
	require.NoError(t, h.handleInbound("ch-1", inboundRaw(t, InboundEvent{
		ID: "M3", From: "5511999999999@s.whatsapp.net", ContentType: "video", Text: "veja", Timestamp: 1,
		Attachments: []InboundAttachment{{Type: "video", MimeType: "video/mp4", Filename: "v.mp4", Data: big}},
	})))
	require.NoError(t, h.handleInbound("ch-1", inboundRaw(t, InboundEvent{
		ID: "M4", From: "5511999999999@s.whatsapp.net", ContentType: "image", MsgType: "sticker", Timestamp: 1,
		Attachments: []InboundAttachment{{Type: "image", MimeType: "image/webp", Data: small}},
	})))
	require.NoError(t, h.handleInbound("ch-1", inboundRaw(t, InboundEvent{
		ID: "M5", From: "5511999999999@s.whatsapp.net", ContentType: "document", Timestamp: 1,
		Attachments: []InboundAttachment{{Type: "document", MimeType: "application/pdf", Filename: "grande.pdf", SizeBytes: 90 << 20, Error: "attachment exceeds 32 MB"}},
	})))

	require.Len(t, producer.InboundMessages, 4)
	voice, video, sticker, doc := producer.InboundMessages[0], producer.InboundMessages[1], producer.InboundMessages[2], producer.InboundMessages[3]

	assert.Equal(t, "ptt", voice.Metadata["msg_type"])
	assert.Equal(t, small, voice.Attachments[0].Metadata[nats.AttachmentDataKey])

	assert.Equal(t, "veja", video.Content)
	assert.Empty(t, video.Attachments[0].Metadata[nats.AttachmentDataKey], "acima do orçamento sem store: só o registro")
	assert.Equal(t, "v.mp4", video.Attachments[0].Filename)

	assert.Equal(t, "true", sticker.Metadata["is_sticker"])

	assert.Equal(t, "attachment exceeds 32 MB", doc.Attachments[0].Metadata["download_error"])
	assert.Equal(t, int64(90<<20), doc.Attachments[0].SizeBytes)
	assert.Empty(t, doc.Attachments[0].Metadata[nats.AttachmentDataKey])
}

// Evento sem texto e sem anexo (recibo, conexão) continua ignorado.
func TestHandleInbound_IgnoresEmptyEvents(t *testing.T) {
	h, producer := newMediaHandler(t)
	require.NoError(t, h.handleInbound("ch-1", inboundRaw(t, InboundEvent{ID: "R1", Type: "receipt", From: "x@s.whatsapp.net"})))
	assert.Empty(t, producer.InboundMessages)
}
