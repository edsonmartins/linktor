package whatsapp

import (
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

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
		select {
		case eventCh <- ConnectionEvent{State: DeviceStateConnected, Time: time.Now()}:
		default:
		}

	case *events.Disconnected:
		c.mu.Lock()
		c.state = DeviceStateDisconnected
		c.mu.Unlock()
		select {
		case eventCh <- ConnectionEvent{State: DeviceStateDisconnected, Time: time.Now()}:
		default:
		}

	case *events.LoggedOut:
		c.mu.Lock()
		c.state = DeviceStateLoggedOut
		c.mu.Unlock()
		select {
		case eventCh <- LogoutEvent{
			Reason:    v.Reason.String(),
			Time:      time.Now(),
			FromPhone: v.OnConnect,
		}:
		default:
		}

	case *events.Message:
		msg := convertMessage(v)
		if msg != nil {
			select {
			case eventCh <- msg:
			default:
			}
		}

	case *events.Receipt:
		receipt := convertReceipt(v)
		if receipt != nil {
			select {
			case eventCh <- receipt:
			default:
			}
		}

	case *events.Presence:
		presence := &PresenceUpdate{
			JID:         v.From,
			Available:   v.Unavailable == false,
			Unavailable: v.Unavailable,
			LastSeenAt:  v.LastSeen,
		}
		select {
		case eventCh <- presence:
		default:
		}

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
		select {
		case eventCh <- chatPresence:
		default:
		}

	case *events.HistorySync:
		select {
		case eventCh <- HistorySyncEvent{
			Complete: true,
			Time:     time.Now(),
		}:
		default:
		}
	}
}

// convertMessage converts a whatsmeow message event to IncomingMessage
func convertMessage(evt *events.Message) *IncomingMessage {
	if evt == nil || evt.Message == nil {
		return nil
	}

	msg := &IncomingMessage{
		ExternalID:  evt.Info.ID,
		SenderJID:   evt.Info.Sender,
		ChatJID:     evt.Info.Chat,
		SenderName:  evt.Info.PushName,
		Timestamp:   evt.Info.Timestamp,
		IsFromMe:    evt.Info.IsFromMe,
		IsGroup:     evt.Info.IsGroup,
		RawMessage:  evt.Message,
	}

	// Extract text content
	if conv := evt.Message.GetConversation(); conv != "" {
		msg.Text = conv
		msg.MessageType = "text"
	} else if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		msg.Text = ext.GetText()
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
	if img := evt.Message.GetImageMessage(); img != nil {
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
		})
		handleContextInfo(msg, img.GetContextInfo())
	}

	// Handle video message
	if video := evt.Message.GetVideoMessage(); video != nil {
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
		})
		handleContextInfo(msg, video.GetContextInfo())
	}

	// Handle audio message
	if audio := evt.Message.GetAudioMessage(); audio != nil {
		if audio.GetPTT() {
			msg.MessageType = "ptt"
		} else {
			msg.MessageType = "audio"
		}
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:     "audio",
			URL:      audio.GetURL(),
			MediaKey: audio.GetMediaKey(),
			SHA256:   audio.GetFileSHA256(),
			EncSHA256: audio.GetFileEncSHA256(),
			MimeType: audio.GetMimetype(),
			FileSize: audio.GetFileLength(),
			Duration: audio.GetSeconds(),
		})
		handleContextInfo(msg, audio.GetContextInfo())
	}

	// Handle document message
	if doc := evt.Message.GetDocumentMessage(); doc != nil {
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
		})
		handleContextInfo(msg, doc.GetContextInfo())
	}

	// Handle sticker message
	if sticker := evt.Message.GetStickerMessage(); sticker != nil {
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
		})
		handleContextInfo(msg, sticker.GetContextInfo())
	}

	// Handle location message
	if loc := evt.Message.GetLocationMessage(); loc != nil {
		msg.MessageType = "location"
		msg.Text = loc.GetName()
		msg.Attachments = append(msg.Attachments, Attachment{
			Type:     "location",
			Caption:  loc.GetAddress(),
			Filename: loc.GetName(),
		})
	}

	// Handle contact message
	if contact := evt.Message.GetContactMessage(); contact != nil {
		msg.MessageType = "contact"
		msg.Text = contact.GetDisplayName()
	}

	// Handle reaction message
	if reaction := evt.Message.GetReactionMessage(); reaction != nil {
		msg.MessageType = "reaction"
		msg.Reaction = &Reaction{
			Emoji:     reaction.GetText(),
			SenderJID: evt.Info.Sender,
			MessageID: reaction.GetKey().GetID(),
			Timestamp: evt.Info.Timestamp,
		}
	}

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

