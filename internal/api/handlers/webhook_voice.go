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

	authToken := channel.Credentials["auth_token"]
	if authToken != "" {
		values, _ := url.ParseQuery(string(body))
		if !sms.ValidateSignature(authToken, requestURL(c), firstValues(values), c.GetHeader("X-Twilio-Signature")) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	} else if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return
	}

	// Reuse the Twilio voice parser to normalize the form into a WebhookEvent.
	headers := map[string]string{
		"X-Twilio-Signature": c.GetHeader("X-Twilio-Signature"),
	}
	event, err := voice.NewTwilioProvider().ParseWebhook(c.Request.Context(), headers, body)
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
	c.String(http.StatusOK, emptyVoiceTwiML())
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
		ExternalID:  e.CallID,
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

func emptyVoiceTwiML() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`
}

// xmlEscape escapes the minimal set of characters unsafe inside an XML text node.
// A single-pass replacer avoids double-escaping the ampersand.
var xmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func xmlEscape(s string) string { return xmlEscaper.Replace(s) }
