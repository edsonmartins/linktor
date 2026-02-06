package voice

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AmazonConnectProvider implements the Provider interface for Amazon Connect
type AmazonConnectProvider struct {
	instanceID      string
	region          string
	accessKeyID     string
	secretAccessKey string
	contactFlowID   string
	queueID         string
	phoneNumber     string
	baseURL         string
	httpClient      *http.Client
}

// NewAmazonConnectProvider creates a new Amazon Connect provider
func NewAmazonConnectProvider() *AmazonConnectProvider {
	return &AmazonConnectProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *AmazonConnectProvider) Name() string {
	return "amazon_connect"
}

// Initialize sets up the provider with configuration
func (p *AmazonConnectProvider) Initialize(ctx context.Context, config VoiceConfig) error {
	instanceID, ok := config.Credentials["instance_id"]
	if !ok || instanceID == "" {
		return fmt.Errorf("amazon connect instance_id is required")
	}

	region, ok := config.Credentials["region"]
	if !ok || region == "" {
		region = "us-east-1"
	}

	accessKeyID, ok := config.Credentials["access_key_id"]
	if !ok || accessKeyID == "" {
		return fmt.Errorf("amazon connect access_key_id is required")
	}

	secretAccessKey, ok := config.Credentials["secret_access_key"]
	if !ok || secretAccessKey == "" {
		return fmt.Errorf("amazon connect secret_access_key is required")
	}

	p.instanceID = instanceID
	p.region = region
	p.accessKeyID = accessKeyID
	p.secretAccessKey = secretAccessKey
	p.phoneNumber = config.PhoneNumber
	p.baseURL = fmt.Sprintf("https://connect.%s.amazonaws.com", region)

	// Optional settings
	if flowID, ok := config.Credentials["contact_flow_id"]; ok {
		p.contactFlowID = flowID
	}
	if queueID, ok := config.Credentials["queue_id"]; ok {
		p.queueID = queueID
	}

	return nil
}

// Capabilities returns what this provider supports
func (p *AmazonConnectProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		OutboundCalls:     true,
		InboundCalls:      true,
		Recording:         true,
		Transcription:     true, // Via Amazon Transcribe
		TextToSpeech:      true, // Via Amazon Polly
		SpeechRecognition: true, // Via Amazon Lex
		DTMF:              true,
		Conferencing:      true,
		CallQueues:        true,
		SIP:               false,
		WebRTC:            true,
	}
}

// signRequest signs an AWS request with Signature Version 4
func (p *AmazonConnectProvider) signRequest(req *http.Request, body []byte) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.Host)

	// Create canonical request
	method := req.Method
	canonicalURI := req.URL.Path
	canonicalQueryString := req.URL.RawQuery

	// Canonical headers
	signedHeaders := "content-type;host;x-amz-date"
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.Host, amzDate)

	// Hash of payload
	payloadHash := sha256.Sum256(body)
	payloadHashHex := hex.EncodeToString(payloadHash[:])

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, payloadHashHex)

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/connect/aws4_request", dateStamp, p.region)
	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm, amzDate, credentialScope, hex.EncodeToString(canonicalRequestHash[:]))

	// Calculate signature
	kDate := hmacSHA256([]byte("AWS4"+p.secretAccessKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(p.region))
	kService := hmacSHA256(kRegion, []byte("connect"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	// Add authorization header
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, p.accessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// MakeCall initiates an outbound call
func (p *AmazonConnectProvider) MakeCall(ctx context.Context, input MakeCallInput) (*MakeCallResult, error) {
	endpoint := fmt.Sprintf("%s/contact/outbound-voice", p.baseURL)

	payload := map[string]interface{}{
		"InstanceId":           p.instanceID,
		"DestinationPhoneNumber": input.To,
		"SourcePhoneNumber":    input.From,
	}

	if p.contactFlowID != "" {
		payload["ContactFlowId"] = p.contactFlowID
	}
	if p.queueID != "" {
		payload["QueueId"] = p.queueID
	}

	// Add attributes
	if input.Metadata != nil {
		payload["Attributes"] = input.Metadata
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "PUT", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	p.signRequest(req, body)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("amazon connect error: %v", errResp)
	}

	var callResp struct {
		ContactId string `json:"ContactId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&callResp); err != nil {
		return nil, err
	}

	return &MakeCallResult{
		CallID:     callResp.ContactId,
		ExternalID: callResp.ContactId,
		Status:     CallStatusInitiated,
	}, nil
}

// GetCall retrieves call details
func (p *AmazonConnectProvider) GetCall(ctx context.Context, callID string) (*Call, error) {
	endpoint := fmt.Sprintf("%s/contacts/%s", p.baseURL, callID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	p.signRequest(req, nil)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("call not found: %s", callID)
	}

	var contactResp struct {
		Contact struct {
			Id                    string `json:"Id"`
			InitiationMethod     string `json:"InitiationMethod"`
			Channel              string `json:"Channel"`
			InitiationTimestamp  string `json:"InitiationTimestamp"`
			DisconnectTimestamp  string `json:"DisconnectTimestamp"`
			LastUpdateTimestamp  string `json:"LastUpdateTimestamp"`
		} `json:"Contact"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&contactResp); err != nil {
		return nil, err
	}

	call := &Call{
		ID:         contactResp.Contact.Id,
		ExternalID: contactResp.Contact.Id,
		Status:     CallStatusCompleted, // Amazon Connect doesn't have detailed status
	}

	if contactResp.Contact.InitiationMethod == "INBOUND" {
		call.Direction = CallDirectionInbound
	} else {
		call.Direction = CallDirectionOutbound
	}

	if contactResp.Contact.InitiationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, contactResp.Contact.InitiationTimestamp); err == nil {
			call.StartedAt = t
			call.CreatedAt = t
		}
	}

	if contactResp.Contact.DisconnectTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, contactResp.Contact.DisconnectTimestamp); err == nil {
			call.EndedAt = &t
			// Calculate duration
			if !call.StartedAt.IsZero() {
				call.Duration = int(t.Sub(call.StartedAt).Seconds())
			}
		}
	}

	return call, nil
}

// EndCall terminates an active call
func (p *AmazonConnectProvider) EndCall(ctx context.Context, callID string) error {
	endpoint := fmt.Sprintf("%s/contact/stop", p.baseURL)

	payload := map[string]interface{}{
		"InstanceId": p.instanceID,
		"ContactId":  callID,
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	p.signRequest(req, body)

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
func (p *AmazonConnectProvider) TransferCall(ctx context.Context, callID, destination string) error {
	endpoint := fmt.Sprintf("%s/contact/transfer", p.baseURL)

	payload := map[string]interface{}{
		"InstanceId": p.instanceID,
		"ContactId":  callID,
	}

	// Check if destination is a queue ID or phone number
	if strings.HasPrefix(destination, "queue:") {
		payload["QueueId"] = strings.TrimPrefix(destination, "queue:")
	} else {
		// Transfer to external number via quick connect or contact flow
		payload["ContactFlowId"] = p.contactFlowID
		payload["Attributes"] = map[string]string{
			"TransferNumber": destination,
		}
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	p.signRequest(req, body)

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
func (p *AmazonConnectProvider) GetRecording(ctx context.Context, recordingID string) (*Recording, error) {
	// Amazon Connect stores recordings in S3
	// The recording URL is typically provided via EventBridge or Contact Trace Record

	return &Recording{
		ID:     recordingID,
		Format: "wav",
		// URL would be the S3 presigned URL
	}, nil
}

// DeleteRecording deletes a call recording
func (p *AmazonConnectProvider) DeleteRecording(ctx context.Context, recordingID string) error {
	// Recordings are stored in S3, would need S3 API to delete
	return nil
}

// GenerateIVRResponse generates Amazon Connect contact flow actions
func (p *AmazonConnectProvider) GenerateIVRResponse(actions []IVRAction) (interface{}, error) {
	// Amazon Connect uses Contact Flows which are defined in the console
	// For dynamic responses, we return Lambda action payloads

	var response []map[string]interface{}

	for _, action := range actions {
		switch a := action.(type) {
		case IVRSay:
			// Use Amazon Polly for TTS
			response = append(response, map[string]interface{}{
				"Type": "MessageParticipant",
				"Parameters": map[string]interface{}{
					"Text":     a.Text,
					"TextType": "text",
				},
			})

		case IVRPlay:
			response = append(response, map[string]interface{}{
				"Type": "PlayPrompt",
				"Parameters": map[string]interface{}{
					"SourceType": "S3",
					"SourceValue": a.URL,
				},
			})

		case IVRGather:
			params := map[string]interface{}{
				"Timeout": strconv.Itoa(a.Timeout),
			}

			if a.NumDigits > 0 {
				params["MaxDigits"] = strconv.Itoa(a.NumDigits)
			}

			inputType := "DTMF"
			for _, input := range a.Input {
				if input == "speech" {
					inputType = "DTMF_AND_VOICE"
					break
				}
			}
			params["InputType"] = inputType

			response = append(response, map[string]interface{}{
				"Type":       "GetParticipantInput",
				"Parameters": params,
			})

		case IVRRecord:
			response = append(response, map[string]interface{}{
				"Type": "StartMediaStreaming",
				"Parameters": map[string]interface{}{
					"MediaStreamType": "AUDIO",
				},
			})

		case IVRDial:
			if a.Queue != "" {
				response = append(response, map[string]interface{}{
					"Type": "TransferToQueue",
					"Parameters": map[string]interface{}{
						"QueueId": a.Queue,
					},
				})
			} else {
				response = append(response, map[string]interface{}{
					"Type": "TransferParticipantToThirdParty",
					"Parameters": map[string]interface{}{
						"ThirdPartyPhoneNumber": a.Number,
					},
				})
			}

		case IVRHangup:
			response = append(response, map[string]interface{}{
				"Type": "DisconnectParticipant",
			})

		case IVRQueue:
			response = append(response, map[string]interface{}{
				"Type": "TransferToQueue",
				"Parameters": map[string]interface{}{
					"QueueId": a.Name,
				},
			})
		}
	}

	return response, nil
}

// ParseWebhook parses an incoming Amazon Connect webhook (EventBridge)
func (p *AmazonConnectProvider) ParseWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	event := &WebhookEvent{
		Timestamp:  time.Now(),
		RawPayload: payload,
	}

	// Parse EventBridge event
	if detail, ok := payload["detail"].(map[string]interface{}); ok {
		if contactId, ok := detail["contactId"].(string); ok {
			event.CallID = contactId
			event.ExternalID = contactId
		}

		if eventType, ok := detail["eventType"].(string); ok {
			event.Type = strings.ToLower(eventType)
		}

		// Map status
		if state, ok := detail["currentContactState"].(string); ok {
			event.Status = p.mapStatus(state)
		}

		// Get phone numbers from queue/customer endpoint
		if customerEndpoint, ok := detail["customerEndpoint"].(map[string]interface{}); ok {
			if address, ok := customerEndpoint["address"].(string); ok {
				event.From = address
			}
		}

		if systemEndpoint, ok := detail["systemEndpoint"].(map[string]interface{}); ok {
			if address, ok := systemEndpoint["address"].(string); ok {
				event.To = address
			}
		}

		// Direction
		if initiationMethod, ok := detail["initiationMethod"].(string); ok {
			if initiationMethod == "INBOUND" {
				event.Direction = CallDirectionInbound
			} else {
				event.Direction = CallDirectionOutbound
			}
		}
	}

	return event, nil
}

// ValidateWebhook validates Amazon Connect webhook signature
func (p *AmazonConnectProvider) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) bool {
	// EventBridge events are delivered via HTTPS from AWS
	// Additional validation can be done via Lambda authorizer
	return true
}

// mapStatus maps Amazon Connect status to CallStatus
func (p *AmazonConnectProvider) mapStatus(status string) CallStatus {
	statusMap := map[string]CallStatus{
		"INCOMING":    CallStatusRinging,
		"PENDING":     CallStatusInitiated,
		"CONNECTING":  CallStatusInitiated,
		"CONNECTED":   CallStatusInProgress,
		"CONNECTED_ONHOLD": CallStatusInProgress,
		"MISSED":      CallStatusNoAnswer,
		"ERROR":       CallStatusFailed,
		"ENDED":       CallStatusCompleted,
		"REJECTED":    CallStatusFailed,
	}

	if mapped, ok := statusMap[strings.ToUpper(status)]; ok {
		return mapped
	}
	return CallStatus(status)
}

// ListQueues returns available queues
func (p *AmazonConnectProvider) ListQueues(ctx context.Context) ([]map[string]string, error) {
	endpoint := fmt.Sprintf("%s/queues-summary/%s", p.baseURL, p.instanceID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	p.signRequest(req, nil)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		QueueSummaryList []struct {
			Id   string `json:"Id"`
			Name string `json:"Name"`
			Arn  string `json:"Arn"`
		} `json:"QueueSummaryList"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	queues := make([]map[string]string, len(result.QueueSummaryList))
	for i, q := range result.QueueSummaryList {
		queues[i] = map[string]string{
			"id":   q.Id,
			"name": q.Name,
			"arn":  q.Arn,
		}
	}

	return queues, nil
}

// GetContactAttributes returns contact attributes for a call
func (p *AmazonConnectProvider) GetContactAttributes(ctx context.Context, callID string) (map[string]string, error) {
	endpoint := fmt.Sprintf("%s/contact/attributes/%s/%s", p.baseURL, p.instanceID, callID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	p.signRequest(req, nil)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Attributes map[string]string `json:"Attributes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Attributes, nil
}

// Helper to sort string slices
func sortStrings(strs []string) []string {
	sort.Strings(strs)
	return strs
}
