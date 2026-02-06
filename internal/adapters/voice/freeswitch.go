package voice

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// FreeSWITCHProvider implements the Provider interface for FreeSWITCH
type FreeSWITCHProvider struct {
	config     VoiceConfig
	httpClient *http.Client

	// ESL (Event Socket Library) connection
	eslConn   net.Conn
	eslMutex  sync.Mutex
	eslReader *bufio.Reader

	// Active calls tracking
	calls      map[string]*Call
	callsMutex sync.RWMutex
}

// FreeSWITCH ESL event
type ESLEvent struct {
	Headers map[string]string
	Body    string
}

// FreeSWITCH dialplan action
type FSAction struct {
	Application string      `json:"application"`
	Data        interface{} `json:"data,omitempty"`
}

// NewFreeSWITCHProvider creates a new FreeSWITCH provider
func NewFreeSWITCHProvider() *FreeSWITCHProvider {
	return &FreeSWITCHProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		calls: make(map[string]*Call),
	}
}

// Name returns the provider name
func (p *FreeSWITCHProvider) Name() string {
	return "freeswitch"
}

// Initialize sets up the FreeSWITCH provider
func (p *FreeSWITCHProvider) Initialize(ctx context.Context, config VoiceConfig) error {
	p.config = config

	// Validate required configuration
	eslHost := config.Credentials["esl_host"]
	eslPort := config.Credentials["esl_port"]
	eslPassword := config.Credentials["esl_password"]

	if eslHost == "" {
		eslHost = "127.0.0.1"
	}
	if eslPort == "" {
		eslPort = "8021"
	}
	if eslPassword == "" {
		return fmt.Errorf("esl_password is required")
	}

	// Connect to FreeSWITCH ESL
	address := fmt.Sprintf("%s:%s", eslHost, eslPort)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to FreeSWITCH ESL: %w", err)
	}

	p.eslConn = conn
	p.eslReader = bufio.NewReader(conn)

	// Authenticate with ESL
	if err := p.eslAuth(eslPassword); err != nil {
		p.eslConn.Close()
		return fmt.Errorf("ESL authentication failed: %w", err)
	}

	// Subscribe to events
	if err := p.eslSubscribe(); err != nil {
		p.eslConn.Close()
		return fmt.Errorf("ESL event subscription failed: %w", err)
	}

	// Start event listener
	go p.eslEventLoop(ctx)

	return nil
}

// eslAuth authenticates with FreeSWITCH ESL
func (p *FreeSWITCHProvider) eslAuth(password string) error {
	// Read auth request
	event, err := p.eslReadEvent()
	if err != nil {
		return err
	}

	if event.Headers["Content-Type"] != "auth/request" {
		return fmt.Errorf("expected auth/request, got %s", event.Headers["Content-Type"])
	}

	// Send auth command
	if err := p.eslSendCommand(fmt.Sprintf("auth %s", password)); err != nil {
		return err
	}

	// Read auth response
	event, err = p.eslReadEvent()
	if err != nil {
		return err
	}

	if event.Headers["Reply-Text"] != "+OK accepted" {
		return fmt.Errorf("authentication failed: %s", event.Headers["Reply-Text"])
	}

	return nil
}

// eslSubscribe subscribes to FreeSWITCH events
func (p *FreeSWITCHProvider) eslSubscribe() error {
	// Subscribe to call-related events
	events := []string{
		"CHANNEL_CREATE",
		"CHANNEL_ANSWER",
		"CHANNEL_HANGUP",
		"CHANNEL_HANGUP_COMPLETE",
		"DTMF",
		"RECORD_START",
		"RECORD_STOP",
		"DETECTED_SPEECH",
		"CUSTOM",
	}

	cmd := fmt.Sprintf("event plain %s", strings.Join(events, " "))
	return p.eslSendCommand(cmd)
}

// eslSendCommand sends a command to FreeSWITCH ESL
func (p *FreeSWITCHProvider) eslSendCommand(cmd string) error {
	p.eslMutex.Lock()
	defer p.eslMutex.Unlock()

	_, err := fmt.Fprintf(p.eslConn, "%s\n\n", cmd)
	return err
}

// eslReadEvent reads an event from FreeSWITCH ESL
func (p *FreeSWITCHProvider) eslReadEvent() (*ESLEvent, error) {
	event := &ESLEvent{
		Headers: make(map[string]string),
	}

	// Read headers
	for {
		line, err := p.eslReader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			event.Headers[parts[0]] = parts[1]
		}
	}

	// Read body if present
	if contentLength := event.Headers["Content-Length"]; contentLength != "" {
		var length int
		fmt.Sscanf(contentLength, "%d", &length)
		if length > 0 {
			body := make([]byte, length)
			_, err := p.eslReader.Read(body)
			if err != nil {
				return nil, err
			}
			event.Body = string(body)
		}
	}

	return event, nil
}

// eslEventLoop processes incoming ESL events
func (p *FreeSWITCHProvider) eslEventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			event, err := p.eslReadEvent()
			if err != nil {
				// Connection closed or error
				return
			}
			p.handleESLEvent(event)
		}
	}
}

// handleESLEvent processes an ESL event
func (p *FreeSWITCHProvider) handleESLEvent(event *ESLEvent) {
	eventName := event.Headers["Event-Name"]
	callID := event.Headers["Unique-ID"]

	if callID == "" {
		return
	}

	p.callsMutex.Lock()
	defer p.callsMutex.Unlock()

	switch eventName {
	case "CHANNEL_CREATE":
		call := &Call{
			ID:         callID,
			ExternalID: callID,
			Status:     CallStatusInitiated,
			From:       event.Headers["Caller-Caller-ID-Number"],
			To:         event.Headers["Caller-Destination-Number"],
			Direction:  p.parseDirection(event.Headers["Call-Direction"]),
			StartedAt:  time.Now(),
		}
		p.calls[callID] = call

	case "CHANNEL_ANSWER":
		if call, ok := p.calls[callID]; ok {
			call.Status = CallStatusInProgress
			now := time.Now()
			call.AnsweredAt = &now
		}

	case "CHANNEL_HANGUP_COMPLETE":
		if call, ok := p.calls[callID]; ok {
			call.Status = p.parseHangupCause(event.Headers["Hangup-Cause"])
			now := time.Now()
			call.EndedAt = &now
			if call.AnsweredAt != nil {
				call.Duration = int(now.Sub(*call.AnsweredAt).Seconds())
			}
		}
	}
}

// parseDirection parses FreeSWITCH call direction
func (p *FreeSWITCHProvider) parseDirection(dir string) string {
	if dir == "inbound" {
		return "inbound"
	}
	return "outbound"
}

// parseHangupCause converts FreeSWITCH hangup cause to call status
func (p *FreeSWITCHProvider) parseHangupCause(cause string) string {
	switch cause {
	case "NORMAL_CLEARING":
		return CallStatusCompleted
	case "USER_BUSY":
		return CallStatusBusy
	case "NO_ANSWER":
		return CallStatusNoAnswer
	case "CALL_REJECTED", "USER_NOT_REGISTERED":
		return CallStatusFailed
	default:
		return CallStatusCompleted
	}
}

// Capabilities returns FreeSWITCH capabilities
func (p *FreeSWITCHProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Outbound:      true,
		Inbound:       true,
		Recording:     true,
		Transcription: false, // Requires external service
		IVR:           true,
		Conferencing:  true,
		Queueing:      true,
		WebRTC:        true,
		SIP:           true,
	}
}

// MakeCall initiates an outbound call via FreeSWITCH
func (p *FreeSWITCHProvider) MakeCall(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	// Build originate command
	// Format: originate {variables}sofia/gateway/gw/destination &app(args)
	gateway := p.config.Credentials["gateway"]
	if gateway == "" {
		gateway = "default"
	}

	// Build channel variables
	vars := []string{
		fmt.Sprintf("origination_caller_id_number=%s", input.From),
		fmt.Sprintf("origination_caller_id_name=%s", input.From),
	}

	if input.CallbackURL != "" {
		vars = append(vars, fmt.Sprintf("linktor_callback_url=%s", input.CallbackURL))
	}

	if input.StatusURL != "" {
		vars = append(vars, fmt.Sprintf("linktor_status_url=%s", input.StatusURL))
	}

	if input.Record {
		vars = append(vars, "RECORD_STEREO=true")
		vars = append(vars, "RECORD_MIN_SEC=1")
	}

	// Custom variables
	for k, v := range input.CustomData {
		vars = append(vars, fmt.Sprintf("linktor_%s=%s", k, v))
	}

	// Build dial string
	dialString := fmt.Sprintf("{%s}sofia/gateway/%s/%s",
		strings.Join(vars, ","),
		gateway,
		input.To,
	)

	// Application to run after answer
	app := "park" // Default: park the call
	if input.CallbackURL != "" {
		app = fmt.Sprintf("socket('%s async full')", p.buildSocketURL(input.CallbackURL))
	}

	// Send originate command via ESL
	cmd := fmt.Sprintf("bgapi originate %s &%s", dialString, app)
	if err := p.eslSendCommand(cmd); err != nil {
		return nil, fmt.Errorf("failed to originate call: %w", err)
	}

	// Read response
	event, err := p.eslReadEvent()
	if err != nil {
		return nil, fmt.Errorf("failed to read originate response: %w", err)
	}

	// Parse response
	// Format: +OK Job-UUID: xxxxx or -ERR message
	replyText := event.Headers["Reply-Text"]
	if strings.HasPrefix(replyText, "-ERR") {
		return nil, fmt.Errorf("originate failed: %s", replyText)
	}

	// Extract call ID from body
	callID := strings.TrimSpace(event.Body)
	if callID == "" {
		// Generate a temporary ID
		callID = fmt.Sprintf("fs_%d", time.Now().UnixNano())
	}

	return &MakeCallResult{
		CallID:     callID,
		ExternalID: callID,
		Status:     CallStatusInitiated,
	}, nil
}

// buildSocketURL builds the outbound socket URL
func (p *FreeSWITCHProvider) buildSocketURL(callbackURL string) string {
	// Convert HTTP callback URL to socket format
	// This assumes you have an ESL outbound socket handler
	u, err := url.Parse(callbackURL)
	if err != nil {
		return callbackURL
	}

	// Use the socket port from config or default
	socketPort := p.config.Credentials["socket_port"]
	if socketPort == "" {
		socketPort = "8085"
	}

	return fmt.Sprintf("%s:%s", u.Hostname(), socketPort)
}

// GetCall retrieves call details
func (p *FreeSWITCHProvider) GetCall(ctx context.Context, callID string) (*Call, error) {
	// First check local cache
	p.callsMutex.RLock()
	if call, ok := p.calls[callID]; ok {
		p.callsMutex.RUnlock()
		return call, nil
	}
	p.callsMutex.RUnlock()

	// Query FreeSWITCH for call info
	cmd := fmt.Sprintf("api uuid_getvar %s", callID)
	if err := p.eslSendCommand(cmd); err != nil {
		return nil, fmt.Errorf("failed to get call: %w", err)
	}

	event, err := p.eslReadEvent()
	if err != nil {
		return nil, fmt.Errorf("failed to read call info: %w", err)
	}

	if strings.HasPrefix(event.Body, "-ERR") {
		return nil, fmt.Errorf("call not found: %s", callID)
	}

	// Parse call info from response
	call := &Call{
		ID:         callID,
		ExternalID: callID,
		Status:     CallStatusInProgress,
	}

	return call, nil
}

// EndCall terminates an active call
func (p *FreeSWITCHProvider) EndCall(ctx context.Context, callID string) error {
	cmd := fmt.Sprintf("api uuid_kill %s NORMAL_CLEARING", callID)
	if err := p.eslSendCommand(cmd); err != nil {
		return fmt.Errorf("failed to end call: %w", err)
	}

	event, err := p.eslReadEvent()
	if err != nil {
		return fmt.Errorf("failed to read end call response: %w", err)
	}

	if strings.HasPrefix(event.Body, "-ERR") {
		return fmt.Errorf("failed to end call: %s", event.Body)
	}

	return nil
}

// TransferCall transfers a call to another destination
func (p *FreeSWITCHProvider) TransferCall(ctx context.Context, callID, destination string) error {
	// Transfer using uuid_transfer
	// Format: uuid_transfer <uuid> <destination> [<dialplan>] [<context>]
	cmd := fmt.Sprintf("api uuid_transfer %s %s XML default", callID, destination)
	if err := p.eslSendCommand(cmd); err != nil {
		return fmt.Errorf("failed to transfer call: %w", err)
	}

	event, err := p.eslReadEvent()
	if err != nil {
		return fmt.Errorf("failed to read transfer response: %w", err)
	}

	if strings.HasPrefix(event.Body, "-ERR") {
		return fmt.Errorf("transfer failed: %s", event.Body)
	}

	return nil
}

// GetRecording retrieves a call recording
func (p *FreeSWITCHProvider) GetRecording(ctx context.Context, recordingID string) (*Recording, error) {
	// FreeSWITCH recordings are stored as files
	// The recordingID is typically the file path
	recordingPath := recordingID

	// Build recording URL
	webURL := p.config.Credentials["recordings_url"]
	if webURL == "" {
		webURL = "http://localhost/recordings"
	}

	recording := &Recording{
		ID:        recordingID,
		CallID:    extractCallIDFromPath(recordingPath),
		URL:       fmt.Sprintf("%s/%s", webURL, recordingPath),
		Status:    "completed",
		CreatedAt: time.Now(),
	}

	return recording, nil
}

// extractCallIDFromPath extracts call ID from recording path
func extractCallIDFromPath(path string) string {
	// Typical format: /var/lib/freeswitch/recordings/{call-id}.wav
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		return strings.TrimSuffix(filename, ".wav")
	}
	return ""
}

// DeleteRecording deletes a call recording
func (p *FreeSWITCHProvider) DeleteRecording(ctx context.Context, recordingID string) error {
	// Use FreeSWITCH API to delete file
	cmd := fmt.Sprintf("api system rm %s", recordingID)
	if err := p.eslSendCommand(cmd); err != nil {
		return fmt.Errorf("failed to delete recording: %w", err)
	}

	return nil
}

// GenerateIVRResponse generates FreeSWITCH dialplan actions from IVR actions
func (p *FreeSWITCHProvider) GenerateIVRResponse(actions []IVRAction) (interface{}, error) {
	var fsActions []FSAction

	for _, action := range actions {
		switch a := action.(type) {
		case IVRSay:
			// Use mod_flite, mod_tts_commandline, or phrase macros
			voice := p.mapVoice(a.Voice, a.Language)
			fsActions = append(fsActions, FSAction{
				Application: "speak",
				Data:        fmt.Sprintf("%s|%s|%s", voice, a.Language, a.Text),
			})

		case IVRPlay:
			fsActions = append(fsActions, FSAction{
				Application: "playback",
				Data:        a.URL,
			})
			// Handle loop
			if a.Loop > 1 {
				fsActions[len(fsActions)-1].Data = fmt.Sprintf("file_string://%s!%s",
					a.URL, strings.Repeat(","+a.URL, a.Loop-1))
			}

		case IVRGather:
			// FreeSWITCH gather is done with play_and_get_digits
			// Format: <min> <max> <tries> <timeout> <terminators> <file> <invalid_file> <var_name> <regexp> <digit_timeout>
			minDigits := 1
			maxDigits := a.NumDigits
			if maxDigits == 0 {
				maxDigits = 20
			}
			timeout := a.Timeout
			if timeout == 0 {
				timeout = 5
			}
			finishOnKey := a.FinishOnKey
			if finishOnKey == "" {
				finishOnKey = "#"
			}

			// First, handle nested speak/play actions
			for _, nested := range a.Nested {
				switch n := nested.(type) {
				case IVRSay:
					voice := p.mapVoice(n.Voice, n.Language)
					fsActions = append(fsActions, FSAction{
						Application: "speak",
						Data:        fmt.Sprintf("%s|%s|%s", voice, n.Language, n.Text),
					})
				case IVRPlay:
					fsActions = append(fsActions, FSAction{
						Application: "playback",
						Data:        n.URL,
					})
				}
			}

			// Add read digits
			fsActions = append(fsActions, FSAction{
				Application: "read",
				Data: fmt.Sprintf("%d %d dtmf_digits %d %s",
					minDigits, maxDigits, timeout*1000, finishOnKey),
			})

			// For speech recognition
			if contains(a.Input, "speech") {
				fsActions = append(fsActions, FSAction{
					Application: "detect_speech",
					Data:        fmt.Sprintf("pocketsphinx %s", a.Language),
				})
			}

		case IVRRecord:
			// Record to file
			filename := fmt.Sprintf("${uuid}.wav")
			maxLength := a.MaxLength
			if maxLength == 0 {
				maxLength = 300
			}
			silenceTimeout := 5
			silenceThresh := 200
			fsActions = append(fsActions, FSAction{
				Application: "record",
				Data: fmt.Sprintf("%s %d %d %d",
					filename, maxLength, silenceThresh, silenceTimeout),
			})

		case IVRDial:
			// Bridge to destination
			dialString := p.buildDialString(a)
			fsActions = append(fsActions, FSAction{
				Application: "bridge",
				Data:        dialString,
			})

		case IVRConference:
			// Join conference
			confOptions := []string{}
			if a.Muted {
				confOptions = append(confOptions, "mute")
			}
			if a.StartOnEnter {
				confOptions = append(confOptions, "moderator")
			}
			if !a.EndOnExit {
				confOptions = append(confOptions, "endconf")
			}
			profile := "default"
			if len(confOptions) > 0 {
				profile = fmt.Sprintf("default+flags{%s}", strings.Join(confOptions, "|"))
			}
			fsActions = append(fsActions, FSAction{
				Application: "conference",
				Data:        fmt.Sprintf("%s@%s", a.Name, profile),
			})

		case IVRQueue:
			// Add to call queue using mod_callcenter
			fsActions = append(fsActions, FSAction{
				Application: "callcenter",
				Data:        a.Name,
			})

		case IVRRedirect:
			// Transfer to new extension
			fsActions = append(fsActions, FSAction{
				Application: "transfer",
				Data:        a.URL,
			})

		case IVRPause:
			// Sleep/pause
			fsActions = append(fsActions, FSAction{
				Application: "sleep",
				Data:        fmt.Sprintf("%d", a.Length*1000),
			})

		case IVRHangup:
			fsActions = append(fsActions, FSAction{
				Application: "hangup",
				Data:        "NORMAL_CLEARING",
			})
		}
	}

	return fsActions, nil
}

// mapVoice maps generic voice names to FreeSWITCH TTS voices
func (p *FreeSWITCHProvider) mapVoice(voice, language string) string {
	// FreeSWITCH TTS engine name
	// Depends on installed module (flite, tts_commandline, etc.)
	ttsEngine := p.config.Credentials["tts_engine"]
	if ttsEngine == "" {
		ttsEngine = "flite"
	}
	return ttsEngine
}

// buildDialString builds a FreeSWITCH dial string
func (p *FreeSWITCHProvider) buildDialString(dial IVRDial) string {
	gateway := p.config.Credentials["gateway"]
	if gateway == "" {
		gateway = "default"
	}

	vars := []string{}
	if dial.CallerID != "" {
		vars = append(vars, fmt.Sprintf("origination_caller_id_number=%s", dial.CallerID))
	}
	if dial.Timeout > 0 {
		vars = append(vars, fmt.Sprintf("call_timeout=%d", dial.Timeout))
	}
	if dial.Record {
		vars = append(vars, "RECORD_STEREO=true")
	}

	varString := ""
	if len(vars) > 0 {
		varString = fmt.Sprintf("{%s}", strings.Join(vars, ","))
	}

	// Build destinations
	destinations := []string{}
	for _, dest := range dial.Numbers {
		destinations = append(destinations, fmt.Sprintf("sofia/gateway/%s/%s", gateway, dest))
	}

	// For SIP URIs
	for _, sip := range dial.SIPURIs {
		destinations = append(destinations, fmt.Sprintf("sofia/external/%s", sip))
	}

	return fmt.Sprintf("%s%s", varString, strings.Join(destinations, ","))
}

// ParseWebhook parses an incoming webhook from FreeSWITCH
func (p *FreeSWITCHProvider) ParseWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error) {
	// FreeSWITCH can send webhooks via mod_http_cache or mod_curl
	// Parse JSON or URL-encoded body

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Try URL-encoded
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("failed to parse webhook body: %w", err)
		}
		data = make(map[string]interface{})
		for k, v := range values {
			if len(v) > 0 {
				data[k] = v[0]
			}
		}
	}

	event := &WebhookEvent{
		Raw: data,
	}

	// Parse common fields
	if eventName, ok := data["Event-Name"].(string); ok {
		event.Type = p.mapEventType(eventName)
	}

	if callID, ok := data["Unique-ID"].(string); ok {
		event.CallID = callID
	}

	if from, ok := data["Caller-Caller-ID-Number"].(string); ok {
		event.From = from
	}

	if to, ok := data["Caller-Destination-Number"].(string); ok {
		event.To = to
	}

	if digits, ok := data["DTMF-Digit"].(string); ok {
		event.Digits = digits
	}

	if speech, ok := data["Detected-Speech"].(string); ok {
		event.SpeechResult = speech
	}

	if hangupCause, ok := data["Hangup-Cause"].(string); ok {
		event.Status = p.parseHangupCause(hangupCause)
	}

	if recordPath, ok := data["Record-File-Path"].(string); ok {
		event.RecordingURL = recordPath
	}

	return event, nil
}

// mapEventType maps FreeSWITCH event names to webhook event types
func (p *FreeSWITCHProvider) mapEventType(eventName string) string {
	switch eventName {
	case "CHANNEL_CREATE":
		return WebhookCallInitiated
	case "CHANNEL_ANSWER":
		return WebhookCallAnswered
	case "CHANNEL_HANGUP", "CHANNEL_HANGUP_COMPLETE":
		return WebhookCallCompleted
	case "DTMF":
		return WebhookDTMFReceived
	case "DETECTED_SPEECH":
		return WebhookSpeechReceived
	case "RECORD_START":
		return WebhookRecordingStarted
	case "RECORD_STOP":
		return WebhookRecordingCompleted
	default:
		return eventName
	}
}

// ValidateWebhook validates an incoming webhook from FreeSWITCH
func (p *FreeSWITCHProvider) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) bool {
	// FreeSWITCH doesn't have built-in webhook signing
	// Implement custom validation using a shared secret

	secret := p.config.Credentials["webhook_secret"]
	if secret == "" {
		// No secret configured, skip validation
		return true
	}

	// Check for custom signature header
	signature := headers["X-FreeSWITCH-Signature"]
	if signature == "" {
		signature = headers["x-freeswitch-signature"]
	}

	if signature == "" {
		return false
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Helper function to check if a slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// Close closes the ESL connection
func (p *FreeSWITCHProvider) Close() error {
	if p.eslConn != nil {
		return p.eslConn.Close()
	}
	return nil
}

// SendESLCommand sends a raw ESL command (for advanced use)
func (p *FreeSWITCHProvider) SendESLCommand(cmd string) (string, error) {
	if err := p.eslSendCommand(fmt.Sprintf("api %s", cmd)); err != nil {
		return "", err
	}

	event, err := p.eslReadEvent()
	if err != nil {
		return "", err
	}

	return event.Body, nil
}

// SetVariable sets a channel variable
func (p *FreeSWITCHProvider) SetVariable(callID, name, value string) error {
	cmd := fmt.Sprintf("api uuid_setvar %s %s %s", callID, name, value)
	_, err := p.SendESLCommand(cmd)
	return err
}

// GetVariable gets a channel variable
func (p *FreeSWITCHProvider) GetVariable(callID, name string) (string, error) {
	cmd := fmt.Sprintf("uuid_getvar %s %s", callID, name)
	return p.SendESLCommand(cmd)
}

// PlayFile plays a sound file on a call
func (p *FreeSWITCHProvider) PlayFile(callID, file string) error {
	cmd := fmt.Sprintf("api uuid_broadcast %s %s both", callID, file)
	_, err := p.SendESLCommand(cmd)
	return err
}

// StartRecording starts recording a call
func (p *FreeSWITCHProvider) StartRecording(callID, path string) error {
	cmd := fmt.Sprintf("api uuid_record %s start %s", callID, path)
	_, err := p.SendESLCommand(cmd)
	return err
}

// StopRecording stops recording a call
func (p *FreeSWITCHProvider) StopRecording(callID, path string) error {
	cmd := fmt.Sprintf("api uuid_record %s stop %s", callID, path)
	_, err := p.SendESLCommand(cmd)
	return err
}

// JoinConference joins a call to a conference
func (p *FreeSWITCHProvider) JoinConference(callID, confName string, muted bool) error {
	flags := ""
	if muted {
		flags = "+flags{mute}"
	}
	cmd := fmt.Sprintf("api uuid_transfer %s conference:%s@default%s inline", callID, confName, flags)
	_, err := p.SendESLCommand(cmd)
	return err
}

// SendDTMF sends DTMF tones to a call
func (p *FreeSWITCHProvider) SendDTMF(callID, digits string) error {
	cmd := fmt.Sprintf("api uuid_send_dtmf %s %s", callID, digits)
	_, err := p.SendESLCommand(cmd)
	return err
}
