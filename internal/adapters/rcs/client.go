package rcs

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the RCS API client
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new RCS client
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendMessage sends an RCS message
func (c *Client) SendMessage(ctx context.Context, msg *OutboundMessage) (*SendResult, error) {
	switch c.config.Provider {
	case ProviderZenvia:
		return c.sendZenviaMessage(ctx, msg)
	case ProviderInfobip:
		return c.sendInfobipMessage(ctx, msg)
	case ProviderPontaltech:
		return c.sendPontaltechMessage(ctx, msg)
	case ProviderGoogle:
		return c.sendGoogleMessage(ctx, msg)
	default:
		return nil, ErrInvalidProvider
	}
}

// ParseWebhook parses an incoming webhook payload
func (c *Client) ParseWebhook(body []byte) (*WebhookPayload, error) {
	switch c.config.Provider {
	case ProviderZenvia:
		return c.parseZenviaWebhook(body)
	case ProviderInfobip:
		return c.parseInfobipWebhook(body)
	case ProviderPontaltech:
		return c.parsePontaltechWebhook(body)
	case ProviderGoogle:
		return c.parseGoogleWebhook(body)
	default:
		return nil, ErrInvalidProvider
	}
}

// ValidateWebhook validates a webhook signature
func (c *Client) ValidateWebhook(signature string, body []byte) bool {
	if c.config.WebhookSecret == "" {
		return true // Skip validation if no secret
	}

	mac := hmac.New(sha256.New, []byte(c.config.WebhookSecret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Handle different signature formats
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signature = signature[7:]
	}

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// GetAgentInfo retrieves information about the RCS agent
func (c *Client) GetAgentInfo(ctx context.Context) (map[string]interface{}, error) {
	switch c.config.Provider {
	case ProviderZenvia:
		return c.getZenviaAgentInfo(ctx)
	case ProviderInfobip:
		return c.getInfobipAgentInfo(ctx)
	default:
		return map[string]interface{}{
			"agent_id": c.config.AgentID,
			"provider": string(c.config.Provider),
		}, nil
	}
}

// ========== ZENVIA PROVIDER ==========

// ZenviaMessage represents a Zenvia RCS message
type ZenviaMessage struct {
	From     string            `json:"from"`
	To       string            `json:"to"`
	Contents []ZenviaContent   `json:"contents"`
}

// ZenviaContent represents Zenvia message content
type ZenviaContent struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	File *ZenviaFile `json:"file,omitempty"`
}

// ZenviaFile represents a Zenvia file
type ZenviaFile struct {
	FileURL   string `json:"fileUrl"`
	FileMIME  string `json:"fileMimeType"`
	FileName  string `json:"fileName,omitempty"`
}

func (c *Client) sendZenviaMessage(ctx context.Context, msg *OutboundMessage) (*SendResult, error) {
	zenviaMsg := &ZenviaMessage{
		From: c.config.AgentID,
		To:   msg.To,
	}

	// Add text content
	if msg.Text != "" {
		zenviaMsg.Contents = append(zenviaMsg.Contents, ZenviaContent{
			Type: "text",
			Text: msg.Text,
		})
	}

	// Add media content
	if msg.MediaURL != "" {
		zenviaMsg.Contents = append(zenviaMsg.Contents, ZenviaContent{
			Type: "file",
			File: &ZenviaFile{
				FileURL:  msg.MediaURL,
				FileMIME: msg.MediaType,
			},
		})
	}

	body, err := json.Marshal(zenviaMsg)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.GetBaseURL()+"/channels/rcs/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-TOKEN", c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return &SendResult{
			Success:   false,
			Error:     string(respBody),
			Timestamp: time.Now(),
		}, nil
	}

	var result struct {
		ID string `json:"id"`
	}
	json.Unmarshal(respBody, &result)

	return &SendResult{
		Success:   true,
		MessageID: result.ID,
		Timestamp: time.Now(),
	}, nil
}

func (c *Client) parseZenviaWebhook(body []byte) (*WebhookPayload, error) {
	var webhook struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
		Type      string `json:"type"` // "MESSAGE", "MESSAGE_STATUS"
		Channel   string `json:"channel"`
		Direction string `json:"direction"`
		Message   *struct {
			ID       string `json:"id"`
			From     string `json:"from"`
			To       string `json:"to"`
			Contents []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			} `json:"contents"`
		} `json:"message,omitempty"`
		MessageStatus *struct {
			Timestamp string `json:"timestamp"`
			Code      string `json:"code"`
		} `json:"messageStatus,omitempty"`
	}

	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, err
	}

	payload := &WebhookPayload{
		Provider:   ProviderZenvia,
		RawPayload: webhook,
	}

	if webhook.Type == "MESSAGE" && webhook.Message != nil {
		timestamp, _ := time.Parse(time.RFC3339, webhook.Timestamp)
		text := ""
		if len(webhook.Message.Contents) > 0 {
			text = webhook.Message.Contents[0].Text
		}

		payload.Type = "message"
		payload.Message = &IncomingMessage{
			ExternalID:  webhook.Message.ID,
			SenderPhone: webhook.Message.From,
			Text:        text,
			Timestamp:   timestamp,
			AgentID:     webhook.Message.To,
		}
	} else if webhook.Type == "MESSAGE_STATUS" && webhook.MessageStatus != nil {
		timestamp, _ := time.Parse(time.RFC3339, webhook.MessageStatus.Timestamp)
		payload.Type = "status"
		payload.Status = &DeliveryReport{
			MessageID: webhook.ID,
			Status:    mapZenviaStatus(webhook.MessageStatus.Code),
			Timestamp: timestamp,
		}
	}

	return payload, nil
}

func mapZenviaStatus(code string) DeliveryStatus {
	switch code {
	case "SENT":
		return StatusSent
	case "DELIVERED":
		return StatusDelivered
	case "READ":
		return StatusRead
	case "FAILED", "REJECTED":
		return StatusFailed
	default:
		return StatusPending
	}
}

func (c *Client) getZenviaAgentInfo(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"provider": "zenvia",
		"agent_id": c.config.AgentID,
	}, nil
}

// ========== INFOBIP PROVIDER ==========

// InfobipMessage represents an Infobip RCS message
type InfobipMessage struct {
	From         string             `json:"from"`
	Destinations []InfobipDest      `json:"destinations"`
	Content      InfobipContent     `json:"content"`
}

// InfobipDest represents a destination
type InfobipDest struct {
	To string `json:"to"`
}

// InfobipContent represents message content
type InfobipContent struct {
	Text string `json:"text,omitempty"`
}

func (c *Client) sendInfobipMessage(ctx context.Context, msg *OutboundMessage) (*SendResult, error) {
	infobipMsg := &InfobipMessage{
		From: c.config.AgentID,
		Destinations: []InfobipDest{
			{To: msg.To},
		},
		Content: InfobipContent{
			Text: msg.Text,
		},
	}

	body, err := json.Marshal(infobipMsg)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.GetBaseURL()+"/rcs/1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "App "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return &SendResult{
			Success:   false,
			Error:     string(respBody),
			Timestamp: time.Now(),
		}, nil
	}

	var result struct {
		Messages []struct {
			MessageID string `json:"messageId"`
		} `json:"messages"`
	}
	json.Unmarshal(respBody, &result)

	messageID := ""
	if len(result.Messages) > 0 {
		messageID = result.Messages[0].MessageID
	}

	return &SendResult{
		Success:   true,
		MessageID: messageID,
		Timestamp: time.Now(),
	}, nil
}

func (c *Client) parseInfobipWebhook(body []byte) (*WebhookPayload, error) {
	var webhook struct {
		Results []struct {
			MessageID   string `json:"messageId"`
			From        string `json:"from"`
			To          string `json:"to"`
			ReceivedAt  string `json:"receivedAt"`
			Text        string `json:"text,omitempty"`
			Status      string `json:"status,omitempty"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, err
	}

	payload := &WebhookPayload{
		Provider:   ProviderInfobip,
		RawPayload: webhook,
	}

	if len(webhook.Results) > 0 {
		result := webhook.Results[0]
		timestamp, _ := time.Parse(time.RFC3339, result.ReceivedAt)

		if result.Text != "" {
			payload.Type = "message"
			payload.Message = &IncomingMessage{
				ExternalID:  result.MessageID,
				SenderPhone: result.From,
				Text:        result.Text,
				Timestamp:   timestamp,
				AgentID:     result.To,
			}
		} else if result.Status != "" {
			payload.Type = "status"
			payload.Status = &DeliveryReport{
				MessageID: result.MessageID,
				Status:    mapInfobipStatus(result.Status),
				Timestamp: timestamp,
			}
		}
	}

	return payload, nil
}

func mapInfobipStatus(status string) DeliveryStatus {
	switch status {
	case "PENDING":
		return StatusPending
	case "DELIVERED":
		return StatusDelivered
	case "SEEN", "READ":
		return StatusRead
	case "FAILED", "REJECTED":
		return StatusFailed
	default:
		return StatusSent
	}
}

func (c *Client) getInfobipAgentInfo(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"provider": "infobip",
		"agent_id": c.config.AgentID,
	}, nil
}

// ========== PONTALTECH PROVIDER ==========

func (c *Client) sendPontaltechMessage(ctx context.Context, msg *OutboundMessage) (*SendResult, error) {
	payload := map[string]interface{}{
		"to":      msg.To,
		"content": msg.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.GetBaseURL()+"/rcs/send", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return &SendResult{
			Success:   false,
			Error:     string(respBody),
			Timestamp: time.Now(),
		}, nil
	}

	var result struct {
		ID string `json:"id"`
	}
	json.Unmarshal(respBody, &result)

	return &SendResult{
		Success:   true,
		MessageID: result.ID,
		Timestamp: time.Now(),
	}, nil
}

func (c *Client) parsePontaltechWebhook(body []byte) (*WebhookPayload, error) {
	var webhook struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		From      string `json:"from"`
		To        string `json:"to"`
		Content   string `json:"content"`
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
	}

	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, err
	}

	payload := &WebhookPayload{
		Provider:   ProviderPontaltech,
		RawPayload: webhook,
	}

	timestamp, _ := time.Parse(time.RFC3339, webhook.Timestamp)

	if webhook.Type == "message" {
		payload.Type = "message"
		payload.Message = &IncomingMessage{
			ExternalID:  webhook.ID,
			SenderPhone: webhook.From,
			Text:        webhook.Content,
			Timestamp:   timestamp,
			AgentID:     webhook.To,
		}
	} else if webhook.Type == "status" {
		payload.Type = "status"
		payload.Status = &DeliveryReport{
			MessageID: webhook.ID,
			Status:    DeliveryStatus(webhook.Status),
			Timestamp: timestamp,
		}
	}

	return payload, nil
}

// ========== GOOGLE RCS BUSINESS MESSAGING ==========

func (c *Client) sendGoogleMessage(ctx context.Context, msg *OutboundMessage) (*SendResult, error) {
	// Google RBM uses a different API structure
	payload := map[string]interface{}{
		"contentMessage": map[string]interface{}{
			"text": msg.Text,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/phones/%s/agentMessages?messageId=%s",
		c.config.GetBaseURL(),
		msg.To,
		generateMessageID(),
	)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return &SendResult{
			Success:   false,
			Error:     string(respBody),
			Timestamp: time.Now(),
		}, nil
	}

	var result struct {
		Name string `json:"name"`
	}
	json.Unmarshal(respBody, &result)

	return &SendResult{
		Success:   true,
		MessageID: result.Name,
		Timestamp: time.Now(),
	}, nil
}

func (c *Client) parseGoogleWebhook(body []byte) (*WebhookPayload, error) {
	var webhook struct {
		SenderPhoneNumber string `json:"senderPhoneNumber"`
		MessageID         string `json:"messageId"`
		SendTime          string `json:"sendTime"`
		Text              string `json:"text,omitempty"`
		AgentMessage      *struct {
			ContentMessage struct {
				Text string `json:"text"`
			} `json:"contentMessage"`
		} `json:"agentMessage,omitempty"`
	}

	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, err
	}

	payload := &WebhookPayload{
		Provider:   ProviderGoogle,
		RawPayload: webhook,
	}

	timestamp, _ := time.Parse(time.RFC3339, webhook.SendTime)

	if webhook.Text != "" {
		payload.Type = "message"
		payload.Message = &IncomingMessage{
			ExternalID:  webhook.MessageID,
			SenderPhone: webhook.SenderPhoneNumber,
			Text:        webhook.Text,
			Timestamp:   timestamp,
			AgentID:     c.config.AgentID,
		}
	}

	return payload, nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
