package sms

import (
	"fmt"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Client wraps the Twilio REST API client
type Client struct {
	api                 *twilio.RestClient
	config              *TwilioConfig
	statusCallbackURL   string
}

// NewClient creates a new Twilio SMS client
func NewClient(config *TwilioConfig) (*Client, error) {
	if config.AccountSID == "" {
		return nil, fmt.Errorf("account_sid is required")
	}

	// Determine auth method
	var client *twilio.RestClient
	if config.APIKeySID != "" && config.APIKeySecret != "" {
		// API Key authentication
		client = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username:   config.APIKeySID,
			Password:   config.APIKeySecret,
			AccountSid: config.AccountSID,
		})
	} else if config.AuthToken != "" {
		// Account SID + Auth Token authentication
		client = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: config.AccountSID,
			Password: config.AuthToken,
		})
	} else {
		return nil, fmt.Errorf("either auth_token or api_key_sid+api_key_secret is required")
	}

	return &Client{
		api:               client,
		config:            config,
		statusCallbackURL: config.StatusCallbackURL,
	}, nil
}

// GetAccountInfo retrieves account information to verify credentials
func (c *Client) GetAccountInfo() (string, error) {
	account, err := c.api.Api.FetchAccount(c.config.AccountSID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch account: %w", err)
	}

	if account.FriendlyName != nil {
		return *account.FriendlyName, nil
	}
	return c.config.AccountSID, nil
}

// SendMessage sends an SMS/MMS message
func (c *Client) SendMessage(to, body string, mediaURLs []string) (*SendResult, error) {
	params := &twilioApi.CreateMessageParams{}

	// Set recipient
	params.SetTo(to)

	// Set message body
	params.SetBody(body)

	// Set sender (messaging service or phone number)
	if c.config.MessagingServiceSID != "" {
		params.SetMessagingServiceSid(c.config.MessagingServiceSID)
	} else if c.config.PhoneNumber != "" {
		params.SetFrom(c.config.PhoneNumber)
	} else {
		return nil, fmt.Errorf("either phone_number or messaging_service_sid is required")
	}

	// Set status callback URL if configured
	if c.statusCallbackURL != "" {
		params.SetStatusCallback(c.statusCallbackURL)
	}

	// Add media URLs for MMS
	if len(mediaURLs) > 0 {
		params.SetMediaUrl(mediaURLs)
	}

	// Send message
	resp, err := c.api.Api.CreateMessage(params)
	if err != nil {
		return &SendResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	result := &SendResult{
		Success: true,
	}

	if resp.Sid != nil {
		result.MessageSID = *resp.Sid
	}
	if resp.Status != nil {
		result.Status = ParseMessageStatus(*resp.Status)
	}
	if resp.ErrorCode != nil {
		result.ErrorCode = fmt.Sprintf("%d", *resp.ErrorCode)
	}
	if resp.ErrorMessage != nil {
		result.Error = *resp.ErrorMessage
		result.Success = false
	}

	return result, nil
}

// GetMessage retrieves information about a sent message
func (c *Client) GetMessage(messageSID string) (*twilioApi.ApiV2010Message, error) {
	params := &twilioApi.FetchMessageParams{}
	return c.api.Api.FetchMessage(messageSID, params)
}

// GetPhoneNumber returns the configured phone number
func (c *Client) GetPhoneNumber() string {
	return c.config.PhoneNumber
}

// GetMessagingServiceSID returns the configured messaging service SID
func (c *Client) GetMessagingServiceSID() string {
	return c.config.MessagingServiceSID
}

// ValidatePhoneNumber validates a phone number format (basic E.164 check)
func ValidatePhoneNumber(phone string) bool {
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}
	if phone[0] != '+' {
		return false
	}
	for _, c := range phone[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// FormatPhoneNumber formats a phone number to E.164 format
// This is a basic implementation - production should use a proper library
func FormatPhoneNumber(phone, defaultCountryCode string) string {
	// Remove spaces, dashes, and parentheses
	cleaned := ""
	for _, c := range phone {
		if c >= '0' && c <= '9' || c == '+' {
			cleaned += string(c)
		}
	}

	// If already in E.164 format, return as is
	if len(cleaned) > 0 && cleaned[0] == '+' {
		return cleaned
	}

	// Add default country code if needed
	if defaultCountryCode != "" {
		if defaultCountryCode[0] != '+' {
			defaultCountryCode = "+" + defaultCountryCode
		}
		return defaultCountryCode + cleaned
	}

	return "+" + cleaned
}
