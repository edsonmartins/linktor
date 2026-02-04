package whatsapp_official

import (
	"context"
	"encoding/json"
	"fmt"
)

// InteractiveObject represents an interactive message
type InteractiveObject struct {
	Type   string              `json:"type"` // button, list, product, product_list
	Header *InteractiveHeader  `json:"header,omitempty"`
	Body   *InteractiveBody    `json:"body"`
	Footer *InteractiveFooter  `json:"footer,omitempty"`
	Action *InteractiveAction  `json:"action"`
}

// InteractiveHeader represents the header of an interactive message
type InteractiveHeader struct {
	Type     string       `json:"type"` // text, image, video, document
	Text     string       `json:"text,omitempty"`
	Image    *MediaObject `json:"image,omitempty"`
	Video    *MediaObject `json:"video,omitempty"`
	Document *DocumentObject `json:"document,omitempty"`
}

// InteractiveBody represents the body of an interactive message
type InteractiveBody struct {
	Text string `json:"text"`
}

// InteractiveFooter represents the footer of an interactive message
type InteractiveFooter struct {
	Text string `json:"text"`
}

// InteractiveAction represents the action of an interactive message
type InteractiveAction struct {
	// For button type
	Buttons []InteractiveButton `json:"buttons,omitempty"`

	// For list type
	Button   string        `json:"button,omitempty"` // Button text for list
	Sections []ListSection `json:"sections,omitempty"`

	// For product types
	CatalogID         string          `json:"catalog_id,omitempty"`
	ProductRetailerID string          `json:"product_retailer_id,omitempty"`
	ProductSections   []ProductSection `json:"product_sections,omitempty"`
}

// InteractiveButton represents a button in interactive message (max 3)
type InteractiveButton struct {
	Type  string          `json:"type"` // reply
	Reply *ButtonReply    `json:"reply"`
}

// ButtonReply represents the reply content of a button
type ButtonReply struct {
	ID    string `json:"id"`    // Max 256 chars
	Title string `json:"title"` // Max 20 chars
}

// ListSection represents a section in a list message
type ListSection struct {
	Title string     `json:"title,omitempty"` // Max 24 chars
	Rows  []ListRow  `json:"rows"`            // Max 10 rows per section
}

// ListRow represents a row in a list section
type ListRow struct {
	ID          string `json:"id"`          // Max 200 chars
	Title       string `json:"title"`       // Max 24 chars
	Description string `json:"description,omitempty"` // Max 72 chars
}

// ProductSection represents a section in a product list
type ProductSection struct {
	Title    string       `json:"title,omitempty"`
	Products []ProductItem `json:"product_items"`
}

// ProductItem represents a product in a product section
type ProductItem struct {
	ProductRetailerID string `json:"product_retailer_id"`
}

// InteractiveBuilder helps build interactive messages
type InteractiveBuilder struct {
	interactive *InteractiveObject
}

// NewButtonMessageBuilder creates a builder for button messages
func NewButtonMessageBuilder(bodyText string) *InteractiveBuilder {
	return &InteractiveBuilder{
		interactive: &InteractiveObject{
			Type: "button",
			Body: &InteractiveBody{Text: bodyText},
			Action: &InteractiveAction{
				Buttons: []InteractiveButton{},
			},
		},
	}
}

// NewListMessageBuilder creates a builder for list messages
func NewListMessageBuilder(bodyText, buttonText string) *InteractiveBuilder {
	return &InteractiveBuilder{
		interactive: &InteractiveObject{
			Type: "list",
			Body: &InteractiveBody{Text: bodyText},
			Action: &InteractiveAction{
				Button:   buttonText,
				Sections: []ListSection{},
			},
		},
	}
}

// SetHeader sets a text header
func (b *InteractiveBuilder) SetHeader(text string) *InteractiveBuilder {
	b.interactive.Header = &InteractiveHeader{
		Type: "text",
		Text: text,
	}
	return b
}

// SetImageHeader sets an image header
func (b *InteractiveBuilder) SetImageHeader(mediaID string) *InteractiveBuilder {
	b.interactive.Header = &InteractiveHeader{
		Type:  "image",
		Image: &MediaObject{ID: mediaID},
	}
	return b
}

// SetImageHeaderURL sets an image header using URL
func (b *InteractiveBuilder) SetImageHeaderURL(url string) *InteractiveBuilder {
	b.interactive.Header = &InteractiveHeader{
		Type:  "image",
		Image: &MediaObject{Link: url},
	}
	return b
}

// SetVideoHeader sets a video header
func (b *InteractiveBuilder) SetVideoHeader(mediaID string) *InteractiveBuilder {
	b.interactive.Header = &InteractiveHeader{
		Type:  "video",
		Video: &MediaObject{ID: mediaID},
	}
	return b
}

// SetDocumentHeader sets a document header
func (b *InteractiveBuilder) SetDocumentHeader(mediaID, filename string) *InteractiveBuilder {
	b.interactive.Header = &InteractiveHeader{
		Type:     "document",
		Document: &DocumentObject{ID: mediaID, Filename: filename},
	}
	return b
}

// SetFooter sets the footer text
func (b *InteractiveBuilder) SetFooter(text string) *InteractiveBuilder {
	b.interactive.Footer = &InteractiveFooter{Text: text}
	return b
}

// AddButton adds a reply button (max 3)
func (b *InteractiveBuilder) AddButton(id, title string) *InteractiveBuilder {
	if len(b.interactive.Action.Buttons) >= 3 {
		return b // Max 3 buttons allowed
	}

	// Truncate title if too long
	if len(title) > 20 {
		title = title[:20]
	}

	b.interactive.Action.Buttons = append(b.interactive.Action.Buttons, InteractiveButton{
		Type: "reply",
		Reply: &ButtonReply{
			ID:    id,
			Title: title,
		},
	})
	return b
}

// AddSection adds a section to a list message
func (b *InteractiveBuilder) AddSection(title string, rows []ListRow) *InteractiveBuilder {
	// Truncate title if too long
	if len(title) > 24 {
		title = title[:24]
	}

	// Limit to 10 rows per section
	if len(rows) > 10 {
		rows = rows[:10]
	}

	b.interactive.Action.Sections = append(b.interactive.Action.Sections, ListSection{
		Title: title,
		Rows:  rows,
	})
	return b
}

// AddListRow is a convenience method to add a single row to the last section
func (b *InteractiveBuilder) AddListRow(id, title, description string) *InteractiveBuilder {
	// Truncate fields if too long
	if len(title) > 24 {
		title = title[:24]
	}
	if len(description) > 72 {
		description = description[:72]
	}
	if len(id) > 200 {
		id = id[:200]
	}

	row := ListRow{
		ID:          id,
		Title:       title,
		Description: description,
	}

	// Add to last section or create new one
	if len(b.interactive.Action.Sections) == 0 {
		b.interactive.Action.Sections = []ListSection{{Rows: []ListRow{row}}}
	} else {
		lastIdx := len(b.interactive.Action.Sections) - 1
		b.interactive.Action.Sections[lastIdx].Rows = append(
			b.interactive.Action.Sections[lastIdx].Rows,
			row,
		)
	}

	return b
}

// Build returns the built interactive object
func (b *InteractiveBuilder) Build() *InteractiveObject {
	return b.interactive
}

// ToJSON converts the interactive message to JSON
func (b *InteractiveBuilder) ToJSON() (string, error) {
	data, err := json.Marshal(b.interactive)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// InteractiveSender sends interactive messages
type InteractiveSender struct {
	client *Client
}

// NewInteractiveSender creates a new interactive sender
func NewInteractiveSender(client *Client) *InteractiveSender {
	return &InteractiveSender{
		client: client,
	}
}

// SendInteractive sends an interactive message
func (s *InteractiveSender) SendInteractive(ctx context.Context, to string, interactive *InteractiveObject) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:          to,
		Type:        MessageTypeInteractive,
		Interactive: interactive,
	}
	return s.client.SendMessage(ctx, req)
}

// SendButtonMessage sends a button message (max 3 buttons)
func (s *InteractiveSender) SendButtonMessage(ctx context.Context, to, bodyText string, buttons []ButtonReply) (*SendMessageResponse, error) {
	if len(buttons) > 3 {
		return nil, fmt.Errorf("maximum 3 buttons allowed")
	}

	builder := NewButtonMessageBuilder(bodyText)
	for _, btn := range buttons {
		builder.AddButton(btn.ID, btn.Title)
	}

	return s.SendInteractive(ctx, to, builder.Build())
}

// SendListMessage sends a list message
func (s *InteractiveSender) SendListMessage(ctx context.Context, to, bodyText, buttonText string, sections []ListSection) (*SendMessageResponse, error) {
	builder := NewListMessageBuilder(bodyText, buttonText)

	for _, section := range sections {
		builder.AddSection(section.Title, section.Rows)
	}

	return s.SendInteractive(ctx, to, builder.Build())
}

// SendSimpleListMessage sends a simple list message with a single section
func (s *InteractiveSender) SendSimpleListMessage(ctx context.Context, to, bodyText, buttonText, sectionTitle string, rows []ListRow) (*SendMessageResponse, error) {
	return s.SendListMessage(ctx, to, bodyText, buttonText, []ListSection{
		{Title: sectionTitle, Rows: rows},
	})
}

// CreateQuickReplyButtons creates buttons from simple string options
func CreateQuickReplyButtons(options ...string) []ButtonReply {
	buttons := make([]ButtonReply, 0, len(options))
	for i, opt := range options {
		if i >= 3 {
			break // Max 3 buttons
		}
		buttons = append(buttons, ButtonReply{
			ID:    fmt.Sprintf("option_%d", i+1),
			Title: opt,
		})
	}
	return buttons
}

// CreateListRows creates list rows from simple options
func CreateListRows(options map[string]string) []ListRow {
	rows := make([]ListRow, 0, len(options))
	i := 0
	for title, description := range options {
		rows = append(rows, ListRow{
			ID:          fmt.Sprintf("row_%d", i+1),
			Title:       title,
			Description: description,
		})
		i++
		if i >= 10 {
			break // Max 10 rows per section
		}
	}
	return rows
}

// InteractiveFromJSON creates an interactive object from JSON
func InteractiveFromJSON(jsonStr string) (*InteractiveObject, error) {
	var interactive InteractiveObject
	if err := json.Unmarshal([]byte(jsonStr), &interactive); err != nil {
		return nil, fmt.Errorf("failed to parse interactive JSON: %w", err)
	}
	return &interactive, nil
}

// ShouldUseList determines if a list should be used instead of buttons
// Based on WhatsApp recommendation: use buttons for ≤3 options, list for >3
func ShouldUseList(optionCount int) bool {
	return optionCount > 3
}

// CreateMenu creates an appropriate interactive message based on option count
func CreateMenu(bodyText, menuButtonText string, options []struct{ ID, Title, Description string }) *InteractiveObject {
	if len(options) <= 3 {
		// Use buttons
		builder := NewButtonMessageBuilder(bodyText)
		for _, opt := range options {
			builder.AddButton(opt.ID, opt.Title)
		}
		return builder.Build()
	}

	// Use list
	builder := NewListMessageBuilder(bodyText, menuButtonText)
	rows := make([]ListRow, 0, len(options))
	for _, opt := range options {
		rows = append(rows, ListRow{
			ID:          opt.ID,
			Title:       opt.Title,
			Description: opt.Description,
		})
	}
	builder.AddSection("", rows)
	return builder.Build()
}

// CTA URL Button types (for templates)
type CTAURLButton struct {
	Type  string `json:"type"` // "url"
	Title string `json:"title"`
	URL   string `json:"url"`
}

// CTA Phone Button
type CTAPhoneButton struct {
	Type        string `json:"type"` // "phone_number"
	Title       string `json:"title"`
	PhoneNumber string `json:"phone_number"`
}
