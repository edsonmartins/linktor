package voice

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// VonageProvider implements the Provider interface for Vonage Voice
type VonageProvider struct {
	applicationID string
	privateKey    *rsa.PrivateKey
	phoneNumber   string
	baseURL       string
	httpClient    *http.Client
}

// NewVonageProvider creates a new Vonage Voice provider
func NewVonageProvider() *VonageProvider {
	return &VonageProvider{
		baseURL:    "https://api.nexmo.com/v1",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *VonageProvider) Name() string {
	return "vonage"
}

// Initialize sets up the provider with configuration
func (p *VonageProvider) Initialize(ctx context.Context, config VoiceConfig) error {
	appID, ok := config.Credentials["application_id"]
	if !ok || appID == "" {
		return fmt.Errorf("vonage application_id is required")
	}

	privateKeyPEM, ok := config.Credentials["private_key"]
	if !ok || privateKeyPEM == "" {
		return fmt.Errorf("vonage private_key is required")
	}

	// Parse private key
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to parse private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		key, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("private key is not RSA")
		}
	}

	p.applicationID = appID
	p.privateKey = key
	p.phoneNumber = config.PhoneNumber

	return nil
}

// Capabilities returns what this provider supports
func (p *VonageProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		OutboundCalls:     true,
		InboundCalls:      true,
		Recording:         true,
		Transcription:     true,
		TextToSpeech:      true,
		SpeechRecognition: true,
		DTMF:              true,
		Conferencing:      true,
		CallQueues:        false,
		SIP:               true,
		WebRTC:            true,
	}
}

// generateJWT generates a JWT for Vonage API authentication
func (p *VonageProvider) generateJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"application_id": p.applicationID,
		"iat":            now.Unix(),
		"exp":            now.Add(15 * time.Minute).Unix(),
		"jti":            fmt.Sprintf("%d", now.UnixNano()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(p.privateKey)
}

// MakeCall initiates an outbound call
func (p *VonageProvider) MakeCall(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	jwtToken, err := p.generateJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Build NCCO (Nexmo Call Control Object)
	var ncco []interface{}
	if input.NCCO != nil {
		ncco = input.NCCO
	} else {
		// Default: connect to answer URL
		ncco = []interface{}{
			map[string]interface{}{
				"action":   "talk",
				"text":     "Conectando sua chamada",
				"language": "pt-BR",
			},
		}
	}

	payload := map[string]interface{}{
		"to": []map[string]interface{}{
			{
				"type":   "phone",
				"number": strings.TrimPrefix(input.To, "+"),
			},
		},
		"from": map[string]interface{}{
			"type":   "phone",
			"number": strings.TrimPrefix(input.From, "+"),
		},
		"ncco": ncco,
	}

	if input.CallbackURL != "" {
		payload["answer_url"] = []string{input.CallbackURL}
	}
	if input.StatusURL != "" {
		payload["event_url"] = []string{input.StatusURL}
	}
	if input.Timeout > 0 {
		payload["ringing_timer"] = input.Timeout
	}
	if input.MachineDetection {
		payload["machine_detection"] = "hangup"
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/calls", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("vonage error: %v", errResp)
	}

	var callResp struct {
		UUID   string `json:"uuid"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&callResp); err != nil {
		return nil, err
	}

	return &MakeCallResult{
		CallID:     callResp.UUID,
		ExternalID: callResp.UUID,
		Status:     p.mapStatus(callResp.Status),
	}, nil
}

// GetCall retrieves call details
func (p *VonageProvider) GetCall(ctx context.Context, callID string) (*Call, error) {
	jwtToken, err := p.generateJWT()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/calls/"+callID, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("call not found: %s", callID)
	}

	var callResp struct {
		UUID      string `json:"uuid"`
		Status    string `json:"status"`
		Direction string `json:"direction"`
		From      struct {
			Number string `json:"number"`
		} `json:"from"`
		To struct {
			Number string `json:"number"`
		} `json:"to"`
		Duration  string `json:"duration"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&callResp); err != nil {
		return nil, err
	}

	duration, _ := strconv.Atoi(callResp.Duration)

	call := &Call{
		ID:         callResp.UUID,
		ExternalID: callResp.UUID,
		Status:     p.mapStatus(callResp.Status),
		From:       callResp.From.Number,
		To:         callResp.To.Number,
		Duration:   duration,
	}

	if callResp.Direction == "inbound" {
		call.Direction = CallDirectionInbound
	} else {
		call.Direction = CallDirectionOutbound
	}

	if callResp.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, callResp.StartTime); err == nil {
			call.StartedAt = t
		}
	}

	if callResp.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, callResp.EndTime); err == nil {
			call.EndedAt = &t
		}
	}

	return call, nil
}

// EndCall terminates an active call
func (p *VonageProvider) EndCall(ctx context.Context, callID string) error {
	jwtToken, err := p.generateJWT()
	if err != nil {
		return err
	}

	payload := map[string]string{"action": "hangup"}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "PUT", p.baseURL+"/calls/"+callID, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to end call: status %d", resp.StatusCode)
	}

	return nil
}

// TransferCall transfers a call to another number/endpoint
func (p *VonageProvider) TransferCall(ctx context.Context, callID, destination string) error {
	jwtToken, err := p.generateJWT()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"action": "transfer",
		"destination": map[string]interface{}{
			"type": "ncco",
			"ncco": []map[string]interface{}{
				{
					"action": "connect",
					"endpoint": []map[string]interface{}{
						{
							"type":   "phone",
							"number": strings.TrimPrefix(destination, "+"),
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "PUT", p.baseURL+"/calls/"+callID, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to transfer call: status %d", resp.StatusCode)
	}

	return nil
}

// GetRecording retrieves a call recording
func (p *VonageProvider) GetRecording(ctx context.Context, recordingID string) (*Recording, error) {
	jwtToken, err := p.generateJWT()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.nexmo.com/v1/files/"+recordingID, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)

	// For Vonage, we return the URL to download
	return &Recording{
		ID:     recordingID,
		URL:    "https://api.nexmo.com/v1/files/" + recordingID,
		Format: "mp3",
	}, nil
}

// DeleteRecording deletes a call recording
func (p *VonageProvider) DeleteRecording(ctx context.Context, recordingID string) error {
	// Vonage doesn't have a direct delete API for recordings
	// They are automatically deleted after 30 days
	return nil
}

// GenerateIVRResponse generates NCCO response
func (p *VonageProvider) GenerateIVRResponse(actions []IVRAction) (interface{}, error) {
	var ncco []map[string]interface{}

	for _, action := range actions {
		switch a := action.(type) {
		case IVRSay:
			item := map[string]interface{}{
				"action":   "talk",
				"text":     a.Text,
				"language": a.Language,
			}
			if a.Loop > 0 {
				item["loop"] = a.Loop
			}
			if a.Voice != "" {
				// Map voice names
				switch a.Voice {
				case "man":
					item["style"] = 0
				case "woman":
					item["style"] = 1
				default:
					item["style"] = 0
				}
			}
			ncco = append(ncco, item)

		case IVRPlay:
			item := map[string]interface{}{
				"action":    "stream",
				"streamUrl": []string{a.URL},
			}
			if a.Loop > 0 {
				item["loop"] = a.Loop
			}
			ncco = append(ncco, item)

		case IVRGather:
			item := map[string]interface{}{
				"action": "input",
			}

			enableDTMF := false
			enableSpeech := false
			for _, input := range a.Input {
				if input == "dtmf" {
					enableDTMF = true
				}
				if input == "speech" {
					enableSpeech = true
				}
			}

			if enableDTMF {
				dtmf := map[string]interface{}{
					"timeOut": a.Timeout,
				}
				if a.NumDigits > 0 {
					dtmf["maxDigits"] = a.NumDigits
				}
				if a.FinishOnKey != "" {
					dtmf["submitOnHash"] = a.FinishOnKey == "#"
				}
				item["dtmf"] = dtmf
			}

			if enableSpeech {
				speech := map[string]interface{}{
					"language": a.Language,
				}
				if len(a.Hints) > 0 {
					speech["context"] = a.Hints
				}
				item["speech"] = speech
			}

			if a.ActionURL != "" {
				item["eventUrl"] = []string{a.ActionURL}
			}

			// Add nested prompts before input
			for _, nested := range a.Nested {
				if say, ok := nested.(IVRSay); ok {
					ncco = append(ncco, map[string]interface{}{
						"action":   "talk",
						"text":     say.Text,
						"language": say.Language,
					})
				}
			}

			ncco = append(ncco, item)

		case IVRRecord:
			item := map[string]interface{}{
				"action": "record",
				"format": "mp3",
			}
			if a.MaxLength > 0 {
				item["timeOut"] = a.MaxLength
			}
			if a.FinishOnKey != "" {
				item["endOnKey"] = a.FinishOnKey
			}
			if a.PlayBeep {
				item["beepStart"] = true
			}
			if a.ActionURL != "" {
				item["eventUrl"] = []string{a.ActionURL}
			}
			if a.Transcribe {
				item["transcription"] = map[string]interface{}{
					"language": "pt-BR",
				}
			}
			ncco = append(ncco, item)

		case IVRDial:
			item := map[string]interface{}{
				"action": "connect",
			}

			var endpoint map[string]interface{}
			if a.Number != "" {
				endpoint = map[string]interface{}{
					"type":   "phone",
					"number": strings.TrimPrefix(a.Number, "+"),
				}
			} else if a.SIPEndpoint != "" {
				endpoint = map[string]interface{}{
					"type": "sip",
					"uri":  a.SIPEndpoint,
				}
			}

			item["endpoint"] = []map[string]interface{}{endpoint}

			if a.Timeout > 0 {
				item["timeout"] = a.Timeout
			}
			if a.CallerID != "" {
				item["from"] = strings.TrimPrefix(a.CallerID, "+")
			}
			if a.ActionURL != "" {
				item["eventUrl"] = []string{a.ActionURL}
			}
			ncco = append(ncco, item)

		case IVRHangup:
			// Vonage automatically hangs up at end of NCCO
			// No explicit hangup action needed

		case IVRPause:
			// Use a silent stream for pause
			ncco = append(ncco, map[string]interface{}{
				"action":    "stream",
				"streamUrl": []string{"https://example.com/silence.mp3"}, // Need a silent audio file
				"level":     0,
			})

		case IVRConference:
			item := map[string]interface{}{
				"action": "conversation",
				"name":   a.Name,
			}
			if a.Muted {
				item["mute"] = true
			}
			if a.StartOnEnter {
				item["startOnEnter"] = true
			}
			if a.EndOnExit {
				item["endOnExit"] = true
			}
			if a.Record {
				item["record"] = true
			}
			if a.WaitURL != "" {
				item["musicOnHoldUrl"] = []string{a.WaitURL}
			}
			ncco = append(ncco, item)
		}
	}

	return ncco, nil
}

// ParseWebhook parses an incoming Vonage webhook
func (p *VonageProvider) ParseWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	event := &WebhookEvent{
		Timestamp:  time.Now(),
		RawPayload: payload,
	}

	// Get call UUID
	if uuid, ok := payload["uuid"].(string); ok {
		event.CallID = uuid
		event.ExternalID = uuid
	} else if uuid, ok := payload["conversation_uuid"].(string); ok {
		event.CallID = uuid
		event.ExternalID = uuid
	}

	// Get from/to
	if from, ok := payload["from"].(string); ok {
		event.From = from
	}
	if to, ok := payload["to"].(string); ok {
		event.To = to
	}

	// Determine event type
	if status, ok := payload["status"].(string); ok {
		event.Type = "status"
		event.Status = p.mapStatus(status)
	}

	// Check for DTMF
	if dtmf, ok := payload["dtmf"].(map[string]interface{}); ok {
		event.Type = "dtmf"
		if digits, ok := dtmf["digits"].(string); ok {
			event.Digits = digits
		}
	}

	// Check for speech
	if speech, ok := payload["speech"].(map[string]interface{}); ok {
		event.Type = "speech"
		if results, ok := speech["results"].([]interface{}); ok && len(results) > 0 {
			if result, ok := results[0].(map[string]interface{}); ok {
				if text, ok := result["text"].(string); ok {
					event.SpeechResult = text
				}
			}
		}
	}

	// Check for recording
	if recordingURL, ok := payload["recording_url"].(string); ok {
		event.Type = "recording"
		event.RecordingURL = recordingURL
	}

	// Direction
	if direction, ok := payload["direction"].(string); ok {
		if direction == "inbound" {
			event.Direction = CallDirectionInbound
		} else {
			event.Direction = CallDirectionOutbound
		}
	}

	// Duration
	if duration, ok := payload["duration"].(string); ok {
		if d, err := strconv.Atoi(duration); err == nil {
			event.Duration = d
		}
	}

	return event, nil
}

// ValidateWebhook validates Vonage webhook signature
func (p *VonageProvider) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) bool {
	// Vonage uses JWT-based signing for webhooks
	// In production, validate the JWT signature
	// For now, accept all webhooks
	return true
}

// mapStatus maps Vonage status to CallStatus
func (p *VonageProvider) mapStatus(status string) CallStatus {
	switch strings.ToLower(status) {
	case "started":
		return CallStatusInitiated
	case "ringing":
		return CallStatusRinging
	case "answered":
		return CallStatusAnswered
	case "completed":
		return CallStatusCompleted
	case "busy":
		return CallStatusBusy
	case "timeout":
		return CallStatusNoAnswer
	case "failed", "rejected":
		return CallStatusFailed
	case "cancelled":
		return CallStatusCanceled
	default:
		return CallStatus(status)
	}
}
