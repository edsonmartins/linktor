package whatsapp_official

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buttonComponent finds the button component by sub_type in a built template.
func buttonComponent(t *testing.T, tpl *TemplateObject, subType string) TemplateComponent {
	t.Helper()
	for _, c := range tpl.Components {
		if c.Type == "button" && c.SubType == subType {
			return c
		}
	}
	t.Fatalf("no button component with sub_type=%q in template", subType)
	return TemplateComponent{}
}

func TestTemplateBuilder_PhoneNumberButton(t *testing.T) {
	tpl := NewTemplateBuilder("call_reminder", "pt_BR").
		AddBodyParameters("Felipe").
		AddPhoneNumberButton(0).
		Build()

	btn := buttonComponent(t, tpl, "phone_number")
	require.NotNil(t, btn.Index)
	assert.Equal(t, 0, *btn.Index)
	// Phone number buttons have no runtime parameters — value is baked in at
	// create time on the approved template.
	assert.Nil(t, btn.Parameters)

	// Spot-check the JSON shape — sub_type is the discriminator Meta uses.
	out, err := json.Marshal(tpl)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"sub_type":"phone_number"`)
}

func TestTemplateBuilder_CopyCodeButton(t *testing.T) {
	tpl := NewTemplateBuilder("otp_copy", "en_US").
		AddBodyParameters("123456").
		AddCopyCodeButton(0, "123456").
		Build()

	btn := buttonComponent(t, tpl, "copy_code")
	require.Len(t, btn.Parameters, 1)
	assert.Equal(t, "coupon_code", btn.Parameters[0].Type)
	assert.Equal(t, "123456", btn.Parameters[0].Text)
}

func TestTemplateBuilder_OTPButton(t *testing.T) {
	tpl := NewTemplateBuilder("otp_onetap", "en_US").
		AddOTPButton(0, "987654").
		Build()

	btn := buttonComponent(t, tpl, "url")
	require.Len(t, btn.Parameters, 1)
	assert.Equal(t, "text", btn.Parameters[0].Type)
	assert.Equal(t, "987654", btn.Parameters[0].Text)
	require.NotNil(t, btn.Index)
	assert.Equal(t, 0, *btn.Index)
}

func TestTemplateBuilder_FlowButton(t *testing.T) {
	tpl := NewTemplateBuilder("signup_flow", "pt_BR").
		AddBodyParameters("Claudia").
		AddFlowButton(0, "flow-token-xyz", map[string]interface{}{
			"customer_id": "c-1",
			"plan":        "premium",
		}).
		Build()

	btn := buttonComponent(t, tpl, "flow")
	require.Len(t, btn.Parameters, 1)
	assert.Equal(t, "action", btn.Parameters[0].Type)
	require.NotNil(t, btn.Parameters[0].Action)
	assert.Equal(t, "flow-token-xyz", btn.Parameters[0].Action["flow_token"])
	data := btn.Parameters[0].Action["flow_action_data"].(map[string]interface{})
	assert.Equal(t, "c-1", data["customer_id"])

	// JSON shape check — Meta requires the action shape verbatim
	out, err := json.Marshal(tpl)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"sub_type":"flow"`)
	assert.Contains(t, s, `"flow_token":"flow-token-xyz"`)
	assert.Contains(t, s, `"flow_action_data"`)
}

func TestTemplateBuilder_FlowButton_NoInitialData(t *testing.T) {
	tpl := NewTemplateBuilder("signup_flow", "pt_BR").
		AddFlowButton(1, "tok-2", nil).
		Build()

	btn := buttonComponent(t, tpl, "flow")
	require.NotNil(t, btn.Parameters[0].Action)
	assert.Equal(t, "tok-2", btn.Parameters[0].Action["flow_token"])
	_, hasData := btn.Parameters[0].Action["flow_action_data"]
	assert.False(t, hasData, "flow_action_data should be omitted when caller passes nil")
}

func TestTemplateBuilder_HeaderLocation(t *testing.T) {
	tpl := NewTemplateBuilder("delivery", "pt_BR").
		AddHeaderLocation(-23.5505, -46.6333, "Sede Linktor", "Av. Paulista 1000").
		Build()

	require.Len(t, tpl.Components, 1)
	h := tpl.Components[0]
	assert.Equal(t, "header", h.Type)
	require.Len(t, h.Parameters, 1)
	assert.Equal(t, "location", h.Parameters[0].Type)
	require.NotNil(t, h.Parameters[0].Location)
	loc := h.Parameters[0].Location
	assert.InDelta(t, -23.5505, loc.Latitude, 0.0001)
	assert.Equal(t, "Sede Linktor", loc.Name)
}

func TestTemplateBuilder_CombinesMultipleButtons(t *testing.T) {
	// Templates can carry up to 10 buttons in various combos. Verify the
	// builder preserves their order and indices when chained.
	tpl := NewTemplateBuilder("multi", "pt_BR").
		AddBodyParameters("Ana").
		AddQuickReplyButton(0, "YES").
		AddQuickReplyButton(1, "NO").
		AddURLButton(2, "track/abc").
		AddPhoneNumberButton(3).
		Build()

	var buttonIndices []int
	var subTypes []string
	for _, c := range tpl.Components {
		if c.Type == "button" {
			subTypes = append(subTypes, c.SubType)
			if c.Index != nil {
				buttonIndices = append(buttonIndices, *c.Index)
			}
		}
	}
	assert.Equal(t, []string{"quick_reply", "quick_reply", "url", "phone_number"}, subTypes)
	assert.Equal(t, []int{0, 1, 2, 3}, buttonIndices)
}
