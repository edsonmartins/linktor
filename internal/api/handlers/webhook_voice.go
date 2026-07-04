package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/msgfy/linktor/internal/adapters/sms"
	"github.com/msgfy/linktor/internal/adapters/voice"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// VoiceWebhook ingests Twilio voice call events (call lifecycle, DTMF, speech,
// recordings, transcriptions) and surfaces them as inbound messages so a call
// becomes part of the caller's conversation. It answers the initial inbound call
// synchronously with a static TwiML greeting + record — the homologation baseline
// (call activity is logged; a dynamic multi-step IVR flow engine is out of scope).
//
// Signature validation reuses the proven Twilio HMAC-SHA1 check (same as the SMS
// handler) rather than the voice adapter's own validator.
func (h *WebhookHandler) VoiceWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	if !h.twilioSignatureOK(c, channel, body) {
		return
	}

	// Reuse the shared Twilio voice parser to normalize the form into a
	// WebhookEvent (stateless — one instance is reused across requests).
	headers := map[string]string{
		"X-Twilio-Signature": c.GetHeader("X-Twilio-Signature"),
	}
	event, err := voiceTwilioParser.ParseWebhook(c.Request.Context(), headers, body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Bridge the call event into the conversation as an inbound message. Some
	// events (e.g. intermediate in-progress status) carry no user-facing content
	// and are skipped.
	if inbound := h.voiceInboundFromEvent(channel, event); inbound != nil {
		if err := h.publishInbound(c.Request.Context(), inbound); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process call event"})
			return
		}
	}

	// Answer the initial inbound call with a greeting + record; every other
	// callback (recording/transcription/status) gets an empty TwiML ack.
	c.Header("Content-Type", "text/xml")
	if isInitialInboundCall(event) {
		c.String(http.StatusOK, voiceGreetingTwiML(channel))
		return
	}
	c.String(http.StatusOK, sms.EmptyTwiMLResponse())
}

// voiceTwilioParser is a shared, stateless parser reused across voice webhook
// requests so each callback doesn't allocate a provider + http.Client.
var voiceTwilioParser = voice.NewTwilioProvider()

// twilioSignatureOK validates the Twilio HMAC-SHA1 signature for the request and
// is shared by the SMS and voice webhooks (both are Twilio). It returns true if
// the handler may proceed; otherwise it has already written a 401. When no
// auth_token is configured it falls back to the requireWebhookSecrets policy.
func (h *WebhookHandler) twilioSignatureOK(c *gin.Context, channel *entity.Channel, body []byte) bool {
	authToken := channel.Credentials["auth_token"]
	if authToken != "" {
		values, _ := url.ParseQuery(string(body))
		if !sms.ValidateSignature(authToken, requestURL(c), firstValues(values), c.GetHeader("X-Twilio-Signature")) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return false
		}
		return true
	}
	if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return false
	}
	return true
}

// isInitialInboundCall reports whether the event is the first webhook for a new
// inbound call — the moment Twilio expects TwiML to drive the answer.
func isInitialInboundCall(e *voice.WebhookEvent) bool {
	if e.Type != "status" || e.Direction != voice.CallDirectionInbound {
		return false
	}
	switch e.Status {
	case voice.CallStatusInitiated, voice.CallStatusRinging, voice.CallStatusInProgress, voice.CallStatusAnswered:
		return true
	default:
		return false
	}
}

// voiceInboundFromEvent maps a call event to an inbound message, or nil to skip.
func (h *WebhookHandler) voiceInboundFromEvent(channel *entity.Channel, e *voice.WebhookEvent) *nats.InboundMessage {
	contentType := "text"
	var content, eventType string
	var attachments []nats.AttachmentData

	switch e.Type {
	case "dtmf":
		content, eventType = e.Digits, "dtmf"
	case "speech":
		content, eventType = e.SpeechResult, "speech"
	case "recording":
		contentType, eventType = "audio", "recording"
		content = "Voice recording"
		if e.RecordingURL != "" {
			attachments = append(attachments, nats.AttachmentData{
				Type: "audio", URL: e.RecordingURL, MimeType: "audio/mpeg",
			})
		}
	case "transcription":
		content, eventType = e.Transcription, "transcription"
	case "status":
		content, eventType = voiceStatusMessage(e)
		if content == "" {
			return nil // intermediate status with nothing to log
		}
	default:
		return nil
	}

	// The caller id keys the conversation; without it we'd create a junk
	// empty-identifier contact, so skip events that carry no caller.
	if e.From == "" {
		return nil
	}

	// The inbound pipeline dedups on (conversation, external_id). Every webhook of
	// one call shares the CallSid, so qualify it with the event type: distinct
	// events (recording/transcription/ended) each persist, while the several
	// lifecycle callbacks that all mean "call started" collapse to a single
	// message. DTMF/speech can legitimately repeat within a call, so include the
	// input value to keep each entry.
	externalID := e.CallID + ":" + eventType
	switch eventType {
	case "dtmf":
		externalID += ":" + e.Digits
	case "speech":
		externalID += ":" + e.SpeechResult
	}

	metadata := map[string]string{
		"sender_id":   e.From,
		"from":        e.From,
		"to":          e.To,
		"call_id":     e.CallID,
		"event_type":  eventType,
		"call_status": string(e.Status),
	}
	if e.Duration > 0 {
		metadata["duration"] = fmt.Sprintf("%d", e.Duration)
	}

	return &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "voice",
		ExternalID:  externalID,
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   e.Timestamp,
	}
}

// voiceStatusMessage renders a human-readable line for a call lifecycle status,
// or "" for statuses that should not produce a message.
func voiceStatusMessage(e *voice.WebhookEvent) (content, eventType string) {
	switch e.Status {
	case voice.CallStatusInitiated, voice.CallStatusRinging, voice.CallStatusAnswered, voice.CallStatusInProgress:
		return "📞 Incoming voice call", "call_started"
	case voice.CallStatusCompleted:
		if e.Duration > 0 {
			return fmt.Sprintf("Voice call ended (%ds)", e.Duration), "call_ended"
		}
		return "Voice call ended", "call_ended"
	case voice.CallStatusBusy, voice.CallStatusNoAnswer, voice.CallStatusFailed, voice.CallStatusCanceled:
		return fmt.Sprintf("Voice call %s", e.Status), "call_status"
	default:
		return "", ""
	}
}

// voiceGreetingTwiML answers an inbound call with a greeting and records the
// caller (Twilio posts the recording + transcription back to this webhook).
func voiceGreetingTwiML(channel *entity.Channel) string {
	greeting := channel.Config["voice_greeting"]
	if greeting == "" {
		greeting = "Hello. Please leave a message after the tone."
	}
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<Response><Say>` + xmlEscape(greeting) + `</Say>` +
		`<Record maxLength="120" transcribe="true" playBeep="true"/></Response>`
}

// xmlEscape escapes the minimal set of characters unsafe inside an XML text node.
// A single-pass replacer avoids double-escaping the ampersand.
var xmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func xmlEscape(s string) string { return xmlEscaper.Replace(s) }
