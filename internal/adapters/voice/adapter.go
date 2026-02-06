package voice

import (
	"context"
	"fmt"
)

// Provider defines the interface for voice providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// Initialize sets up the provider with configuration
	Initialize(ctx context.Context, config VoiceConfig) error

	// Capabilities returns what this provider supports
	Capabilities() ProviderCapabilities

	// MakeCall initiates an outbound call
	MakeCall(ctx context.Context, input MakeCallInput) (*MakeCallResult, error)

	// GetCall retrieves call details
	GetCall(ctx context.Context, callID string) (*Call, error)

	// EndCall terminates an active call
	EndCall(ctx context.Context, callID string) error

	// TransferCall transfers a call to another number/endpoint
	TransferCall(ctx context.Context, callID, destination string) error

	// GetRecording retrieves a call recording
	GetRecording(ctx context.Context, recordingID string) (*Recording, error)

	// DeleteRecording deletes a call recording
	DeleteRecording(ctx context.Context, recordingID string) error

	// GenerateIVRResponse generates provider-specific IVR response
	GenerateIVRResponse(actions []IVRAction) (interface{}, error)

	// ParseWebhook parses an incoming webhook request
	ParseWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error)

	// ValidateWebhook validates webhook signature
	ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) bool
}

// Adapter is the main voice adapter that manages providers
type Adapter struct {
	provider Provider
	config   VoiceConfig
	handler  WebhookHandler
}

// WebhookHandler handles incoming voice webhooks
type WebhookHandler func(ctx context.Context, event *WebhookEvent) ([]IVRAction, error)

// NewAdapter creates a new voice adapter
func NewAdapter(config VoiceConfig) (*Adapter, error) {
	var provider Provider

	switch config.Provider {
	case "twilio":
		provider = NewTwilioProvider()
	case "vonage":
		provider = NewVonageProvider()
	case "amazon_connect":
		provider = NewAmazonConnectProvider()
	case "asterisk":
		provider = NewAsteriskProvider()
	case "freeswitch":
		provider = NewFreeSWITCHProvider()
	default:
		return nil, fmt.Errorf("unsupported voice provider: %s", config.Provider)
	}

	return &Adapter{
		provider: provider,
		config:   config,
	}, nil
}

// Initialize initializes the adapter
func (a *Adapter) Initialize(ctx context.Context) error {
	return a.provider.Initialize(ctx, a.config)
}

// SetWebhookHandler sets the handler for incoming webhooks
func (a *Adapter) SetWebhookHandler(handler WebhookHandler) {
	a.handler = handler
}

// Name returns the adapter name
func (a *Adapter) Name() string {
	return "voice"
}

// Type returns the adapter type
func (a *Adapter) Type() string {
	return "voice"
}

// Version returns the adapter version
func (a *Adapter) Version() string {
	return "1.0.0"
}

// Provider returns the current provider name
func (a *Adapter) Provider() string {
	return a.provider.Name()
}

// Capabilities returns provider capabilities
func (a *Adapter) Capabilities() ProviderCapabilities {
	return a.provider.Capabilities()
}

// MakeCall initiates an outbound call
func (a *Adapter) MakeCall(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	// Set defaults from config
	if input.From == "" {
		input.From = a.config.PhoneNumber
	}
	if input.CallbackURL == "" {
		input.CallbackURL = a.config.WebhookURL
	}
	if input.StatusURL == "" {
		input.StatusURL = a.config.StatusURL
	}
	if a.config.RecordCalls {
		input.Record = true
	}
	if a.config.TranscribeCalls {
		input.Transcribe = true
	}

	return a.provider.MakeCall(ctx, input)
}

// GetCall retrieves call details
func (a *Adapter) GetCall(ctx context.Context, callID string) (*Call, error) {
	return a.provider.GetCall(ctx, callID)
}

// EndCall terminates an active call
func (a *Adapter) EndCall(ctx context.Context, callID string) error {
	return a.provider.EndCall(ctx, callID)
}

// TransferCall transfers a call to another number/endpoint
func (a *Adapter) TransferCall(ctx context.Context, callID, destination string) error {
	return a.provider.TransferCall(ctx, callID, destination)
}

// GetRecording retrieves a call recording
func (a *Adapter) GetRecording(ctx context.Context, recordingID string) (*Recording, error) {
	return a.provider.GetRecording(ctx, recordingID)
}

// DeleteRecording deletes a call recording
func (a *Adapter) DeleteRecording(ctx context.Context, recordingID string) error {
	return a.provider.DeleteRecording(ctx, recordingID)
}

// HandleWebhook processes an incoming webhook
func (a *Adapter) HandleWebhook(ctx context.Context, headers map[string]string, body []byte) (interface{}, error) {
	// Validate webhook signature
	if !a.provider.ValidateWebhook(ctx, headers, body) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	// Parse webhook event
	event, err := a.provider.ParseWebhook(ctx, headers, body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	// If no handler is set, return empty response
	if a.handler == nil {
		return a.provider.GenerateIVRResponse(nil)
	}

	// Process event and get IVR actions
	actions, err := a.handler(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("webhook handler error: %w", err)
	}

	// Generate provider-specific response
	return a.provider.GenerateIVRResponse(actions)
}

// GenerateIVRResponse generates IVR response for the current provider
func (a *Adapter) GenerateIVRResponse(actions []IVRAction) (interface{}, error) {
	return a.provider.GenerateIVRResponse(actions)
}

// BuildIVRMenu creates a simple IVR menu
func BuildIVRMenu(greeting string, options map[string]string, timeout int) []IVRAction {
	var actions []IVRAction

	// Add greeting
	actions = append(actions, IVRSay{
		Text:     greeting,
		Language: "pt-BR",
	})

	// Build options text
	optionsText := ""
	for digit, description := range options {
		optionsText += fmt.Sprintf("Pressione %s para %s. ", digit, description)
	}

	// Add gather with options
	actions = append(actions, IVRGather{
		Input:     []string{"dtmf", "speech"},
		Timeout:   timeout,
		NumDigits: 1,
		Nested: []IVRAction{
			IVRSay{
				Text:     optionsText,
				Language: "pt-BR",
			},
		},
	})

	// Add fallback for no input
	actions = append(actions, IVRSay{
		Text:     "Desculpe, não entendi sua escolha. Por favor, tente novamente.",
		Language: "pt-BR",
	})

	return actions
}

// BuildCallbackRequest creates IVR actions for a callback request
func BuildCallbackRequest() []IVRAction {
	return []IVRAction{
		IVRSay{
			Text:     "Por favor, diga ou digite o número de telefone para retornarmos a ligação.",
			Language: "pt-BR",
		},
		IVRGather{
			Input:       []string{"dtmf", "speech"},
			Timeout:     10,
			NumDigits:   11,
			FinishOnKey: "#",
		},
		IVRSay{
			Text:     "Obrigado. Retornaremos sua ligação em breve.",
			Language: "pt-BR",
		},
		IVRHangup{},
	}
}

// BuildHoldMusic creates IVR actions for hold music
func BuildHoldMusic(musicURL string, message string, messageInterval int) []IVRAction {
	return []IVRAction{
		IVRSay{
			Text:     message,
			Language: "pt-BR",
		},
		IVRPlay{
			URL:  musicURL,
			Loop: 10,
		},
	}
}
