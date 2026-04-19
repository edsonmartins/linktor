package whatsapp_official

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSendPayload_NilTemplateErrors(t *testing.T) {
	_, err := BuildSendPayload(nil, SendValues{})
	require.Error(t, err)
}

func TestBuildSendPayload_PreservesNameAndLanguage(t *testing.T) {
	tpl := &entity.Template{Name: "welcome", Language: "pt_BR"}
	payload, err := BuildSendPayload(tpl, SendValues{})
	require.NoError(t, err)
	assert.Equal(t, "welcome", payload.Name)
	assert.Equal(t, "pt_BR", payload.Language.Code)
	assert.Equal(t, "deterministic", payload.Language.Policy)
}

func TestBuildSendPayload_BodyPositional(t *testing.T) {
	tpl := &entity.Template{
		Name:     "order",
		Language: "pt_BR",
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "Hi {{1}}, order {{2}}"},
		},
	}
	payload, err := BuildSendPayload(tpl, SendValues{
		BodyParams: []string{"Ana", "ORD-42"},
	})
	require.NoError(t, err)
	require.Len(t, payload.Components, 1)

	body := payload.Components[0]
	assert.Equal(t, "body", body.Type)
	require.Len(t, body.Parameters, 2)
	assert.Equal(t, "text", body.Parameters[0].Type)
	assert.Equal(t, "Ana", body.Parameters[0].Text)
	assert.Equal(t, "ORD-42", body.Parameters[1].Text)
}

func TestBuildSendPayload_BodyNamedTakesPrecedence(t *testing.T) {
	tpl := &entity.Template{
		Name: "order", Language: "pt_BR",
		Components: []entity.TemplateComponent{{Type: "BODY", Text: "Hi {{name}}"}},
	}
	payload, err := BuildSendPayload(tpl, SendValues{
		BodyParams: []string{"ignored"},
		NamedBody:  map[string]string{"name": "Ana"},
	})
	require.NoError(t, err)
	require.Len(t, payload.Components, 1)
	require.Len(t, payload.Components[0].Parameters, 1)
	assert.Equal(t, "Ana", payload.Components[0].Parameters[0].Text)
}

func TestBuildSendPayload_HeaderByFormat(t *testing.T) {
	cases := []struct {
		name     string
		comp     entity.TemplateComponent
		values   SendValues
		wantType string
		assertFn func(*testing.T, TemplateParameter)
	}{
		{
			name:     "text",
			comp:     entity.TemplateComponent{Type: "HEADER", Format: "TEXT"},
			values:   SendValues{HeaderText: "Olá"},
			wantType: "text",
			assertFn: func(t *testing.T, p TemplateParameter) { assert.Equal(t, "Olá", p.Text) },
		},
		{
			name:     "image id",
			comp:     entity.TemplateComponent{Type: "HEADER", Format: "IMAGE"},
			values:   SendValues{HeaderImageID: "media-1"},
			wantType: "image",
			assertFn: func(t *testing.T, p TemplateParameter) { assert.Equal(t, "media-1", p.Image.ID) },
		},
		{
			name:     "image url",
			comp:     entity.TemplateComponent{Type: "HEADER", Format: "IMAGE"},
			values:   SendValues{HeaderImageURL: "https://cdn/x.jpg"},
			wantType: "image",
			assertFn: func(t *testing.T, p TemplateParameter) { assert.Equal(t, "https://cdn/x.jpg", p.Image.Link) },
		},
		{
			name:     "video id",
			comp:     entity.TemplateComponent{Type: "HEADER", Format: "VIDEO"},
			values:   SendValues{HeaderVideoID: "vid-1"},
			wantType: "video",
			assertFn: func(t *testing.T, p TemplateParameter) { assert.Equal(t, "vid-1", p.Video.ID) },
		},
		{
			name:     "document with filename",
			comp:     entity.TemplateComponent{Type: "HEADER", Format: "DOCUMENT"},
			values:   SendValues{HeaderDocumentID: "doc-1", HeaderDocumentFilename: "invoice.pdf"},
			wantType: "document",
			assertFn: func(t *testing.T, p TemplateParameter) {
				assert.Equal(t, "doc-1", p.Document.ID)
				assert.Equal(t, "invoice.pdf", p.Document.Filename)
			},
		},
		{
			name:     "location",
			comp:     entity.TemplateComponent{Type: "HEADER", Format: "LOCATION"},
			values:   SendValues{HeaderLocation: &LocationObject{Latitude: 1, Longitude: 2}},
			wantType: "location",
			assertFn: func(t *testing.T, p TemplateParameter) { assert.InDelta(t, 1.0, p.Location.Latitude, 0.001) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tpl := &entity.Template{Name: "t", Language: "pt_BR", Components: []entity.TemplateComponent{tc.comp}}
			payload, err := BuildSendPayload(tpl, tc.values)
			require.NoError(t, err)
			require.Len(t, payload.Components, 1)
			require.Equal(t, "header", payload.Components[0].Type)
			require.Len(t, payload.Components[0].Parameters, 1)
			p := payload.Components[0].Parameters[0]
			assert.Equal(t, tc.wantType, p.Type)
			tc.assertFn(t, p)
		})
	}
}

func TestBuildSendPayload_HeaderMissingValueSkipped(t *testing.T) {
	// No matching value for the declared header format → component is
	// silently dropped rather than emitting an invalid payload.
	tpl := &entity.Template{
		Name: "t", Language: "pt_BR",
		Components: []entity.TemplateComponent{{Type: "HEADER", Format: "IMAGE"}},
	}
	payload, err := BuildSendPayload(tpl, SendValues{})
	require.NoError(t, err)
	assert.Empty(t, payload.Components)
}

func TestBuildSendPayload_Buttons(t *testing.T) {
	tpl := &entity.Template{
		Name: "multi", Language: "pt_BR",
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "Check out {{1}}"},
			{
				Type: "BUTTONS",
				Buttons: []entity.TemplateButton{
					{Type: "QUICK_REPLY", Text: "Yes"},
					{Type: "URL", Text: "Open"},
					{Type: "COPY_CODE", Text: "Copy"},
					{Type: "FLOW", Text: "Start"},
					{Type: "PHONE_NUMBER", Text: "Call"},
				},
			},
		},
	}

	payload, err := BuildSendPayload(tpl, SendValues{
		BodyParams: []string{"product"},
		ButtonValues: map[int]string{
			0: "CONFIRM",
			1: "track/abc",
			2: "PROMO123",
			3: "flow-token-xyz",
		},
		FlowExtras: map[int]map[string]interface{}{
			3: {"screen": "WELCOME"},
		},
	})
	require.NoError(t, err)

	// Body + 5 buttons (phone_number included, no values needed)
	require.Len(t, payload.Components, 6)

	// Spot-check sub_types in order
	assert.Equal(t, "body", payload.Components[0].Type)
	assert.Equal(t, "quick_reply", payload.Components[1].SubType)
	assert.Equal(t, "url", payload.Components[2].SubType)
	assert.Equal(t, "copy_code", payload.Components[3].SubType)
	assert.Equal(t, "flow", payload.Components[4].SubType)
	assert.Equal(t, "phone_number", payload.Components[5].SubType)

	// Index numbering preserved
	require.NotNil(t, payload.Components[1].Index)
	assert.Equal(t, 0, *payload.Components[1].Index)
	require.NotNil(t, payload.Components[5].Index)
	assert.Equal(t, 4, *payload.Components[5].Index)

	// Flow extras land inside the action payload
	flowParams := payload.Components[4].Parameters
	require.Len(t, flowParams, 1)
	assert.Equal(t, "flow-token-xyz", flowParams[0].Action["flow_token"])
	screens := flowParams[0].Action["flow_action_data"].(map[string]interface{})
	assert.Equal(t, "WELCOME", screens["screen"])
}

func TestBuildSendPayload_ButtonWithoutValueSkipped(t *testing.T) {
	tpl := &entity.Template{
		Name: "t", Language: "pt_BR",
		Components: []entity.TemplateComponent{
			{Type: "BUTTONS", Buttons: []entity.TemplateButton{
				{Type: "URL", Text: "Go"},
			}},
		},
	}
	// No ButtonValues → URL button skipped; payload has no components.
	payload, err := BuildSendPayload(tpl, SendValues{})
	require.NoError(t, err)
	assert.Empty(t, payload.Components)
}

func TestBuildSendPayload_StaticComponentsDroppedFromSend(t *testing.T) {
	// Footer, Carousel, and LTO are static at send time — their content is
	// defined on the approved template and Meta doesn't want us echoing it.
	tpl := &entity.Template{
		Name: "t", Language: "pt_BR",
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "Hi {{1}}"},
			{Type: "FOOTER", Text: "Static footer"},
			{Type: "CAROUSEL", Cards: []entity.TemplateCarouselCard{
				{Components: []entity.TemplateComponent{{Type: "BODY", Text: "card"}}},
			}},
			{Type: "LIMITED_TIME_OFFER"},
		},
	}
	payload, err := BuildSendPayload(tpl, SendValues{
		BodyParams: []string{"Ana"},
	})
	require.NoError(t, err)
	require.Len(t, payload.Components, 1)
	assert.Equal(t, "body", payload.Components[0].Type)
}
