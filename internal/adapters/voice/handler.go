package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPHandler provides HTTP handlers for voice webhooks
type HTTPHandler struct {
	adapter *Adapter
}

// NewHTTPHandler creates a new HTTP handler for voice webhooks
func NewHTTPHandler(adapter *Adapter) *HTTPHandler {
	return &HTTPHandler{
		adapter: adapter,
	}
}

// ServeHTTP implements http.Handler for voice webhooks
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Extract headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
			// Also store lowercase version
			headers[strings.ToLower(key)] = values[0]
		}
	}

	// Process webhook
	ctx := r.Context()
	response, err := h.adapter.HandleWebhook(ctx, headers, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response based on provider
	h.writeResponse(w, response)
}

// writeResponse writes the IVR response in provider-specific format
func (h *HTTPHandler) writeResponse(w http.ResponseWriter, response interface{}) {
	if response == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch h.adapter.Provider() {
	case "twilio":
		// TwiML response
		w.Header().Set("Content-Type", "application/xml")
		if twiml, ok := response.(string); ok {
			w.Write([]byte(twiml))
		}

	case "vonage":
		// NCCO JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case "amazon_connect":
		// Amazon Connect contact flow actions
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case "asterisk":
		// AGI response or ARI JSON
		if agiResp, ok := response.(string); ok {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(agiResp))
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}

	case "freeswitch":
		// FreeSWITCH dialplan actions
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		// Default JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// HandleStatusCallback handles call status webhooks
func (h *HTTPHandler) HandleStatusCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
			headers[strings.ToLower(key)] = values[0]
		}
	}

	// Validate webhook
	if !h.adapter.provider.ValidateWebhook(r.Context(), headers, body) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the webhook event
	event, err := h.adapter.provider.ParseWebhook(r.Context(), headers, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If there's a handler, call it but don't expect IVR actions
	if h.adapter.handler != nil {
		h.adapter.handler(r.Context(), event)
	}

	w.WriteHeader(http.StatusOK)
}

// HandleRecordingCallback handles recording completion webhooks
func (h *HTTPHandler) HandleRecordingCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
			headers[strings.ToLower(key)] = values[0]
		}
	}

	// Validate webhook
	if !h.adapter.provider.ValidateWebhook(r.Context(), headers, body) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the webhook event
	event, err := h.adapter.provider.ParseWebhook(r.Context(), headers, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set recording-specific type if not already set
	if event.Type == "" {
		event.Type = WebhookRecordingCompleted
	}

	// Call handler if set
	if h.adapter.handler != nil {
		h.adapter.handler(r.Context(), event)
	}

	w.WriteHeader(http.StatusOK)
}

// HandleTranscriptionCallback handles transcription completion webhooks
func (h *HTTPHandler) HandleTranscriptionCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
			headers[strings.ToLower(key)] = values[0]
		}
	}

	// Validate webhook
	if !h.adapter.provider.ValidateWebhook(r.Context(), headers, body) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the webhook event
	event, err := h.adapter.provider.ParseWebhook(r.Context(), headers, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set transcription-specific type
	if event.Type == "" {
		event.Type = WebhookTranscriptionCompleted
	}

	// Call handler if set
	if h.adapter.handler != nil {
		h.adapter.handler(r.Context(), event)
	}

	w.WriteHeader(http.StatusOK)
}

// Routes returns a map of routes for easy registration with a router
func (h *HTTPHandler) Routes() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"/voice/webhook":       h.ServeHTTP,
		"/voice/status":        h.HandleStatusCallback,
		"/voice/recording":     h.HandleRecordingCallback,
		"/voice/transcription": h.HandleTranscriptionCallback,
	}
}

// CallFlowHandler manages IVR call flows
type CallFlowHandler struct {
	flows map[string]CallFlow
}

// CallFlow represents an IVR call flow
type CallFlow struct {
	Name        string
	Description string
	Steps       []CallFlowStep
}

// CallFlowStep represents a step in a call flow
type CallFlowStep struct {
	ID        string
	Type      string // say, play, gather, dial, record, conference, queue, hangup
	Config    map[string]interface{}
	NextSteps map[string]string // Maps input to next step ID
}

// NewCallFlowHandler creates a new call flow handler
func NewCallFlowHandler() *CallFlowHandler {
	return &CallFlowHandler{
		flows: make(map[string]CallFlow),
	}
}

// RegisterFlow registers a call flow
func (h *CallFlowHandler) RegisterFlow(id string, flow CallFlow) {
	h.flows[id] = flow
}

// ExecuteFlow executes a call flow step and returns IVR actions
func (h *CallFlowHandler) ExecuteFlow(ctx context.Context, flowID, stepID string, event *WebhookEvent) ([]IVRAction, string, error) {
	flow, ok := h.flows[flowID]
	if !ok {
		return nil, "", fmt.Errorf("flow not found: %s", flowID)
	}

	var step *CallFlowStep
	for i := range flow.Steps {
		if flow.Steps[i].ID == stepID {
			step = &flow.Steps[i]
			break
		}
	}

	if step == nil {
		// If no step specified, use first step
		if len(flow.Steps) > 0 {
			step = &flow.Steps[0]
		} else {
			return nil, "", fmt.Errorf("step not found: %s", stepID)
		}
	}

	actions, err := h.buildStepActions(step)
	if err != nil {
		return nil, "", err
	}

	// Determine next step based on event
	nextStep := h.determineNextStep(step, event)

	return actions, nextStep, nil
}

// buildStepActions converts a flow step to IVR actions
func (h *CallFlowHandler) buildStepActions(step *CallFlowStep) ([]IVRAction, error) {
	var actions []IVRAction

	switch step.Type {
	case "say":
		text, _ := step.Config["text"].(string)
		language, _ := step.Config["language"].(string)
		if language == "" {
			language = "pt-BR"
		}
		voice, _ := step.Config["voice"].(string)
		actions = append(actions, IVRSay{
			Text:     text,
			Language: language,
			Voice:    voice,
		})

	case "play":
		url, _ := step.Config["url"].(string)
		loop := 1
		if l, ok := step.Config["loop"].(float64); ok {
			loop = int(l)
		}
		actions = append(actions, IVRPlay{
			URL:  url,
			Loop: loop,
		})

	case "gather":
		input := []string{"dtmf"}
		if inputs, ok := step.Config["input"].([]interface{}); ok {
			input = make([]string, len(inputs))
			for i, v := range inputs {
				input[i] = v.(string)
			}
		}
		timeout := 5
		if t, ok := step.Config["timeout"].(float64); ok {
			timeout = int(t)
		}
		numDigits := 1
		if n, ok := step.Config["numDigits"].(float64); ok {
			numDigits = int(n)
		}
		finishOnKey, _ := step.Config["finishOnKey"].(string)

		gather := IVRGather{
			Input:       input,
			Timeout:     timeout,
			NumDigits:   numDigits,
			FinishOnKey: finishOnKey,
		}

		// Nested prompts
		if prompts, ok := step.Config["prompts"].([]interface{}); ok {
			for _, p := range prompts {
				if pm, ok := p.(map[string]interface{}); ok {
					if text, ok := pm["text"].(string); ok {
						lang, _ := pm["language"].(string)
						if lang == "" {
							lang = "pt-BR"
						}
						gather.Nested = append(gather.Nested, IVRSay{
							Text:     text,
							Language: lang,
						})
					}
				}
			}
		}

		actions = append(actions, gather)

	case "dial":
		numbers := []string{}
		if nums, ok := step.Config["numbers"].([]interface{}); ok {
			for _, n := range nums {
				numbers = append(numbers, n.(string))
			}
		}
		timeout := 30
		if t, ok := step.Config["timeout"].(float64); ok {
			timeout = int(t)
		}
		callerID, _ := step.Config["callerId"].(string)
		record, _ := step.Config["record"].(bool)

		actions = append(actions, IVRDial{
			Numbers:  numbers,
			Timeout:  timeout,
			CallerID: callerID,
			Record:   record,
		})

	case "record":
		maxLength := 300
		if m, ok := step.Config["maxLength"].(float64); ok {
			maxLength = int(m)
		}
		transcribe, _ := step.Config["transcribe"].(bool)
		playBeep := true
		if b, ok := step.Config["playBeep"].(bool); ok {
			playBeep = b
		}

		actions = append(actions, IVRRecord{
			MaxLength:  maxLength,
			Transcribe: transcribe,
			PlayBeep:   playBeep,
		})

	case "conference":
		name, _ := step.Config["name"].(string)
		muted, _ := step.Config["muted"].(bool)
		startOnEnter := true
		if s, ok := step.Config["startOnEnter"].(bool); ok {
			startOnEnter = s
		}
		endOnExit, _ := step.Config["endOnExit"].(bool)

		actions = append(actions, IVRConference{
			Name:         name,
			Muted:        muted,
			StartOnEnter: startOnEnter,
			EndOnExit:    endOnExit,
		})

	case "queue":
		name, _ := step.Config["name"].(string)
		actions = append(actions, IVRQueue{
			Name: name,
		})

	case "hangup":
		actions = append(actions, IVRHangup{})

	case "pause":
		length := 1
		if l, ok := step.Config["length"].(float64); ok {
			length = int(l)
		}
		actions = append(actions, IVRPause{
			Length: length,
		})
	}

	return actions, nil
}

// determineNextStep determines the next step based on user input
func (h *CallFlowHandler) determineNextStep(step *CallFlowStep, event *WebhookEvent) string {
	if event == nil || len(step.NextSteps) == 0 {
		return ""
	}

	// Check for DTMF input
	if event.Digits != "" {
		if next, ok := step.NextSteps[event.Digits]; ok {
			return next
		}
	}

	// Check for speech input (simplified)
	if event.SpeechResult != "" {
		// Try to match keywords
		speechLower := strings.ToLower(event.SpeechResult)
		for keyword, next := range step.NextSteps {
			if strings.Contains(speechLower, strings.ToLower(keyword)) {
				return next
			}
		}
	}

	// Default next step
	if next, ok := step.NextSteps["default"]; ok {
		return next
	}

	// No input/timeout
	if next, ok := step.NextSteps["timeout"]; ok {
		return next
	}

	return ""
}

// BuildWelcomeFlow creates a standard welcome flow
func BuildWelcomeFlow(name, greeting string, menuOptions map[string]struct {
	Label  string
	NextID string
}) CallFlow {
	// Build menu text
	menuText := greeting + " "
	for digit, option := range menuOptions {
		menuText += fmt.Sprintf("Para %s, pressione %s. ", option.Label, digit)
	}

	// Build next steps map
	nextSteps := make(map[string]string)
	for digit, option := range menuOptions {
		nextSteps[digit] = option.NextID
	}
	nextSteps["timeout"] = "welcome" // Repeat on timeout
	nextSteps["default"] = "invalid" // Invalid input

	flow := CallFlow{
		Name:        name,
		Description: "Welcome and menu flow",
		Steps: []CallFlowStep{
			{
				ID:   "welcome",
				Type: "gather",
				Config: map[string]interface{}{
					"input":     []string{"dtmf", "speech"},
					"timeout":   5,
					"numDigits": 1,
					"prompts": []map[string]interface{}{
						{
							"text":     menuText,
							"language": "pt-BR",
						},
					},
				},
				NextSteps: nextSteps,
			},
			{
				ID:   "invalid",
				Type: "say",
				Config: map[string]interface{}{
					"text":     "Opção inválida. Por favor, tente novamente.",
					"language": "pt-BR",
				},
				NextSteps: map[string]string{
					"default": "welcome",
				},
			},
		},
	}

	return flow
}

// BuildQueueFlow creates a queue flow with hold music
func BuildQueueFlow(queueName, holdMessage, holdMusicURL string) CallFlow {
	return CallFlow{
		Name:        queueName,
		Description: "Queue with hold music",
		Steps: []CallFlowStep{
			{
				ID:   "queue_message",
				Type: "say",
				Config: map[string]interface{}{
					"text":     holdMessage,
					"language": "pt-BR",
				},
				NextSteps: map[string]string{
					"default": "hold_music",
				},
			},
			{
				ID:   "hold_music",
				Type: "play",
				Config: map[string]interface{}{
					"url":  holdMusicURL,
					"loop": 10,
				},
				NextSteps: map[string]string{
					"default": "queue",
				},
			},
			{
				ID:   "queue",
				Type: "queue",
				Config: map[string]interface{}{
					"name": queueName,
				},
			},
		},
	}
}
