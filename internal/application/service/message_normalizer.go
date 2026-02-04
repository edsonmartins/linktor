package service

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// MessageNormalizer handles conversion between channel-specific formats and canonical format
type MessageNormalizer struct{}

// NewMessageNormalizer creates a new message normalizer
func NewMessageNormalizer() *MessageNormalizer {
	return &MessageNormalizer{}
}

// NormalizedMessage represents a message in canonical format
type NormalizedMessage struct {
	ID             string
	TenantID       string
	ChannelID      string
	ChannelType    string
	ConversationID string
	ContactID      string
	SenderType     entity.SenderType
	SenderID       string
	ContentType    entity.ContentType
	Content        string
	Metadata       map[string]string
	ExternalID     string
	Attachments    []*entity.MessageAttachment
	Timestamp      time.Time
}

// NormalizeInbound converts an inbound NATS message to canonical format
func (n *MessageNormalizer) NormalizeInbound(msg *nats.InboundMessage) *NormalizedMessage {
	contentType := n.normalizeContentType(msg.ContentType)
	content := n.normalizeContent(msg.Content, contentType)

	normalized := &NormalizedMessage{
		ID:             msg.ID,
		TenantID:       msg.TenantID,
		ChannelID:      msg.ChannelID,
		ChannelType:    msg.ChannelType,
		ConversationID: msg.ConversationID,
		ContactID:      msg.ContactID,
		SenderType:     entity.SenderTypeContact,
		ContentType:    contentType,
		Content:        content,
		Metadata:       msg.Metadata,
		ExternalID:     msg.ExternalID,
		Attachments:    n.normalizeAttachments(msg.Attachments),
		Timestamp:      msg.Timestamp,
	}

	if normalized.Metadata == nil {
		normalized.Metadata = make(map[string]string)
	}

	return normalized
}

// ToEntity converts a normalized message to a domain entity
func (n *MessageNormalizer) ToEntity(msg *NormalizedMessage) *entity.Message {
	message := &entity.Message{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderType:     msg.SenderType,
		SenderID:       msg.SenderID,
		ContentType:    msg.ContentType,
		Content:        msg.Content,
		Metadata:       msg.Metadata,
		Status:         entity.MessageStatusPending,
		ExternalID:     msg.ExternalID,
		Attachments:    msg.Attachments,
		CreatedAt:      msg.Timestamp,
	}

	if message.Metadata == nil {
		message.Metadata = make(map[string]string)
	}

	return message
}

// ToOutbound converts a domain message to an outbound NATS message
func (n *MessageNormalizer) ToOutbound(msg *entity.Message, channelType, channelID, tenantID, contactID, recipientID string) *nats.OutboundMessage {
	attachments := make([]nats.AttachmentData, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, nats.AttachmentData{
			Type:         att.Type,
			URL:          att.URL,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
			SizeBytes:    att.SizeBytes,
			ThumbnailURL: att.ThumbnailURL,
			Metadata:     att.Metadata,
		})
	}

	return &nats.OutboundMessage{
		ID:             msg.ID,
		TenantID:       tenantID,
		ChannelID:      channelID,
		ChannelType:    channelType,
		ConversationID: msg.ConversationID,
		ContactID:      contactID,
		RecipientID:    recipientID,
		ContentType:    string(msg.ContentType),
		Content:        msg.Content,
		Metadata:       msg.Metadata,
		Attachments:    attachments,
		Timestamp:      time.Now(),
	}
}

// normalizeContentType converts channel-specific content types to canonical types
func (n *MessageNormalizer) normalizeContentType(contentType string) entity.ContentType {
	ct := strings.ToLower(contentType)

	switch ct {
	case "text", "text/plain", "plain":
		return entity.ContentTypeText
	case "image", "image/jpeg", "image/png", "image/gif", "image/webp":
		return entity.ContentTypeImage
	case "video", "video/mp4", "video/3gpp":
		return entity.ContentTypeVideo
	case "audio", "audio/ogg", "audio/mpeg", "audio/mp3", "ptt", "voice":
		return entity.ContentTypeAudio
	case "document", "file", "application/pdf":
		return entity.ContentTypeDocument
	case "location", "geo":
		return entity.ContentTypeLocation
	case "contact", "vcard", "contacts":
		return entity.ContentTypeContact
	case "template", "hsm":
		return entity.ContentTypeTemplate
	case "interactive", "button", "list", "buttons", "list_reply":
		return entity.ContentTypeInteractive
	default:
		return entity.ContentTypeText
	}
}

// normalizeContent cleans and normalizes message content
func (n *MessageNormalizer) normalizeContent(content string, contentType entity.ContentType) string {
	// Trim whitespace
	content = strings.TrimSpace(content)

	// For text messages, normalize line breaks
	if contentType == entity.ContentTypeText {
		content = strings.ReplaceAll(content, "\r\n", "\n")
		content = strings.ReplaceAll(content, "\r", "\n")
	}

	return content
}

// normalizeAttachments converts NATS attachments to entity attachments
func (n *MessageNormalizer) normalizeAttachments(attachments []nats.AttachmentData) []*entity.MessageAttachment {
	if len(attachments) == 0 {
		return nil
	}

	result := make([]*entity.MessageAttachment, 0, len(attachments))
	for _, att := range attachments {
		attachment := &entity.MessageAttachment{
			ID:           uuid.New().String(),
			Type:         att.Type,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
			SizeBytes:    att.SizeBytes,
			URL:          att.URL,
			ThumbnailURL: att.ThumbnailURL,
			Metadata:     att.Metadata,
			CreatedAt:    time.Now(),
		}

		if attachment.Metadata == nil {
			attachment.Metadata = make(map[string]string)
		}

		result = append(result, attachment)
	}

	return result
}

// DenormalizeForChannel converts canonical message to channel-specific format
func (n *MessageNormalizer) DenormalizeForChannel(msg *entity.Message, channelType string) map[string]interface{} {
	result := make(map[string]interface{})

	switch channelType {
	case "whatsapp", "whatsapp_official":
		result = n.toWhatsAppFormat(msg)
	case "telegram":
		result = n.toTelegramFormat(msg)
	case "webchat":
		result = n.toWebChatFormat(msg)
	case "sms":
		result = n.toSMSFormat(msg)
	default:
		result = n.toGenericFormat(msg)
	}

	return result
}

func (n *MessageNormalizer) toWhatsAppFormat(msg *entity.Message) map[string]interface{} {
	result := map[string]interface{}{
		"type": string(msg.ContentType),
	}

	switch msg.ContentType {
	case entity.ContentTypeText:
		result["text"] = map[string]interface{}{
			"body": msg.Content,
		}
	case entity.ContentTypeImage:
		if len(msg.Attachments) > 0 {
			result["image"] = map[string]interface{}{
				"link":    msg.Attachments[0].URL,
				"caption": msg.Content,
			}
		}
	case entity.ContentTypeDocument:
		if len(msg.Attachments) > 0 {
			result["document"] = map[string]interface{}{
				"link":     msg.Attachments[0].URL,
				"filename": msg.Attachments[0].Filename,
				"caption":  msg.Content,
			}
		}
	case entity.ContentTypeAudio:
		if len(msg.Attachments) > 0 {
			result["audio"] = map[string]interface{}{
				"link": msg.Attachments[0].URL,
			}
		}
	case entity.ContentTypeVideo:
		if len(msg.Attachments) > 0 {
			result["video"] = map[string]interface{}{
				"link":    msg.Attachments[0].URL,
				"caption": msg.Content,
			}
		}
	case entity.ContentTypeLocation:
		result["location"] = map[string]interface{}{
			"latitude":  msg.Metadata["latitude"],
			"longitude": msg.Metadata["longitude"],
			"name":      msg.Metadata["name"],
			"address":   msg.Content,
		}
	}

	return result
}

func (n *MessageNormalizer) toTelegramFormat(msg *entity.Message) map[string]interface{} {
	result := map[string]interface{}{
		"parse_mode": "HTML",
	}

	switch msg.ContentType {
	case entity.ContentTypeText:
		result["text"] = msg.Content
	case entity.ContentTypeImage:
		if len(msg.Attachments) > 0 {
			result["photo"] = msg.Attachments[0].URL
			result["caption"] = msg.Content
		}
	case entity.ContentTypeDocument:
		if len(msg.Attachments) > 0 {
			result["document"] = msg.Attachments[0].URL
			result["caption"] = msg.Content
		}
	default:
		result["text"] = msg.Content
	}

	return result
}

func (n *MessageNormalizer) toWebChatFormat(msg *entity.Message) map[string]interface{} {
	attachments := make([]map[string]interface{}, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, map[string]interface{}{
			"type":         att.Type,
			"url":          att.URL,
			"filename":     att.Filename,
			"mimeType":     att.MimeType,
			"sizeBytes":    att.SizeBytes,
			"thumbnailUrl": att.ThumbnailURL,
		})
	}

	return map[string]interface{}{
		"id":          msg.ID,
		"contentType": string(msg.ContentType),
		"content":     msg.Content,
		"senderType":  string(msg.SenderType),
		"senderId":    msg.SenderID,
		"attachments": attachments,
		"metadata":    msg.Metadata,
		"status":      string(msg.Status),
		"createdAt":   msg.CreatedAt.Format(time.RFC3339),
	}
}

func (n *MessageNormalizer) toSMSFormat(msg *entity.Message) map[string]interface{} {
	// SMS only supports text
	content := msg.Content
	if len(content) > 160 {
		content = content[:157] + "..."
	}

	return map[string]interface{}{
		"body": content,
	}
}

func (n *MessageNormalizer) toGenericFormat(msg *entity.Message) map[string]interface{} {
	return map[string]interface{}{
		"id":          msg.ID,
		"contentType": string(msg.ContentType),
		"content":     msg.Content,
		"metadata":    msg.Metadata,
	}
}
