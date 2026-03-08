package whatsapp_official

import (
	"context"
	"fmt"

	"github.com/msgfy/linktor/pkg/plugin"
)

// MockMessageHandler captures message handler calls for testing
type MockMessageHandler struct {
	Calls     []*plugin.InboundMessage
	ReturnErr error
}

func (m *MockMessageHandler) Handle(ctx context.Context, msg *plugin.InboundMessage) error {
	m.Calls = append(m.Calls, msg)
	return m.ReturnErr
}

func (m *MockMessageHandler) Handler() plugin.MessageHandler {
	return func(ctx context.Context, msg *plugin.InboundMessage) error {
		return m.Handle(ctx, msg)
	}
}

// MockStatusHandler captures status handler calls for testing
type MockStatusHandler struct {
	Calls     []*plugin.StatusCallback
	ReturnErr error
}

func (m *MockStatusHandler) Handle(ctx context.Context, status *plugin.StatusCallback) error {
	m.Calls = append(m.Calls, status)
	return m.ReturnErr
}

func (m *MockStatusHandler) Handler() plugin.StatusHandler {
	return func(ctx context.Context, status *plugin.StatusCallback) error {
		return m.Handle(ctx, status)
	}
}

// OfficialTestFixtures provides common test data for WhatsApp Official tests
type OfficialTestFixtures struct{}

func NewOfficialTestFixtures() *OfficialTestFixtures {
	return &OfficialTestFixtures{}
}

// ValidConfig returns a valid adapter configuration with all fields
func (f *OfficialTestFixtures) ValidConfig() map[string]string {
	return map[string]string{
		"access_token":    "EAABx0yZ00B0BAKtest1234567890abcdef",
		"phone_number_id": "109876543210",
		"verify_token":    "my_verify_token_secret",
		"webhook_secret":  "whsec_test_webhook_secret_1234",
		"business_id":     "987654321098765",
	}
}

// MinimalConfig returns minimal required configuration
func (f *OfficialTestFixtures) MinimalConfig() map[string]string {
	return map[string]string{
		"access_token":    "EAABx0yZ00B0BAKtest1234567890abcdef",
		"phone_number_id": "109876543210",
		"verify_token":    "my_verify_token_secret",
	}
}

// ConfigMissingAccessToken returns config without access_token
func (f *OfficialTestFixtures) ConfigMissingAccessToken() map[string]string {
	return map[string]string{
		"phone_number_id": "109876543210",
		"verify_token":    "my_verify_token_secret",
		"webhook_secret":  "whsec_test_webhook_secret_1234",
		"business_id":     "987654321098765",
	}
}

// ConfigMissingPhoneNumberID returns config without phone_number_id
func (f *OfficialTestFixtures) ConfigMissingPhoneNumberID() map[string]string {
	return map[string]string{
		"access_token":   "EAABx0yZ00B0BAKtest1234567890abcdef",
		"verify_token":   "my_verify_token_secret",
		"webhook_secret": "whsec_test_webhook_secret_1234",
		"business_id":    "987654321098765",
	}
}

// ConfigMissingVerifyToken returns config without verify_token
func (f *OfficialTestFixtures) ConfigMissingVerifyToken() map[string]string {
	return map[string]string{
		"access_token":    "EAABx0yZ00B0BAKtest1234567890abcdef",
		"phone_number_id": "109876543210",
		"webhook_secret":  "whsec_test_webhook_secret_1234",
		"business_id":     "987654321098765",
	}
}

// SampleTextOutbound returns a sample outbound text message
func (f *OfficialTestFixtures) SampleTextOutbound(to, content string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		Content:     content,
		ContentType: plugin.ContentTypeText,
		Metadata:    make(map[string]string),
	}
}

// SampleImageOutbound returns a sample outbound image message
func (f *OfficialTestFixtures) SampleImageOutbound(to, caption, mediaID string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		Content:     caption,
		ContentType: plugin.ContentTypeImage,
		Metadata: map[string]string{
			"media_id": mediaID,
		},
	}
}

// SampleDocumentOutbound returns a sample outbound document message
func (f *OfficialTestFixtures) SampleDocumentOutbound(to, caption, mediaID, filename string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		Content:     caption,
		ContentType: plugin.ContentTypeDocument,
		Metadata: map[string]string{
			"media_id": mediaID,
			"filename": filename,
		},
	}
}

// SampleLocationOutbound returns a sample outbound location message
func (f *OfficialTestFixtures) SampleLocationOutbound(to string, lat, lon float64, name, address string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		ContentType: plugin.ContentTypeLocation,
		Metadata: map[string]string{
			"latitude":  fmt.Sprintf("%f", lat),
			"longitude": fmt.Sprintf("%f", lon),
			"name":      name,
			"address":   address,
		},
	}
}

// SampleTemplateOutbound returns a sample outbound template message
func (f *OfficialTestFixtures) SampleTemplateOutbound(to, templateName, language string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		ContentType: plugin.ContentTypeTemplate,
		Metadata: map[string]string{
			"template_name":     templateName,
			"template_language": language,
		},
	}
}

// SampleInteractiveOutbound returns a sample outbound interactive message
func (f *OfficialTestFixtures) SampleInteractiveOutbound(to, interactiveType, content string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		Content:     content,
		ContentType: plugin.ContentTypeInteractive,
		Metadata: map[string]string{
			"interactive_type": interactiveType,
		},
	}
}

// SampleTextWebhookPayload returns a valid Meta webhook JSON payload for a text message
func (f *OfficialTestFixtures) SampleTextWebhookPayload(messageID, from, body string) []byte {
	return []byte(fmt.Sprintf(`{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "987654321098765",
      "changes": [
        {
          "field": "messages",
          "value": {
            "messaging_product": "whatsapp",
            "metadata": {
              "display_phone_number": "+5511999990000",
              "phone_number_id": "109876543210"
            },
            "contacts": [
              {
                "wa_id": "%s",
                "profile": {
                  "name": "Test Contact"
                }
              }
            ],
            "messages": [
              {
                "id": "%s",
                "from": "%s",
                "timestamp": "1700000000",
                "type": "text",
                "text": {
                  "body": "%s"
                }
              }
            ]
          }
        }
      ]
    }
  ]
}`, from, messageID, from, body))
}

// SampleImageWebhookPayload returns a valid Meta webhook JSON payload for an image message
func (f *OfficialTestFixtures) SampleImageWebhookPayload(messageID, from, mediaID, caption string) []byte {
	return []byte(fmt.Sprintf(`{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "987654321098765",
      "changes": [
        {
          "field": "messages",
          "value": {
            "messaging_product": "whatsapp",
            "metadata": {
              "display_phone_number": "+5511999990000",
              "phone_number_id": "109876543210"
            },
            "contacts": [
              {
                "wa_id": "%s",
                "profile": {
                  "name": "Test Contact"
                }
              }
            ],
            "messages": [
              {
                "id": "%s",
                "from": "%s",
                "timestamp": "1700000000",
                "type": "image",
                "image": {
                  "id": "%s",
                  "caption": "%s",
                  "mime_type": "image/jpeg",
                  "sha256": "abc123def456"
                }
              }
            ]
          }
        }
      ]
    }
  ]
}`, from, messageID, from, mediaID, caption))
}

// SampleStatusWebhookPayload returns a valid Meta webhook JSON payload for a status update
func (f *OfficialTestFixtures) SampleStatusWebhookPayload(messageID, recipientID, status string) []byte {
	return []byte(fmt.Sprintf(`{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "987654321098765",
      "changes": [
        {
          "field": "messages",
          "value": {
            "messaging_product": "whatsapp",
            "metadata": {
              "display_phone_number": "+5511999990000",
              "phone_number_id": "109876543210"
            },
            "statuses": [
              {
                "id": "%s",
                "recipient_id": "%s",
                "status": "%s",
                "timestamp": "1700000000",
                "conversation": {
                  "id": "conv_001",
                  "origin": {
                    "type": "user_initiated"
                  }
                },
                "pricing": {
                  "billable": true,
                  "pricing_model": "CBP",
                  "category": "service"
                }
              }
            ]
          }
        }
      ]
    }
  ]
}`, messageID, recipientID, status))
}

// SampleTextEchoWebhookPayload returns a valid Meta webhook JSON payload for an echo message
func (f *OfficialTestFixtures) SampleTextEchoWebhookPayload(messageID, to, body string) []byte {
	return []byte(fmt.Sprintf(`{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "987654321098765",
      "changes": [
        {
          "field": "message_echoes",
          "value": {
            "messaging_product": "whatsapp",
            "metadata": {
              "display_phone_number": "+5511999990000",
              "phone_number_id": "109876543210"
            },
            "contacts": [
              {
                "wa_id": "%s",
                "profile": {
                  "name": "Echo Recipient"
                }
              }
            ],
            "messages": [
              {
                "id": "%s",
                "from": "%s",
                "timestamp": "1700000000",
                "type": "text",
                "text": {
                  "body": "%s"
                }
              }
            ]
          }
        }
      ]
    }
  ]
}`, to, messageID, to, body))
}
