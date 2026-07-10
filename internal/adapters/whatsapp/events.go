package whatsapp

import (
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// dispatch performs a non-blocking send of an event onto the client's event
// channel. If the channel buffer is full it logs a warning instead of silently
// dropping the event, so back-pressure/consumer stalls become visible instead
// of manifesting as lost inbound messages or receipts.
func (c *Client) dispatch(eventCh chan any, evt any, kind string) {
	select {
	case eventCh <- evt:
	default:
		if c.logger != nil {
			c.logger.Warnf("event channel full (cap=%d), dropping %s event", cap(eventCh), kind)
		}
	}
}

// handleEvent handles incoming WhatsApp events
func (c *Client) handleEvent(evt any) {
	c.mu.Lock()
	eventCh := c.eventCh
	c.mu.Unlock()

	switch v := evt.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.state = DeviceStateConnected
		c.mu.Unlock()
		c.dispatch(eventCh, ConnectionEvent{State: DeviceStateConnected, Time: time.Now()}, "connected")

	case *events.Disconnected:
		c.mu.Lock()
		c.state = DeviceStateDisconnected
		c.mu.Unlock()
		c.dispatch(eventCh, ConnectionEvent{State: DeviceStateDisconnected, Time: time.Now()}, "disconnected")

	case *events.LoggedOut:
		c.mu.Lock()
		c.state = DeviceStateLoggedOut
		c.mu.Unlock()
		c.dispatch(eventCh, LogoutEvent{
			Reason:    v.Reason.String(),
			Time:      time.Now(),
			FromPhone: v.OnConnect,
		}, "logout")

	case *events.Message:
		msg := convertMessage(v)
		if msg != nil {
			c.dispatch(eventCh, msg, "message")
		}

	case *events.Receipt:
		receipt := convertReceipt(v)
		if receipt != nil {
			c.dispatch(eventCh, receipt, "receipt")
		}

	case *events.Presence:
		presence := &PresenceUpdate{
			JID:         v.From,
			Available:   v.Unavailable == false,
			Unavailable: v.Unavailable,
			LastSeenAt:  v.LastSeen,
		}
		c.dispatch(eventCh, presence, "presence")

	case *events.ChatPresence:
		chatPresence := &ChatPresence{
			JID:  v.MessageSource.Sender,
			Chat: v.MessageSource.Chat,
		}
		switch v.State {
		case "composing":
			chatPresence.State = ChatPresenceComposing
		case "recording":
			chatPresence.State = ChatPresenceRecording
		default:
			chatPresence.State = ChatPresencePaused
		}
		c.dispatch(eventCh, chatPresence, "chat_presence")

	case *events.HistorySync:
		c.dispatch(eventCh, HistorySyncEvent{
			Complete: true,
			Time:     time.Now(),
		}, "history_sync")
	}
}

// maxEnvelopeDepth bounds how many nested wrapper messages unwrapEnvelopes will
// peel, guarding against pathologically nested (or malicious) payloads.
const maxEnvelopeDepth = 8

// unwrapEnvelopes peels wrapper messages (disappearing/ephemeral, view-once in
// all its variants, device-sent, and edited) down to the real payload so
// downstream classification sees the actual image/text/etc. instead of an empty
// envelope. It also reports whether an edit envelope was unwrapped.
func unwrapEnvelopes(m *waE2E.Message) (*waE2E.Message, bool) {
	edited := false
	for i := 0; m != nil && i < maxEnvelopeDepth; i++ {
		switch {
		case m.GetEphemeralMessage().GetMessage() != nil:
			m = m.GetEphemeralMessage().GetMessage()
		case m.GetViewOnceMessage().GetMessage() != nil:
			m = m.GetViewOnceMessage().GetMessage()
		case m.GetViewOnceMessageV2().GetMessage() != nil:
			m = m.GetViewOnceMessageV2().GetMessage()
		case m.GetViewOnceMessageV2Extension().GetMessage() != nil:
			m = m.GetViewOnceMessageV2Extension().GetMessage()
		case m.GetDeviceSentMessage().GetMessage() != nil:
			m = m.GetDeviceSentMessage().GetMessage()
		case m.GetEditedMessage().GetMessage() != nil:
			edited = true
			m = m.GetEditedMessage().GetMessage()
		case m.GetProtocolMessage().GetEditedMessage() != nil:
			edited = true
			m = m.GetProtocolMessage().GetEditedMessage()
		default:
			return m, edited
		}
	}
	return m, edited
}

// convertMessage converts a whatsmeow message event to IncomingMessage
func convertMessage(evt *events.Message) *IncomingMessage {
	if evt == nil || evt.Message == nil {
		return nil
	}

	// Unwrap envelope messages down to the real payload before classifying.
	content, edited := unwrapEnvelopes(evt.Message)
	if content == nil {
		content = evt.Message
	}

	msg := &IncomingMessage{
		ExternalID: evt.Info.ID,
		SenderJID:  evt.Info.Sender,
		ChatJID:    evt.Info.Chat,
		SenderName: evt.Info.PushName,
		Timestamp:  evt.Info.Timestamp,
		IsFromMe:   evt.Info.IsFromMe,
		IsGroup:    evt.Info.IsGroup,
		IsEdit:     edited,
		RawMessage: content,
	}

	// Extract text content
	if conv := content.GetConversation(); conv != "" {
		msg.Text = conv
		msg.MessageType = "text"
	} else if ext := content.GetExtendedTextMessage(); ext != nil {
		// Extended text also carries Click-to-WhatsApp ad CTAs (matchedText)
		// and link-preview descriptions — fall back to those when the body is
		// empty so the ad/link context is not lost.
		txt := ext.GetText()
		if strings.TrimSpace(txt) == "" {
			txt = ext.GetMatchedText()
		}
		if strings.TrimSpace(txt) == "" {
			txt = ext.GetDescription()
		}
		msg.Text = txt
		msg.MessageType = "text"

		// Handle context info (reply, mentions)
		if ctx := ext.GetContextInfo(); ctx != nil {
			if ctx.StanzaID != nil {
				msg.QuotedID = *ctx.StanzaID
				msg.ReplyTo = &ReplyInfo{
					MessageID: *ctx.StanzaID,
				}
				if ctx.Participant != nil {
					participantJID, _ := types.ParseJID(*ctx.Participant)
					msg.ReplyTo.SenderJID = participantJID
				}
				if ctx.QuotedMessage != nil {
					msg.ReplyTo.Text = ctx.QuotedMessage.GetConversation()
				}
			}
			msg.Mentions = ctx.GetMentionedJID()
			msg.IsForwarded = ctx.GetIsForwarded()
		}
	}

	// Handle image message
	if img := content.GetImageMessage(); img != nil {
		msg.MessageType = "image"
		msg.Text = img.GetCaption()
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:      "image",
			URL:       img.GetURL(),
			MediaKey:  img.GetMediaKey(),
			SHA256:    img.GetFileSHA256(),
			EncSHA256: img.GetFileEncSHA256(),
			MimeType:  img.GetMimetype(),
			FileSize:  img.GetFileLength(),
			Width:     img.GetWidth(),
			Height:    img.GetHeight(),
			Thumbnail: img.GetJPEGThumbnail(),
			download:  img,
		})
		handleContextInfo(msg, img.GetContextInfo())
	}

	// Handle video message
	if video := content.GetVideoMessage(); video != nil {
		msg.MessageType = "video"
		msg.Text = video.GetCaption()
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:      "video",
			URL:       video.GetURL(),
			MediaKey:  video.GetMediaKey(),
			SHA256:    video.GetFileSHA256(),
			EncSHA256: video.GetFileEncSHA256(),
			MimeType:  video.GetMimetype(),
			FileSize:  video.GetFileLength(),
			Width:     video.GetWidth(),
			Height:    video.GetHeight(),
			Duration:  video.GetSeconds(),
			Thumbnail: video.GetJPEGThumbnail(),
			download:  video,
		})
		handleContextInfo(msg, video.GetContextInfo())
	}

	// Handle audio message
	if audio := content.GetAudioMessage(); audio != nil {
		if audio.GetPTT() {
			msg.MessageType = "ptt"
		} else {
			msg.MessageType = "audio"
		}
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:      "audio",
			URL:       audio.GetURL(),
			MediaKey:  audio.GetMediaKey(),
			SHA256:    audio.GetFileSHA256(),
			EncSHA256: audio.GetFileEncSHA256(),
			MimeType:  audio.GetMimetype(),
			FileSize:  audio.GetFileLength(),
			Duration:  audio.GetSeconds(),
			download:  audio,
		})
		handleContextInfo(msg, audio.GetContextInfo())
	}

	// Handle document message
	if doc := content.GetDocumentMessage(); doc != nil {
		msg.MessageType = "document"
		msg.Text = doc.GetCaption()
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:      "document",
			URL:       doc.GetURL(),
			MediaKey:  doc.GetMediaKey(),
			SHA256:    doc.GetFileSHA256(),
			EncSHA256: doc.GetFileEncSHA256(),
			MimeType:  doc.GetMimetype(),
			FileSize:  doc.GetFileLength(),
			Filename:  doc.GetFileName(),
			Thumbnail: doc.GetJPEGThumbnail(),
			download:  doc,
		})
		handleContextInfo(msg, doc.GetContextInfo())
	}

	// Handle sticker message
	if sticker := content.GetStickerMessage(); sticker != nil {
		msg.MessageType = "sticker"
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:      "sticker",
			URL:       sticker.GetURL(),
			MediaKey:  sticker.GetMediaKey(),
			SHA256:    sticker.GetFileSHA256(),
			EncSHA256: sticker.GetFileEncSHA256(),
			MimeType:  sticker.GetMimetype(),
			FileSize:  sticker.GetFileLength(),
			Width:     sticker.GetWidth(),
			Height:    sticker.GetHeight(),
			download:  sticker,
		})
		handleContextInfo(msg, sticker.GetContextInfo())
	}

	// Handle location message
	if loc := content.GetLocationMessage(); loc != nil {
		msg.MessageType = "location"
		msg.Text = loc.GetName()
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:      "location",
			Caption:   loc.GetAddress(),
			Filename:  loc.GetName(),
			Latitude:  loc.GetDegreesLatitude(),
			Longitude: loc.GetDegreesLongitude(),
		})
	}

	// Handle live location message (continuously updated position)
	if live := content.GetLiveLocationMessage(); live != nil {
		msg.MessageType = "location"
		msg.Text = live.GetCaption()
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:      "location",
			Caption:   live.GetCaption(),
			Latitude:  live.GetDegreesLatitude(),
			Longitude: live.GetDegreesLongitude(),
		})
	}

	// Handle contact message
	if contact := content.GetContactMessage(); contact != nil {
		msg.MessageType = "contact"
		msg.Text = contact.GetDisplayName()
	}

	// Handle contacts array (multiple vCards shared at once)
	if arr := content.GetContactsArrayMessage(); arr != nil {
		msg.MessageType = "contact"
		if name := arr.GetDisplayName(); name != "" {
			msg.Text = name
		}
	}

	// Handle reaction message
	if reaction := content.GetReactionMessage(); reaction != nil {
		msg.MessageType = "reaction"
		msg.Reaction = &Reaction{
			Emoji:     reaction.GetText(),
			SenderJID: evt.Info.Sender,
			MessageID: reaction.GetKey().GetID(),
			Timestamp: evt.Info.Timestamp,
		}
	}

	// Handle interactive replies (native-flow button/list taps, template
	// buttons). These carry no plain conversation text; the tapped option's id
	// and label are surfaced via SelectedID/Text.
	parseInteractiveReply(content, msg)

	return msg
}

// handleContextInfo extracts context info from a message
func handleContextInfo(msg *IncomingMessage, ctx interface {
	GetStanzaID() string
	GetParticipant() string
	GetMentionedJID() []string
	GetIsForwarded() bool
}) {
	if ctx == nil {
		return
	}

	if stanzaID := ctx.GetStanzaID(); stanzaID != "" {
		msg.QuotedID = stanzaID
		msg.ReplyTo = &ReplyInfo{
			MessageID: stanzaID,
		}
		if participant := ctx.GetParticipant(); participant != "" {
			participantJID, _ := types.ParseJID(participant)
			msg.ReplyTo.SenderJID = participantJID
		}
	}

	msg.Mentions = ctx.GetMentionedJID()
	msg.IsForwarded = ctx.GetIsForwarded()
}

// convertReceipt converts a whatsmeow receipt event to Receipt
func convertReceipt(evt *events.Receipt) *Receipt {
	if evt == nil {
		return nil
	}

	receipt := &Receipt{
		MessageIDs: evt.MessageIDs,
		SenderJID:  evt.MessageSource.Sender,
		ChatJID:    evt.MessageSource.Chat,
		Timestamp:  evt.Timestamp,
	}

	switch evt.Type {
	case events.ReceiptTypeDelivered:
		receipt.Type = ReceiptTypeDelivered
	case events.ReceiptTypeRead:
		receipt.Type = ReceiptTypeRead
	case events.ReceiptTypePlayed:
		receipt.Type = ReceiptTypePlayed
	default:
		receipt.Type = ReceiptTypeDelivered
	}

	return receipt
}
