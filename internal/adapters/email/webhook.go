package email

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// maxFormBytes é o teto do formulário de um webhook de entrada em memória; o
// que passar disso o Go grava em disco e RemoveAll apaga.
const maxFormBytes = 32 << 20

// ParseFormBody lê o corpo de um webhook de formulário, urlencoded ou
// multipart. O form só vem no multipart (é onde os anexos estão) e quem chama
// deve RemoveAll nele. O Mailgun posta a entrada em multipart, e o
// url.ParseQuery que havia aqui recusava o corpo inteiro: 400 na mensagem e,
// na assinatura, 401.
func ParseFormBody(body []byte, contentType string) (url.Values, *multipart.Form, error) {
	tipo, params, _ := mime.ParseMediaType(contentType)
	if tipo == "multipart/form-data" {
		form, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(maxFormBytes)
		if err != nil {
			return nil, nil, err
		}
		return url.Values(form.Value), form, nil
	}
	values, err := url.ParseQuery(string(body))
	return values, nil, err
}

// anexosDoForm lê os arquivos cujo campo começa com o prefixo (attachment-1,
// attachment-2… no Mailgun), em ordem de nome.
func anexosDoForm(form *multipart.Form, prefixo string) []*Attachment {
	if form == nil {
		return nil
	}
	campos := make([]string, 0, len(form.File))
	for campo := range form.File {
		if strings.HasPrefix(campo, prefixo) {
			campos = append(campos, campo)
		}
	}
	sort.Strings(campos)

	var saida []*Attachment
	for _, campo := range campos {
		for _, fh := range form.File[campo] {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			dados, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				continue
			}
			saida = append(saida, &Attachment{
				Filename:    fh.Filename,
				ContentType: fh.Header.Get("Content-Type"),
				Content:     dados,
				Size:        int64(len(dados)),
			})
		}
	}
	return saida
}

// ParseWebhook parses an incoming webhook from any supported provider
func ParseWebhook(provider Provider, body []byte, headers map[string]string) (*WebhookPayload, error) {
	switch provider {
	case ProviderSendGrid:
		return parseSendGridWebhook(body, headers)
	case ProviderMailgun:
		return parseMailgunWebhook(body, headers)
	case ProviderSES:
		return parseSESWebhook(body, headers)
	case ProviderPostmark:
		return parsePostmarkWebhook(body, headers)
	default:
		return nil, fmt.Errorf("unsupported provider for webhooks: %s", provider)
	}
}

// parseSendGridWebhook parses SendGrid webhook payloads
func parseSendGridWebhook(body []byte, headers map[string]string) (*WebhookPayload, error) {
	contentType := headers["Content-Type"]

	// Check if this is an inbound parse webhook (form data) or event webhook (JSON)
	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return parseSendGridInbound(body)
	}

	return parseSendGridEvents(body)
}

// parseSendGridInbound parses SendGrid Inbound Parse webhook
func parseSendGridInbound(body []byte) (*WebhookPayload, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse form data: %w", err)
	}

	inbound := &SendGridInboundWebhook{
		Headers:   values.Get("headers"),
		To:        values.Get("to"),
		From:      values.Get("from"),
		Subject:   values.Get("subject"),
		Text:      values.Get("text"),
		HTML:      values.Get("html"),
		SPF:       values.Get("SPF"),
		DKIM:      values.Get("dkim"),
		SpamScore: values.Get("spam_score"),
		Envelope:  values.Get("envelope"),
	}

	if attachments := values.Get("attachments"); attachments != "" {
		inbound.Attachments, _ = strconv.Atoi(attachments)
	}

	// Convert to IncomingEmail
	email := &IncomingEmail{
		From:       inbound.From,
		To:         parseAddressList(inbound.To),
		Subject:    inbound.Subject,
		TextBody:   inbound.Text,
		HTMLBody:   inbound.HTML,
		ReceivedAt: time.Now(),
		Headers:    parseRawHeaders(inbound.Headers),
		Metadata:   make(map[string]string),
	}

	if inbound.SpamScore != "" {
		if score, err := strconv.ParseFloat(inbound.SpamScore, 64); err == nil {
			email.SpamScore = score
		}
	}

	email.Metadata["spf"] = inbound.SPF
	email.Metadata["dkim"] = inbound.DKIM

	return &WebhookPayload{
		Provider:      ProviderSendGrid,
		Type:          "inbound",
		IncomingEmail: email,
		RawPayload:    body,
	}, nil
}

// parseSendGridEvents parses SendGrid Event Webhook
func parseSendGridEvents(body []byte) (*WebhookPayload, error) {
	var events []SendGridEventWebhook
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, fmt.Errorf("failed to parse SendGrid events: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no events in payload")
	}

	// Process first event (typically webhooks send one event at a time)
	event := events[0]

	status := mapSendGridEventToStatus(event.Event)

	callback := &StatusCallback{
		ExternalID:   event.MessageID,
		Status:       status,
		Recipient:    event.Email,
		ErrorMessage: event.Reason,
		Timestamp:    time.Unix(event.Timestamp, 0),
		Metadata:     make(map[string]string),
	}

	if event.Response != "" {
		callback.Metadata["response"] = event.Response
	}
	if event.URL != "" {
		callback.Metadata["url"] = event.URL
	}

	return &WebhookPayload{
		Provider:       ProviderSendGrid,
		Type:           "status",
		StatusCallback: callback,
		RawPayload:     body,
	}, nil
}

// mapSendGridEventToStatus maps SendGrid event type to EmailStatus
func mapSendGridEventToStatus(event string) EmailStatus {
	switch event {
	case "processed":
		return StatusQueued
	case "dropped":
		return StatusFailed
	case "delivered":
		return StatusDelivered
	case "deferred":
		return StatusQueued
	case "bounce":
		return StatusBounced
	case "open":
		return StatusOpened
	case "click":
		return StatusClicked
	case "spam_report":
		return StatusSpam
	case "unsubscribe":
		return StatusUnsubscribed
	default:
		return StatusSent
	}
}

// parseMailgunWebhook parses Mailgun webhook payloads
func parseMailgunWebhook(body []byte, headers map[string]string) (*WebhookPayload, error) {
	contentType := headers["Content-Type"]

	// Check if this is an inbound webhook (form data) or event webhook (JSON)
	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return parseMailgunInbound(body, contentType)
	}

	return parseMailgunEvents(body)
}

// parseMailgunInbound parses Mailgun Routes webhook (inbound)
func parseMailgunInbound(body []byte, contentType string) (*WebhookPayload, error) {
	values, form, err := ParseFormBody(body, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse form data: %w", err)
	}
	if form != nil {
		defer form.RemoveAll()
	}

	inbound := &MailgunInboundWebhook{
		Recipient:   values.Get("recipient"),
		Sender:      values.Get("sender"),
		From:        values.Get("from"),
		Subject:     values.Get("subject"),
		BodyPlain:   values.Get("body-plain"),
		BodyHTML:    values.Get("body-html"),
		MessageID:   values.Get("Message-Id"),
		Timestamp:   values.Get("timestamp"),
		Token:       values.Get("token"),
		Signature:   values.Get("signature"),
		ContentType: values.Get("Content-Type"),
	}

	// O Mailgun manda o texto duas vezes: body-plain inteiro e stripped-text,
	// sem a citação do email anterior nem a assinatura. Para o chat, o segundo.
	// Os cabeçalhos do email chegam como campos com o nome original.
	from, nome := splitAddress(primeiro(inbound.From, inbound.Sender))
	email := &IncomingEmail{
		MessageID:   strings.Trim(inbound.MessageID, "<>"),
		From:        from,
		FromName:    nome,
		To:          []string{inbound.Recipient},
		Subject:     inbound.Subject,
		TextBody:    primeiro(values.Get("stripped-text"), inbound.BodyPlain),
		HTMLBody:    inbound.BodyHTML,
		InReplyTo:   strings.Trim(values.Get("In-Reply-To"), "<>"),
		References:  values.Get("References"),
		Attachments: anexosDoForm(form, "attachment-"),
		ReceivedAt:  time.Now(),
		Headers:     make(map[string]string),
		Metadata: map[string]string{
			"sender": inbound.Sender,
		},
	}

	if inbound.Timestamp != "" {
		if ts, err := strconv.ParseInt(inbound.Timestamp, 10, 64); err == nil {
			email.ReceivedAt = time.Unix(ts, 0)
		}
	}

	return &WebhookPayload{
		Provider:      ProviderMailgun,
		Type:          "inbound",
		IncomingEmail: email,
		RawPayload:    body,
	}, nil
}

// parseMailgunEvents parses Mailgun Event webhook
func parseMailgunEvents(body []byte) (*WebhookPayload, error) {
	var event MailgunEventWebhook
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse Mailgun event: %w", err)
	}

	// Falha temporária: o próprio Mailgun vai tentar de novo; não há status a
	// aplicar na mensagem.
	if event.EventData.Event == "failed" && event.EventData.Severity == "temporary" {
		return &WebhookPayload{Provider: ProviderMailgun, Type: "status", RawPayload: body}, nil
	}

	status := mapMailgunEventToStatus(event.EventData.Event)

	// O que casa com a mensagem enviada é o Message-Id do cabeçalho (o mesmo
	// que o envio guardou, sem colchetes). Antes ele ia em MessageID, que o
	// consumidor trata como id interno, e o id do evento ia em ExternalID:
	// nenhum dos dois achava a mensagem, e Entregue/Lido nunca apareciam.
	callback := &StatusCallback{
		ExternalID: strings.Trim(event.EventData.Message.Headers.MessageID, "<>"),
		Status:     status,
		Recipient:  event.EventData.Recipient,
		Timestamp:  time.Unix(int64(event.EventData.Timestamp), 0),
		Metadata:   map[string]string{"event_id": event.EventData.ID},
	}

	if event.EventData.Severity != "" {
		callback.Metadata["severity"] = event.EventData.Severity
	}

	return &WebhookPayload{
		Provider:       ProviderMailgun,
		Type:           "status",
		StatusCallback: callback,
		RawPayload:     body,
	}, nil
}

// mapMailgunEventToStatus maps Mailgun event type to EmailStatus
func mapMailgunEventToStatus(event string) EmailStatus {
	switch event {
	case "accepted":
		return StatusQueued
	case "delivered":
		return StatusDelivered
	case "failed":
		return StatusFailed
	case "opened":
		return StatusOpened
	case "clicked":
		return StatusClicked
	case "unsubscribed":
		return StatusUnsubscribed
	case "complained":
		return StatusSpam
	default:
		return StatusSent
	}
}

// parseSESWebhook parses AWS SES webhook (via SNS)
func parseSESWebhook(body []byte, headers map[string]string) (*WebhookPayload, error) {
	var notification SESNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		return nil, fmt.Errorf("failed to parse SNS notification: %w", err)
	}

	// Handle subscription confirmation
	if notification.Type == "SubscriptionConfirmation" {
		return &WebhookPayload{
			Provider:   ProviderSES,
			Type:       "subscription_confirmation",
			RawPayload: body,
			StatusCallback: &StatusCallback{
				Metadata: map[string]string{
					"subscribe_url": notification.SubscribeURL,
				},
			},
		}, nil
	}

	// Parse the SES message
	var sesMessage SESMessage
	if err := json.Unmarshal([]byte(notification.Message), &sesMessage); err != nil {
		return nil, fmt.Errorf("failed to parse SES message: %w", err)
	}

	callback := &StatusCallback{
		ExternalID: sesMessage.Mail.MessageId,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}

	switch sesMessage.NotificationType {
	case "Delivery":
		callback.Status = StatusDelivered
		if sesMessage.Delivery != nil && len(sesMessage.Delivery.Recipients) > 0 {
			callback.Recipient = sesMessage.Delivery.Recipients[0]
		}
	case "Bounce":
		callback.Status = StatusBounced
		if sesMessage.Bounce != nil {
			callback.Metadata["bounce_type"] = sesMessage.Bounce.BounceType
			callback.Metadata["bounce_sub_type"] = sesMessage.Bounce.BounceSubType
			if len(sesMessage.Bounce.BouncedRecipients) > 0 {
				callback.Recipient = sesMessage.Bounce.BouncedRecipients[0].EmailAddress
				callback.ErrorMessage = sesMessage.Bounce.BouncedRecipients[0].DiagnosticCode
			}
		}
	case "Complaint":
		callback.Status = StatusSpam
		if sesMessage.Complaint != nil && len(sesMessage.Complaint.ComplainedRecipients) > 0 {
			callback.Recipient = sesMessage.Complaint.ComplainedRecipients[0].EmailAddress
		}
	}

	return &WebhookPayload{
		Provider:       ProviderSES,
		Type:           "status",
		StatusCallback: callback,
		RawPayload:     body,
	}, nil
}

// parsePostmarkWebhook parses Postmark webhook payloads
func parsePostmarkWebhook(body []byte, headers map[string]string) (*WebhookPayload, error) {
	// Check if it's an inbound webhook or event webhook
	// Inbound has "From", "To", "Subject" etc.
	// Events have "RecordType"

	var probe map[string]interface{}
	if err := json.Unmarshal(body, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse Postmark webhook: %w", err)
	}

	if _, hasFrom := probe["From"]; hasFrom {
		return parsePostmarkInbound(body)
	}

	return parsePostmarkEvent(body)
}

// parsePostmarkInbound parses Postmark Inbound webhook
func parsePostmarkInbound(body []byte) (*WebhookPayload, error) {
	var inbound PostmarkInboundWebhook
	if err := json.Unmarshal(body, &inbound); err != nil {
		return nil, fmt.Errorf("failed to parse Postmark inbound: %w", err)
	}

	email := &IncomingEmail{
		MessageID:  inbound.MessageID,
		From:       inbound.From,
		FromName:   inbound.FromName,
		To:         parseAddressList(inbound.To),
		Subject:    inbound.Subject,
		TextBody:   inbound.TextBody,
		HTMLBody:   inbound.HtmlBody,
		ReceivedAt: time.Now(),
		Headers:    make(map[string]string),
		Metadata:   make(map[string]string),
	}

	if inbound.CC != "" {
		email.CC = parseAddressList(inbound.CC)
	}

	// Parse headers
	for _, h := range inbound.Headers {
		email.Headers[h.Name] = h.Value
		if strings.ToLower(h.Name) == "in-reply-to" {
			email.InReplyTo = h.Value
		}
		if strings.ToLower(h.Name) == "references" {
			email.References = h.Value
		}
	}

	// Parse attachments
	for _, att := range inbound.Attachments {
		content, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			continue
		}
		email.Attachments = append(email.Attachments, &Attachment{
			Filename:    att.Name,
			ContentType: att.ContentType,
			Content:     content,
			Size:        att.ContentLength,
			ContentID:   att.ContentID,
		})
	}

	email.Metadata["original_recipient"] = inbound.OriginalRecipient
	if inbound.ReplyTo != "" {
		email.Metadata["reply_to"] = inbound.ReplyTo
	}

	return &WebhookPayload{
		Provider:      ProviderPostmark,
		Type:          "inbound",
		IncomingEmail: email,
		RawPayload:    body,
	}, nil
}

// parsePostmarkEvent parses Postmark Event webhook
func parsePostmarkEvent(body []byte) (*WebhookPayload, error) {
	var event PostmarkEventWebhook
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse Postmark event: %w", err)
	}

	status := mapPostmarkEventToStatus(event.RecordType)

	callback := &StatusCallback{
		ExternalID: event.MessageID,
		Status:     status,
		Recipient:  event.Recipient,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}

	// Parse timestamp
	if event.DeliveredAt != "" {
		if t, err := time.Parse(time.RFC3339, event.DeliveredAt); err == nil {
			callback.Timestamp = t
		}
	} else if event.BouncedAt != "" {
		if t, err := time.Parse(time.RFC3339, event.BouncedAt); err == nil {
			callback.Timestamp = t
		}
	}

	if event.Type != "" {
		callback.Metadata["bounce_type"] = event.Type
	}
	if event.Description != "" {
		callback.ErrorMessage = event.Description
	}

	return &WebhookPayload{
		Provider:       ProviderPostmark,
		Type:           "status",
		StatusCallback: callback,
		RawPayload:     body,
	}, nil
}

// mapPostmarkEventToStatus maps Postmark record type to EmailStatus
func mapPostmarkEventToStatus(recordType string) EmailStatus {
	switch recordType {
	case "Delivery":
		return StatusDelivered
	case "Bounce":
		return StatusBounced
	case "SpamComplaint":
		return StatusSpam
	case "Open":
		return StatusOpened
	case "Click":
		return StatusClicked
	case "SubscriptionChange":
		return StatusUnsubscribed
	default:
		return StatusSent
	}
}

// parseRawHeaders parses raw email headers string into a map
func parseRawHeaders(rawHeaders string) map[string]string {
	headers := make(map[string]string)
	lines := strings.Split(rawHeaders, "\n")
	for _, line := range lines {
		if idx := strings.Index(line, ": "); idx > 0 {
			key := line[:idx]
			value := strings.TrimSpace(line[idx+2:])
			headers[key] = value
		}
	}
	return headers
}

// ValidateSendGridWebhook validates SendGrid webhook signature
func ValidateSendGridWebhook(publicKey, signature, timestamp, payload string) bool {
	// SendGrid uses public key verification
	// In production, would use proper verification
	_ = publicKey
	_ = signature
	_ = timestamp
	_ = payload
	return true
}

// ValidateMailgunWebhook validates Mailgun webhook signature
func ValidateMailgunWebhook(apiKey, token, timestamp, signature string) bool {
	data := timestamp + token
	h := hmac.New(sha256.New, []byte(apiKey))
	h.Write([]byte(data))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}

// ValidatePostmarkWebhook validates Postmark webhook (if configured)
func ValidatePostmarkWebhook(webhookPassword, authorization string) bool {
	if webhookPassword == "" {
		return true
	}
	return authorization == webhookPassword
}
