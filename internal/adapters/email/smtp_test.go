package email

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMTPProvider(t *testing.T) {
	cfg := &Config{
		Provider:  ProviderSMTP,
		FromEmail: "test@example.com",
		SMTPHost:  "smtp.example.com",
		SMTPPort:  587,
	}
	provider, err := NewSMTPProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestSMTPProvider_ValidateConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p, _ := NewSMTPProvider(&Config{
			Provider:  ProviderSMTP,
			FromEmail: "test@example.com",
			SMTPHost:  "smtp.example.com",
			SMTPPort:  587,
		})
		assert.NoError(t, p.ValidateConfig())
	})

	t.Run("missing host", func(t *testing.T) {
		p, _ := NewSMTPProvider(&Config{
			Provider:  ProviderSMTP,
			FromEmail: "test@example.com",
			SMTPPort:  587,
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "smtp_host is required")
	})

	t.Run("missing port", func(t *testing.T) {
		p, _ := NewSMTPProvider(&Config{
			Provider:  ProviderSMTP,
			FromEmail: "test@example.com",
			SMTPHost:  "smtp.example.com",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "smtp_port is required")
	})

	t.Run("missing from_email", func(t *testing.T) {
		p, _ := NewSMTPProvider(&Config{
			Provider: ProviderSMTP,
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from_email is required")
	})
}

func TestSMTPProvider_GetProviderName(t *testing.T) {
	p, _ := NewSMTPProvider(&Config{})
	assert.Equal(t, "smtp", p.GetProviderName())
}

func TestExtractDomain(t *testing.T) {
	assert.Equal(t, "example.com", extractDomain("user@example.com"))
	assert.Equal(t, "localhost", extractDomain("user"))
	assert.Equal(t, "sub.example.com", extractDomain("user@sub.example.com"))
}

func TestEncodeBase64(t *testing.T) {
	data := []byte("Hello, World!")
	encoded := encodeBase64(data)

	// The output should be valid base64 when we strip the line wrapping
	expected := base64.StdEncoding.EncodeToString(data)
	// encodeBase64 wraps at 76 chars and adds \r\n, so the trimmed content should match
	assert.Contains(t, encoded, expected)
}

func TestBase64EncodedLen(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{1, 4},
		{2, 4},
		{3, 4},
		{4, 8},
		{5, 8},
		{6, 8},
		{7, 12},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, base64EncodedLen(tt.input))
	}
}
