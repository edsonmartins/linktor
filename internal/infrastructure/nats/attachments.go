package nats

// Chaves de metadata que carregam o conteúdo de um anexo inline (base64) até
// que ele seja re-hospedado no object store. É o contrato entre os adaptadores
// que recebem bytes (WhatsApp embutido, e-mail, bridge) e rehostInboundMedia.
const (
	AttachmentDataKey         = "data"
	AttachmentDataEncodingKey = "data_encoding"
	AttachmentDataBase64      = "base64"
)

// TrimInlineAttachments tira os bytes inline dos anexos que não cabem no
// orçamento, na ordem em que vieram, para a mensagem caber no limite de payload
// do NATS. Quem perde os bytes mantém nome e tipo. Devolve os nomes dos anexos
// esvaziados, para o chamador registrar.
func TrimInlineAttachments(attachments []AttachmentData, budget int) []string {
	remaining := budget
	var dropped []string
	for i := range attachments {
		a := &attachments[i]
		data := a.Metadata[AttachmentDataKey]
		if data == "" {
			continue
		}
		if len(data) <= remaining {
			remaining -= len(data)
			continue
		}
		delete(a.Metadata, AttachmentDataKey)
		delete(a.Metadata, AttachmentDataEncodingKey)
		dropped = append(dropped, a.Filename)
	}
	return dropped
}
