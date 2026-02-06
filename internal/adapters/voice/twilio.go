package voice

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TwilioProvider implements the Provider interface for Twilio Voice
type TwilioProvider struct {
	accountSID  string
	authToken   string
	phoneNumber string
	baseURL     string
	httpClient  *http.Client
}

// NewTwilioProvider creates a new Twilio Voice provider
func NewTwilioProvider() *TwilioProvider {
	return &TwilioProvider{
		baseURL:    "https://api.twilio.com/2010-04-01",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *TwilioProvider) Name() string {
	return "twilio"
}

// Initialize sets up the provider with configuration
func (p *TwilioProvider) Initialize(ctx context.Context, config VoiceConfig) error {
	accountSID, ok := config.Credentials["account_sid"]
	if !ok || accountSID == "" {
		return fmt.Errorf("twilio account_sid is required")
	}

	authToken, ok := config.Credentials["auth_token"]
	if !ok || authToken == "" {
		return fmt.Errorf("twilio auth_token is required")
	}

	p.accountSID = accountSID
	p.authToken = authToken
	p.phoneNumber = config.PhoneNumber

	return nil
}

// Capabilities returns what this provider supports
func (p *TwilioProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		OutboundCalls:     true,
		InboundCalls:      true,
		Recording:         true,
		Transcription:     true,
		TextToSpeech:      true,
		SpeechRecognition: true,
		DTMF:              true,
		Conferencing:      true,
		CallQueues:        true,
		SIP:               true,
		WebRTC:            true,
	}
}

// MakeCall initiates an outbound call
func (p *TwilioProvider) MakeCall(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	endpoint := fmt.Sprintf("%s/Accounts/%s/Calls.json", p.baseURL, p.accountSID)

	data := url.Values{}
	data.Set("To", input.To)
	data.Set("From", input.From)

	if input.CallbackURL != "" {
		data.Set("Url", input.CallbackURL)
	}
	if input.StatusURL != "" {
		data.Set("StatusCallback", input.StatusURL)
		data.Set("StatusCallbackEvent", "initiated ringing answered completed")
	}
	if input.TwiML != "" {
		data.Set("Twiml", input.TwiML)
	}
	if input.Record {
		data.Set("Record", "true")
	}
	if input.Timeout > 0 {
		data.Set("Timeout", strconv.Itoa(input.Timeout))
	}
	if input.MachineDetection {
		data.Set("MachineDetection", "Enable")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("twilio error %d: %s", errResp.Code, errResp.Message)
	}

	var callResp struct {
		SID    string `json:"sid"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&callResp); err != nil {
		return nil, err
	}

	return &MakeCallResult{
		CallID:     callResp.SID,
		ExternalID: callResp.SID,
		Status:     p.mapStatus(callResp.Status),
	}, nil
}

// GetCall retrieves call details
func (p *TwilioProvider) GetCall(ctx context.Context, callID string) (*Call, error) {
	endpoint := fmt.Sprintf("%s/Accounts/%s/Calls/%s.json", p.baseURL, p.accountSID, callID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.accountSID, p.authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("call not found: %s", callID)
	}

	var callResp struct {
		SID         string `json:"sid"`
		Status      string `json:"status"`
		Direction   string `json:"direction"`
		From        string `json:"from"`
		To          string `json:"to"`
		Duration    string `json:"duration"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
		DateCreated string `json:"date_created"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&callResp); err != nil {
		return nil, err
	}

	duration, _ := strconv.Atoi(callResp.Duration)

	call := &Call{
		ID:         callResp.SID,
		ExternalID: callResp.SID,
		Status:     p.mapStatus(callResp.Status),
		From:       callResp.From,
		To:         callResp.To,
		Duration:   duration,
	}

	if callResp.Direction == "inbound" {
		call.Direction = CallDirectionInbound
	} else {
		call.Direction = CallDirectionOutbound
	}

	if callResp.StartTime != "" {
		if t, err := time.Parse(time.RFC1123Z, callResp.StartTime); err == nil {
			call.StartedAt = t
		}
	}

	if callResp.EndTime != "" {
		if t, err := time.Parse(time.RFC1123Z, callResp.EndTime); err == nil {
			call.EndedAt = &t
		}
	}

	if callResp.DateCreated != "" {
		if t, err := time.Parse(time.RFC1123Z, callResp.DateCreated); err == nil {
			call.CreatedAt = t
		}
	}

	return call, nil
}

// EndCall terminates an active call
func (p *TwilioProvider) EndCall(ctx context.Context, callID string) error {
	endpoint := fmt.Sprintf("%s/Accounts/%s/Calls/%s.json", p.baseURL, p.accountSID, callID)

	data := url.Values{}
	data.Set("Status", "completed")

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
func (p *TwilioProvider) TransferCall(ctx context.Context, callID, destination string) error {
	endpoint := fmt.Sprintf("%s/Accounts/%s/Calls/%s.json", p.baseURL, p.accountSID, callID)

	// Generate TwiML for transfer
	twiml := fmt.Sprintf(`<Response><Dial>%s</Dial></Response>`, destination)

	data := url.Values{}
	data.Set("Twiml", twiml)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
func (p *TwilioProvider) GetRecording(ctx context.Context, recordingID string) (*Recording, error) {
	endpoint := fmt.Sprintf("%s/Accounts/%s/Recordings/%s.json", p.baseURL, p.accountSID, recordingID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.accountSID, p.authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("recording not found: %s", recordingID)
	}

	var recResp struct {
		SID         string `json:"sid"`
		CallSID     string `json:"call_sid"`
		Duration    string `json:"duration"`
		DateCreated string `json:"date_created"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&recResp); err != nil {
		return nil, err
	}

	duration, _ := strconv.Atoi(recResp.Duration)

	recording := &Recording{
		ID:       recResp.SID,
		CallID:   recResp.CallSID,
		URL:      fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Recordings/%s.mp3", p.accountSID, recResp.SID),
		Duration: duration,
		Format:   "mp3",
	}

	if recResp.DateCreated != "" {
		if t, err := time.Parse(time.RFC1123Z, recResp.DateCreated); err == nil {
			recording.CreatedAt = t
		}
	}

	return recording, nil
}

// DeleteRecording deletes a call recording
func (p *TwilioProvider) DeleteRecording(ctx context.Context, recordingID string) error {
	endpoint := fmt.Sprintf("%s/Accounts/%s/Recordings/%s.json", p.baseURL, p.accountSID, recordingID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(p.accountSID, p.authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to delete recording: status %d", resp.StatusCode)
	}

	return nil
}

// GenerateIVRResponse generates TwiML response
func (p *TwilioProvider) GenerateIVRResponse(actions []IVRAction) (interface{}, error) {
	var twiml strings.Builder
	twiml.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Response>\n")

	for _, action := range actions {
		switch a := action.(type) {
		case IVRSay:
			twiml.WriteString(fmt.Sprintf("  <Say language=\"%s\"", a.Language))
			if a.Voice != "" {
				twiml.WriteString(fmt.Sprintf(" voice=\"%s\"", a.Voice))
			}
			if a.Loop > 0 {
				twiml.WriteString(fmt.Sprintf(" loop=\"%d\"", a.Loop))
			}
			twiml.WriteString(fmt.Sprintf(">%s</Say>\n", a.Text))

		case IVRPlay:
			twiml.WriteString("  <Play")
			if a.Loop > 0 {
				twiml.WriteString(fmt.Sprintf(" loop=\"%d\"", a.Loop))
			}
			if a.Digits != "" {
				twiml.WriteString(fmt.Sprintf(" digits=\"%s\"", a.Digits))
			}
			twiml.WriteString(fmt.Sprintf(">%s</Play>\n", a.URL))

		case IVRGather:
			twiml.WriteString("  <Gather")
			if len(a.Input) > 0 {
				twiml.WriteString(fmt.Sprintf(" input=\"%s\"", strings.Join(a.Input, " ")))
			}
			if a.Timeout > 0 {
				twiml.WriteString(fmt.Sprintf(" timeout=\"%d\"", a.Timeout))
			}
			if a.NumDigits > 0 {
				twiml.WriteString(fmt.Sprintf(" numDigits=\"%d\"", a.NumDigits))
			}
			if a.FinishOnKey != "" {
				twiml.WriteString(fmt.Sprintf(" finishOnKey=\"%s\"", a.FinishOnKey))
			}
			if a.ActionURL != "" {
				twiml.WriteString(fmt.Sprintf(" action=\"%s\"", a.ActionURL))
			}
			if a.Language != "" {
				twiml.WriteString(fmt.Sprintf(" language=\"%s\"", a.Language))
			}
			if len(a.Hints) > 0 {
				twiml.WriteString(fmt.Sprintf(" hints=\"%s\"", strings.Join(a.Hints, ",")))
			}
			twiml.WriteString(">\n")

			// Nested actions
			for _, nested := range a.Nested {
				if say, ok := nested.(IVRSay); ok {
					twiml.WriteString(fmt.Sprintf("    <Say language=\"%s\">%s</Say>\n", say.Language, say.Text))
				} else if play, ok := nested.(IVRPlay); ok {
					twiml.WriteString(fmt.Sprintf("    <Play>%s</Play>\n", play.URL))
				}
			}

			twiml.WriteString("  </Gather>\n")

		case IVRRecord:
			twiml.WriteString("  <Record")
			if a.MaxLength > 0 {
				twiml.WriteString(fmt.Sprintf(" maxLength=\"%d\"", a.MaxLength))
			}
			if a.Timeout > 0 {
				twiml.WriteString(fmt.Sprintf(" timeout=\"%d\"", a.Timeout))
			}
			if a.FinishOnKey != "" {
				twiml.WriteString(fmt.Sprintf(" finishOnKey=\"%s\"", a.FinishOnKey))
			}
			if a.Transcribe {
				twiml.WriteString(" transcribe=\"true\"")
			}
			if a.PlayBeep {
				twiml.WriteString(" playBeep=\"true\"")
			}
			if a.ActionURL != "" {
				twiml.WriteString(fmt.Sprintf(" action=\"%s\"", a.ActionURL))
			}
			twiml.WriteString("/>\n")

		case IVRDial:
			twiml.WriteString("  <Dial")
			if a.Timeout > 0 {
				twiml.WriteString(fmt.Sprintf(" timeout=\"%d\"", a.Timeout))
			}
			if a.CallerID != "" {
				twiml.WriteString(fmt.Sprintf(" callerId=\"%s\"", a.CallerID))
			}
			if a.Record {
				twiml.WriteString(" record=\"record-from-answer\"")
			}
			if a.ActionURL != "" {
				twiml.WriteString(fmt.Sprintf(" action=\"%s\"", a.ActionURL))
			}
			twiml.WriteString(">")
			if a.Number != "" {
				twiml.WriteString(a.Number)
			} else if a.SIPEndpoint != "" {
				twiml.WriteString(fmt.Sprintf("<Sip>%s</Sip>", a.SIPEndpoint))
			} else if a.Queue != "" {
				twiml.WriteString(fmt.Sprintf("<Queue>%s</Queue>", a.Queue))
			}
			twiml.WriteString("</Dial>\n")

		case IVRHangup:
			twiml.WriteString("  <Hangup/>\n")

		case IVRPause:
			twiml.WriteString(fmt.Sprintf("  <Pause length=\"%d\"/>\n", a.Length))

		case IVRRedirect:
			twiml.WriteString(fmt.Sprintf("  <Redirect method=\"%s\">%s</Redirect>\n", a.Method, a.URL))

		case IVRQueue:
			twiml.WriteString("  <Enqueue")
			if a.WaitURL != "" {
				twiml.WriteString(fmt.Sprintf(" waitUrl=\"%s\"", a.WaitURL))
			}
			if a.ActionURL != "" {
				twiml.WriteString(fmt.Sprintf(" action=\"%s\"", a.ActionURL))
			}
			twiml.WriteString(fmt.Sprintf(">%s</Enqueue>\n", a.Name))

		case IVRConference:
			twiml.WriteString("  <Dial><Conference")
			if a.Muted {
				twiml.WriteString(" muted=\"true\"")
			}
			if a.StartOnEnter {
				twiml.WriteString(" startConferenceOnEnter=\"true\"")
			}
			if a.EndOnExit {
				twiml.WriteString(" endConferenceOnExit=\"true\"")
			}
			if a.WaitURL != "" {
				twiml.WriteString(fmt.Sprintf(" waitUrl=\"%s\"", a.WaitURL))
			}
			if a.MaxParticipants > 0 {
				twiml.WriteString(fmt.Sprintf(" maxParticipants=\"%d\"", a.MaxParticipants))
			}
			if a.Record {
				twiml.WriteString(" record=\"record-from-start\"")
			}
			twiml.WriteString(fmt.Sprintf(">%s</Conference></Dial>\n", a.Name))
		}
	}

	twiml.WriteString("</Response>")
	return twiml.String(), nil
}

// ParseWebhook parses an incoming Twilio webhook
func (p *TwilioProvider) ParseWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error) {
	// Parse form data
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	event := &WebhookEvent{
		CallID:     values.Get("CallSid"),
		ExternalID: values.Get("CallSid"),
		From:       values.Get("From"),
		To:         values.Get("To"),
		Timestamp:  time.Now(),
		RawPayload: make(map[string]interface{}),
	}

	// Copy all values to raw payload
	for k, v := range values {
		if len(v) > 0 {
			event.RawPayload[k] = v[0]
		}
	}

	// Determine event type and status
	callStatus := values.Get("CallStatus")
	event.Status = p.mapStatus(callStatus)

	// Check for DTMF digits
	if digits := values.Get("Digits"); digits != "" {
		event.Type = "dtmf"
		event.Digits = digits
	} else if speechResult := values.Get("SpeechResult"); speechResult != "" {
		event.Type = "speech"
		event.SpeechResult = speechResult
	} else if recordingURL := values.Get("RecordingUrl"); recordingURL != "" {
		event.Type = "recording"
		event.RecordingURL = recordingURL
	} else if transcription := values.Get("TranscriptionText"); transcription != "" {
		event.Type = "transcription"
		event.Transcription = transcription
	} else {
		event.Type = "status"
	}

	// Direction
	direction := values.Get("Direction")
	if direction == "inbound" {
		event.Direction = CallDirectionInbound
	} else {
		event.Direction = CallDirectionOutbound
	}

	// Duration
	if duration := values.Get("CallDuration"); duration != "" {
		if d, err := strconv.Atoi(duration); err == nil {
			event.Duration = d
		}
	}

	return event, nil
}

// ValidateWebhook validates Twilio webhook signature
func (p *TwilioProvider) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) bool {
	signature := headers["X-Twilio-Signature"]
	if signature == "" {
		signature = headers["x-twilio-signature"]
	}

	if signature == "" {
		return false
	}

	// Get the full URL from headers
	requestURL := headers["X-Forwarded-Proto"] + "://" + headers["Host"] + headers["X-Original-URI"]
	if requestURL == "://" {
		// Fallback if headers not available
		return true // Skip validation in development
	}

	// Parse body parameters
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return false
	}

	// Build the validation string
	validationString := requestURL

	// Sort parameter names and append to validation string
	var keys []string
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		validationString += k + values.Get(k)
	}

	// Compute HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(p.authToken))
	mac.Write([]byte(validationString))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// mapStatus maps Twilio status to CallStatus
func (p *TwilioProvider) mapStatus(status string) CallStatus {
	switch strings.ToLower(status) {
	case "queued", "initiated":
		return CallStatusInitiated
	case "ringing":
		return CallStatusRinging
	case "in-progress":
		return CallStatusInProgress
	case "completed":
		return CallStatusCompleted
	case "busy":
		return CallStatusBusy
	case "no-answer":
		return CallStatusNoAnswer
	case "failed":
		return CallStatusFailed
	case "canceled":
		return CallStatusCanceled
	default:
		return CallStatus(status)
	}
}
