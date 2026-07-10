package officialcalls

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client speaks the WhatsApp Business Calling signaling API over the Graph API.
type Client struct {
	http          *http.Client
	baseURL       string // e.g. https://graph.facebook.com
	apiVersion    string // e.g. v23.0
	accessToken   string
	phoneNumberID string
}

// Config configures a signaling Client.
type Config struct {
	BaseURL       string // defaults to https://graph.facebook.com
	APIVersion    string // defaults to v23.0
	AccessToken   string
	PhoneNumberID string
	HTTPClient    *http.Client
}

const (
	defaultBaseURL    = "https://graph.facebook.com"
	defaultAPIVersion = "v23.0"
)

// NewClient builds a signaling client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if cfg.PhoneNumberID == "" {
		return nil, fmt.Errorf("phone number id is required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	ver := cfg.APIVersion
	if ver == "" {
		ver = defaultAPIVersion
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		http:          hc,
		baseURL:       strings.TrimRight(base, "/"),
		apiVersion:    ver,
		accessToken:   cfg.AccessToken,
		phoneNumberID: cfg.PhoneNumberID,
	}, nil
}

// EnableCalling turns calling on for the phone number.
func (c *Client) EnableCalling(ctx context.Context) error {
	return c.SetCallingSettings(ctx, &CallingSettings{Status: CallingEnabled})
}

// SetCallingSettings applies calling settings via POST /{phone_number_id}/settings.
func (c *Client) SetCallingSettings(ctx context.Context, s *CallingSettings) error {
	if s == nil {
		return fmt.Errorf("settings required")
	}
	_, err := c.do(ctx, http.MethodPost, c.path("/settings"), settingsRequest{Calling: s})
	return err
}

// GetCallPermission returns the business's permission to place a call to userWaID.
func (c *Client) GetCallPermission(ctx context.Context, userWaID string) (*CallPermission, error) {
	q := url.Values{"user_wa_id": {userWaID}}
	body, err := c.do(ctx, http.MethodGet, c.path("/call_permissions")+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	// The Graph response wraps the permission under a data array or a top-level
	// object depending on version; accept both.
	var wrapper struct {
		Data []CallPermission `json:"data"`
		CallPermission
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("parse call permission: %w", err)
	}
	perm := wrapper.CallPermission
	if len(wrapper.Data) > 0 {
		perm = wrapper.Data[0]
	}
	// Keep the API's own can_place_call when it grants permission; only fall back
	// to the status allow-list, so a new/renamed granting status isn't clobbered.
	perm.CanPlaceCall = perm.CanPlaceCall || perm.Status == "temporary" || perm.Status == "permanent"
	return &perm, nil
}

// Connect places a business-initiated call to `to` with our SDP offer and
// returns the assigned call id.
func (c *Client) Connect(ctx context.Context, to, offerSDP string) (string, error) {
	resp, err := c.callAction(ctx, callActionRequest{
		MessagingProduct: "whatsapp",
		Action:           ActionConnect,
		To:               to,
		Session:          &Session{SDPType: SDPOffer, SDP: offerSDP},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Calls) == 0 || resp.Calls[0].ID == "" {
		return "", fmt.Errorf("connect: no call id in response")
	}
	return resp.Calls[0].ID, nil
}

// PreAccept pre-accepts an inbound call with an early SDP answer so media can
// start negotiating before the full accept.
func (c *Client) PreAccept(ctx context.Context, callID, answerSDP string) error {
	_, err := c.callAction(ctx, callActionRequest{
		MessagingProduct: "whatsapp",
		Action:           ActionPreAccept,
		CallID:           callID,
		Session:          &Session{SDPType: SDPAnswer, SDP: answerSDP},
	})
	return err
}

// Accept accepts an inbound call with our SDP answer.
func (c *Client) Accept(ctx context.Context, callID, answerSDP string) error {
	_, err := c.callAction(ctx, callActionRequest{
		MessagingProduct: "whatsapp",
		Action:           ActionAccept,
		CallID:           callID,
		Session:          &Session{SDPType: SDPAnswer, SDP: answerSDP},
	})
	return err
}

// Reject rejects an inbound call.
func (c *Client) Reject(ctx context.Context, callID string) error {
	_, err := c.callAction(ctx, callActionRequest{
		MessagingProduct: "whatsapp",
		Action:           ActionReject,
		CallID:           callID,
	})
	return err
}

// Terminate ends an active call.
func (c *Client) Terminate(ctx context.Context, callID string) error {
	_, err := c.callAction(ctx, callActionRequest{
		MessagingProduct: "whatsapp",
		Action:           ActionTerminate,
		CallID:           callID,
	})
	return err
}

func (c *Client) callAction(ctx context.Context, req callActionRequest) (*callActionResponse, error) {
	body, err := c.do(ctx, http.MethodPost, c.path("/calls"), req)
	if err != nil {
		return nil, err
	}
	var resp callActionResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse call response: %w", err)
		}
	}
	return &resp, nil
}

// path builds "/{apiVersion}/{phoneNumberID}{suffix}".
func (c *Client) path(suffix string) string {
	return fmt.Sprintf("/%s/%s%s", c.apiVersion, c.phoneNumberID, suffix)
}

// do performs an authenticated Graph request and returns the raw body, mapping
// Graph error envelopes to Go errors.
func (c *Client) do(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var reader io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, parseGraphError(resp.StatusCode, body)
	}
	// Graph can return HTTP 200 with an embedded error envelope; treat that as a
	// failure rather than silently proceeding as success.
	if gerr := graphErrorFromBody(resp.StatusCode, body); gerr != nil {
		return nil, gerr
	}
	return body, nil
}

// GraphError is a WhatsApp/Graph API error.
type GraphError struct {
	HTTPStatus int
	Code       int
	Message    string
}

func (e *GraphError) Error() string {
	return fmt.Sprintf("whatsapp calling API error (http %d, code %d): %s", e.HTTPStatus, e.Code, e.Message)
}

// graphErrorFromBody returns a *GraphError if the body carries a Graph error
// envelope ({"error":{...}}), else nil.
func graphErrorFromBody(status int, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var env struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Message != "" {
		return &GraphError{HTTPStatus: status, Code: env.Error.Code, Message: env.Error.Message}
	}
	return nil
}

func parseGraphError(status int, body []byte) error {
	if gerr := graphErrorFromBody(status, body); gerr != nil {
		return gerr
	}
	return &GraphError{HTTPStatus: status, Message: strings.TrimSpace(string(body))}
}
