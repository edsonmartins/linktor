package whatsapp_official

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
)

// WebhookProcessor processes incoming webhooks from WhatsApp
type WebhookProcessor struct {
	config *Config
}

// NewWebhookProcessor creates a new webhook processor
func NewWebhookProcessor(config *Config) *WebhookProcessor {
	return &WebhookProcessor{
		config: config,
	}
}

// ValidateSignature validates the HMAC-SHA256 signature of a webhook request
func ValidateSignature(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyChallenge handles the webhook verification challenge from Meta
func VerifyChallenge(verifyToken, mode, token, challenge string) (string, bool) {
	if mode != "subscribe" {
		return "", false
	}

	if token != verifyToken {
		return "", false
	}

	return challenge, true
}

// ParseWebhook parses a webhook payload
func (p *WebhookProcessor) ParseWebhook(body []byte) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	if payload.Object != "whatsapp_business_account" {
		return nil, fmt.Errorf("invalid webhook object: %s", payload.Object)
	}

	return &payload, nil
}

// ExtractMessages extracts inbound messages from webhook payload
func (p *WebhookProcessor) ExtractMessages(payload *WebhookPayload) []*ParsedMessage {
	var messages []*ParsedMessage

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			// Build contact map for quick lookup
			contactMap := make(map[string]ContactInfo)
			for _, contact := range change.Value.Contacts {
				contactMap[contact.WaID] = contact
			}

			// Process messages
			for _, msg := range change.Value.Messages {
				parsed := p.parseMessage(&msg, contactMap, &change.Value.Metadata)
				if parsed != nil {
					messages = append(messages, parsed)
				}
			}
		}
	}

	return messages
}

// ExtractStatuses extracts status updates from webhook payload
func (p *WebhookProcessor) ExtractStatuses(payload *WebhookPayload) []*ParsedStatus {
	var statuses []*ParsedStatus

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			for _, status := range change.Value.Statuses {
				parsed := p.parseStatus(&status)
				if parsed != nil {
					statuses = append(statuses, parsed)
				}
			}
		}
	}

	return statuses
}

// ParsedMessage represents a parsed inbound message
type ParsedMessage struct {
	ExternalID    string
	From          string
	SenderName    string
	ContentType   plugin.ContentType
	Content       string
	Attachments   []*plugin.Attachment
	Metadata      map[string]string
	ReplyToID     string
	Timestamp     time.Time
	PhoneNumberID string
}

// ParsedStatus represents a parsed status update
type ParsedStatus struct {
	MessageID    string
	RecipientID  string
	Status       plugin.MessageStatus
	ErrorMessage string
	Timestamp    time.Time
}

// parseMessage parses a single incoming message
func (p *WebhookProcessor) parseMessage(msg *IncomingMessage, contacts map[string]ContactInfo, metadata *WebhookMetadata) *ParsedMessage {
	if msg == nil {
		return nil
	}

	parsed := &ParsedMessage{
		ExternalID:    msg.ID,
		From:          msg.From,
		Metadata:      make(map[string]string),
		PhoneNumberID: metadata.PhoneNumberID,
	}

	// Get sender name from contacts
	if contact, ok := contacts[msg.From]; ok {
		parsed.SenderName = contact.Profile.Name
	}

	// Parse timestamp
	if ts, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
		parsed.Timestamp = time.Unix(ts, 0)
	} else {
		parsed.Timestamp = time.Now()
	}

	// Store reply-to context
	if msg.Context != nil {
		parsed.ReplyToID = msg.Context.MessageID
		parsed.Metadata["reply_to_id"] = msg.Context.MessageID
		parsed.Metadata["reply_to_from"] = msg.Context.From
		if msg.Context.Forwarded {
			parsed.Metadata["forwarded"] = "true"
		}
		if msg.Context.FrequentlyForwarded {
			parsed.Metadata["frequently_forwarded"] = "true"
		}
	}

	// Parse message based on type
	switch msg.Type {
	case MessageTypeText:
		parsed.ContentType = plugin.ContentTypeText
		if msg.Text != nil {
			parsed.Content = msg.Text.Body
		}

	case MessageTypeImage:
		parsed.ContentType = plugin.ContentTypeImage
		if msg.Image != nil {
			parsed.Content = msg.Image.Caption
			parsed.Attachments = append(parsed.Attachments, &plugin.Attachment{
				Type:     "image",
				URL:      msg.Image.ID, // Media ID - will be resolved to URL
				MimeType: msg.Image.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Image.ID,
					"sha256":   msg.Image.SHA256,
				},
			})
		}

	case MessageTypeVideo:
		parsed.ContentType = plugin.ContentTypeVideo
		if msg.Video != nil {
			parsed.Content = msg.Video.Caption
			parsed.Attachments = append(parsed.Attachments, &plugin.Attachment{
				Type:     "video",
				URL:      msg.Video.ID,
				MimeType: msg.Video.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Video.ID,
					"sha256":   msg.Video.SHA256,
				},
			})
		}

	case MessageTypeAudio:
		parsed.ContentType = plugin.ContentTypeAudio
		if msg.Audio != nil {
			parsed.Attachments = append(parsed.Attachments, &plugin.Attachment{
				Type:     "audio",
				URL:      msg.Audio.ID,
				MimeType: msg.Audio.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Audio.ID,
					"sha256":   msg.Audio.SHA256,
				},
			})
		}

	case MessageTypeDocument:
		parsed.ContentType = plugin.ContentTypeDocument
		if msg.Document != nil {
			parsed.Content = msg.Document.Caption
			parsed.Attachments = append(parsed.Attachments, &plugin.Attachment{
				Type:     "document",
				URL:      msg.Document.ID,
				Filename: msg.Document.Filename,
				MimeType: msg.Document.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Document.ID,
					"sha256":   msg.Document.SHA256,
				},
			})
		}

	case MessageTypeSticker:
		parsed.ContentType = plugin.ContentTypeImage
		parsed.Metadata["is_sticker"] = "true"
		if msg.Sticker != nil {
			parsed.Attachments = append(parsed.Attachments, &plugin.Attachment{
				Type:     "sticker",
				URL:      msg.Sticker.ID,
				MimeType: msg.Sticker.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Sticker.ID,
					"sha256":   msg.Sticker.SHA256,
					"animated": fmt.Sprintf("%t", msg.Sticker.Animated),
				},
			})
		}

	case MessageTypeLocation:
		parsed.ContentType = plugin.ContentTypeLocation
		if msg.Location != nil {
			locationData, _ := json.Marshal(msg.Location)
			parsed.Content = string(locationData)
			parsed.Metadata["latitude"] = fmt.Sprintf("%f", msg.Location.Latitude)
			parsed.Metadata["longitude"] = fmt.Sprintf("%f", msg.Location.Longitude)
			if msg.Location.Name != "" {
				parsed.Metadata["location_name"] = msg.Location.Name
			}
			if msg.Location.Address != "" {
				parsed.Metadata["location_address"] = msg.Location.Address
			}
		}

	case MessageTypeContacts:
		parsed.ContentType = plugin.ContentTypeContact
		if len(msg.Contacts) > 0 {
			contactsData, _ := json.Marshal(msg.Contacts)
			parsed.Content = string(contactsData)
			parsed.Metadata["contact_count"] = fmt.Sprintf("%d", len(msg.Contacts))
		}

	case MessageTypeInteractive:
		parsed.ContentType = plugin.ContentTypeInteractive
		if msg.Interactive != nil {
			parsed.Metadata["interactive_type"] = msg.Interactive.Type
			if msg.Interactive.ButtonReply != nil {
				parsed.Content = msg.Interactive.ButtonReply.Title
				parsed.Metadata["button_id"] = msg.Interactive.ButtonReply.ID
				parsed.Metadata["button_title"] = msg.Interactive.ButtonReply.Title
			} else if msg.Interactive.ListReply != nil {
				parsed.Content = msg.Interactive.ListReply.Title
				parsed.Metadata["list_id"] = msg.Interactive.ListReply.ID
				parsed.Metadata["list_title"] = msg.Interactive.ListReply.Title
				if msg.Interactive.ListReply.Description != "" {
					parsed.Metadata["list_description"] = msg.Interactive.ListReply.Description
				}
			} else if msg.Interactive.NfmReply != nil {
				// WhatsApp Flow response (Native Flow Message)
				parsed.Metadata["is_flow_response"] = "true"
				parsed.Metadata["flow_name"] = msg.Interactive.NfmReply.Name
				parsed.Metadata["flow_body"] = msg.Interactive.NfmReply.Body
				parsed.Metadata["flow_response_json"] = msg.Interactive.NfmReply.ResponseJSON
				if msg.Interactive.NfmReply.FlowToken != "" {
					parsed.Metadata["flow_token"] = msg.Interactive.NfmReply.FlowToken
				}
				// Store the full response JSON as content for processing
				parsed.Content = msg.Interactive.NfmReply.ResponseJSON
			}
		}

	case MessageTypeOrder:
		parsed.ContentType = plugin.ContentTypeInteractive
		parsed.Metadata["is_order"] = "true"
		if msg.Order != nil {
			parsed.Metadata["catalog_id"] = msg.Order.CatalogID
			if msg.Order.Text != "" {
				parsed.Content = msg.Order.Text
			}
			// Serialize order items to JSON
			orderItemsJSON, _ := json.Marshal(msg.Order.ProductItems)
			parsed.Metadata["order_items"] = string(orderItemsJSON)
			parsed.Metadata["order_item_count"] = fmt.Sprintf("%d", len(msg.Order.ProductItems))
		}

	case MessageTypeButton:
		parsed.ContentType = plugin.ContentTypeInteractive
		if msg.Button != nil {
			parsed.Content = msg.Button.Text
			parsed.Metadata["button_payload"] = msg.Button.Payload
		}

	case MessageTypeReaction:
		parsed.ContentType = plugin.ContentTypeText
		parsed.Metadata["is_reaction"] = "true"
		if msg.Reaction != nil {
			parsed.Content = msg.Reaction.Emoji
			parsed.Metadata["reaction_message_id"] = msg.Reaction.MessageID
			parsed.Metadata["reaction_emoji"] = msg.Reaction.Emoji
		}

	default:
		parsed.ContentType = plugin.ContentTypeText
		parsed.Metadata["original_type"] = string(msg.Type)
	}

	// Store phone number in metadata
	parsed.Metadata["phone"] = msg.From
	parsed.Metadata["sender_id"] = msg.From
	parsed.Metadata["sender_name"] = parsed.SenderName
	parsed.Metadata["phone_number_id"] = metadata.PhoneNumberID
	parsed.Metadata["display_phone_number"] = metadata.DisplayPhoneNumber

	// Handle referral data (click-to-WhatsApp ads)
	if msg.Referral != nil {
		parsed.Metadata["referral_source_type"] = msg.Referral.SourceType
		parsed.Metadata["referral_source_id"] = msg.Referral.SourceID
		if msg.Referral.Headline != "" {
			parsed.Metadata["referral_headline"] = msg.Referral.Headline
		}
		if msg.Referral.Body != "" {
			parsed.Metadata["referral_body"] = msg.Referral.Body
		}
	}

	return parsed
}

// parseStatus parses a single status update
func (p *WebhookProcessor) parseStatus(status *StatusUpdate) *ParsedStatus {
	if status == nil {
		return nil
	}

	parsed := &ParsedStatus{
		MessageID:   status.ID,
		RecipientID: status.RecipientID,
	}

	// Parse timestamp
	if ts, err := strconv.ParseInt(status.Timestamp, 10, 64); err == nil {
		parsed.Timestamp = time.Unix(ts, 0)
	} else {
		parsed.Timestamp = time.Now()
	}

	// Map status
	switch status.Status {
	case StatusSent:
		parsed.Status = plugin.MessageStatusSent
	case StatusDelivered:
		parsed.Status = plugin.MessageStatusDelivered
	case StatusRead:
		parsed.Status = plugin.MessageStatusRead
	case StatusFailed:
		parsed.Status = plugin.MessageStatusFailed
		if len(status.Errors) > 0 {
			parsed.ErrorMessage = status.Errors[0].Message
		}
	default:
		parsed.Status = plugin.MessageStatusPending
	}

	return parsed
}

// ToInboundMessage converts a ParsedMessage to plugin.InboundMessage
func (pm *ParsedMessage) ToInboundMessage() *plugin.InboundMessage {
	return &plugin.InboundMessage{
		ExternalID:  pm.ExternalID,
		SenderID:    pm.From,
		SenderName:  pm.SenderName,
		ContentType: pm.ContentType,
		Content:     pm.Content,
		Attachments: pm.Attachments,
		Metadata:    pm.Metadata,
		Timestamp:   pm.Timestamp,
	}
}

// ToStatusCallback converts a ParsedStatus to plugin.StatusCallback
func (ps *ParsedStatus) ToStatusCallback() *plugin.StatusCallback {
	return &plugin.StatusCallback{
		ExternalID:   ps.MessageID,
		Status:       ps.Status,
		ErrorMessage: ps.ErrorMessage,
		Timestamp:    ps.Timestamp,
	}
}

// IsMessageWebhook checks if the webhook contains messages
func IsMessageWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == "messages" && len(change.Value.Messages) > 0 {
				return true
			}
		}
	}
	return false
}

// IsStatusWebhook checks if the webhook contains status updates
func IsStatusWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == "messages" && len(change.Value.Statuses) > 0 {
				return true
			}
		}
	}
	return false
}

// GetWebhookPhoneNumberID extracts the phone number ID from webhook
func GetWebhookPhoneNumberID(payload *WebhookPayload) string {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Value.Metadata.PhoneNumberID != "" {
				return change.Value.Metadata.PhoneNumberID
			}
		}
	}
	return ""
}

// =============================================================================
// Extended Webhook Processing - All 13 Subscription Fields
// =============================================================================

// WebhookFieldType represents the type of webhook field received
type WebhookFieldType string

const (
	FieldMessages                  WebhookFieldType = "messages"
	FieldMessageTemplateStatus     WebhookFieldType = "message_template_status_update"
	FieldMessageTemplateQuality    WebhookFieldType = "message_template_quality_update"
	FieldAccountAlerts             WebhookFieldType = "account_alerts"
	FieldAccountUpdate             WebhookFieldType = "account_update"
	FieldAccountReviewUpdate       WebhookFieldType = "account_review_update"
	FieldPhoneNumberNameUpdate     WebhookFieldType = "phone_number_name_update"
	FieldPhoneNumberQualityUpdate  WebhookFieldType = "phone_number_quality_update"
	FieldTemplateCategoryUpdate    WebhookFieldType = "template_category_update"
	FieldSecurity                  WebhookFieldType = "security"
	FieldFlows                     WebhookFieldType = "flows"
	FieldBusinessCapabilityUpdate  WebhookFieldType = "business_capability_update"
	FieldMessageEchoes             WebhookFieldType = "message_echoes"
)

// GetWebhookFields returns all fields present in the webhook payload
func (p *WebhookProcessor) GetWebhookFields(payload *WebhookPayload) []WebhookFieldType {
	fieldsMap := make(map[WebhookFieldType]bool)
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			fieldsMap[WebhookFieldType(change.Field)] = true
		}
	}

	fields := make([]WebhookFieldType, 0, len(fieldsMap))
	for field := range fieldsMap {
		fields = append(fields, field)
	}
	return fields
}

// ExtractTemplateStatusUpdates extracts template status updates from webhook
func (p *WebhookProcessor) ExtractTemplateStatusUpdates(payload *WebhookPayload) []*ParsedTemplateStatusEvent {
	var events []*ParsedTemplateStatusEvent

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != string(FieldMessageTemplateStatus) {
				continue
			}

			// Access fields directly from WebhookChangeValue
			v := change.Value
			events = append(events, &ParsedTemplateStatusEvent{
				TemplateID:   v.MessageTemplateID,
				TemplateName: v.MessageTemplateName,
				Language:     v.MessageTemplateLanguage,
				Event:        v.Event,
				Reason:       v.Reason,
				Timestamp:    time.Now(),
			})
		}
	}

	return events
}

// ExtractTemplateQualityUpdates extracts template quality updates from webhook
func (p *WebhookProcessor) ExtractTemplateQualityUpdates(payload *WebhookPayload) []*ParsedTemplateQualityEvent {
	var events []*ParsedTemplateQualityEvent

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != string(FieldMessageTemplateQuality) {
				continue
			}

			v := change.Value
			events = append(events, &ParsedTemplateQualityEvent{
				TemplateID:      v.MessageTemplateID,
				TemplateName:    v.MessageTemplateName,
				Language:        v.MessageTemplateLanguage,
				PreviousQuality: v.PreviousQualityScore,
				NewQuality:      v.NewQualityScore,
				Timestamp:       time.Now(),
			})
		}
	}

	return events
}

// ExtractAccountAlerts extracts account alerts from webhook
func (p *WebhookProcessor) ExtractAccountAlerts(payload *WebhookPayload) []*ParsedAccountAlertEvent {
	var events []*ParsedAccountAlertEvent

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != string(FieldAccountAlerts) {
				continue
			}

			v := change.Value
			events = append(events, &ParsedAccountAlertEvent{
				Title:     v.Title,
				Message:   v.Message,
				Timestamp: time.Now(),
			})
		}
	}

	return events
}

// ExtractPhoneQualityUpdates extracts phone number quality updates from webhook
func (p *WebhookProcessor) ExtractPhoneQualityUpdates(payload *WebhookPayload) []*ParsedPhoneQualityEvent {
	var events []*ParsedPhoneQualityEvent

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != string(FieldPhoneNumberQualityUpdate) {
				continue
			}

			v := change.Value
			events = append(events, &ParsedPhoneQualityEvent{
				PhoneNumber:  v.DisplayPhoneNumber,
				Event:        v.Event,
				CurrentLimit: v.CurrentLimit,
				Timestamp:    time.Now(),
			})
		}
	}

	return events
}

// ExtractFlowEvents extracts flow lifecycle events from webhook
func (p *WebhookProcessor) ExtractFlowEvents(payload *WebhookPayload) []*ParsedFlowEvent {
	var events []*ParsedFlowEvent

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != string(FieldFlows) {
				continue
			}

			v := change.Value
			events = append(events, &ParsedFlowEvent{
				FlowID:       v.FlowID,
				FlowName:     v.FlowName,
				Event:        v.Event,
				OldStatus:    v.OldStatus,
				NewStatus:    v.NewStatus,
				ErrorType:    v.ErrorType,
				ErrorMessage: v.ErrorMessage,
				Timestamp:    time.Now(),
			})
		}
	}

	return events
}

// ExtractSecurityEvents extracts security events from webhook
func (p *WebhookProcessor) ExtractSecurityEvents(payload *WebhookPayload) []*ParsedSecurityEvent {
	var events []*ParsedSecurityEvent

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != string(FieldSecurity) {
				continue
			}

			v := change.Value
			events = append(events, &ParsedSecurityEvent{
				Event:       v.Event,
				PhoneNumber: v.DisplayPhoneNumber,
				Timestamp:   time.Now(),
			})
		}
	}

	return events
}

// IsTemplateStatusWebhook checks if webhook contains template status updates
func IsTemplateStatusWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == string(FieldMessageTemplateStatus) {
				return true
			}
		}
	}
	return false
}

// IsTemplateQualityWebhook checks if webhook contains template quality updates
func IsTemplateQualityWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == string(FieldMessageTemplateQuality) {
				return true
			}
		}
	}
	return false
}

// IsFlowWebhook checks if webhook contains flow events
func IsFlowWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == string(FieldFlows) {
				return true
			}
		}
	}
	return false
}

// IsAccountAlertWebhook checks if webhook contains account alerts
func IsAccountAlertWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == string(FieldAccountAlerts) {
				return true
			}
		}
	}
	return false
}

// IsPhoneQualityWebhook checks if webhook contains phone quality updates
func IsPhoneQualityWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == string(FieldPhoneNumberQualityUpdate) {
				return true
			}
		}
	}
	return false
}

// IsSecurityWebhook checks if webhook contains security events
func IsSecurityWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == string(FieldSecurity) {
				return true
			}
		}
	}
	return false
}

// =============================================================================
// Message Echoes - WhatsApp Coexistence Support
// =============================================================================

// ParsedMessageEcho represents a message sent via WhatsApp Business App (echo)
type ParsedMessageEcho struct {
	ExternalID    string
	To            string            // Recipient phone number
	RecipientName string            // Recipient contact name (if available)
	ContentType   plugin.ContentType
	Content       string
	Attachments   []*plugin.Attachment
	Metadata      map[string]string
	Timestamp     time.Time
	PhoneNumberID string
	SenderPhone   string // The business phone that sent the message
}

// ExtractMessageEchoes extracts message echoes from webhook payload
// Message echoes are messages sent via WhatsApp Business App that are
// synced to the Cloud API when using WhatsApp Coexistence
func (p *WebhookProcessor) ExtractMessageEchoes(payload *WebhookPayload) []*ParsedMessageEcho {
	var echoes []*ParsedMessageEcho

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != string(FieldMessageEchoes) {
				continue
			}

			// Build contact map for quick lookup
			contactMap := make(map[string]ContactInfo)
			for _, contact := range change.Value.Contacts {
				contactMap[contact.WaID] = contact
			}

			// Process echo messages
			for _, msg := range change.Value.Messages {
				parsed := p.parseMessageEcho(&msg, contactMap, &change.Value.Metadata)
				if parsed != nil {
					echoes = append(echoes, parsed)
				}
			}
		}
	}

	return echoes
}

// parseMessageEcho parses a single message echo (message sent via Business App)
func (p *WebhookProcessor) parseMessageEcho(msg *IncomingMessage, contacts map[string]ContactInfo, metadata *WebhookMetadata) *ParsedMessageEcho {
	if msg == nil {
		return nil
	}

	echo := &ParsedMessageEcho{
		ExternalID:    msg.ID,
		To:            msg.From, // In echoes, "from" is the recipient
		Metadata:      make(map[string]string),
		PhoneNumberID: metadata.PhoneNumberID,
		SenderPhone:   metadata.DisplayPhoneNumber, // The business phone that sent the message
	}

	// Get recipient name from contacts
	if contact, ok := contacts[msg.From]; ok {
		echo.RecipientName = contact.Profile.Name
	}

	// Parse timestamp
	if ts, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
		echo.Timestamp = time.Unix(ts, 0)
	} else {
		echo.Timestamp = time.Now()
	}

	// Mark as echo/business_app source
	echo.Metadata["source"] = "business_app"
	echo.Metadata["is_echo"] = "true"
	echo.Metadata["recipient_phone"] = msg.From
	echo.Metadata["sender_phone"] = metadata.DisplayPhoneNumber
	echo.Metadata["phone_number_id"] = metadata.PhoneNumberID

	// Parse message based on type (reusing same logic as regular messages)
	switch msg.Type {
	case MessageTypeText:
		echo.ContentType = plugin.ContentTypeText
		if msg.Text != nil {
			echo.Content = msg.Text.Body
		}

	case MessageTypeImage:
		echo.ContentType = plugin.ContentTypeImage
		if msg.Image != nil {
			echo.Content = msg.Image.Caption
			echo.Attachments = append(echo.Attachments, &plugin.Attachment{
				Type:     "image",
				URL:      msg.Image.ID,
				MimeType: msg.Image.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Image.ID,
					"sha256":   msg.Image.SHA256,
				},
			})
		}

	case MessageTypeVideo:
		echo.ContentType = plugin.ContentTypeVideo
		if msg.Video != nil {
			echo.Content = msg.Video.Caption
			echo.Attachments = append(echo.Attachments, &plugin.Attachment{
				Type:     "video",
				URL:      msg.Video.ID,
				MimeType: msg.Video.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Video.ID,
					"sha256":   msg.Video.SHA256,
				},
			})
		}

	case MessageTypeAudio:
		echo.ContentType = plugin.ContentTypeAudio
		if msg.Audio != nil {
			echo.Attachments = append(echo.Attachments, &plugin.Attachment{
				Type:     "audio",
				URL:      msg.Audio.ID,
				MimeType: msg.Audio.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Audio.ID,
					"sha256":   msg.Audio.SHA256,
				},
			})
		}

	case MessageTypeDocument:
		echo.ContentType = plugin.ContentTypeDocument
		if msg.Document != nil {
			echo.Content = msg.Document.Caption
			echo.Attachments = append(echo.Attachments, &plugin.Attachment{
				Type:     "document",
				URL:      msg.Document.ID,
				Filename: msg.Document.Filename,
				MimeType: msg.Document.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Document.ID,
					"sha256":   msg.Document.SHA256,
				},
			})
		}

	case MessageTypeSticker:
		echo.ContentType = plugin.ContentTypeImage
		echo.Metadata["is_sticker"] = "true"
		if msg.Sticker != nil {
			echo.Attachments = append(echo.Attachments, &plugin.Attachment{
				Type:     "sticker",
				URL:      msg.Sticker.ID,
				MimeType: msg.Sticker.MimeType,
				Metadata: map[string]string{
					"media_id": msg.Sticker.ID,
					"sha256":   msg.Sticker.SHA256,
					"animated": fmt.Sprintf("%t", msg.Sticker.Animated),
				},
			})
		}

	case MessageTypeLocation:
		echo.ContentType = plugin.ContentTypeLocation
		if msg.Location != nil {
			locationData, _ := json.Marshal(msg.Location)
			echo.Content = string(locationData)
			echo.Metadata["latitude"] = fmt.Sprintf("%f", msg.Location.Latitude)
			echo.Metadata["longitude"] = fmt.Sprintf("%f", msg.Location.Longitude)
			if msg.Location.Name != "" {
				echo.Metadata["location_name"] = msg.Location.Name
			}
			if msg.Location.Address != "" {
				echo.Metadata["location_address"] = msg.Location.Address
			}
		}

	case MessageTypeContacts:
		echo.ContentType = plugin.ContentTypeContact
		if len(msg.Contacts) > 0 {
			contactsData, _ := json.Marshal(msg.Contacts)
			echo.Content = string(contactsData)
			echo.Metadata["contact_count"] = fmt.Sprintf("%d", len(msg.Contacts))
		}

	default:
		echo.ContentType = plugin.ContentTypeText
		echo.Metadata["original_type"] = string(msg.Type)
	}

	return echo
}

// ToInboundMessage converts a ParsedMessageEcho to plugin.InboundMessage
// This is used to save the echo as a message in the conversation
func (pm *ParsedMessageEcho) ToInboundMessage() *plugin.InboundMessage {
	return &plugin.InboundMessage{
		ExternalID:  pm.ExternalID,
		SenderID:    pm.SenderPhone, // The business phone that sent the message
		SenderName:  pm.SenderPhone, // Use phone as sender name
		ContentType: pm.ContentType,
		Content:     pm.Content,
		Attachments: pm.Attachments,
		Metadata:    pm.Metadata,
		Timestamp:   pm.Timestamp,
	}
}

// IsMessageEchoWebhook checks if the webhook contains message echoes
func IsMessageEchoWebhook(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == string(FieldMessageEchoes) && len(change.Value.Messages) > 0 {
				return true
			}
		}
	}
	return false
}
