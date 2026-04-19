package service

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOTPButton_CopyCodePermissive(t *testing.T) {
	err := validateOTPButton(0, 0, entity.TemplateButton{
		Type: "OTP", Text: "Copy", OTPType: "COPY_CODE",
	})
	assert.NoError(t, err)
}

func TestValidateOTPButton_OneTapRequiresSupportedApps(t *testing.T) {
	err := validateOTPButton(0, 0, entity.TemplateButton{
		Type: "OTP", Text: "Autofill", OTPType: "ONE_TAP",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "supported_apps")
}

func TestValidateOTPButton_SupportedAppsRequireBothFields(t *testing.T) {
	err := validateOTPButton(0, 0, entity.TemplateButton{
		Type: "OTP", Text: "Autofill", OTPType: "ONE_TAP",
		SupportedApps: []entity.TemplateOTPApp{
			{PackageName: "com.linktor.app"}, // missing signature_hash
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package_name and signature_hash")
}

func TestValidateOTPButton_ZeroTapRequiresTermsAccepted(t *testing.T) {
	err := validateOTPButton(0, 0, entity.TemplateButton{
		Type: "OTP", Text: "Autofill", OTPType: "ZERO_TAP",
		SupportedApps: []entity.TemplateOTPApp{
			{PackageName: "com.linktor.app", SignatureHash: "abcd1234"},
		},
		// ZeroTapTermsAccepted deliberately omitted
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zero_tap_terms_accepted")
}

func TestValidateOTPButton_ZeroTapHappyPath(t *testing.T) {
	err := validateOTPButton(0, 0, entity.TemplateButton{
		Type: "OTP", Text: "Autofill", OTPType: "ZERO_TAP",
		SupportedApps: []entity.TemplateOTPApp{
			{PackageName: "com.linktor.app", SignatureHash: "abcd1234"},
		},
		ZeroTapTermsAccepted: true,
	})
	assert.NoError(t, err)
}

func TestValidateOTPButton_UnknownType(t *testing.T) {
	err := validateOTPButton(0, 0, entity.TemplateButton{
		Type: "OTP", Text: "?", OTPType: "NEW_FANCY_MODE",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown otp_type")
}

func TestValidateTemplateComponents_OTPErrorBubblesUp(t *testing.T) {
	// Integration: the full validation chain must reject an OTP button
	// missing required fields when it runs through validateTemplateComponents.
	err := validateTemplateComponents([]entity.TemplateComponent{
		{
			Type: "BUTTONS",
			Buttons: []entity.TemplateButton{
				{Type: "OTP", Text: "Copy", OTPType: "ONE_TAP"},
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "supported_apps")
}
