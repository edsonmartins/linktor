package whatsapp_official

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Template Builder Tests (template.go)
// =============================================================================

func TestNewTemplateBuilder(t *testing.T) {
	b := NewTemplateBuilder("order_confirm", "pt_BR")
	tmpl := b.Build()

	assert.Equal(t, "order_confirm", tmpl.Name)
	assert.Equal(t, "pt_BR", tmpl.Language.Code)
	assert.Equal(t, "deterministic", tmpl.Language.Policy)
	assert.Empty(t, tmpl.Components)
}

func TestTemplateBuilder_AddHeaderText(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddHeaderText("Hello Header").
		Build()

	require.Len(t, tmpl.Components, 1)
	comp := tmpl.Components[0]
	assert.Equal(t, "header", comp.Type)
	require.Len(t, comp.Parameters, 1)
	assert.Equal(t, "text", comp.Parameters[0].Type)
	assert.Equal(t, "Hello Header", comp.Parameters[0].Text)
}

func TestTemplateBuilder_AddHeaderImage(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddHeaderImage("media-123").
		Build()

	require.Len(t, tmpl.Components, 1)
	comp := tmpl.Components[0]
	assert.Equal(t, "header", comp.Type)
	require.Len(t, comp.Parameters, 1)
	p := comp.Parameters[0]
	assert.Equal(t, "image", p.Type)
	require.NotNil(t, p.Image)
	assert.Equal(t, "media-123", p.Image.ID)
}

func TestTemplateBuilder_AddHeaderImageURL(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddHeaderImageURL("https://example.com/img.png").
		Build()

	require.Len(t, tmpl.Components, 1)
	p := tmpl.Components[0].Parameters[0]
	assert.Equal(t, "image", p.Type)
	require.NotNil(t, p.Image)
	assert.Equal(t, "https://example.com/img.png", p.Image.Link)
}

func TestTemplateBuilder_AddHeaderDocument(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddHeaderDocument("doc-456", "invoice.pdf").
		Build()

	require.Len(t, tmpl.Components, 1)
	p := tmpl.Components[0].Parameters[0]
	assert.Equal(t, "document", p.Type)
	require.NotNil(t, p.Document)
	assert.Equal(t, "doc-456", p.Document.ID)
	assert.Equal(t, "invoice.pdf", p.Document.Filename)
}

func TestTemplateBuilder_AddHeaderVideo(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddHeaderVideo("vid-789").
		Build()

	require.Len(t, tmpl.Components, 1)
	p := tmpl.Components[0].Parameters[0]
	assert.Equal(t, "video", p.Type)
	require.NotNil(t, p.Video)
	assert.Equal(t, "vid-789", p.Video.ID)
}

func TestTemplateBuilder_AddBodyParameters(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddBodyParameters("Alice", "Order #42", "$99.00").
		Build()

	require.Len(t, tmpl.Components, 1)
	comp := tmpl.Components[0]
	assert.Equal(t, "body", comp.Type)
	require.Len(t, comp.Parameters, 3)
	for _, p := range comp.Parameters {
		assert.Equal(t, "text", p.Type)
	}
	assert.Equal(t, "Alice", comp.Parameters[0].Text)
	assert.Equal(t, "Order #42", comp.Parameters[1].Text)
	assert.Equal(t, "$99.00", comp.Parameters[2].Text)
}

func TestTemplateBuilder_AddBodyCurrency(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddBodyCurrency("$10.50", "USD", 10500).
		Build()

	require.Len(t, tmpl.Components, 1)
	comp := tmpl.Components[0]
	assert.Equal(t, "body", comp.Type)
	require.Len(t, comp.Parameters, 1)
	p := comp.Parameters[0]
	assert.Equal(t, "currency", p.Type)
	require.NotNil(t, p.Currency)
	assert.Equal(t, "$10.50", p.Currency.FallbackValue)
	assert.Equal(t, "USD", p.Currency.Code)
	assert.Equal(t, int64(10500), p.Currency.Amount1000)
}

func TestTemplateBuilder_AddBodyDateTime(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddBodyDateTime("March 7, 2026").
		Build()

	require.Len(t, tmpl.Components, 1)
	comp := tmpl.Components[0]
	assert.Equal(t, "body", comp.Type)
	require.Len(t, comp.Parameters, 1)
	p := comp.Parameters[0]
	assert.Equal(t, "date_time", p.Type)
	require.NotNil(t, p.DateTime)
	assert.Equal(t, "March 7, 2026", p.DateTime.FallbackValue)
}

func TestTemplateBuilder_AddQuickReplyButton(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddQuickReplyButton(0, "confirm_yes").
		Build()

	require.Len(t, tmpl.Components, 1)
	comp := tmpl.Components[0]
	assert.Equal(t, "button", comp.Type)
	assert.Equal(t, "quick_reply", comp.SubType)
	require.NotNil(t, comp.Index)
	assert.Equal(t, 0, *comp.Index)
	require.Len(t, comp.Parameters, 1)
	assert.Equal(t, "payload", comp.Parameters[0].Type)
	assert.Equal(t, "confirm_yes", comp.Parameters[0].Text)
}

func TestTemplateBuilder_AddURLButton(t *testing.T) {
	tmpl := NewTemplateBuilder("t", "en").
		AddURLButton(1, "/tracking/ABC123").
		Build()

	require.Len(t, tmpl.Components, 1)
	comp := tmpl.Components[0]
	assert.Equal(t, "button", comp.Type)
	assert.Equal(t, "url", comp.SubType)
	require.NotNil(t, comp.Index)
	assert.Equal(t, 1, *comp.Index)
	require.Len(t, comp.Parameters, 1)
	assert.Equal(t, "text", comp.Parameters[0].Type)
	assert.Equal(t, "/tracking/ABC123", comp.Parameters[0].Text)
}

func TestTemplateBuilder_Build(t *testing.T) {
	b := NewTemplateBuilder("my_template", "en")
	result := b.Build()
	assert.IsType(t, &TemplateObject{}, result)
	assert.Equal(t, "my_template", result.Name)
}

func TestTemplateBuilder_ToJSON(t *testing.T) {
	b := NewTemplateBuilder("greet", "en").
		AddBodyParameters("World")

	jsonStr, err := b.ToJSON()
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "greet", parsed["name"])
}

func TestTemplateBuilder_ChainedBuild(t *testing.T) {
	tmpl := NewTemplateBuilder("full_template", "pt_BR").
		AddHeaderImageURL("https://example.com/header.jpg").
		AddBodyParameters("param1", "param2").
		AddQuickReplyButton(0, "yes_payload").
		AddURLButton(1, "/details/123").
		Build()

	require.Len(t, tmpl.Components, 4)
	assert.Equal(t, "header", tmpl.Components[0].Type)
	assert.Equal(t, "body", tmpl.Components[1].Type)
	assert.Equal(t, "button", tmpl.Components[2].Type)
	assert.Equal(t, "quick_reply", tmpl.Components[2].SubType)
	assert.Equal(t, "button", tmpl.Components[3].Type)
	assert.Equal(t, "url", tmpl.Components[3].SubType)
}

// Tests for CreateOTPTemplate / CreateShippingUpdateTemplate were removed
// alongside those helpers (see P3 cleanup in template.go). The equivalent
// build-from-scratch flow is exercised by TestTemplateBuilder_* cases in
// template_test.go, and the stored-template path by TestBuildSendPayload_*.

func TestTemplateFromJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		jsonStr := `{"name":"test","language":{"policy":"deterministic","code":"en"},"components":[]}`
		tmpl, err := TemplateFromJSON(jsonStr)
		require.NoError(t, err)
		assert.Equal(t, "test", tmpl.Name)
		assert.Equal(t, "en", tmpl.Language.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := TemplateFromJSON("{invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse template JSON")
	})
}

// =============================================================================
// Interactive Builder Tests (interactive.go)
// =============================================================================

func TestNewButtonMessageBuilder(t *testing.T) {
	b := NewButtonMessageBuilder("Choose an option")
	obj := b.Build()

	assert.Equal(t, "button", obj.Type)
	assert.Equal(t, "Choose an option", obj.Body.Text)
	require.NotNil(t, obj.Action)
	assert.Empty(t, obj.Action.Buttons)
}

func TestNewListMessageBuilder(t *testing.T) {
	b := NewListMessageBuilder("Pick one", "View Options")
	obj := b.Build()

	assert.Equal(t, "list", obj.Type)
	assert.Equal(t, "Pick one", obj.Body.Text)
	assert.Equal(t, "View Options", obj.Action.Button)
	assert.Empty(t, obj.Action.Sections)
}

func TestInteractiveBuilder_SetHeader(t *testing.T) {
	obj := NewButtonMessageBuilder("body").
		SetHeader("My Header").
		Build()

	require.NotNil(t, obj.Header)
	assert.Equal(t, "text", obj.Header.Type)
	assert.Equal(t, "My Header", obj.Header.Text)
}

func TestInteractiveBuilder_SetImageHeader(t *testing.T) {
	obj := NewButtonMessageBuilder("body").
		SetImageHeader("img-001").
		Build()

	require.NotNil(t, obj.Header)
	assert.Equal(t, "image", obj.Header.Type)
	require.NotNil(t, obj.Header.Image)
	assert.Equal(t, "img-001", obj.Header.Image.ID)
}

func TestInteractiveBuilder_SetImageHeaderURL(t *testing.T) {
	obj := NewButtonMessageBuilder("body").
		SetImageHeaderURL("https://example.com/photo.jpg").
		Build()

	require.NotNil(t, obj.Header)
	assert.Equal(t, "image", obj.Header.Type)
	require.NotNil(t, obj.Header.Image)
	assert.Equal(t, "https://example.com/photo.jpg", obj.Header.Image.Link)
}

func TestInteractiveBuilder_SetVideoHeader(t *testing.T) {
	obj := NewButtonMessageBuilder("body").
		SetVideoHeader("vid-001").
		Build()

	require.NotNil(t, obj.Header)
	assert.Equal(t, "video", obj.Header.Type)
	require.NotNil(t, obj.Header.Video)
	assert.Equal(t, "vid-001", obj.Header.Video.ID)
}

func TestInteractiveBuilder_SetDocumentHeader(t *testing.T) {
	obj := NewButtonMessageBuilder("body").
		SetDocumentHeader("doc-001", "report.pdf").
		Build()

	require.NotNil(t, obj.Header)
	assert.Equal(t, "document", obj.Header.Type)
	require.NotNil(t, obj.Header.Document)
	assert.Equal(t, "doc-001", obj.Header.Document.ID)
	assert.Equal(t, "report.pdf", obj.Header.Document.Filename)
}

func TestInteractiveBuilder_SetFooter(t *testing.T) {
	obj := NewButtonMessageBuilder("body").
		SetFooter("Powered by Linktor").
		Build()

	require.NotNil(t, obj.Footer)
	assert.Equal(t, "Powered by Linktor", obj.Footer.Text)
}

func TestInteractiveBuilder_AddButton(t *testing.T) {
	t.Run("adds up to 3 buttons, 4th ignored", func(t *testing.T) {
		b := NewButtonMessageBuilder("body")
		b.AddButton("b1", "Button 1")
		b.AddButton("b2", "Button 2")
		b.AddButton("b3", "Button 3")
		b.AddButton("b4", "Button 4") // should be ignored
		obj := b.Build()

		require.Len(t, obj.Action.Buttons, 3)
		for _, btn := range obj.Action.Buttons {
			assert.Equal(t, "reply", btn.Type)
			require.NotNil(t, btn.Reply)
		}
		assert.Equal(t, "b1", obj.Action.Buttons[0].Reply.ID)
		assert.Equal(t, "Button 1", obj.Action.Buttons[0].Reply.Title)
		assert.Equal(t, "b3", obj.Action.Buttons[2].Reply.ID)
	})
}

func TestInteractiveBuilder_AddButton_TruncatesTitle(t *testing.T) {
	longTitle := "This title is way too long for WhatsApp" // 40 chars
	obj := NewButtonMessageBuilder("body").
		AddButton("id1", longTitle).
		Build()

	require.Len(t, obj.Action.Buttons, 1)
	assert.Len(t, obj.Action.Buttons[0].Reply.Title, 20)
	assert.Equal(t, longTitle[:20], obj.Action.Buttons[0].Reply.Title)
}

func TestInteractiveBuilder_AddSection(t *testing.T) {
	rows := make([]ListRow, 12)
	for i := range rows {
		rows[i] = ListRow{ID: "r", Title: "row"}
	}

	obj := NewListMessageBuilder("body", "Menu").
		AddSection("Section A", rows).
		Build()

	require.Len(t, obj.Action.Sections, 1)
	sec := obj.Action.Sections[0]
	assert.Equal(t, "Section A", sec.Title)
	assert.Len(t, sec.Rows, 10, "rows should be capped at 10")
}

func TestInteractiveBuilder_AddSection_TruncatesTitle(t *testing.T) {
	longTitle := "This section title is way too long" // >24 chars
	obj := NewListMessageBuilder("body", "Menu").
		AddSection(longTitle, []ListRow{{ID: "1", Title: "row"}}).
		Build()

	require.Len(t, obj.Action.Sections, 1)
	assert.Len(t, obj.Action.Sections[0].Title, 24)
	assert.Equal(t, longTitle[:24], obj.Action.Sections[0].Title)
}

func TestInteractiveBuilder_AddListRow(t *testing.T) {
	longTitle := strings.Repeat("T", 30)       // >24
	longDesc := strings.Repeat("D", 80)        // >72
	longID := strings.Repeat("I", 210)         // >200

	obj := NewListMessageBuilder("body", "Menu").
		AddListRow(longID, longTitle, longDesc).
		Build()

	require.Len(t, obj.Action.Sections, 1)
	require.Len(t, obj.Action.Sections[0].Rows, 1)

	row := obj.Action.Sections[0].Rows[0]
	assert.Len(t, row.Title, 24)
	assert.Len(t, row.Description, 72)
	assert.Len(t, row.ID, 200)
}

func TestInteractiveBuilder_Build(t *testing.T) {
	b := NewButtonMessageBuilder("text")
	result := b.Build()
	assert.IsType(t, &InteractiveObject{}, result)
	assert.Equal(t, "button", result.Type)
}

func TestCreateQuickReplyButtons(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		btns := CreateQuickReplyButtons("Yes", "No")
		require.Len(t, btns, 2)
		assert.Equal(t, "option_1", btns[0].ID)
		assert.Equal(t, "Yes", btns[0].Title)
		assert.Equal(t, "option_2", btns[1].ID)
		assert.Equal(t, "No", btns[1].Title)
	})

	t.Run("max 3", func(t *testing.T) {
		btns := CreateQuickReplyButtons("A", "B", "C", "D", "E")
		assert.Len(t, btns, 3)
	})
}

func TestShouldUseList(t *testing.T) {
	tests := []struct {
		count    int
		expected bool
	}{
		{1, false},
		{2, false},
		{3, false},
		{4, true},
		{10, true},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, ShouldUseList(tc.count), "count=%d", tc.count)
	}
}

func TestCreateMenu(t *testing.T) {
	makeOpts := func(n int) []struct{ ID, Title, Description string } {
		opts := make([]struct{ ID, Title, Description string }, n)
		for i := range opts {
			opts[i] = struct{ ID, Title, Description string }{
				ID: "id", Title: "title", Description: "desc",
			}
		}
		return opts
	}

	t.Run("3 or fewer options uses button type", func(t *testing.T) {
		obj := CreateMenu("Pick", "Menu", makeOpts(3))
		assert.Equal(t, "button", obj.Type)
		assert.Len(t, obj.Action.Buttons, 3)
	})

	t.Run("more than 3 options uses list type", func(t *testing.T) {
		obj := CreateMenu("Pick", "Menu", makeOpts(5))
		assert.Equal(t, "list", obj.Type)
		require.Len(t, obj.Action.Sections, 1)
		assert.Len(t, obj.Action.Sections[0].Rows, 5)
	})
}

// =============================================================================
// Carousel Builder Tests (carousel.go)
// =============================================================================

func TestNewCarouselBuilder(t *testing.T) {
	b := NewCarouselBuilder("promo_carousel", "en", CarouselHeaderImage)
	assert.Equal(t, "promo_carousel", b.name)
	assert.Equal(t, "en", b.language)
	assert.Equal(t, CarouselHeaderImage, b.headerType)
	assert.Empty(t, b.cards)
}

func TestCarouselBuilder_AddCard(t *testing.T) {
	b := NewCarouselBuilder("c", "en", CarouselHeaderImage)
	b.AddCard(CarouselCardInput{
		HeaderMediaURL: "https://example.com/img.jpg",
		BodyParams:     []string{"Product A", "$10"},
		Buttons: []CarouselButtonInput{
			{Type: "quick_reply", Payload: "buy_a"},
		},
	})

	require.Len(t, b.cards, 1)
	card := b.cards[0]
	assert.Equal(t, 0, card.CardIndex)
	// header + body + 1 button = 3 components
	require.Len(t, card.Components, 3)
	assert.Equal(t, "header", card.Components[0].Type)
	assert.Equal(t, "body", card.Components[1].Type)
	assert.Equal(t, "button", card.Components[2].Type)
}

func TestCarouselBuilder_AddCard_MaxCards(t *testing.T) {
	b := NewCarouselBuilder("c", "en", CarouselHeaderImage)
	for i := 0; i < 11; i++ {
		b.AddCard(CarouselCardInput{HeaderMediaURL: "https://example.com/img.jpg"})
	}
	assert.Len(t, b.cards, 10, "11th card should be ignored")
}

func TestCarouselBuilder_AddImageCard(t *testing.T) {
	b := NewCarouselBuilder("c", "en", CarouselHeaderImage)
	b.AddImageCard("https://example.com/img.jpg", []string{"p1"}, []CarouselButtonInput{
		{Type: "url", Payload: "/details"},
	})

	require.Len(t, b.cards, 1)
	card := b.cards[0]
	// header + body + button
	require.Len(t, card.Components, 3)

	// Verify header has image link
	headerParam := card.Components[0].Parameters[0]
	assert.Equal(t, "image", headerParam.Type)
	require.NotNil(t, headerParam.Image)
	assert.Equal(t, "https://example.com/img.jpg", headerParam.Image.Link)
}

func TestCarouselBuilder_BuildRaw(t *testing.T) {
	b := NewCarouselBuilder("carousel_tmpl", "en", CarouselHeaderImage)
	b.AddImageCard("https://example.com/1.jpg", []string{"A"}, nil)
	b.AddImageCard("https://example.com/2.jpg", []string{"B"}, nil)

	raw, err := b.BuildRaw()
	require.NoError(t, err)
	require.NotNil(t, raw)

	// Verify can marshal to JSON
	data, err := json.Marshal(raw)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestCarouselBuilder_BuildRaw_TooFewCards(t *testing.T) {
	b := NewCarouselBuilder("c", "en", CarouselHeaderImage)
	b.AddImageCard("https://example.com/1.jpg", []string{"A"}, nil)

	_, err := b.BuildRaw()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 cards")
}

func TestCarouselBuilder_BuildRaw_Structure(t *testing.T) {
	b := NewCarouselBuilder("my_carousel", "pt_BR", CarouselHeaderImage)
	b.SetBodyParams("intro text")
	b.AddImageCard("https://example.com/1.jpg", []string{"Card1"}, nil)
	b.AddImageCard("https://example.com/2.jpg", []string{"Card2"}, nil)

	raw, err := b.BuildRaw()
	require.NoError(t, err)

	assert.Equal(t, "my_carousel", raw["name"])

	lang, ok := raw["language"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "deterministic", lang["policy"])
	assert.Equal(t, "pt_BR", lang["code"])

	components, ok := raw["components"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, components, 2) // body + carousel

	assert.Equal(t, "body", components[0]["type"])
	assert.Equal(t, "carousel", components[1]["type"])

	cards, ok := components[1]["cards"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, cards, 2)
}

func TestCreateProductCarousel(t *testing.T) {
	t.Run("valid 2-10 products", func(t *testing.T) {
		products := []ProductCarouselItem{
			{ImageURL: "https://example.com/1.jpg", Name: "P1", Price: "$10", Description: "Desc1", ViewDetailsURL: "/p1"},
			{ImageURL: "https://example.com/2.jpg", Name: "P2", Price: "$20", Description: "Desc2", AddToCartPayload: "add_p2"},
		}
		builder, err := CreateProductCarousel("products", "en", products)
		require.NoError(t, err)
		assert.Len(t, builder.cards, 2)

		// Ensure it builds successfully
		raw, err := builder.BuildRaw()
		require.NoError(t, err)
		assert.NotNil(t, raw)
	})

	t.Run("1 product returns error", func(t *testing.T) {
		_, err := CreateProductCarousel("p", "en", []ProductCarouselItem{
			{ImageURL: "https://example.com/1.jpg", Name: "P1", Price: "$10", Description: "d"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "2-10 products")
	})

	t.Run("11 products returns error", func(t *testing.T) {
		products := make([]ProductCarouselItem, 11)
		for i := range products {
			products[i] = ProductCarouselItem{ImageURL: "https://example.com/x.jpg", Name: "P", Price: "$1", Description: "d"}
		}
		_, err := CreateProductCarousel("p", "en", products)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "2-10 products")
	})
}

// =============================================================================
// Auth Template Builder Tests (auth_templates.go)
// =============================================================================

func TestNewAuthTemplateBuilder(t *testing.T) {
	b := NewAuthTemplateBuilder("auth_otp", "en")

	assert.Equal(t, "auth_otp", b.config.TemplateName)
	assert.Equal(t, "en", b.config.LanguageCode)
	assert.Equal(t, AuthTypeCopyCode, b.config.AuthType)
	assert.Equal(t, 10, b.config.ExpirationMinutes)
}

func TestAuthTemplateBuilder_Build(t *testing.T) {
	tmpl, err := NewAuthTemplateBuilder("auth_otp", "en").
		SetOTP("654321").
		Build()

	require.NoError(t, err)
	assert.Equal(t, "auth_otp", tmpl.Name)
	assert.Equal(t, "en", tmpl.Language.Code)
	assert.Equal(t, "deterministic", tmpl.Language.Policy)

	require.Len(t, tmpl.Components, 2)

	// Body component
	body := tmpl.Components[0]
	assert.Equal(t, "body", body.Type)
	require.Len(t, body.Parameters, 1)
	assert.Equal(t, "text", body.Parameters[0].Type)
	assert.Equal(t, "654321", body.Parameters[0].Text)

	// Button component
	btn := tmpl.Components[1]
	assert.Equal(t, "button", btn.Type)
	assert.Equal(t, "copy_code", btn.SubType)
	require.NotNil(t, btn.Index)
	assert.Equal(t, 0, *btn.Index)
	require.Len(t, btn.Parameters, 1)
	assert.Equal(t, "654321", btn.Parameters[0].Text)
}

func TestAuthTemplateBuilder_Build_NoOTP(t *testing.T) {
	_, err := NewAuthTemplateBuilder("auth_otp", "en").Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OTP is required")
}

func TestAuthTemplateBuilder_Build_OneTap_NoPackage(t *testing.T) {
	_, err := NewAuthTemplateBuilder("auth_otp", "en").
		SetOTP("123456").
		SetAuthType(AuthTypeOneTap).
		Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "package_name")
}

func TestAuthTemplateBuilder_BuildRaw(t *testing.T) {
	raw, err := NewAuthTemplateBuilder("auth_otp", "en").
		SetOTP("999999").
		BuildRaw()

	require.NoError(t, err)
	assert.Equal(t, "auth_otp", raw["name"])

	lang, ok := raw["language"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "deterministic", lang["policy"])
	assert.Equal(t, "en", lang["code"])

	components, ok := raw["components"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, components, 2)
	assert.Equal(t, "body", components[0]["type"])
	assert.Equal(t, "button", components[1]["type"])
	assert.Equal(t, "copy_code", components[1]["sub_type"])
}

func TestOTPSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"expired", time.Now().Add(-1 * time.Minute), true},
		{"not expired", time.Now().Add(5 * time.Minute), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &OTPSession{ExpiresAt: tc.expiresAt}
			assert.Equal(t, tc.expected, s.IsExpired())
		})
	}
}

func TestOTPSession_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		session  OTPSession
		expected bool
	}{
		{
			name: "valid session",
			session: OTPSession{
				ExpiresAt: time.Now().Add(5 * time.Minute),
				Verified:  false,
				Attempts:  0,
			},
			expected: true,
		},
		{
			name: "expired session",
			session: OTPSession{
				ExpiresAt: time.Now().Add(-1 * time.Minute),
				Verified:  false,
				Attempts:  0,
			},
			expected: false,
		},
		{
			name: "already verified",
			session: OTPSession{
				ExpiresAt: time.Now().Add(5 * time.Minute),
				Verified:  true,
				Attempts:  1,
			},
			expected: false,
		},
		{
			name: "too many attempts",
			session: OTPSession{
				ExpiresAt: time.Now().Add(5 * time.Minute),
				Verified:  false,
				Attempts:  3,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.session.IsValid())
		})
	}
}

func TestOTPSession_Verify(t *testing.T) {
	t.Run("correct OTP", func(t *testing.T) {
		s := &OTPSession{
			OTP:       "123456",
			ExpiresAt: time.Now().Add(5 * time.Minute),
			Verified:  false,
			Attempts:  0,
		}
		result := s.Verify("123456")
		assert.True(t, result)
		assert.True(t, s.Verified)
		assert.Equal(t, 1, s.Attempts)
	})

	t.Run("wrong OTP increments attempts", func(t *testing.T) {
		s := &OTPSession{
			OTP:       "123456",
			ExpiresAt: time.Now().Add(5 * time.Minute),
			Verified:  false,
			Attempts:  0,
		}
		result := s.Verify("000000")
		assert.False(t, result)
		assert.False(t, s.Verified)
		assert.Equal(t, 1, s.Attempts)
	})

	t.Run("max attempts reached makes session invalid", func(t *testing.T) {
		s := &OTPSession{
			OTP:       "123456",
			ExpiresAt: time.Now().Add(5 * time.Minute),
			Verified:  false,
			Attempts:  0,
		}
		s.Verify("wrong1")
		s.Verify("wrong2")
		s.Verify("wrong3")
		// Session should now be invalid (3 attempts = MaxOTPAttempts)
		assert.False(t, s.IsValid())
		// Further verify should return false
		result := s.Verify("123456")
		assert.False(t, result)
	})
}

// =============================================================================
// LTO Template Builder Tests (lto_templates.go)
// =============================================================================

func TestNewLTOTemplateBuilder(t *testing.T) {
	b := NewLTOTemplateBuilder("flash_sale", "pt_BR")

	assert.Equal(t, "flash_sale", b.config.TemplateName)
	assert.Equal(t, "pt_BR", b.config.LanguageCode)
	assert.Equal(t, LTOTypeCountdown, b.config.OfferType)
	assert.Empty(t, b.config.BodyParams)
}

func TestLTOTemplateBuilder_BuildRaw_Countdown(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour)
	raw, err := NewLTOTemplateBuilder("countdown_tmpl", "en").
		SetExpiration(expiration).
		SetBodyParams("50% OFF").
		BuildRaw()

	require.NoError(t, err)
	assert.Equal(t, "countdown_tmpl", raw["name"])

	components, ok := raw["components"].([]map[string]interface{})
	require.True(t, ok)

	// Find the limited_time_offer component
	found := false
	for _, comp := range components {
		if comp["type"] == "limited_time_offer" {
			found = true
			params, ok := comp["parameters"].([]map[string]interface{})
			require.True(t, ok)
			require.Len(t, params, 1)
			assert.Equal(t, "limited_time_offer", params[0]["type"])
			lto, ok := params[0]["limited_time_offer"].(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, lto, "expiration_time_ms")
		}
	}
	assert.True(t, found, "should have a limited_time_offer component")
}

func TestLTOTemplateBuilder_BuildRaw_Coupon(t *testing.T) {
	raw, err := NewLTOTemplateBuilder("coupon_tmpl", "en").
		SetCouponCode("SAVE20").
		BuildRaw()

	require.NoError(t, err)

	components, ok := raw["components"].([]map[string]interface{})
	require.True(t, ok)

	// Find the copy_code button component
	found := false
	for _, comp := range components {
		if comp["type"] == "button" && comp["sub_type"] == "copy_code" {
			found = true
			params, ok := comp["parameters"].([]map[string]interface{})
			require.True(t, ok)
			require.Len(t, params, 1)
			assert.Equal(t, "coupon_code", params[0]["type"])
			assert.Equal(t, "SAVE20", params[0]["coupon_code"])
		}
	}
	assert.True(t, found, "should have a copy_code button component")
}

func TestLTOTemplateBuilder_BuildRaw_WithCTA(t *testing.T) {
	raw, err := NewLTOTemplateBuilder("cta_tmpl", "en").
		SetCTAButton("https://shop.example.com/sale").
		BuildRaw()

	require.NoError(t, err)

	components, ok := raw["components"].([]map[string]interface{})
	require.True(t, ok)

	found := false
	for _, comp := range components {
		if comp["type"] == "button" && comp["sub_type"] == "url" {
			found = true
			params, ok := comp["parameters"].([]map[string]interface{})
			require.True(t, ok)
			require.Len(t, params, 1)
			assert.Equal(t, "text", params[0]["type"])
			assert.Equal(t, "https://shop.example.com/sale", params[0]["text"])
		}
	}
	assert.True(t, found, "should have a url button component")
}

func TestLTOTemplateBuilder_BuildRaw_HeaderImage(t *testing.T) {
	raw, err := NewLTOTemplateBuilder("img_tmpl", "en").
		SetHeaderImage("https://example.com/banner.jpg").
		BuildRaw()

	require.NoError(t, err)

	components, ok := raw["components"].([]map[string]interface{})
	require.True(t, ok)

	found := false
	for _, comp := range components {
		if comp["type"] == "header" {
			found = true
			params, ok := comp["parameters"].([]map[string]interface{})
			require.True(t, ok)
			require.Len(t, params, 1)
			assert.Equal(t, "image", params[0]["type"])
			img, ok := params[0]["image"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "https://example.com/banner.jpg", img["link"])
		}
	}
	assert.True(t, found, "should have a header component with image")
}

func TestLTOTemplateBuilder_BuildRaw_HeaderVideo(t *testing.T) {
	raw, err := NewLTOTemplateBuilder("vid_tmpl", "en").
		SetHeaderVideo("https://example.com/promo.mp4").
		BuildRaw()

	require.NoError(t, err)

	components, ok := raw["components"].([]map[string]interface{})
	require.True(t, ok)

	found := false
	for _, comp := range components {
		if comp["type"] == "header" {
			found = true
			params, ok := comp["parameters"].([]map[string]interface{})
			require.True(t, ok)
			require.Len(t, params, 1)
			assert.Equal(t, "video", params[0]["type"])
			vid, ok := params[0]["video"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "https://example.com/promo.mp4", vid["link"])
		}
	}
	assert.True(t, found, "should have a header component with video")
}

func TestScheduledPromotion_IsActive(t *testing.T) {
	tests := []struct {
		name      string
		start     time.Time
		end       time.Time
		expected  bool
	}{
		{
			name:     "active - within range",
			start:    time.Now().Add(-1 * time.Hour),
			end:      time.Now().Add(1 * time.Hour),
			expected: true,
		},
		{
			name:     "not active - before start",
			start:    time.Now().Add(1 * time.Hour),
			end:      time.Now().Add(2 * time.Hour),
			expected: false,
		},
		{
			name:     "not active - after end",
			start:    time.Now().Add(-2 * time.Hour),
			end:      time.Now().Add(-1 * time.Hour),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &ScheduledPromotion{StartTime: tc.start, EndTime: tc.end}
			assert.Equal(t, tc.expected, p.IsActive())
		})
	}
}

func TestScheduledPromotion_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		end      time.Time
		expected bool
	}{
		{"expired", time.Now().Add(-1 * time.Hour), true},
		{"not expired", time.Now().Add(1 * time.Hour), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &ScheduledPromotion{EndTime: tc.end}
			assert.Equal(t, tc.expected, p.IsExpired())
		})
	}
}

func TestScheduledPromotion_FormatCountdown(t *testing.T) {
	tests := []struct {
		name     string
		end      time.Time
		contains string
	}{
		{
			name:     "expired",
			end:      time.Now().Add(-1 * time.Hour),
			contains: "Expired",
		},
		{
			name:     "hours and minutes",
			end:      time.Now().Add(3*time.Hour + 30*time.Minute),
			contains: "hours",
		},
		{
			name:     "days and hours",
			end:      time.Now().Add(50 * time.Hour), // >24h
			contains: "days",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &ScheduledPromotion{EndTime: tc.end}
			result := p.FormatCountdown()
			assert.Contains(t, result, tc.contains)
		})
	}
}
