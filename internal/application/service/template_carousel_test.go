package service

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// CAROUSEL
// -----------------------------------------------------------------------------

func TestValidateCarousel_RequiresAtLeastOneCard(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "CAROUSEL"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one card")
}

func TestValidateCarousel_RejectsMoreThan10Cards(t *testing.T) {
	cards := make([]entity.TemplateCarouselCard, 11)
	for i := range cards {
		cards[i] = entity.TemplateCarouselCard{
			Components: []entity.TemplateComponent{
				{Type: "BODY", Text: "Card"},
			},
		}
	}

	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "CAROUSEL", Cards: cards},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 10")
}

func TestValidateCarousel_CardRequiresBody(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "CAROUSEL", Cards: []entity.TemplateCarouselCard{
			{Components: []entity.TemplateComponent{
				{Type: "HEADER", Format: "IMAGE", Example: &entity.TemplateExample{HeaderHandle: []string{"h"}}},
			}},
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BODY sub-component")
}

func TestValidateCarousel_CardInheritsValidation(t *testing.T) {
	// Card body with {{1}} but no example — same rule as root-level BODY.
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "CAROUSEL", Cards: []entity.TemplateCarouselCard{
			{Components: []entity.TemplateComponent{
				{Type: "BODY", Text: "Hi {{1}}"},
			}},
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cards[0]")
	assert.Contains(t, err.Error(), "example")
}

func TestValidateCarousel_HappyPath(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "BODY", Text: "Check our offers"},
		{Type: "CAROUSEL", Cards: []entity.TemplateCarouselCard{
			{Components: []entity.TemplateComponent{
				{
					Type: "HEADER", Format: "IMAGE",
					Example: &entity.TemplateExample{HeaderHandle: []string{"4:AAAb..."}},
				},
				{
					Type: "BODY", Text: "Product 1",
				},
				{
					Type: "BUTTONS",
					Buttons: []entity.TemplateButton{
						{Type: "QUICK_REPLY", Text: "Add to cart"},
					},
				},
			}},
			{Components: []entity.TemplateComponent{
				{
					Type: "HEADER", Format: "IMAGE",
					Example: &entity.TemplateExample{HeaderHandle: []string{"4:AAAc..."}},
				},
				{Type: "BODY", Text: "Product 2"},
			}},
		}},
	})
	assert.NoError(t, err)
}

// -----------------------------------------------------------------------------
// LIMITED_TIME_OFFER
// -----------------------------------------------------------------------------

func TestValidateLimitedTimeOffer_RequiresPayload(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "LIMITED_TIME_OFFER"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limited_time_offer payload")
}

func TestValidateLimitedTimeOffer_ExpirationRequiresTimestamp(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "LIMITED_TIME_OFFER",
			LimitedTimeOffer: &entity.TemplateLimitedTimeOffer{
				Text:          "Offer ends soon",
				HasExpiration: true,
				// ExpirationTimeMS omitted — must fail
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiration_time_ms")
}

func TestValidateLimitedTimeOffer_NoExpiration(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "LIMITED_TIME_OFFER",
			LimitedTimeOffer: &entity.TemplateLimitedTimeOffer{
				Text:          "Limited offer",
				HasExpiration: false,
			},
		},
	})
	assert.NoError(t, err)
}

func TestValidateLimitedTimeOffer_HappyPath(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "LIMITED_TIME_OFFER",
			LimitedTimeOffer: &entity.TemplateLimitedTimeOffer{
				Text:             "Offer ends",
				HasExpiration:    true,
				ExpirationTimeMS: 1800000000000,
			},
		},
	})
	assert.NoError(t, err)
}
