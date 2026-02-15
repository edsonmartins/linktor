package whatsapp_official

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// AuthTemplateType defines the type of authentication template
type AuthTemplateType string

const (
	// AuthTypeCopyCode is the standard copy code button OTP
	AuthTypeCopyCode AuthTemplateType = "copy_code"
	// AuthTypeOneTap enables one-tap autofill (Android and iOS)
	AuthTypeOneTap AuthTemplateType = "one_tap"
	// AuthTypeZeroTap enables zero-tap autofill (Android only)
	AuthTypeZeroTap AuthTemplateType = "zero_tap"
)

// AuthTemplateConfig represents configuration for an authentication template
type AuthTemplateConfig struct {
	// TemplateName is the name of the approved auth template
	TemplateName string

	// LanguageCode is the template language (e.g., "en", "pt_BR")
	LanguageCode string

	// OTP is the one-time password to send
	OTP string

	// AuthType determines the button type
	AuthType AuthTemplateType

	// ExpirationMinutes is how long the OTP is valid (shown in message)
	ExpirationMinutes int

	// PackageName is the Android package name (required for one_tap and zero_tap)
	PackageName string

	// SignatureHash is the Android app signature hash (required for one_tap and zero_tap)
	SignatureHash string
}

// AuthTemplateBuilder helps build authentication template messages
type AuthTemplateBuilder struct {
	config *AuthTemplateConfig
}

// NewAuthTemplateBuilder creates a new authentication template builder
func NewAuthTemplateBuilder(templateName, languageCode string) *AuthTemplateBuilder {
	return &AuthTemplateBuilder{
		config: &AuthTemplateConfig{
			TemplateName:      templateName,
			LanguageCode:      languageCode,
			AuthType:          AuthTypeCopyCode,
			ExpirationMinutes: 10,
		},
	}
}

// SetOTP sets the one-time password
func (b *AuthTemplateBuilder) SetOTP(otp string) *AuthTemplateBuilder {
	b.config.OTP = otp
	return b
}

// SetAuthType sets the authentication type
func (b *AuthTemplateBuilder) SetAuthType(authType AuthTemplateType) *AuthTemplateBuilder {
	b.config.AuthType = authType
	return b
}

// SetExpiration sets the expiration time in minutes
func (b *AuthTemplateBuilder) SetExpiration(minutes int) *AuthTemplateBuilder {
	b.config.ExpirationMinutes = minutes
	return b
}

// SetAndroidApp sets the Android app details for one-tap/zero-tap
func (b *AuthTemplateBuilder) SetAndroidApp(packageName, signatureHash string) *AuthTemplateBuilder {
	b.config.PackageName = packageName
	b.config.SignatureHash = signatureHash
	return b
}

// Build creates the template object for sending
func (b *AuthTemplateBuilder) Build() (*TemplateObject, error) {
	if b.config.OTP == "" {
		return nil, fmt.Errorf("OTP is required")
	}

	// Validate configuration for one-tap and zero-tap
	if b.config.AuthType == AuthTypeOneTap || b.config.AuthType == AuthTypeZeroTap {
		if b.config.PackageName == "" || b.config.SignatureHash == "" {
			return nil, fmt.Errorf("package_name and signature_hash are required for %s", b.config.AuthType)
		}
	}

	components := []TemplateComponent{
		{
			Type: "body",
			Parameters: []TemplateParameter{
				{Type: "text", Text: b.config.OTP},
			},
		},
		{
			Type:    "button",
			SubType: string(b.config.AuthType),
			Index:   intPtr(0),
			Parameters: []TemplateParameter{
				{Type: "text", Text: b.config.OTP},
			},
		},
	}

	return &TemplateObject{
		Name: b.config.TemplateName,
		Language: &TemplateLanguage{
			Policy: "deterministic",
			Code:   b.config.LanguageCode,
		},
		Components: components,
	}, nil
}

// BuildRaw creates a raw map structure for the API
func (b *AuthTemplateBuilder) BuildRaw() (map[string]interface{}, error) {
	if b.config.OTP == "" {
		return nil, fmt.Errorf("OTP is required")
	}

	// Validate configuration for one-tap and zero-tap
	if b.config.AuthType == AuthTypeOneTap || b.config.AuthType == AuthTypeZeroTap {
		if b.config.PackageName == "" || b.config.SignatureHash == "" {
			return nil, fmt.Errorf("package_name and signature_hash are required for %s", b.config.AuthType)
		}
	}

	components := []map[string]interface{}{
		{
			"type": "body",
			"parameters": []map[string]interface{}{
				{"type": "text", "text": b.config.OTP},
			},
		},
	}

	// Build button component based on auth type
	buttonParams := []map[string]interface{}{
		{"type": "text", "text": b.config.OTP},
	}

	buttonComponent := map[string]interface{}{
		"type":       "button",
		"sub_type":   string(b.config.AuthType),
		"index":      0,
		"parameters": buttonParams,
	}

	components = append(components, buttonComponent)

	result := map[string]interface{}{
		"name": b.config.TemplateName,
		"language": map[string]interface{}{
			"policy": "deterministic",
			"code":   b.config.LanguageCode,
		},
		"components": components,
	}

	// Add package_name and signature_hash for one-tap and zero-tap
	if b.config.AuthType == AuthTypeOneTap || b.config.AuthType == AuthTypeZeroTap {
		result["package_name"] = b.config.PackageName
		result["signature_hash"] = b.config.SignatureHash
	}

	return result, nil
}

// AuthTemplateSender sends authentication template messages
type AuthTemplateSender struct {
	client *Client
}

// NewAuthTemplateSender creates a new authentication template sender
func NewAuthTemplateSender(client *Client) *AuthTemplateSender {
	return &AuthTemplateSender{client: client}
}

// SendOTP sends an OTP using the copy-code authentication template
func (s *AuthTemplateSender) SendOTP(ctx context.Context, to, templateName, languageCode, otp string) (*SendMessageResponse, error) {
	template, err := NewAuthTemplateBuilder(templateName, languageCode).
		SetOTP(otp).
		SetAuthType(AuthTypeCopyCode).
		Build()
	if err != nil {
		return nil, err
	}

	req := &SendMessageRequest{
		To:       to,
		Type:     MessageType("template"),
		Template: template,
	}
	return s.client.SendMessage(ctx, req)
}

// SendOneTapOTP sends an OTP with one-tap autofill (Android and iOS)
func (s *AuthTemplateSender) SendOneTapOTP(ctx context.Context, to, templateName, languageCode, otp, packageName, signatureHash string) (*SendMessageResponse, error) {
	templateRaw, err := NewAuthTemplateBuilder(templateName, languageCode).
		SetOTP(otp).
		SetAuthType(AuthTypeOneTap).
		SetAndroidApp(packageName, signatureHash).
		BuildRaw()
	if err != nil {
		return nil, err
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

// SendZeroTapOTP sends an OTP with zero-tap autofill (Android only)
func (s *AuthTemplateSender) SendZeroTapOTP(ctx context.Context, to, templateName, languageCode, otp, packageName, signatureHash string) (*SendMessageResponse, error) {
	templateRaw, err := NewAuthTemplateBuilder(templateName, languageCode).
		SetOTP(otp).
		SetAuthType(AuthTypeZeroTap).
		SetAndroidApp(packageName, signatureHash).
		BuildRaw()
	if err != nil {
		return nil, err
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

// OTPGenerator handles OTP generation
type OTPGenerator struct {
	length int
	digits bool
}

// NewOTPGenerator creates a new OTP generator
func NewOTPGenerator(length int, digitsOnly bool) *OTPGenerator {
	return &OTPGenerator{
		length: length,
		digits: digitsOnly,
	}
}

// Generate generates a new OTP
func (g *OTPGenerator) Generate() (string, error) {
	const digits = "0123456789"
	const alphanumeric = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	charset := alphanumeric
	if g.digits {
		charset = digits
	}

	otp := make([]byte, g.length)
	for i := range otp {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate OTP: %w", err)
		}
		otp[i] = charset[n.Int64()]
	}

	return string(otp), nil
}

// OTPSession represents an active OTP session
type OTPSession struct {
	PhoneNumber string
	OTP         string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Verified    bool
	Attempts    int
}

// IsExpired checks if the OTP session has expired
func (s *OTPSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the OTP session is valid for verification
func (s *OTPSession) IsValid() bool {
	return !s.Verified && !s.IsExpired() && s.Attempts < 3
}

// Verify attempts to verify an OTP
func (s *OTPSession) Verify(otp string) bool {
	if !s.IsValid() {
		return false
	}
	s.Attempts++
	if s.OTP == otp {
		s.Verified = true
		return true
	}
	return false
}

// OTPManager manages OTP sessions
type OTPManager struct {
	generator *OTPGenerator
	sender    *AuthTemplateSender
	sessions  map[string]*OTPSession // key: phone number
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewOTPManager creates a new OTP manager
func NewOTPManager(client *Client, otpLength int, ttlMinutes int) *OTPManager {
	return &OTPManager{
		generator: NewOTPGenerator(otpLength, true),
		sender:    NewAuthTemplateSender(client),
		sessions:  make(map[string]*OTPSession),
		ttl:       time.Duration(ttlMinutes) * time.Minute,
	}
}

// SendOTP generates and sends an OTP to the given phone number
func (m *OTPManager) SendOTP(ctx context.Context, phoneNumber, templateName, languageCode string) (*OTPSession, error) {
	otp, err := m.generator.Generate()
	if err != nil {
		return nil, err
	}

	session := &OTPSession{
		PhoneNumber: phoneNumber,
		OTP:         otp,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(m.ttl),
		Verified:    false,
		Attempts:    0,
	}

	_, err = m.sender.SendOTP(ctx, phoneNumber, templateName, languageCode, otp)
	if err != nil {
		return nil, fmt.Errorf("failed to send OTP: %w", err)
	}

	m.mu.Lock()
	m.sessions[phoneNumber] = session
	m.mu.Unlock()

	return session, nil
}

// VerifyOTP verifies an OTP for the given phone number
func (m *OTPManager) VerifyOTP(phoneNumber, otp string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[phoneNumber]
	if !exists {
		return false, fmt.Errorf("no OTP session found for %s", phoneNumber)
	}

	if !session.IsValid() {
		delete(m.sessions, phoneNumber)
		return false, fmt.Errorf("OTP session expired or too many attempts")
	}

	if session.Verify(otp) {
		delete(m.sessions, phoneNumber)
		return true, nil
	}

	return false, nil
}

// CleanupExpired removes expired OTP sessions
func (m *OTPManager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for phone, session := range m.sessions {
		if session.IsExpired() {
			delete(m.sessions, phone)
		}
	}
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

// Common authentication template constants
const (
	// DefaultOTPLength is the default OTP length
	DefaultOTPLength = 6

	// DefaultOTPTTL is the default OTP TTL in minutes
	DefaultOTPTTL = 10

	// MaxOTPAttempts is the maximum number of verification attempts
	MaxOTPAttempts = 3
)

// AuthTemplateMessage represents the message structure for auth templates
type AuthTemplateMessage struct {
	To       string `json:"to"`
	Template struct {
		Name       string `json:"name"`
		Language   struct {
			Code string `json:"code"`
		} `json:"language"`
		Components []struct {
			Type       string `json:"type"`
			SubType    string `json:"sub_type,omitempty"`
			Index      int    `json:"index,omitempty"`
			Parameters []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"parameters"`
		} `json:"components"`
	} `json:"template"`
}
