package whatsapp_official

import (
	"context"
	"encoding/json"
	"fmt"
)

// TemplateObject represents a template message to send
type TemplateObject struct {
	Name       string              `json:"name"`
	Language   *TemplateLanguage   `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
}

// TemplateLanguage represents the template language
type TemplateLanguage struct {
	Policy string `json:"policy,omitempty"` // deterministic
	Code   string `json:"code"`             // en, pt_BR, es, etc.
}

// TemplateComponent represents a component in a template message.
// For buttons Type=="button" and SubType narrows the kind.
// For carousels Type=="carousel" and Cards carries the per-card
// sub-components (each card gets its own body + header + button row).
type TemplateComponent struct {
	Type       string              `json:"type"` // header, body, button, carousel
	SubType    string              `json:"sub_type,omitempty"`
	Index      *int                `json:"index,omitempty"`
	Parameters []TemplateParameter `json:"parameters,omitempty"`
	Cards      []TemplateCard      `json:"cards,omitempty"`
}

// TemplateCard is one card in a carousel send payload. CardIndex binds the
// runtime components to the nth card on the approved template definition.
type TemplateCard struct {
	CardIndex  int                 `json:"card_index"`
	Components []TemplateComponent `json:"components"`
}

// TemplateParameter represents a parameter in a template component.
// The Type field names the field that holds the value:
//   - text / payload / coupon_code → Text
//   - currency → Currency
//   - date_time → DateTime
//   - image / video → Image / Video (MediaObject with ID or Link)
//   - document → Document (DocumentObject)
//   - location → Location (LocationObject)
//   - action → Action (flow button navigation payload)
//
// ParameterName is set only when the template was created with
// parameter_format=NAMED; Meta then uses the name to match the value to
// the correct placeholder instead of relying on array position.
type TemplateParameter struct {
	Type          string                 `json:"type"`
	ParameterName string                 `json:"parameter_name,omitempty"`
	Text          string                 `json:"text,omitempty"`
	Currency      *CurrencyParameter     `json:"currency,omitempty"`
	DateTime      *DateTimeParameter     `json:"date_time,omitempty"`
	Image         *MediaObject           `json:"image,omitempty"`
	Document      *DocumentObject        `json:"document,omitempty"`
	Video         *MediaObject           `json:"video,omitempty"`
	Location      *LocationObject        `json:"location,omitempty"`
	Action        map[string]interface{} `json:"action,omitempty"`
}

// CurrencyParameter represents a currency parameter
type CurrencyParameter struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"` // USD, EUR, BRL, etc.
	Amount1000    int64  `json:"amount_1000"` // Amount in 1/1000 units
}

// DateTimeParameter represents a date/time parameter
type DateTimeParameter struct {
	FallbackValue string `json:"fallback_value"`
}

// TemplateBuilder helps build template messages
type TemplateBuilder struct {
	template *TemplateObject
}

// NewTemplateBuilder creates a new template builder
func NewTemplateBuilder(name, languageCode string) *TemplateBuilder {
	return &TemplateBuilder{
		template: &TemplateObject{
			Name: name,
			Language: &TemplateLanguage{
				Policy: "deterministic",
				Code:   languageCode,
			},
			Components: []TemplateComponent{},
		},
	}
}

// AddHeaderText adds a text header component
func (b *TemplateBuilder) AddHeaderText(text string) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "header",
		Parameters: []TemplateParameter{
			{Type: "text", Text: text},
		},
	})
	return b
}

// AddHeaderImage adds an image header component
func (b *TemplateBuilder) AddHeaderImage(mediaID string) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "header",
		Parameters: []TemplateParameter{
			{Type: "image", Image: &MediaObject{ID: mediaID}},
		},
	})
	return b
}

// AddHeaderImageURL adds an image header component using URL
func (b *TemplateBuilder) AddHeaderImageURL(url string) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "header",
		Parameters: []TemplateParameter{
			{Type: "image", Image: &MediaObject{Link: url}},
		},
	})
	return b
}

// AddHeaderDocument adds a document header component
func (b *TemplateBuilder) AddHeaderDocument(mediaID, filename string) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "header",
		Parameters: []TemplateParameter{
			{Type: "document", Document: &DocumentObject{ID: mediaID, Filename: filename}},
		},
	})
	return b
}

// AddHeaderVideo adds a video header component
func (b *TemplateBuilder) AddHeaderVideo(mediaID string) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "header",
		Parameters: []TemplateParameter{
			{Type: "video", Video: &MediaObject{ID: mediaID}},
		},
	})
	return b
}

// AddBodyParameters adds body text parameters
func (b *TemplateBuilder) AddBodyParameters(params ...string) *TemplateBuilder {
	parameters := make([]TemplateParameter, len(params))
	for i, p := range params {
		parameters[i] = TemplateParameter{Type: "text", Text: p}
	}
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type:       "body",
		Parameters: parameters,
	})
	return b
}

// AddBodyCurrency adds a currency parameter to the body
func (b *TemplateBuilder) AddBodyCurrency(fallback, code string, amount1000 int64) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "body",
		Parameters: []TemplateParameter{
			{
				Type: "currency",
				Currency: &CurrencyParameter{
					FallbackValue: fallback,
					Code:          code,
					Amount1000:    amount1000,
				},
			},
		},
	})
	return b
}

// AddBodyDateTime adds a date/time parameter to the body
func (b *TemplateBuilder) AddBodyDateTime(fallback string) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "body",
		Parameters: []TemplateParameter{
			{
				Type: "date_time",
				DateTime: &DateTimeParameter{
					FallbackValue: fallback,
				},
			},
		},
	})
	return b
}

// AddQuickReplyButton adds a quick reply button
func (b *TemplateBuilder) AddQuickReplyButton(index int, payload string) *TemplateBuilder {
	idx := index
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type:    "button",
		SubType: "quick_reply",
		Index:   &idx,
		Parameters: []TemplateParameter{
			{Type: "payload", Text: payload},
		},
	})
	return b
}

// AddURLButton adds a URL button with dynamic parameter
func (b *TemplateBuilder) AddURLButton(index int, urlSuffix string) *TemplateBuilder {
	idx := index
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type:    "button",
		SubType: "url",
		Index:   &idx,
		Parameters: []TemplateParameter{
			{Type: "text", Text: urlSuffix},
		},
	})
	return b
}

// AddPhoneNumberButton adds a click-to-call button. Phone number buttons on
// templates have no runtime parameters — the phone number is baked in at
// create time — but Meta still expects the button block in the send payload
// when the template defines one.
func (b *TemplateBuilder) AddPhoneNumberButton(index int) *TemplateBuilder {
	idx := index
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type:    "button",
		SubType: "phone_number",
		Index:   &idx,
	})
	return b
}

// AddCopyCodeButton adds a copy-code button (authentication templates). The
// coupon is the dynamic value the user will copy — typically the OTP itself.
func (b *TemplateBuilder) AddCopyCodeButton(index int, coupon string) *TemplateBuilder {
	idx := index
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type:    "button",
		SubType: "copy_code",
		Index:   &idx,
		Parameters: []TemplateParameter{
			{Type: "coupon_code", Text: coupon},
		},
	})
	return b
}

// AddOTPButton adds a one-time password button for authentication templates.
// `otp` is the generated code that Meta will surface to the user via
// one-tap/zero-tap autofill or copy-to-clipboard depending on how the
// template was created.
func (b *TemplateBuilder) AddOTPButton(index int, otp string) *TemplateBuilder {
	idx := index
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type:    "button",
		SubType: "url",
		Index:   &idx,
		Parameters: []TemplateParameter{
			{Type: "text", Text: otp},
		},
	})
	return b
}

// AddFlowButton adds a button that opens a WhatsApp Flow. `flowToken` tracks
// the user's flow session and `initialData` feeds the first screen.
func (b *TemplateBuilder) AddFlowButton(index int, flowToken string, initialData map[string]interface{}) *TemplateBuilder {
	idx := index
	payload := map[string]interface{}{
		"flow_token": flowToken,
	}
	if initialData != nil {
		payload["flow_action_data"] = initialData
	}
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type:    "button",
		SubType: "flow",
		Index:   &idx,
		Parameters: []TemplateParameter{
			{Type: "action", Action: payload},
		},
	})
	return b
}

// AddHeaderLocation adds a location header. The entity already modelled the
// LOCATION format but the builder didn't expose it — this closes the gap so
// delivery/dispatch templates that carry a location can actually be sent.
func (b *TemplateBuilder) AddHeaderLocation(latitude, longitude float64, name, address string) *TemplateBuilder {
	b.template.Components = append(b.template.Components, TemplateComponent{
		Type: "header",
		Parameters: []TemplateParameter{
			{
				Type: "location",
				Location: &LocationObject{
					Latitude:  latitude,
					Longitude: longitude,
					Name:      name,
					Address:   address,
				},
			},
		},
	})
	return b
}

// Build returns the built template object
func (b *TemplateBuilder) Build() *TemplateObject {
	return b.template
}

// ToJSON converts the template to JSON
func (b *TemplateBuilder) ToJSON() (string, error) {
	data, err := json.Marshal(b.template)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TemplateSender sends template messages
type TemplateSender struct {
	client *Client
}

// NewTemplateSender creates a new template sender
func NewTemplateSender(client *Client) *TemplateSender {
	return &TemplateSender{
		client: client,
	}
}

// SendTemplate sends a template message
func (s *TemplateSender) SendTemplate(ctx context.Context, to string, template *TemplateObject) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:       to,
		Type:     MessageType("template"),
		Template: template,
	}
	return s.client.SendMessage(ctx, req)
}

// SendSimpleTemplate sends a simple template without parameters
func (s *TemplateSender) SendSimpleTemplate(ctx context.Context, to, templateName, languageCode string) (*SendMessageResponse, error) {
	template := NewTemplateBuilder(templateName, languageCode).Build()
	return s.SendTemplate(ctx, to, template)
}

// SendTemplateWithBodyParams sends a template with body text parameters
func (s *TemplateSender) SendTemplateWithBodyParams(ctx context.Context, to, templateName, languageCode string, params ...string) (*SendMessageResponse, error) {
	template := NewTemplateBuilder(templateName, languageCode).
		AddBodyParameters(params...).
		Build()
	return s.SendTemplate(ctx, to, template)
}

// SendTemplateWithImage sends a template with image header
func (s *TemplateSender) SendTemplateWithImage(ctx context.Context, to, templateName, languageCode, imageURL string, bodyParams ...string) (*SendMessageResponse, error) {
	builder := NewTemplateBuilder(templateName, languageCode).
		AddHeaderImageURL(imageURL)

	if len(bodyParams) > 0 {
		builder.AddBodyParameters(bodyParams...)
	}

	return s.SendTemplate(ctx, to, builder.Build())
}

// SendTemplateWithDocument sends a template with document header
func (s *TemplateSender) SendTemplateWithDocument(ctx context.Context, to, templateName, languageCode, mediaID, filename string, bodyParams ...string) (*SendMessageResponse, error) {
	builder := NewTemplateBuilder(templateName, languageCode).
		AddHeaderDocument(mediaID, filename)

	if len(bodyParams) > 0 {
		builder.AddBodyParameters(bodyParams...)
	}

	return s.SendTemplate(ctx, to, builder.Build())
}

// TemplateFromJSON creates a template object from JSON
func TemplateFromJSON(jsonStr string) (*TemplateObject, error) {
	var template TemplateObject
	if err := json.Unmarshal([]byte(jsonStr), &template); err != nil {
		return nil, fmt.Errorf("failed to parse template JSON: %w", err)
	}
	return &template, nil
}

// TemplateInfo represents information about a template
type TemplateInfo struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Category   string   `json:"category"`
	Language   string   `json:"language"`
	Components []string `json:"components,omitempty"`
}

// Use-case factory helpers (CreateOTPTemplate, CreateOrderConfirmationTemplate,
// CreateAppointmentReminderTemplate, CreateShippingUpdateTemplate) were
// removed as part of the P3 cleanup. Their names implied they created
// templates via the Management API (Create*), but in reality they produced
// outgoing *TemplateObject payloads — conflating the two concepts.
// Callers should now use TemplateBuilder directly, or BuildSendPayload
// when turning a stored entity.Template into a send payload.
