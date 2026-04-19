package service

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateParameterFormat_NamedWithPositional_Rejected(t *testing.T) {
	err := validateParameterFormat(entity.TemplateParameterFormatNamed, []entity.TemplateComponent{
		{Type: "BODY", Text: "Hi {{1}}"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "positional placeholders but parameter_format=NAMED")
}

func TestValidateParameterFormat_PositionalWithNamed_Rejected(t *testing.T) {
	err := validateParameterFormat(entity.TemplateParameterFormatPositional, []entity.TemplateComponent{
		{Type: "BODY", Text: "Hi {{first_name}}"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "named placeholders but parameter_format=POSITIONAL")
}

func TestValidateParameterFormat_DefaultIsPositional(t *testing.T) {
	// An empty format must behave identically to POSITIONAL (Meta's default).
	err := validateParameterFormat("", []entity.TemplateComponent{
		{Type: "BODY", Text: "Hi {{customer_name}}"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set parameter_format=NAMED")
}

func TestValidateParameterFormat_MixedInSameComponent(t *testing.T) {
	err := validateParameterFormat(entity.TemplateParameterFormatNamed, []entity.TemplateComponent{
		{Type: "BODY", Text: "Hi {{first_name}}, code {{1}}"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mixes")
}

func TestValidateParameterFormat_BothSidesAgree(t *testing.T) {
	assert.NoError(t, validateParameterFormat(entity.TemplateParameterFormatPositional, []entity.TemplateComponent{
		{Type: "BODY", Text: "Hi {{1}}, code {{2}}"},
	}))
	assert.NoError(t, validateParameterFormat(entity.TemplateParameterFormatNamed, []entity.TemplateComponent{
		{Type: "BODY", Text: "Hi {{first_name}}, code {{verification_code}}"},
	}))
	assert.NoError(t, validateParameterFormat("", []entity.TemplateComponent{
		{Type: "BODY", Text: "No variables here"},
	}))
}

