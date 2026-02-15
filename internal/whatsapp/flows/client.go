package flows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

// FlowStatus represents the status of a WhatsApp Flow
type FlowStatus string

const (
	FlowStatusDraft      FlowStatus = "DRAFT"
	FlowStatusPublished  FlowStatus = "PUBLISHED"
	FlowStatusDeprecated FlowStatus = "DEPRECATED"
	FlowStatusBlocked    FlowStatus = "BLOCKED"
	FlowStatusThrottled  FlowStatus = "THROTTLED"
)

// FlowCategory represents the category of a WhatsApp Flow
type FlowCategory string

const (
	FlowCategorySignUp          FlowCategory = "SIGN_UP"
	FlowCategorySignIn          FlowCategory = "SIGN_IN"
	FlowCategoryAppointment     FlowCategory = "APPOINTMENT_BOOKING"
	FlowCategoryLeadGen         FlowCategory = "LEAD_GENERATION"
	FlowCategoryContactUs       FlowCategory = "CONTACT_US"
	FlowCategoryCustomerSupport FlowCategory = "CUSTOMER_SUPPORT"
	FlowCategorySurvey          FlowCategory = "SURVEY"
	FlowCategoryOther           FlowCategory = "OTHER"
)

// FlowInfo represents information about a WhatsApp Flow
type FlowInfo struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Status            FlowStatus   `json:"status"`
	Categories        []string     `json:"categories,omitempty"`
	ValidationErrors  []FlowError  `json:"validation_errors,omitempty"`
	JSONVersion       string       `json:"json_version,omitempty"`
	DataAPIVersion    string       `json:"data_api_version,omitempty"`
	EndpointURI       string       `json:"endpoint_uri,omitempty"`
	PreviewURL        string       `json:"preview_url,omitempty"`
	WhatsAppBusinessAccountID string `json:"whatsapp_business_account_id,omitempty"`
}

// FlowError represents a validation error in a flow
type FlowError struct {
	Error            string `json:"error"`
	ErrorType        string `json:"error_type"`
	Message          string `json:"message"`
	LineStart        int    `json:"line_start,omitempty"`
	LineEnd          int    `json:"line_end,omitempty"`
	ColumnStart      int    `json:"column_start,omitempty"`
	ColumnEnd        int    `json:"column_end,omitempty"`
}

// CreateFlowRequest represents a request to create a new flow
type CreateFlowRequest struct {
	Name           string       `json:"name"`
	Categories     []string     `json:"categories,omitempty"`
	EndpointURI    string       `json:"endpoint_uri,omitempty"`
	CloneFlowID    string       `json:"clone_flow_id,omitempty"`
}

// UpdateFlowRequest represents a request to update a flow
type UpdateFlowRequest struct {
	Name           string   `json:"name,omitempty"`
	Categories     []string `json:"categories,omitempty"`
	EndpointURI    string   `json:"endpoint_uri,omitempty"`
	ApplicationID  string   `json:"application_id,omitempty"`
}

// FlowAssets represents the JSON assets of a flow
type FlowAssets struct {
	FlowJSON string `json:"flow_json"`
}

// FlowClient is a client for the WhatsApp Flows API
type FlowClient struct {
	httpClient     *http.Client
	accessToken    string
	wabaID         string // WhatsApp Business Account ID
	apiVersion     string
	baseURL        string
}

// FlowClientConfig represents configuration for the flow client
type FlowClientConfig struct {
	AccessToken string
	WABAID      string
	APIVersion  string
}

// NewFlowClient creates a new flow client
func NewFlowClient(config *FlowClientConfig) *FlowClient {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	return &FlowClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken: config.AccessToken,
		wabaID:      config.WABAID,
		apiVersion:  apiVersion,
		baseURL:     "https://graph.facebook.com",
	}
}

// buildURL builds the API URL
func (c *FlowClient) buildURL(path string) string {
	return fmt.Sprintf("%s/%s%s", c.baseURL, c.apiVersion, path)
}

// doRequest executes an HTTP request
func (c *FlowClient) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
	}

	return respBody, nil
}

// CreateFlow creates a new flow
func (c *FlowClient) CreateFlow(ctx context.Context, req *CreateFlowRequest) (*FlowInfo, error) {
	url := c.buildURL(fmt.Sprintf("/%s/flows", c.wabaID))

	respBody, err := c.doRequest(ctx, http.MethodPost, url, req)
	if err != nil {
		return nil, err
	}

	var result FlowInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetFlow retrieves a flow by ID
func (c *FlowClient) GetFlow(ctx context.Context, flowID string) (*FlowInfo, error) {
	url := c.buildURL(fmt.Sprintf("/%s", flowID))
	url += "?fields=id,name,status,categories,validation_errors,json_version,data_api_version,endpoint_uri,preview_url"

	respBody, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result FlowInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ListFlows lists all flows for the WABA
func (c *FlowClient) ListFlows(ctx context.Context) ([]FlowInfo, error) {
	url := c.buildURL(fmt.Sprintf("/%s/flows", c.wabaID))
	url += "?fields=id,name,status,categories"

	respBody, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []FlowInfo `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data, nil
}

// UpdateFlow updates a flow
func (c *FlowClient) UpdateFlow(ctx context.Context, flowID string, req *UpdateFlowRequest) (*FlowInfo, error) {
	url := c.buildURL(fmt.Sprintf("/%s", flowID))

	respBody, err := c.doRequest(ctx, http.MethodPost, url, req)
	if err != nil {
		return nil, err
	}

	var result FlowInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// UpdateFlowJSON updates the JSON of a flow
func (c *FlowClient) UpdateFlowJSON(ctx context.Context, flowID, flowJSON string) error {
	url := c.buildURL(fmt.Sprintf("/%s/assets", flowID))

	// Flow JSON needs to be sent as multipart/form-data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add name field
	if err := writer.WriteField("name", "flow.json"); err != nil {
		return fmt.Errorf("failed to write name field: %w", err)
	}

	// Add asset_type field
	if err := writer.WriteField("asset_type", "FLOW_JSON"); err != nil {
		return fmt.Errorf("failed to write asset_type field: %w", err)
	}

	// Add file field
	part, err := writer.CreateFormFile("file", "flow.json")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write([]byte(flowJSON)); err != nil {
		return fmt.Errorf("failed to write flow JSON: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PublishFlow publishes a flow (changes status from DRAFT to PUBLISHED)
func (c *FlowClient) PublishFlow(ctx context.Context, flowID string) error {
	url := c.buildURL(fmt.Sprintf("/%s/publish", flowID))
	_, err := c.doRequest(ctx, http.MethodPost, url, nil)
	return err
}

// DeprecateFlow deprecates a flow
func (c *FlowClient) DeprecateFlow(ctx context.Context, flowID string) error {
	url := c.buildURL(fmt.Sprintf("/%s/deprecate", flowID))
	_, err := c.doRequest(ctx, http.MethodPost, url, nil)
	return err
}

// DeleteFlow deletes a flow (only DRAFT flows can be deleted)
func (c *FlowClient) DeleteFlow(ctx context.Context, flowID string) error {
	url := c.buildURL(fmt.Sprintf("/%s", flowID))
	_, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	return err
}

// GetFlowPreviewURL gets the preview URL for a flow
func (c *FlowClient) GetFlowPreviewURL(ctx context.Context, flowID string) (string, error) {
	flow, err := c.GetFlow(ctx, flowID)
	if err != nil {
		return "", err
	}
	return flow.PreviewURL, nil
}

// GetFlowAssets retrieves the flow JSON
func (c *FlowClient) GetFlowAssets(ctx context.Context, flowID string) (*FlowAssets, error) {
	url := c.buildURL(fmt.Sprintf("/%s/assets", flowID))

	respBody, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Name      string `json:"name"`
			AssetType string `json:"asset_type"`
			DownloadURL string `json:"download_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Download flow.json
	for _, asset := range result.Data {
		if asset.AssetType == "FLOW_JSON" {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+c.accessToken)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			return &FlowAssets{FlowJSON: string(data)}, nil
		}
	}

	return nil, fmt.Errorf("flow.json not found in assets")
}

// =============================================================================
// Flow Message Sending
// =============================================================================

// FlowMessage represents a flow message to send
type FlowMessage struct {
	To          string                 `json:"to"`
	Type        string                 `json:"type"` // "interactive"
	Interactive FlowInteractiveContent `json:"interactive"`
}

// FlowInteractiveContent represents the interactive content for a flow
type FlowInteractiveContent struct {
	Type   string           `json:"type"` // "flow"
	Header *FlowHeader      `json:"header,omitempty"`
	Body   FlowBody         `json:"body"`
	Footer *FlowFooter      `json:"footer,omitempty"`
	Action FlowAction       `json:"action"`
}

// FlowHeader represents the header of a flow message
type FlowHeader struct {
	Type     string `json:"type"` // "text", "image", "video", "document"
	Text     string `json:"text,omitempty"`
	Image    *MediaRef `json:"image,omitempty"`
	Video    *MediaRef `json:"video,omitempty"`
	Document *MediaRef `json:"document,omitempty"`
}

// MediaRef represents a media reference
type MediaRef struct {
	ID   string `json:"id,omitempty"`
	Link string `json:"link,omitempty"`
}

// FlowBody represents the body of a flow message
type FlowBody struct {
	Text string `json:"text"`
}

// FlowFooter represents the footer of a flow message
type FlowFooter struct {
	Text string `json:"text"`
}

// FlowAction represents the action configuration for a flow
type FlowAction struct {
	Name       string                 `json:"name"` // "flow"
	Parameters FlowActionParameters   `json:"parameters"`
}

// FlowActionParameters represents the parameters for a flow action
type FlowActionParameters struct {
	FlowMessageVersion string                 `json:"flow_message_version"` // "3"
	FlowToken          string                 `json:"flow_token"`
	FlowID             string                 `json:"flow_id"`
	FlowCTA            string                 `json:"flow_cta"`
	FlowAction         string                 `json:"flow_action"` // "navigate", "data_exchange"
	FlowActionPayload  map[string]interface{} `json:"flow_action_payload,omitempty"`
	Mode               string                 `json:"mode,omitempty"` // "draft", "published"
}

// SendFlowInput represents input for sending a flow message
type SendFlowInput struct {
	To           string
	FlowID       string
	FlowCTA      string // Button text
	BodyText     string
	HeaderText   string
	FooterText   string
	FlowToken    string // Token to track the flow session
	InitialScreen string
	InitialData  map[string]interface{}
	Mode         string // "draft" for testing, empty for published
}

// FlowSender sends flow messages
type FlowSender struct {
	phoneNumberID string
	accessToken   string
	apiVersion    string
	httpClient    *http.Client
}

// NewFlowSender creates a new flow sender
func NewFlowSender(phoneNumberID, accessToken, apiVersion string) *FlowSender {
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	return &FlowSender{
		phoneNumberID: phoneNumberID,
		accessToken:   accessToken,
		apiVersion:    apiVersion,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// buildFlowAction builds the flow action with proper nil handling
func (s *FlowSender) buildFlowAction(input *SendFlowInput) map[string]interface{} {
	payload := map[string]interface{}{
		"screen": input.InitialScreen,
	}
	if input.InitialData != nil {
		payload["data"] = input.InitialData
	}

	params := map[string]interface{}{
		"flow_message_version": "3",
		"flow_token":           input.FlowToken,
		"flow_id":              input.FlowID,
		"flow_cta":             input.FlowCTA,
		"flow_action":          "navigate",
		"flow_action_payload":  payload,
	}

	if input.Mode != "" {
		params["mode"] = input.Mode
	}

	return map[string]interface{}{
		"name":       "flow",
		"parameters": params,
	}
}

// SendFlow sends a flow message
func (s *FlowSender) SendFlow(ctx context.Context, input *SendFlowInput) (string, error) {
	msg := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                input.To,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "flow",
			"body": map[string]string{
				"text": input.BodyText,
			},
			"action": s.buildFlowAction(input),
		},
	}

	// Add optional header
	if input.HeaderText != "" {
		msg["interactive"].(map[string]interface{})["header"] = map[string]interface{}{
			"type": "text",
			"text": input.HeaderText,
		}
	}

	// Add optional footer
	if input.FooterText != "" {
		msg["interactive"].(map[string]interface{})["footer"] = map[string]interface{}{
			"text": input.FooterText,
		}
	}

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", s.apiVersion, s.phoneNumberID)

	body, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return result.Messages[0].ID, nil
}

// =============================================================================
// NFM Reply Processing
// =============================================================================

// NFMReply represents a reply from a completed flow
type NFMReply struct {
	FlowToken    string                 `json:"flow_token"`
	Screen       string                 `json:"screen"`
	ResponseJSON map[string]interface{} `json:"response_json"`
}

// ParseNFMReply parses an NFM reply from webhook data
func ParseNFMReply(data map[string]interface{}) (*NFMReply, error) {
	interactive, ok := data["interactive"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing interactive field")
	}

	nfmReply, ok := interactive["nfm_reply"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing nfm_reply field")
	}

	result := &NFMReply{}

	if name, ok := nfmReply["name"].(string); ok {
		result.Screen = name
	}

	if body, ok := nfmReply["body"].(string); ok {
		result.FlowToken = body
	}

	if responseJSON, ok := nfmReply["response_json"].(string); ok {
		if err := json.Unmarshal([]byte(responseJSON), &result.ResponseJSON); err != nil {
			return nil, fmt.Errorf("failed to parse response_json: %w", err)
		}
	}

	return result, nil
}

// NFMReplyHandler handles NFM replies
type NFMReplyHandler struct {
	mu       sync.RWMutex
	handlers map[string]func(*NFMReply) error
}

// NewNFMReplyHandler creates a new NFM reply handler
func NewNFMReplyHandler() *NFMReplyHandler {
	return &NFMReplyHandler{
		handlers: make(map[string]func(*NFMReply) error),
	}
}

// RegisterHandler registers a handler for a specific screen
func (h *NFMReplyHandler) RegisterHandler(screen string, handler func(*NFMReply) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[screen] = handler
}

// Handle processes an NFM reply
func (h *NFMReplyHandler) Handle(reply *NFMReply) error {
	h.mu.RLock()
	handler, ok := h.handlers[reply.Screen]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no handler for screen: %s", reply.Screen)
	}
	return handler(reply)
}

// ExtractFormData extracts form data from an NFM reply
func (reply *NFMReply) ExtractFormData() map[string]interface{} {
	if reply.ResponseJSON == nil {
		return nil
	}
	return reply.ResponseJSON
}

// GetString gets a string value from the response
func (reply *NFMReply) GetString(key string) string {
	if reply.ResponseJSON == nil {
		return ""
	}
	if val, ok := reply.ResponseJSON[key].(string); ok {
		return val
	}
	return ""
}

// GetBool gets a boolean value from the response
func (reply *NFMReply) GetBool(key string) bool {
	if reply.ResponseJSON == nil {
		return false
	}
	if val, ok := reply.ResponseJSON[key].(bool); ok {
		return val
	}
	return false
}

// GetFloat gets a float value from the response
func (reply *NFMReply) GetFloat(key string) float64 {
	if reply.ResponseJSON == nil {
		return 0
	}
	if val, ok := reply.ResponseJSON[key].(float64); ok {
		return val
	}
	return 0
}

// GetStringSlice gets a string slice from the response
func (reply *NFMReply) GetStringSlice(key string) []string {
	if reply.ResponseJSON == nil {
		return nil
	}
	if val, ok := reply.ResponseJSON[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
