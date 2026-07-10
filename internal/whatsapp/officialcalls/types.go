// Package officialcalls implements the WhatsApp Business Calling API (Meta Cloud
// API) — the real SDP/WebRTC signaling protocol, distinct from the abstract
// call domain model in internal/whatsapp/calling.
//
// Signaling is over the Graph API: POST /{phone_number_id}/calls with an action
// (connect/pre_accept/accept/reject/terminate) and a WebRTC SDP session; inbound
// calls arrive via the "calls" webhook field (event=connect carries the user's
// SDP offer, event=terminate ends the call). Media is standard WebRTC
// (ICE + DTLS + SRTP, OPUS) handled by pion (see the webrtc session in this
// package), so no proprietary codec/relay is required — unlike the unofficial
// whatsmeow path.
package officialcalls

// SDPType is a WebRTC session description type.
type SDPType string

const (
	SDPOffer  SDPType = "offer"
	SDPAnswer SDPType = "answer"
)

// Session is the SDP session exchanged in call actions and webhooks.
type Session struct {
	SDPType SDPType `json:"sdp_type"`
	SDP     string  `json:"sdp"`
}

// CallAction is an action on POST /{phone_number_id}/calls.
type CallAction string

const (
	ActionConnect   CallAction = "connect"    // business-initiated call, or offer
	ActionPreAccept CallAction = "pre_accept" // pre-accept inbound with an early SDP answer
	ActionAccept    CallAction = "accept"     // accept inbound with SDP answer
	ActionReject    CallAction = "reject"     // reject inbound
	ActionTerminate CallAction = "terminate"  // end an active call
)

// CallDirection as reported by Meta on the webhook.
type CallDirection string

const (
	DirectionBusinessInitiated CallDirection = "BUSINESS_INITIATED"
	DirectionUserInitiated     CallDirection = "USER_INITIATED"
)

// Webhook event names in the "calls" field.
const (
	WebhookEventConnect   = "connect"
	WebhookEventTerminate = "terminate"
)

// CallingStatus toggles calling on a phone number.
type CallingStatus string

const (
	CallingEnabled  CallingStatus = "ENABLED"
	CallingDisabled CallingStatus = "DISABLED"
)

// CallingSettings is the "calling" object of POST /{phone_number_id}/settings.
type CallingSettings struct {
	Status                   CallingStatus `json:"status"`
	CallIconVisibility       string        `json:"call_icon_visibility,omitempty"`
	CallbackPermissionStatus string        `json:"callback_permission_status,omitempty"`
	CallHours                *CallHours    `json:"call_hours,omitempty"`
}

// CallHours restricts when the business accepts calls.
type CallHours struct {
	Status          string   `json:"status,omitempty"`
	TimezoneID      string   `json:"timezone_id,omitempty"`
	WeeklyOperating []DayHour `json:"weekly_operating_hours,omitempty"`
}

// DayHour is one entry of the weekly operating hours.
type DayHour struct {
	DayOfWeek string `json:"day_of_week"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// settingsRequest is the POST /settings body.
type settingsRequest struct {
	Calling *CallingSettings `json:"calling,omitempty"`
}

// callActionRequest is the POST /calls body.
type callActionRequest struct {
	MessagingProduct string     `json:"messaging_product"`
	Action           CallAction `json:"action"`
	CallID           string     `json:"call_id,omitempty"`
	To               string     `json:"to,omitempty"`
	Session          *Session   `json:"session,omitempty"`
	BizOpaqueData    string     `json:"biz_opaque_callback_data,omitempty"`
}

// callActionResponse is the POST /calls response.
type callActionResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Calls            []struct {
		ID string `json:"id"`
	} `json:"calls"`
}

// CallPermission is the per-user permission to place a business-initiated call.
type CallPermission struct {
	Status         string `json:"status"`           // e.g. "no_permission", "temporary", "permanent"
	CanPlaceCall   bool   `json:"can_place_call"`   // convenience, derived below
	ExpirationTime int64  `json:"expiration_time,omitempty"`
}

// WebhookCallEvent is one entry of the webhook "calls" array.
type WebhookCallEvent struct {
	ID        string        `json:"id"`
	From      string        `json:"from"`
	To        string        `json:"to"`
	Event     string        `json:"event"` // "connect" | "terminate"
	Timestamp string        `json:"timestamp"`
	Direction CallDirection `json:"direction,omitempty"`
	Session   *Session      `json:"session,omitempty"` // present on connect
	// Terminate details (present on event=terminate).
	Status   string `json:"status,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

// IsConnect reports whether this is an inbound offer to answer.
func (e WebhookCallEvent) IsConnect() bool { return e.Event == WebhookEventConnect }

// IsTerminate reports whether the call ended.
func (e WebhookCallEvent) IsTerminate() bool { return e.Event == WebhookEventTerminate }
