package whatsapp_official

import (
	"context"
	"fmt"
)

// CarouselComponent represents a component in a carousel template
type CarouselComponent struct {
	Type       string             `json:"type"`                 // header, body, button, carousel
	Parameters []TemplateParameter `json:"parameters,omitempty"`
	Cards      []CarouselCard     `json:"cards,omitempty"`      // For carousel type
	SubType    string             `json:"sub_type,omitempty"`   // For buttons
	Index      *int               `json:"index,omitempty"`      // For buttons
}

// CarouselCard represents a single card in a carousel (2-10 cards allowed)
type CarouselCard struct {
	CardIndex  int                    `json:"card_index"`
	Components []CarouselCardComponent `json:"components"`
}

// CarouselCardComponent represents a component within a carousel card
type CarouselCardComponent struct {
	Type       string              `json:"type"`                 // header, body, button
	Parameters []TemplateParameter `json:"parameters,omitempty"`
	SubType    string              `json:"sub_type,omitempty"`   // For buttons: quick_reply, url
	Index      *int                `json:"index,omitempty"`      // For button index
}

// CarouselHeaderType defines the allowed header types for carousel cards
type CarouselHeaderType string

const (
	CarouselHeaderImage CarouselHeaderType = "image"
	CarouselHeaderVideo CarouselHeaderType = "video"
)

// CarouselCardInput represents input data for a carousel card
type CarouselCardInput struct {
	// Header media (image or video)
	HeaderMediaID  string // Media ID from Meta
	HeaderMediaURL string // Or URL

	// Body parameters (variable replacements)
	BodyParams []string

	// Button configurations
	Buttons []CarouselButtonInput
}

// CarouselButtonInput represents input for a carousel card button
type CarouselButtonInput struct {
	Type    string // quick_reply, url
	Payload string // For quick_reply: payload; for url: URL suffix
}

// CarouselBuilder helps build carousel template messages
type CarouselBuilder struct {
	name         string
	language     string
	headerType   CarouselHeaderType
	bodyParams   []string // Template body parameters (before carousel)
	cards        []CarouselCard
}

// NewCarouselBuilder creates a new carousel template builder
func NewCarouselBuilder(templateName, languageCode string, headerType CarouselHeaderType) *CarouselBuilder {
	return &CarouselBuilder{
		name:       templateName,
		language:   languageCode,
		headerType: headerType,
		cards:      make([]CarouselCard, 0),
	}
}

// SetBodyParams sets the body parameters for the template (before carousel section)
func (b *CarouselBuilder) SetBodyParams(params ...string) *CarouselBuilder {
	b.bodyParams = params
	return b
}

// AddCard adds a card to the carousel (max 10 cards)
func (b *CarouselBuilder) AddCard(input CarouselCardInput) *CarouselBuilder {
	if len(b.cards) >= 10 {
		return b // Max 10 cards
	}

	cardIndex := len(b.cards)
	card := CarouselCard{
		CardIndex:  cardIndex,
		Components: make([]CarouselCardComponent, 0),
	}

	// Add header component
	headerParams := make([]TemplateParameter, 0)
	if input.HeaderMediaID != "" {
		if b.headerType == CarouselHeaderImage {
			headerParams = append(headerParams, TemplateParameter{
				Type:  "image",
				Image: &MediaObject{ID: input.HeaderMediaID},
			})
		} else {
			headerParams = append(headerParams, TemplateParameter{
				Type:  "video",
				Video: &MediaObject{ID: input.HeaderMediaID},
			})
		}
	} else if input.HeaderMediaURL != "" {
		if b.headerType == CarouselHeaderImage {
			headerParams = append(headerParams, TemplateParameter{
				Type:  "image",
				Image: &MediaObject{Link: input.HeaderMediaURL},
			})
		} else {
			headerParams = append(headerParams, TemplateParameter{
				Type:  "video",
				Video: &MediaObject{Link: input.HeaderMediaURL},
			})
		}
	}

	if len(headerParams) > 0 {
		card.Components = append(card.Components, CarouselCardComponent{
			Type:       "header",
			Parameters: headerParams,
		})
	}

	// Add body component
	if len(input.BodyParams) > 0 {
		bodyParams := make([]TemplateParameter, len(input.BodyParams))
		for i, p := range input.BodyParams {
			bodyParams[i] = TemplateParameter{Type: "text", Text: p}
		}
		card.Components = append(card.Components, CarouselCardComponent{
			Type:       "body",
			Parameters: bodyParams,
		})
	}

	// Add button components
	for i, btn := range input.Buttons {
		idx := i
		buttonParams := []TemplateParameter{
			{Type: "payload", Text: btn.Payload},
		}
		if btn.Type == "url" {
			buttonParams = []TemplateParameter{
				{Type: "text", Text: btn.Payload},
			}
		}

		card.Components = append(card.Components, CarouselCardComponent{
			Type:       "button",
			SubType:    btn.Type,
			Index:      &idx,
			Parameters: buttonParams,
		})
	}

	b.cards = append(b.cards, card)
	return b
}

// AddImageCard is a convenience method to add an image card
func (b *CarouselBuilder) AddImageCard(imageURL string, bodyParams []string, buttons []CarouselButtonInput) *CarouselBuilder {
	return b.AddCard(CarouselCardInput{
		HeaderMediaURL: imageURL,
		BodyParams:     bodyParams,
		Buttons:        buttons,
	})
}

// AddVideoCard is a convenience method to add a video card
func (b *CarouselBuilder) AddVideoCard(videoURL string, bodyParams []string, buttons []CarouselButtonInput) *CarouselBuilder {
	return b.AddCard(CarouselCardInput{
		HeaderMediaURL: videoURL,
		BodyParams:     bodyParams,
		Buttons:        buttons,
	})
}

// BuildRaw returns the raw carousel structure for API submission
// Note: Carousel templates require raw map structure due to nested card components
func (b *CarouselBuilder) BuildRaw() (map[string]interface{}, error) {
	if len(b.cards) < 2 {
		return nil, fmt.Errorf("carousel requires at least 2 cards, got %d", len(b.cards))
	}
	if len(b.cards) > 10 {
		return nil, fmt.Errorf("carousel allows maximum 10 cards, got %d", len(b.cards))
	}

	components := make([]map[string]interface{}, 0)

	// Add body component if there are body params
	if len(b.bodyParams) > 0 {
		bodyParams := make([]map[string]interface{}, len(b.bodyParams))
		for i, p := range b.bodyParams {
			bodyParams[i] = map[string]interface{}{"type": "text", "text": p}
		}
		components = append(components, map[string]interface{}{
			"type":       "body",
			"parameters": bodyParams,
		})
	}

	// Build cards for carousel
	cardsArray := make([]map[string]interface{}, len(b.cards))
	for i, card := range b.cards {
		cardComponents := make([]map[string]interface{}, len(card.Components))
		for j, comp := range card.Components {
			compMap := map[string]interface{}{
				"type": comp.Type,
			}
			if len(comp.Parameters) > 0 {
				params := make([]map[string]interface{}, len(comp.Parameters))
				for k, p := range comp.Parameters {
					paramMap := map[string]interface{}{"type": p.Type}
					if p.Text != "" {
						paramMap["text"] = p.Text
					}
					if p.Image != nil {
						if p.Image.ID != "" {
							paramMap["image"] = map[string]interface{}{"id": p.Image.ID}
						} else if p.Image.Link != "" {
							paramMap["image"] = map[string]interface{}{"link": p.Image.Link}
						}
					}
					if p.Video != nil {
						if p.Video.ID != "" {
							paramMap["video"] = map[string]interface{}{"id": p.Video.ID}
						} else if p.Video.Link != "" {
							paramMap["video"] = map[string]interface{}{"link": p.Video.Link}
						}
					}
					params[k] = paramMap
				}
				compMap["parameters"] = params
			}
			if comp.SubType != "" {
				compMap["sub_type"] = comp.SubType
			}
			if comp.Index != nil {
				compMap["index"] = *comp.Index
			}
			cardComponents[j] = compMap
		}

		cardsArray[i] = map[string]interface{}{
			"card_index": card.CardIndex,
			"components": cardComponents,
		}
	}

	// Add carousel component
	components = append(components, map[string]interface{}{
		"type":  "carousel",
		"cards": cardsArray,
	})

	return map[string]interface{}{
		"name": b.name,
		"language": map[string]interface{}{
			"policy": "deterministic",
			"code":   b.language,
		},
		"components": components,
	}, nil
}

// CarouselSender sends carousel template messages
type CarouselSender struct {
	client *Client
}

// NewCarouselSender creates a new carousel sender
func NewCarouselSender(client *Client) *CarouselSender {
	return &CarouselSender{client: client}
}

// SendCarousel sends a carousel template message
func (s *CarouselSender) SendCarousel(ctx context.Context, to string, builder *CarouselBuilder) (*SendMessageResponse, error) {
	templateRaw, err := builder.BuildRaw()
	if err != nil {
		return nil, fmt.Errorf("failed to build carousel template: %w", err)
	}

	return s.SendCarouselRaw(ctx, to, templateRaw)
}

// SendCarouselRaw sends a carousel using raw map structure
func (s *CarouselSender) SendCarouselRaw(ctx context.Context, to string, templateData map[string]interface{}) (*SendMessageResponse, error) {
	req := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "template",
		"template":          templateData,
	}

	return s.client.SendRawRequest(ctx, req)
}


// CreateProductCarousel creates a carousel for showcasing products
func CreateProductCarousel(templateName, languageCode string, products []ProductCarouselItem) (*CarouselBuilder, error) {
	if len(products) < 2 || len(products) > 10 {
		return nil, fmt.Errorf("product carousel requires 2-10 products, got %d", len(products))
	}

	builder := NewCarouselBuilder(templateName, languageCode, CarouselHeaderImage)

	for _, product := range products {
		buttons := make([]CarouselButtonInput, 0)
		if product.ViewDetailsURL != "" {
			buttons = append(buttons, CarouselButtonInput{
				Type:    "url",
				Payload: product.ViewDetailsURL,
			})
		}
		if product.AddToCartPayload != "" {
			buttons = append(buttons, CarouselButtonInput{
				Type:    "quick_reply",
				Payload: product.AddToCartPayload,
			})
		}

		builder.AddImageCard(
			product.ImageURL,
			[]string{product.Name, product.Price, product.Description},
			buttons,
		)
	}

	return builder, nil
}

// ProductCarouselItem represents a product in a carousel
type ProductCarouselItem struct {
	ImageURL         string
	Name             string
	Price            string
	Description      string
	ViewDetailsURL   string
	AddToCartPayload string
}

// CreateServiceCarousel creates a carousel for showcasing services
func CreateServiceCarousel(templateName, languageCode string, services []ServiceCarouselItem) (*CarouselBuilder, error) {
	if len(services) < 2 || len(services) > 10 {
		return nil, fmt.Errorf("service carousel requires 2-10 services, got %d", len(services))
	}

	builder := NewCarouselBuilder(templateName, languageCode, CarouselHeaderImage)

	for _, service := range services {
		buttons := make([]CarouselButtonInput, 0)
		if service.BookNowPayload != "" {
			buttons = append(buttons, CarouselButtonInput{
				Type:    "quick_reply",
				Payload: service.BookNowPayload,
			})
		}
		if service.LearnMoreURL != "" {
			buttons = append(buttons, CarouselButtonInput{
				Type:    "url",
				Payload: service.LearnMoreURL,
			})
		}

		builder.AddImageCard(
			service.ImageURL,
			[]string{service.Name, service.Duration, service.Price},
			buttons,
		)
	}

	return builder, nil
}

// ServiceCarouselItem represents a service in a carousel
type ServiceCarouselItem struct {
	ImageURL       string
	Name           string
	Duration       string
	Price          string
	BookNowPayload string
	LearnMoreURL   string
}
