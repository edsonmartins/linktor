package whatsapp_official

import (
	"context"
	"fmt"
	"time"
)

// LTOTemplateType defines the type of limited-time offer template
type LTOTemplateType string

const (
	// LTOTypeCountdown shows an expiration countdown
	LTOTypeCountdown LTOTemplateType = "limited_time_offer"
	// LTOTypeCoupon shows a coupon code with copy button
	LTOTypeCoupon LTOTemplateType = "copy_code"
)

// LTOTemplateConfig represents configuration for a Limited-Time Offer template
type LTOTemplateConfig struct {
	// TemplateName is the name of the approved LTO template
	TemplateName string

	// LanguageCode is the template language (e.g., "en", "pt_BR")
	LanguageCode string

	// HeaderImageURL is the URL of the header image
	HeaderImageURL string

	// HeaderImageID is the Media ID if using uploaded media
	HeaderImageID string

	// HeaderVideoURL is the URL of the header video (alternative to image)
	HeaderVideoURL string

	// BodyParams are the variable parameters for the body text
	BodyParams []string

	// ExpirationTime is when the offer expires (for countdown)
	ExpirationTime time.Time

	// CouponCode is the code to be copied (for coupon templates)
	CouponCode string

	// CTAButtonURL is the URL for the CTA button
	CTAButtonURL string

	// OfferType determines the template behavior
	OfferType LTOTemplateType
}

// LTOTemplateBuilder helps build LTO template messages
type LTOTemplateBuilder struct {
	config *LTOTemplateConfig
}

// NewLTOTemplateBuilder creates a new LTO template builder
func NewLTOTemplateBuilder(templateName, languageCode string) *LTOTemplateBuilder {
	return &LTOTemplateBuilder{
		config: &LTOTemplateConfig{
			TemplateName: templateName,
			LanguageCode: languageCode,
			OfferType:    LTOTypeCountdown,
			BodyParams:   make([]string, 0),
		},
	}
}

// SetHeaderImage sets the header image by URL
func (b *LTOTemplateBuilder) SetHeaderImage(url string) *LTOTemplateBuilder {
	b.config.HeaderImageURL = url
	b.config.HeaderVideoURL = ""
	return b
}

// SetHeaderImageID sets the header image by Media ID
func (b *LTOTemplateBuilder) SetHeaderImageID(mediaID string) *LTOTemplateBuilder {
	b.config.HeaderImageID = mediaID
	b.config.HeaderVideoURL = ""
	return b
}

// SetHeaderVideo sets the header video by URL
func (b *LTOTemplateBuilder) SetHeaderVideo(url string) *LTOTemplateBuilder {
	b.config.HeaderVideoURL = url
	b.config.HeaderImageURL = ""
	b.config.HeaderImageID = ""
	return b
}

// SetBodyParams sets the body text parameters
func (b *LTOTemplateBuilder) SetBodyParams(params ...string) *LTOTemplateBuilder {
	b.config.BodyParams = params
	return b
}

// SetExpiration sets the offer expiration time (for countdown)
func (b *LTOTemplateBuilder) SetExpiration(expirationTime time.Time) *LTOTemplateBuilder {
	b.config.ExpirationTime = expirationTime
	b.config.OfferType = LTOTypeCountdown
	return b
}

// SetCouponCode sets the coupon code (for copy-code button)
func (b *LTOTemplateBuilder) SetCouponCode(code string) *LTOTemplateBuilder {
	b.config.CouponCode = code
	b.config.OfferType = LTOTypeCoupon
	return b
}

// SetCTAButton sets the CTA button URL
func (b *LTOTemplateBuilder) SetCTAButton(url string) *LTOTemplateBuilder {
	b.config.CTAButtonURL = url
	return b
}

// BuildRaw creates a raw map structure for the API
func (b *LTOTemplateBuilder) BuildRaw() (map[string]interface{}, error) {
	components := make([]map[string]interface{}, 0)

	// Add header component
	if b.config.HeaderImageURL != "" || b.config.HeaderImageID != "" {
		headerParams := []map[string]interface{}{}
		if b.config.HeaderImageID != "" {
			headerParams = append(headerParams, map[string]interface{}{
				"type":  "image",
				"image": map[string]interface{}{"id": b.config.HeaderImageID},
			})
		} else {
			headerParams = append(headerParams, map[string]interface{}{
				"type":  "image",
				"image": map[string]interface{}{"link": b.config.HeaderImageURL},
			})
		}
		components = append(components, map[string]interface{}{
			"type":       "header",
			"parameters": headerParams,
		})
	} else if b.config.HeaderVideoURL != "" {
		components = append(components, map[string]interface{}{
			"type": "header",
			"parameters": []map[string]interface{}{
				{
					"type":  "video",
					"video": map[string]interface{}{"link": b.config.HeaderVideoURL},
				},
			},
		})
	}

	// Add body component
	if len(b.config.BodyParams) > 0 {
		bodyParams := make([]map[string]interface{}, len(b.config.BodyParams))
		for i, p := range b.config.BodyParams {
			bodyParams[i] = map[string]interface{}{"type": "text", "text": p}
		}
		components = append(components, map[string]interface{}{
			"type":       "body",
			"parameters": bodyParams,
		})
	}

	// Add limited_time_offer component (for countdown)
	if b.config.OfferType == LTOTypeCountdown && !b.config.ExpirationTime.IsZero() {
		components = append(components, map[string]interface{}{
			"type": "limited_time_offer",
			"parameters": []map[string]interface{}{
				{
					"type":                "limited_time_offer",
					"limited_time_offer": map[string]interface{}{
						"expiration_time_ms": b.config.ExpirationTime.UnixMilli(),
					},
				},
			},
		})
	}

	// Add button components
	buttonIndex := 0

	// Copy code button (for coupon templates)
	if b.config.CouponCode != "" {
		components = append(components, map[string]interface{}{
			"type":     "button",
			"sub_type": "copy_code",
			"index":    buttonIndex,
			"parameters": []map[string]interface{}{
				{"type": "coupon_code", "coupon_code": b.config.CouponCode},
			},
		})
		buttonIndex++
	}

	// CTA URL button
	if b.config.CTAButtonURL != "" {
		components = append(components, map[string]interface{}{
			"type":     "button",
			"sub_type": "url",
			"index":    buttonIndex,
			"parameters": []map[string]interface{}{
				{"type": "text", "text": b.config.CTAButtonURL},
			},
		})
	}

	return map[string]interface{}{
		"name": b.config.TemplateName,
		"language": map[string]interface{}{
			"policy": "deterministic",
			"code":   b.config.LanguageCode,
		},
		"components": components,
	}, nil
}

// LTOTemplateSender sends Limited-Time Offer template messages
type LTOTemplateSender struct {
	client *Client
}

// NewLTOTemplateSender creates a new LTO template sender
func NewLTOTemplateSender(client *Client) *LTOTemplateSender {
	return &LTOTemplateSender{client: client}
}

// SendLTO sends a Limited-Time Offer template message
func (s *LTOTemplateSender) SendLTO(ctx context.Context, to string, builder *LTOTemplateBuilder) (*SendMessageResponse, error) {
	templateRaw, err := builder.BuildRaw()
	if err != nil {
		return nil, fmt.Errorf("failed to build LTO template: %w", err)
	}

	req := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "template",
		"template":          templateRaw,
	}

	return s.client.SendRawRequest(ctx, req)
}

// SendCountdownOffer sends a countdown offer with expiration timer
func (s *LTOTemplateSender) SendCountdownOffer(ctx context.Context, to, templateName, languageCode, imageURL string, expirationTime time.Time, bodyParams ...string) (*SendMessageResponse, error) {
	builder := NewLTOTemplateBuilder(templateName, languageCode).
		SetHeaderImage(imageURL).
		SetExpiration(expirationTime).
		SetBodyParams(bodyParams...)

	return s.SendLTO(ctx, to, builder)
}

// SendCouponOffer sends a coupon offer with copy-to-clipboard
func (s *LTOTemplateSender) SendCouponOffer(ctx context.Context, to, templateName, languageCode, imageURL, couponCode, ctaURL string, bodyParams ...string) (*SendMessageResponse, error) {
	builder := NewLTOTemplateBuilder(templateName, languageCode).
		SetHeaderImage(imageURL).
		SetCouponCode(couponCode).
		SetCTAButton(ctaURL).
		SetBodyParams(bodyParams...)

	return s.SendLTO(ctx, to, builder)
}

// SendFlashSale sends a flash sale offer with countdown and coupon
func (s *LTOTemplateSender) SendFlashSale(ctx context.Context, to, templateName, languageCode string, config *FlashSaleConfig) (*SendMessageResponse, error) {
	builder := NewLTOTemplateBuilder(templateName, languageCode)

	if config.ImageURL != "" {
		builder.SetHeaderImage(config.ImageURL)
	}

	if !config.EndTime.IsZero() {
		builder.SetExpiration(config.EndTime)
	}

	if config.CouponCode != "" {
		builder.SetCouponCode(config.CouponCode)
	}

	if config.ShopNowURL != "" {
		builder.SetCTAButton(config.ShopNowURL)
	}

	// Build body params
	bodyParams := []string{
		config.ProductName,
		config.OriginalPrice,
		config.SalePrice,
		config.DiscountPercentage,
	}
	builder.SetBodyParams(bodyParams...)

	return s.SendLTO(ctx, to, builder)
}

// FlashSaleConfig represents configuration for a flash sale
type FlashSaleConfig struct {
	ImageURL           string
	ProductName        string
	OriginalPrice      string
	SalePrice          string
	DiscountPercentage string
	CouponCode         string
	EndTime            time.Time
	ShopNowURL         string
}

// CouponTemplateBuilder helps build coupon-specific templates
type CouponTemplateBuilder struct {
	config *CouponConfig
}

// CouponConfig represents configuration for a coupon template
type CouponConfig struct {
	TemplateName  string
	LanguageCode  string
	HeaderImage   string
	StoreName     string
	DiscountText  string
	CouponCode    string
	ValidUntil    time.Time
	TermsURL      string
	RedeemURL     string
}

// NewCouponTemplateBuilder creates a new coupon template builder
func NewCouponTemplateBuilder(templateName, languageCode string) *CouponTemplateBuilder {
	return &CouponTemplateBuilder{
		config: &CouponConfig{
			TemplateName: templateName,
			LanguageCode: languageCode,
		},
	}
}

// SetHeader sets the coupon header image
func (b *CouponTemplateBuilder) SetHeader(imageURL string) *CouponTemplateBuilder {
	b.config.HeaderImage = imageURL
	return b
}

// SetStore sets the store name
func (b *CouponTemplateBuilder) SetStore(name string) *CouponTemplateBuilder {
	b.config.StoreName = name
	return b
}

// SetDiscount sets the discount text
func (b *CouponTemplateBuilder) SetDiscount(text string) *CouponTemplateBuilder {
	b.config.DiscountText = text
	return b
}

// SetCouponCode sets the coupon code
func (b *CouponTemplateBuilder) SetCouponCode(code string) *CouponTemplateBuilder {
	b.config.CouponCode = code
	return b
}

// SetValidity sets the coupon validity period
func (b *CouponTemplateBuilder) SetValidity(validUntil time.Time) *CouponTemplateBuilder {
	b.config.ValidUntil = validUntil
	return b
}

// SetTermsURL sets the terms and conditions URL
func (b *CouponTemplateBuilder) SetTermsURL(url string) *CouponTemplateBuilder {
	b.config.TermsURL = url
	return b
}

// SetRedeemURL sets the redeem URL
func (b *CouponTemplateBuilder) SetRedeemURL(url string) *CouponTemplateBuilder {
	b.config.RedeemURL = url
	return b
}

// BuildRaw creates a raw map structure for the API
func (b *CouponTemplateBuilder) BuildRaw() (map[string]interface{}, error) {
	if b.config.CouponCode == "" {
		return nil, fmt.Errorf("coupon code is required")
	}

	components := make([]map[string]interface{}, 0)

	// Add header component
	if b.config.HeaderImage != "" {
		components = append(components, map[string]interface{}{
			"type": "header",
			"parameters": []map[string]interface{}{
				{
					"type":  "image",
					"image": map[string]interface{}{"link": b.config.HeaderImage},
				},
			},
		})
	}

	// Add body component with parameters
	bodyParams := make([]map[string]interface{}, 0)
	if b.config.StoreName != "" {
		bodyParams = append(bodyParams, map[string]interface{}{"type": "text", "text": b.config.StoreName})
	}
	if b.config.DiscountText != "" {
		bodyParams = append(bodyParams, map[string]interface{}{"type": "text", "text": b.config.DiscountText})
	}
	if !b.config.ValidUntil.IsZero() {
		bodyParams = append(bodyParams, map[string]interface{}{"type": "text", "text": b.config.ValidUntil.Format("02/01/2006")})
	}

	if len(bodyParams) > 0 {
		components = append(components, map[string]interface{}{
			"type":       "body",
			"parameters": bodyParams,
		})
	}

	// Add copy code button
	buttonIndex := 0
	components = append(components, map[string]interface{}{
		"type":     "button",
		"sub_type": "copy_code",
		"index":    buttonIndex,
		"parameters": []map[string]interface{}{
			{"type": "coupon_code", "coupon_code": b.config.CouponCode},
		},
	})
	buttonIndex++

	// Add URL buttons
	if b.config.RedeemURL != "" {
		components = append(components, map[string]interface{}{
			"type":     "button",
			"sub_type": "url",
			"index":    buttonIndex,
			"parameters": []map[string]interface{}{
				{"type": "text", "text": b.config.RedeemURL},
			},
		})
		buttonIndex++
	}

	if b.config.TermsURL != "" {
		components = append(components, map[string]interface{}{
			"type":     "button",
			"sub_type": "url",
			"index":    buttonIndex,
			"parameters": []map[string]interface{}{
				{"type": "text", "text": b.config.TermsURL},
			},
		})
	}

	return map[string]interface{}{
		"name": b.config.TemplateName,
		"language": map[string]interface{}{
			"policy": "deterministic",
			"code":   b.config.LanguageCode,
		},
		"components": components,
	}, nil
}

// CouponTemplateSender sends coupon template messages
type CouponTemplateSender struct {
	client *Client
}

// NewCouponTemplateSender creates a new coupon template sender
func NewCouponTemplateSender(client *Client) *CouponTemplateSender {
	return &CouponTemplateSender{client: client}
}

// SendCoupon sends a coupon template message
func (s *CouponTemplateSender) SendCoupon(ctx context.Context, to string, builder *CouponTemplateBuilder) (*SendMessageResponse, error) {
	templateRaw, err := builder.BuildRaw()
	if err != nil {
		return nil, fmt.Errorf("failed to build coupon template: %w", err)
	}

	req := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "template",
		"template":          templateRaw,
	}

	return s.client.SendRawRequest(ctx, req)
}

// SendSimpleCoupon sends a simple coupon with just the code
func (s *CouponTemplateSender) SendSimpleCoupon(ctx context.Context, to, templateName, languageCode, couponCode string) (*SendMessageResponse, error) {
	builder := NewCouponTemplateBuilder(templateName, languageCode).
		SetCouponCode(couponCode)

	return s.SendCoupon(ctx, to, builder)
}

// SendFullCoupon sends a complete coupon with all details
func (s *CouponTemplateSender) SendFullCoupon(ctx context.Context, to string, config *CouponConfig) (*SendMessageResponse, error) {
	builder := NewCouponTemplateBuilder(config.TemplateName, config.LanguageCode).
		SetHeader(config.HeaderImage).
		SetStore(config.StoreName).
		SetDiscount(config.DiscountText).
		SetCouponCode(config.CouponCode).
		SetValidity(config.ValidUntil).
		SetTermsURL(config.TermsURL).
		SetRedeemURL(config.RedeemURL)

	return s.SendCoupon(ctx, to, builder)
}

// PromotionScheduler helps schedule and send promotional messages
type PromotionScheduler struct {
	sender *LTOTemplateSender
}

// NewPromotionScheduler creates a new promotion scheduler
func NewPromotionScheduler(client *Client) *PromotionScheduler {
	return &PromotionScheduler{
		sender: NewLTOTemplateSender(client),
	}
}

// SendPromotion sends a promotion to all recipients
func (s *PromotionScheduler) SendPromotion(ctx context.Context, promo *ScheduledPromotion) ([]SendResult, error) {
	if !promo.IsActive() {
		return nil, fmt.Errorf("promotion is not active")
	}

	builder := NewLTOTemplateBuilder(promo.TemplateName, promo.LanguageCode)

	if promo.ImageURL != "" {
		builder.SetHeaderImage(promo.ImageURL)
	}
	if !promo.EndTime.IsZero() {
		builder.SetExpiration(promo.EndTime)
	}
	if promo.CouponCode != "" {
		builder.SetCouponCode(promo.CouponCode)
	}
	if len(promo.BodyParams) > 0 {
		builder.SetBodyParams(promo.BodyParams...)
	}

	results := make([]SendResult, 0, len(promo.Recipients))
	for _, recipient := range promo.Recipients {
		resp, err := s.sender.SendLTO(ctx, recipient, builder)
		result := SendResult{
			Recipient: recipient,
		}
		if err != nil {
			result.Error = err.Error()
			result.Success = false
		} else {
			result.Success = true
			if len(resp.Messages) > 0 {
				result.MessageID = resp.Messages[0].ID
			}
		}
		results = append(results, result)
	}

	return results, nil
}

// SendResult represents the result of sending a promotion to a recipient
type SendResult struct {
	Recipient string
	Success   bool
	MessageID string
	Error     string
}

// ScheduledPromotion represents a scheduled promotional message
type ScheduledPromotion struct {
	ID              string
	TemplateName    string
	LanguageCode    string
	Recipients      []string
	ImageURL        string
	BodyParams      []string
	CouponCode      string
	StartTime       time.Time
	EndTime         time.Time
	SendAtStart     bool
	SendReminders   bool
	ReminderHours   []int // Hours before end to send reminder
}

// GetTimeUntilExpiration returns the duration until expiration
func (p *ScheduledPromotion) GetTimeUntilExpiration() time.Duration {
	return time.Until(p.EndTime)
}

// IsActive checks if the promotion is currently active
func (p *ScheduledPromotion) IsActive() bool {
	now := time.Now()
	return now.After(p.StartTime) && now.Before(p.EndTime)
}

// IsExpired checks if the promotion has expired
func (p *ScheduledPromotion) IsExpired() bool {
	return time.Now().After(p.EndTime)
}

// FormatCountdown formats the time remaining as a human-readable string
func (p *ScheduledPromotion) FormatCountdown() string {
	remaining := p.GetTimeUntilExpiration()
	if remaining <= 0 {
		return "Expired"
	}

	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		return fmt.Sprintf("%d days, %d hours", days, hours%24)
	}

	return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
}
