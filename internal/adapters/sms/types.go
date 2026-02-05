package sms

import (
	"time"
)

// TwilioConfig holds the configuration for a Twilio SMS channel
type TwilioConfig struct {
	AccountSID          string `json:"account_sid"`
	AuthToken           string `json:"auth_token"`
	APIKeySID           string `json:"api_key_sid,omitempty"`       // Optional: for API Key auth
	APIKeySecret        string `json:"api_key_secret,omitempty"`    // Optional: for API Key auth
	PhoneNumber         string `json:"phone_number,omitempty"`      // Either this or MessagingServiceSID
	MessagingServiceSID string `json:"messaging_service_sid,omitempty"` // Preferred over PhoneNumber
	StatusCallbackURL   string `json:"status_callback_url,omitempty"`
}

// MessageStatus represents Twilio message status
type MessageStatus string

const (
	StatusQueued      MessageStatus = "queued"
	StatusSending     MessageStatus = "sending"
	StatusSent        MessageStatus = "sent"
	StatusDelivered   MessageStatus = "delivered"
	StatusUndelivered MessageStatus = "undelivered"
	StatusFailed      MessageStatus = "failed"
	StatusReceived    MessageStatus = "received"
	StatusAccepted    MessageStatus = "accepted"
	StatusScheduled   MessageStatus = "scheduled"
	StatusRead        MessageStatus = "read"
	StatusCanceled    MessageStatus = "canceled"
)

// IncomingMessage represents an incoming SMS/MMS message from Twilio webhook
type IncomingMessage struct {
	MessageSID          string
	AccountSID          string
	From                string
	To                  string
	Body                string
	NumMedia            int
	MediaContentTypes   []string
	MediaURLs           []string
	FromCity            string
	FromState           string
	FromZip             string
	FromCountry         string
	ToCity              string
	ToState             string
	ToZip               string
	ToCountry           string
	APIVersion          string
	Timestamp           time.Time
}

// StatusCallback represents a delivery status callback from Twilio
type StatusCallback struct {
	MessageSID    string
	MessageStatus MessageStatus
	ErrorCode     string
	ErrorMessage  string
	To            string
	From          string
	Timestamp     time.Time
}

// OutgoingMessage represents a message to be sent via Twilio
type OutgoingMessage struct {
	To        string   // Phone number in E.164 format
	Body      string   // Message body
	MediaURLs []string // Optional: URLs for MMS media
}

// SendResult contains the result of sending an SMS
type SendResult struct {
	MessageSID string
	Status     MessageStatus
	Success    bool
	ErrorCode  string
	Error      string
}

// WebhookPayload represents the raw Twilio webhook payload
type WebhookPayload struct {
	// Message fields
	MessageSID        string `form:"MessageSid"`
	SmsSID            string `form:"SmsSid"`
	AccountSID        string `form:"AccountSid"`
	MessagingServiceSID string `form:"MessagingServiceSid"`
	From              string `form:"From"`
	To                string `form:"To"`
	Body              string `form:"Body"`
	NumMedia          string `form:"NumMedia"`

	// Status callback fields
	MessageStatus     string `form:"MessageStatus"`
	SmsStatus         string `form:"SmsStatus"`
	ErrorCode         string `form:"ErrorCode"`
	ErrorMessage      string `form:"ErrorMessage"`

	// Location fields (inbound only)
	FromCity          string `form:"FromCity"`
	FromState         string `form:"FromState"`
	FromZip           string `form:"FromZip"`
	FromCountry       string `form:"FromCountry"`
	ToCity            string `form:"ToCity"`
	ToState           string `form:"ToState"`
	ToZip             string `form:"ToZip"`
	ToCountry         string `form:"ToCountry"`

	// API info
	APIVersion        string `form:"ApiVersion"`
}

// ParseMessageStatus converts a Twilio status string to MessageStatus
func ParseMessageStatus(status string) MessageStatus {
	switch status {
	case "queued":
		return StatusQueued
	case "sending":
		return StatusSending
	case "sent":
		return StatusSent
	case "delivered":
		return StatusDelivered
	case "undelivered":
		return StatusUndelivered
	case "failed":
		return StatusFailed
	case "received":
		return StatusReceived
	case "accepted":
		return StatusAccepted
	case "scheduled":
		return StatusScheduled
	case "read":
		return StatusRead
	case "canceled":
		return StatusCanceled
	default:
		return MessageStatus(status)
	}
}

// IsTerminalStatus returns true if the status is a terminal (final) status
func (s MessageStatus) IsTerminalStatus() bool {
	switch s {
	case StatusDelivered, StatusUndelivered, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// IsSuccessStatus returns true if the message was successfully delivered
func (s MessageStatus) IsSuccessStatus() bool {
	return s == StatusDelivered || s == StatusRead
}

// IsFailureStatus returns true if the message failed
func (s MessageStatus) IsFailureStatus() bool {
	return s == StatusFailed || s == StatusUndelivered
}
