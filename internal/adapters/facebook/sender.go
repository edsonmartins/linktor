package facebook

import (
	"context"
	"errors"
	"strings"

	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for Facebook Messenger.
type SenderFactory struct{}

// NewSenderFactory creates a Facebook Messenger sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "facebook" }

// New builds a Sender from a channel's Facebook page credentials.
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	client, err := NewClient(ConfigFromMap(creds))
	if err != nil {
		return nil, err
	}
	return &fbSender{client: client}, nil
}

type fbSender struct {
	client *Client
}

func (s *fbSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	if msg.To == "" {
		return nil, outbound.Permanentf("recipient PSID is required")
	}

	mt, tag := messagingType(msg)

	switch c := msg.Content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return nil, outbound.Permanentf("text body is required")
		}
		return receiptOrErr(s.client.SendTextMessageAs(ctx, msg.To, c.Body, mt, tag))

	case outbound.Template:
		text := strings.TrimSpace(strings.Join(c.BodyParams, " "))
		if text == "" {
			return nil, outbound.Permanentf("empty template payload")
		}
		return receiptOrErr(s.client.SendTextMessageAs(ctx, msg.To, text, mt, tag))

	case outbound.Media:
		if c.URL == "" {
			return nil, outbound.Permanentf("media requires a URL")
		}
		// O Messenger não aceita texto no mesmo envio do anexo: a legenda vai
		// antes, como mensagem própria, e o recibo devolvido é o do anexo. Se o
		// anexo falhar de forma passageira, a nova tentativa repete a legenda;
		// melhor isso do que perdê-la, que era o que acontecia.
		if c.Caption != "" {
			if _, err := receiptOrErr(s.client.SendTextMessageAs(ctx, msg.To, c.Caption, mt, tag)); err != nil {
				return nil, err
			}
		}
		return receiptOrErr(s.client.SendAttachmentAs(ctx, msg.To, fbAttachmentType(c.Type), c.URL, mt, tag))

	default:
		return nil, outbound.Permanentf("unsupported content kind %q", msg.Content.Kind())
	}
}

// messagingType resolves the Meta messaging_type and optional message tag from
// the outbound message metadata, defaulting to RESPONSE (a reply inside the 24h
// window). Proactive / out-of-window sends set metadata["messaging_type"] to
// UPDATE or MESSAGE_TAG (with metadata["message_tag"]).
func messagingType(msg *outbound.Message) (string, string) {
	mt := msg.Meta("messaging_type")
	if mt == "" {
		mt = meta.MessagingTypeResponse
	}
	return mt, msg.Meta("message_tag")
}

// fbAttachmentType maps an outbound media type to the Messenger attachment type.
func fbAttachmentType(t outbound.MediaType) string {
	switch t {
	case outbound.MediaImage:
		return "image"
	case outbound.MediaVideo:
		return "video"
	case outbound.MediaAudio:
		return "audio"
	default:
		return "file"
	}
}

func receiptOrErr(resp *meta.SendMessageResponse, err error) (*outbound.Receipt, error) {
	if err != nil {
		return nil, classificarErro(err)
	}
	return &outbound.Receipt{ProviderMessageID: resp.MessageID}, nil
}

// classificarErro separa o que vale repetir do que não vale. Recusa da Graph
// que não é limite de taxa nem 5xx (token inválido, destinatário fora da
// janela, PSID errado) é permanente: repetir só deixava a mensagem "pendente"
// até a fila desistir, sem nunca virar "falhou". Rede e erro desconhecido
// seguem transitórios; o que já veio marcado permanente passa intacto.
func classificarErro(err error) error {
	if outbound.IsPermanent(err) {
		return err
	}
	var apiErr *meta.APIError
	if errors.As(err, &apiErr) && !apiErr.Transient() {
		return outbound.Permanent(err)
	}
	return err
}
