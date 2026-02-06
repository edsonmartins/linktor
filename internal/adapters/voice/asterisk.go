package voice

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AsteriskProvider implements the Provider interface for Asterisk PBX
// Supports both AMI (Asterisk Manager Interface) and ARI (Asterisk REST Interface)
type AsteriskProvider struct {
	ariURL       string
	ariUser      string
	ariPassword  string
	amiHost      string
	amiPort      int
	amiUser      string
	amiPassword  string
	context      string
	phoneNumber  string
	httpClient   *http.Client
	amiConn      net.Conn
	amiMutex     sync.Mutex
	eventHandler func(event map[string]string)
}

// NewAsteriskProvider creates a new Asterisk provider
func NewAsteriskProvider() *AsteriskProvider {
	return &AsteriskProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		context:    "default",
	}
}

// Name returns the provider name
func (p *AsteriskProvider) Name() string {
	return "asterisk"
}

// Initialize sets up the provider with configuration
func (p *AsteriskProvider) Initialize(ctx context.Context, config VoiceConfig) error {
	// ARI settings
	if ariURL, ok := config.Credentials["ari_url"]; ok {
		p.ariURL = ariURL
	}
	if ariUser, ok := config.Credentials["ari_user"]; ok {
		p.ariUser = ariUser
	}
	if ariPassword, ok := config.Credentials["ari_password"]; ok {
		p.ariPassword = ariPassword
	}

	// AMI settings (optional, for legacy support)
	if amiHost, ok := config.Credentials["ami_host"]; ok {
		p.amiHost = amiHost
	}
	if amiPort, ok := config.Credentials["ami_port"]; ok {
		if port, err := strconv.Atoi(amiPort); err == nil {
			p.amiPort = port
		}
	}
	if p.amiPort == 0 {
		p.amiPort = 5038
	}
	if amiUser, ok := config.Credentials["ami_user"]; ok {
		p.amiUser = amiUser
	}
	if amiPassword, ok := config.Credentials["ami_password"]; ok {
		p.amiPassword = amiPassword
	}

	// Context for dialplan
	if ctx, ok := config.Credentials["context"]; ok {
		p.context = ctx
	}

	p.phoneNumber = config.PhoneNumber

	// Validate configuration
	if p.ariURL == "" && p.amiHost == "" {
		return fmt.Errorf("either ari_url or ami_host is required")
	}

	// Connect to AMI if configured
	if p.amiHost != "" {
		if err := p.connectAMI(ctx); err != nil {
			return fmt.Errorf("failed to connect to AMI: %w", err)
		}
	}

	return nil
}

// Capabilities returns what this provider supports
func (p *AsteriskProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		OutboundCalls:     true,
		InboundCalls:      true,
		Recording:         true,
		Transcription:     false, // Requires external service
		TextToSpeech:      true,  // Via Festival, eSpeak, or cloud
		SpeechRecognition: false, // Requires external service
		DTMF:              true,
		Conferencing:      true,
		CallQueues:        true,
		SIP:               true,
		WebRTC:            true, // Via WebRTC module
	}
}

// connectAMI establishes connection to Asterisk Manager Interface
func (p *AsteriskProvider) connectAMI(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", p.amiHost, p.amiPort)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}

	p.amiConn = conn

	// Read the initial banner
	reader := bufio.NewReader(conn)
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	// Login
	loginCmd := fmt.Sprintf("Action: Login\r\nUsername: %s\r\nSecret: %s\r\n\r\n",
		p.amiUser, p.amiPassword)
	_, err = conn.Write([]byte(loginCmd))
	if err != nil {
		return err
	}

	// Read login response
	response, err := p.readAMIResponse(reader)
	if err != nil {
		return err
	}

	if response["Response"] != "Success" {
		return fmt.Errorf("AMI login failed: %s", response["Message"])
	}

	// Start event listener in background
	go p.listenAMIEvents(reader)

	return nil
}

// readAMIResponse reads a single AMI response
func (p *AsteriskProvider) readAMIResponse(reader *bufio.Reader) (map[string]string, error) {
	response := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			response[parts[0]] = parts[1]
		}
	}

	return response, nil
}

// listenAMIEvents listens for AMI events
func (p *AsteriskProvider) listenAMIEvents(reader *bufio.Reader) {
	for {
		event, err := p.readAMIResponse(reader)
		if err != nil {
			// Connection closed
			return
		}

		if p.eventHandler != nil && event["Event"] != "" {
			p.eventHandler(event)
		}
	}
}

// sendAMIAction sends an action to AMI
func (p *AsteriskProvider) sendAMIAction(action string, params map[string]string) (map[string]string, error) {
	p.amiMutex.Lock()
	defer p.amiMutex.Unlock()

	if p.amiConn == nil {
		return nil, fmt.Errorf("AMI not connected")
	}

	// Build command
	var cmd strings.Builder
	cmd.WriteString(fmt.Sprintf("Action: %s\r\n", action))
	for k, v := range params {
		cmd.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	cmd.WriteString("\r\n")

	_, err := p.amiConn.Write([]byte(cmd.String()))
	if err != nil {
		return nil, err
	}

	// Read response
	reader := bufio.NewReader(p.amiConn)
	return p.readAMIResponse(reader)
}

// MakeCall initiates an outbound call
func (p *AsteriskProvider) MakeCall(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	// Use ARI if available
	if p.ariURL != "" {
		return p.makeCallARI(ctx, input)
	}

	// Fall back to AMI
	return p.makeCallAMI(ctx, input)
}

// makeCallARI makes a call using ARI
func (p *AsteriskProvider) makeCallARI(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	endpoint := fmt.Sprintf("%s/channels", p.ariURL)

	params := map[string]interface{}{
		"endpoint": fmt.Sprintf("PJSIP/%s", input.To),
		"app":      "linktor",
		"callerId": input.From,
	}

	if input.Timeout > 0 {
		params["timeout"] = input.Timeout
	}

	body, _ := json.Marshal(params)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.ariUser, p.ariPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("ARI error: %v", errResp)
	}

	var channel struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, err
	}

	return &MakeCallResult{
		CallID:     channel.ID,
		ExternalID: channel.ID,
		Status:     p.mapState(channel.State),
	}, nil
}

// makeCallAMI makes a call using AMI Originate
func (p *AsteriskProvider) makeCallAMI(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	params := map[string]string{
		"Channel":   fmt.Sprintf("PJSIP/%s", input.To),
		"Context":   p.context,
		"Exten":     input.To,
		"Priority":  "1",
		"CallerID":  input.From,
		"Async":     "true",
	}

	if input.Timeout > 0 {
		params["Timeout"] = strconv.Itoa(input.Timeout * 1000) // milliseconds
	}

	response, err := p.sendAMIAction("Originate", params)
	if err != nil {
		return nil, err
	}

	if response["Response"] != "Success" {
		return nil, fmt.Errorf("originate failed: %s", response["Message"])
	}

	// Generate a call ID (AMI doesn't return one directly)
	callID := fmt.Sprintf("ast_%d", time.Now().UnixNano())

	return &MakeCallResult{
		CallID:     callID,
		ExternalID: response["ActionID"],
		Status:     CallStatusInitiated,
	}, nil
}

// GetCall retrieves call details
func (p *AsteriskProvider) GetCall(ctx context.Context, callID string) (*Call, error) {
	if p.ariURL != "" {
		return p.getCallARI(ctx, callID)
	}
	return nil, fmt.Errorf("get call not supported via AMI only")
}

// getCallARI gets call details via ARI
func (p *AsteriskProvider) getCallARI(ctx context.Context, callID string) (*Call, error) {
	endpoint := fmt.Sprintf("%s/channels/%s", p.ariURL, callID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.ariUser, p.ariPassword)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("call not found: %s", callID)
	}

	var channel struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		State     string `json:"state"`
		Caller    struct {
			Number string `json:"number"`
			Name   string `json:"name"`
		} `json:"caller"`
		Connected struct {
			Number string `json:"number"`
			Name   string `json:"name"`
		} `json:"connected"`
		Creationtime time.Time `json:"creationtime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, err
	}

	call := &Call{
		ID:         channel.ID,
		ExternalID: channel.ID,
		Status:     p.mapState(channel.State),
		From:       channel.Caller.Number,
		CallerName: channel.Caller.Name,
		To:         channel.Connected.Number,
		CreatedAt:  channel.Creationtime,
		StartedAt:  channel.Creationtime,
	}

	return call, nil
}

// EndCall terminates an active call
func (p *AsteriskProvider) EndCall(ctx context.Context, callID string) error {
	if p.ariURL != "" {
		return p.endCallARI(ctx, callID)
	}
	return p.endCallAMI(ctx, callID)
}

// endCallARI ends a call via ARI
func (p *AsteriskProvider) endCallARI(ctx context.Context, callID string) error {
	endpoint := fmt.Sprintf("%s/channels/%s", p.ariURL, callID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(p.ariUser, p.ariPassword)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("failed to end call: status %d", resp.StatusCode)
	}

	return nil
}

// endCallAMI ends a call via AMI
func (p *AsteriskProvider) endCallAMI(ctx context.Context, callID string) error {
	_, err := p.sendAMIAction("Hangup", map[string]string{
		"Channel": callID,
	})
	return err
}

// TransferCall transfers a call to another number/endpoint
func (p *AsteriskProvider) TransferCall(ctx context.Context, callID, destination string) error {
	if p.ariURL != "" {
		// Use ARI to redirect channel
		endpoint := fmt.Sprintf("%s/channels/%s/redirect", p.ariURL, callID)

		params := map[string]interface{}{
			"endpoint": fmt.Sprintf("PJSIP/%s", destination),
		}
		body, _ := json.Marshal(params)

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.SetBasicAuth(p.ariUser, p.ariPassword)
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("failed to transfer: status %d", resp.StatusCode)
		}

		return nil
	}

	// AMI redirect
	_, err := p.sendAMIAction("Redirect", map[string]string{
		"Channel":  callID,
		"Context":  p.context,
		"Exten":    destination,
		"Priority": "1",
	})
	return err
}

// GetRecording retrieves a call recording
func (p *AsteriskProvider) GetRecording(ctx context.Context, recordingID string) (*Recording, error) {
	if p.ariURL == "" {
		return nil, fmt.Errorf("recordings require ARI")
	}

	endpoint := fmt.Sprintf("%s/recordings/stored/%s", p.ariURL, recordingID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.ariUser, p.ariPassword)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("recording not found: %s", recordingID)
	}

	var rec struct {
		Name      string `json:"name"`
		Format    string `json:"format"`
		Duration  int    `json:"duration_seconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
		return nil, err
	}

	return &Recording{
		ID:       recordingID,
		URL:      fmt.Sprintf("%s/recordings/stored/%s/file", p.ariURL, recordingID),
		Duration: rec.Duration,
		Format:   rec.Format,
	}, nil
}

// DeleteRecording deletes a call recording
func (p *AsteriskProvider) DeleteRecording(ctx context.Context, recordingID string) error {
	if p.ariURL == "" {
		return fmt.Errorf("recordings require ARI")
	}

	endpoint := fmt.Sprintf("%s/recordings/stored/%s", p.ariURL, recordingID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(p.ariUser, p.ariPassword)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("failed to delete recording: status %d", resp.StatusCode)
	}

	return nil
}

// GenerateIVRResponse generates Asterisk dialplan actions (AGI/FastAGI compatible)
func (p *AsteriskProvider) GenerateIVRResponse(actions []IVRAction) (interface{}, error) {
	var agiCommands []string

	for _, action := range actions {
		switch a := action.(type) {
		case IVRSay:
			// Use Festival, eSpeak, or Google TTS
			agiCommands = append(agiCommands, fmt.Sprintf("EXEC Swift \"%s\"", a.Text))

		case IVRPlay:
			// Play sound file
			file := strings.TrimSuffix(a.URL, ".wav")
			file = strings.TrimSuffix(file, ".gsm")
			agiCommands = append(agiCommands, fmt.Sprintf("EXEC Playback \"%s\"", file))

		case IVRGather:
			// Read DTMF
			timeout := a.Timeout
			if timeout == 0 {
				timeout = 10
			}
			digits := a.NumDigits
			if digits == 0 {
				digits = 1
			}
			agiCommands = append(agiCommands, fmt.Sprintf("EXEC Read \"DIGITS,%s,%d,%d\"",
				"beep", digits, timeout))

		case IVRRecord:
			// Record audio
			maxLength := a.MaxLength
			if maxLength == 0 {
				maxLength = 60
			}
			agiCommands = append(agiCommands, fmt.Sprintf("EXEC Record \"/tmp/recording_%d.wav,%d,%d\"",
				time.Now().UnixNano(), maxLength, a.Timeout))

		case IVRDial:
			if a.Number != "" {
				agiCommands = append(agiCommands, fmt.Sprintf("EXEC Dial \"PJSIP/%s,%d\"",
					a.Number, a.Timeout))
			} else if a.Queue != "" {
				agiCommands = append(agiCommands, fmt.Sprintf("EXEC Queue \"%s\"", a.Queue))
			}

		case IVRHangup:
			agiCommands = append(agiCommands, "HANGUP")

		case IVRPause:
			agiCommands = append(agiCommands, fmt.Sprintf("EXEC Wait \"%d\"", a.Length))

		case IVRConference:
			agiCommands = append(agiCommands, fmt.Sprintf("EXEC ConfBridge \"%s\"", a.Name))

		case IVRQueue:
			agiCommands = append(agiCommands, fmt.Sprintf("EXEC Queue \"%s\"", a.Name))
		}
	}

	return agiCommands, nil
}

// ParseWebhook parses an incoming Asterisk webhook (Stasis/ARI events)
func (p *AsteriskProvider) ParseWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	event := &WebhookEvent{
		Timestamp:  time.Now(),
		RawPayload: payload,
	}

	// Parse ARI Stasis event
	if eventType, ok := payload["type"].(string); ok {
		event.Type = strings.ToLower(eventType)
	}

	// Get channel info
	if channel, ok := payload["channel"].(map[string]interface{}); ok {
		if id, ok := channel["id"].(string); ok {
			event.CallID = id
			event.ExternalID = id
		}
		if state, ok := channel["state"].(string); ok {
			event.Status = p.mapState(state)
		}
		if caller, ok := channel["caller"].(map[string]interface{}); ok {
			if number, ok := caller["number"].(string); ok {
				event.From = number
			}
		}
		if connected, ok := channel["connected"].(map[string]interface{}); ok {
			if number, ok := connected["number"].(string); ok {
				event.To = number
			}
		}
	}

	// Check for DTMF
	if digit, ok := payload["digit"].(string); ok {
		event.Type = "dtmf"
		event.Digits = digit
	}

	// Check for recording
	if recording, ok := payload["recording"].(map[string]interface{}); ok {
		event.Type = "recording"
		if name, ok := recording["name"].(string); ok {
			event.RecordingURL = fmt.Sprintf("%s/recordings/stored/%s/file", p.ariURL, name)
		}
	}

	return event, nil
}

// ValidateWebhook validates Asterisk webhook
func (p *AsteriskProvider) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) bool {
	// ARI webhooks are typically from localhost or internal network
	// Add IP whitelisting or auth token validation as needed
	return true
}

// mapState maps Asterisk channel state to CallStatus
func (p *AsteriskProvider) mapState(state string) CallStatus {
	stateMap := map[string]CallStatus{
		"Down":      CallStatusInitiated,
		"Rsrvd":     CallStatusInitiated,
		"OffHook":   CallStatusInitiated,
		"Dialing":   CallStatusInitiated,
		"Ring":      CallStatusRinging,
		"Ringing":   CallStatusRinging,
		"Up":        CallStatusInProgress,
		"Busy":      CallStatusBusy,
		"Unknown":   CallStatusFailed,
	}

	if mapped, ok := stateMap[state]; ok {
		return mapped
	}
	return CallStatus(strings.ToLower(state))
}

// Close closes the AMI connection
func (p *AsteriskProvider) Close() error {
	if p.amiConn != nil {
		p.sendAMIAction("Logoff", nil)
		return p.amiConn.Close()
	}
	return nil
}
