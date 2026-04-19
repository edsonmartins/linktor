package service

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTemplateComponents_NoVariables_Passes(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "BODY", Text: "Welcome to our store!"},
		{Type: "FOOTER", Text: "Powered by Linktor"},
	})
	assert.NoError(t, err)
}

func TestValidateTemplateComponents_BodyWithVariables_RequiresExample(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "BODY", Text: "Hello {{1}}, your order {{2}} is ready."},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BODY")
	assert.Contains(t, err.Error(), "example")
}

func TestValidateTemplateComponents_BodyWithMatchingExample_Passes(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "BODY",
			Text: "Hello {{1}}, your order {{2}} is ready.",
			Example: &entity.TemplateExample{
				BodyText: [][]string{{"Ana", "ORD-42"}},
			},
		},
	})
	assert.NoError(t, err)
}

func TestValidateTemplateComponents_BodyWithShortExample_Rejected(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "BODY",
			Text: "Hello {{1}}, your order {{2}} is ready.",
			Example: &entity.TemplateExample{
				BodyText: [][]string{{"Ana"}}, // missing the second example
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 variable(s)")
}

func TestValidateTemplateComponents_HeaderTextWithVariable_RequiresExample(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "HEADER", Format: "TEXT", Text: "Hi {{1}}"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HEADER")

	err = validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "HEADER", Format: "TEXT", Text: "Hi {{1}}",
			Example: &entity.TemplateExample{HeaderText: []string{"Ana"}},
		},
	})
	assert.NoError(t, err)
}

func TestValidateTemplateComponents_HeaderMedia_RequiresHandle(t *testing.T) {
	for _, format := range []string{"IMAGE", "VIDEO", "DOCUMENT"} {
		t.Run(format, func(t *testing.T) {
			err := validateTemplateComponents([]entity.TemplateComponent{
				{Type: "HEADER", Format: format},
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "header_handle")

			err = validateTemplateComponents([]entity.TemplateComponent{
				{
					Type: "HEADER", Format: format,
					Example: &entity.TemplateExample{HeaderHandle: []string{"4:aHR0..."}},
				},
			})
			assert.NoError(t, err)
		})
	}
}

func TestValidateTemplateComponents_FooterRejectsVariables(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "FOOTER", Text: "Code: {{1}}"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FOOTER")
	assert.Contains(t, err.Error(), "variables")
}

func TestValidateTemplateComponents_ButtonTextRejectsVariables(t *testing.T) {
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "BUTTONS",
			Buttons: []entity.TemplateButton{
				{Type: "QUICK_REPLY", Text: "Confirm {{1}}"},
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "buttons[0]")
}

func TestValidateTemplateComponents_NamedPlaceholder_RequiresExample(t *testing.T) {
	// Named variables still require examples; our entity carries them in
	// the same example shape, so require at least one matching row length.
	err := validateTemplateComponents([]entity.TemplateComponent{
		{Type: "BODY", Text: "Hello {{first_name}}, order {{order_id}} ready."},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BODY")
}

func TestValidateTemplateComponents_MaxIndexDetection(t *testing.T) {
	// {{3}} without {{1}}/{{2}} — Meta would reject this regardless, but
	// our validator uses max index so the example needs 3 entries.
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "BODY",
			Text: "Only one var: {{3}}",
			Example: &entity.TemplateExample{
				BodyText: [][]string{{"a", "b"}}, // only 2 — fails
			},
		},
	})
	require.Error(t, err)

	err = validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "BODY",
			Text: "Only one var: {{3}}",
			Example: &entity.TemplateExample{
				BodyText: [][]string{{"a", "b", "c"}},
			},
		},
	})
	assert.NoError(t, err)
}
